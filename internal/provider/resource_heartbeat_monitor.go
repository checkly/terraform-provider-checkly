package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

var (
	_ resource.Resource                 = (*HeartbeatMonitorResource)(nil)
	_ resource.ResourceWithConfigure    = (*HeartbeatMonitorResource)(nil)
	_ resource.ResourceWithUpgradeState = (*HeartbeatMonitorResource)(nil)
	_ resource.ResourceWithImportState  = (*HeartbeatMonitorResource)(nil)
	_ resource.ResourceWithMoveState    = (*HeartbeatMonitorResource)(nil)
)

// NewHeartbeatMonitorResource returns a new checkly_heartbeat_monitor
// resource.
func NewHeartbeatMonitorResource() resource.Resource {
	return &HeartbeatMonitorResource{}
}

// HeartbeatMonitorResource implements the checkly_heartbeat_monitor resource.
type HeartbeatMonitorResource struct {
	client checkly.Client
}

// HeartbeatMonitorResourceModel describes the resource data model.
type HeartbeatMonitorResourceModel struct {
	ID                        types.String `tfsdk:"id"`
	Name                      types.String `tfsdk:"name"`
	Description               types.String `tfsdk:"description"`
	Activated                 types.Bool   `tfsdk:"activated"`
	Muted                     types.Bool   `tfsdk:"muted"`
	Tags                      types.Set    `tfsdk:"tags"`
	AlertSettings             types.Object `tfsdk:"alert_settings"`
	UseGlobalAlertSettings    types.Bool   `tfsdk:"use_global_alert_settings"`
	Heartbeat                 types.Object `tfsdk:"heartbeat"`
	AlertChannelSubscriptions types.Set    `tfsdk:"alert_channel_subscriptions"`
	TriggerIncident           types.Object `tfsdk:"trigger_incident"`
}

// HeartbeatMonitorHeartbeatModel describes the heartbeat attribute data model.
type HeartbeatMonitorHeartbeatModel struct {
	Period     types.Int64  `tfsdk:"period"`
	PeriodUnit types.String `tfsdk:"period_unit"`
	Grace      types.Int64  `tfsdk:"grace"`
	GraceUnit  types.String `tfsdk:"grace_unit"`
	PingToken  types.String `tfsdk:"ping_token"`
}

var heartbeatMonitorHeartbeatAttrTypes = map[string]attr.Type{
	"period":      types.Int64Type,
	"period_unit": types.StringType,
	"grace":       types.Int64Type,
	"grace_unit":  types.StringType,
	"ping_token":  types.StringType,
}

func (r *HeartbeatMonitorResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_heartbeat_monitor"
}

func (r *HeartbeatMonitorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     sdkv2SchemaVersion + 1,
		Description: "Heartbeats allows you to monitor your cron jobs and set up alerting, so you get a notification when things break or slow down.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the monitor.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the check.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "A description of the monitor.",
			},
			"activated": schema.BoolAttribute{
				Required:    true,
				Description: "Determines if the check is running or not. Possible values `true`, and `false`.",
			},
			"muted": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Determines if any notifications will be sent out when a check fails/degrades/recovers.",
			},
			"tags": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "A list of tags for organizing and filtering checks.",
			},
			triggerIncidentAttributeName: makeTriggerIncidentSchema(),
			alertSettingsAttributeName: makeAlertSettingsAttributeSchema(AlertSettingsAttributeSchemaOptions{
				Monitor: true,
			}),
			useGlobalAlertSettingsAttributeName: makeUseGlobalAlertSettingsAttributeSchema("monitor"),
			"heartbeat": schema.SingleNestedAttribute{
				Required:    true,
				Description: "The heartbeat schedule and grace period.",
				Validators: []validator.Object{
					heartbeatMonitorDurationValidator{},
				},
				Attributes: map[string]schema.Attribute{
					"period": schema.Int64Attribute{
						Required:    true,
						Description: "How often you expect a ping to the ping URL. The combination of `period` and `period_unit` must be between 30 seconds and 365 days.",
					},
					"period_unit": schema.StringAttribute{
						Required:    true,
						Description: "The unit of `period`. Possible values `seconds`, `minutes`, `hours` and `days`.",
						Validators: []validator.String{
							stringvalidator.OneOf("seconds", "minutes", "hours", "days"),
						},
					},
					"grace": schema.Int64Attribute{
						Required:    true,
						Description: "How long Checkly should wait before triggering any alerts when a ping does not arrive within the set period. The combination of `grace` and `grace_unit` must be at most 365 days.",
					},
					"grace_unit": schema.StringAttribute{
						Required:    true,
						Description: "The unit of `grace`. Possible values `seconds`, `minutes`, `hours` and `days`.",
						Validators: []validator.String{
							stringvalidator.OneOf("seconds", "minutes", "hours", "days"),
						},
					},
					"ping_token": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Computed:    true,
						Description: "Custom token to generate your ping URL. Checkly will expect a ping to `https://ping.checklyhq.com/[PING_TOKEN]`.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			alertChannelSubscriptionAttributeName: makeAlertChannelSubscriptionSchema(AlertChannelSubscriptionAttributeSchemaOptions{
				Monitor: true,
			}),
		},
	}
}

