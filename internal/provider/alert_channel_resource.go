package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/checkly/terraform-provider-checkly/internal/sdkutil"
)

var (
	_ resource.Resource                     = (*AlertChannelResource)(nil)
	_ resource.ResourceWithConfigure        = (*AlertChannelResource)(nil)
	_ resource.ResourceWithImportState      = (*AlertChannelResource)(nil)
	_ resource.ResourceWithConfigValidators = (*AlertChannelResource)(nil)
)

type AlertChannelResource struct {
	client checkly.Client
}

func NewAlertChannelResource() resource.Resource {
	return &AlertChannelResource{}
}

func (r *AlertChannelResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_alert_channel"
}

func (r *AlertChannelResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	// TODO: Investigate UpgradeState's potential ability to allow prior
	// 1-length Sets and Lists to be seamlessly converted to
	// SingleNestedAttributes.
	resp.Schema = schema.Schema{
		Description: "Allows you to define alerting channels for the checks and groups in your account.",
		Attributes: map[string]schema.Attribute{
			"id":           IDResourceAttributeSchema,
			"last_updated": LastUpdatedAttributeSchema,
			"email": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"address": schema.StringAttribute{
						Required:    true,
						Description: "The email address of this email alert channel.",
					},
				},
			},
			"slack": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"url": schema.StringAttribute{
						Required:    true,
						Description: "The Slack webhook URL",
					},
					"channel": schema.StringAttribute{
						Required:    true,
						Description: "The name of the alert's Slack channel",
					},
				},
			},
			"sms": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required:    true,
						Description: "The name of this alert channel",
					},
					"number": schema.StringAttribute{
						Required:    true,
						Description: "The mobile number to receive the alerts",
					},
				},
			},
			"call": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required:    true,
						Description: "The name of this alert channel",
					},
					"number": schema.StringAttribute{
						Required:    true,
						Description: "The mobile number to receive the alerts",
					},
				},
			},
			"webhook": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required: true,
					},
					"method": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("POST"),
						Description: "(Default `POST`)",
					},
					"headers": schema.MapAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
					},
					"query_parameters": schema.MapAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
					},
					"template": schema.StringAttribute{
						Optional: true,
					},
					"url": schema.StringAttribute{
						Required: true,
					},
					"webhook_secret": schema.StringAttribute{
						Optional: true,
					},
					"webhook_type": schema.StringAttribute{
						Optional: true,
						Validators: []validator.String{
							stringvalidator.OneOf(
								"WEBHOOK_DISCORD",
								"WEBHOOK_FIREHYDRANT",
								"WEBHOOK_GITLAB_ALERT",
								"WEBHOOK_SPIKESH",
								"WEBHOOK_SPLUNK",
								"WEBHOOK_MSTEAMS",
								"WEBHOOK_TELEGRAM",
							),
						},
						Description: "Type of the webhook. Possible values are 'WEBHOOK_DISCORD', 'WEBHOOK_FIREHYDRANT', 'WEBHOOK_GITLAB_ALERT', 'WEBHOOK_SPIKESH', 'WEBHOOK_SPLUNK', 'WEBHOOK_MSTEAMS' and 'WEBHOOK_TELEGRAM'.",
					},
				},
			},
			"opsgenie": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required: true,
					},
					"api_key": schema.StringAttribute{
						Required: true,
					},
					"region": schema.StringAttribute{
						Required: true,
					},
					"priority": schema.StringAttribute{
						Required: true,
					},
				},
			},
			"pagerduty": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"service_key": schema.StringAttribute{
						Required: true,
					},
					"service_name": schema.StringAttribute{
						Optional: true,
					},
					"account": schema.StringAttribute{
						Optional: true,
					},
				},
			},
			"send_recovery": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "(Default `true`)",
			},
			"send_failure": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "(Default `true`)",
			},
			"send_degraded": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "(Default `false`)",
			},
			"ssl_expiry": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "(Default `false`)",
			},
			"ssl_expiry_threshold": schema.Int32Attribute{
				Optional: true,
				Computed: true,
				Default:  int32default.StaticInt32(30),
				Validators: []validator.Int32{
					int32validator.Between(1, 30),
				},
				Description: "Value must be between 1 and 30 (Default `30`)",
			},
		},
	}
}

