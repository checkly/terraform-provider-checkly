package provider

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/checkly/checkly-go-sdk/v2"
)

var (
	_ resource.Resource                     = (*AlertChannelResource)(nil)
	_ resource.ResourceWithConfigure        = (*AlertChannelResource)(nil)
	_ resource.ResourceWithUpgradeState     = (*AlertChannelResource)(nil)
	_ resource.ResourceWithConfigValidators = (*AlertChannelResource)(nil)
	_ resource.ResourceWithImportState      = (*AlertChannelResource)(nil)
)

// NewAlertChannelResource returns a new checkly_alert_channel resource.
func NewAlertChannelResource() resource.Resource {
	return &AlertChannelResource{}
}

// AlertChannelResource implements the checkly_alert_channel resource.
type AlertChannelResource struct {
	client checkly.Client
}

// AlertChannelResourceModel describes the resource data model.
type AlertChannelResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Email              types.Object `tfsdk:"email"`
	Slack              types.Object `tfsdk:"slack"`
	SlackApp           types.Object `tfsdk:"slack_app"`
	SMS                types.Object `tfsdk:"sms"`
	Call               types.Object `tfsdk:"call"`
	Webhook            types.Object `tfsdk:"webhook"`
	Opsgenie           types.Object `tfsdk:"opsgenie"`
	Pagerduty          types.Object `tfsdk:"pagerduty"`
	SendRecovery       types.Bool   `tfsdk:"send_recovery"`
	SendFailure        types.Bool   `tfsdk:"send_failure"`
	SendDegraded       types.Bool   `tfsdk:"send_degraded"`
	SSLExpiry          types.Bool   `tfsdk:"ssl_expiry"`
	SSLExpiryThreshold types.Int64  `tfsdk:"ssl_expiry_threshold"`
	AutoSubscribe      types.Bool   `tfsdk:"auto_subscribe"`
}

type alertChannelEmailModel struct {
	Address types.String `tfsdk:"address"`
}

var alertChannelEmailAttrTypes = map[string]attr.Type{
	"address": types.StringType,
}

type alertChannelSlackModel struct {
	URL     types.String `tfsdk:"url"`
	Channel types.String `tfsdk:"channel"`
}

var alertChannelSlackAttrTypes = map[string]attr.Type{
	"url":     types.StringType,
	"channel": types.StringType,
}

type alertChannelSlackAppModel struct {
	SlackChannels types.List `tfsdk:"slack_channels"`
}

var alertChannelSlackAppAttrTypes = map[string]attr.Type{
	"slack_channels": types.ListType{ElemType: types.StringType},
}

type alertChannelSMSModel struct {
	Name   types.String `tfsdk:"name"`
	Number types.String `tfsdk:"number"`
}

var alertChannelSMSAttrTypes = map[string]attr.Type{
	"name":   types.StringType,
	"number": types.StringType,
}

type alertChannelCallModel struct {
	Name   types.String `tfsdk:"name"`
	Number types.String `tfsdk:"number"`
}

var alertChannelCallAttrTypes = map[string]attr.Type{
	"name":   types.StringType,
	"number": types.StringType,
}

type alertChannelWebhookModel struct {
	Name            types.String `tfsdk:"name"`
	Method          types.String `tfsdk:"method"`
	Headers         types.Map    `tfsdk:"headers"`
	QueryParameters types.Map    `tfsdk:"query_parameters"`
	Template        types.String `tfsdk:"template"`
	URL             types.String `tfsdk:"url"`
	WebhookSecret   types.String `tfsdk:"webhook_secret"`
	WebhookType     types.String `tfsdk:"webhook_type"`
}

var alertChannelWebhookAttrTypes = map[string]attr.Type{
	"name":             types.StringType,
	"method":           types.StringType,
	"headers":          types.MapType{ElemType: types.StringType},
	"query_parameters": types.MapType{ElemType: types.StringType},
	"template":         types.StringType,
	"url":              types.StringType,
	"webhook_secret":   types.StringType,
	"webhook_type":     types.StringType,
}

type alertChannelOpsgenieModel struct {
	Name     types.String `tfsdk:"name"`
	APIKey   types.String `tfsdk:"api_key"`
	Region   types.String `tfsdk:"region"`
	Priority types.String `tfsdk:"priority"`
}

