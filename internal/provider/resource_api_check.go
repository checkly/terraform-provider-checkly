package provider

import (
	"context"
	"fmt"
	"strings"

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
	ID                       types.String `tfsdk:"id"`
	Name                     types.String `tfsdk:"name"`
	Description              types.String `tfsdk:"description"`
	Frequency                types.Int64  `tfsdk:"frequency"`
	FrequencyOffset          types.Int64  `tfsdk:"frequency_offset"`
	Activated                types.Bool   `tfsdk:"activated"`
	Muted                    types.Bool   `tfsdk:"muted"`
	ShouldFail               types.Bool   `tfsdk:"should_fail"`
	RunParallel              types.Bool   `tfsdk:"run_parallel"`
	Locations                types.Set    `tfsdk:"locations"`
	DegradedResponseTime     types.Int64  `tfsdk:"degraded_response_time"`
	MaxResponseTime          types.Int64  `tfsdk:"max_response_time"`
	Tags                     types.Set    `tfsdk:"tags"`
	SetupSnippetID           types.Int64  `tfsdk:"setup_snippet_id"`
	TeardownSnippetID        types.Int64  `tfsdk:"teardown_snippet_id"`
	LocalSetupScript         types.String `tfsdk:"local_setup_script"`
	LocalTeardownScript      types.String `tfsdk:"local_teardown_script"`
	RuntimeID                types.String `tfsdk:"runtime_id"`
	AlertChannelSubscription types.Set    `tfsdk:"alert_channel_subscription"`
	PrivateLocations         types.Set    `tfsdk:"private_locations"`
	AlertSettings            types.Object `tfsdk:"alert_settings"`
	UseGlobalAlertSettings   types.Bool   `tfsdk:"use_global_alert_settings"`
	Request                  types.Set    `tfsdk:"request"`
	GroupID                  types.Int64  `tfsdk:"group_id"`
	GroupOrder               types.Int64  `tfsdk:"group_order"`
	RetryStrategy            types.Object `tfsdk:"retry_strategy"`
	TriggerIncident          types.Set    `tfsdk:"trigger_incident"`
}

