package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
)

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
	client checkly.Client
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
	AlertChannelSubscriptions types.Set    `tfsdk:"alert_channel_subscription"`
	PrivateLocations          types.Set    `tfsdk:"private_locations"`
	AlertSettings             types.Object `tfsdk:"alert_settings"`
	UseGlobalAlertSettings    types.Bool   `tfsdk:"use_global_alert_settings"`
	Request                   types.Set    `tfsdk:"request"`
	GroupID                   types.Int64  `tfsdk:"group_id"`
	GroupOrder                types.Int64  `tfsdk:"group_order"`
	RetryStrategy             types.Object `tfsdk:"retry_strategy"`
	TriggerIncident           types.Set    `tfsdk:"trigger_incident"`
}

// URLMonitorRequestModel describes the request block data model.
type URLMonitorRequestModel struct {
	URL             types.String `tfsdk:"url"`
	FollowRedirects types.Bool   `tfsdk:"follow_redirects"`
	SkipSSL         types.Bool   `tfsdk:"skip_ssl"`
	Assertions      types.Set    `tfsdk:"assertion"`
	IPFamily        types.String `tfsdk:"ip_family"`
}

// URLMonitorAssertionModel describes the request.assertion block data model.
type URLMonitorAssertionModel struct {
	Source     types.String `tfsdk:"source"`
	Property   types.String `tfsdk:"property"`
	Comparison types.String `tfsdk:"comparison"`
	Target     types.String `tfsdk:"target"`
}

var urlMonitorAssertionAttrTypes = map[string]attr.Type{
	"source":     types.StringType,
	"property":   types.StringType,
	"comparison": types.StringType,
	"target":     types.StringType,
}

var urlMonitorRequestAttrTypes = map[string]attr.Type{
	"url":              types.StringType,
	"follow_redirects": types.BoolType,
	"skip_ssl":         types.BoolType,
	"assertion":        types.SetType{ElemType: types.ObjectType{AttrTypes: urlMonitorAssertionAttrTypes}},
	"ip_family":        types.StringType,
}

func (r *URLMonitorResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_url_monitor"
}

