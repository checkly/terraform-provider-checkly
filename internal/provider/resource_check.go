package provider

import (
	"context"
	"fmt"

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
)

var (
	_ resource.Resource                     = (*CheckResource)(nil)
	_ resource.ResourceWithConfigure        = (*CheckResource)(nil)
	_ resource.ResourceWithUpgradeState     = (*CheckResource)(nil)
	_ resource.ResourceWithImportState      = (*CheckResource)(nil)
	_ resource.ResourceWithConfigValidators = (*CheckResource)(nil)
	_ resource.ResourceWithValidateConfig   = (*CheckResource)(nil)
	_ resource.ResourceWithModifyPlan       = (*CheckResource)(nil)
)

// NewCheckResource returns a new checkly_check resource.
func NewCheckResource() resource.Resource {
	return &CheckResource{}
}

// CheckResource implements the checkly_check resource.
type CheckResource struct {
	client checkly.Client
}

// CheckResourceModel describes the resource data model.
type CheckResourceModel struct {
	ID                       types.String `tfsdk:"id"`
	Name                     types.String `tfsdk:"name"`
	Description              types.String `tfsdk:"description"`
	Type                     types.String `tfsdk:"type"`
	Frequency                types.Int64  `tfsdk:"frequency"`
	FrequencyOffset          types.Int64  `tfsdk:"frequency_offset"`
	Activated                types.Bool   `tfsdk:"activated"`
	Muted                    types.Bool   `tfsdk:"muted"`
	ShouldFail               types.Bool   `tfsdk:"should_fail"`
	RunParallel              types.Bool   `tfsdk:"run_parallel"`
	Locations                types.Set    `tfsdk:"locations"`
	Script                   types.String `tfsdk:"script"`
	DegradedResponseTime     types.Int64  `tfsdk:"degraded_response_time"`
	MaxResponseTime          types.Int64  `tfsdk:"max_response_time"`
	EnvironmentVariable      types.List   `tfsdk:"environment_variable"`
	Tags                     types.Set    `tfsdk:"tags"`
	SSLCheckDomain           types.String `tfsdk:"ssl_check_domain"`
	SetupSnippetID           types.Int64  `tfsdk:"setup_snippet_id"`
	TeardownSnippetID        types.Int64  `tfsdk:"teardown_snippet_id"`
	LocalSetupScript         types.String `tfsdk:"local_setup_script"`
	LocalTeardownScript      types.String `tfsdk:"local_teardown_script"`
	RuntimeID                types.String `tfsdk:"runtime_id"`
	AlertChannelSubscription types.Set    `tfsdk:"alert_channel_subscription"`
	PrivateLocations         types.Set    `tfsdk:"private_locations"`
	AlertSettings            types.Object `tfsdk:"alert_settings"`
	UseGlobalAlertSettings   types.Bool   `tfsdk:"use_global_alert_settings"`
	DoubleCheck              types.Bool   `tfsdk:"double_check"`
	Request                  types.Set    `tfsdk:"request"`
	GroupID                  types.Int64  `tfsdk:"group_id"`
	GroupOrder               types.Int64  `tfsdk:"group_order"`
	RetryStrategy            types.Object `tfsdk:"retry_strategy"`
	TriggerIncident          types.Set    `tfsdk:"trigger_incident"`
}

func (r *CheckResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_check"
}

