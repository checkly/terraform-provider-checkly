package provider

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/checkly/terraform-provider-checkly/internal/sdkutil"
)

var (
	_ resource.Resource                = (*HeartbeatResource)(nil)
	_ resource.ResourceWithConfigure   = (*HeartbeatResource)(nil)
	_ resource.ResourceWithImportState = (*HeartbeatResource)(nil)
)

type HeartbeatResource struct {
	client checkly.Client
}

func NewHeartbeatResource() resource.Resource {
	return &HeartbeatResource{}
}

func (r *HeartbeatResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_heartbeat"
}

func (r *HeartbeatResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description: "Heartbeats allows you to monitor your cron jobs and set up alerting, so you get a notification when things break or slow down.",
		Attributes: map[string]schema.Attribute{
			"id":           IDResourceAttributeSchema,
			"last_updated": LastUpdatedAttributeSchema,
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the check.",
			},
			"activated": schema.BoolAttribute{
				Required:    true,
				Description: "Determines if the check is running or not. Possible values `true`, and `false`.",
			},
			"muted": schema.BoolAttribute{
				Optional:    true,
				Description: "Determines if any notifications will be sent out when a check fails/degrades/recovers.",
			},
			"tags": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "A list of tags for organizing and filtering checks.",
			},
			"alert_settings": CheckAlertSettingsAttributeSchema,
			"use_global_alert_settings": schema.BoolAttribute{
				Optional:    true,
				Description: "When true, the account level alert settings will be used, not the alert setting defined on this check.",
			},
			"heartbeat": schema.SingleNestedAttribute{
				Required:   true,
				Validators: []validator.Object{
					// TODO: period * period_unit must be between 30s and 365 days
					// TODO: grace * grace_unit must be less than 365 days
				},
				Attributes: map[string]schema.Attribute{
					"period": schema.Int32Attribute{
						Required:    true,
						Description: "How often you expect a ping to the ping URL.",
					},
					"period_unit": schema.StringAttribute{
						Required: true,
						Validators: []validator.String{
							stringvalidator.OneOf("seconds", "minutes", "hours", "days"),
						},
						Description: "Possible values `seconds`, `minutes`, `hours` and `days`.",
					},
					"grace": schema.Int32Attribute{
						Required:    true,
						Description: "How long Checkly should wait before triggering any alerts when a ping does not arrive within the set period.",
					},
					"grace_unit": schema.StringAttribute{
						Required: true,
						Validators: []validator.String{
							stringvalidator.OneOf("seconds", "minutes", "hours", "days"),
						},
						Description: "Possible values `seconds`, `minutes`, `hours` and `days`.",
					},
					"ping_token": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Custom token to generate your ping URL. Checkly will expect a ping to `https://ping.checklyhq.com/[PING_TOKEN]`.",
					},
				},
			},
			"alert_channel_subscription": CheckAlertChannelSubscriptionAttributeSchema,
		},
	}
}

func (r *HeartbeatResource) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	client, diags := ClientFromProviderData(req.ProviderData)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	r.client = client
}

func (r *HeartbeatResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *HeartbeatResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan HeartbeatResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var desiredModel checkly.HeartbeatCheck
	resp.Diagnostics.Append(plan.Render(ctx, &desiredModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realizedModel, err := r.client.CreateHeartbeat(ctx, desiredModel)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Checkly Heartbeat",
			fmt.Sprintf("Could not create heartbeat, unexpected error: %s", err),
		)

		return
	}

	resp.Diagnostics.Append(plan.Refresh(ctx, realizedModel, ModelCreated)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *HeartbeatResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state HeartbeatResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteCheck(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Checkly Heartbeat",
			fmt.Sprintf("Could not delete heartbeat, unexpected error: %s", err),
		)

		return
	}
}

