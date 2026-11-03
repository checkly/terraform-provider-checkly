package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// sdkv2SchemaVersion is the schema version of all state written by the
// provider's Terraform Plugin SDK v2 implementation (releases before v2.0.0).
//
// That state differs from what the framework implementation writes in two
// ways:
//
//   - Single nested objects were blocks limited to one element, which SDK v2
//     stores as a JSON array: `[{...}]`, or `[]` when the block is absent.
//   - SDK v2 cannot store null for an unset string, number or collection, so
//     state refreshed by it holds `""`, `0`, `[]` or `{}` instead. Several Read
//     implementations keep such prior values, so they would never become
//     null again and every plan would propose removing them.
//
// Zero values of nested fields that the API only ignores in some cases, such
// as the retry_strategy fields NO_RETRIES does not use, are not normalized
// here: ignoredFieldsPlanModifier keeps whatever is stored for them.
//
// Every resource type that the SDK v2 implementation provided therefore has
// a newer schema version and upgrades the old state with sdkv2StateUpgraders.
const sdkv2SchemaVersion = 0

// sdkv2StateUpgraders returns the state upgraders of a resource whose state
// may have been written at sdkv2SchemaVersion. The upgrade is driven entirely
// by the current schema: every value at a single nested position is unwrapped
// from its one-element array, empty strings, zero numbers and empty
// collections of optional, non-computed attributes become null, and attributes the current schema no
// longer has are dropped. All other values carry over unchanged; Read
// refreshes them from the API afterwards.
//
// Because the upgrade always targets the current schema, a later schema
// version that renames or reshapes an attribute must also apply that change
// to SDK v2 state, on top of this normalization: attributes the current
// schema does not know are silently dropped.
func sdkv2StateUpgraders(ctx context.Context, r resource.Resource) map[int64]resource.StateUpgrader {
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema

	return map[int64]resource.StateUpgrader{
		sdkv2SchemaVersion: {
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				value, err := upgradeSDKv2RawState(ctx, req.RawState, s)
				if err != nil {
					resp.Diagnostics.AddError(
						"Unable to Upgrade Resource State",
						"The resource state written by an earlier version of the provider could not be "+
							"converted to the current schema. Please report this to the provider developers.\n\n"+
							err.Error(),
					)
					return
				}
				resp.State.Raw = value
			},
		},
	}
}

// upgradeSDKv2RawState converts raw state written by the SDK v2
// implementation to the shape of schema s and decodes it.
func upgradeSDKv2RawState(ctx context.Context, raw *tfprotov6.RawState, s schema.Schema) (tftypes.Value, error) {
	if raw == nil || raw.JSON == nil {
		return tftypes.Value{}, fmt.Errorf("the state has no JSON representation")
	}

	// UseNumber keeps numbers (IDs in particular) exact instead of rounding
	// them through float64.
	decoder := json.NewDecoder(bytes.NewReader(raw.JSON))
	decoder.UseNumber()
	var obj map[string]any
	if err := decoder.Decode(&obj); err != nil {
		return tftypes.Value{}, fmt.Errorf("decoding state JSON: %w", err)
	}

	normalizeSDKv2Object(obj, s.Attributes, s.Blocks)

	upgraded, err := json.Marshal(obj)
	if err != nil {
		return tftypes.Value{}, fmt.Errorf("encoding state JSON: %w", err)
	}

	return (&tfprotov6.RawState{JSON: upgraded}).UnmarshalWithOpts(
		s.Type().TerraformType(ctx),
		tfprotov6.UnmarshalOpts{
			ValueFromJSONOpts: tftypes.ValueFromJSONOpts{
				// Attributes removed from the schema since SDK v2.
				IgnoreUndefinedAttributes: true,
			},
		},
	)
}

