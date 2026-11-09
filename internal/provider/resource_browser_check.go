package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
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
	ShouldFail                types.Bool   `tfsdk:"should_fail"`
	RunParallel               types.Bool   `tfsdk:"run_parallel"`
	Locations                 types.Set    `tfsdk:"locations"`
	Script                    types.String `tfsdk:"script"`
	ScriptPath                types.String `tfsdk:"script_path"`
	Dependency                types.List   `tfsdk:"dependency"`
	EnvironmentVariable       types.List   `tfsdk:"environment_variable"`
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

// browserCheckFrequencySchema returns the schema for the frequency attribute.
// Browser checks do not support high frequency scheduling, and the attribute
// defaults to 10 minutes.
func browserCheckFrequencySchema() schema.Int64Attribute {
	frequency := makeFrequencyAttributeSchema(FrequencyAttributeSchemaOptions{})
	frequency.Required = false
	frequency.Optional = true
	frequency.Computed = true
	frequency.Default = int64default.StaticInt64(10)
	frequency.Description += " (Default `10`)."
	return frequency
}

func (r *BrowserCheckResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Browser checks allow you to monitor key user flows in a real browser using Playwright, so you get notified when flows break or slow down.",
		Attributes: map[string]schema.Attribute{
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
			frequencyAttributeName: browserCheckFrequencySchema(),
			"activated": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Determines if the check is running or not. Possible values `true`, and `false`. (Default `true`).",
			},
			"muted": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Determines if any notifications will be sent out when a check fails/degrades/recovers. (Default `false`).",
			},
			"should_fail": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Allows to invert the behaviour of when a check is considered to fail. (Default `false`).",
			},
			"run_parallel": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Determines if the check should run in all selected locations in parallel or round-robin. (Default `false`).",
			},
			"locations": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "An array of one or more data center locations where to run this check. (Default [\"us-east-1\"])",
			},
			"script": schema.StringAttribute{
				Required:    true,
				Description: "A valid piece of Node.js JavaScript code describing a browser interaction with the Playwright framework.",
			},
			"script_path": schema.StringAttribute{
				Optional:    true,
				Description: "The path of the check's script file when using editor-style multi-file scripts (e.g. `entrypoint.spec.ts`).",
			},
			"tags": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "A list of tags for organizing and filtering checks.",
			},
			"ssl_check_domain": schema.StringAttribute{
				Optional:    true,
				Description: "A valid fully qualified domain name (FQDN) to check its SSL certificate.",
			},
			"runtime_id": schema.StringAttribute{
				Optional:    true,
				Description: "The id of the runtime to use for this check.",
			},
			"private_locations": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
				Description: "An array of one or more private locations slugs.",
			},
			triggerIncidentAttributeName:     makeTriggerIncidentSchema(),
			checkIntentAttributeName:         makeCheckIntentSchema(),
			aiAutoRepairEnabledAttributeName: makeAIAutoRepairEnabledSchema(),
			alertSettingsAttributeName:       makeAlertSettingsAttributeSchema(AlertSettingsAttributeSchemaOptions{}),
			"use_global_alert_settings": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "When true, the account level alert settings will be used, not the alert setting defined on this check. (Default `false`).",
			},
			"group_id": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "The id of the check group this check is part of.",
			},
			"group_order": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "The position of this check in a check group. It determines in what order checks are run when a group is triggered from the API or from CI/CD.",
			},
			retryStrategyAttributeName: makeRetryStrategyAttributeSchema(RetryStrategyAttributeSchemaOptions{
				SupportsOnlyOnNetworkError: true,
			}),
			alertChannelSubscriptionAttributeName: makeAlertChannelSubscriptionSchema(AlertChannelSubscriptionAttributeSchemaOptions{}),
		},
		Blocks: map[string]schema.Block{
			"dependency": schema.ListNestedBlock{
				Description: "Script dependency files for editor-style multi-file scripts. Use together with `script_path` to mirror a multi-file layout.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"path": schema.StringAttribute{
							Required:    true,
							Description: "The path of the dependency file.",
						},
						"content": schema.StringAttribute{
							Required:    true,
							Description: "The content of the dependency file.",
						},
					},
				},
			},
			environmentVariableAttributeName: makeEnvironmentVariableSchema(EnvironmentVariableAttributeSchemaOptions{
				Description: "Insert environment variables into the runtime environment. Use global environment variables whenever possible.",
			}),
		},
	}
}

