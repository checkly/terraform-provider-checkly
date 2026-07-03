package provider

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
)

var (
	_ resource.Resource                     = (*AlertChannelResource)(nil)
	_ resource.ResourceWithConfigure        = (*AlertChannelResource)(nil)
	_ resource.ResourceWithConfigValidators = (*AlertChannelResource)(nil)
	_ resource.ResourceWithImportState      = (*AlertChannelResource)(nil)
	_ resource.ResourceWithModifyPlan       = (*AlertChannelResource)(nil)
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
	Email              types.Set    `tfsdk:"email"`
	Slack              types.Set    `tfsdk:"slack"`
	SlackApp           types.Set    `tfsdk:"slack_app"`
	SMS                types.Set    `tfsdk:"sms"`
	Call               types.Set    `tfsdk:"call"`
	Webhook            types.Set    `tfsdk:"webhook"`
	Opsgenie           types.Set    `tfsdk:"opsgenie"`
	Pagerduty          types.Set    `tfsdk:"pagerduty"`
	SendRecovery       types.Bool   `tfsdk:"send_recovery"`
	SendFailure        types.Bool   `tfsdk:"send_failure"`
	SendDegraded       types.Bool   `tfsdk:"send_degraded"`
	SSLExpiry          types.Bool   `tfsdk:"ssl_expiry"`
	SSLExpiryThreshold types.Int64  `tfsdk:"ssl_expiry_threshold"`
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

// alertChannelTypeBlockNames lists the schema blocks that each configure one
// alert channel type. Exactly one of them must be present in a configuration.
var alertChannelTypeBlockNames = []string{
	"email",
	"slack",
	"slack_app",
	"sms",
	"call",
	"webhook",
	"opsgenie",
	"pagerduty",
}

func (r *AlertChannelResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_channel"
}

func (r *AlertChannelResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
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
			"ssl_expiry_threshold": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(30),
				Description: "Value must be between 1 and 30 (Default `30`)",
				Validators: []validator.Int64{
					int64validator.Between(1, 30),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"email": schema.SetNestedBlock{
				Validators: []validator.Set{
					setvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"address": schema.StringAttribute{
							Required:    true,
							Description: "The email address of this email alert channel.",
						},
					},
				},
			},
			"slack": schema.SetNestedBlock{
				DeprecationMessage: "The `slack` block uses the legacy Slack webhook integration and will be removed in a future major release. Use `slack_app` instead.",
				Validators: []validator.Set{
					setvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
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
			},
			"slack_app": schema.SetNestedBlock{
				Validators: []validator.Set{
					setvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"slack_channels": schema.ListAttribute{
							Required:    true,
							ElementType: types.StringType,
							Description: "The Slack channels or users to notify, e.g. `[\"#ops\", \"@John\"]`.",
							Validators: []validator.List{
								listvalidator.SizeAtLeast(1),
							},
						},
					},
				},
			},
			"sms": schema.SetNestedBlock{
				Validators: []validator.Set{
					setvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
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
			},
			"call": schema.SetNestedBlock{
				Validators: []validator.Set{
					setvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
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
			},
			"webhook": schema.SetNestedBlock{
				Validators: []validator.Set{
					setvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					// Attributes nested in this set block must not use schema
					// defaults. The framework looks up configuration values
					// for set elements by full element value; once computed
					// attributes resolve, the lookup misses, the configuration
					// is treated as null, and defaults would override values
					// the practitioner actually configured, producing a
					// perpetual diff. Default values are instead applied in
					// ModifyPlan.
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required: true,
						},
						"method": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "(Default `POST`)",
						},
						"headers": schema.MapAttribute{
							Optional:    true,
							Computed:    true,
							ElementType: types.StringType,
						},
						"query_parameters": schema.MapAttribute{
							Optional:    true,
							Computed:    true,
							ElementType: types.StringType,
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
							Optional:    true,
							Description: "Type of the webhook. " + alertChannelWebhookTypes.String(),
							Validators: []validator.String{
								stringvalidator.OneOf(alertChannelWebhookTypes.Values()...),
							},
						},
					},
				},
			},
			"opsgenie": schema.SetNestedBlock{
				Validators: []validator.Set{
					setvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
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
			},
			"pagerduty": schema.SetNestedBlock{
				Validators: []validator.Set{
					setvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
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
			},
		},
	}
}

