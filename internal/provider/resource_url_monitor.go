package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
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

	"github.com/checkly/checkly-go-sdk/v2"
)

var urlMonitorAssertionSourceValues = allowedValues[string]{
	{Value: "STATUS_CODE"},
}

var urlMonitorAssertionComparisonValues = allowedValues[string]{
	{Value: "EQUALS"},
	{Value: "NOT_EQUALS"},
	{Value: "GREATER_THAN"},
	{Value: "LESS_THAN"},
}

var (
	_ resource.Resource                     = (*URLMonitorResource)(nil)
	_ resource.ResourceWithConfigure        = (*URLMonitorResource)(nil)
	_ resource.ResourceWithUpgradeState     = (*URLMonitorResource)(nil)
	_ resource.ResourceWithConfigValidators = (*URLMonitorResource)(nil)
	_ resource.ResourceWithImportState      = (*URLMonitorResource)(nil)
	_ resource.ResourceWithModifyPlan       = (*URLMonitorResource)(nil)
)

// NewURLMonitorResource returns a new checkly_url_monitor resource.
func NewURLMonitorResource() resource.Resource {
	return &URLMonitorResource{}
}

// URLMonitorResource implements the checkly_url_monitor resource.
type URLMonitorResource struct {
	client  checkly.Client
	account *accountLookup
}

// URLMonitorResourceModel describes the resource data model.
type URLMonitorResourceModel struct {
	ID                        types.String `tfsdk:"id"`
	Name                      types.String `tfsdk:"name"`
	Description               types.String `tfsdk:"description"`
	Frequency                 types.Int64  `tfsdk:"frequency"`
	FrequencyOffset           types.Int64  `tfsdk:"frequency_offset"`
	Activated                 types.Bool   `tfsdk:"activated"`
	Muted                     types.Bool   `tfsdk:"muted"`
	ShouldFail                types.Bool   `tfsdk:"should_fail"`
	RunParallel               types.Bool   `tfsdk:"run_parallel"`
	Locations                 types.Set    `tfsdk:"locations"`
	DegradedResponseTime      types.Int64  `tfsdk:"degraded_response_time"`
	MaxResponseTime           types.Int64  `tfsdk:"max_response_time"`
	Tags                      types.Set    `tfsdk:"tags"`
	AlertChannelSubscriptions types.Set    `tfsdk:"alert_channel_subscriptions"`
	PrivateLocations          types.Set    `tfsdk:"private_locations"`
	AlertSettings             types.Object `tfsdk:"alert_settings"`
	UseGlobalAlertSettings    types.Bool   `tfsdk:"use_global_alert_settings"`
	Request                   types.Object `tfsdk:"request"`
	GroupID                   types.Int64  `tfsdk:"group_id"`
	GroupOrder                types.Int64  `tfsdk:"group_order"`
	RetryStrategy             types.Object `tfsdk:"retry_strategy"`
	TriggerIncident           types.Object `tfsdk:"trigger_incident"`
}

// URLMonitorRequestModel describes the request attribute data model.
type URLMonitorRequestModel struct {
	URL             types.String `tfsdk:"url"`
	FollowRedirects types.Bool   `tfsdk:"follow_redirects"`
	SkipSSL         types.Bool   `tfsdk:"skip_ssl"`
	Assertions      types.Set    `tfsdk:"assertions"`
	IPFamily        types.String `tfsdk:"ip_family"`
}

var urlMonitorRequestAttrTypes = map[string]attr.Type{
	"url":              types.StringType,
	"follow_redirects": types.BoolType,
	"skip_ssl":         types.BoolType,
	"assertions":       types.SetType{ElemType: types.ObjectType{AttrTypes: assertionAttrTypes}},
	"ip_family":        types.StringType,
}

func (r *URLMonitorResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_url_monitor"
}