func (r *URLMonitorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     sdkv2SchemaVersion + 1,
		Description: "Creates a URL Monitor to check HTTP endpoint availability and response times.",
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
				Description: "The name of the monitor.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "A description of the monitor.",
			},
			frequencyAttributeName: makeFrequencyAttributeSchema(FrequencyAttributeSchemaOptions{
				Monitor:            true,
				AllowHighFrequency: true,
			}),
			frequencyOffsetAttributeName: makeFrequencyOffsetAttributeSchema(FrequencyOffsetAttributeSchemaOptions{
				Monitor: true,
			}),
			"activated": schema.BoolAttribute{
				Required:    true,
				Description: "Determines whether the monitor will run periodically or not after being deployed.",
			},
			"muted": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Determines if any notifications will be sent out when the monitor fails and/or recovers. (Default `false`).",
			},
			"should_fail": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Allows to invert the behaviour of when the monitor is considered to fail. (Default `false`).",
			},
			"run_parallel": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Determines whether the monitor should run on all selected locations in parallel or round-robin. (Default `false`).",
			},
			"locations": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "An array of one or more data center locations where to run the this monitor.",
			},
			"degraded_response_time": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(3000),
				Description: "The response time in milliseconds where the monitor should be considered degraded. Possible values are between `0` and `30000`. (Default `3000`).",
				Validators: []validator.Int64{
					int64validator.Between(0, 30000),
				},
			},
			"max_response_time": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(5000),
				Description: "The response time in milliseconds where the monitor should be considered failing. Possible values are between `0` and `30000`. (Default `5000`).",
				Validators: []validator.Int64{
					int64validator.Between(0, 30000),
				},
			},
			"tags": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "A list of tags for organizing and filtering checks and monitors.",
			},
			"private_locations": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "An array of one or more private locations slugs.",
			},
			alertSettingsAttributeName: makeAlertSettingsAttributeSchema(AlertSettingsAttributeSchemaOptions{
				Monitor: true,
			}),
			"use_global_alert_settings": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "When true, the account level alert settings will be used, not the alert setting defined on this monitor. (Default `true`).",
			},
			"group_id": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "The ID of the check group that this monitor is part of.",
			},
			"group_order": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "The position of the monitor in the check group. It determines in what order checks and monitors are run when a group is triggered from the API or from CI/CD.",
			},
			retryStrategyAttributeName: makeRetryStrategyAttributeSchema(RetryStrategyAttributeSchemaOptions{
				SupportsOnlyOnNetworkError: true,
			}),
		},
		Blocks: map[string]schema.Block{
			alertChannelSubscriptionAttributeName: makeAlertChannelSubscriptionSchema(AlertChannelSubscriptionAttributeSchemaOptions{
				Monitor: true,
			}),
			"request": schema.SetNestedBlock{
				Description: "The parameters of the HTTP request.",
				Validators: []validator.Set{
					setvalidator.IsRequired(),
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
						"url": schema.StringAttribute{
							Required:    true,
							Description: "The URL to monitor. Must be a valid HTTP or HTTPS URL.",
						},
						"follow_redirects": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Whether to follow HTTP redirects automatically. (Default `true`).",
						},
						"skip_ssl": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Whether to skip SSL certificate verification. (Default `false`).",
						},
						"ip_family": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "IP family version to use for the connection. The value can be either `IPv4` or `IPv6`. (Default `IPv4`).",
							Validators: []validator.String{
								stringvalidator.OneOf("IPv4", "IPv6"),
							},
						},
					},
					Blocks: map[string]schema.Block{
						"assertion": schema.SetNestedBlock{
							Description: "Assertions to validate the HTTP response. URL monitors only support status code assertions.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"source": schema.StringAttribute{
										Required:    true,
										Description: "The source of the asserted value. The only allowed value is `STATUS_CODE`.",
										Validators: []validator.String{
											stringvalidator.OneOf("STATUS_CODE"),
										},
									},
									"property": schema.StringAttribute{
										Optional: true,
										Computed: true,
									},
									"comparison": schema.StringAttribute{
										Required:    true,
										Description: "The type of comparison to be executed between expected and actual value of the assertion. Possible values are `EQUALS`, `NOT_EQUALS`, `GREATER_THAN` and `LESS_THAN`.",
										Validators: []validator.String{
											stringvalidator.OneOf("EQUALS", "NOT_EQUALS", "GREATER_THAN", "LESS_THAN"),
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
			},
			triggerIncidentAttributeName: makeTriggerIncidentSchema(),
		},
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
}

// ModifyPlan applies default values to the optional+computed attributes
// nested in the request set block. Schema defaults cannot be used for these
// attributes (see the schema for details), so the plan is adjusted here
// instead. Substituting static values keeps every set element fully known
// and stable across plans, which set element correlation requires.
func (r *URLMonitorResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// A destroy plan has no planned state to default values in.
	if req.Plan.Raw.IsNull() {
		return
	}

	var configRequest types.Set
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("request"), &configRequest)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// When the configuration contains unknown values (references to
	// attributes of resources that have not been created yet), null values
	// cannot be distinguished from values that will resolve at apply time.
	// Leave the plan untouched; the apply-time re-plan applies the defaults
	// once the configuration values are known.
	if valueContainsUnknown(configRequest) {
		return
	}

	var request types.Set
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("request"), &request)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request = urlMonitorRequestWithDefaults(ctx, request, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("request"), request)...)
}