func (r *AlertChannelResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		alertChannelExactlyOneChannelValidator{},
	}
}

// alertChannelExactlyOneChannelValidator validates that a configuration
// contains exactly one channel type block. The Checkly API stores a single
// channel configuration per alert channel.
type alertChannelExactlyOneChannelValidator struct{}

func (v alertChannelExactlyOneChannelValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("Exactly one of these blocks must be configured: %s.", strings.Join(alertChannelTypeBlockNames, ", "))
}

func (v alertChannelExactlyOneChannelValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v alertChannelExactlyOneChannelValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var configured []string
	unknown := false

	for _, name := range alertChannelTypeBlockNames {
		var set types.Set

		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(name), &set)...)
		if resp.Diagnostics.HasError() {
			return
		}

		if set.IsUnknown() {
			unknown = true
			continue
		}

		if !set.IsNull() && len(set.Elements()) > 0 {
			configured = append(configured, name)
		}
	}

	if len(configured) > 1 {
		resp.Diagnostics.AddError(
			"Invalid Alert Channel Configuration",
			fmt.Sprintf("An alert channel cannot contain more than one channel type, but these are configured: %s.", strings.Join(configured, ", ")),
		)
		return
	}

	if len(configured) == 0 && !unknown {
		resp.Diagnostics.AddError(
			"Invalid Alert Channel Configuration",
			fmt.Sprintf("An alert channel requires exactly one channel type block: %s.", strings.Join(alertChannelTypeBlockNames, ", ")),
		)
	}
}

func (r *AlertChannelResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

// ModifyPlan applies default values to the optional+computed attributes
// nested in the webhook set block. Schema defaults cannot be used for these
// attributes (see the schema for details), so the plan is adjusted here
// instead. Substituting static values keeps every set element fully known
// and stable across plans, which set element correlation requires.
func (r *AlertChannelResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// A destroy plan has no planned state to default values in.
	if req.Plan.Raw.IsNull() {
		return
	}

	var configWebhook types.Set
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("webhook"), &configWebhook)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// When the configuration contains unknown values (references to
	// attributes of resources that have not been created yet), null values
	// cannot be distinguished from values that will resolve at apply time.
	// Leave the plan untouched; the apply-time re-plan applies the defaults
	// once the configuration values are known.
	if valueContainsUnknown(configWebhook) {
		return
	}

	var webhook types.Set
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("webhook"), &webhook)...)
	if resp.Diagnostics.HasError() {
		return
	}

	webhook = alertChannelWebhookWithDefaults(ctx, webhook, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("webhook"), webhook)...)
}

// alertChannelWebhookWithDefaults returns the webhook block value with
// default values substituted for null and unknown optional+computed
// attributes.
func alertChannelWebhookWithDefaults(ctx context.Context, set types.Set, diags *diag.Diagnostics) types.Set {
	if set.IsNull() || set.IsUnknown() {
		return set
	}

	var models []alertChannelWebhookModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return set
	}

	for i, model := range models {
		models[i].Method = stringOrDefault(model.Method, "POST")
		models[i].Headers = mapOrEmpty(model.Headers, types.StringType)
		models[i].QueryParameters = mapOrEmpty(model.QueryParameters, types.StringType)
	}

	result, d := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: alertChannelWebhookAttrTypes}, models)
	diags.Append(d...)

	return result
}

