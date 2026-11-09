package provider

import (
	"context"
	"fmt"
	"slices"

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

	"github.com/checkly/checkly-go-sdk/v2"
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
	Request                  types.Object `tfsdk:"request"`
	GroupID                  types.Int64  `tfsdk:"group_id"`
	GroupOrder               types.Int64  `tfsdk:"group_order"`
	RetryStrategy            types.Object `tfsdk:"retry_strategy"`
	TriggerIncident          types.Object `tfsdk:"trigger_incident"`
	Intent                   types.Object `tfsdk:"intent"`
	AIAutoRepairEnabled      types.Bool   `tfsdk:"ai_auto_repair_enabled"`
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
				Description: "A valid piece of Node.js JavaScript code describing a browser interaction with the Puppeteer/Playwright framework or a reference to an external JavaScript file. Required when `type` is `BROWSER` or `MULTI_STEP`; has no effect when it is `API`. Updating a browser or multi-step check clears the script path and dependencies that the Checkly CLI sets for multi-file scripts.",
			},
			"degraded_response_time": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(15000),
				// https://checklyhq.com/docs/api-checks/limits/
				Validators: []validator.Int64{
					int64validator.Between(0, 30000),
				},
				Description: "The response time in milliseconds starting from which a check should be considered degraded. Possible values are between 0 and 30000. Only relevant when `type` is `API`. (Default `15000`).",
			},
			"max_response_time": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(30000),
				// https://checklyhq.com/docs/api-checks/limits/
				Validators: []validator.Int64{
					int64validator.Between(0, 30000),
				},
				Description: "The response time in milliseconds starting from which a check should be considered failing. Possible values are between 0 and 30000. Only relevant when `type` is `API`. (Default `30000`).",
			},
			"tags": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "A list of tags for organizing and filtering checks.",
			},
			"ssl_check_domain": schema.StringAttribute{
				Optional:    true,
				Description: "A valid fully qualified domain name (FQDN) to check its SSL certificate. Only relevant when `type` is `BROWSER`.",
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
				Description: "A valid piece of Node.js code to run in the setup phase of an API check.",
			},
			"local_teardown_script": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "A valid piece of Node.js code to run in the teardown phase of an API check.",
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
			triggerIncidentAttributeName: makeTriggerIncidentSchema(),
			alertSettingsAttributeName:   makeAlertSettingsAttributeSchema(AlertSettingsAttributeSchemaOptions{}),
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
			doubleCheckAttributeName:         doubleCheckAttributeSchema(),
			checkIntentAttributeName:         makeCheckIntentSchema(),
			aiAutoRepairEnabledAttributeName: makeAIAutoRepairEnabledSchema(),
			"request": apiCheckRequestAttributeSchema(apiCheckRequestAttributeSchemaOptions{
				Description: "An API check might have one request config. Only relevant when `type` is `API`.",
			}),
		},
		Blocks: map[string]schema.Block{
			environmentVariableAttributeName: makeEnvironmentVariableSchema(EnvironmentVariableAttributeSchemaOptions{
				Description: "Insert environment variables into the runtime environment. Only relevant for browser checks. Use global environment variables whenever possible.",
			}),
			alertChannelSubscriptionAttributeName: makeAlertChannelSubscriptionSchema(AlertChannelSubscriptionAttributeSchemaOptions{}),
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
	if checkType.ValueString() == string(checkly.CheckTypeBrowser) &&
		!frequency.IsNull() && !frequency.IsUnknown() &&
		frequency.ValueInt64() == 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root(frequencyAttributeName),
			"Invalid Attribute Combination",
			"property frequency could only be 0 for API checks",
		)
	}

	// SSL certificate checks are only available for browser checks. An
	// empty domain passes the API, which doesn't store it either.
	if checkType.ValueString() == string(checkly.CheckTypeAPI) &&
		!sslCheckDomain.IsNull() && !sslCheckDomain.IsUnknown() {
		if sslCheckDomain.ValueString() != "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("ssl_check_domain"),
				"Invalid Attribute Combination",
				"ssl_check_domain is allowed only for Browser checks",
			)
		} else {
			addIgnoredCheckAttributeWarning(&resp.Diagnostics, "ssl_check_domain", checkType.ValueString())
		}
	}

	// The API ignores environment variables of API checks, so they would
	// show as a change in every plan.
	if checkType.ValueString() == string(checkly.CheckTypeAPI) &&
		!environmentVariables.IsUnknown() && len(environmentVariables.Elements()) > 0 {
		resp.Diagnostics.AddAttributeWarning(
			path.Root(environmentVariableAttributeName),
			"Environment Variables Have No Effect",
			"API checks do not use environment variables, so the API does not store them and every plan "+
				"shows them as a change. Remove the environment_variable blocks from this check.",
		)
	}

	for _, name := range checkIgnoredAttributes[checkly.CheckType(checkType.ValueString())] {
		var value attr.Value
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(name), &value)...)
		if value == nil || value.IsNull() || value.IsUnknown() {
			continue
		}
		addIgnoredCheckAttributeWarning(&resp.Diagnostics, name, checkType.ValueString())
	}
}

