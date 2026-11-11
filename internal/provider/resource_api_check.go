package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

var (
	_ resource.Resource                     = (*APICheckResource)(nil)
	_ resource.ResourceWithConfigure        = (*APICheckResource)(nil)
	_ resource.ResourceWithImportState      = (*APICheckResource)(nil)
	_ resource.ResourceWithConfigValidators = (*APICheckResource)(nil)
	_ resource.ResourceWithModifyPlan       = (*APICheckResource)(nil)
	_ resource.ResourceWithMoveState        = (*APICheckResource)(nil)
)

// NewAPICheckResource returns a new checkly_api_check resource.
func NewAPICheckResource() resource.Resource {
	return &APICheckResource{}
}

// APICheckResource implements the checkly_api_check resource.
type APICheckResource struct {
	client checkly.Client
}

// APICheckResourceModel describes the resource data model.
type APICheckResourceModel struct {
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
	SetupSnippetID            types.Int64  `tfsdk:"setup_snippet_id"`
	TeardownSnippetID         types.Int64  `tfsdk:"teardown_snippet_id"`
	LocalSetupScript          types.String `tfsdk:"local_setup_script"`
	LocalTeardownScript       types.String `tfsdk:"local_teardown_script"`
	RuntimeID                 types.String `tfsdk:"runtime_id"`
	AlertChannelSubscriptions types.Set    `tfsdk:"alert_channel_subscriptions"`
	PrivateLocations          types.Set    `tfsdk:"private_locations"`
	AlertSettings             types.Object `tfsdk:"alert_settings"`
	UseGlobalAlertSettings    types.Bool   `tfsdk:"use_global_alert_settings"`
	Request                   types.Object `tfsdk:"request"`
	GroupID                   types.Int64  `tfsdk:"group_id"`
	GroupOrder                types.Int64  `tfsdk:"group_order"`
	RetryStrategy             types.Object `tfsdk:"retry_strategy"`
	TriggerIncident           types.Object `tfsdk:"trigger_incident"`
	Intent                    types.Object `tfsdk:"intent"`
	AIAutoRepairEnabled       types.Bool   `tfsdk:"ai_auto_repair_enabled"`
}

func (r *APICheckResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_check"
}

func (r *APICheckResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "API checks allow you to monitor your API endpoints for correctness, performance and availability, " +
			"and set up alerting, so you get a notification when things break or slow down.",
		Attributes: withCommonCheckAttributes(map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the check.",
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
				Description: "A description of the check.",
			},
			frequencyAttributeName:       makeFrequencyAttributeSchema(FrequencyAttributeSchemaOptions{AllowHighFrequency: true}),
			frequencyOffsetAttributeName: makeFrequencyOffsetAttributeSchema(FrequencyOffsetAttributeSchemaOptions{}),
			"degraded_response_time": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(5000),
				// https://checklyhq.com/docs/api-checks/limits/
				Validators: []validator.Int64{
					int64validator.Between(0, 30000),
				},
				Description: "The response time in milliseconds starting from which a check should be considered degraded. Possible values are between 0 and 30000. (Default `5000`).",
			},
			"max_response_time": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(20000),
				// https://checklyhq.com/docs/api-checks/limits/
				Validators: []validator.Int64{
					int64validator.Between(0, 30000),
				},
				Description: "The response time in milliseconds starting from which a check should be considered failing. Possible values are between 0 and 30000. (Default `20000`).",
			},
			"setup_snippet_id": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "An ID reference to a snippet to use in the setup phase of an API check.",
			},
			"teardown_snippet_id": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "An ID reference to a snippet to use in the teardown phase of an API check.",
			},
			"local_setup_script": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "A valid piece of Node.js code to run in the setup phase.",
			},
			"local_teardown_script": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "A valid piece of Node.js code to run in the teardown phase.",
			},
			triggerIncidentAttributeName:     makeTriggerIncidentSchema(),
			checkIntentAttributeName:         makeCheckIntentSchema(),
			aiAutoRepairEnabledAttributeName: makeAIAutoRepairEnabledSchema(),
			alertSettingsAttributeName:       makeAlertSettingsAttributeSchema(AlertSettingsAttributeSchemaOptions{}),
			retryStrategyAttributeName: makeRetryStrategyAttributeSchema(RetryStrategyAttributeSchemaOptions{
				SupportsOnlyOnNetworkError: true,
			}),
			"request": apiCheckRequestAttributeSchema(apiCheckRequestAttributeSchemaOptions{
				Description: "The parameters of the HTTP request.",
				Required:    true,
			}),
			alertChannelSubscriptionAttributeName: makeAlertChannelSubscriptionSchema(AlertChannelSubscriptionAttributeSchemaOptions{}),
		}, CommonCheckAttributeSchemaOptions{Subject: subjectCheck, ShouldFail: true, RunParallel: true, Locations: true, GroupMembership: true, RuntimeID: true, ShouldFailNote: shouldFailErrorStatusNote}),
	}
}