func (r *BrowserCheckResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *BrowserCheckResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BrowserCheckResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	check := browserCheckFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.CreateBrowserCheck(apiCtx, check)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Browser Check",
			fmt.Sprintf("Checkly API error: %s", err),
		)
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
		resp.Diagnostics.AddError(
			"Unable to Read Browser Check",
			fmt.Sprintf("Checkly API error: %s", err),
		)
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

	check := browserCheckFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.UpdateBrowserCheck(apiCtx, check.ID, check)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Browser Check",
			fmt.Sprintf("Checkly API error: %s", err),
		)
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
		resp.Diagnostics.AddError(
			"Unable to Delete Browser Check",
			fmt.Sprintf("Checkly API error: %s", err),
		)
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
// and are discarded. The script_path and dependency attributes are new and
// have no source value.
func browserCheckModelFromCheckModel(source *CheckResourceModel) BrowserCheckResourceModel {
	return BrowserCheckResourceModel{
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
		Dependency:                types.ListValueMust(types.ObjectType{AttrTypes: scriptDependencyAttrTypes}, []attr.Value{}),
		EnvironmentVariable:       source.EnvironmentVariable,
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
		ShouldFail:                model.ShouldFail.ValueBool(),
		RunParallel:               model.RunParallel.ValueBool(),
		Locations:                 locations,
		Script:                    model.Script.ValueString(),
		ScriptPath:                optionalStringPointer(model.ScriptPath),
		Dependencies:              scriptDependenciesFromList(ctx, model.Dependency, diags),
		EnvironmentVariables:      environmentVariablesFromList(ctx, model.EnvironmentVariable, diags),
		SSLCheckDomain:            model.SSLCheckDomain.ValueString(),
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

// browserCheckResolveComputedValues finalizes a planned model after a
// successful create or update API call. All practitioner-configurable values
// are already known at plan time (defaults included) and must be kept as
// planned; only the values that may still be unknown in the plan (the ID and
// the optional+computed alert_settings and retry_strategy objects) are
// resolved from the API response. Notably, the dependency block value is kept
// as planned because create and update API responses do not echo
// dependencies; only reads include them.
func browserCheckResolveComputedValues(ctx context.Context, c *checkly.BrowserCheck, model *BrowserCheckResourceModel, diags *diag.Diagnostics) {
	model.ID = types.StringValue(c.ID)

	if c.AlertSettings != nil {
		model.AlertSettings = alertSettingsToObject(ctx, *c.AlertSettings, model.AlertSettings, diags)
	} else if model.AlertSettings.IsUnknown() {
		model.AlertSettings = types.ObjectNull(alertSettingsAttrTypes)
	}

	model.RetryStrategy = retryStrategyToObject(ctx, c.RetryStrategy, model.RetryStrategy, diags)
}

// browserCheckToModel updates the model to match the API representation of
// the check. Values already present in the model act as the prior state for
// the prior-aware conversions.
func browserCheckToModel(ctx context.Context, c *checkly.BrowserCheck, model *BrowserCheckResourceModel, diags *diag.Diagnostics) {
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
	model.Locations = stringSetFromAPIEmptyIfUnknown(ctx, c.Locations, prior.Locations, diags)
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

	model.Dependency = scriptDependenciesToList(ctx, c.Dependencies, prior.Dependency, diags)
	model.EnvironmentVariable = environmentVariablesToList(ctx, c.EnvironmentVariables, prior.EnvironmentVariable, diags)
	model.Tags = stringSetFromAPIEmptyIfUnknown(ctx, c.Tags, prior.Tags, diags)

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

	if c.RuntimeID != nil && *c.RuntimeID != "" {
		model.RuntimeID = types.StringValue(*c.RuntimeID)
	} else {
		model.RuntimeID = types.StringNull()
	}

	model.AlertChannelSubscriptions = alertChannelSubscriptionsToSet(ctx, c.AlertChannelSubscriptions, prior.AlertChannelSubscriptions, diags)

	if c.PrivateLocations != nil {
		model.PrivateLocations = setFromStrings(ctx, *c.PrivateLocations, diags)
	} else {
		model.PrivateLocations = setFromStrings(ctx, nil, diags)
	}

	if c.AlertSettings != nil {
		model.AlertSettings = alertSettingsToObject(ctx, *c.AlertSettings, prior.AlertSettings, diags)
	} else if model.AlertSettings.IsUnknown() {
		model.AlertSettings = types.ObjectNull(alertSettingsAttrTypes)
	}

	model.UseGlobalAlertSettings = types.BoolValue(c.UseGlobalAlertSettings)
	model.GroupID = types.Int64Value(c.GroupID)
	model.GroupOrder = types.Int64Value(int64(c.GroupOrder))
	model.RetryStrategy = retryStrategyToObject(ctx, c.RetryStrategy, prior.RetryStrategy, diags)
	model.TriggerIncident = triggerIncidentToObject(ctx, c.TriggerIncident, diags)
	model.Intent = checkIntentToObject(ctx, c.Intent, prior.Intent, diags)
	model.AIAutoRepairEnabled = types.BoolPointerValue(c.AIAutoRepairEnabled)
}
