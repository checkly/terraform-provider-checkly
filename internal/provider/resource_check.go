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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

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

// CheckRequestModel describes the request block data model.
type CheckRequestModel struct {
	Method          types.String `tfsdk:"method"`
	URL             types.String `tfsdk:"url"`
	FollowRedirects types.Bool   `tfsdk:"follow_redirects"`
	SkipSSL         types.Bool   `tfsdk:"skip_ssl"`
	Headers         types.Map    `tfsdk:"headers"`
	QueryParameters types.Map    `tfsdk:"query_parameters"`
	Body            types.String `tfsdk:"body"`
	BodyType        types.String `tfsdk:"body_type"`
	Assertion       types.Set    `tfsdk:"assertion"`
	BasicAuth       types.Object `tfsdk:"basic_auth"`
	IPFamily        types.String `tfsdk:"ip_family"`
}

// CheckAssertionModel describes the request.assertion block data model.
type CheckAssertionModel struct {
	Source     types.String `tfsdk:"source"`
	Property   types.String `tfsdk:"property"`
	Comparison types.String `tfsdk:"comparison"`
	Target     types.String `tfsdk:"target"`
}

// CheckBasicAuthModel describes the request.basic_auth attribute data model.
type CheckBasicAuthModel struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

var checkAssertionAttrTypes = map[string]attr.Type{
	"source":     types.StringType,
	"property":   types.StringType,
	"comparison": types.StringType,
	"target":     types.StringType,
}

var checkBasicAuthAttrTypes = map[string]attr.Type{
	"username": types.StringType,
	"password": types.StringType,
}

var checkRequestAttrTypes = map[string]attr.Type{
	"method":           types.StringType,
	"url":              types.StringType,
	"follow_redirects": types.BoolType,
	"skip_ssl":         types.BoolType,
	"headers":          types.MapType{ElemType: types.StringType},
	"query_parameters": types.MapType{ElemType: types.StringType},
	"body":             types.StringType,
	"body_type":        types.StringType,
	"assertion":        types.SetType{ElemType: types.ObjectType{AttrTypes: checkAssertionAttrTypes}},
	"basic_auth":       types.ObjectType{AttrTypes: checkBasicAuthAttrTypes},
	"ip_family":        types.StringType,
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
			"request": schema.SetNestedBlock{
				Description: "An API check might have one request config.",
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
						"method": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "The HTTP method to use for this API check. Possible values are `GET`, `POST`, `PUT`, `HEAD`, `DELETE`, `PATCH`. (Default `GET`).",
						},
						"url": schema.StringAttribute{
							Required: true,
						},
						"follow_redirects": schema.BoolAttribute{
							Optional: true,
							Computed: true,
						},
						"skip_ssl": schema.BoolAttribute{
							Optional: true,
							Computed: true,
						},
						"headers": schema.MapAttribute{
							Optional:    true,
							ElementType: types.StringType,
						},
						"query_parameters": schema.MapAttribute{
							Optional:    true,
							ElementType: types.StringType,
						},
						"body": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "The body of the request.",
						},
						"body_type": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "The `Content-Type` header of the request. Possible values `NONE`, `JSON`, `FORM`, `RAW`, and `GRAPHQL`. (Default `NONE`).",
							Validators: []validator.String{
								stringvalidator.OneOf("NONE", "JSON", "FORM", "RAW", "GRAPHQL"),
							},
						},
						"basic_auth": schema.SingleNestedAttribute{
							Optional:    true,
							Description: "Set up HTTP basic authentication (username & password).",
							Attributes: map[string]schema.Attribute{
								"username": schema.StringAttribute{
									Required: true,
								},
								"password": schema.StringAttribute{
									Required: true,
								},
							},
						},
						"ip_family": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "IP Family to be used when executing the api check. The value can be either IPv4 or IPv6. (Default `IPv4`).",
							Validators: []validator.String{
								stringvalidator.OneOf("IPv4", "IPv6"),
							},
						},
					},
					Blocks: map[string]schema.Block{
						"assertion": schema.SetNestedBlock{
							Description: "A request can have multiple assertions.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"source": schema.StringAttribute{
										Required:    true,
										Description: "The source of the asserted value. Possible values `STATUS_CODE`, `JSON_BODY`, `HEADERS`, `TEXT_BODY`, and `RESPONSE_TIME`.",
									},
									"property": schema.StringAttribute{
										Optional: true,
										Computed: true,
									},
									"comparison": schema.StringAttribute{
										Required:    true,
										Description: "The type of comparison to be executed between expected and actual value of the assertion. Possible values `EQUALS`, `NOT_EQUALS`, `HAS_KEY`, `NOT_HAS_KEY`, `HAS_VALUE`, `NOT_HAS_VALUE`, `IS_EMPTY`, `NOT_EMPTY`, `GREATER_THAN`, `LESS_THAN`, `CONTAINS`, `NOT_CONTAINS`, `IS_NULL`, and `NOT_NULL`.",
									},
									"target": schema.StringAttribute{
										Optional: true,
										Computed: true,
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

	request = checkRequestWithDefaults(ctx, request, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("request"), request)...)
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

	privateLocations := stringsFromSet(ctx, model.PrivateLocations, diags)
	if privateLocations == nil {
		privateLocations = []string{}
	}
	check.PrivateLocations = &privateLocations

	// The request and frequency_offset properties only apply to API checks.
	// Leaving them out for other check types prevents them from causing
	// configuration drift.
	if check.Type == checkly.TypeAPI {
		check.Request = checkRequestFromSet(ctx, model.Request, diags)
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
	model.Locations = checkStringSetFromAPI(ctx, c.Locations, prior.Locations, diags)
	model.Script = types.StringValue(c.Script)
	model.DegradedResponseTime = types.Int64Value(int64(c.DegradedResponseTime))
	model.MaxResponseTime = types.Int64Value(int64(c.MaxResponseTime))
	model.EnvironmentVariable = environmentVariablesToList(ctx, c.EnvironmentVariables, prior.EnvironmentVariable, diags)
	model.Tags = checkStringSetFromAPI(ctx, c.Tags, prior.Tags, diags)
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
		model.Request = checkRequestToSet(ctx, c.Request, prior.Request, diags)
	}
	model.GroupID = types.Int64Value(c.GroupID)
	model.GroupOrder = types.Int64Value(int64(c.GroupOrder))
	model.RetryStrategy = retryStrategyToObject(ctx, c.RetryStrategy, prior.RetryStrategy, diags)
	model.TriggerIncident = triggerIncidentToSet(ctx, c.TriggerIncident, diags)
}

