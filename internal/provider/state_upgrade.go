package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

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
//     They are single nested attributes now.
//   - SDK v2 cannot store null for an unset string, number or collection, so
//     state refreshed by it holds `""`, `0`, `[]` or `{}` instead. Several Read
//     implementations keep such prior values, so they would never become
//     null again and every plan would propose removing them.
//   - Collections have plural names now, where SDK v2 used the singular name
//     of a repeated block or attribute (see sdkv2PluralRenames), and the
//     deprecated environment_variables map has been folded into the
//     environment_variables list.
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
// by the current schema: values stored under the singular name of a
// collection the schema now names in the plural are moved to the plural name,
// every value at a single nested position is unwrapped from its one-element
// array, empty strings, zero numbers and empty collections of optional,
// non-computed attributes become null, and attributes the current schema no
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

// normalizeSDKv2Object rewrites obj in place, at any depth below it: values
// under the SDK v2 names of renamed collections move to the names in attrs
// (see renameSDKv2Collections), every single nested attribute described by
// attrs holds an object (or null) instead of a list of at most one object,
// and every optional, non-computed attribute holds null instead of an empty
// string, zero number or empty collection. The remaining blocks are all
// repeated; they keep their empty lists, which is how the framework stores
// absent blocks too, and only their elements are normalized.
func normalizeSDKv2Object(obj map[string]any, attrs map[string]schema.Attribute, blocks map[string]schema.Block) {
	renameSDKv2Collections(obj, attrs)

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

// sdkv2PluralRenames maps the current plural name of a collection to the
// singular name SDK v2 stored it under, as a repeated block or as an
// attribute. A rename applies wherever the current schema has an attribute
// with the plural name, at any depth.
var sdkv2PluralRenames = map[string]string{
	"alert_channel_subscriptions": "alert_channel_subscription",
	"assertions":                  "assertion",
	"cards":                       "card",
	"components":                  "component",
	"devices":                     "device",
	"environment_variables":       "environment_variable",
	"service_attachments":         "service_attachment",
}

// renameSDKv2Collections moves the value obj holds under the SDK v2 name of
// every collection in attrs that sdkv2PluralRenames renames to the current
// name, and deletes the SDK v2 name. A non-null value already stored under the
// current name is kept, and the value under the SDK v2 name is discarded.
//
// SDK v2 also stored a deprecated environment_variables map of names to
// values next to the environment_variable list, keeping at most one of them
// non-empty. Where attrs has an environment_variables list, a non-empty map is
// converted to list elements that are neither locked nor secret, sorted by
// name because the map has no order; otherwise the list is used. A JSON
// object at any other list position is left as it is: no SDK v2 state has
// one there, and decoding it fails instead of losing data silently.
func renameSDKv2Collections(obj map[string]any, attrs map[string]schema.Attribute) {
	for name, attr := range attrs {
		oldName, ok := sdkv2PluralRenames[name]
		if !ok {
			continue
		}
		oldValue, hasOldValue := obj[oldName]
		delete(obj, oldName)

		value := obj[name]
		if m, ok := value.(map[string]any); ok && name == environmentVariablesSDKv2MapName {
			if _, ok := attr.(schema.ListNestedAttribute); ok {
				value = environmentVariableMapToList(m)
			}
		}

		if value == nil && hasOldValue {
			value = oldValue
		}
		obj[name] = value
	}
}

// environmentVariablesSDKv2MapName is the name of the deprecated SDK v2
// environment variable map, which the environment variable list has taken
// over.
const environmentVariablesSDKv2MapName = "environment_variables"

// environmentVariableMapToList converts an SDK v2 environment variable map to
// list elements sorted by key, or returns nil for an empty map.
func environmentVariableMapToList(m map[string]any) any {
	if len(m) == 0 {
		return nil
	}
	list := make([]any, 0, len(m))
	for _, key := range slices.Sorted(maps.Keys(m)) {
		list = append(list, map[string]any{
			"key":    key,
			"value":  m[key],
			"locked": false,
			"secret": false,
		})
	}
	return list
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

// moveSourceState returns the state of a `moved` block's source resource,
// decoded with sourceSchema, if this provider wrote it as one of
// sourceTypeNames. For any other source it returns nil without diagnostics,
// leaving the move to Terraform's other state movers.
//
// The framework only decodes source state that already matches sourceSchema;
// state still at sdkv2SchemaVersion is upgraded first, so that resources can
// be moved in the same plan that upgrades the provider. If the source state
// can be neither upgraded nor decoded, it adds an error to diags and returns
// nil.
func moveSourceState(ctx context.Context, req resource.MoveStateRequest, sourceSchema schema.Schema, diags *diag.Diagnostics, sourceTypeNames ...string) *tfsdk.State {
	if !strings.HasSuffix(req.SourceProviderAddress, "/checkly") {
		return nil
	}
	if !slices.Contains(sourceTypeNames, req.SourceTypeName) {
		return nil
	}

	if req.SourceSchemaVersion == sdkv2SchemaVersion && req.SourceRawState != nil {
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

	if req.SourceState == nil {
		diags.AddError(
			"Unable to Move Resource State",
			fmt.Sprintf(
				"The source %s state could not be decoded. The state may "+
					"have been created by an older version of the provider. "+
					"Upgrade the source resource to the current provider "+
					"version before moving it.",
				req.SourceTypeName,
			),
		)
		return nil
	}

	return req.SourceState
}

// renamedResourceStateMovers returns the state movers of resource r for
// `moved` blocks from sourceTypeNames, resource types of this provider that
// share the exact schema of r. The source state is used as the target state
// without transformation.
func renamedResourceStateMovers(ctx context.Context, r resource.Resource, sourceTypeNames ...string) []resource.StateMover {
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	sourceSchema := schemaResp.Schema

	return []resource.StateMover{
		{
			SourceSchema: &sourceSchema,
			StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				sourceState := moveSourceState(ctx, req, sourceSchema, &resp.Diagnostics, sourceTypeNames...)
				if sourceState == nil {
					return
				}

				resp.TargetState = *sourceState
			},
		},
	}
}
