package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

var (
	_ resource.Resource                = (*MultistepCheckResource)(nil)
	_ resource.ResourceWithConfigure   = (*MultistepCheckResource)(nil)
	_ resource.ResourceWithImportState = (*MultistepCheckResource)(nil)
	_ resource.ResourceWithMoveState   = (*MultistepCheckResource)(nil)
)

// NewMultistepCheckResource returns a new checkly_multistep_check resource.
func NewMultistepCheckResource() resource.Resource {
	return &MultistepCheckResource{}
}

// MultistepCheckResource implements the checkly_multistep_check resource.
type MultistepCheckResource struct {
	client checkly.Client
}

// MultistepCheckResourceModel describes the resource data model.
type MultistepCheckResourceModel struct {
	ID                        types.String `tfsdk:"id"`
	Name                      types.String `tfsdk:"name"`
	Description               types.String `tfsdk:"description"`
	Frequency                 types.Int64  `tfsdk:"frequency"`
	Activated                 types.Bool   `tfsdk:"activated"`
	Muted                     types.Bool   `tfsdk:"muted"`
	ShouldFail                types.Bool   `tfsdk:"should_fail"`
	RunParallel               types.Bool   `tfsdk:"run_parallel"`
	Locations                 types.Set    `tfsdk:"locations"`
	Script                    types.String `tfsdk:"script"`
	ScriptPath                types.String `tfsdk:"script_path"`
	Dependencies              types.List   `tfsdk:"dependencies"`
	EnvironmentVariables      types.List   `tfsdk:"environment_variables"`
	Tags                      types.Set    `tfsdk:"tags"`
	RuntimeID                 types.String `tfsdk:"runtime_id"`
	AlertChannelSubscriptions types.Set    `tfsdk:"alert_channel_subscriptions"`
	PrivateLocations          types.Set    `tfsdk:"private_locations"`
	AlertSettings             types.Object `tfsdk:"alert_settings"`
	UseGlobalAlertSettings    types.Bool   `tfsdk:"use_global_alert_settings"`
	GroupID                   types.Int64  `tfsdk:"group_id"`
	GroupOrder                types.Int64  `tfsdk:"group_order"`
	RetryStrategy             types.Object `tfsdk:"retry_strategy"`
	TriggerIncident           types.Object `tfsdk:"trigger_incident"`
	Intent                    types.Object `tfsdk:"intent"`
	AIAutoRepairEnabled       types.Bool   `tfsdk:"ai_auto_repair_enabled"`
}

func (r *MultistepCheckResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_multistep_check"
}

func (r *MultistepCheckResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Multi-step checks allow you to monitor complex API interactions consisting of multiple requests using Playwright's API testing mode, so you get notified when flows break or slow down.",
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
			frequencyAttributeName: makeFrequencyAttributeSchema(FrequencyAttributeSchemaOptions{}),
			"script": schema.StringAttribute{
				Required:    true,
				Description: "A valid piece of Node.js JavaScript code describing a multi-step interaction with the Playwright framework.",
			},
			"script_path": schema.StringAttribute{
				Optional:    true,
				Description: "The path of the check's script file when using editor-style multi-file scripts (e.g. `entrypoint.spec.ts`).",
			},
			triggerIncidentAttributeName:     makeTriggerIncidentSchema(),
			checkIntentAttributeName:         makeCheckIntentSchema(),
			aiAutoRepairEnabledAttributeName: makeAIAutoRepairEnabledSchema(),
			alertSettingsAttributeName:       makeAlertSettingsAttributeSchema(AlertSettingsAttributeSchemaOptions{}),
			retryStrategyAttributeName: makeRetryStrategyAttributeSchema(RetryStrategyAttributeSchemaOptions{
				SupportsOnlyOnNetworkError: true,
			}),
			alertChannelSubscriptionAttributeName: makeAlertChannelSubscriptionSchema(AlertChannelSubscriptionAttributeSchemaOptions{}),
			environmentVariableAttributeName: makeEnvironmentVariableSchema(EnvironmentVariableAttributeSchemaOptions{
				Description: "Insert environment variables into the runtime environment. Use global environment variables whenever possible.",
			}),
			scriptDependenciesAttributeName: makeScriptDependenciesSchema(),
		}, CommonCheckAttributeSchemaOptions{Subject: subjectCheck, ShouldFail: true, RunParallel: true, Locations: true, GroupMembership: true, RuntimeID: true, RuntimeIDNote: "The runtime must support multi-step checks."}),
	}
}