// apiCheckRequestModel describes the request block data model.
type apiCheckRequestModel struct {
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

// apiCheckAssertionModel describes the request.assertion block data model.
type apiCheckAssertionModel struct {
	Source     types.String `tfsdk:"source"`
	Property   types.String `tfsdk:"property"`
	Comparison types.String `tfsdk:"comparison"`
	Target     types.String `tfsdk:"target"`
}

// apiCheckBasicAuthModel describes the request.basic_auth attribute data
// model.
type apiCheckBasicAuthModel struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

var apiCheckAssertionAttrTypes = map[string]attr.Type{
	"source":     types.StringType,
	"property":   types.StringType,
	"comparison": types.StringType,
	"target":     types.StringType,
}

var apiCheckBasicAuthAttrTypes = map[string]attr.Type{
	"username": types.StringType,
	"password": types.StringType,
}

var apiCheckRequestAttrTypes = map[string]attr.Type{
	"method":           types.StringType,
	"url":              types.StringType,
	"follow_redirects": types.BoolType,
	"skip_ssl":         types.BoolType,
	"headers":          types.MapType{ElemType: types.StringType},
	"query_parameters": types.MapType{ElemType: types.StringType},
	"body":             types.StringType,
	"body_type":        types.StringType,
	"assertion":        types.SetType{ElemType: types.ObjectType{AttrTypes: apiCheckAssertionAttrTypes}},
	"basic_auth":       types.ObjectType{AttrTypes: apiCheckBasicAuthAttrTypes},
	"ip_family":        types.StringType,
}

func (r *APICheckResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_check"
}

// apiCheckFrequencyAttributeSchema returns the schema for the frequency
// attribute. Unlike the shared frequency attribute helper, the attribute is
// optional+computed with a default of 10 minutes, matching the Checkly API
// default for API checks.
func apiCheckFrequencyAttributeSchema() schema.Int64Attribute {
	allow := allowedValues[int64]{
		{
			Value:       0,
			Description: "high frequency - use `frequency_offset` to define the actual frequency",
		},
		{
			Value:       1,
			Description: "1 minute",
		},
		{
			Value:       2,
			Description: "2 minutes",
		},
		{
			Value:       5,
			Description: "5 minutes",
		},
		{
			Value:       10,
			Description: "10 minutes",
		},
		{
			Value:       15,
			Description: "15 minutes",
		},
		{
			Value:       30,
			Description: "30 minutes",
		},
		{
			Value:       60,
			Description: "1 hour",
		},
		{
			Value:       120,
			Description: "2 hours",
		},
		{
			Value:       180,
			Description: "3 hours",
		},
		{
			Value:       360,
			Description: "6 hours",
		},
		{
			Value:       720,
			Description: "12 hours",
		},
		{
			Value:       1440,
			Description: "24 hours",
		},
	}

	return schema.Int64Attribute{
		Optional:    true,
		Computed:    true,
		Default:     int64default.StaticInt64(10),
		Description: fmt.Sprintf("Controls how often the check should run. Defined in minutes. %s (Default `10`).", allow.String()),
		Validators: []validator.Int64{
			int64validator.OneOf(allow.Values()...),
		},
	}
}

func (r *APICheckResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "API checks allow you to monitor your API endpoints for correctness, performance and availability, " +
			"and set up alerting, so you get a notification when things break or slow down.",
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
			frequencyAttributeName: apiCheckFrequencyAttributeSchema(),
			frequencyOffsetAttributeName: schema.Int64Attribute{
				Optional: true,
				Description: "Spreads the scheduling of the check within its frequency window, defined in seconds. " +
					"When `frequency` is `0` (high frequency), `frequency_offset` is required and defines how often " +
					"the check runs; the allowed values are `10`, `20` and `30` seconds. " +
					"For frequencies of 1 minute and above, `frequency_offset` is optional and offsets the run " +
					"schedule to spread out load; the allowed range is 1 to `frequency * 10` when `frequency` is " +
					"at most `60`, and 1 to `ceil(frequency / 5)` otherwise.",
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
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
			"tags": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "A list of tags for organizing and filtering checks.",
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
		},
		Blocks: map[string]schema.Block{
			alertChannelSubscriptionAttributeName: makeAlertChannelSubscriptionSchema(AlertChannelSubscriptionAttributeSchemaOptions{}),
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
						"method": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "The HTTP method to use for this API check. Possible values are `GET`, `POST`, `PUT`, `HEAD`, `DELETE`, `PATCH`. (Default `GET`).",
						},
						"url": schema.StringAttribute{
							Required:    true,
							Description: "The URL to call.",
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

func (r *APICheckResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		newAPICheckFrequencyOffsetConfigValidator(),
	}
}

func (r *APICheckResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

// ModifyPlan applies default values to the optional+computed attributes
// nested in the request set block. Schema defaults cannot be used for these
// attributes (see the schema for details), so the plan is adjusted here
// instead. Substituting static values keeps every set element fully known
// and stable across plans, which set element correlation requires.
func (r *APICheckResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
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

	request = apiCheckRequestWithDefaults(ctx, request, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("request"), request)...)
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
		resp.Diagnostics.AddError(
			"Unable to Create API Check",
			fmt.Sprintf("Checkly API error: %s", err),
		)
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
		resp.Diagnostics.AddError(
			"Unable to Read API Check",
			fmt.Sprintf("Checkly API error: %s", err),
		)
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
		resp.Diagnostics.AddError(
			"Unable to Update API Check",
			fmt.Sprintf("Checkly API error: %s", err),
		)
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
		resp.Diagnostics.AddError(
			"Unable to Delete API Check",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}
}

func (r *APICheckResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// MoveState enables `moved` blocks from the deprecated checkly_check resource
// type. Only checks with type = "API" can be moved; the schemas are otherwise
// compatible, so the source values are mapped over one-to-one, dropping the
// attributes that do not apply to API checks (type, script,
// environment_variable and ssl_check_domain).
func (r *APICheckResource) MoveState(ctx context.Context) []resource.StateMover {
	var schemaResp resource.SchemaResponse
	(&CheckResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	sourceSchema := schemaResp.Schema

	return []resource.StateMover{
		{
			SourceSchema: &sourceSchema,
			StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				if !strings.HasSuffix(req.SourceProviderAddress, "/checkly") {
					return
				}

				if req.SourceTypeName != "checkly_check" {
					return
				}

				sourceState := sdkv2MoveSourceState(ctx, req, sourceSchema, &resp.Diagnostics)
				if resp.Diagnostics.HasError() {
					return
				}
				if sourceState == nil {
					resp.Diagnostics.AddError(
						"Unable to Move Resource State",
						fmt.Sprintf(
							"The source %s state could not be decoded. The state may "+
								"have been created by an older version of the provider. "+
								"Upgrade the source resource to the current provider "+
								"version before moving it.",
							req.SourceTypeName,
						),
					)
					return
				}

				var source CheckResourceModel
				resp.Diagnostics.Append(sourceState.Get(ctx, &source)...)
				if resp.Diagnostics.HasError() {
					return
				}

				if source.Type.ValueString() != checkly.TypeAPI {
					targetResource := "a dedicated resource for this check type"
					switch source.Type.ValueString() {
					case checkly.TypeBrowser:
						targetResource = "the checkly_browser_check resource"
					case "MULTI_STEP":
						targetResource = "the checkly_multistep_check resource"
					}
					resp.Diagnostics.AddError(
						"Unable to Move Resource State",
						fmt.Sprintf(
							"checkly_api_check can only replace checkly_check resources "+
								"with type = \"API\", but the source check has type = %q. "+
								"Use %s instead.",
							source.Type.ValueString(),
							targetResource,
						),
					)
					return
				}

				target := APICheckResourceModel{
					ID:                       source.ID,
					Name:                     source.Name,
					Description:              source.Description,
					Frequency:                source.Frequency,
					FrequencyOffset:          source.FrequencyOffset,
					Activated:                source.Activated,
					Muted:                    source.Muted,
					ShouldFail:               source.ShouldFail,
					RunParallel:              source.RunParallel,
					Locations:                source.Locations,
					DegradedResponseTime:     source.DegradedResponseTime,
					MaxResponseTime:          source.MaxResponseTime,
					Tags:                     source.Tags,
					SetupSnippetID:           source.SetupSnippetID,
					TeardownSnippetID:        source.TeardownSnippetID,
					LocalSetupScript:         source.LocalSetupScript,
					LocalTeardownScript:      source.LocalTeardownScript,
					RuntimeID:                source.RuntimeID,
					AlertChannelSubscription: source.AlertChannelSubscription,
					PrivateLocations:         source.PrivateLocations,
					AlertSettings:            source.AlertSettings,
					UseGlobalAlertSettings:   source.UseGlobalAlertSettings,
					Request:                  source.Request,
					GroupID:                  source.GroupID,
					GroupOrder:               source.GroupOrder,
					RetryStrategy:            source.RetryStrategy,
					TriggerIncident:          source.TriggerIncident,
				}

				resp.Diagnostics.Append(resp.TargetState.Set(ctx, &target)...)
			},
		},
	}
}

var _ resource.ConfigValidator = (*apiCheckFrequencyOffsetConfigValidator)(nil)

// apiCheckFrequencyOffsetConfigValidator validates the frequency_offset
// attribute against the frequency attribute, mirroring the Checkly API's
// validation rules for API checks: when frequency is 0 (high frequency),
// frequency_offset is required and must be 10, 20 or 30 seconds; for
// frequencies of at most 60 minutes, an optional frequency_offset must be
// between 1 and frequency * 10; for higher frequencies it must be between 1
// and ceil(frequency / 5).
type apiCheckFrequencyOffsetConfigValidator struct{}

// newAPICheckFrequencyOffsetConfigValidator returns a resource config
// validator for the frequency_offset attribute of API checks.
func newAPICheckFrequencyOffsetConfigValidator() resource.ConfigValidator {
	return &apiCheckFrequencyOffsetConfigValidator{}
}

func (v *apiCheckFrequencyOffsetConfigValidator) Description(ctx context.Context) string {
	return "frequency_offset is required and must be 10, 20 or 30 when frequency is 0; " +
		"otherwise it is optional and its allowed range depends on frequency"
}

func (v *apiCheckFrequencyOffsetConfigValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *apiCheckFrequencyOffsetConfigValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var frequency, offset types.Int64

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(frequencyAttributeName), &frequency)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(frequencyOffsetAttributeName), &offset)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if frequency.IsUnknown() || offset.IsUnknown() {
		return
	}

	// The frequency attribute is optional+computed with a static default; an
	// unset frequency validates against the default value.
	frequencyValue := int64(10)
	if !frequency.IsNull() {
		frequencyValue = frequency.ValueInt64()
	}

	if offset.IsNull() {
		if frequencyValue == 0 {
			resp.Diagnostics.AddAttributeError(
				path.Root(frequencyOffsetAttributeName),
				"Missing Required Attribute",
				`"frequency_offset" is required when "frequency" is 0`,
			)
		}
		return
	}

	offsetValue := offset.ValueInt64()

	switch {
	case frequencyValue == 0:
		if offsetValue != 10 && offsetValue != 20 && offsetValue != 30 {
			resp.Diagnostics.AddAttributeError(
				path.Root(frequencyOffsetAttributeName),
				"Invalid Attribute Value",
				`"frequency_offset" must be 10, 20 or 30 when "frequency" is 0`,
			)
		}
	case frequencyValue <= 60:
		if max := frequencyValue * 10; offsetValue < 1 || offsetValue > max {
			resp.Diagnostics.AddAttributeError(
				path.Root(frequencyOffsetAttributeName),
				"Invalid Attribute Value",
				fmt.Sprintf(`"frequency_offset" must be between 1 and %d when "frequency" is %d`, max, frequencyValue),
			)
		}
	default:
		// ceil(frequency / 5) in integer arithmetic.
		if max := (frequencyValue + 4) / 5; offsetValue < 1 || offsetValue > max {
			resp.Diagnostics.AddAttributeError(
				path.Root(frequencyOffsetAttributeName),
				"Invalid Attribute Value",
				fmt.Sprintf(`"frequency_offset" must be between 1 and %d when "frequency" is %d`, max, frequencyValue),
			)
		}
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
	locations := stringsFromSet(ctx, model.Locations, diags)
	if locations == nil {
		locations = []string{}
	}

	privateLocations := stringsFromSet(ctx, model.PrivateLocations, diags)
	if privateLocations == nil {
		privateLocations = []string{}
	}

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
		Request:                   apiCheckRequestFromSet(ctx, model.Request, diags),
		SetupSnippetID:            model.SetupSnippetID.ValueInt64(),
		TearDownSnippetID:         model.TeardownSnippetID.ValueInt64(),
		LocalSetupScript:          localSetupScript,
		LocalTearDownScript:       localTeardownScript,
		GroupID:                   model.GroupID.ValueInt64(),
		GroupOrder:                int(model.GroupOrder.ValueInt64()),
		AlertChannelSubscriptions: alertChannelSubscriptionsFromSet(ctx, model.AlertChannelSubscription, diags),
		PrivateLocations:          &privateLocations,
		RetryStrategy:             retryStrategyFromObject(ctx, model.RetryStrategy, diags),
		TriggerIncident:           triggerIncidentFromSet(ctx, model.TriggerIncident, diags),
	}

	if runtimeID := model.RuntimeID.ValueString(); runtimeID != "" {
		check.RuntimeID = &runtimeID
	}

	return check
}

// apiCheckResolveComputedValues finalizes a planned model after a successful
// create or update API call. All practitioner-configurable values are already
// known at plan time (defaults included; the attributes nested in the request
// block are defaulted in ModifyPlan) and must be kept as planned; only the
// values that may still be unknown in the plan (the ID and the
// optional+computed alert_settings and retry_strategy objects) are resolved
// from the API response.
func apiCheckResolveComputedValues(ctx context.Context, c *checkly.APICheck, model *APICheckResourceModel, diags *diag.Diagnostics) {
	model.ID = types.StringValue(c.ID)

	// When the API reports no alert settings, the planned value is kept;
	// converting the absence of a value would overwrite planned values with
	// empty ones. Unknown values cannot be stored in state and resolve to
	// null.
	if c.AlertSettings != nil {
		model.AlertSettings = alertSettingsToObject(ctx, *c.AlertSettings, model.AlertSettings, diags)
	} else if model.AlertSettings.IsUnknown() {
		model.AlertSettings = types.ObjectNull(alertSettingsAttrTypes)
	}

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
	// The API only persists a configured frequency offset for high frequency
	// scheduling (frequency 0). For regular frequencies the API validates the
	// configured value but stores an automatically assigned offset instead,
	// so echoing the reported value back would cause perpetual drift; the
	// prior value is preserved to stay consistent with the configuration.
	if c.Frequency == 0 {
		model.FrequencyOffset = types.Int64Value(int64(c.FrequencyOffset))
	}
	model.Activated = types.BoolValue(c.Activated)
	model.Muted = types.BoolValue(c.Muted)
	model.ShouldFail = types.BoolValue(c.ShouldFail)
	model.RunParallel = types.BoolValue(c.RunParallel)
	model.Locations = apiCheckStringSetFromAPI(ctx, c.Locations, prior.Locations, diags)
	model.DegradedResponseTime = types.Int64Value(int64(c.DegradedResponseTime))
	model.MaxResponseTime = types.Int64Value(int64(c.MaxResponseTime))
	model.Tags = apiCheckStringSetFromAPI(ctx, c.Tags, prior.Tags, diags)
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
	// When the API reports no alert settings, the prior value is kept;
	// converting the absence of a value would overwrite the stored values
	// with empty ones. Unknown values cannot be stored in state and resolve
	// to null.
	if c.AlertSettings != nil {
		model.AlertSettings = alertSettingsToObject(ctx, *c.AlertSettings, prior.AlertSettings, diags)
	} else if model.AlertSettings.IsUnknown() {
		model.AlertSettings = types.ObjectNull(alertSettingsAttrTypes)
	}
	model.UseGlobalAlertSettings = types.BoolValue(c.UseGlobalAlertSettings)
	model.Request = apiCheckRequestToSet(ctx, c.Request, prior.Request, diags)
	model.GroupID = types.Int64Value(c.GroupID)
	model.GroupOrder = types.Int64Value(int64(c.GroupOrder))
	model.RetryStrategy = retryStrategyToObject(ctx, c.RetryStrategy, prior.RetryStrategy, diags)
	model.TriggerIncident = triggerIncidentToSet(ctx, c.TriggerIncident, diags)
}

// apiCheckStringSetFromAPI converts a string slice from the API to a set of
// strings. An empty value converts to null when the prior value is null, so
// that an omitted collection does not cause drift against an empty API
// response.
func apiCheckStringSetFromAPI(ctx context.Context, values []string, prior types.Set, diags *diag.Diagnostics) types.Set {
	if len(values) == 0 && prior.IsNull() {
		return prior
	}

	return setFromStrings(ctx, values, diags)
}

// apiCheckRequestWithDefaults returns the request block value with default
// values substituted for null and unknown optional+computed attributes.
func apiCheckRequestWithDefaults(ctx context.Context, set types.Set, diags *diag.Diagnostics) types.Set {
	if set.IsNull() || set.IsUnknown() {
		return set
	}

	var models []apiCheckRequestModel
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
		models[i].Assertion = apiCheckAssertionsWithDefaults(ctx, model.Assertion, diags)
	}

	result, d := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: apiCheckRequestAttrTypes}, models)
	diags.Append(d...)

	return result
}

