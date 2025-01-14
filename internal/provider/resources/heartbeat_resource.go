package resources

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/checkly/terraform-provider-checkly/internal/provider/interop"
	"github.com/checkly/terraform-provider-checkly/internal/provider/resources/attributes"
	"github.com/checkly/terraform-provider-checkly/internal/sdkutil"
)

var (
	_ resource.Resource                   = (*HeartbeatResource)(nil)
	_ resource.ResourceWithConfigure      = (*HeartbeatResource)(nil)
	_ resource.ResourceWithImportState    = (*HeartbeatResource)(nil)
	_ resource.ResourceWithValidateConfig = (*HeartbeatResource)(nil)
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
			"id": attributes.IDAttributeSchema,
			"name": schema.StringAttribute{
				Description: "The name of the check.",
				Required:    true,
			},
			"activated": schema.BoolAttribute{
				Description: "Determines if the check is running or not. Possible values `true`, and `false`.",
				Required:    true,
			},
			"muted": schema.BoolAttribute{
				Description: "Determines if any notifications will be sent out when a check fails/degrades/recovers.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"tags": schema.SetAttribute{
				Description: "A list of tags for organizing and filtering checks.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"alert_settings": attributes.AlertSettingsAttributeSchema,
			"use_global_alert_settings": schema.BoolAttribute{
				Description: "When true, the account level alert settings will be used, not the alert setting defined on this check.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"heartbeat":                  HeartbeatAttributeSchema,
			"alert_channel_subscription": attributes.AlertChannelSubscriptionAttributeSchema,
		},
	}
}

func valueWithUnitToSeconds(value int32, unit string) int32 {
	switch unit {
	case "seconds":
		return value * 1
	case "minutes":
		return value * 60
	case "hours":
		return value * 3600
	case "days":
		return value * 3600 * 24
	default:
		return 0
	}
}

func (r *HeartbeatResource) ValidateConfig(
	ctx context.Context,
	req resource.ValidateConfigRequest,
	resp *resource.ValidateConfigResponse,
) {
	var config HeartbeatResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, heartbeatAttributeModel, diags := HeartbeatAttributeGluer.RenderFromObject(ctx, config.Heartbeat)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	if !heartbeatAttributeModel.Period.IsUnknown() && !heartbeatAttributeModel.PeriodUnit.IsUnknown() {
		value := heartbeatAttributeModel.Period.ValueInt32()
		valuePath := path.Root("heartbeat").AtName("period")

		unit := heartbeatAttributeModel.PeriodUnit.ValueString()
		unitPath := path.Root("heartbeat").AtName("period_unit")

		seconds := valueWithUnitToSeconds(value, unit)

		if seconds < 30 {
			resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
				valuePath,
				fmt.Sprintf(`value in combination with %s must be greater than or equal to 30s`, unitPath.String()),
				fmt.Sprintf("%d %s", value, unit),
			))
		}

		if seconds > 3600*24*365 {
			resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
				valuePath,
				fmt.Sprintf(`value in combination with %s must be less than or equal to 365 days`, unitPath.String()),
				fmt.Sprintf("%d %s", value, unit),
			))
		}
	}

	if !heartbeatAttributeModel.Grace.IsUnknown() && !heartbeatAttributeModel.GraceUnit.IsUnknown() {
		value := heartbeatAttributeModel.Grace.ValueInt32()
		valuePath := path.Root("heartbeat").AtName("grace")

		unit := heartbeatAttributeModel.GraceUnit.ValueString()
		unitPath := path.Root("heartbeat").AtName("grace_unit")

		seconds := valueWithUnitToSeconds(value, unit)

		if seconds > 3600*24*365 {
			resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
				valuePath,
				fmt.Sprintf(`value in combination with %s must be less than or equal to 365 days`, unitPath.String()),
				fmt.Sprintf("%d %s", value, unit),
			))
		}
	}
}