var alertChannelOpsgenieAttrTypes = map[string]attr.Type{
	"name":     types.StringType,
	"api_key":  types.StringType,
	"region":   types.StringType,
	"priority": types.StringType,
}

type alertChannelPagerdutyModel struct {
	Account     types.String `tfsdk:"account"`
	ServiceKey  types.String `tfsdk:"service_key"`
	ServiceName types.String `tfsdk:"service_name"`
}

var alertChannelPagerdutyAttrTypes = map[string]attr.Type{
	"account":      types.StringType,
	"service_key":  types.StringType,
	"service_name": types.StringType,
}

var alertChannelOpsgenieRegions = allowedValues[string]{
	{Value: string(checkly.OpsgenieRegionEU)},
	{Value: string(checkly.OpsgenieRegionUS)},
}

var alertChannelOpsgeniePriorities = allowedValues[string]{
	{Value: string(checkly.OpsgeniePriorityP1)},
	{Value: string(checkly.OpsgeniePriorityP2)},
	{Value: string(checkly.OpsgeniePriorityP3)},
	{Value: string(checkly.OpsgeniePriorityP4)},
	{Value: string(checkly.OpsgeniePriorityP5)},
}

var alertChannelWebhookTypes = allowedValues[string]{
	{Value: "WEBHOOK_CORALOGIX"},
	{Value: "WEBHOOK_DISCORD"},
	{Value: "WEBHOOK_FIREHYDRANT"},
	{Value: "WEBHOOK_GITLAB_ALERT"},
	{Value: "WEBHOOK_ILERT"},
	{Value: "WEBHOOK_INCIDENTIO"},
	{Value: "WEBHOOK_MSTEAMS"},
	{Value: "WEBHOOK_ROOTLY"},
	{Value: "WEBHOOK_SPIKESH"},
	{Value: "WEBHOOK_SPLUNK"},
	{Value: "WEBHOOK_TELEGRAM"},
}

// alertChannelSlackDestinationRegex mirrors the API's rule for a slack_app
// destination: a channel name that starts with "#", a user that starts with
// "@", or a whole Slack channel ID ("C" or "G" followed by at least eight
// uppercase letters or digits). The API drops an element that fails the rule
// without an error, so the provider refuses it instead. User and direct message
// IDs ("U…", "D…") are not destinations; a user is addressed by "@" handle.
var alertChannelSlackDestinationRegex = regexp.MustCompile(`^(?:[#@]|[CG][A-Z0-9]{8,}$)`)

func (r *AlertChannelResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_channel"
}