// urlMonitorRequestWithDefaults returns the request block value with default
// values substituted for null and unknown optional+computed attributes.
func urlMonitorRequestWithDefaults(ctx context.Context, set types.Set, diags *diag.Diagnostics) types.Set {
	if set.IsNull() || set.IsUnknown() {
		return set
	}

	var models []URLMonitorRequestModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return set
	}

	for i, model := range models {
		models[i].FollowRedirects = boolOrDefault(model.FollowRedirects, true)
		models[i].SkipSSL = boolOrDefault(model.SkipSSL, false)
		models[i].IPFamily = stringOrDefault(model.IPFamily, "IPv4")
		models[i].Assertions = urlMonitorAssertionsWithDefaults(ctx, model.Assertions, diags)
	}

	result, d := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: urlMonitorRequestAttrTypes}, models)
	diags.Append(d...)

	return result
}

// urlMonitorAssertionsWithDefaults returns the request.assertion block value
// with default values substituted for null and unknown optional+computed
// attributes.
func urlMonitorAssertionsWithDefaults(ctx context.Context, set types.Set, diags *diag.Diagnostics) types.Set {
	if set.IsNull() || set.IsUnknown() {
		return set
	}

	var models []URLMonitorAssertionModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return set
	}

	for i, model := range models {
		models[i].Property = stringOrDefault(model.Property, "")
	}

	result, d := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: urlMonitorAssertionAttrTypes}, models)
	diags.Append(d...)

	return result
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
		resp.Diagnostics.AddError(
			"Unable to Create URL Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
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
		resp.Diagnostics.AddError(
			"Unable to Read URL Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
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
		resp.Diagnostics.AddError(
			"Unable to Update URL Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
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
		resp.Diagnostics.AddError(
			"Unable to Delete URL Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
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
	locations := stringsFromSet(ctx, model.Locations, diags)
	if locations == nil {
		locations = []string{}
	}

	privateLocations := stringsFromSet(ctx, model.PrivateLocations, diags)
	if privateLocations == nil {
		privateLocations = []string{}
	}

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
		Request:                   urlMonitorRequestFromSet(ctx, model.Request, diags),
		GroupID:                   model.GroupID.ValueInt64(),
		GroupOrder:                int(model.GroupOrder.ValueInt64()),
		AlertChannelSubscriptions: alertChannelSubscriptionsFromSet(ctx, model.AlertChannelSubscriptions, diags),
		PrivateLocations:          &privateLocations,
		RetryStrategy:             retryStrategyFromObject(ctx, model.RetryStrategy, diags),
		TriggerIncident:           triggerIncidentFromSet(ctx, model.TriggerIncident, diags),
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

	// The API only tracks an offset for high frequency scheduling
	// (frequency 0). For regular frequencies a configured value of 0
	// (equivalent to unset) is preserved; any other value is cleared.
	if c.Frequency == 0 {
		model.FrequencyOffset = types.Int64Value(int64(c.FrequencyOffset))
	} else if model.FrequencyOffset.IsUnknown() ||
		(!model.FrequencyOffset.IsNull() && model.FrequencyOffset.ValueInt64() != 0) {
		model.FrequencyOffset = types.Int64Null()
	}

	model.Activated = types.BoolValue(c.Activated)
	model.Muted = types.BoolValue(c.Muted)
	model.ShouldFail = types.BoolValue(c.ShouldFail)
	model.RunParallel = types.BoolValue(c.RunParallel)
	model.Locations = urlMonitorStringSet(ctx, c.Locations, model.Locations, diags)
	model.DegradedResponseTime = types.Int64Value(int64(c.DegradedResponseTime))
	model.MaxResponseTime = types.Int64Value(int64(c.MaxResponseTime))
	model.Tags = urlMonitorStringSet(ctx, c.Tags, model.Tags, diags)

	if c.AlertSettings != nil {
		model.AlertSettings = alertSettingsToObject(ctx, *c.AlertSettings, model.AlertSettings, diags)
	} else if model.AlertSettings.IsUnknown() {
		model.AlertSettings = types.ObjectNull(alertSettingsAttrTypes)
	}

	model.UseGlobalAlertSettings = types.BoolValue(c.UseGlobalAlertSettings)
	model.Request = urlMonitorRequestToSet(ctx, c.Request, diags)
	model.GroupID = types.Int64Value(c.GroupID)
	model.GroupOrder = types.Int64Value(int64(c.GroupOrder))

	var privateLocations []string
	if c.PrivateLocations != nil {
		privateLocations = *c.PrivateLocations
	}
	model.PrivateLocations = urlMonitorStringSet(ctx, privateLocations, model.PrivateLocations, diags)

	model.AlertChannelSubscriptions = alertChannelSubscriptionsToSet(ctx, c.AlertChannelSubscriptions, diags)
	model.RetryStrategy = retryStrategyToObject(ctx, c.RetryStrategy, model.RetryStrategy, diags)
	model.TriggerIncident = triggerIncidentToSet(ctx, c.TriggerIncident, diags)
}

// urlMonitorStringSet converts a string slice from the API to a set value.
// An empty slice resolves to null when the prior value is null, so that an
// optional set that is absent from the configuration stays null in state.
func urlMonitorStringSet(ctx context.Context, values []string, prior types.Set, diags *diag.Diagnostics) types.Set {
	if len(values) == 0 && (prior.IsNull() || prior.IsUnknown()) {
		return types.SetNull(types.StringType)
	}

	return setFromStrings(ctx, values, diags)
}

// urlMonitorRequestFromSet converts the request block value to its API
// representation.
func urlMonitorRequestFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) checkly.URLRequest {
	if set.IsNull() || set.IsUnknown() {
		return checkly.URLRequest{}
	}

	var models []URLMonitorRequestModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() || len(models) == 0 {
		return checkly.URLRequest{}
	}

	model := models[0]

	return checkly.URLRequest{
		URL:             model.URL.ValueString(),
		FollowRedirects: model.FollowRedirects.ValueBool(),
		SkipSSL:         model.SkipSSL.ValueBool(),
		Assertions:      urlMonitorAssertionsFromSet(ctx, model.Assertions, diags),
		IPFamily:        model.IPFamily.ValueString(),
	}
}

// urlMonitorRequestToSet converts the API representation of the request to
// the request block value.
func urlMonitorRequestToSet(ctx context.Context, r checkly.URLRequest, diags *diag.Diagnostics) types.Set {
	set, d := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: urlMonitorRequestAttrTypes}, []URLMonitorRequestModel{
		{
			URL:             types.StringValue(r.URL),
			FollowRedirects: types.BoolValue(r.FollowRedirects),
			SkipSSL:         types.BoolValue(r.SkipSSL),
			Assertions:      urlMonitorAssertionsToSet(ctx, r.Assertions, diags),
			IPFamily:        types.StringValue(r.IPFamily),
		},
	})
	diags.Append(d...)

	return set
}