// apiCheckAssertionsWithDefaults returns the request.assertion block value
// with default values substituted for null and unknown optional+computed
// attributes.
func apiCheckAssertionsWithDefaults(ctx context.Context, set types.Set, diags *diag.Diagnostics) types.Set {
	if set.IsNull() || set.IsUnknown() {
		return set
	}

	var models []apiCheckAssertionModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return set
	}

	for i, model := range models {
		models[i].Property = stringOrDefault(model.Property, "")
		models[i].Target = stringOrDefault(model.Target, "")
	}

	result, d := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: apiCheckAssertionAttrTypes}, models)
	diags.Append(d...)

	return result
}

// apiCheckRequestFromSet converts the request block value to its API
// representation. An empty, null or unknown value converts to a zero-valued
// request.
func apiCheckRequestFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) checkly.Request {
	if set.IsNull() || set.IsUnknown() {
		return checkly.Request{}
	}

	var models []apiCheckRequestModel
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
		Headers:         apiCheckKeyValuesFromMap(ctx, model.Headers, diags),
		QueryParameters: apiCheckKeyValuesFromMap(ctx, model.QueryParameters, diags),
		Assertions:      apiCheckAssertionsFromSet(ctx, model.Assertion, diags),
		BasicAuth:       apiCheckBasicAuthFromObject(ctx, model.BasicAuth, diags),
		IPFamily:        model.IPFamily.ValueString(),
	}
}