func (r *MultistepCheckResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *MultistepCheckResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MultistepCheckResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	check := multistepCheckFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	r.validateMultistepCheckRuntimeSupport(ctx, &check, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.CreateMultiStepCheck(apiCtx, check)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to Create Multi-step Check", err)
		return
	}

	multistepCheckResolveComputedValues(ctx, result, &plan, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MultistepCheckResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state MultistepCheckResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	check, err := r.client.GetMultiStepCheck(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		addAPIError(&resp.Diagnostics, "Unable to Read Multi-step Check", err)
		return
	}

	multistepCheckToModel(ctx, check, &state, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *MultistepCheckResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan MultistepCheckResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	check := multistepCheckFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	r.validateMultistepCheckRuntimeSupport(ctx, &check, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.UpdateMultiStepCheck(apiCtx, check.ID, check)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to Update Multi-step Check", err)
		return
	}

	multistepCheckResolveComputedValues(ctx, result, &plan, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MultistepCheckResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state MultistepCheckResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	err := r.client.DeleteMultiStepCheck(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			return
		}
		addAPIError(&resp.Diagnostics, "Unable to Delete Multi-step Check", err)
		return
	}
}

func (r *MultistepCheckResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// MoveState enables `moved` blocks from the deprecated checkly_check resource
// type. Only MULTI_STEP checks can be moved to this resource type; the other
// check types have their own dedicated resource types.
func (r *MultistepCheckResource) MoveState(ctx context.Context) []resource.StateMover {
	return checkStateMovers(ctx, checkly.CheckTypeMultiStep, multistepCheckModelFromCheckModel)
}

// multistepCheckModelFromCheckModel converts a checkly_check model of a
// multi-step check. The frequency_offset, request, response time, snippet,
// local script and ssl_check_domain attributes of checkly_check do not exist
// on checkly_multistep_check and are discarded. The script_path and
// dependencies attributes are new and have no source value.
func multistepCheckModelFromCheckModel(source *CheckResourceModel) MultistepCheckResourceModel {
	return MultistepCheckResourceModel{
		ID:                        source.ID,
		Name:                      source.Name,
		Description:               source.Description,
		Frequency:                 source.Frequency,
		Activated:                 source.Activated,
		Muted:                     source.Muted,
		ShouldFail:                source.ShouldFail,
		RunParallel:               source.RunParallel,
		Locations:                 source.Locations,
		Script:                    source.Script,
		ScriptPath:                types.StringNull(),
		Dependencies:              types.ListNull(types.ObjectType{AttrTypes: scriptDependencyAttrTypes}),
		EnvironmentVariables:      source.EnvironmentVariables,
		Tags:                      source.Tags,
		RuntimeID:                 source.RuntimeID,
		AlertChannelSubscriptions: source.AlertChannelSubscriptions,
		PrivateLocations:          source.PrivateLocations,
		AlertSettings:             source.AlertSettings,
		UseGlobalAlertSettings:    source.UseGlobalAlertSettings,
		GroupID:                   source.GroupID,
		GroupOrder:                source.GroupOrder,
		RetryStrategy:             source.RetryStrategy,
		TriggerIncident:           source.TriggerIncident,
		Intent:                    source.Intent,
		AIAutoRepairEnabled:       source.AIAutoRepairEnabled,
	}
}

// validateMultistepCheckRuntimeSupport validates that the configured runtime
// supports multi-step checks when the check has a runtime ID set. Note that
// if the runtime ID is coming from the account defaults or group, then it is
// not validated.
func (r *MultistepCheckResource) validateMultistepCheckRuntimeSupport(ctx context.Context, check *checkly.MultiStepCheck, diags *diag.Diagnostics) {
	if check.RuntimeID == nil {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	checkRuntime, err := r.client.GetRuntime(apiCtx, *check.RuntimeID)
	if err != nil {
		diags.AddError(
			"Unable to Validate Runtime Support",
			fmt.Sprintf("API error while fetching runtimes: %s", err),
		)
		return
	}

	if !checkRuntime.MultiStepSupport {
		diags.AddError(
			"Unsupported Runtime",
			fmt.Sprintf("runtime %s does not support MULTI_STEP checks", *check.RuntimeID),
		)
	}
}

// multistepCheckFromModel converts the resource data model to its API
// representation.
func multistepCheckFromModel(ctx context.Context, model *MultistepCheckResourceModel, diags *diag.Diagnostics) checkly.MultiStepCheck {
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

	check := checkly.MultiStepCheck{
		ID:                        model.ID.ValueString(),
		Name:                      model.Name.ValueString(),
		Description:               description,
		Frequency:                 int(model.Frequency.ValueInt64()),
		Activated:                 model.Activated.ValueBool(),
		Muted:                     model.Muted.ValueBool(),
		ShouldFail:                model.ShouldFail.ValueBool(),
		RunParallel:               model.RunParallel.ValueBool(),
		Locations:                 locations,
		Script:                    model.Script.ValueString(),
		ScriptPath:                optionalStringPointer(model.ScriptPath),
		Dependencies:              scriptDependenciesFromList(ctx, model.Dependencies, diags),
		EnvironmentVariables:      environmentVariablesFromList(ctx, model.EnvironmentVariables, diags),
		Tags:                      stringsFromSet(ctx, model.Tags, diags),
		AlertSettings:             &alertSettings,
		UseGlobalAlertSettings:    model.UseGlobalAlertSettings.ValueBool(),
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

// multistepCheckResolveComputedValues finalizes a planned model after a
// successful create or update API call. All practitioner-configurable values
// are already known at plan time (defaults included) and must be kept as
// planned; only the values that may still be unknown in the plan (the ID and
// the optional+computed alert_settings and retry_strategy objects) are
// resolved from the API response. Notably, the dependencies value is kept
// as planned because create and update API responses do not echo
// dependencies; only reads include them.
func multistepCheckResolveComputedValues(ctx context.Context, c *checkly.MultiStepCheck, model *MultistepCheckResourceModel, diags *diag.Diagnostics) {
	model.ID = types.StringValue(c.ID)

	model.AlertSettings = alertSettingsFromAPI(ctx, c.AlertSettings, model.AlertSettings, diags)

	model.RetryStrategy = retryStrategyToObject(ctx, c.RetryStrategy, model.RetryStrategy, diags)
}

// multistepCheckToModel updates the model to match the API representation of
// the check. Values already present in the model act as the prior state for
// the prior-aware conversions.
func multistepCheckToModel(ctx context.Context, c *checkly.MultiStepCheck, model *MultistepCheckResourceModel, diags *diag.Diagnostics) {
	prior := *model

	model.ID = types.StringValue(c.ID)
	model.Name = types.StringValue(c.Name)

	// The attribute defaults to "", so a missing description maps to "".
	if c.Description != nil {
		model.Description = types.StringValue(*c.Description)
	} else {
		model.Description = types.StringValue("")
	}

	model.Frequency = types.Int64Value(int64(c.Frequency))
	model.Activated = types.BoolValue(c.Activated)
	model.Muted = types.BoolValue(c.Muted)
	model.ShouldFail = types.BoolValue(c.ShouldFail)
	model.RunParallel = types.BoolValue(c.RunParallel)
	model.Locations = stringSetFromAPINullIfUnset(ctx, c.Locations, prior.Locations, diags)
	model.Script = types.StringValue(c.Script)

	// An empty script path is not tracked by the Checkly API. A configured
	// empty value is preserved so that it does not drift to null.
	switch {
	case c.ScriptPath != nil && *c.ScriptPath != "":
		model.ScriptPath = types.StringValue(*c.ScriptPath)
	case !prior.ScriptPath.IsNull() && !prior.ScriptPath.IsUnknown() && prior.ScriptPath.ValueString() == "":
		model.ScriptPath = prior.ScriptPath
	default:
		model.ScriptPath = types.StringNull()
	}

	model.Dependencies = scriptDependenciesToList(ctx, c.Dependencies, prior.Dependencies, diags)
	model.EnvironmentVariables = environmentVariablesToList(ctx, c.EnvironmentVariables, prior.EnvironmentVariables, diags)
	model.Tags = stringSetFromAPINullIfUnset(ctx, c.Tags, prior.Tags, diags)

	model.RuntimeID = stringValueOrPriorEmpty(valueOrZero(c.RuntimeID), prior.RuntimeID)

	model.AlertChannelSubscriptions = alertChannelSubscriptionsToSet(ctx, c.AlertChannelSubscriptions, prior.AlertChannelSubscriptions, diags)

	model.PrivateLocations = stringSetFromAPINullIfUnset(ctx, valueOrZero(c.PrivateLocations), prior.PrivateLocations, diags)

	model.AlertSettings = alertSettingsFromAPI(ctx, c.AlertSettings, prior.AlertSettings, diags)

	model.UseGlobalAlertSettings = types.BoolValue(c.UseGlobalAlertSettings)
	model.GroupID = int64ValueOrNull(c.GroupID)
	model.GroupOrder = groupOrderFromAPI(c.GroupOrder, model.GroupOrder)
	model.RetryStrategy = retryStrategyToObject(ctx, c.RetryStrategy, prior.RetryStrategy, diags)
	model.TriggerIncident = triggerIncidentToObject(ctx, c.TriggerIncident, diags)
	model.Intent = checkIntentToObject(ctx, c.Intent, prior.Intent, diags)
	model.AIAutoRepairEnabled = types.BoolPointerValue(c.AIAutoRepairEnabled)
}