func (r *HeartbeatResource) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	client, diags := interop.ClientFromProviderData(req.ProviderData)
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

	resp.Diagnostics.Append(plan.Refresh(ctx, realizedModel, interop.Created)...)
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

	resp.Diagnostics.Append(state.Refresh(ctx, realizedModel, interop.Loaded)...)
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

	resp.Diagnostics.Append(plan.Refresh(ctx, realizedModel, interop.Updated)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

var (
	_ interop.Model[checkly.HeartbeatCheck] = (*HeartbeatResourceModel)(nil)
	_ interop.Model[checkly.Heartbeat]      = (*HeartbeatAttributeModel)(nil)
)

type HeartbeatResourceModel struct {
	ID                        types.String `tfsdk:"id"`
	Name                      types.String `tfsdk:"name"`
	Activated                 types.Bool   `tfsdk:"activated"`
	Muted                     types.Bool   `tfsdk:"muted"`
	Tags                      types.Set    `tfsdk:"tags"`
	AlertSettings             types.Object `tfsdk:"alert_settings"`
	UseGlobalAlertSettings    types.Bool   `tfsdk:"use_global_alert_settings"`
	Heartbeat                 types.Object `tfsdk:"heartbeat"`
	AlertChannelSubscriptions types.List   `tfsdk:"alert_channel_subscription"`
}

func (m *HeartbeatResourceModel) Refresh(ctx context.Context, from *checkly.HeartbeatCheck, flags interop.RefreshFlags) diag.Diagnostics {
	var diags diag.Diagnostics

	if flags.Created() {
		m.ID = types.StringValue(from.ID)
	}

	m.Name = types.StringValue(from.Name)
	m.Activated = types.BoolValue(from.Activated)
	m.Muted = types.BoolValue(from.Muted)

	slices.Sort(from.Tags)
	m.Tags = interop.IntoUntypedStringSet(&from.Tags)

	m.AlertSettings, _, diags = attributes.AlertSettingsAttributeGluer.RefreshToObject(ctx, &from.AlertSettings, flags)
	if diags.HasError() {
		return diags
	}

	m.UseGlobalAlertSettings = types.BoolValue(from.UseGlobalAlertSettings)

	m.Heartbeat, _, diags = HeartbeatAttributeGluer.RefreshToObject(ctx, &from.Heartbeat, flags)
	if diags.HasError() {
		return diags
	}

	m.AlertChannelSubscriptions, _, diags = attributes.AlertChannelSubscriptionAttributeGluer.RefreshToList(ctx, &from.AlertChannelSubscriptions, flags)
	if diags.HasError() {
		return diags
	}

	return diags
}

func (m *HeartbeatResourceModel) Render(ctx context.Context, into *checkly.HeartbeatCheck) diag.Diagnostics {
	var diags diag.Diagnostics

	into.Name = m.Name.ValueString()
	into.Activated = m.Activated.ValueBool()
	into.Muted = m.Muted.ValueBool()
	into.Tags = interop.FromUntypedStringSet(m.Tags)

	into.AlertSettings, _, diags = attributes.AlertSettingsAttributeGluer.RenderFromObject(ctx, m.AlertSettings)
	if diags.HasError() {
		return diags
	}

	into.UseGlobalAlertSettings = m.UseGlobalAlertSettings.ValueBool()

	into.Heartbeat, _, diags = HeartbeatAttributeGluer.RenderFromObject(ctx, m.Heartbeat)
	if diags.HasError() {
		return diags
	}

	into.AlertChannelSubscriptions, _, diags = attributes.AlertChannelSubscriptionAttributeGluer.RenderFromList(ctx, m.AlertChannelSubscriptions)
	if diags.HasError() {
		return diags
	}

	return diags
}

var HeartbeatAttributeSchema = schema.SingleNestedAttribute{
	Required: true,
	Attributes: map[string]schema.Attribute{
		"period": schema.Int32Attribute{
			Description: "How often you expect a ping to the ping URL.",
			Required:    true,
		},
		"period_unit": schema.StringAttribute{
			Description: "Possible values `seconds`, `minutes`, `hours` and `days`.",
			Required:    true,
			Validators: []validator.String{
				stringvalidator.OneOf(
					"seconds",
					"minutes",
					"hours",
					"days",
				),
			},
		},
		"grace": schema.Int32Attribute{
			Description: "How long Checkly should wait before triggering any alerts when a ping does not arrive within the set period.",
			Required:    true,
		},
		"grace_unit": schema.StringAttribute{
			Description: "Possible values `seconds`, `minutes`, `hours` and `days`.",
			Required:    true,
			Validators: []validator.String{
				stringvalidator.OneOf(
					"seconds",
					"minutes",
					"hours",
					"days",
				),
			},
		},
		"ping_token": schema.StringAttribute{
			Description: "Custom token to generate your ping URL. Checkly will expect a ping to `https://ping.checklyhq.com/[PING_TOKEN]`.",
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
	},
}

type HeartbeatAttributeModel struct {
	Period     types.Int32  `tfsdk:"period"`
	PeriodUnit types.String `tfsdk:"period_unit"`
	Grace      types.Int32  `tfsdk:"grace"`
	GraceUnit  types.String `tfsdk:"grace_unit"`
	PingToken  types.String `tfsdk:"ping_token"`
}

var HeartbeatAttributeGluer = interop.GluerForSingleNestedAttribute[
	checkly.Heartbeat,
	HeartbeatAttributeModel,
](HeartbeatAttributeSchema)

func (m *HeartbeatAttributeModel) Refresh(ctx context.Context, from *checkly.Heartbeat, flags interop.RefreshFlags) diag.Diagnostics {
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