// checkAPIOnlyAttributes are the checkly_check attributes that only API
// checks use.
var checkAPIOnlyAttributes = []string{
	"request",
	frequencyOffsetAttributeName,
	"degraded_response_time",
	"max_response_time",
	"setup_snippet_id",
	"teardown_snippet_id",
	"local_setup_script",
	"local_teardown_script",
}

// checkIgnoredAttributes lists, for each check type, the checkly_check
// attributes that the API endpoint of that type has no field for (see
// apiCheckModelFromCheckModel and its siblings), so a configured value only
// changes the state. environment_variable and ssl_check_domain on API checks
// have diagnostics of their own.
var checkIgnoredAttributes = map[checkly.CheckType][]string{
	checkly.CheckTypeAPI:       {"script"},
	checkly.CheckTypeBrowser:   checkAPIOnlyAttributes,
	checkly.CheckTypeMultiStep: append(slices.Clone(checkAPIOnlyAttributes), "ssl_check_domain"),
}

// addIgnoredCheckAttributeWarning warns that a configured attribute has no
// effect on checks of the given type.
func addIgnoredCheckAttributeWarning(diags *diag.Diagnostics, name string, checkType string) {
	detail := fmt.Sprintf(`%s checks don't use %q: checkly_check doesn't send it to the API, so changing it only changes the state. `+
		`Remove it from this check, or move the check to %s, which only has the attributes of its check type.`,
		checkType, name, checkResourceTypeForCheckType(checkly.CheckType(checkType)))
	if name == frequencyOffsetAttributeName {
		detail += ` High frequency scheduling ("frequency" 0), which needs "frequency_offset", is only available for API checks.`
	}
	diags.AddAttributeWarning(path.Root(name), "Attribute Has No Effect", detail)
}

// UpgradeState upgrades state written by the provider's SDK v2
// implementation; see sdkv2StateUpgraders.
func (r *CheckResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return sdkv2StateUpgraders(ctx, r)
}

func (r *CheckResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

// ModifyPlan applies default values to the optional+computed attributes of
// the request assertions. Schema defaults cannot be used for attributes
// nested in set elements (see the schema for details), so the plan is
// adjusted here instead. Substituting static values keeps every set element
// fully known and stable across plans, which set element correlation
// requires.
func (r *CheckResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planNestedDefaults(ctx, req, resp, path.Root("request"), objectWithSetDefaults("assertion", assertionsWithPropertyAndTargetDefaults))

	// An empty script is reported in the plan when it is known there.
	// Create and Update check again for a script known only at apply time.
	if req.Plan.Raw.IsNull() {
		return
	}
	var planned CheckResourceModel
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("type"), &planned.Type)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("script"), &planned.Script)...)
	if resp.Diagnostics.HasError() || planned.Type.IsNull() || planned.Type.IsUnknown() || planned.Script.IsUnknown() {
		return
	}
	validateCheckScript(&planned, &resp.Diagnostics)
}