// UpgradeState upgrades state written by the provider's SDK v2
// implementation; see sdkv2StateUpgraders.
func (r *HeartbeatMonitorResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return sdkv2StateUpgraders(ctx, r)
}

func (r *HeartbeatMonitorResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *HeartbeatMonitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan HeartbeatMonitorResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	monitor := heartbeatMonitorFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.CreateHeartbeatMonitor(apiCtx, monitor)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Heartbeat Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	plan.ID = types.StringValue(result.ID)
	heartbeatMonitorToModel(ctx, result, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HeartbeatMonitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state HeartbeatMonitorResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	monitor, err := r.client.GetHeartbeatMonitor(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Read Heartbeat Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	heartbeatMonitorToModel(ctx, monitor, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *HeartbeatMonitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan HeartbeatMonitorResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	monitor := heartbeatMonitorFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.UpdateHeartbeatMonitor(apiCtx, plan.ID.ValueString(), monitor)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Heartbeat Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	heartbeatMonitorToModel(ctx, result, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HeartbeatMonitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state HeartbeatMonitorResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	err := r.client.DeleteHeartbeatMonitor(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Delete Heartbeat Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}
}

func (r *HeartbeatMonitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// MoveState enables `moved` blocks from the legacy checkly_heartbeat resource
// type. Both resource types share the exact same schema, so the source state
// can be used as the target state without transformation.
func (r *HeartbeatMonitorResource) MoveState(ctx context.Context) []resource.StateMover {
	return renamedResourceStateMovers(ctx, r, "checkly_heartbeat", "checkly_heartbeat_monitor")
}

// heartbeatMonitorFromModel converts the resource model to its API
// representation.
func heartbeatMonitorFromModel(ctx context.Context, model *HeartbeatMonitorResourceModel, diags *diag.Diagnostics) checkly.HeartbeatMonitor {
	monitor := checkly.HeartbeatMonitor{
		ID:                        model.ID.ValueString(),
		Name:                      model.Name.ValueString(),
		Activated:                 model.Activated.ValueBool(),
		Muted:                     model.Muted.ValueBool(),
		Tags:                      stringsFromSet(ctx, model.Tags, diags),
		AlertSettings:             alertSettingsFromObject(ctx, model.AlertSettings, diags),
		UseGlobalAlertSettings:    model.UseGlobalAlertSettings.ValueBool(),
		Heartbeat:                 heartbeatMonitorHeartbeatFromObject(ctx, model.Heartbeat, diags),
		AlertChannelSubscriptions: alertChannelSubscriptionsFromSet(ctx, model.AlertChannelSubscriptions, diags),
		TriggerIncident:           triggerIncidentFromObject(ctx, model.TriggerIncident, diags),
	}

	// An empty description is not tracked by the Checkly API; it is
	// equivalent to no description at all.
	if description := model.Description.ValueString(); description != "" {
		monitor.Description = &description
	}

	return monitor
}

// heartbeatMonitorToModel updates the model to match the API representation
// of the monitor. The model must hold prior values (the plan on
// create/update, the state on read); they are preserved for attributes that
// the API does not track or echo faithfully.
func heartbeatMonitorToModel(ctx context.Context, c *checkly.HeartbeatMonitor, model *HeartbeatMonitorResourceModel, diags *diag.Diagnostics) {
	model.Name = types.StringValue(c.Name)

	if c.Description != nil {
		model.Description = types.StringValue(*c.Description)
	} else {
		model.Description = types.StringValue("")
	}

	model.Activated = types.BoolValue(c.Activated)
	model.Muted = types.BoolValue(c.Muted)

	// The API reports no tags as an empty array. Preserve a prior null value
	// so that an unset attribute stays unset.
	model.Tags = stringSetFromAPINullIfUnset(ctx, c.Tags, model.Tags, diags)

	model.AlertSettings = alertSettingsToObject(ctx, c.AlertSettings, model.AlertSettings, diags)
	model.UseGlobalAlertSettings = types.BoolValue(c.UseGlobalAlertSettings)
	model.Heartbeat = heartbeatMonitorHeartbeatToObject(ctx, c.Heartbeat, diags)
	model.AlertChannelSubscriptions = alertChannelSubscriptionsToSet(ctx, c.AlertChannelSubscriptions, model.AlertChannelSubscriptions, diags)
	model.TriggerIncident = triggerIncidentToObject(ctx, c.TriggerIncident, diags)
}

// heartbeatMonitorHeartbeatFromObject converts the heartbeat attribute value
// to its API representation.
func heartbeatMonitorHeartbeatFromObject(ctx context.Context, obj types.Object, diags *diag.Diagnostics) checkly.Heartbeat {
	model := objectAs[HeartbeatMonitorHeartbeatModel](ctx, obj, diags)
	if model == nil {
		return checkly.Heartbeat{}
	}

	return checkly.Heartbeat{
		Period:     int(model.Period.ValueInt64()),
		PeriodUnit: checkly.HeartbeatUnit(model.PeriodUnit.ValueString()),
		Grace:      int(model.Grace.ValueInt64()),
		GraceUnit:  checkly.HeartbeatUnit(model.GraceUnit.ValueString()),
		PingToken:  model.PingToken.ValueString(),
	}
}

// heartbeatMonitorHeartbeatToObject converts the API representation of a
// heartbeat to the heartbeat attribute value.
func heartbeatMonitorHeartbeatToObject(ctx context.Context, heartbeat checkly.Heartbeat, diags *diag.Diagnostics) types.Object {
	obj, d := types.ObjectValue(heartbeatMonitorHeartbeatAttrTypes, map[string]attr.Value{
		"period":      types.Int64Value(int64(heartbeat.Period)),
		"period_unit": types.StringValue(string(heartbeat.PeriodUnit)),
		"grace":       types.Int64Value(int64(heartbeat.Grace)),
		"grace_unit":  types.StringValue(string(heartbeat.GraceUnit)),
		"ping_token":  types.StringValue(heartbeat.PingToken),
	})
	diags.Append(d...)

	return obj
}

var _ validator.Object = heartbeatMonitorDurationValidator{}

// heartbeatMonitorDurationValidator validates the period and grace durations
// of the heartbeat attribute. The period must be between 30 seconds and 365 days,
// and the grace period must be at most 365 days.
type heartbeatMonitorDurationValidator struct{}

func (v heartbeatMonitorDurationValidator) Description(ctx context.Context) string {
	return "period must be between 30 seconds and 365 days, and grace must be less than 365 days"
}

func (v heartbeatMonitorDurationValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v heartbeatMonitorDurationValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	attrs := req.ConfigValue.Attributes()

	period, periodOk := heartbeatMonitorDuration(attrs["period"], attrs["period_unit"])
	if periodOk && (period < 30*time.Second || period > 365*24*time.Hour) {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("period"),
			"Invalid Attribute Value",
			"period must be between 30 seconds and 365 days",
		)
	}

	grace, graceOk := heartbeatMonitorDuration(attrs["grace"], attrs["grace_unit"])
	if graceOk && grace > 365*24*time.Hour {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("grace"),
			"Invalid Attribute Value",
			"grace must be less than 365 days",
		)
	}
}

// heartbeatMonitorDuration combines an amount and a unit attribute value into
// a duration. It reports false when either value is null, unknown or not a
// valid unit, in which case there is nothing to validate.
func heartbeatMonitorDuration(amountValue, unitValue attr.Value) (time.Duration, bool) {
	amount, ok := amountValue.(types.Int64)
	if !ok || amount.IsNull() || amount.IsUnknown() {
		return 0, false
	}

	unit, ok := unitValue.(types.String)
	if !ok || unit.IsNull() || unit.IsUnknown() {
		return 0, false
	}

	var unitDuration time.Duration
	switch unit.ValueString() {
	case "seconds":
		unitDuration = time.Second
	case "minutes":
		unitDuration = time.Minute
	case "hours":
		unitDuration = time.Hour
	case "days":
		unitDuration = 24 * time.Hour
	default:
		return 0, false
	}

	return time.Duration(amount.ValueInt64()) * unitDuration, true
}