func (r *APICheckResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		newFrequencyOffsetConfigValidator(),
	}
}

func (r *APICheckResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

// ModifyPlan applies default values to the optional+computed attributes of
// the request assertions. Schema defaults cannot be used for attributes
// nested in set elements (see the schema for details), so the plan is
// adjusted here instead. Substituting static values keeps every set element
// fully known and stable across plans, which set element correlation
// requires.
func (r *APICheckResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planNestedDefaults(ctx, req, resp, path.Root("request"), objectWithSetDefaults("assertions", assertionsWithPropertyAndTargetDefaults))
}

func (r *APICheckResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan APICheckResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	check := apiCheckFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.CreateAPICheck(apiCtx, check)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to Create API Check", err)
		return
	}

	apiCheckResolveComputedValues(ctx, result, &plan, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *APICheckResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state APICheckResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	check, err := r.client.GetAPICheck(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		addAPIError(&resp.Diagnostics, "Unable to Read API Check", err)
		return
	}

	apiCheckToModel(ctx, check, &state, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *APICheckResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan APICheckResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	check := apiCheckFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.UpdateAPICheck(apiCtx, check.ID, check)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to Update API Check", err)
		return
	}

	apiCheckResolveComputedValues(ctx, result, &plan, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *APICheckResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state APICheckResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	err := r.client.DeleteAPICheck(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			return
		}
		addAPIError(&resp.Diagnostics, "Unable to Delete API Check", err)
		return
	}
}

func (r *APICheckResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// MoveState enables `moved` blocks from the deprecated checkly_check resource
// type. Only checks with type = "API" can be moved; the schemas are otherwise
// compatible.
func (r *APICheckResource) MoveState(ctx context.Context) []resource.StateMover {
	return checkStateMovers(ctx, checkly.CheckTypeAPI, apiCheckModelFromCheckModel)
}

// apiCheckModelFromCheckModel converts a checkly_check model of an API check.
// The source values are mapped over one-to-one, dropping the attributes that
// do not apply to API checks (type, script, environment_variables and
// ssl_check_domain).
func apiCheckModelFromCheckModel(source *CheckResourceModel) APICheckResourceModel {
	return APICheckResourceModel{
		ID:                        source.ID,
		Name:                      source.Name,
		Description:               source.Description,
		Frequency:                 source.Frequency,
		FrequencyOffset:           source.FrequencyOffset,
		Activated:                 source.Activated,
		Muted:                     source.Muted,
		ShouldFail:                source.ShouldFail,
		RunParallel:               source.RunParallel,
		Locations:                 source.Locations,
		DegradedResponseTime:      source.DegradedResponseTime,
		MaxResponseTime:           source.MaxResponseTime,
		Tags:                      source.Tags,
		SetupSnippetID:            source.SetupSnippetID,
		TeardownSnippetID:         source.TeardownSnippetID,
		LocalSetupScript:          source.LocalSetupScript,
		LocalTeardownScript:       source.LocalTeardownScript,
		RuntimeID:                 source.RuntimeID,
		AlertChannelSubscriptions: source.AlertChannelSubscriptions,
		PrivateLocations:          source.PrivateLocations,
		AlertSettings:             source.AlertSettings,
		UseGlobalAlertSettings:    source.UseGlobalAlertSettings,
		Request:                   source.Request,
		GroupID:                   source.GroupID,
		GroupOrder:                source.GroupOrder,
		RetryStrategy:             source.RetryStrategy,
		TriggerIncident:           source.TriggerIncident,
		Intent:                    source.Intent,
		AIAutoRepairEnabled:       source.AIAutoRepairEnabled,
	}
}

// apiCheckFromModel converts the resource data model to its API
// representation.
func apiCheckFromModel(ctx context.Context, model *APICheckResourceModel, diags *diag.Diagnostics) checkly.APICheck {
	// An empty description is not tracked by the Checkly API; send null
	// instead so that updates clear a previously configured description.
	var description *string
	if value := model.Description.ValueString(); value != "" {
		description = &value
	}

	// Empty setup and teardown scripts are not tracked by the Checkly API;
	// send null instead so that updates clear previously configured scripts.
	var localSetupScript *string
	if value := model.LocalSetupScript.ValueString(); value != "" {
		localSetupScript = &value
	}
	var localTeardownScript *string
	if value := model.LocalTeardownScript.ValueString(); value != "" {
		localTeardownScript = &value
	}

	// The locations property is serialized even when empty.
	locations := stringsFromSetOrEmpty(ctx, model.Locations, diags)

	privateLocations := stringsFromSetOrEmpty(ctx, model.PrivateLocations, diags)

	alertSettings := alertSettingsFromObject(ctx, model.AlertSettings, diags)

	check := checkly.APICheck{
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
		Request:                   apiCheckRequestFromObject(ctx, model.Request, diags),
		SetupSnippetID:            model.SetupSnippetID.ValueInt64(),
		TearDownSnippetID:         model.TeardownSnippetID.ValueInt64(),
		LocalSetupScript:          localSetupScript,
		LocalTearDownScript:       localTeardownScript,
		GroupID:                   model.GroupID.ValueInt64(),
		GroupOrder:                int(model.GroupOrder.ValueInt64()),
		AlertChannelSubscriptions: alertChannelSubscriptionsFromSet(ctx, model.AlertChannelSubscriptions, diags),
		PrivateLocations:          &privateLocations,
		RetryStrategy:             retryStrategyFromObject(ctx, model.RetryStrategy, diags),
		TriggerIncident:           triggerIncidentFromObject(ctx, model.TriggerIncident, diags),
		Intent:                    checkIntentFromObject(ctx, model.Intent, diags),
		AIAutoRepairEnabled:       model.AIAutoRepairEnabled.ValueBoolPointer(),
	}

	if runtimeID := model.RuntimeID.ValueString(); runtimeID != "" {
		check.RuntimeID = &runtimeID
	}

	return check
}

// apiCheckResolveComputedValues finalizes a planned model after a successful
// create or update API call. All practitioner-configurable values are already
// known at plan time (defaults included; the attributes nested in the request
// assertions are defaulted in ModifyPlan) and must be kept as planned; only the
// values that may still be unknown in the plan (the ID and the
// optional+computed alert_settings and retry_strategy objects) are resolved
// from the API response.
func apiCheckResolveComputedValues(ctx context.Context, c *checkly.APICheck, model *APICheckResourceModel, diags *diag.Diagnostics) {
	model.ID = types.StringValue(c.ID)

	model.AlertSettings = alertSettingsFromAPI(ctx, c.AlertSettings, model.AlertSettings, diags)

	model.RetryStrategy = retryStrategyToObject(ctx, c.RetryStrategy, model.RetryStrategy, diags)
}

// apiCheckToModel updates the model to match the API representation of the
// check. Values already present in the model act as the prior state for the
// prior-aware conversions.
func apiCheckToModel(ctx context.Context, c *checkly.APICheck, model *APICheckResourceModel, diags *diag.Diagnostics) {
	prior := *model

	model.ID = types.StringValue(c.ID)
	model.Name = types.StringValue(c.Name)
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
	model.Locations = stringSetFromAPINullIfUnset(ctx, c.Locations, prior.Locations, diags)
	model.DegradedResponseTime = types.Int64Value(int64(c.DegradedResponseTime))
	model.MaxResponseTime = types.Int64Value(int64(c.MaxResponseTime))
	model.Tags = stringSetFromAPINullIfUnset(ctx, c.Tags, prior.Tags, diags)
	model.SetupSnippetID = types.Int64Value(c.SetupSnippetID)
	model.TeardownSnippetID = types.Int64Value(c.TearDownSnippetID)
	if c.LocalSetupScript != nil {
		model.LocalSetupScript = types.StringValue(*c.LocalSetupScript)
	} else {
		model.LocalSetupScript = types.StringValue("")
	}
	if c.LocalTearDownScript != nil {
		model.LocalTeardownScript = types.StringValue(*c.LocalTearDownScript)
	} else {
		model.LocalTeardownScript = types.StringValue("")
	}
	model.RuntimeID = stringValueOrPriorEmpty(valueOrZero(c.RuntimeID), prior.RuntimeID)
	model.AlertChannelSubscriptions = alertChannelSubscriptionsToSet(ctx, c.AlertChannelSubscriptions, prior.AlertChannelSubscriptions, diags)
	model.PrivateLocations = stringSetFromAPINullIfUnset(ctx, valueOrZero(c.PrivateLocations), prior.PrivateLocations, diags)
	model.AlertSettings = alertSettingsFromAPI(ctx, c.AlertSettings, prior.AlertSettings, diags)
	model.UseGlobalAlertSettings = types.BoolValue(c.UseGlobalAlertSettings)
	model.Request = apiCheckRequestToObject(ctx, c.Request, prior.Request, diags)
	model.GroupID = types.Int64Value(c.GroupID)
	model.GroupOrder = types.Int64Value(int64(c.GroupOrder))
	model.RetryStrategy = retryStrategyToObject(ctx, c.RetryStrategy, prior.RetryStrategy, diags)
	model.TriggerIncident = triggerIncidentToObject(ctx, c.TriggerIncident, diags)
	model.Intent = checkIntentToObject(ctx, c.Intent, prior.Intent, diags)
	model.AIAutoRepairEnabled = types.BoolPointerValue(c.AIAutoRepairEnabled)
}