// urlMonitorAssertionsFromSet converts the request.assertion block value to
// its API representation.
func urlMonitorAssertionsFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) []checkly.Assertion {
	assertions := []checkly.Assertion{}

	if set.IsNull() || set.IsUnknown() {
		return assertions
	}

	var models []URLMonitorAssertionModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return assertions
	}

	for _, model := range models {
		assertions = append(assertions, checkly.Assertion{
			Source:     model.Source.ValueString(),
			Property:   model.Property.ValueString(),
			Comparison: model.Comparison.ValueString(),
			Target:     model.Target.ValueString(),
		})
	}

	return assertions
}

// urlMonitorAssertionsToSet converts the API representation of assertions to
// the request.assertion block value.
func urlMonitorAssertionsToSet(ctx context.Context, assertions []checkly.Assertion, diags *diag.Diagnostics) types.Set {
	models := make([]URLMonitorAssertionModel, len(assertions))
	for i, assertion := range assertions {
		models[i] = URLMonitorAssertionModel{
			Source:     types.StringValue(assertion.Source),
			Property:   types.StringValue(assertion.Property),
			Comparison: types.StringValue(assertion.Comparison),
			Target:     types.StringValue(assertion.Target),
		}
	}

	set, d := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: urlMonitorAssertionAttrTypes}, models)
	diags.Append(d...)

	return set
}