// apiCheckRequestToSet converts the API representation of an API check
// request config to the request block value. The prior block value (the
// previously stored state) is consulted so that collections omitted from the
// configuration stay null instead of drifting to empty values.
func apiCheckRequestToSet(ctx context.Context, r checkly.Request, prior types.Set, diags *diag.Diagnostics) types.Set {
	elementType := types.ObjectType{AttrTypes: apiCheckRequestAttrTypes}

	priorModel := apiCheckRequestModel{
		Headers:         types.MapNull(types.StringType),
		QueryParameters: types.MapNull(types.StringType),
	}
	if !prior.IsNull() && !prior.IsUnknown() {
		var priorModels []apiCheckRequestModel
		diags.Append(prior.ElementsAs(ctx, &priorModels, false)...)
		if diags.HasError() {
			return types.SetNull(elementType)
		}
		if len(priorModels) > 0 {
			priorModel = priorModels[0]
		}
	}

	model := apiCheckRequestModel{
		Method:          types.StringValue(r.Method),
		URL:             types.StringValue(r.URL),
		FollowRedirects: types.BoolValue(r.FollowRedirects),
		SkipSSL:         types.BoolValue(r.SkipSSL),
		Headers:         apiCheckMapFromKeyValues(ctx, r.Headers, priorModel.Headers, diags),
		QueryParameters: apiCheckMapFromKeyValues(ctx, r.QueryParameters, priorModel.QueryParameters, diags),
		Body:            types.StringValue(r.Body),
		BodyType:        types.StringValue(r.BodyType),
		Assertion:       apiCheckAssertionsToSet(ctx, r.Assertions, diags),
		BasicAuth:       apiCheckBasicAuthToObject(ctx, r.BasicAuth, diags),
		IPFamily:        types.StringValue(r.IPFamily),
	}

	set, d := types.SetValueFrom(ctx, elementType, []apiCheckRequestModel{model})
	diags.Append(d...)

	return set
}