// checkStringSetFromAPI converts a string slice from the API to a set of
// strings. An empty value converts to null when the prior value is null, so
// that an omitted collection does not cause drift against an empty API
// response.
func checkStringSetFromAPI(ctx context.Context, values []string, prior types.Set, diags *diag.Diagnostics) types.Set {
	if len(values) == 0 && prior.IsNull() {
		return prior
	}

	return setFromStrings(ctx, values, diags)
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

// checkRequestWithDefaults returns the request block value with default
// values substituted for null and unknown optional+computed attributes.
func checkRequestWithDefaults(ctx context.Context, set types.Set, diags *diag.Diagnostics) types.Set {
	if set.IsNull() || set.IsUnknown() {
		return set
	}

	var models []CheckRequestModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return set
	}

	for i, model := range models {
		models[i].Method = stringOrDefault(model.Method, "GET")
		models[i].FollowRedirects = boolOrDefault(model.FollowRedirects, false)
		models[i].SkipSSL = boolOrDefault(model.SkipSSL, false)
		models[i].Body = stringOrDefault(model.Body, "")
		models[i].BodyType = stringOrDefault(model.BodyType, "NONE")
		models[i].IPFamily = stringOrDefault(model.IPFamily, "IPv4")
		models[i].Assertion = checkAssertionsWithDefaults(ctx, model.Assertion, diags)
	}

	result, d := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: checkRequestAttrTypes}, models)
	diags.Append(d...)

	return result
}

// checkAssertionsWithDefaults returns the request.assertion block value with
// default values substituted for null and unknown optional+computed
// attributes.
func checkAssertionsWithDefaults(ctx context.Context, set types.Set, diags *diag.Diagnostics) types.Set {
	if set.IsNull() || set.IsUnknown() {
		return set
	}

	var models []CheckAssertionModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return set
	}

	for i, model := range models {
		models[i].Property = stringOrDefault(model.Property, "")
		models[i].Target = stringOrDefault(model.Target, "")
	}

	result, d := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: checkAssertionAttrTypes}, models)
	diags.Append(d...)

	return result
}

// checkRequestFromSet converts the request block value to its API
// representation. An empty, null or unknown value converts to a zero-valued
// request.
func checkRequestFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) checkly.Request {
	if set.IsNull() || set.IsUnknown() {
		return checkly.Request{}
	}

	var models []CheckRequestModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() || len(models) == 0 {
		return checkly.Request{}
	}

	model := models[0]

	return checkly.Request{
		Method:          model.Method.ValueString(),
		URL:             model.URL.ValueString(),
		FollowRedirects: model.FollowRedirects.ValueBool(),
		SkipSSL:         model.SkipSSL.ValueBool(),
		Body:            model.Body.ValueString(),
		BodyType:        model.BodyType.ValueString(),
		Headers:         checkKeyValuesFromMap(ctx, model.Headers, diags),
		QueryParameters: checkKeyValuesFromMap(ctx, model.QueryParameters, diags),
		Assertions:      checkAssertionsFromSet(ctx, model.Assertion, diags),
		BasicAuth:       checkBasicAuthFromObject(ctx, model.BasicAuth, diags),
		IPFamily:        model.IPFamily.ValueString(),
	}
}