func (r *CheckResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     sdkv2SchemaVersion + 1,
		Description: "Checks allows you to monitor key webapp flows, backend API's and set up alerting, so you get a notification when things break or slow down.",
		DeprecationMessage: "The `checkly_check` resource is deprecated in favor of dedicated resource types for each check type: " +
			"`checkly_api_check`, `checkly_browser_check`, and `checkly_multistep_check`. " +
			"These expose only the properties relevant to their check type. " +
			"Use a `moved` block to migrate existing resources without recreating them. " +
			"When migrating an API check, note that the default `degraded_response_time` and `max_response_time` " +
			"changed from 15000/30000 to 5000/20000 — set them explicitly to keep the previous thresholds. " +
			"`checkly_api_check` has no `environment_variable` blocks, which never had an effect on API checks.",
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
			"type": schema.StringAttribute{
				Required:    true,
				Description: "The type of the check. Possible values are `API`, `BROWSER`, and `MULTI_STEP`.",
				Validators: []validator.String{
					stringvalidator.OneOf("API", "BROWSER", "MULTI_STEP"),
				},
			},
			frequencyAttributeName: makeFrequencyAttributeSchema(FrequencyAttributeSchemaOptions{
				AllowHighFrequency: true,
			}),
			frequencyOffsetAttributeName: makeFrequencyOffsetAttributeSchema(FrequencyOffsetAttributeSchemaOptions{
				Disclaimer: "Only relevant when `type` is `API`.",
			}),
			"activated": schema.BoolAttribute{
				Required:    true,
				Description: "Determines if the check is running or not. Possible values `true`, and `false`.",
			},
			"muted": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Determines if any notifications will be sent out when a check fails/degrades/recovers.",
			},
			"should_fail": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Allows to invert the behaviour of when a check is considered to fail. Allows for validating error status like 404.",
			},
			"run_parallel": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Determines if the check should run in all selected locations in parallel or round-robin.",
			},
			"locations": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "An array of one or more data center locations where to run the this check. (Default [\"us-east-1\"])",
			},
			"script": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "A valid piece of Node.js JavaScript code describing a browser interaction with the Puppeteer/Playwright framework or a reference to an external JavaScript file.",
			},
			"degraded_response_time": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(15000),
				// https://checklyhq.com/docs/api-checks/limits/
				Validators: []validator.Int64{
					int64validator.Between(0, 30000),
				},
				Description: "The response time in milliseconds starting from which a check should be considered degraded. Possible values are between 0 and 30000. (Default `15000`).",
			},
			"max_response_time": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(30000),
				// https://checklyhq.com/docs/api-checks/limits/
				Validators: []validator.Int64{
					int64validator.Between(0, 30000),
				},
				Description: "The response time in milliseconds starting from which a check should be considered failing. Possible values are between 0 and 30000. (Default `30000`).",
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
			alertSettingsAttributeName: makeAlertSettingsAttributeSchema(AlertSettingsAttributeSchemaOptions{}),
			"use_global_alert_settings": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "When true, the account level alert settings will be used, not the alert setting defined on this check.",
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
				DoubleCheck:                true,
			}),
			doubleCheckAttributeName: doubleCheckAttributeSchema(),
		},
		Blocks: map[string]schema.Block{
			environmentVariableAttributeName: makeEnvironmentVariableSchema(EnvironmentVariableAttributeSchemaOptions{
				Description: "Insert environment variables into the runtime environment. Only relevant for browser checks. Use global environment variables whenever possible.",
			}),
			alertChannelSubscriptionAttributeName: makeAlertChannelSubscriptionSchema(AlertChannelSubscriptionAttributeSchemaOptions{}),
			"request": apiCheckRequestBlockSchema(apiCheckRequestBlockSchemaOptions{
				Description: "An API check might have one request config.",
			}),
			triggerIncidentAttributeName: makeTriggerIncidentSchema(),
		},
	}
}

func (r *CheckResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		newFrequencyOffsetConfigValidator(),
	}
}

func (r *CheckResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var checkType types.String
	var frequency types.Int64
	var sslCheckDomain types.String
	var environmentVariables types.List

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("type"), &checkType)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(frequencyAttributeName), &frequency)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("ssl_check_domain"), &sslCheckDomain)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(environmentVariableAttributeName), &environmentVariables)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if checkType.IsNull() || checkType.IsUnknown() {
		return
	}

	// High frequency scheduling (frequency 0) is only available for API
	// checks.
	if checkType.ValueString() == checkly.TypeBrowser &&
		!frequency.IsNull() && !frequency.IsUnknown() &&
		frequency.ValueInt64() == 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root(frequencyAttributeName),
			"Invalid Attribute Combination",
			"property frequency could only be 0 for API checks",
		)
	}

	// SSL certificate checks are only available for browser checks.
	if checkType.ValueString() == checkly.TypeAPI &&
		!sslCheckDomain.IsNull() && !sslCheckDomain.IsUnknown() &&
		sslCheckDomain.ValueString() != "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("ssl_check_domain"),
			"Invalid Attribute Combination",
			"ssl_check_domain is allowed only for Browser checks",
		)
	}

	// The API ignores environment variables of API checks, so they would
	// show as a change in every plan.
	if checkType.ValueString() == checkly.TypeAPI &&
		!environmentVariables.IsUnknown() && len(environmentVariables.Elements()) > 0 {
		resp.Diagnostics.AddAttributeWarning(
			path.Root(environmentVariableAttributeName),
			"Environment Variables Have No Effect",
			"API checks do not use environment variables, so the API does not store them and every plan "+
				"shows them as a change. Remove the environment_variable blocks from this check.",
		)
	}
}

