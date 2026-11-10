package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/checkly/checkly-go-sdk/v2"
)

const alertSettingsAttributeName = "alert_settings"

// AlertSettingsModel describes the alert_settings attribute data model.
type AlertSettingsModel struct {
	EscalationType              types.String `tfsdk:"escalation_type"`
	RunBasedEscalation          types.Object `tfsdk:"run_based_escalation"`
	TimeBasedEscalation         types.Object `tfsdk:"time_based_escalation"`
	Reminders                   types.Object `tfsdk:"reminders"`
	ParallelRunFailureThreshold types.Object `tfsdk:"parallel_run_failure_threshold"`
}

// RunBasedEscalationModel describes the
// alert_settings.run_based_escalation attribute data model.
type RunBasedEscalationModel struct {
	FailedRunThreshold types.Int64 `tfsdk:"failed_run_threshold"`
}

// TimeBasedEscalationModel describes the
// alert_settings.time_based_escalation attribute data model.
type TimeBasedEscalationModel struct {
	MinutesFailingThreshold types.Int64 `tfsdk:"minutes_failing_threshold"`
}

// RemindersModel describes the alert_settings.reminders attribute data
// model.
type RemindersModel struct {
	Amount   types.Int64 `tfsdk:"amount"`
	Interval types.Int64 `tfsdk:"interval"`
}

// ParallelRunFailureThresholdModel describes the
// alert_settings.parallel_run_failure_threshold attribute data model.
type ParallelRunFailureThresholdModel struct {
	Enabled    types.Bool  `tfsdk:"enabled"`
	Percentage types.Int64 `tfsdk:"percentage"`
}

var runBasedEscalationAttrTypes = map[string]attr.Type{
	"failed_run_threshold": types.Int64Type,
}

var timeBasedEscalationAttrTypes = map[string]attr.Type{
	"minutes_failing_threshold": types.Int64Type,
}

var remindersAttrTypes = map[string]attr.Type{
	"amount":   types.Int64Type,
	"interval": types.Int64Type,
}

var parallelRunFailureThresholdAttrTypes = map[string]attr.Type{
	"enabled":    types.BoolType,
	"percentage": types.Int64Type,
}

var alertSettingsAttrTypes = map[string]attr.Type{
	"escalation_type":                types.StringType,
	"run_based_escalation":           types.ObjectType{AttrTypes: runBasedEscalationAttrTypes},
	"time_based_escalation":          types.ObjectType{AttrTypes: timeBasedEscalationAttrTypes},
	"reminders":                      types.ObjectType{AttrTypes: remindersAttrTypes},
	"parallel_run_failure_threshold": types.ObjectType{AttrTypes: parallelRunFailureThresholdAttrTypes},
}

// AlertSettingsAttributeSchemaOptions customizes the alert_settings
// attribute schema for a specific resource.
type AlertSettingsAttributeSchemaOptions struct {
	// Monitor adapts descriptions to talk about monitors instead of checks.
	Monitor bool
}