// apiCheckKeyValuesFromMap converts a map of strings to the API's key-value
// pair representation. Null and unknown maps convert to an empty (non-nil)
// slice, which the API stores as "no entries".
func apiCheckKeyValuesFromMap(ctx context.Context, m types.Map, diags *diag.Diagnostics) []checkly.KeyValue {
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

// apiCheckMapFromKeyValues converts the API's key-value pair representation
// to a map of strings. An empty value converts to null when the prior value
// is null, so that an omitted map does not cause drift against an empty API
// response.
func apiCheckMapFromKeyValues(ctx context.Context, kvs []checkly.KeyValue, prior types.Map, diags *diag.Diagnostics) types.Map {
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

// apiCheckAssertionsFromSet converts the request.assertion block value to
// its API representation.
func apiCheckAssertionsFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) []checkly.Assertion {
	assertions := []checkly.Assertion{}

	if set.IsNull() || set.IsUnknown() {
		return assertions
	}

	var models []apiCheckAssertionModel
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

// apiCheckAssertionsToSet converts the API representation of assertions to
// the request.assertion block value.
func apiCheckAssertionsToSet(ctx context.Context, assertions []checkly.Assertion, diags *diag.Diagnostics) types.Set {
	models := make([]apiCheckAssertionModel, len(assertions))
	for i, assertion := range assertions {
		models[i] = apiCheckAssertionModel{
			Source:     types.StringValue(assertion.Source),
			Property:   types.StringValue(assertion.Property),
			Comparison: types.StringValue(assertion.Comparison),
			Target:     types.StringValue(assertion.Target),
		}
	}

	set, d := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: apiCheckAssertionAttrTypes}, models)
	diags.Append(d...)

	return set
}

