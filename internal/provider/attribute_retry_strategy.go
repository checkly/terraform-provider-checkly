package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
)

const retryStrategyAttributeName = "retry_strategy"

var retryStrategyOnlyOnAttrTypes = map[string]attr.Type{
	"network_error": types.BoolType,
}

// retryStrategyAttrTypes returns the attribute types of the retry_strategy
// attribute. The only_on attribute is only present for resources whose API
// supports retry conditions.
func retryStrategyAttrTypes(supportsOnlyOn bool) map[string]attr.Type {
	attrTypes := map[string]attr.Type{
		"type":                 types.StringType,
		"base_backoff_seconds": types.Int64Type,
		"max_retries":          types.Int64Type,
		"max_duration_seconds": types.Int64Type,
		"same_region":          types.BoolType,
	}

	if supportsOnlyOn {
		attrTypes["only_on"] = types.ObjectType{AttrTypes: retryStrategyOnlyOnAttrTypes}
	}

	return attrTypes
}

// noRetriesObject returns the canonical attribute value for the NO_RETRIES
// strategy, which the Checkly API defaults to when no retry strategy is
// configured.
func noRetriesObject(supportsOnlyOn bool) types.Object {
	attrTypes := retryStrategyAttrTypes(supportsOnlyOn)

	attrValues := map[string]attr.Value{
		"type":                 types.StringValue("NO_RETRIES"),
		"base_backoff_seconds": types.Int64Null(),
		"max_retries":          types.Int64Null(),
		"max_duration_seconds": types.Int64Null(),
		"same_region":          types.BoolNull(),
	}

	if supportsOnlyOn {
		attrValues["only_on"] = types.ObjectNull(retryStrategyOnlyOnAttrTypes)
	}

	return types.ObjectValueMust(attrTypes, attrValues)
}

// RetryStrategyAttributeSchemaOptions customizes the retry_strategy
// attribute schema for a specific resource.
type RetryStrategyAttributeSchemaOptions struct {
	// SupportsOnlyOnNetworkError adds the only_on.network_error condition,
	// for resources whose API supports retry conditions.
	SupportsOnlyOnNetworkError bool
	// Required makes the attribute required instead of optional+computed
	// with a NO_RETRIES default.
	Required bool
	// DoubleCheck plans the retry strategy that the deprecated double_check
	// attribute implies, for resources that have it.
	DoubleCheck bool
}