func (r *URLMonitorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     sdkv2SchemaVersion + 1,
		Description: "Creates a URL Monitor to check HTTP endpoint availability and response times.",
		Attributes: withCommonCheckAttributes(map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the monitor.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the monitor.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "A description of the monitor.",
			},
			frequencyAttributeName: makeFrequencyAttributeSchema(FrequencyAttributeSchemaOptions{
				Subject:            subjectMonitor,
				AllowHighFrequency: true,
			}),
			frequencyOffsetAttributeName: makeFrequencyOffsetAttributeSchema(FrequencyOffsetAttributeSchemaOptions{
				Subject: subjectMonitor,
			}),
			"degraded_response_time": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(3000),
				Description: "The response time in milliseconds where the monitor should be considered degraded. Possible values are between `0` and `30000`, or up to `45000` if extended limits have been approved for your account. Defaults to `3000`.",
				Validators: []validator.Int64{
					int64validator.Between(0, extendedResponseTimeLimit),
				},
			},
			"max_response_time": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(5000),
				Description: "The response time in milliseconds where the monitor should be considered failing. Possible values are between `0` and `30000`, or up to `45000` if extended limits have been approved for your account. Defaults to `5000`.",
				Validators: []validator.Int64{
					int64validator.Between(0, extendedResponseTimeLimit),
				},
			},
			triggerIncidentAttributeName: makeTriggerIncidentSchema(),
			alertSettingsAttributeName: makeAlertSettingsAttributeSchema(AlertSettingsAttributeSchemaOptions{
				Subject: subjectMonitor,
			}),
			retryStrategyAttributeName: makeRetryStrategyAttributeSchema(RetryStrategyAttributeSchemaOptions{
				SupportsOnlyOnNetworkError: true,
			}),
			"request": schema.SingleNestedAttribute{
				Required:    true,
				Description: "The parameters of the HTTP request.",
				Attributes: map[string]schema.Attribute{
					"url": schema.StringAttribute{
						Required:    true,
						Description: "The URL to monitor. Must be between `1` and `2048` characters. Must be a valid HTTP or HTTPS URL.",
						Validators: []validator.String{
							utf16LengthBetween(1, 2048),
						},
					},
					"follow_redirects": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(true),
						Description: "Whether to follow HTTP redirects automatically. Defaults to `true`.",
					},
					"skip_ssl": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
						Description: "Whether to skip SSL certificate verification. Defaults to `false`.",
					},
					"ip_family": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("IPv4"),
						Description: "IP family version to use for the connection. " + ipFamilyValues.String() + " Defaults to `IPv4`.",
						Validators: []validator.String{
							stringvalidator.OneOf(ipFamilyValues.Values()...),
						},
					},
					"assertions": schema.SetNestedAttribute{
						Optional:    true,
						Computed:    true,
						Default:     emptySetDefault(types.ObjectType{AttrTypes: assertionAttrTypes}),
						Description: "Assertions to validate the HTTP response. URL monitors only support status code assertions.",
						NestedObject: schema.NestedAttributeObject{
							// Attributes nested in this set attribute must
							// not use schema defaults. The framework looks
							// up configuration values for set elements by
							// full element value; once computed attributes
							// resolve, the lookup misses, the configuration
							// is treated as null, and defaults would
							// override values the practitioner actually
							// configured, producing a perpetual diff.
							// Default values are instead applied in
							// ModifyPlan.
							Attributes: map[string]schema.Attribute{
								"source": schema.StringAttribute{
									Required:    true,
									Description: "The source of the asserted value. " + urlMonitorAssertionSourceValues.String(),
									Validators: []validator.String{
										stringvalidator.OneOf(urlMonitorAssertionSourceValues.Values()...),
									},
								},
								"property": schema.StringAttribute{
									Optional:    true,
									Computed:    true,
									Description: "Not used by `STATUS_CODE`, the only allowed source; leave empty.",
								},
								"comparison": schema.StringAttribute{
									Required:    true,
									Description: "The type of comparison to be executed between expected and actual value of the assertion. " + urlMonitorAssertionComparisonValues.String(),
									Validators: []validator.String{
										stringvalidator.OneOf(urlMonitorAssertionComparisonValues.Values()...),
									},
								},
								"target": schema.StringAttribute{
									Required:    true,
									Description: "The target value. Typically `200` when the source is `STATUS_CODE`.",
								},
							},
						},
					},
				},
			},
			alertChannelSubscriptionAttributeName: makeAlertChannelSubscriptionSchema(AlertChannelSubscriptionAttributeSchemaOptions{
				Subject: subjectMonitor,
			}),
		}, CommonCheckAttributeSchemaOptions{Subject: subjectMonitor, ShouldFail: true, RunParallel: true, Locations: true, GroupMembership: true, ShouldFailNote: shouldFailErrorStatusNote}),
	}
}