// makeAlertSettingsAttributeSchema returns the schema for the alert_settings
// attribute. The attribute is optional+computed: when it is absent from the
// configuration, its value is filled in from the Checkly API.
func makeAlertSettingsAttributeSchema(options AlertSettingsAttributeSchemaOptions) schema.SingleNestedAttribute {
	name := "check"
	if options.Monitor {
		name = "monitor"
	}

	return schema.SingleNestedAttribute{
		Optional:    true,
		Computed:    true,
		Description: fmt.Sprintf("Determines the alert escalation policy for the %s.", name),
		PlanModifiers: []planmodifier.Object{
			objectplanmodifier.UseStateForUnknown(),
		},
		Attributes: map[string]schema.Attribute{
			"escalation_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("RUN_BASED"),
				Description: "Determines the type of escalation to use. Possible values are `RUN_BASED` and `TIME_BASED`. (Default `RUN_BASED`).",
				Validators: []validator.String{
					stringvalidator.OneOf("RUN_BASED", "TIME_BASED"),
				},
			},
			"run_based_escalation": schema.SingleNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Configuration for run-based escalation.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseNonNullStateForUnknown(),
					inactiveEscalationPlanModifier{escalationType: "RUN_BASED"},
				},
				Attributes: map[string]schema.Attribute{
					"failed_run_threshold": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Default:     int64default.StaticInt64(1),
						Description: fmt.Sprintf("Send an alert notification after the given number of consecutive %s runs have failed. Possible values are between `1` and `5`. (Default `1`).", name),
						Validators: []validator.Int64{
							int64validator.Between(1, 5),
						},
					},
				},
			},
			"time_based_escalation": schema.SingleNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Configuration for time-based escalation.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseNonNullStateForUnknown(),
					inactiveEscalationPlanModifier{escalationType: "TIME_BASED"},
				},
				Attributes: map[string]schema.Attribute{
					"minutes_failing_threshold": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Default:     int64default.StaticInt64(5),
						Description: fmt.Sprintf("Send an alert notification after the %s has been failing for the given amount of time (in minutes). Possible values are `5`, `10`, `15`, and `30`. (Default `5`).", name),
						Validators: []validator.Int64{
							int64validator.OneOf(5, 10, 15, 30),
						},
					},
				},
			},
			"reminders": schema.SingleNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Defines how often to send reminder notifications after the initial alert.",
				// A stored null, which a check group without alert settings
				// holds, must not be planned again once alert_settings is
				// configured: the API then returns the object, so it stays
				// unknown until Read fills it in.
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseNonNullStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"amount": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Default:     int64default.StaticInt64(0),
						Description: "Number of reminder notifications to send. Possible values are `0`, `1`, `2`, `3`, `4`, `5`, and `100000` (`0` to disable, `100000` for unlimited). (Default `0`).",
						Validators: []validator.Int64{
							int64validator.OneOf(0, 1, 2, 3, 4, 5, 100000),
						},
					},
					"interval": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Default:     int64default.StaticInt64(5),
						Description: "Interval between reminder notifications in minutes. Possible values are `5`, `10`, `15`, and `30`. (Default `5`).",
						Validators: []validator.Int64{
							int64validator.OneOf(5, 10, 15, 30),
						},
					},
				},
			},
			"parallel_run_failure_threshold": schema.SingleNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Configuration for parallel run failure threshold.",
				// A stored null, which a check group without alert settings
				// holds, must not be planned again once alert_settings is
				// configured: the API then returns the object, so it stays
				// unknown until Read fills it in.
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseNonNullStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
						Description: fmt.Sprintf("Whether parallel run failure threshold is enabled. Only applies if the %s is scheduled for multiple locations in parallel. (Default `false`).", name),
					},
					"percentage": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Default:     int64default.StaticInt64(10),
						Description: "Percentage of runs that must fail to trigger alert. Possible values are `10`, `20`, `30`, `40`, `50`, `60`, `70`, `80`, `90`, and `100`. (Default `10`).",
						Validators: []validator.Int64{
							int64validator.OneOf(10, 20, 30, 40, 50, 60, 70, 80, 90, 100),
						},
					},
				},
			},
		},
	}
}