// makeRetryStrategyAttributeSchema returns the schema for the retry_strategy
// attribute. Unless options.Required is set, the attribute is
// optional+computed and defaults to the NO_RETRIES strategy, mirroring the
// Checkly API default. The default also ensures that removing a previously
// configured retry_strategy from the configuration plans a change back to
// NO_RETRIES.
func makeRetryStrategyAttributeSchema(options RetryStrategyAttributeSchemaOptions) schema.SingleNestedAttribute {
	attributes := map[string]schema.Attribute{
		"type": schema.StringAttribute{
			Required:    true,
			Description: "Determines which type of retry strategy to use. Possible values are `FIXED`, `LINEAR`, `EXPONENTIAL`, `SINGLE_RETRY`, and `NO_RETRIES`.",
			Validators: []validator.String{
				stringvalidator.OneOf("FIXED", "LINEAR", "EXPONENTIAL", "SINGLE_RETRY", "NO_RETRIES"),
			},
		},
		"base_backoff_seconds": schema.Int64Attribute{
			Optional:    true,
			Computed:    true,
			Default:     int64default.StaticInt64(60),
			Description: "The number of seconds to wait before the first retry attempt. Not available when `type` is `NO_RETRIES`. (Default `60`).",
		},
		"max_retries": schema.Int64Attribute{
			Optional:    true,
			Computed:    true,
			Default:     int64default.StaticInt64(2),
			Description: "The maximum number of times to retry the check/monitor. Value must be between `1` and `10`. Available when `type` is `FIXED`, `LINEAR`, or `EXPONENTIAL`. (Default `2`).",
			Validators: []validator.Int64{
				int64validator.Between(1, 10),
			},
		},
		"max_duration_seconds": schema.Int64Attribute{
			Optional:    true,
			Computed:    true,
			Default:     int64default.StaticInt64(600),
			Description: "The total amount of time to continue retrying the check/monitor (maximum 600 seconds). Available when `type` is `FIXED`, `LINEAR`, or `EXPONENTIAL`. (Default `600`).",
			Validators: []validator.Int64{
				int64validator.Between(0, 600),
			},
		},
		"same_region": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(true),
			Description: "Whether retries should be run in the same region as the initial check/monitor run. Not available when `type` is `NO_RETRIES`. (Default `true`).",
		},
	}

	if options.SupportsOnlyOnNetworkError {
		attributes["only_on"] = schema.SingleNestedAttribute{
			Optional:    true,
			Description: "Apply the retry strategy only if the defined conditions match.",
			Attributes: map[string]schema.Attribute{
				"network_error": schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(false),
					Description: "When `true`, retry only if the cause of the failure is a network error. (Default `false`).",
				},
			},
		}
	}

	var objectDefault defaults.Object
	if !options.Required {
		objectDefault = objectdefault.StaticValue(noRetriesObject(options.SupportsOnlyOnNetworkError))
	}

	var planModifiers []planmodifier.Object
	if options.DoubleCheck {
		planModifiers = append(planModifiers, doubleCheckRetryStrategyPlanModifier{})
	}
	planModifiers = append(planModifiers, retryStrategyIgnoredFields)

	return schema.SingleNestedAttribute{
		Optional:      !options.Required,
		Required:      options.Required,
		Computed:      !options.Required,
		Default:       objectDefault,
		Description:   "A strategy for retrying failed check/monitor runs.",
		Attributes:    attributes,
		PlanModifiers: planModifiers,
	}
}

// retryStrategyFromObject converts the retry_strategy attribute value to its
// API representation. Null and unknown values convert to the NO_RETRIES
// strategy, which is what the Checkly API defaults to.
func retryStrategyFromObject(ctx context.Context, obj types.Object, diags *diag.Diagnostics) *checkly.RetryStrategy {
	if obj.IsNull() || obj.IsUnknown() {
		return &checkly.RetryStrategy{
			Type: "NO_RETRIES",
		}
	}

	attrs := obj.Attributes()

	kind := stringAttrValue(attrs["type"])

	switch kind {
	case "NO_RETRIES":
		return &checkly.RetryStrategy{
			Type: "NO_RETRIES",
		}
	case "SINGLE_RETRY":
		return &checkly.RetryStrategy{
			Type:               kind,
			BaseBackoffSeconds: int(int64AttrValue(attrs["base_backoff_seconds"])),
			SameRegion:         boolAttrValue(attrs["same_region"]),
			OnlyOn:             retryStrategyOnlyOnFromValue(attrs["only_on"]),
		}
	default:
		return &checkly.RetryStrategy{
			Type:               kind,
			BaseBackoffSeconds: int(int64AttrValue(attrs["base_backoff_seconds"])),
			MaxRetries:         int(int64AttrValue(attrs["max_retries"])),
			MaxDurationSeconds: int(int64AttrValue(attrs["max_duration_seconds"])),
			SameRegion:         boolAttrValue(attrs["same_region"]),
			OnlyOn:             retryStrategyOnlyOnFromValue(attrs["only_on"]),
		}
	}
}

func retryStrategyOnlyOnFromValue(value attr.Value) []string {
	obj, ok := value.(types.Object)
	if !ok || obj.IsNull() || obj.IsUnknown() {
		return nil
	}

	var conditions []string
	if boolAttrValue(obj.Attributes()["network_error"]) {
		conditions = append(conditions, "NETWORK_ERROR")
	}

	return conditions
}

