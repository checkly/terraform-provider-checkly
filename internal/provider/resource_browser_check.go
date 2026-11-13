package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	_ resource.Resource                = (*BrowserCheckResource)(nil)
	_ resource.ResourceWithConfigure   = (*BrowserCheckResource)(nil)
	_ resource.ResourceWithImportState = (*BrowserCheckResource)(nil)
	_ resource.ResourceWithMoveState   = (*BrowserCheckResource)(nil)
	_ resource.ResourceWithModifyPlan  = (*BrowserCheckResource)(nil)
)

// NewBrowserCheckResource returns a new checkly_browser_check resource.
func NewBrowserCheckResource() resource.Resource {
	return &BrowserCheckResource{}
}

// BrowserCheckResource implements the checkly_browser_check resource.
type BrowserCheckResource struct {
	client checkly.Client
}

// BrowserCheckResourceModel describes the resource data model.
type BrowserCheckResourceModel struct {
	ID                        types.String `tfsdk:"id"`
	Name                      types.String `tfsdk:"name"`
	Description               types.String `tfsdk:"description"`
	Frequency                 types.Int64  `tfsdk:"frequency"`
	Activated                 types.Bool   `tfsdk:"activated"`
	Muted                     types.Bool   `tfsdk:"muted"`
	RunParallel               types.Bool   `tfsdk:"run_parallel"`
	Locations                 types.Set    `tfsdk:"locations"`
	Script                    types.String `tfsdk:"script"`
	ScriptPath                types.String `tfsdk:"script_path"`
	Dependencies              types.List   `tfsdk:"dependencies"`
	EnvironmentVariables      types.List   `tfsdk:"environment_variables"`
	Tags                      types.Set    `tfsdk:"tags"`
	SSLCheckDomain            types.String `tfsdk:"ssl_check_domain"`
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

func (r *BrowserCheckResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_browser_check"
}

func (r *BrowserCheckResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Browser checks allow you to monitor key user flows in a real browser using Playwright, so you get notified when flows break or slow down.",
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
			frequencyAttributeName: makeFrequencyAttributeSchema(FrequencyAttributeSchemaOptions{Subject: subjectCheck}),
			"script": schema.StringAttribute{
				Required:    true,
				Description: "A valid piece of Node.js JavaScript code describing a browser interaction with the Playwright framework.",
			},
			"script_path": schema.StringAttribute{
				Optional:    true,
				Description: "The path of the check's script file when using editor-style multi-file scripts (e.g. `entrypoint.spec.ts`).",
			},
			"ssl_check_domain": schema.StringAttribute{
				Optional:    true,
				Description: "A valid fully qualified domain name (FQDN) to check its SSL certificate.",
			},
			triggerIncidentAttributeName:     makeTriggerIncidentSchema(),
			checkIntentAttributeName:         makeCheckIntentSchema(),
			aiAutoRepairEnabledAttributeName: makeAIAutoRepairEnabledSchema(),
			alertSettingsAttributeName:       makeAlertSettingsAttributeSchema(AlertSettingsAttributeSchemaOptions{Subject: subjectCheck}),
			retryStrategyAttributeName: makeRetryStrategyAttributeSchema(RetryStrategyAttributeSchemaOptions{
				SupportsOnlyOnNetworkError: true,
			}),
			alertChannelSubscriptionAttributeName: makeAlertChannelSubscriptionSchema(AlertChannelSubscriptionAttributeSchemaOptions{Subject: subjectCheck}),
			environmentVariableAttributeName: makeEnvironmentVariableSchema(EnvironmentVariableAttributeSchemaOptions{
				Description: "Insert environment variables into the runtime environment. Use global environment variables whenever possible.",
			}),
			scriptDependenciesAttributeName: makeScriptDependenciesSchema(),
		}, CommonCheckAttributeSchemaOptions{Subject: subjectCheck, RunParallel: true, Locations: true, GroupMembership: true, RuntimeID: true}),
	}
}