func (r *AlertChannelResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     sdkv2SchemaVersion + 1,
		Description: "Allows you to define alerting channels for the checks and groups in your account",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the alert channel.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"send_recovery": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether to send an alert when a check recovers. (Default `true`).",
			},
			"send_failure": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether to send an alert when a check fails. (Default `true`).",
			},
			"send_degraded": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether to send an alert when a check is degraded. (Default `false`).",
			},
			"ssl_expiry": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether to send an alert for expiring SSL certificates. (Default `false`).",
			},
			"ssl_expiry_threshold": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(30),
				Description: "How many days before an SSL certificate expires to start sending alerts, when `ssl_expiry` is enabled. Value must be between 1 and 30. (Default `30`).",
				Validators: []validator.Int64{
					int64validator.Between(1, 30),
				},
			},
			"auto_subscribe": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether new checks are subscribed to this alert channel automatically. Only the web app, which preselects the channel when you create a check there, and the Vercel integration, for the checks it creates, act on it. Checks created through the API, the Checkly CLI or Terraform are not subscribed, and neither are existing checks; subscribe them with `alert_channel_subscriptions`. (Default `false`).",
			},
			"email": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Sends alerts by email.",
				Attributes: map[string]schema.Attribute{
					"address": schema.StringAttribute{
						Required:    true,
						Description: "The email address of this email alert channel.",
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
				},
			},
			"slack": schema.SingleNestedAttribute{
				Optional:           true,
				Description:        "Sends alerts to a Slack channel through a legacy Slack incoming webhook.",
				DeprecationMessage: "The `slack` attribute uses the legacy Slack webhook integration and will be removed in a future major release. Use `slack_app` instead.",
				Attributes: map[string]schema.Attribute{
					"url": schema.StringAttribute{
						Required:    true,
						Sensitive:   true,
						Description: "The Slack webhook URL",
					},
					"channel": schema.StringAttribute{
						Required:    true,
						Description: "The name of the alert's Slack channel",
					},
				},
			},
			"slack_app": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Sends alerts to Slack channels or users through the Checkly Slack app.",
				Attributes: map[string]schema.Attribute{
					"slack_channels": schema.ListAttribute{
						Required:    true,
						ElementType: types.StringType,
						Description: "The Slack channels or users to notify, e.g. `[\"#ops\", \"@John\", \"C0123456789\"]`. Each element starts with `#` for a channel or `@` for a user, or is a Slack channel ID, such as `C0123456789`, and has no trailing whitespace. A private channel can be given by its ID.",
						Validators: []validator.List{
							listvalidator.SizeAtLeast(1),
							listvalidator.ValueStringsAre(
								stringvalidator.RegexMatches(alertChannelSlackDestinationRegex, "must start with # for a channel, such as #operations, or @ for a user, such as @john, or be a channel ID, such as C0123456789"),
								trimmedStringValidator{},
							),
						},
					},
				},
			},
			"sms": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Sends alerts as SMS messages.",
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
				Optional:    true,
				Description: "Sends alerts as phone calls.",
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
				Optional:    true,
				Description: "Sends alerts as HTTP requests to a webhook.",
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required:    true,
						Description: "The name of this alert channel.",
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
					"method": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("POST"),
						Description: "The HTTP method of the webhook request, such as `GET`, `POST`, `PUT`, `HEAD`, `DELETE` or `PATCH`. Must not be empty. (Default `POST`).",
						Validators: []validator.String{
							// The API stores any method as given and the
							// webhook sender passes it on, so only an empty
							// one is rejected: the API would store POST
							// instead.
							stringvalidator.LengthAtLeast(1),
						},
					},
					"headers": schema.MapAttribute{
						Optional:    true,
						Computed:    true,
						Default:     emptyMapDefault(types.StringType),
						ElementType: types.StringType,
						Description: "HTTP headers to send with the webhook request, as a map of header names to values. The values are not sensitive and show in plan output. To keep a credential in a header out of plan output, take it from a variable declared with `sensitive = true` or wrap it in `sensitive(...)`.",
					},
					"query_parameters": schema.MapAttribute{
						Optional:    true,
						Computed:    true,
						Default:     emptyMapDefault(types.StringType),
						ElementType: types.StringType,
						Description: "Query parameters to add to the webhook URL, as a map of parameter names to values. The values are not sensitive and show in plan output. To keep a credential in a parameter out of plan output, take it from a variable declared with `sensitive = true` or wrap it in `sensitive(...)`.",
					},
					"template": schema.StringAttribute{
						Optional:    true,
						Description: "The body of the webhook request, which can contain template variables such as `{{ALERT_TITLE}}` and `{{RESULT_LINK}}`. When not set, Checkly sends its default JSON payload.",
					},
					"url": schema.StringAttribute{
						Required:    true,
						Sensitive:   true,
						Description: "The URL to send the webhook request to.",
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
					"webhook_secret": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "A secret to sign webhook requests with. When set, Checkly sends an HMAC-SHA256 signature of the request body, keyed with this secret, in the `x-checkly-signature` header.",
					},
					"webhook_type": schema.StringAttribute{
						Optional:    true,
						Description: "Type of the webhook. " + alertChannelWebhookTypes.String(),
						Validators: []validator.String{
							stringvalidator.OneOf(alertChannelWebhookTypes.Values()...),
						},
					},
				},
			},
			"opsgenie": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Creates alerts in Opsgenie.",
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required:    true,
						Description: "The name of this alert channel.",
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
					"api_key": schema.StringAttribute{
						Required:    true,
						Sensitive:   true,
						Description: "The API key of the Opsgenie integration to create alerts with.",
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
					"region": schema.StringAttribute{
						Required:    true,
						Description: "The region of the Opsgenie account, which selects the Opsgenie API to call. " + alertChannelOpsgenieRegions.String(),
						Validators: []validator.String{
							stringvalidator.OneOf(alertChannelOpsgenieRegions.Values()...),
						},
					},
					"priority": schema.StringAttribute{
						Required:    true,
						Description: "The priority of the alerts created in Opsgenie. " + alertChannelOpsgeniePriorities.String(),
						Validators: []validator.String{
							stringvalidator.OneOf(alertChannelOpsgeniePriorities.Values()...),
						},
					},
				},
			},
			"pagerduty": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Sends alerts to a PagerDuty service.",
				Attributes: map[string]schema.Attribute{
					"service_key": schema.StringAttribute{
						Required:    true,
						Sensitive:   true,
						Description: "The integration key of the PagerDuty service to send alerts to.",
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
					"service_name": schema.StringAttribute{
						Optional:    true,
						Description: "The name of the PagerDuty service.",
					},
					"account": schema.StringAttribute{
						Optional:    true,
						Description: "The name of the PagerDuty account.",
					},
				},
			},
		},
	}
}