// retryStrategyToObject converts the API representation of a retry strategy
// to the retry_strategy attribute value.
//
// The Checkly API does not echo strategy fields that are irrelevant for the
// strategy type (for example, max_retries for NO_RETRIES). For those fields
// the value from prior (the planned or previously stored attribute value) is
// preserved so that the resulting value stays consistent with the plan.
//
// The prior value must originate from a schema-derived model so that its
// object type is populated; it determines whether the result includes the
// only_on attribute.
func retryStrategyToObject(ctx context.Context, rs *checkly.RetryStrategy, prior types.Object, diags *diag.Diagnostics) types.Object {
	attrTypes := prior.AttributeTypes(ctx)
	_, supportsOnlyOn := attrTypes["only_on"]

	attrValues := map[string]attr.Value{
		"type":                 types.StringNull(),
		"base_backoff_seconds": types.Int64Null(),
		"max_retries":          types.Int64Null(),
		"max_duration_seconds": types.Int64Null(),
		"same_region":          types.BoolNull(),
	}
	if supportsOnlyOn {
		attrValues["only_on"] = types.ObjectNull(retryStrategyOnlyOnAttrTypes)
	}

	if !prior.IsNull() && !prior.IsUnknown() {
		for name, value := range prior.Attributes() {
			// Values preserved from the plan may still be unknown
			// (optional+computed fields that the practitioner did not
			// configure). Unknown values cannot be stored in state; the API
			// tracks no value for them, so they resolve to null.
			if !value.IsUnknown() {
				attrValues[name] = value
			}
		}
	}

	priorOnlyOn := attrValues["only_on"]

	if rs == nil {
		attrValues["type"] = types.StringValue("NO_RETRIES")
	} else {
		attrValues["type"] = types.StringValue(rs.Type)

		switch rs.Type {
		case "NO_RETRIES":
			// Keep prior values; the API does not track other fields for
			// this strategy type.
		case "SINGLE_RETRY":
			attrValues["base_backoff_seconds"] = types.Int64Value(int64(rs.BaseBackoffSeconds))
			attrValues["same_region"] = types.BoolValue(rs.SameRegion)
			if supportsOnlyOn {
				attrValues["only_on"] = retryStrategyOnlyOnToValue(rs.OnlyOn, priorOnlyOn)
			}
		default:
			attrValues["base_backoff_seconds"] = types.Int64Value(int64(rs.BaseBackoffSeconds))
			attrValues["max_retries"] = types.Int64Value(int64(rs.MaxRetries))
			attrValues["max_duration_seconds"] = types.Int64Value(int64(rs.MaxDurationSeconds))
			attrValues["same_region"] = types.BoolValue(rs.SameRegion)
			if supportsOnlyOn {
				attrValues["only_on"] = retryStrategyOnlyOnToValue(rs.OnlyOn, priorOnlyOn)
			}
		}
	}

	obj, d := types.ObjectValue(retryStrategyAttrTypes(supportsOnlyOn), attrValues)
	diags.Append(d...)

	return obj
}

func retryStrategyOnlyOnToValue(conditions []string, prior attr.Value) attr.Value {
	if len(conditions) == 1 && conditions[0] == "NETWORK_ERROR" {
		return types.ObjectValueMust(retryStrategyOnlyOnAttrTypes, map[string]attr.Value{
			"network_error": types.BoolValue(true),
		})
	}

	// The API reports no conditions. Preserve the prior value: it is either
	// null (not configured) or an object whose network_error value is false,
	// both of which are consistent with no conditions.
	if prior != nil && !prior.IsUnknown() {
		return prior
	}

	return types.ObjectNull(retryStrategyOnlyOnAttrTypes)
}

// stringAttrValue extracts a Go string from an attribute value, returning
// the zero value for null and unknown values.
func stringAttrValue(value attr.Value) string {
	if v, ok := value.(types.String); ok {
		return v.ValueString()
	}
	return ""
}

// int64AttrValue extracts a Go int64 from an attribute value, returning the
// zero value for null and unknown values.
func int64AttrValue(value attr.Value) int64 {
	if v, ok := value.(types.Int64); ok {
		return v.ValueInt64()
	}
	return 0
}

// boolAttrValue extracts a Go bool from an attribute value, returning the
// zero value for null and unknown values.
func boolAttrValue(value attr.Value) bool {
	if v, ok := value.(types.Bool); ok {
		return v.ValueBool()
	}
	return false
}