func (r *CheckResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CheckResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	validateCheckScript(&plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	r.validateCheckRuntimeSupport(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.createCheck(apiCtx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
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
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *CheckResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CheckResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	validateCheckScript(&plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	r.validateCheckRuntimeSupport(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.updateCheck(apiCtx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
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
func checkStateMovers[T any](ctx context.Context, checkType checkly.CheckType, toTarget func(source *CheckResourceModel) T) []resource.StateMover {
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

				if sourceType := checkly.CheckType(source.Type.ValueString()); sourceType != checkType {
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
func checkResourceTypeForCheckType(checkType checkly.CheckType) string {
	switch checkType {
	case checkly.CheckTypeAPI:
		return "checkly_api_check"
	case checkly.CheckTypeBrowser:
		return "checkly_browser_check"
	case checkly.CheckTypeMultiStep:
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
func (r *CheckResource) validateCheckRuntimeSupport(ctx context.Context, model *CheckResourceModel, diags *diag.Diagnostics) {
	runtimeID := model.RuntimeID.ValueString()
	if model.Type.ValueString() != string(checkly.CheckTypeMultiStep) || runtimeID == "" {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	checkRuntime, err := r.client.GetRuntime(apiCtx, runtimeID)
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
			fmt.Sprintf("runtime %s does not support MULTI_STEP checks", runtimeID),
		)
	}
}

// validateCheckScript rejects a browser or multi-step check with an empty
// planned script. The API would accept the null that an empty script is sent
// as, and clear a stored script. The planned value is what counts: the
// configuration can't show a value known only at apply time, and with
// ignore_changes = [script] the plan holds the stored script. ModifyPlan
// calls it for a script known at plan time, and Create and Update before any
// request.
func validateCheckScript(model *CheckResourceModel, diags *diag.Diagnostics) {
	checkType := model.Type.ValueString()
	if checkType == string(checkly.CheckTypeAPI) || model.Script.ValueString() != "" {
		return
	}

	diags.AddAttributeError(
		path.Root("script"),
		"Missing Check Script",
		fmt.Sprintf(`A %s check needs a script, but "script" is empty: it is not set, null, or resolved to an empty string. Set it to the code of the check.`, checkType),
	)
}

// createCheck creates the check through the endpoint of its planned type.
// With an error in diags, it returns nil values without sending a request.
func (r *CheckResource) createCheck(ctx context.Context, model *CheckResourceModel, diags *diag.Diagnostics) (checkly.Check, error) {
	switch checkType := checkly.CheckType(model.Type.ValueString()); checkType {
	case checkly.CheckTypeAPI:
		target := apiCheckModelFromCheckModel(model)
		check := apiCheckFromModel(ctx, &target, diags)
		if diags.HasError() {
			return nil, nil
		}
		return r.client.CreateAPICheck(ctx, check)
	case checkly.CheckTypeBrowser:
		target := browserCheckModelFromCheckModel(model)
		check := browserCheckFromModel(ctx, &target, diags)
		if diags.HasError() {
			return nil, nil
		}
		return r.client.CreateBrowserCheck(ctx, check)
	case checkly.CheckTypeMultiStep:
		target := multistepCheckModelFromCheckModel(model)
		check := multistepCheckFromModel(ctx, &target, diags)
		if diags.HasError() {
			return nil, nil
		}
		return r.client.CreateMultiStepCheck(ctx, check)
	default:
		return nil, plannedCheckTypeError(checkType)
	}
}

// updateCheck updates the check through the endpoint of its planned type,
// which also changes the type of a stored check of another type. The update
// clears the script path and dependencies of browser and multi-step checks,
// which checkly_check has no attributes for. With an error in diags, it
// returns nil values without sending a request.
func (r *CheckResource) updateCheck(ctx context.Context, model *CheckResourceModel, diags *diag.Diagnostics) (checkly.Check, error) {
	id := model.ID.ValueString()

	switch checkType := checkly.CheckType(model.Type.ValueString()); checkType {
	case checkly.CheckTypeAPI:
		target := apiCheckModelFromCheckModel(model)
		check := apiCheckFromModel(ctx, &target, diags)
		if diags.HasError() {
			return nil, nil
		}
		return r.client.UpdateAPICheck(ctx, id, check)
	case checkly.CheckTypeBrowser:
		target := browserCheckModelFromCheckModel(model)
		check := browserCheckFromModel(ctx, &target, diags)
		if diags.HasError() {
			return nil, nil
		}
		return r.client.UpdateBrowserCheck(ctx, id, check)
	case checkly.CheckTypeMultiStep:
		target := multistepCheckModelFromCheckModel(model)
		check := multistepCheckFromModel(ctx, &target, diags)
		if diags.HasError() {
			return nil, nil
		}
		return r.client.UpdateMultiStepCheck(ctx, id, check)
	default:
		return nil, plannedCheckTypeError(checkType)
	}
}

// plannedCheckTypeError reports a planned type that checkly_check does not
// manage, which the schema validation normally rejects first.
func plannedCheckTypeError(checkType checkly.CheckType) error {
	return fmt.Errorf("checkly_check only manages API, BROWSER and MULTI_STEP checks, and this check has type %q", checkType)
}

// unmanagedCheckTypeError reports a stored check of a type that checkly_check
// does not manage, and the resource type that does.
func unmanagedCheckTypeError(check checkly.Check) error {
	var checkType checkly.CheckType
	var resourceType string

	switch check.(type) {
	case *checkly.PlaywrightCheck:
		checkType, resourceType = checkly.CheckTypePlaywright, "checkly_playwright_check_suite"
	case *checkly.HeartbeatMonitor:
		checkType, resourceType = checkly.CheckTypeHeartbeat, "checkly_heartbeat_monitor"
	case *checkly.URLMonitor:
		checkType, resourceType = checkly.CheckTypeURL, "checkly_url_monitor"
	case *checkly.TCPMonitor:
		checkType, resourceType = checkly.CheckTypeTCP, "checkly_tcp_monitor"
	case *checkly.DNSMonitor:
		checkType, resourceType = checkly.CheckTypeDNS, "checkly_dns_monitor"
	case *checkly.ICMPMonitor:
		checkType, resourceType = checkly.CheckTypeICMP, "checkly_icmp_monitor"
	case *checkly.GRPCMonitor:
		checkType, resourceType = checkly.CheckTypeGRPC, "checkly_grpc_monitor"
	case *checkly.SSLMonitor:
		checkType, resourceType = checkly.CheckTypeSSL, "checkly_ssl_monitor"
	case *checkly.TracerouteMonitor:
		checkType, resourceType = checkly.CheckTypeTraceroute, "checkly_traceroute_monitor"
	default:
		// A check type added by a later version of the SDK.
		checkType, resourceType = checkly.CheckType(fmt.Sprintf("%T", check)), "the dedicated resource type for its check type"
	}

	return fmt.Errorf("checkly_check only manages API, BROWSER and MULTI_STEP checks, and this check has type %q: use %s instead", checkType, resourceType)
}

// commonCheckFields holds the fields that the API, browser and multi-step
// check types share, so that checkly_check reads them once.
type commonCheckFields struct {
	ID                        string
	Name                      string
	Description               *string
	Frequency                 int
	Activated                 bool
	Muted                     bool
	ShouldFail                bool
	RunParallel               bool
	Locations                 []string
	Tags                      []string
	AlertSettings             *checkly.AlertSettings
	UseGlobalAlertSettings    bool
	GroupID                   int64
	GroupOrder                int
	AlertChannelSubscriptions []checkly.AlertChannelSubscription
	PrivateLocations          *[]string
	RuntimeID                 *string
	RetryStrategy             *checkly.RetryStrategy
	TriggerIncident           *checkly.IncidentTrigger
	Intent                    *checkly.CheckIntent
	AIAutoRepairEnabled       *bool
}

// commonCheckFieldsFrom returns the shared fields of an API, browser or
// multi-step check, and an error for a check of another type.
func commonCheckFieldsFrom(check checkly.Check) (commonCheckFields, error) {
	switch c := check.(type) {
	case *checkly.APICheck:
		return commonCheckFields{
			ID:                        c.ID,
			Name:                      c.Name,
			Description:               c.Description,
			Frequency:                 c.Frequency,
			Activated:                 c.Activated,
			Muted:                     c.Muted,
			ShouldFail:                c.ShouldFail,
			RunParallel:               c.RunParallel,
			Locations:                 c.Locations,
			Tags:                      c.Tags,
			AlertSettings:             c.AlertSettings,
			UseGlobalAlertSettings:    c.UseGlobalAlertSettings,
			GroupID:                   c.GroupID,
			GroupOrder:                c.GroupOrder,
			AlertChannelSubscriptions: c.AlertChannelSubscriptions,
			PrivateLocations:          c.PrivateLocations,
			RuntimeID:                 c.RuntimeID,
			RetryStrategy:             c.RetryStrategy,
			TriggerIncident:           c.TriggerIncident,
			Intent:                    c.Intent,
			AIAutoRepairEnabled:       c.AIAutoRepairEnabled,
		}, nil
	case *checkly.BrowserCheck:
		return commonCheckFields{
			ID:                        c.ID,
			Name:                      c.Name,
			Description:               c.Description,
			Frequency:                 c.Frequency,
			Activated:                 c.Activated,
			Muted:                     c.Muted,
			ShouldFail:                c.ShouldFail,
			RunParallel:               c.RunParallel,
			Locations:                 c.Locations,
			Tags:                      c.Tags,
			AlertSettings:             c.AlertSettings,
			UseGlobalAlertSettings:    c.UseGlobalAlertSettings,
			GroupID:                   c.GroupID,
			GroupOrder:                c.GroupOrder,
			AlertChannelSubscriptions: c.AlertChannelSubscriptions,
			PrivateLocations:          c.PrivateLocations,
			RuntimeID:                 c.RuntimeID,
			RetryStrategy:             c.RetryStrategy,
			TriggerIncident:           c.TriggerIncident,
			Intent:                    c.Intent,
			AIAutoRepairEnabled:       c.AIAutoRepairEnabled,
		}, nil
	case *checkly.MultiStepCheck:
		return commonCheckFields{
			ID:                        c.ID,
			Name:                      c.Name,
			Description:               c.Description,
			Frequency:                 c.Frequency,
			Activated:                 c.Activated,
			Muted:                     c.Muted,
			ShouldFail:                c.ShouldFail,
			RunParallel:               c.RunParallel,
			Locations:                 c.Locations,
			Tags:                      c.Tags,
			AlertSettings:             c.AlertSettings,
			UseGlobalAlertSettings:    c.UseGlobalAlertSettings,
			GroupID:                   c.GroupID,
			GroupOrder:                c.GroupOrder,
			AlertChannelSubscriptions: c.AlertChannelSubscriptions,
			PrivateLocations:          c.PrivateLocations,
			RuntimeID:                 c.RuntimeID,
			RetryStrategy:             c.RetryStrategy,
			TriggerIncident:           c.TriggerIncident,
			Intent:                    c.Intent,
			AIAutoRepairEnabled:       c.AIAutoRepairEnabled,
		}, nil
	default:
		return commonCheckFields{}, unmanagedCheckTypeError(check)
	}
}

// checkResolveComputedValues finalizes a planned model after a successful
// create or update API call. All practitioner-configurable values are already
// known at plan time (defaults included; the attributes nested in the request
// assertions are defaulted in ModifyPlan) and must be kept as planned; only the
// values that may still be unknown in the plan (the ID and the
// optional+computed alert_settings and retry_strategy objects) are resolved
// from the API response.
func checkResolveComputedValues(ctx context.Context, check checkly.Check, model *CheckResourceModel, diags *diag.Diagnostics) {
	c, err := commonCheckFieldsFrom(check)
	if err != nil {
		diags.AddError("Unable to Read Check", err.Error())
		return
	}

	model.ID = types.StringValue(c.ID)
	model.AlertSettings = alertSettingsToObject(ctx, valueOrZero(c.AlertSettings), model.AlertSettings, diags)
	model.RetryStrategy = retryStrategyToObject(ctx, c.RetryStrategy, model.RetryStrategy, diags)
}

// checkToModel updates the model to match the API representation of the
// check. Values already present in the model act as the prior state for the
// prior-aware conversions.
//
// The per-type API representations leave out the attributes that don't apply
// to their type, which have no effect on such a check. Those attributes keep
// their prior value, or their schema default when there is none, as on import.
func checkToModel(ctx context.Context, check checkly.Check, model *CheckResourceModel, diags *diag.Diagnostics) {
	c, err := commonCheckFieldsFrom(check)
	if err != nil {
		diags.AddError("Unable to Read Check", err.Error())
		return
	}

	prior := *model

	model.ID = types.StringValue(c.ID)
	model.Name = types.StringValue(c.Name)
	model.Description = types.StringValue(valueOrZero(c.Description))
	model.Frequency = types.Int64Value(int64(c.Frequency))
	model.Activated = types.BoolValue(c.Activated)
	model.Muted = types.BoolValue(c.Muted)
	model.ShouldFail = types.BoolValue(c.ShouldFail)
	model.RunParallel = types.BoolValue(c.RunParallel)
	model.Locations = stringSetFromAPIEmptyIfUnknown(ctx, c.Locations, prior.Locations, diags)
	model.Tags = stringSetFromAPIEmptyIfUnknown(ctx, c.Tags, prior.Tags, diags)
	model.RuntimeID = stringValueOrNull(valueOrZero(c.RuntimeID))
	model.AlertChannelSubscription = alertChannelSubscriptionsToSet(ctx, c.AlertChannelSubscriptions, diags)
	model.PrivateLocations = setFromStrings(ctx, valueOrZero(c.PrivateLocations), diags)
	model.AlertSettings = alertSettingsToObject(ctx, valueOrZero(c.AlertSettings), prior.AlertSettings, diags)
	model.UseGlobalAlertSettings = types.BoolValue(c.UseGlobalAlertSettings)
	// The SDK's check types have no double_check, so the value in the model
	// stands, and a value that was never set, as on import, is false. A change
	// made outside Terraform shows up through retry_strategy.
	model.DoubleCheck = types.BoolValue(prior.DoubleCheck.ValueBool())
	model.GroupID = types.Int64Value(c.GroupID)
	model.GroupOrder = types.Int64Value(int64(c.GroupOrder))
	model.RetryStrategy = retryStrategyToObject(ctx, c.RetryStrategy, prior.RetryStrategy, diags)
	model.TriggerIncident = triggerIncidentToObject(ctx, c.TriggerIncident, diags)
	model.Intent = checkIntentToObject(ctx, c.Intent, prior.Intent, diags)
	model.AIAutoRepairEnabled = types.BoolPointerValue(c.AIAutoRepairEnabled)

	switch typed := check.(type) {
	case *checkly.APICheck:
		model.Type = types.StringValue(string(checkly.CheckTypeAPI))
		// The API only tracks a frequency offset for high frequency scheduling
		// (frequency 0). Otherwise the prior value is preserved: it is either
		// null or 0, both of which mean the offset is disabled.
		if typed.Frequency == 0 {
			model.FrequencyOffset = types.Int64Value(int64(typed.FrequencyOffset))
		}
		model.Script = stringOrDefault(prior.Script, "")
		model.DegradedResponseTime = types.Int64Value(int64(typed.DegradedResponseTime))
		model.MaxResponseTime = types.Int64Value(int64(typed.MaxResponseTime))
		// The API does not store environment variables of API checks, so
		// configured ones show as a change in every plan; ValidateConfig warns
		// about them.
		model.EnvironmentVariable = environmentVariablesToList(ctx, nil, prior.EnvironmentVariable, diags)
		model.SetupSnippetID = types.Int64Value(typed.SetupSnippetID)
		model.TeardownSnippetID = types.Int64Value(typed.TearDownSnippetID)
		model.LocalSetupScript = types.StringValue(valueOrZero(typed.LocalSetupScript))
		model.LocalTeardownScript = types.StringValue(valueOrZero(typed.LocalTearDownScript))
		model.Request = apiCheckRequestToObject(ctx, typed.Request, prior.Request, diags)
	case *checkly.BrowserCheck:
		model.Type = types.StringValue(string(checkly.CheckTypeBrowser))
		model.Script = types.StringValue(typed.Script)
		model.EnvironmentVariable = environmentVariablesToList(ctx, typed.EnvironmentVariables, prior.EnvironmentVariable, diags)
		// The prior value is preserved when the API reports no domain.
		if typed.SSLCheckDomain != "" {
			model.SSLCheckDomain = types.StringValue(typed.SSLCheckDomain)
		}
		checkKeepAPICheckAttributes(model)
	case *checkly.MultiStepCheck:
		model.Type = types.StringValue(string(checkly.CheckTypeMultiStep))
		model.Script = types.StringValue(typed.Script)
		model.EnvironmentVariable = environmentVariablesToList(ctx, typed.EnvironmentVariables, prior.EnvironmentVariable, diags)
		checkKeepAPICheckAttributes(model)
	}
}

// checkKeepAPICheckAttributes resolves the attributes that browser and
// multi-step checks don't have to their schema defaults when the model has no
// value for them. The request attribute has no default, and stays as it is.
func checkKeepAPICheckAttributes(model *CheckResourceModel) {
	model.DegradedResponseTime = int64OrDefault(model.DegradedResponseTime, 15000)
	model.MaxResponseTime = int64OrDefault(model.MaxResponseTime, 30000)
	model.SetupSnippetID = int64OrDefault(model.SetupSnippetID, 0)
	model.TeardownSnippetID = int64OrDefault(model.TeardownSnippetID, 0)
	model.LocalSetupScript = stringOrDefault(model.LocalSetupScript, "")
	model.LocalTeardownScript = stringOrDefault(model.LocalTeardownScript, "")
}

// stringOrDefault returns the given value, or the given default value when
// the value is null or unknown. It is used to apply default values during
// plan modification to attributes nested in set elements, where schema
// defaults cannot be used.
func stringOrDefault(value types.String, defaultValue string) types.String {
	if value.IsNull() || value.IsUnknown() {
		return types.StringValue(defaultValue)
	}
	return value
}

// int64OrDefault returns the given value, or the given default value when the
// value is null or unknown.
func int64OrDefault(value types.Int64, defaultValue int64) types.Int64 {
	if value.IsNull() || value.IsUnknown() {
		return types.Int64Value(defaultValue)
	}
	return value
}