func (r *AlertChannelResource) ConfigValidators(
	ctx context.Context,
) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("email"),
			path.MatchRoot("slack"),
			path.MatchRoot("sms"),
			path.MatchRoot("call"),
			path.MatchRoot("webhook"),
			path.MatchRoot("opsgenie"),
			path.MatchRoot("pagerduty"),
		),
	}
}

func (r *AlertChannelResource) Configure(
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

func (r *AlertChannelResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *AlertChannelResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan AlertChannelResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var desiredModel checkly.AlertChannel
	resp.Diagnostics.Append(plan.Render(ctx, &desiredModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realizedModel, err := r.client.CreateAlertChannel(ctx, desiredModel)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Checkly Alert Channel",
			fmt.Sprintf("Could not create alert channel, unexpected error: %s", err),
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

func (r *AlertChannelResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state AlertChannelResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, diags := AlertChannelID.FromString(state.ID)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	err := r.client.DeleteAlertChannel(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Checkly Alert Channel",
			fmt.Sprintf("Could not delete alert channel, unexpected error: %s", err),
		)

		return
	}
}

func (r *AlertChannelResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state AlertChannelResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, diags := AlertChannelID.FromString(state.ID)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	realizedModel, err := r.client.GetAlertChannel(ctx, id)
	if err != nil {
		if sdkutil.IsHTTPNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading Checkly Alert Channel",
			fmt.Sprintf("Could not retrieve alert channel, unexpected error: %s", err),
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

func (r *AlertChannelResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan AlertChannelResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, diags := AlertChannelID.FromString(plan.ID)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	var desiredModel checkly.AlertChannel
	resp.Diagnostics.Append(plan.Render(ctx, &desiredModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realizedModel, err := r.client.UpdateAlertChannel(
		ctx,
		id,
		desiredModel,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Checkly Alert Channel",
			fmt.Sprintf("Could not update alert channel, unexpected error: %s", err),
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

var AlertChannelID = sdkutil.Identifier{
	Path:  path.Root("id"),
	Title: "Checkly Alert Channel ID",
}

var (
	_ ResourceModel[checkly.AlertChannel]          = (*AlertChannelResourceModel)(nil)
	_ ResourceModel[checkly.AlertChannelEmail]     = (*EmailAttributeModel)(nil)
	_ ResourceModel[checkly.AlertChannelSlack]     = (*SlackAttributeModel)(nil)
	_ ResourceModel[checkly.AlertChannelSMS]       = (*SMSAttributeModel)(nil)
	_ ResourceModel[checkly.AlertChannelCall]      = (*CallAttributeModel)(nil)
	_ ResourceModel[checkly.AlertChannelWebhook]   = (*WebhookAttributeModel)(nil)
	_ ResourceModel[checkly.AlertChannelOpsgenie]  = (*OpsgenieAttributeModel)(nil)
	_ ResourceModel[checkly.AlertChannelPagerduty] = (*PagerdutyAttributeModel)(nil)
)

type AlertChannelResourceModel struct {
	ID                 types.String             `tfsdk:"id"`
	LastUpdated        types.String             `tfsdk:"last_updated"` // FIXME: Keep this? Old code did not have it.
	Email              *EmailAttributeModel     `tfsdk:"email"`
	Slack              *SlackAttributeModel     `tfsdk:"slack"`
	SMS                *SMSAttributeModel       `tfsdk:"sms"`
	Call               *CallAttributeModel      `tfsdk:"call"`
	Webhook            *WebhookAttributeModel   `tfsdk:"webhook"`
	Opsgenie           *OpsgenieAttributeModel  `tfsdk:"opsgenie"`
	Pagerduty          *PagerdutyAttributeModel `tfsdk:"pagerduty"`
	SendRecovery       types.Bool               `tfsdk:"send_recovery"`
	SendFailure        types.Bool               `tfsdk:"send_failure"`
	SendDegraded       types.Bool               `tfsdk:"send_degraded"`
	SSLExpiry          types.Bool               `tfsdk:"ssl_expiry"`
	SSLExpiryThreshold types.Int32              `tfsdk:"ssl_expiry_threshold"`
}

func (m *AlertChannelResourceModel) Refresh(ctx context.Context, from *checkly.AlertChannel, flags RefreshFlags) diag.Diagnostics {
	var diags diag.Diagnostics

	if flags.Created() {
		m.ID = AlertChannelID.IntoString(from.ID)
	}

	if flags.Created() || flags.Updated() {
		m.LastUpdated = LastUpdatedNow()
	}

	m.Email = nil
	m.Slack = nil
	m.SMS = nil
	m.Call = nil
	m.Webhook = nil
	m.Opsgenie = nil
	m.Pagerduty = nil

	switch from.Type {
	case checkly.AlertTypeEmail:
		m.Email = new(EmailAttributeModel)
		diags.Append(m.Email.Refresh(ctx, from.Email, flags)...)
	case checkly.AlertTypeSlack:
		m.Slack = new(SlackAttributeModel)
		diags.Append(m.Slack.Refresh(ctx, from.Slack, flags)...)
	case checkly.AlertTypeSMS:
		m.SMS = new(SMSAttributeModel)
		diags.Append(m.SMS.Refresh(ctx, from.SMS, flags)...)
	case checkly.AlertTypeCall:
		m.Call = new(CallAttributeModel)
		diags.Append(m.Call.Refresh(ctx, from.CALL, flags)...)
	case checkly.AlertTypeWebhook:
		m.Webhook = new(WebhookAttributeModel)
		diags.Append(m.Webhook.Refresh(ctx, from.Webhook, flags)...)
	case checkly.AlertTypeOpsgenie:
		m.Opsgenie = new(OpsgenieAttributeModel)
		diags.Append(m.Opsgenie.Refresh(ctx, from.Opsgenie, flags)...)
	case checkly.AlertTypePagerduty:
		m.Pagerduty = new(PagerdutyAttributeModel)
		diags.Append(m.Pagerduty.Refresh(ctx, from.Pagerduty, flags)...)
	default:
		// TODO diags
	}

	if from.SendRecovery != nil {
		m.SendRecovery = types.BoolValue(*from.SendRecovery)
	} else {
		m.SendRecovery = types.BoolNull()
	}

	if from.SendFailure != nil {
		m.SendFailure = types.BoolValue(*from.SendFailure)
	} else {
		m.SendFailure = types.BoolNull()
	}

	if from.SendDegraded != nil {
		m.SendDegraded = types.BoolValue(*from.SendDegraded)
	} else {
		m.SendDegraded = types.BoolNull()
	}

	if from.SSLExpiry != nil {
		m.SSLExpiry = types.BoolValue(*from.SSLExpiry)
	} else {
		m.SSLExpiry = types.BoolNull()
	}

	if from.SSLExpiryThreshold != nil {
		m.SSLExpiryThreshold = types.Int32Value(int32(*from.SSLExpiryThreshold))
	} else {
		m.SSLExpiryThreshold = types.Int32Null()
	}

	return nil
}

func (m *AlertChannelResourceModel) Render(ctx context.Context, into *checkly.AlertChannel) diag.Diagnostics {
	var diags diag.Diagnostics

	switch {
	case m.Email != nil:
		into.Type = checkly.AlertTypeEmail
		into.Email = new(checkly.AlertChannelEmail)
		diags.Append(m.Email.Render(ctx, into.Email)...)
	case m.Slack != nil:
		into.Type = checkly.AlertTypeSlack
		into.Slack = new(checkly.AlertChannelSlack)
		diags.Append(m.Slack.Render(ctx, into.Slack)...)
	case m.SMS != nil:
		into.Type = checkly.AlertTypeSMS
		into.SMS = new(checkly.AlertChannelSMS)
		diags.Append(m.SMS.Render(ctx, into.SMS)...)
	case m.Call != nil:
		into.Type = checkly.AlertTypeCall
		into.CALL = new(checkly.AlertChannelCall)
		diags.Append(m.Call.Render(ctx, into.CALL)...)
	case m.Opsgenie != nil:
		into.Type = checkly.AlertTypeOpsgenie
		into.Opsgenie = new(checkly.AlertChannelOpsgenie)
		diags.Append(m.Opsgenie.Render(ctx, into.Opsgenie)...)
	case m.Webhook != nil:
		into.Type = checkly.AlertTypeWebhook
		into.Webhook = new(checkly.AlertChannelWebhook)
		diags.Append(m.Webhook.Render(ctx, into.Webhook)...)
	case m.Pagerduty != nil:
		into.Type = checkly.AlertTypePagerduty
		into.Pagerduty = new(checkly.AlertChannelPagerduty)
		diags.Append(m.Pagerduty.Render(ctx, into.Pagerduty)...)
	default:
		// TODO: Use diags instead
		panic("bug: impossible AlertChannelResourceModel state: no type set")
	}

	into.SendRecovery = m.SendRecovery.ValueBoolPointer()
	into.SendFailure = m.SendFailure.ValueBoolPointer()
	into.SendDegraded = m.SendDegraded.ValueBoolPointer()
	into.SSLExpiry = m.SSLExpiry.ValueBoolPointer()

	if !m.SSLExpiryThreshold.IsNull() {
		value := int(m.SSLExpiryThreshold.ValueInt32())
		into.SSLExpiryThreshold = &value
	}

	return diags
}

type EmailAttributeModel struct {
	Address types.String `tfsdk:"address"`
}

func (m *EmailAttributeModel) Refresh(ctx context.Context, from *checkly.AlertChannelEmail, flags RefreshFlags) diag.Diagnostics {
	m.Address = types.StringValue(from.Address)

	return nil
}

func (m *EmailAttributeModel) Render(ctx context.Context, into *checkly.AlertChannelEmail) diag.Diagnostics {
	into.Address = m.Address.ValueString()

	return nil
}

type SlackAttributeModel struct {
	URL     types.String `tfsdk:"url"`
	Channel types.String `tfsdk:"channel"`
}

func (m *SlackAttributeModel) Refresh(ctx context.Context, from *checkly.AlertChannelSlack, flags RefreshFlags) diag.Diagnostics {
	m.URL = types.StringValue(from.WebhookURL)
	m.Channel = types.StringValue(from.Channel)

	return nil
}

func (m *SlackAttributeModel) Render(ctx context.Context, into *checkly.AlertChannelSlack) diag.Diagnostics {
	into.WebhookURL = m.URL.ValueString()
	into.Channel = m.Channel.ValueString()

	return nil
}

type SMSAttributeModel struct {
	Name   types.String `tfsdk:"name"`
	Number types.String `tfsdk:"number"`
}

func (m *SMSAttributeModel) Refresh(ctx context.Context, from *checkly.AlertChannelSMS, flags RefreshFlags) diag.Diagnostics {
	m.Name = types.StringValue(from.Name)
	m.Number = types.StringValue(from.Number)

	return nil
}

func (m *SMSAttributeModel) Render(ctx context.Context, into *checkly.AlertChannelSMS) diag.Diagnostics {
	into.Name = m.Name.ValueString()
	into.Number = m.Number.ValueString()

	return nil
}

type CallAttributeModel struct {
	Name   types.String `tfsdk:"name"`
	Number types.String `tfsdk:"number"`
}

func (m *CallAttributeModel) Refresh(ctx context.Context, from *checkly.AlertChannelCall, flags RefreshFlags) diag.Diagnostics {
	m.Name = types.StringValue(from.Name)
	m.Number = types.StringValue(from.Number)

	return nil
}

func (m *CallAttributeModel) Render(ctx context.Context, into *checkly.AlertChannelCall) diag.Diagnostics {
	into.Name = m.Name.ValueString()
	into.Number = m.Number.ValueString()

	return nil
}

type WebhookAttributeModel struct {
	Name            types.String `tfsdk:"name"`
	Method          types.String `tfsdk:"method"`
	Headers         types.Map    `tfsdk:"headers"`
	QueryParameters types.Map    `tfsdk:"query_parameters"`
	Template        types.String `tfsdk:"template"`
	URL             types.String `tfsdk:"url"`
	WebhookSecret   types.String `tfsdk:"webhook_secret"`
	WebhookType     types.String `tfsdk:"webhook_type"`
}

func (m *WebhookAttributeModel) Refresh(ctx context.Context, from *checkly.AlertChannelWebhook, flags RefreshFlags) diag.Diagnostics {
	m.Name = types.StringValue(from.Name)
	m.Method = types.StringValue(from.Method)
	m.Headers = sdkutil.KeyValuesIntoMap(&from.Headers)
	m.QueryParameters = sdkutil.KeyValuesIntoMap(&from.QueryParameters)
	m.Template = types.StringValue(from.Template)
	m.URL = types.StringValue(from.URL)
	m.WebhookSecret = types.StringValue(from.WebhookSecret)
	m.WebhookType = types.StringValue(from.WebhookType)

	return nil
}

func (m *WebhookAttributeModel) Render(ctx context.Context, into *checkly.AlertChannelWebhook) diag.Diagnostics {
	into.Name = m.Name.ValueString()
	into.Method = m.Method.ValueString()
	into.Headers = sdkutil.KeyValuesFromMap(m.Headers)
	into.QueryParameters = sdkutil.KeyValuesFromMap(m.QueryParameters)
	into.Template = m.Template.ValueString()
	into.URL = m.URL.ValueString()
	into.WebhookSecret = m.WebhookSecret.ValueString()
	into.WebhookType = m.WebhookType.ValueString()

	return nil
}

type OpsgenieAttributeModel struct {
	Name     types.String `tfsdk:"name"`
	APIKey   types.String `tfsdk:"api_key"`
	Region   types.String `tfsdk:"region"`
	Priority types.String `tfsdk:"priority"`
}

func (m *OpsgenieAttributeModel) Refresh(ctx context.Context, from *checkly.AlertChannelOpsgenie, flags RefreshFlags) diag.Diagnostics {
	m.Name = types.StringValue(from.Name)
	m.APIKey = types.StringValue(from.APIKey)
	m.Region = types.StringValue(from.Region)
	m.Priority = types.StringValue(from.Priority)

	return nil
}

func (m *OpsgenieAttributeModel) Render(ctx context.Context, into *checkly.AlertChannelOpsgenie) diag.Diagnostics {
	into.Name = m.Name.ValueString()
	into.APIKey = m.APIKey.ValueString()
	into.Region = m.Region.ValueString()
	into.Priority = m.Priority.ValueString()

	return nil
}

type PagerdutyAttributeModel struct {
	ServiceKey  types.String `tfsdk:"service_key"`
	ServiceName types.String `tfsdk:"service_name"`
	Account     types.String `tfsdk:"account"`
}

func (m *PagerdutyAttributeModel) Refresh(ctx context.Context, from *checkly.AlertChannelPagerduty, flags RefreshFlags) diag.Diagnostics {
	m.ServiceKey = types.StringValue(from.ServiceKey)
	m.ServiceName = types.StringValue(from.ServiceName)
	m.Account = types.StringValue(from.Account)

	return nil
}

func (m *PagerdutyAttributeModel) Render(ctx context.Context, into *checkly.AlertChannelPagerduty) diag.Diagnostics {
	into.ServiceKey = m.ServiceKey.ValueString()
	into.ServiceName = m.ServiceName.ValueString()
	into.Account = m.Account.ValueString()

	return nil
}