func (r *URLMonitorResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		newFrequencyOffsetConfigValidator(),
	}
}

// UpgradeState upgrades state written by the provider's SDK v2
// implementation; see sdkv2StateUpgraders.
func (r *URLMonitorResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return sdkv2StateUpgraders(ctx, r)
}

func (r *URLMonitorResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
	r.account = accountFromProviderData(req.ProviderData)
}

// ModifyPlan applies default values to the optional+computed attributes of
// the request assertions. Schema defaults cannot be used for attributes
// nested in set elements (see the schema for details), so the plan is
// adjusted here instead. Substituting static values keeps every set element
// fully known and stable across plans, which set element correlation
// requires.
//
// It also checks the response time limits against the account, which needs
// the configured provider and so can't be done in ValidateConfig.
func (r *URLMonitorResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planNestedDefaults(ctx, req, resp, path.Root("request"), objectWithSetDefaults("assertions", assertionsWithPropertyDefault))
	if resp.Diagnostics.HasError() {
		return
	}

	r.account.validateResponseTimeLimits(ctx, req.Plan, httpStandardResponseTimeLimit, &resp.Diagnostics)
}

func (r *URLMonitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan URLMonitorResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	monitor := urlMonitorFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.CreateURLMonitor(apiCtx, monitor)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to Create URL Monitor", err)
		return
	}

	// State is written even when the conversion produces error diagnostics.
	// The monitor already exists remotely at this point, and returning
	// without saving state would leave it unmanaged.
	urlMonitorToModel(ctx, result, &plan, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *URLMonitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state URLMonitorResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	monitor, err := r.client.GetURLMonitor(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		addAPIError(&resp.Diagnostics, "Unable to Read URL Monitor", err)
		return
	}

	urlMonitorToModel(ctx, monitor, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *URLMonitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan URLMonitorResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	monitor := urlMonitorFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.UpdateURLMonitor(apiCtx, monitor.ID, monitor)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to Update URL Monitor", err)
		return
	}

	urlMonitorToModel(ctx, result, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *URLMonitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state URLMonitorResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	err := r.client.DeleteURLMonitor(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			return
		}
		addAPIError(&resp.Diagnostics, "Unable to Delete URL Monitor", err)
		return
	}
}