// mapOrEmpty returns the given value, or an empty map when the value is null
// or unknown. It is used to apply default values during plan modification to
// attributes nested in set blocks, where schema defaults cannot be used.
func mapOrEmpty(value types.Map, elementType attr.Type) types.Map {
	if value.IsNull() || value.IsUnknown() {
		return types.MapValueMust(elementType, map[string]attr.Value{})
	}
	return value
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
		resp.Diagnostics.AddError(
			"Unable to Create Alert Channel",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	// Every other attribute is known in the plan (optional+computed
	// attributes are defaulted by the schema or in ModifyPlan), so only the
	// ID needs to be filled in from the API response.
	plan.ID = types.StringValue(encodeNumericID(result.ID))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
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
		resp.Diagnostics.AddError(
			"Unable to Read Alert Channel",
			fmt.Sprintf("Checkly API error: %s", err),
		)
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
		resp.Diagnostics.AddError(
			"Unable to Update Alert Channel",
			fmt.Sprintf("Checkly API error: %s", err),
		)
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
		resp.Diagnostics.AddError(
			"Unable to Delete Alert Channel",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}
}

func (r *AlertChannelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// alertChannelFirstElement extracts the single element of a MaxItems:1
// channel type block into its model type. It returns nil when the block is
// not configured.
func alertChannelFirstElement[T any](ctx context.Context, set types.Set, diags *diag.Diagnostics) *T {
	if set.IsNull() || set.IsUnknown() || len(set.Elements()) == 0 {
		return nil
	}

	var models []T
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() || len(models) == 0 {
		return nil
	}

	return &models[0]
}

// alertChannelFromModel converts the resource model to its API
// representation.
func alertChannelFromModel(ctx context.Context, model *AlertChannelResourceModel, diags *diag.Diagnostics) checkly.AlertChannel {
	ac := checkly.AlertChannel{
		SendRecovery: model.SendRecovery.ValueBoolPointer(),
		SendFailure:  model.SendFailure.ValueBoolPointer(),
		SendDegraded: model.SendDegraded.ValueBoolPointer(),
		SSLExpiry:    model.SSLExpiry.ValueBoolPointer(),
	}

	if !model.SSLExpiryThreshold.IsNull() && !model.SSLExpiryThreshold.IsUnknown() {
		threshold := int(model.SSLExpiryThreshold.ValueInt64())
		ac.SSLExpiryThreshold = &threshold
	}

	if m := alertChannelFirstElement[alertChannelEmailModel](ctx, model.Email, diags); m != nil {
		ac.Type = checkly.AlertTypeEmail
		ac.Email = &checkly.AlertChannelEmail{
			Address: m.Address.ValueString(),
		}
	}

	if m := alertChannelFirstElement[alertChannelSlackModel](ctx, model.Slack, diags); m != nil {
		ac.Type = checkly.AlertTypeSlack
		ac.Slack = &checkly.AlertChannelSlack{
			WebhookURL: m.URL.ValueString(),
			Channel:    m.Channel.ValueString(),
		}
	}

	if m := alertChannelFirstElement[alertChannelSlackAppModel](ctx, model.SlackApp, diags); m != nil {
		ac.Type = checkly.AlertTypeSlackApp
		ac.SlackApp = &checkly.AlertChannelSlackApp{
			SlackChannels: stringsFromList(ctx, m.SlackChannels, diags),
		}
	}

	if m := alertChannelFirstElement[alertChannelSMSModel](ctx, model.SMS, diags); m != nil {
		ac.Type = checkly.AlertTypeSMS
		ac.SMS = &checkly.AlertChannelSMS{
			Name:   m.Name.ValueString(),
			Number: m.Number.ValueString(),
		}
	}

	if m := alertChannelFirstElement[alertChannelCallModel](ctx, model.Call, diags); m != nil {
		ac.Type = checkly.AlertTypeCall
		ac.CALL = &checkly.AlertChannelCall{
			Name:   m.Name.ValueString(),
			Number: m.Number.ValueString(),
		}
	}

	if m := alertChannelFirstElement[alertChannelWebhookModel](ctx, model.Webhook, diags); m != nil {
		ac.Type = checkly.AlertTypeWebhook
		ac.Webhook = &checkly.AlertChannelWebhook{
			Name:            m.Name.ValueString(),
			Method:          m.Method.ValueString(),
			Template:        m.Template.ValueString(),
			URL:             m.URL.ValueString(),
			WebhookSecret:   m.WebhookSecret.ValueString(),
			WebhookType:     m.WebhookType.ValueString(),
			Headers:         alertChannelKeyValuesFromMap(ctx, m.Headers, diags),
			QueryParameters: alertChannelKeyValuesFromMap(ctx, m.QueryParameters, diags),
		}
	}

	if m := alertChannelFirstElement[alertChannelOpsgenieModel](ctx, model.Opsgenie, diags); m != nil {
		ac.Type = checkly.AlertTypeOpsgenie
		ac.Opsgenie = &checkly.AlertChannelOpsgenie{
			Name:     m.Name.ValueString(),
			APIKey:   m.APIKey.ValueString(),
			Region:   m.Region.ValueString(),
			Priority: m.Priority.ValueString(),
		}
	}

	if m := alertChannelFirstElement[alertChannelPagerdutyModel](ctx, model.Pagerduty, diags); m != nil {
		ac.Type = checkly.AlertTypePagerduty
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
	model.Email = alertChannelEmailToSet(ctx, ac.Email, diags)
	model.Slack = alertChannelSlackToSet(ctx, ac.Slack, diags)
	model.SlackApp = alertChannelSlackAppToSet(ctx, ac.SlackApp, diags)
	model.SMS = alertChannelSMSToSet(ctx, ac.SMS, diags)
	model.Call = alertChannelCallToSet(ctx, ac.CALL, diags)
	model.Webhook = alertChannelWebhookToSet(ctx, ac.Webhook, diags)
	model.Opsgenie = alertChannelOpsgenieToSet(ctx, ac.Opsgenie, diags)
	model.Pagerduty = alertChannelPagerdutyToSet(ctx, ac.Pagerduty, diags)

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
}

func alertChannelEmailToSet(ctx context.Context, cfg *checkly.AlertChannelEmail, diags *diag.Diagnostics) types.Set {
	elemType := types.ObjectType{AttrTypes: alertChannelEmailAttrTypes}
	if cfg == nil {
		return types.SetNull(elemType)
	}

	set, d := types.SetValueFrom(ctx, elemType, []alertChannelEmailModel{
		{
			Address: types.StringValue(cfg.Address),
		},
	})
	diags.Append(d...)

	return set
}

func alertChannelSlackToSet(ctx context.Context, cfg *checkly.AlertChannelSlack, diags *diag.Diagnostics) types.Set {
	elemType := types.ObjectType{AttrTypes: alertChannelSlackAttrTypes}
	if cfg == nil {
		return types.SetNull(elemType)
	}

	set, d := types.SetValueFrom(ctx, elemType, []alertChannelSlackModel{
		{
			URL:     types.StringValue(cfg.WebhookURL),
			Channel: types.StringValue(cfg.Channel),
		},
	})
	diags.Append(d...)

	return set
}

func alertChannelSlackAppToSet(ctx context.Context, cfg *checkly.AlertChannelSlackApp, diags *diag.Diagnostics) types.Set {
	elemType := types.ObjectType{AttrTypes: alertChannelSlackAppAttrTypes}
	if cfg == nil {
		return types.SetNull(elemType)
	}

	set, d := types.SetValueFrom(ctx, elemType, []alertChannelSlackAppModel{
		{
			SlackChannels: listFromStrings(ctx, cfg.SlackChannels, diags),
		},
	})
	diags.Append(d...)

	return set
}

func alertChannelSMSToSet(ctx context.Context, cfg *checkly.AlertChannelSMS, diags *diag.Diagnostics) types.Set {
	elemType := types.ObjectType{AttrTypes: alertChannelSMSAttrTypes}
	if cfg == nil {
		return types.SetNull(elemType)
	}

	set, d := types.SetValueFrom(ctx, elemType, []alertChannelSMSModel{
		{
			Name:   types.StringValue(cfg.Name),
			Number: types.StringValue(cfg.Number),
		},
	})
	diags.Append(d...)

	return set
}

func alertChannelCallToSet(ctx context.Context, cfg *checkly.AlertChannelCall, diags *diag.Diagnostics) types.Set {
	elemType := types.ObjectType{AttrTypes: alertChannelCallAttrTypes}
	if cfg == nil {
		return types.SetNull(elemType)
	}

	set, d := types.SetValueFrom(ctx, elemType, []alertChannelCallModel{
		{
			Name:   types.StringValue(cfg.Name),
			Number: types.StringValue(cfg.Number),
		},
	})
	diags.Append(d...)

	return set
}

func alertChannelWebhookToSet(ctx context.Context, cfg *checkly.AlertChannelWebhook, diags *diag.Diagnostics) types.Set {
	elemType := types.ObjectType{AttrTypes: alertChannelWebhookAttrTypes}
	if cfg == nil {
		return types.SetNull(elemType)
	}

	set, d := types.SetValueFrom(ctx, elemType, []alertChannelWebhookModel{
		{
			Name:            types.StringValue(cfg.Name),
			Method:          types.StringValue(cfg.Method),
			Headers:         alertChannelMapFromKeyValues(ctx, cfg.Headers, diags),
			QueryParameters: alertChannelMapFromKeyValues(ctx, cfg.QueryParameters, diags),
			// The API returns "" for optional string fields that are not
			// set. Unset optional attributes are null, so map "" back to
			// null to keep refresh consistent with the configuration.
			Template:      stringValueOrNull(cfg.Template),
			URL:           types.StringValue(cfg.URL),
			WebhookSecret: stringValueOrNull(cfg.WebhookSecret),
			WebhookType:   stringValueOrNull(cfg.WebhookType),
		},
	})
	diags.Append(d...)

	return set
}

func alertChannelOpsgenieToSet(ctx context.Context, cfg *checkly.AlertChannelOpsgenie, diags *diag.Diagnostics) types.Set {
	elemType := types.ObjectType{AttrTypes: alertChannelOpsgenieAttrTypes}
	if cfg == nil {
		return types.SetNull(elemType)
	}

	set, d := types.SetValueFrom(ctx, elemType, []alertChannelOpsgenieModel{
		{
			Name:     types.StringValue(cfg.Name),
			APIKey:   types.StringValue(cfg.APIKey),
			Region:   types.StringValue(cfg.Region),
			Priority: types.StringValue(cfg.Priority),
		},
	})
	diags.Append(d...)

	return set
}

func alertChannelPagerdutyToSet(ctx context.Context, cfg *checkly.AlertChannelPagerduty, diags *diag.Diagnostics) types.Set {
	elemType := types.ObjectType{AttrTypes: alertChannelPagerdutyAttrTypes}
	if cfg == nil {
		return types.SetNull(elemType)
	}

	set, d := types.SetValueFrom(ctx, elemType, []alertChannelPagerdutyModel{
		{
			// The API returns "" for optional string fields that are not
			// set. Unset optional attributes are null, so map "" back to
			// null to keep refresh consistent with the configuration.
			Account:     stringValueOrNull(cfg.Account),
			ServiceKey:  types.StringValue(cfg.ServiceKey),
			ServiceName: stringValueOrNull(cfg.ServiceName),
		},
	})
	diags.Append(d...)

	return set
}

// alertChannelKeyValuesFromMap converts a map of strings to the API's
// key-value pair representation, in deterministic key order.
func alertChannelKeyValuesFromMap(ctx context.Context, m types.Map, diags *diag.Diagnostics) []checkly.KeyValue {
	kvs := []checkly.KeyValue{}

	if m.IsNull() || m.IsUnknown() {
		return kvs
	}

	var values map[string]string
	diags.Append(m.ElementsAs(ctx, &values, false)...)
	if diags.HasError() {
		return kvs
	}

	for _, key := range slices.Sorted(maps.Keys(values)) {
		kvs = append(kvs, checkly.KeyValue{
			Key:   key,
			Value: values[key],
		})
	}

	return kvs
}

// alertChannelMapFromKeyValues converts the API's key-value pair
// representation to a map of strings.
func alertChannelMapFromKeyValues(ctx context.Context, kvs []checkly.KeyValue, diags *diag.Diagnostics) types.Map {
	values := map[string]string{}
	for _, kv := range kvs {
		values[kv.Key] = kv.Value
	}

	m, d := types.MapValueFrom(ctx, types.StringType, values)
	diags.Append(d...)

	return m
}