// checkRequestToSet converts the API representation of an API check request
// config to the request block value. The prior block value (the previously
// stored state) is consulted so that collections omitted from the
// configuration stay null instead of drifting to empty values.
func checkRequestToSet(ctx context.Context, r checkly.Request, prior types.Set, diags *diag.Diagnostics) types.Set {
	elementType := types.ObjectType{AttrTypes: checkRequestAttrTypes}

	priorModel := CheckRequestModel{
		Headers:         types.MapNull(types.StringType),
		QueryParameters: types.MapNull(types.StringType),
	}
	if !prior.IsNull() && !prior.IsUnknown() {
		var priorModels []CheckRequestModel
		diags.Append(prior.ElementsAs(ctx, &priorModels, false)...)
		if diags.HasError() {
			return types.SetNull(elementType)
		}
		if len(priorModels) > 0 {
			priorModel = priorModels[0]
		}
	}

	model := CheckRequestModel{
		Method:          types.StringValue(r.Method),
		URL:             types.StringValue(r.URL),
		FollowRedirects: types.BoolValue(r.FollowRedirects),
		SkipSSL:         types.BoolValue(r.SkipSSL),
		Headers:         checkMapFromKeyValues(ctx, r.Headers, priorModel.Headers, diags),
		QueryParameters: checkMapFromKeyValues(ctx, r.QueryParameters, priorModel.QueryParameters, diags),
		Body:            types.StringValue(r.Body),
		BodyType:        types.StringValue(r.BodyType),
		Assertion:       checkAssertionsToSet(ctx, r.Assertions, diags),
		BasicAuth:       checkBasicAuthToObject(ctx, r.BasicAuth, diags),
		IPFamily:        types.StringValue(r.IPFamily),
	}

	set, d := types.SetValueFrom(ctx, elementType, []CheckRequestModel{model})
	diags.Append(d...)

	return set
}

// checkKeyValuesFromMap converts a map of strings to the API's key-value
// pair representation. Null and unknown maps convert to an empty (non-nil)
// slice, which the API stores as "no entries".
func checkKeyValuesFromMap(ctx context.Context, m types.Map, diags *diag.Diagnostics) []checkly.KeyValue {
	kvs := []checkly.KeyValue{}

	if m.IsNull() || m.IsUnknown() {
		return kvs
	}

	var values map[string]string
	diags.Append(m.ElementsAs(ctx, &values, false)...)
	if diags.HasError() {
		return kvs
	}

	for key, value := range values {
		kvs = append(kvs, checkly.KeyValue{
			Key:   key,
			Value: value,
		})
	}

	return kvs
}

// checkMapFromKeyValues converts the API's key-value pair representation to
// a map of strings. An empty value converts to null when the prior value is
// null, so that an omitted map does not cause drift against an empty API
// response.
func checkMapFromKeyValues(ctx context.Context, kvs []checkly.KeyValue, prior types.Map, diags *diag.Diagnostics) types.Map {
	if len(kvs) == 0 && prior.IsNull() {
		return prior
	}

	values := map[string]string{}
	for _, kv := range kvs {
		values[kv.Key] = kv.Value
	}

	m, d := types.MapValueFrom(ctx, types.StringType, values)
	diags.Append(d...)

	return m
}

// checkAssertionsFromSet converts the request.assertion block value to its
// API representation.
func checkAssertionsFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) []checkly.Assertion {
	assertions := []checkly.Assertion{}

	if set.IsNull() || set.IsUnknown() {
		return assertions
	}

	var models []CheckAssertionModel
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

// checkAssertionsToSet converts the API representation of assertions to the
// request.assertion block value.
func checkAssertionsToSet(ctx context.Context, assertions []checkly.Assertion, diags *diag.Diagnostics) types.Set {
	models := make([]CheckAssertionModel, len(assertions))
	for i, assertion := range assertions {
		models[i] = CheckAssertionModel{
			Source:     types.StringValue(assertion.Source),
			Property:   types.StringValue(assertion.Property),
			Comparison: types.StringValue(assertion.Comparison),
			Target:     types.StringValue(assertion.Target),
		}
	}

	set, d := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: checkAssertionAttrTypes}, models)
	diags.Append(d...)

	return set
}

// checkBasicAuthFromObject converts the request.basic_auth attribute value
// to its API representation. Null and unknown values convert to nil, which
// disables basic authentication.
func checkBasicAuthFromObject(ctx context.Context, obj types.Object, diags *diag.Diagnostics) *checkly.BasicAuth {
	if obj.IsNull() || obj.IsUnknown() {
		return nil
	}

	var model CheckBasicAuthModel
	diags.Append(obj.As(ctx, &model, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil
	}

	return &checkly.BasicAuth{
		Username: model.Username.ValueString(),
		Password: model.Password.ValueString(),
	}
}

// checkBasicAuthToObject converts the API representation of basic
// authentication credentials to the request.basic_auth attribute value. The
// API returns no credentials when basic authentication is not configured,
// which converts to null.
func checkBasicAuthToObject(ctx context.Context, ba *checkly.BasicAuth, diags *diag.Diagnostics) types.Object {
	if ba == nil {
		return types.ObjectNull(checkBasicAuthAttrTypes)
	}

	obj, d := types.ObjectValueFrom(ctx, checkBasicAuthAttrTypes, CheckBasicAuthModel{
		Username: types.StringValue(ba.Username),
		Password: types.StringValue(ba.Password),
	})
	diags.Append(d...)

	return obj
}