func (r *URLMonitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// urlMonitorFromModel converts the resource data model to its API
// representation.
func urlMonitorFromModel(ctx context.Context, model *URLMonitorResourceModel, diags *diag.Diagnostics) checkly.URLMonitor {
	// An empty description is not tracked by the Checkly API; send null
	// instead so that updates clear a previously configured description.
	var description *string
	if value := model.Description.ValueString(); value != "" {
		description = &value
	}

	// The locations property is serialized even when empty.
	locations := stringsFromSetOrEmpty(ctx, model.Locations, diags)

	privateLocations := stringsFromSetOrEmpty(ctx, model.PrivateLocations, diags)

	alertSettings := alertSettingsFromObject(ctx, model.AlertSettings, diags)

	return checkly.URLMonitor{
		ID:                        model.ID.ValueString(),
		Name:                      model.Name.ValueString(),
		Description:               description,
		Frequency:                 int(model.Frequency.ValueInt64()),
		FrequencyOffset:           int(model.FrequencyOffset.ValueInt64()),
		Activated:                 model.Activated.ValueBool(),
		Muted:                     model.Muted.ValueBool(),
		ShouldFail:                model.ShouldFail.ValueBool(),
		RunParallel:               model.RunParallel.ValueBool(),
		Locations:                 locations,
		DegradedResponseTime:      int(model.DegradedResponseTime.ValueInt64()),
		MaxResponseTime:           int(model.MaxResponseTime.ValueInt64()),
		Tags:                      stringsFromSet(ctx, model.Tags, diags),
		AlertSettings:             &alertSettings,
		UseGlobalAlertSettings:    model.UseGlobalAlertSettings.ValueBool(),
		Request:                   urlMonitorRequestFromObject(ctx, model.Request, diags),
		GroupID:                   model.GroupID.ValueInt64(),
		GroupOrder:                groupOrderToAPI(model.GroupOrder),
		AlertChannelSubscriptions: alertChannelSubscriptionsFromSet(ctx, model.AlertChannelSubscriptions, diags),
		PrivateLocations:          &privateLocations,
		RetryStrategy:             retryStrategyFromObject(ctx, model.RetryStrategy, diags),
		TriggerIncident:           triggerIncidentFromObject(ctx, model.TriggerIncident, diags),
	}
}

// urlMonitorToModel updates the model to match the API representation of the
// monitor. The values already present in the model (from the plan on
// create/update, from the state on read) are used to resolve fields that the
// API does not track faithfully.
func urlMonitorToModel(ctx context.Context, c *checkly.URLMonitor, model *URLMonitorResourceModel, diags *diag.Diagnostics) {
	model.ID = types.StringValue(c.ID)
	model.Name = types.StringValue(c.Name)

	// The attribute defaults to "", so a missing description maps to "".
	if c.Description != nil {
		model.Description = types.StringValue(*c.Description)
	} else {
		model.Description = types.StringValue("")
	}

	model.Frequency = types.Int64Value(int64(c.Frequency))
	model.FrequencyOffset = frequencyOffsetFromAPI(c.Frequency, c.FrequencyOffset, model.FrequencyOffset)

	model.Activated = types.BoolValue(c.Activated)
	model.Muted = types.BoolValue(c.Muted)
	model.ShouldFail = types.BoolValue(c.ShouldFail)
	model.RunParallel = types.BoolValue(c.RunParallel)
	model.Locations = setFromStrings(ctx, c.Locations, diags)
	model.DegradedResponseTime = types.Int64Value(int64(c.DegradedResponseTime))
	model.MaxResponseTime = types.Int64Value(int64(c.MaxResponseTime))
	model.Tags = setFromStrings(ctx, c.Tags, diags)

	model.AlertSettings = alertSettingsFromAPI(ctx, c.AlertSettings, model.AlertSettings, diags)

	model.UseGlobalAlertSettings = types.BoolValue(c.UseGlobalAlertSettings)
	model.Request = urlMonitorRequestToObject(ctx, c.Request, diags)
	model.GroupID = int64ValueOrNull(c.GroupID)
	model.GroupOrder = groupOrderFromAPI(c.GroupOrder, model.GroupOrder)

	var privateLocations []string
	if c.PrivateLocations != nil {
		privateLocations = *c.PrivateLocations
	}
	model.PrivateLocations = setFromStrings(ctx, privateLocations, diags)

	model.AlertChannelSubscriptions = alertChannelSubscriptionsToSet(ctx, c.AlertChannelSubscriptions, diags)
	model.RetryStrategy = retryStrategyToObject(ctx, c.RetryStrategy, model.RetryStrategy, diags)
	model.TriggerIncident = triggerIncidentToObject(ctx, c.TriggerIncident, diags)
}

// urlMonitorRequestFromObject converts the request attribute value to its
// API representation.
func urlMonitorRequestFromObject(ctx context.Context, obj types.Object, diags *diag.Diagnostics) checkly.URLRequest {
	model := objectAs[URLMonitorRequestModel](ctx, obj, diags)
	if model == nil {
		return checkly.URLRequest{}
	}

	return checkly.URLRequest{
		URL:             model.URL.ValueString(),
		FollowRedirects: model.FollowRedirects.ValueBool(),
		SkipSSL:         model.SkipSSL.ValueBool(),
		Assertions:      assertionsFromSet(ctx, model.Assertions, diags),
		IPFamily:        checkly.IPFamily(model.IPFamily.ValueString()),
	}
}

// urlMonitorRequestToObject converts the API representation of the request
// to the request attribute value.
func urlMonitorRequestToObject(ctx context.Context, r checkly.URLRequest, diags *diag.Diagnostics) types.Object {
	obj, d := types.ObjectValueFrom(ctx, urlMonitorRequestAttrTypes, URLMonitorRequestModel{
		URL:             types.StringValue(r.URL),
		FollowRedirects: types.BoolValue(r.FollowRedirects),
		SkipSSL:         types.BoolValue(r.SkipSSL),
		Assertions:      assertionsToSet(ctx, r.Assertions, diags),
		IPFamily:        types.StringValue(string(r.IPFamily)),
	})
	diags.Append(d...)

	return obj
}
