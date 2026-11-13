package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const doubleCheckAttributeName = "double_check"

// doubleCheckAttributeSchema returns the schema of the deprecated
// double_check attribute, which predates retry_strategy.
func doubleCheckAttributeSchema() schema.BoolAttribute {
	return schema.BoolAttribute{
		Optional:           true,
		Computed:           true,
		Default:            booldefault.StaticBool(false),
		Description:        "Setting this to `true` will trigger a retry when a check fails from the failing region and another, randomly selected region before marking the check as failed. Defaults to `false`.",
		DeprecationMessage: "The property `double_check` is deprecated and will be removed in a future version. To enable retries for failed check runs, use the `retry_strategy` property instead.",
		Validators: []validator.Bool{
			boolvalidator.ConflictsWith(path.MatchRoot(retryStrategyAttributeName)),
		},
	}
}

// doubleCheckRetryStrategy is the retry strategy the API applies to a check or
// group with double_check enabled.
func doubleCheckRetryStrategy(supportsOnlyOn bool) types.Object {
	attrValues := map[string]attr.Value{
		"type":                 types.StringValue("FIXED"),
		"base_backoff_seconds": types.Int64Value(0),
		"max_retries":          types.Int64Value(1),
		"max_duration_seconds": types.Int64Value(600),
		"same_region":          types.BoolValue(false),
	}
	if supportsOnlyOn {
		attrValues["only_on"] = types.ObjectNull(retryStrategyOnlyOnAttrTypes)
	}

	return types.ObjectValueMust(retryStrategyAttrTypes(supportsOnlyOn), attrValues)
}

var _ planmodifier.Object = doubleCheckRetryStrategyPlanModifier{}

// doubleCheckRetryStrategyPlanModifier plans the retry strategy that the API
// applies when double_check is enabled and retry_strategy is not configured,
// instead of the NO_RETRIES default, which the API would never report back.
type doubleCheckRetryStrategyPlanModifier struct{}

func (m doubleCheckRetryStrategyPlanModifier) Description(ctx context.Context) string {
	return "Plans the retry strategy that double_check implies when retry_strategy is not configured."
}

func (m doubleCheckRetryStrategyPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m doubleCheckRetryStrategyPlanModifier) PlanModifyObject(ctx context.Context, req planmodifier.ObjectRequest, resp *planmodifier.ObjectResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}

	var doubleCheck types.Bool
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(doubleCheckAttributeName), &doubleCheck)...)
	if resp.Diagnostics.HasError() {
		return
	}

	attrTypes := resp.PlanValue.AttributeTypes(ctx)
	if doubleCheck.IsUnknown() {
		// The retry strategy depends on a value known only at apply time.
		resp.PlanValue = types.ObjectUnknown(attrTypes)
		return
	}
	if !doubleCheck.ValueBool() {
		return
	}

	_, supportsOnlyOn := attrTypes["only_on"]
	resp.PlanValue = doubleCheckRetryStrategy(supportsOnlyOn)
}