// ConfigValidators requires exactly one channel type attribute, because the
// Checkly API stores a single channel configuration per alert channel.
func (r *AlertChannelResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("email"),
			path.MatchRoot("slack"),
			path.MatchRoot("slack_app"),
			path.MatchRoot("sms"),
			path.MatchRoot("call"),
			path.MatchRoot("webhook"),
			path.MatchRoot("opsgenie"),
			path.MatchRoot("pagerduty"),
		),
	}
}

// UpgradeState upgrades state written by the provider's SDK v2
// implementation; see sdkv2StateUpgraders.
func (r *AlertChannelResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return sdkv2StateUpgraders(ctx, r)
}

func (r *AlertChannelResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *AlertChannelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AlertChannelResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ac := alertChannelFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.CreateAlertChannel(apiCtx, ac)
	if err != nil {
		addAlertChannelWriteError(&resp.Diagnostics, "Unable to Create Alert Channel", err)
		return
	}

	// Every other attribute is known in the plan (optional+computed
	// attributes are defaulted by the schema), so only the ID needs to be
	// filled in from the API response.
	plan.ID = types.StringValue(encodeNumericID(result.ID))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// addAlertChannelWriteError adds the diagnostic for a failed create or
// update. A Slack App channel that the account's Slack workspace cannot
// serve gets an error on slack_app that says how to fix it; the SDK's text
// for these errors is meant for users and leaves out the request. Other
// errors get the generic API error under summary.
func addAlertChannelWriteError(diags *diag.Diagnostics, summary string, err error) {
	var notConnected *checkly.SlackIntegrationNotConnectedError
	var unresolvable *checkly.UnresolvableSlackHandlesError
	switch {
	case errors.As(err, &notConnected):
		diags.AddAttributeError(path.Root("slack_app"), "Slack Workspace Not Connected", notConnected.Error())
	case errors.As(err, &unresolvable):
		diags.AddAttributeError(path.Root("slack_app").AtName("slack_channels"), "Unresolvable Slack Handles", unresolvable.Error())
	default:
		addAPIError(diags, summary, err)
	}
}

func (r *AlertChannelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AlertChannelResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := decodeNumericID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Alert Channel ID",
			fmt.Sprintf("Alert channel ID must be numeric: %s", err),
		)
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	ac, err := r.client.GetAlertChannel(apiCtx, id)
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		addAPIError(&resp.Diagnostics, "Unable to Read Alert Channel", err)
		return
	}

	alertChannelToModel(ctx, ac, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AlertChannelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AlertChannelResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := decodeNumericID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Alert Channel ID",
			fmt.Sprintf("Alert channel ID must be numeric: %s", err),
		)
		return
	}

	ac := alertChannelFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	ac.ID = id

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	_, err = r.client.UpdateAlertChannel(apiCtx, id, ac)
	if err != nil {
		addAlertChannelWriteError(&resp.Diagnostics, "Unable to Update Alert Channel", err)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AlertChannelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AlertChannelResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := decodeNumericID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Alert Channel ID",
			fmt.Sprintf("Alert channel ID must be numeric: %s", err),
		)
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	err = r.client.DeleteAlertChannel(apiCtx, id)
	if err != nil {
		if isNotFoundError(err) {
			return
		}
		addAPIError(&resp.Diagnostics, "Unable to Delete Alert Channel", err)
		return
	}
}

