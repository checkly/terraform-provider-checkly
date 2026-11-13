package provider

import (
	"context"
	"maps"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ planmodifier.Object = ignoredFieldsPlanModifier{}

// ignoredFieldsPlanModifier keeps the stored value of nested fields that the
// API ignores, as long as they are not configured. The API does not return
// such fields, so their state is whatever was stored before; v1.x of the
// provider, for example, stored zero values for them. Planning them back to
// their defaults would be a change without effect.
type ignoredFieldsPlanModifier struct {
	description string

	// ignoredFields returns the fields the API ignores in both the planned
	// and the stored object, or nil if there are none.
	ignoredFields func(plan, state map[string]attr.Value) []string
}

func (m ignoredFieldsPlanModifier) Description(ctx context.Context) string {
	return m.description
}

func (m ignoredFieldsPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.description
}

func (m ignoredFieldsPlanModifier) PlanModifyObject(ctx context.Context, req planmodifier.ObjectRequest, resp *planmodifier.ObjectResponse) {
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() || resp.PlanValue.IsNull() || resp.PlanValue.IsUnknown() {
		return
	}

	plan := resp.PlanValue.Attributes()
	state := req.StateValue.Attributes()

	var config map[string]attr.Value
	if !req.ConfigValue.IsNull() && !req.ConfigValue.IsUnknown() {
		config = req.ConfigValue.Attributes()
	}

	values := maps.Clone(plan)
	changed := false
	for _, name := range m.ignoredFields(plan, state) {
		if configured, ok := config[name]; ok && !configured.IsNull() {
			continue
		}
		stored, ok := state[name]
		if !ok || stored.IsUnknown() || stored.Equal(values[name]) {
			continue
		}
		values[name] = stored
		changed = true
	}

	if !changed {
		return
	}

	obj, diags := types.ObjectValue(resp.PlanValue.AttributeTypes(ctx), values)
	resp.Diagnostics.Append(diags...)
	if !diags.HasError() {
		resp.PlanValue = obj
	}
}

// retryStrategyIgnoredFields keeps the fields that the strategy type does
// not use: all of them for NO_RETRIES, and max_retries and
// max_duration_seconds for SINGLE_RETRY.
var retryStrategyIgnoredFields = ignoredFieldsPlanModifier{
	description: "Keeps the stored value of fields that the retry strategy type does not use, unless they are configured.",
	ignoredFields: func(plan, state map[string]attr.Value) []string {
		if !plan["type"].Equal(state["type"]) {
			return nil
		}

		switch stringAttrValue(plan["type"]) {
		case "NO_RETRIES":
			return []string{"base_backoff_seconds", "max_retries", "max_duration_seconds", "same_region", "only_on"}
		case "SINGLE_RETRY":
			return []string{"max_retries", "max_duration_seconds"}
		}
		return nil
	},
}

// checkGroupV2DisabledEnforcementIgnoredFields keeps all fields of an
// enforcement other than enabled while it is disabled.
var checkGroupV2DisabledEnforcementIgnoredFields = ignoredFieldsPlanModifier{
	description: "Keeps the stored value of the fields of a disabled enforcement, unless they are configured.",
	ignoredFields: func(plan, state map[string]attr.Value) []string {
		if !plan["enabled"].Equal(types.BoolValue(false)) || !state["enabled"].Equal(types.BoolValue(false)) {
			return nil
		}

		var fields []string
		for name := range plan {
			if name != "enabled" {
				fields = append(fields, name)
			}
		}
		return fields
	},
}