func (r *HeartbeatResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state HeartbeatResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realizedModel, err := r.client.GetHeartbeatCheck(ctx, state.ID.ValueString())
	if err != nil {
		if sdkutil.IsHTTPNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading Checkly Heartbeat",
			fmt.Sprintf("Could not retrieve heartbeat, unexpected error: %s", err),
		)

		return
	}

	resp.Diagnostics.Append(state.Refresh(ctx, realizedModel, ModelLoaded)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *HeartbeatResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan HeartbeatResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var desiredModel checkly.HeartbeatCheck
	resp.Diagnostics.Append(plan.Render(ctx, &desiredModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realizedModel, err := r.client.UpdateHeartbeat(
		ctx,
		plan.ID.ValueString(),
		desiredModel,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Checkly Heartbeat",
			fmt.Sprintf("Could not update heartbeat, unexpected error: %s", err),
		)

		return
	}

	resp.Diagnostics.Append(plan.Refresh(ctx, realizedModel, ModelUpdated)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

var (
	_ ResourceModel[checkly.HeartbeatCheck] = (*HeartbeatResourceModel)(nil)
	_ ResourceModel[checkly.Heartbeat]      = (*HeartbeatAttributeModel)(nil)
)

type HeartbeatResourceModel struct {
	ID                        types.String                                  `tfsdk:"id"`
	LastUpdated               types.String                                  `tfsdk:"last_updated"` // FIXME: Keep this? Old code did not have it.
	Name                      types.String                                  `tfsdk:"name"`
	Activated                 types.Bool                                    `tfsdk:"activated"`
	Muted                     types.Bool                                    `tfsdk:"muted"`
	Tags                      types.Set                                     `tfsdk:"tags"`
	AlertSettings             CheckAlertSettingsAttributeModel              `tfsdk:"alert_settings"`
	UseGlobalAlertSettings    types.Bool                                    `tfsdk:"use_global_alert_settings"`
	Heartbeat                 HeartbeatAttributeModel                       `tfsdk:"heartbeat"`
	AlertChannelSubscriptions []CheckAlertChannelSubscriptionAttributeModel `tfsdk:"alert_channel_subscription"`
}

func (m *HeartbeatResourceModel) Refresh(ctx context.Context, from *checkly.HeartbeatCheck, flags RefreshFlags) diag.Diagnostics {
	var diags diag.Diagnostics

	if flags.Created() {
		m.ID = types.StringValue(from.ID)
	}

	if flags.Created() || flags.Updated() {
		m.LastUpdated = LastUpdatedNow()
	}

	m.Name = types.StringValue(from.Name)
	m.Activated = types.BoolValue(from.Activated)
	m.Muted = types.BoolValue(from.Muted)

	slices.Sort(from.Tags)
	m.Tags = IntoUntypedStringSet(&from.Tags)

	diags.Append(m.AlertSettings.Refresh(ctx, &from.AlertSettings, flags)...)
	if diags.HasError() {
		return diags
	}

	m.UseGlobalAlertSettings = types.BoolValue(from.UseGlobalAlertSettings)

	diags.Append(m.Heartbeat.Refresh(ctx, &from.Heartbeat, flags)...)
	if diags.HasError() {
		return diags
	}

	m.AlertChannelSubscriptions = nil
	for _, sub := range from.AlertChannelSubscriptions {
		var subModel CheckAlertChannelSubscriptionAttributeModel
		diags.Append(subModel.Refresh(ctx, &sub, flags)...)
		if diags.HasError() {
			return diags
		}

		m.AlertChannelSubscriptions = append(m.AlertChannelSubscriptions, subModel)
	}

	return diags
}

func (m *HeartbeatResourceModel) Render(ctx context.Context, into *checkly.HeartbeatCheck) diag.Diagnostics {
	var diags diag.Diagnostics

	into.Name = m.Name.ValueString()
	into.Activated = m.Activated.ValueBool()
	into.Muted = m.Muted.ValueBool()
	into.Tags = FromUntypedStringSet(m.Tags)

	diags.Append(m.AlertSettings.Render(ctx, &into.AlertSettings)...)

	into.UseGlobalAlertSettings = m.UseGlobalAlertSettings.ValueBool()

	diags.Append(m.Heartbeat.Render(ctx, &into.Heartbeat)...)

	return diags
}

type HeartbeatAttributeModel struct {
	Period     types.Int32  `tfsdk:"period"`
	PeriodUnit types.String `tfsdk:"period_unit"`
	Grace      types.Int32  `tfsdk:"grace"`
	GraceUnit  types.String `tfsdk:"grace_unit"`
	PingToken  types.String `tfsdk:"ping_token"`
}

func (m *HeartbeatAttributeModel) Refresh(ctx context.Context, from *checkly.Heartbeat, flags RefreshFlags) diag.Diagnostics {
	m.Period = types.Int32Value(int32(from.Period))
	m.PeriodUnit = types.StringValue(from.PeriodUnit)
	m.Grace = types.Int32Value(int32(from.Grace))
	m.GraceUnit = types.StringValue(from.GraceUnit)
	m.PingToken = types.StringValue(from.PingToken)

	return nil
}

func (m *HeartbeatAttributeModel) Render(ctx context.Context, into *checkly.Heartbeat) diag.Diagnostics {
	into.Period = int(m.Period.ValueInt32())
	into.PeriodUnit = m.PeriodUnit.ValueString()
	into.Grace = int(m.Grace.ValueInt32())
	into.GraceUnit = m.GraceUnit.ValueString()
	into.PingToken = m.PingToken.ValueString()

	return nil
}