func (r *AlertChannelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// alertChannelFromModel converts the resource model to its API
// representation.
func alertChannelFromModel(ctx context.Context, model *AlertChannelResourceModel, diags *diag.Diagnostics) checkly.AlertChannel {
	ac := checkly.AlertChannel{
		SendRecovery: model.SendRecovery.ValueBoolPointer(),
		SendFailure:  model.SendFailure.ValueBoolPointer(),
		SendDegraded: model.SendDegraded.ValueBoolPointer(),
		SSLExpiry:    model.SSLExpiry.ValueBoolPointer(),
		// The API resets an autoSubscribe that a request leaves out to
		// false, so it is always sent.
		AutoSubscribe: model.AutoSubscribe.ValueBoolPointer(),
	}

	if !model.SSLExpiryThreshold.IsNull() && !model.SSLExpiryThreshold.IsUnknown() {
		threshold := int(model.SSLExpiryThreshold.ValueInt64())
		ac.SSLExpiryThreshold = &threshold
	}

	if m := objectAs[alertChannelEmailModel](ctx, model.Email, diags); m != nil {
		ac.Type = checkly.AlertChannelTypeEmail
		ac.Email = &checkly.AlertChannelEmail{
			Address: m.Address.ValueString(),
		}
	}

	if m := objectAs[alertChannelSlackModel](ctx, model.Slack, diags); m != nil {
		ac.Type = checkly.AlertChannelTypeSlack
		ac.Slack = &checkly.AlertChannelSlack{
			WebhookURL: m.URL.ValueString(),
			Channel:    m.Channel.ValueString(),
		}
	}

	if m := objectAs[alertChannelSlackAppModel](ctx, model.SlackApp, diags); m != nil {
		ac.Type = checkly.AlertChannelTypeSlackApp
		ac.SlackApp = &checkly.AlertChannelSlackApp{
			SlackChannels: stringsFromList(ctx, m.SlackChannels, diags),
		}
	}

	if m := objectAs[alertChannelSMSModel](ctx, model.SMS, diags); m != nil {
		ac.Type = checkly.AlertChannelTypeSMS
		ac.SMS = &checkly.AlertChannelSMS{
			Name:   m.Name.ValueString(),
			Number: m.Number.ValueString(),
		}
	}

	if m := objectAs[alertChannelCallModel](ctx, model.Call, diags); m != nil {
		ac.Type = checkly.AlertChannelTypeCall
		ac.CALL = &checkly.AlertChannelCall{
			Name:   m.Name.ValueString(),
			Number: m.Number.ValueString(),
		}
	}

	if m := objectAs[alertChannelWebhookModel](ctx, model.Webhook, diags); m != nil {
		ac.Type = checkly.AlertChannelTypeWebhook
		ac.Webhook = &checkly.AlertChannelWebhook{
			Name:            m.Name.ValueString(),
			Method:          m.Method.ValueString(),
			Template:        m.Template.ValueString(),
			URL:             m.URL.ValueString(),
			WebhookSecret:   m.WebhookSecret.ValueString(),
			WebhookType:     m.WebhookType.ValueString(),
			Headers:         keyValuesFromStringMap(ctx, m.Headers, diags),
			QueryParameters: keyValuesFromStringMap(ctx, m.QueryParameters, diags),
		}
	}

	if m := objectAs[alertChannelOpsgenieModel](ctx, model.Opsgenie, diags); m != nil {
		ac.Type = checkly.AlertChannelTypeOpsgenie
		ac.Opsgenie = &checkly.AlertChannelOpsgenie{
			Name:     m.Name.ValueString(),
			APIKey:   m.APIKey.ValueString(),
			Region:   checkly.OpsgenieRegion(m.Region.ValueString()),
			Priority: checkly.OpsgeniePriority(m.Priority.ValueString()),
		}
	}

	if m := objectAs[alertChannelPagerdutyModel](ctx, model.Pagerduty, diags); m != nil {
		ac.Type = checkly.AlertChannelTypePagerduty
		ac.Pagerduty = &checkly.AlertChannelPagerduty{
			Account:     m.Account.ValueString(),
			ServiceKey:  m.ServiceKey.ValueString(),
			ServiceName: m.ServiceName.ValueString(),
		}
	}

	return ac
}

// alertChannelToModel updates the model to match the API representation of
// the alert channel. The send_* and ssl_expiry* fields are only updated when
// the API tracks a value for them, preserving the prior state otherwise.
func alertChannelToModel(ctx context.Context, ac *checkly.AlertChannel, model *AlertChannelResourceModel, diags *diag.Diagnostics) {
	model.ID = types.StringValue(encodeNumericID(ac.ID))
	model.Email = alertChannelEmailToObject(ctx, ac.Email, diags)
	model.Slack = alertChannelSlackToObject(ctx, ac.Slack, diags)
	model.SlackApp = alertChannelSlackAppToObject(ctx, ac.SlackApp, diags)
	model.SMS = alertChannelSMSToObject(ctx, ac.SMS, diags)
	model.Call = alertChannelCallToObject(ctx, ac.CALL, diags)
	model.Webhook = alertChannelWebhookToObject(ctx, ac.Webhook, model.Webhook, diags)
	model.Opsgenie = alertChannelOpsgenieToObject(ctx, ac.Opsgenie, diags)
	model.Pagerduty = alertChannelPagerdutyToObject(ctx, ac.Pagerduty, model.Pagerduty, diags)

	if ac.SendRecovery != nil {
		model.SendRecovery = types.BoolValue(*ac.SendRecovery)
	}
	if ac.SendFailure != nil {
		model.SendFailure = types.BoolValue(*ac.SendFailure)
	}
	if ac.SendDegraded != nil {
		model.SendDegraded = types.BoolValue(*ac.SendDegraded)
	}
	if ac.SSLExpiry != nil {
		model.SSLExpiry = types.BoolValue(*ac.SSLExpiry)
	}
	if ac.SSLExpiryThreshold != nil {
		model.SSLExpiryThreshold = types.Int64Value(int64(*ac.SSLExpiryThreshold))
	}
	if ac.AutoSubscribe != nil {
		model.AutoSubscribe = types.BoolValue(*ac.AutoSubscribe)
	}
}

// alertChannelObjectFrom converts a channel type model to its attribute
// value.
func alertChannelObjectFrom(ctx context.Context, attrTypes map[string]attr.Type, model any, diags *diag.Diagnostics) types.Object {
	obj, d := types.ObjectValueFrom(ctx, attrTypes, model)
	diags.Append(d...)

	return obj
}

func alertChannelEmailToObject(ctx context.Context, cfg *checkly.AlertChannelEmail, diags *diag.Diagnostics) types.Object {
	if cfg == nil {
		return types.ObjectNull(alertChannelEmailAttrTypes)
	}

	return alertChannelObjectFrom(ctx, alertChannelEmailAttrTypes, alertChannelEmailModel{
		Address: types.StringValue(cfg.Address),
	}, diags)
}

func alertChannelSlackToObject(ctx context.Context, cfg *checkly.AlertChannelSlack, diags *diag.Diagnostics) types.Object {
	if cfg == nil {
		return types.ObjectNull(alertChannelSlackAttrTypes)
	}

	return alertChannelObjectFrom(ctx, alertChannelSlackAttrTypes, alertChannelSlackModel{
		URL:     types.StringValue(cfg.WebhookURL),
		Channel: types.StringValue(cfg.Channel),
	}, diags)
}

func alertChannelSlackAppToObject(ctx context.Context, cfg *checkly.AlertChannelSlackApp, diags *diag.Diagnostics) types.Object {
	if cfg == nil {
		return types.ObjectNull(alertChannelSlackAppAttrTypes)
	}

	return alertChannelObjectFrom(ctx, alertChannelSlackAppAttrTypes, alertChannelSlackAppModel{
		SlackChannels: listFromStrings(ctx, cfg.SlackChannels, diags),
	}, diags)
}

func alertChannelSMSToObject(ctx context.Context, cfg *checkly.AlertChannelSMS, diags *diag.Diagnostics) types.Object {
	if cfg == nil {
		return types.ObjectNull(alertChannelSMSAttrTypes)
	}

	return alertChannelObjectFrom(ctx, alertChannelSMSAttrTypes, alertChannelSMSModel{
		Name:   types.StringValue(cfg.Name),
		Number: types.StringValue(cfg.Number),
	}, diags)
}

func alertChannelCallToObject(ctx context.Context, cfg *checkly.AlertChannelCall, diags *diag.Diagnostics) types.Object {
	if cfg == nil {
		return types.ObjectNull(alertChannelCallAttrTypes)
	}

	return alertChannelObjectFrom(ctx, alertChannelCallAttrTypes, alertChannelCallModel{
		Name:   types.StringValue(cfg.Name),
		Number: types.StringValue(cfg.Number),
	}, diags)
}

func alertChannelWebhookToObject(ctx context.Context, cfg *checkly.AlertChannelWebhook, prior types.Object, diags *diag.Diagnostics) types.Object {
	if cfg == nil {
		return types.ObjectNull(alertChannelWebhookAttrTypes)
	}

	var priorModel alertChannelWebhookModel
	alertChannelPriorObject(ctx, prior, &priorModel)

	return alertChannelObjectFrom(ctx, alertChannelWebhookAttrTypes, alertChannelWebhookModel{
		Name:            types.StringValue(cfg.Name),
		Method:          types.StringValue(cfg.Method),
		Headers:         stringMapFromKeyValues(ctx, cfg.Headers, diags),
		QueryParameters: stringMapFromKeyValues(ctx, cfg.QueryParameters, diags),
		// The API returns "" for optional string fields that are not set.
		Template:      stringValueOrPriorEmpty(cfg.Template, priorModel.Template),
		URL:           types.StringValue(cfg.URL),
		WebhookSecret: stringValueOrPriorEmpty(cfg.WebhookSecret, priorModel.WebhookSecret),
		WebhookType:   stringValueOrPriorEmpty(cfg.WebhookType, priorModel.WebhookType),
	}, diags)
}

func alertChannelOpsgenieToObject(ctx context.Context, cfg *checkly.AlertChannelOpsgenie, diags *diag.Diagnostics) types.Object {
	if cfg == nil {
		return types.ObjectNull(alertChannelOpsgenieAttrTypes)
	}

	return alertChannelObjectFrom(ctx, alertChannelOpsgenieAttrTypes, alertChannelOpsgenieModel{
		Name:     types.StringValue(cfg.Name),
		APIKey:   types.StringValue(cfg.APIKey),
		Region:   types.StringValue(string(cfg.Region)),
		Priority: types.StringValue(string(cfg.Priority)),
	}, diags)
}

func alertChannelPagerdutyToObject(ctx context.Context, cfg *checkly.AlertChannelPagerduty, prior types.Object, diags *diag.Diagnostics) types.Object {
	if cfg == nil {
		return types.ObjectNull(alertChannelPagerdutyAttrTypes)
	}

	var priorModel alertChannelPagerdutyModel
	alertChannelPriorObject(ctx, prior, &priorModel)

	return alertChannelObjectFrom(ctx, alertChannelPagerdutyAttrTypes, alertChannelPagerdutyModel{
		// The API returns "" for optional string fields that are not set.
		Account:     stringValueOrPriorEmpty(cfg.Account, priorModel.Account),
		ServiceKey:  types.StringValue(cfg.ServiceKey),
		ServiceName: stringValueOrPriorEmpty(cfg.ServiceName, priorModel.ServiceName),
	}, diags)
}

// alertChannelPriorObject decodes a prior channel type attribute value into
// target, leaving target zero-valued (with null attributes) when there is no
// prior value.
func alertChannelPriorObject(ctx context.Context, prior types.Object, target any) {
	if prior.IsNull() || prior.IsUnknown() {
		return
	}

	// A prior value that fails to decode is no worse than none at all.
	_ = prior.As(ctx, target, basetypes.ObjectAsOptions{})
}