// UpgradeState upgrades state written by the provider's SDK v2
// implementation; see sdkv2StateUpgraders.
func (r *CheckResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return sdkv2StateUpgraders(ctx, r)
}

func (r *CheckResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

// ModifyPlan applies default values to the optional+computed attributes
// nested in the request set block. Schema defaults cannot be used for these
// attributes (see the schema for details), so the plan is adjusted here
// instead. Substituting static values keeps every set element fully known
// and stable across plans, which set element correlation requires.
func (r *CheckResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planNestedDefaults(ctx, req, resp, path.Root("request"), apiCheckRequestWithDefaults)
}

func (r *CheckResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CheckResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	check := checkFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	r.validateCheckRuntimeSupport(ctx, &check, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.CreateCheck(apiCtx, check)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Check",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	checkResolveComputedValues(ctx, result, &plan, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CheckResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CheckResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	check, err := r.client.GetCheck(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Read Check",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	checkToModel(ctx, check, &state, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *CheckResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CheckResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	check := checkFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	r.validateCheckRuntimeSupport(ctx, &check, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.UpdateCheck(apiCtx, check.ID, check)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Check",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	checkResolveComputedValues(ctx, result, &plan, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CheckResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CheckResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	err := r.client.DeleteCheck(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Delete Check",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}
}

func (r *CheckResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// checkStateMovers returns the state movers of the dedicated check resource
// type targetTypeName for `moved` blocks from checkly_check. Only checks of
// type checkType can be moved; toTarget maps the source model to the target
// model, which is stored as the target state.
func checkStateMovers[T any](ctx context.Context, checkType string, toTarget func(source *CheckResourceModel) T) []resource.StateMover {
	targetTypeName := checkResourceTypeForCheckType(checkType)
	var schemaResp resource.SchemaResponse
	(&CheckResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	sourceSchema := schemaResp.Schema

	return []resource.StateMover{
		{
			SourceSchema: &sourceSchema,
			StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				sourceState := moveSourceState(ctx, req, sourceSchema, &resp.Diagnostics, "checkly_check")
				if sourceState == nil {
					return
				}

				var source CheckResourceModel
				resp.Diagnostics.Append(sourceState.Get(ctx, &source)...)
				if resp.Diagnostics.HasError() {
					return
				}

				if sourceType := source.Type.ValueString(); sourceType != checkType {
					resp.Diagnostics.AddError(
						"Unable to Move Resource State",
						fmt.Sprintf(
							"The source checkly_check resource has type %q and cannot "+
								"be moved to %s, which only supports %s checks. Move the "+
								"resource to %s instead.",
							sourceType,
							targetTypeName,
							checkType,
							checkResourceTypeForCheckType(sourceType),
						),
					)
					return
				}

				target := toTarget(&source)
				resp.Diagnostics.Append(resp.TargetState.Set(ctx, &target)...)
			},
		},
	}
}

// checkResourceTypeForCheckType maps a checkly_check type value to the name
// of the dedicated resource type that replaces it.
func checkResourceTypeForCheckType(checkType string) string {
	switch checkType {
	case checkly.TypeAPI:
		return "checkly_api_check"
	case checkly.TypeBrowser:
		return "checkly_browser_check"
	case "MULTI_STEP":
		return "checkly_multistep_check"
	default:
		return "the dedicated resource type for its check type"
	}
}

// validateCheckRuntimeSupport validates that the configured runtime supports
// multistep checks when the check has a runtime ID set. Note that if the
// runtime ID is coming from the account defaults or group, then it is not
// validated. Adding validation there as well would be a nice improvement,
// though.
func (r *CheckResource) validateCheckRuntimeSupport(ctx context.Context, check *checkly.Check, diags *diag.Diagnostics) {
	if check.Type != "MULTI_STEP" || check.RuntimeID == nil {
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

// checkFromModel converts the resource data model to its API representation.
func checkFromModel(ctx context.Context, model *CheckResourceModel, diags *diag.Diagnostics) checkly.Check {
	// An empty description is not tracked by the Checkly API; send null
	// instead so that updates clear a previously configured description.
	var description *string
	if value := model.Description.ValueString(); value != "" {
		description = &value
	}

	check := checkly.Check{
		ID:                        model.ID.ValueString(),
		Name:                      model.Name.ValueString(),
		Description:               description,
		Type:                      model.Type.ValueString(),
		Frequency:                 int(model.Frequency.ValueInt64()),
		Activated:                 model.Activated.ValueBool(),
		Muted:                     model.Muted.ValueBool(),
		ShouldFail:                model.ShouldFail.ValueBool(),
		RunParallel:               model.RunParallel.ValueBool(),
		Locations:                 stringsFromSet(ctx, model.Locations, diags),
		Script:                    model.Script.ValueString(),
		DegradedResponseTime:      int(model.DegradedResponseTime.ValueInt64()),
		MaxResponseTime:           int(model.MaxResponseTime.ValueInt64()),
		EnvironmentVariables:      environmentVariablesFromList(ctx, model.EnvironmentVariable, diags),
		Tags:                      stringsFromSet(ctx, model.Tags, diags),
		SSLCheckDomain:            model.SSLCheckDomain.ValueString(),
		SetupSnippetID:            model.SetupSnippetID.ValueInt64(),
		TearDownSnippetID:         model.TeardownSnippetID.ValueInt64(),
		LocalSetupScript:          model.LocalSetupScript.ValueString(),
		LocalTearDownScript:       model.LocalTeardownScript.ValueString(),
		AlertSettings:             alertSettingsFromObject(ctx, model.AlertSettings, diags),
		UseGlobalAlertSettings:    model.UseGlobalAlertSettings.ValueBool(),
		DoubleCheck:               model.DoubleCheck.ValueBool(),
		GroupID:                   model.GroupID.ValueInt64(),
		GroupOrder:                int(model.GroupOrder.ValueInt64()),
		AlertChannelSubscriptions: alertChannelSubscriptionsFromSet(ctx, model.AlertChannelSubscription, diags),
		RetryStrategy:             retryStrategyFromObject(ctx, model.RetryStrategy, diags),
		TriggerIncident:           triggerIncidentFromSet(ctx, model.TriggerIncident, diags),
	}

	// The locations property is serialized even when empty.
	if check.Locations == nil {
		check.Locations = []string{}
	}

	if runtimeID := model.RuntimeID.ValueString(); runtimeID != "" {
		check.RuntimeID = &runtimeID
	}

	privateLocations := stringsFromSetOrEmpty(ctx, model.PrivateLocations, diags)
	check.PrivateLocations = &privateLocations

	// The request and frequency_offset properties only apply to API checks.
	// Leaving them out for other check types prevents them from causing
	// configuration drift.
	if check.Type == checkly.TypeAPI {
		check.Request = apiCheckRequestFromSet(ctx, model.Request, diags)
		check.FrequencyOffset = int(model.FrequencyOffset.ValueInt64())
	}

	return check
}

// checkResolveComputedValues finalizes a planned model after a successful
// create or update API call. All practitioner-configurable values are already
// known at plan time (defaults included; the attributes nested in the request
// block are defaulted in ModifyPlan) and must be kept as planned; only the
// values that may still be unknown in the plan (the ID and the
// optional+computed alert_settings and retry_strategy objects) are resolved
// from the API response.
func checkResolveComputedValues(ctx context.Context, c *checkly.Check, model *CheckResourceModel, diags *diag.Diagnostics) {
	model.ID = types.StringValue(c.ID)
	model.AlertSettings = alertSettingsToObject(ctx, c.AlertSettings, model.AlertSettings, diags)
	model.RetryStrategy = retryStrategyToObject(ctx, c.RetryStrategy, model.RetryStrategy, diags)
}

// checkToModel updates the model to match the API representation of the
// check. Values already present in the model act as the prior state for the
// prior-aware conversions.
func checkToModel(ctx context.Context, c *checkly.Check, model *CheckResourceModel, diags *diag.Diagnostics) {
	prior := *model

	model.ID = types.StringValue(c.ID)
	model.Name = types.StringValue(c.Name)
	if c.Description != nil {
		model.Description = types.StringValue(*c.Description)
	} else {
		model.Description = types.StringValue("")
	}
	model.Type = types.StringValue(c.Type)
	model.Frequency = types.Int64Value(int64(c.Frequency))
	// The API only tracks a frequency offset for high frequency scheduling
	// (frequency 0). Otherwise the prior value is preserved: it is either
	// null or 0, both of which mean the offset is disabled.
	if c.Frequency == 0 {
		model.FrequencyOffset = types.Int64Value(int64(c.FrequencyOffset))
	}
	model.Activated = types.BoolValue(c.Activated)
	model.Muted = types.BoolValue(c.Muted)
	model.ShouldFail = types.BoolValue(c.ShouldFail)
	model.RunParallel = types.BoolValue(c.RunParallel)
	model.Locations = stringSetFromAPIEmptyIfUnknown(ctx, c.Locations, prior.Locations, diags)
	model.Script = types.StringValue(c.Script)
	model.DegradedResponseTime = types.Int64Value(int64(c.DegradedResponseTime))
	model.MaxResponseTime = types.Int64Value(int64(c.MaxResponseTime))
	model.EnvironmentVariable = environmentVariablesToList(ctx, c.EnvironmentVariables, prior.EnvironmentVariable, diags)
	model.Tags = stringSetFromAPIEmptyIfUnknown(ctx, c.Tags, prior.Tags, diags)
	// The ssl_check_domain property is only supported for browser checks.
	// The prior value is preserved when the API reports no domain.
	if c.Type == checkly.TypeBrowser && c.SSLCheckDomain != "" {
		model.SSLCheckDomain = types.StringValue(c.SSLCheckDomain)
	}
	model.SetupSnippetID = types.Int64Value(c.SetupSnippetID)
	model.TeardownSnippetID = types.Int64Value(c.TearDownSnippetID)
	model.LocalSetupScript = types.StringValue(c.LocalSetupScript)
	model.LocalTeardownScript = types.StringValue(c.LocalTearDownScript)
	if c.RuntimeID != nil && *c.RuntimeID != "" {
		model.RuntimeID = types.StringValue(*c.RuntimeID)
	} else {
		model.RuntimeID = types.StringNull()
	}
	model.AlertChannelSubscription = alertChannelSubscriptionsToSet(ctx, c.AlertChannelSubscriptions, diags)
	if c.PrivateLocations != nil {
		model.PrivateLocations = setFromStrings(ctx, *c.PrivateLocations, diags)
	} else {
		model.PrivateLocations = setFromStrings(ctx, nil, diags)
	}
	model.AlertSettings = alertSettingsToObject(ctx, c.AlertSettings, prior.AlertSettings, diags)
	model.UseGlobalAlertSettings = types.BoolValue(c.UseGlobalAlertSettings)
	model.DoubleCheck = types.BoolValue(c.DoubleCheck)
	// The request property only applies to API checks; the API reports a
	// zero-valued request config for other check types. For non-API checks
	// the prior value is preserved: a configured request block is ignored by
	// the API, and overwriting it here would cause drift.
	if c.Type == checkly.TypeAPI {
		model.Request = apiCheckRequestToSet(ctx, c.Request, prior.Request, diags)
	}
	model.GroupID = types.Int64Value(c.GroupID)
	model.GroupOrder = types.Int64Value(int64(c.GroupOrder))
	model.RetryStrategy = retryStrategyToObject(ctx, c.RetryStrategy, prior.RetryStrategy, diags)
	model.TriggerIncident = triggerIncidentToSet(ctx, c.TriggerIncident, diags)
}

// stringOrDefault returns the given value, or the given default value when
// the value is null or unknown. It is used to apply default values during
// plan modification to attributes nested in set blocks, where schema
// defaults cannot be used.
func stringOrDefault(value types.String, defaultValue string) types.String {
	if value.IsNull() || value.IsUnknown() {
		return types.StringValue(defaultValue)
	}
	return value
}

// boolOrDefault returns the given value, or the given default value when the
// value is null or unknown. It is used to apply default values during plan
// modification to attributes nested in set blocks, where schema defaults
// cannot be used.
func boolOrDefault(value types.Bool, defaultValue bool) types.Bool {
	if value.IsNull() || value.IsUnknown() {
		return types.BoolValue(defaultValue)
	}
	return value
}