// apiCheckBasicAuthFromObject converts the request.basic_auth attribute
// value to its API representation. Null and unknown values convert to nil,
// which disables basic authentication.
func apiCheckBasicAuthFromObject(ctx context.Context, obj types.Object, diags *diag.Diagnostics) *checkly.BasicAuth {
	if obj.IsNull() || obj.IsUnknown() {
		return nil
	}

	var model apiCheckBasicAuthModel
	diags.Append(obj.As(ctx, &model, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil
	}

	return &checkly.BasicAuth{
		Username: model.Username.ValueString(),
		Password: model.Password.ValueString(),
	}
}

// apiCheckBasicAuthToObject converts the API representation of basic
// authentication credentials to the request.basic_auth attribute value. The
// API returns no credentials when basic authentication is not configured,
// which converts to null.
func apiCheckBasicAuthToObject(ctx context.Context, ba *checkly.BasicAuth, diags *diag.Diagnostics) types.Object {
	if ba == nil {
		return types.ObjectNull(apiCheckBasicAuthAttrTypes)
	}

	obj, d := types.ObjectValueFrom(ctx, apiCheckBasicAuthAttrTypes, apiCheckBasicAuthModel{
		Username: types.StringValue(ba.Username),
		Password: types.StringValue(ba.Password),
	})
	diags.Append(d...)

	return obj
}