func (r *BrowserCheckResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

// ModifyPlan reports an empty script in the plan when it is known there.
// Create and Update check again for a script known only at apply time.
func (r *BrowserCheckResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	validatePlannedCheckScript(ctx, req, resp, string(checkly.CheckTypeBrowser))
}

func (r *BrowserCheckResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BrowserCheckResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	requireCheckScript(plan.Script, string(checkly.CheckTypeBrowser), &resp.Diagnostics)
	check := browserCheckFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.CreateBrowserCheck(apiCtx, check)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to Create Browser Check", err)
		return
	}

	browserCheckResolveComputedValues(ctx, result, &plan, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BrowserCheckResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BrowserCheckResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	check, err := r.client.GetBrowserCheck(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		addAPIError(&resp.Diagnostics, "Unable to Read Browser Check", err)
		return
	}

	browserCheckToModel(ctx, check, &state, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *BrowserCheckResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BrowserCheckResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	requireCheckScript(plan.Script, string(checkly.CheckTypeBrowser), &resp.Diagnostics)
	check := browserCheckFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.UpdateBrowserCheck(apiCtx, check.ID, check)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to Update Browser Check", err)
		return
	}

	browserCheckResolveComputedValues(ctx, result, &plan, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BrowserCheckResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BrowserCheckResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	err := r.client.DeleteBrowserCheck(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			return
		}
		addAPIError(&resp.Diagnostics, "Unable to Delete Browser Check", err)
		return
	}
}

func (r *BrowserCheckResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// MoveState enables `moved` blocks from the deprecated checkly_check resource
// type. Only BROWSER checks can be moved to this resource type; the other
// check types have their own dedicated resource types.
func (r *BrowserCheckResource) MoveState(ctx context.Context) []resource.StateMover {
	return checkStateMovers(ctx, checkly.CheckTypeBrowser, browserCheckModelFromCheckModel)
}

// browserCheckModelFromCheckModel converts a checkly_check model of a browser
// check. The frequency_offset, request, response time, snippet and local
// script attributes of checkly_check do not exist on checkly_browser_check
// and are discarded. The script_path and dependencies attributes are new and
// have no source value.
func browserCheckModelFromCheckModel(source *CheckResourceModel) BrowserCheckResourceModel {
	return BrowserCheckResourceModel{
		ID:                        source.ID,
		Name:                      source.Name,
		Description:               source.Description,
		Frequency:                 source.Frequency,
		Activated:                 source.Activated,
		Muted:                     source.Muted,
		RunParallel:               source.RunParallel,
		Locations:                 source.Locations,
		Script:                    source.Script,
		ScriptPath:                types.StringNull(),
		Dependencies:              types.ListValueMust(types.ObjectType{AttrTypes: scriptDependencyAttrTypes}, []attr.Value{}),
		EnvironmentVariables:      source.EnvironmentVariables,
		Tags:                      source.Tags,
		SSLCheckDomain:            source.SSLCheckDomain,
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

// browserCheckFromModel converts the resource data model to its API
// representation.
func browserCheckFromModel(ctx context.Context, model *BrowserCheckResourceModel, diags *diag.Diagnostics) checkly.BrowserCheck {
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

	check := checkly.BrowserCheck{
		ID:                        model.ID.ValueString(),
		Name:                      model.Name.ValueString(),
		Description:               description,
		Frequency:                 int(model.Frequency.ValueInt64()),
		Activated:                 model.Activated.ValueBool(),
		Muted:                     model.Muted.ValueBool(),
		RunParallel:               model.RunParallel.ValueBool(),
		Locations:                 locations,
		Script:                    model.Script.ValueString(),
		ScriptPath:                optionalStringPointer(model.ScriptPath),
		Dependencies:              scriptDependenciesFromList(ctx, model.Dependencies, diags),
		EnvironmentVariables:      environmentVariablesFromList(ctx, model.EnvironmentVariables, diags),
		SSLCheckDomain:            model.SSLCheckDomain.ValueString(),
		Tags:                      stringsFromSet(ctx, model.Tags, diags),
		AlertSettings:             &alertSettings,
		UseGlobalAlertSettings:    model.UseGlobalAlertSettings.ValueBool(),
		GroupID:                   model.GroupID.ValueInt64(),
		GroupOrder:                groupOrderToAPI(model.GroupOrder),
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

// browserCheckResolveComputedValues finalizes a planned model after a
// successful create or update API call. All practitioner-configurable values
// are already known at plan time (defaults included) and must be kept as
// planned; only the values that may still be unknown in the plan (the ID and
// the optional+computed alert_settings and retry_strategy objects) are
// resolved from the API response. Notably, the dependencies value is kept
// as planned because create and update API responses do not echo
// dependencies; only reads include them.
func browserCheckResolveComputedValues(ctx context.Context, c *checkly.BrowserCheck, model *BrowserCheckResourceModel, diags *diag.Diagnostics) {
	model.ID = types.StringValue(c.ID)

	model.AlertSettings = alertSettingsFromAPI(ctx, c.AlertSettings, model.AlertSettings, diags)

	model.RetryStrategy = retryStrategyToObject(ctx, c.RetryStrategy, model.RetryStrategy, diags)
}

// browserCheckToModel updates the model to match the API representation of
// the check. Values already present in the model act as the prior state for
// the prior-aware conversions.
func browserCheckToModel(ctx context.Context, c *checkly.BrowserCheck, model *BrowserCheckResourceModel, diags *diag.Diagnostics) {
	// Read calls this, so the warning also reaches imported checks and
	// checks changed outside Terraform.
	if c.ShouldFail {
		addUnsupportedShouldFailWarning(diags, "checkly_browser_check")
	}

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
	model.RunParallel = types.BoolValue(c.RunParallel)
	model.Locations = setFromStrings(ctx, c.Locations, diags)
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
	model.Tags = setFromStrings(ctx, c.Tags, diags)

	// An empty SSL check domain is not tracked by the Checkly API. A
	// configured empty value is preserved so that it does not drift to null.
	switch {
	case c.SSLCheckDomain != "":
		model.SSLCheckDomain = types.StringValue(c.SSLCheckDomain)
	case !prior.SSLCheckDomain.IsNull() && !prior.SSLCheckDomain.IsUnknown() && prior.SSLCheckDomain.ValueString() == "":
		model.SSLCheckDomain = prior.SSLCheckDomain
	default:
		model.SSLCheckDomain = types.StringNull()
	}

	model.RuntimeID = stringValueOrPriorEmpty(valueOrZero(c.RuntimeID), prior.RuntimeID)

	model.AlertChannelSubscriptions = alertChannelSubscriptionsToSet(ctx, c.AlertChannelSubscriptions, diags)

	model.PrivateLocations = setFromStrings(ctx, valueOrZero(c.PrivateLocations), diags)

	model.AlertSettings = alertSettingsFromAPI(ctx, c.AlertSettings, prior.AlertSettings, diags)

	model.UseGlobalAlertSettings = types.BoolValue(c.UseGlobalAlertSettings)
	model.GroupID = int64ValueOrNull(c.GroupID)
	model.GroupOrder = groupOrderFromAPI(c.GroupOrder, model.GroupOrder)
	model.RetryStrategy = retryStrategyToObject(ctx, c.RetryStrategy, prior.RetryStrategy, diags)
	model.TriggerIncident = triggerIncidentToObject(ctx, c.TriggerIncident, diags)
	model.Intent = checkIntentToObject(ctx, c.Intent, diags)
	model.AIAutoRepairEnabled = types.BoolPointerValue(c.AIAutoRepairEnabled)
}