// normalizeSDKv2Object rewrites obj in place, at any depth below it: every
// single nested attribute or block described by attrs and blocks holds an
// object (or null) instead of a list of at most one object, and every
// optional, non-computed attribute holds null instead of an empty string,
// zero number or empty collection. Nested blocks other than single ones keep
// their empty lists, which is how the framework stores absent blocks too.
func normalizeSDKv2Object(obj map[string]any, attrs map[string]schema.Attribute, blocks map[string]schema.Block) {
	for name, attr := range attrs {
		value, ok := obj[name]
		if !ok || value == nil {
			continue
		}

		switch attr := attr.(type) {
		case schema.SingleNestedAttribute:
			value = unwrapList(value)
			if nested, ok := value.(map[string]any); ok {
				normalizeSDKv2Object(nested, attr.Attributes, nil)
			}
		case schema.ListNestedAttribute:
			forEachObject(value, func(nested map[string]any) {
				normalizeSDKv2Object(nested, attr.NestedObject.Attributes, nil)
			})
		case schema.SetNestedAttribute:
			forEachObject(value, func(nested map[string]any) {
				normalizeSDKv2Object(nested, attr.NestedObject.Attributes, nil)
			})
		case schema.MapNestedAttribute:
			if m, ok := value.(map[string]any); ok {
				for _, element := range m {
					if nested, ok := element.(map[string]any); ok {
						normalizeSDKv2Object(nested, attr.NestedObject.Attributes, nil)
					}
				}
			}
		}

		if attr.IsOptional() && !attr.IsComputed() && isSDKv2ZeroValue(value) {
			value = nil
		}
		obj[name] = value
	}

	for name, block := range blocks {
		value, ok := obj[name]
		if !ok || value == nil {
			continue
		}

		switch block := block.(type) {
		case schema.SingleNestedBlock:
			obj[name] = unwrapList(value)
			if nested, ok := obj[name].(map[string]any); ok {
				normalizeSDKv2Object(nested, block.Attributes, block.Blocks)
			}
		case schema.ListNestedBlock:
			forEachObject(value, func(nested map[string]any) {
				normalizeSDKv2Object(nested, block.NestedObject.Attributes, block.NestedObject.Blocks)
			})
		case schema.SetNestedBlock:
			forEachObject(value, func(nested map[string]any) {
				normalizeSDKv2Object(nested, block.NestedObject.Attributes, block.NestedObject.Blocks)
			})
		}
	}
}

// isSDKv2ZeroValue reports whether value is an empty JSON string, a zero
// number, or an empty array or object.
func isSDKv2ZeroValue(value any) bool {
	switch value := value.(type) {
	case string:
		return value == ""
	case json.Number:
		f, err := value.Float64()
		return err == nil && f == 0
	case []any:
		return len(value) == 0
	case map[string]any:
		return len(value) == 0
	}
	return false
}

// unwrapList returns the first element of a list, or nil for an empty one.
// SDK v2 limited these lists to one element, except for a few nested lists
// (such as alert_settings.reminders) that were unbounded in the schema but
// only ever read and written as their first element. Values that are not
// lists are returned unchanged.
func unwrapList(value any) any {
	list, ok := value.([]any)
	if !ok {
		return value
	}
	if len(list) == 0 {
		return nil
	}
	return list[0]
}

// forEachObject calls fn for every object element of a JSON list.
func forEachObject(value any, fn func(map[string]any)) {
	list, ok := value.([]any)
	if !ok {
		return
	}
	for _, element := range list {
		if nested, ok := element.(map[string]any); ok {
			fn(nested)
		}
	}
}

// sdkv2MoveSourceState returns the state of a `moved` block's source
// resource, decoded with sourceSchema. The framework only decodes source
// state that already matches sourceSchema; state still at
// sdkv2SchemaVersion is upgraded first, so that resources can be moved in the
// same plan that upgrades the provider. If that upgrade fails, it adds an
// error to diags and returns nil. It also returns nil if the framework could
// not decode the source state.
func sdkv2MoveSourceState(ctx context.Context, req resource.MoveStateRequest, sourceSchema schema.Schema, diags *diag.Diagnostics) *tfsdk.State {
	if req.SourceSchemaVersion != sdkv2SchemaVersion || req.SourceRawState == nil {
		return req.SourceState
	}

	value, err := upgradeSDKv2RawState(ctx, req.SourceRawState, sourceSchema)
	if err != nil {
		diags.AddError(
			"Unable to Move Resource State",
			fmt.Sprintf(
				"The %s state written by an earlier version of the provider could not be "+
					"converted to the current schema. Please report this to the provider developers.\n\n%s",
				req.SourceTypeName, err,
			),
		)
		return nil
	}

	return &tfsdk.State{Schema: sourceSchema, Raw: value}
}