// alertSettingsFromObject converts the alert_settings attribute value to its
// API representation. Null and unknown values convert to the default alert
// settings.
func alertSettingsFromObject(ctx context.Context, obj types.Object, diags *diag.Diagnostics) checkly.AlertSettings {
	if obj.IsNull() || obj.IsUnknown() {
		return checkly.AlertSettings{
			EscalationType: checkly.EscalationTypeRunBased,
			RunBasedEscalation: checkly.RunBasedEscalation{
				FailedRunThreshold: 1,
			},
		}
	}

	var model AlertSettingsModel
	diags.Append(obj.As(ctx, &model, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return checkly.AlertSettings{}
	}

	alertSettings := checkly.AlertSettings{
		EscalationType: checkly.EscalationType(model.EscalationType.ValueString()),
	}

	if !model.Reminders.IsNull() && !model.Reminders.IsUnknown() {
		var reminders RemindersModel
		diags.Append(model.Reminders.As(ctx, &reminders, basetypes.ObjectAsOptions{})...)
		alertSettings.Reminders = checkly.Reminders{
			Amount:   int(reminders.Amount.ValueInt64()),
			Interval: int(reminders.Interval.ValueInt64()),
		}
	}

	if !model.ParallelRunFailureThreshold.IsNull() && !model.ParallelRunFailureThreshold.IsUnknown() {
		var threshold ParallelRunFailureThresholdModel
		diags.Append(model.ParallelRunFailureThreshold.As(ctx, &threshold, basetypes.ObjectAsOptions{})...)
		alertSettings.ParallelRunFailureThreshold = checkly.ParallelRunFailureThreshold{
			Enabled:    threshold.Enabled.ValueBool(),
			Percentage: int(threshold.Percentage.ValueInt64()),
		}
	}

	if alertSettings.EscalationType == checkly.EscalationTypeRunBased {
		alertSettings.RunBasedEscalation = checkly.RunBasedEscalation{
			FailedRunThreshold: 1,
		}
		if !model.RunBasedEscalation.IsNull() && !model.RunBasedEscalation.IsUnknown() {
			var escalation RunBasedEscalationModel
			diags.Append(model.RunBasedEscalation.As(ctx, &escalation, basetypes.ObjectAsOptions{})...)
			alertSettings.RunBasedEscalation = checkly.RunBasedEscalation{
				FailedRunThreshold: int(escalation.FailedRunThreshold.ValueInt64()),
			}
		}
	} else {
		if alertSettings.EscalationType == checkly.EscalationTypeTimeBased {
			// The documented default of time_based_escalation, which the
			// plan leaves unknown when the object is not configured.
			alertSettings.TimeBasedEscalation = checkly.TimeBasedEscalation{
				MinutesFailingThreshold: 5,
			}
		}
		if !model.TimeBasedEscalation.IsNull() && !model.TimeBasedEscalation.IsUnknown() {
			var escalation TimeBasedEscalationModel
			diags.Append(model.TimeBasedEscalation.As(ctx, &escalation, basetypes.ObjectAsOptions{})...)
			alertSettings.TimeBasedEscalation = checkly.TimeBasedEscalation{
				MinutesFailingThreshold: int(escalation.MinutesFailingThreshold.ValueInt64()),
			}
		}
	}

	return alertSettings
}

// alertSettingsToObject converts the API representation of alert settings to
// the alert_settings attribute value.
//
// The Checkly API only tracks the escalation configuration matching the
// escalation type. For the other escalation configuration the value from
// prior (the planned or previously stored attribute value) is preserved so
// that the resulting value stays consistent with the plan.
func alertSettingsToObject(ctx context.Context, as checkly.AlertSettings, prior types.Object, diags *diag.Diagnostics) types.Object {
	var priorModel AlertSettingsModel
	if !prior.IsNull() && !prior.IsUnknown() {
		diags.Append(prior.As(ctx, &priorModel, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return types.ObjectNull(alertSettingsAttrTypes)
		}
	} else {
		priorModel = AlertSettingsModel{
			EscalationType:              types.StringNull(),
			RunBasedEscalation:          types.ObjectNull(runBasedEscalationAttrTypes),
			TimeBasedEscalation:         types.ObjectNull(timeBasedEscalationAttrTypes),
			Reminders:                   types.ObjectNull(remindersAttrTypes),
			ParallelRunFailureThreshold: types.ObjectNull(parallelRunFailureThresholdAttrTypes),
		}
	}

	model := priorModel

	model.EscalationType = types.StringValue(string(as.EscalationType))

	if as.EscalationType == checkly.EscalationTypeRunBased {
		escalation, d := types.ObjectValueFrom(ctx, runBasedEscalationAttrTypes, RunBasedEscalationModel{
			FailedRunThreshold: types.Int64Value(int64(as.RunBasedEscalation.FailedRunThreshold)),
		})
		diags.Append(d...)
		model.RunBasedEscalation = escalation
	} else {
		escalation, d := types.ObjectValueFrom(ctx, timeBasedEscalationAttrTypes, TimeBasedEscalationModel{
			MinutesFailingThreshold: types.Int64Value(int64(as.TimeBasedEscalation.MinutesFailingThreshold)),
		})
		diags.Append(d...)
		model.TimeBasedEscalation = escalation
	}

	reminders, d := types.ObjectValueFrom(ctx, remindersAttrTypes, RemindersModel{
		Amount:   types.Int64Value(int64(as.Reminders.Amount)),
		Interval: types.Int64Value(int64(as.Reminders.Interval)),
	})
	diags.Append(d...)
	model.Reminders = reminders

	threshold, d := types.ObjectValueFrom(ctx, parallelRunFailureThresholdAttrTypes, ParallelRunFailureThresholdModel{
		Enabled:    types.BoolValue(as.ParallelRunFailureThreshold.Enabled),
		Percentage: types.Int64Value(int64(as.ParallelRunFailureThreshold.Percentage)),
	})
	diags.Append(d...)
	model.ParallelRunFailureThreshold = threshold

	// The escalation configuration for the non-active escalation type may
	// still be unknown when it was absent from the plan. Unknown values
	// cannot be stored in state; the API tracks no value for them, so they
	// resolve to null.
	if model.RunBasedEscalation.IsUnknown() {
		model.RunBasedEscalation = types.ObjectNull(runBasedEscalationAttrTypes)
	}
	if model.TimeBasedEscalation.IsUnknown() {
		model.TimeBasedEscalation = types.ObjectNull(timeBasedEscalationAttrTypes)
	}

	obj, objDiags := types.ObjectValueFrom(ctx, alertSettingsAttrTypes, model)
	diags.Append(objDiags...)

	return obj
}

// planUnconfiguredAlertSettingsAsUnknown plans every sub-object that the
// configuration leaves out of alert_settings as unknown, when the stored
// alert_settings has an empty escalation_type. A group without alert settings
// is stored that way, with zero values that are not settings: left to the
// sub-objects' plan modifiers they would be planned as stored, which the API
// either rejects or returns differently. Unknown sub-objects are skipped by
// alertSettingsFromObject and resolved by alertSettingsToObject.
func planUnconfiguredAlertSettingsAsUnknown(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}

	attributePath := path.Root(alertSettingsAttributeName)

	var state, config types.Object
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, attributePath, &state)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, attributePath, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if state.IsNull() || config.IsNull() || config.IsUnknown() {
		return
	}
	if escalationType, ok := state.Attributes()["escalation_type"].(types.String); !ok || escalationType.ValueString() != "" {
		return
	}

	for name, value := range config.Attributes() {
		subObject, ok := value.(types.Object)
		if !ok || !subObject.IsNull() {
			continue
		}
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, attributePath.AtName(name), types.ObjectUnknown(subObject.AttributeTypes(ctx)))...)
	}
}

// inactiveEscalationPlanModifier plans the stored value, null included, of
// an unconfigured escalation object that belongs to escalationType while the
// planned escalation_type is a different one. The API does not track the
// inactive escalation, so Read resolves it to its stored value or null.
// Without this, any other change of the resource would plan the inactive
// escalation as unknown, because UseNonNullStateForUnknown leaves a null
// stored value unknown.
type inactiveEscalationPlanModifier struct {
	escalationType string
}

func (m inactiveEscalationPlanModifier) Description(ctx context.Context) string {
	return fmt.Sprintf("Keeps the stored value while escalation_type is not %s.", m.escalationType)
}

func (m inactiveEscalationPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m inactiveEscalationPlanModifier) PlanModifyObject(ctx context.Context, req planmodifier.ObjectRequest, resp *planmodifier.ObjectResponse) {
	if !resp.PlanValue.IsUnknown() || !req.ConfigValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}

	var escalationType types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, req.Path.ParentPath().AtName("escalation_type"), &escalationType)...)
	if resp.Diagnostics.HasError() || escalationType.IsNull() || escalationType.IsUnknown() || escalationType.ValueString() == m.escalationType {
		return
	}

	resp.PlanValue = req.StateValue
}
