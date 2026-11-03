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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
)

var (
	_ resource.Resource                     = (*TCPMonitorResource)(nil)
	_ resource.ResourceWithConfigure        = (*TCPMonitorResource)(nil)
	_ resource.ResourceWithUpgradeState     = (*TCPMonitorResource)(nil)
	_ resource.ResourceWithConfigValidators = (*TCPMonitorResource)(nil)
	_ resource.ResourceWithImportState      = (*TCPMonitorResource)(nil)
	_ resource.ResourceWithMoveState        = (*TCPMonitorResource)(nil)
	_ resource.ResourceWithModifyPlan       = (*TCPMonitorResource)(nil)
)

// NewTCPMonitorResource returns a new checkly_tcp_monitor resource.
func NewTCPMonitorResource() resource.Resource {
	return &TCPMonitorResource{}
}

// TCPMonitorResource implements the checkly_tcp_monitor resource.
type TCPMonitorResource struct {
	client checkly.Client
}

// TCPMonitorResourceModel describes the resource data model.
type TCPMonitorResourceModel struct {
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
	RuntimeID                types.String `tfsdk:"runtime_id"`
	PrivateLocations         types.Set    `tfsdk:"private_locations"`
	AlertSettings            types.Object `tfsdk:"alert_settings"`
	UseGlobalAlertSettings   types.Bool   `tfsdk:"use_global_alert_settings"`
	GroupID                  types.Int64  `tfsdk:"group_id"`
	GroupOrder               types.Int64  `tfsdk:"group_order"`
	RetryStrategy            types.Object `tfsdk:"retry_strategy"`
	Request                  types.Set    `tfsdk:"request"`
	AlertChannelSubscription types.Set    `tfsdk:"alert_channel_subscription"`
	TriggerIncident          types.Set    `tfsdk:"trigger_incident"`
}

// TCPMonitorRequestModel describes the request block data model.
type TCPMonitorRequestModel struct {
	Hostname  types.String `tfsdk:"hostname"`
	Port      types.Int64  `tfsdk:"port"`
	Data      types.String `tfsdk:"data"`
	Assertion types.Set    `tfsdk:"assertion"`
	IPFamily  types.String `tfsdk:"ip_family"`
}

// TCPMonitorAssertionModel describes the request.assertion block data model.
type TCPMonitorAssertionModel struct {
	Source     types.String `tfsdk:"source"`
	Property   types.String `tfsdk:"property"`
	Comparison types.String `tfsdk:"comparison"`
	Target     types.String `tfsdk:"target"`
}

var tcpMonitorAssertionAttrTypes = map[string]attr.Type{
	"source":     types.StringType,
	"property":   types.StringType,
	"comparison": types.StringType,
	"target":     types.StringType,
}

var tcpMonitorRequestAttrTypes = map[string]attr.Type{
	"hostname":  types.StringType,
	"port":      types.Int64Type,
	"data":      types.StringType,
	"assertion": types.SetType{ElemType: types.ObjectType{AttrTypes: tcpMonitorAssertionAttrTypes}},
	"ip_family": types.StringType,
}

func (r *TCPMonitorResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tcp_monitor"
}

func (r *TCPMonitorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     sdkv2SchemaVersion + 1,
		Description: "TCP checks allow you to monitor remote endpoints at a lower level.",
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
				Description: "The name of the check.",
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
				Description: "Allows to invert the behaviour of when a check is considered to fail.",
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
				Description: "An array of one or more data center locations where to run the this check.",
			},
			"degraded_response_time": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(4000),
				Description: "The response time in milliseconds starting from which a check should be considered degraded. Possible values are between 0 and 5000. (Default `4000`).",
				Validators: []validator.Int64{
					int64validator.Between(0, 5000),
				},
			},
			"max_response_time": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(5000),
				Description: "The response time in milliseconds starting from which a check should be considered failing. Possible values are between 0 and 5000. (Default `5000`).",
				Validators: []validator.Int64{
					int64validator.Between(0, 5000),
				},
			},
			"tags": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "A list of tags for organizing and filtering checks.",
			},
			"runtime_id": schema.StringAttribute{
				Optional:    true,
				Description: "The ID of the runtime to use for this check.",
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
				SupportsOnlyOnNetworkError: false,
			}),
		},
		Blocks: map[string]schema.Block{
			"request": schema.SetNestedBlock{
				Description: "The parameters for the TCP connection.",
				Validators: []validator.Set{
					// Terraform represents an unconfigured nested block set as
					// an empty set rather than null, so a size check is needed
					// in addition to IsRequired to enforce presence.
					setvalidator.IsRequired(),
					setvalidator.SizeAtLeast(1),
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
						"hostname": schema.StringAttribute{
							Required:    true,
							Description: "The hostname or IP to connect to. Do not include a scheme or a port in this value.",
						},
						"port": schema.Int64Attribute{
							Required:    true,
							Description: "The port number to connect to.",
						},
						"data": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "The data to send to the target host.",
						},
						"ip_family": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "The IP family to use when executing the TCP check. The value can be either `IPv4` or `IPv6`. (Default `IPv4`).",
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
										Description: "The source of the asserted value. Possible values are `RESPONSE_DATA` and `RESPONSE_TIME`.",
									},
									"property": schema.StringAttribute{
										Optional: true,
										Computed: true,
									},
									"comparison": schema.StringAttribute{
										Required:    true,
										Description: "The type of comparison to be executed between expected and actual value of the assertion. Possible values are `EQUALS`, `NOT_EQUALS`, `HAS_KEY`, `NOT_HAS_KEY`, `HAS_VALUE`, `NOT_HAS_VALUE`, `IS_EMPTY`, `NOT_EMPTY`, `GREATER_THAN`, `LESS_THAN`, `CONTAINS`, `NOT_CONTAINS`, `IS_NULL`, and `NOT_NULL`.",
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
			alertChannelSubscriptionAttributeName: makeAlertChannelSubscriptionSchema(AlertChannelSubscriptionAttributeSchemaOptions{
				Monitor: true,
			}),
			triggerIncidentAttributeName: makeTriggerIncidentSchema(),
		},
	}
}

func (r *TCPMonitorResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		newFrequencyOffsetConfigValidator(),
	}
}

// UpgradeState upgrades state written by the provider's SDK v2
// implementation; see sdkv2StateUpgraders.
func (r *TCPMonitorResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return sdkv2StateUpgraders(ctx, r)
}

func (r *TCPMonitorResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

// ModifyPlan applies default values to the optional+computed attributes
// nested in the request set block. Schema defaults cannot be used for these
// attributes (see the schema for details), so the plan is adjusted here
// instead. Substituting static values keeps every set element fully known
// and stable across plans, which set element correlation requires.
func (r *TCPMonitorResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
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

	request = tcpMonitorRequestWithDefaults(ctx, request, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("request"), request)...)
}

// tcpMonitorRequestWithDefaults returns the request block value with default
// values substituted for null and unknown optional+computed attributes.
func tcpMonitorRequestWithDefaults(ctx context.Context, set types.Set, diags *diag.Diagnostics) types.Set {
	if set.IsNull() || set.IsUnknown() {
		return set
	}

	var models []TCPMonitorRequestModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return set
	}

	for i, model := range models {
		models[i].Data = stringOrDefault(model.Data, "")
		models[i].IPFamily = stringOrDefault(model.IPFamily, "IPv4")
		models[i].Assertion = tcpMonitorAssertionsWithDefaults(ctx, model.Assertion, diags)
	}

	result, d := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: tcpMonitorRequestAttrTypes}, models)
	diags.Append(d...)

	return result
}

// tcpMonitorAssertionsWithDefaults returns the request.assertion block value
// with default values substituted for null and unknown optional+computed
// attributes.
func tcpMonitorAssertionsWithDefaults(ctx context.Context, set types.Set, diags *diag.Diagnostics) types.Set {
	if set.IsNull() || set.IsUnknown() {
		return set
	}

	var models []TCPMonitorAssertionModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return set
	}

	for i, model := range models {
		models[i].Property = stringOrDefault(model.Property, "")
		models[i].Target = stringOrDefault(model.Target, "")
	}

	result, d := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: tcpMonitorAssertionAttrTypes}, models)
	diags.Append(d...)

	return result
}

func (r *TCPMonitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TCPMonitorResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	monitor := tcpMonitorFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.CreateTCPMonitor(apiCtx, monitor)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create TCP Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	// State is written even when the conversion produces error diagnostics.
	// The monitor already exists remotely at this point, and returning
	// without saving state would leave it unmanaged.
	plan.ID = types.StringValue(result.ID)
	tcpMonitorToModel(ctx, result, &plan, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TCPMonitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TCPMonitorResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	monitor, err := r.client.GetTCPMonitor(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Read TCP Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	tcpMonitorToModel(ctx, monitor, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TCPMonitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TCPMonitorResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	monitor := tcpMonitorFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.UpdateTCPMonitor(apiCtx, plan.ID.ValueString(), monitor)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update TCP Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	tcpMonitorToModel(ctx, result, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TCPMonitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TCPMonitorResourceModel

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
			"Unable to Delete TCP Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}
}

func (r *TCPMonitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// MoveState enables `moved` blocks from the legacy checkly_tcp_check resource
// type. Both resource types share the exact same schema, so the source state
// can be used as the target state without transformation.
func (r *TCPMonitorResource) MoveState(ctx context.Context) []resource.StateMover {
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	sourceSchema := schemaResp.Schema

	return []resource.StateMover{
		{
			SourceSchema: &sourceSchema,
			StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				if !strings.HasSuffix(req.SourceProviderAddress, "/checkly") {
					return
				}

				switch req.SourceTypeName {
				case "checkly_tcp_check", "checkly_tcp_monitor":
				default:
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

				resp.TargetState = *sourceState
			},
		},
	}
}

// tcpMonitorFromModel converts the resource model to its API representation.
func tcpMonitorFromModel(ctx context.Context, model *TCPMonitorResourceModel, diags *diag.Diagnostics) checkly.TCPMonitor {
	// The locations property is serialized even when empty.
	locations := stringsFromSet(ctx, model.Locations, diags)
	if locations == nil {
		locations = []string{}
	}

	privateLocations := stringsFromSet(ctx, model.PrivateLocations, diags)
	if privateLocations == nil {
		privateLocations = []string{}
	}

	monitor := checkly.TCPMonitor{
		ID:                        model.ID.ValueString(),
		Name:                      model.Name.ValueString(),
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
		UseGlobalAlertSettings:    model.UseGlobalAlertSettings.ValueBool(),
		GroupID:                   model.GroupID.ValueInt64(),
		GroupOrder:                int(model.GroupOrder.ValueInt64()),
		PrivateLocations:          &privateLocations,
		Request:                   tcpMonitorRequestFromSet(ctx, model.Request, diags),
		AlertChannelSubscriptions: alertChannelSubscriptionsFromSet(ctx, model.AlertChannelSubscription, diags),
		RetryStrategy:             retryStrategyFromObject(ctx, model.RetryStrategy, diags),
		TriggerIncident:           triggerIncidentFromSet(ctx, model.TriggerIncident, diags),
	}

	// An empty description is not tracked by the Checkly API; it is
	// equivalent to no description at all.
	if description := model.Description.ValueString(); description != "" {
		monitor.Description = &description
	}

	if runtimeID := model.RuntimeID.ValueString(); runtimeID != "" {
		monitor.RuntimeID = &runtimeID
	}

	alertSettings := alertSettingsFromObject(ctx, model.AlertSettings, diags)
	monitor.AlertSettings = &alertSettings

	return monitor
}

// tcpMonitorToModel updates the model to match the API representation of the
// monitor. The model must hold prior values (the plan on create/update, the
// state on read); they are preserved for attributes that the API does not
// track or echo faithfully.
func tcpMonitorToModel(ctx context.Context, c *checkly.TCPMonitor, model *TCPMonitorResourceModel, diags *diag.Diagnostics) {
	model.Name = types.StringValue(c.Name)

	if c.Description != nil {
		model.Description = types.StringValue(*c.Description)
	} else {
		model.Description = types.StringValue("")
	}

	model.Frequency = types.Int64Value(int64(c.Frequency))

	// The API only tracks a frequency offset for high frequency scheduling
	// (frequency 0). Otherwise the prior value (null or a configured 0) is
	// preserved.
	if c.Frequency == 0 {
		model.FrequencyOffset = types.Int64Value(int64(c.FrequencyOffset))
	} else {
		model.FrequencyOffset = int64OrNullWhenUnknown(model.FrequencyOffset)
	}

	model.Activated = types.BoolValue(c.Activated)
	model.Muted = types.BoolValue(c.Muted)
	model.ShouldFail = types.BoolValue(c.ShouldFail)
	model.RunParallel = types.BoolValue(c.RunParallel)

	// The API reports no locations as an empty array. Preserve a prior null
	// value so that an unset attribute stays unset.
	if len(c.Locations) == 0 && (model.Locations.IsNull() || model.Locations.IsUnknown()) {
		model.Locations = types.SetNull(types.StringType)
	} else {
		model.Locations = setFromStrings(ctx, c.Locations, diags)
	}

	model.DegradedResponseTime = types.Int64Value(int64(c.DegradedResponseTime))
	model.MaxResponseTime = types.Int64Value(int64(c.MaxResponseTime))

	// The API reports no tags as an empty array. Preserve a prior null value
	// so that an unset attribute stays unset.
	if len(c.Tags) == 0 && (model.Tags.IsNull() || model.Tags.IsUnknown()) {
		model.Tags = types.SetNull(types.StringType)
	} else {
		model.Tags = setFromStrings(ctx, c.Tags, diags)
	}

	// The API does not track a runtime ID for monitors using the default
	// runtime. A prior empty string is preserved because it converts to the
	// same API representation as null.
	if c.RuntimeID != nil {
		model.RuntimeID = types.StringValue(*c.RuntimeID)
	} else if !model.RuntimeID.IsNull() && model.RuntimeID.ValueString() != "" {
		model.RuntimeID = types.StringNull()
	}

	var privateLocations []string
	if c.PrivateLocations != nil {
		privateLocations = *c.PrivateLocations
	}
	if len(privateLocations) == 0 && (model.PrivateLocations.IsNull() || model.PrivateLocations.IsUnknown()) {
		model.PrivateLocations = types.SetNull(types.StringType)
	} else {
		model.PrivateLocations = setFromStrings(ctx, privateLocations, diags)
	}

	if c.AlertSettings != nil {
		model.AlertSettings = alertSettingsToObject(ctx, *c.AlertSettings, model.AlertSettings, diags)
	} else if model.AlertSettings.IsUnknown() {
		model.AlertSettings = types.ObjectNull(alertSettingsAttrTypes)
	}

	model.UseGlobalAlertSettings = types.BoolValue(c.UseGlobalAlertSettings)
	model.GroupID = types.Int64Value(c.GroupID)
	model.GroupOrder = types.Int64Value(int64(c.GroupOrder))
	model.RetryStrategy = retryStrategyToObject(ctx, c.RetryStrategy, model.RetryStrategy, diags)
	model.Request = tcpMonitorRequestToSet(ctx, c.Request, diags)
	model.AlertChannelSubscription = alertChannelSubscriptionsToSet(ctx, c.AlertChannelSubscriptions, diags)
	model.TriggerIncident = triggerIncidentToSet(ctx, c.TriggerIncident, diags)
}

// tcpMonitorRequestFromSet converts the request block value to its API
// representation.
func tcpMonitorRequestFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) checkly.TCPRequest {
	if set.IsNull() || set.IsUnknown() {
		return checkly.TCPRequest{}
	}

	var models []TCPMonitorRequestModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() || len(models) == 0 {
		return checkly.TCPRequest{}
	}

	model := models[0]

	return checkly.TCPRequest{
		Hostname:   model.Hostname.ValueString(),
		Port:       uint16(model.Port.ValueInt64()),
		Data:       model.Data.ValueString(),
		Assertions: tcpMonitorAssertionsFromSet(ctx, model.Assertion, diags),
		IPFamily:   model.IPFamily.ValueString(),
	}
}

// tcpMonitorRequestToSet converts the API representation of a TCP request to
// the request block value.
func tcpMonitorRequestToSet(ctx context.Context, request checkly.TCPRequest, diags *diag.Diagnostics) types.Set {
	elementType := types.ObjectType{AttrTypes: tcpMonitorRequestAttrTypes}

	obj, d := types.ObjectValue(tcpMonitorRequestAttrTypes, map[string]attr.Value{
		"hostname":  types.StringValue(request.Hostname),
		"port":      types.Int64Value(int64(request.Port)),
		"data":      types.StringValue(request.Data),
		"assertion": tcpMonitorAssertionsToSet(ctx, request.Assertions, diags),
		"ip_family": types.StringValue(request.IPFamily),
	})
	diags.Append(d...)

	set, d := types.SetValue(elementType, []attr.Value{obj})
	diags.Append(d...)

	return set
}

// tcpMonitorAssertionsFromSet converts the request.assertion block value to
// its API representation.
func tcpMonitorAssertionsFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) []checkly.Assertion {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}

	var models []TCPMonitorAssertionModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return nil
	}

	assertions := make([]checkly.Assertion, len(models))
	for i, model := range models {
		assertions[i] = checkly.Assertion{
			Source:     model.Source.ValueString(),
			Property:   model.Property.ValueString(),
			Comparison: model.Comparison.ValueString(),
			Target:     model.Target.ValueString(),
		}
	}

	return assertions
}

// tcpMonitorAssertionsToSet converts the API representation of assertions to
// the request.assertion block value.
func tcpMonitorAssertionsToSet(ctx context.Context, assertions []checkly.Assertion, diags *diag.Diagnostics) types.Set {
	elementType := types.ObjectType{AttrTypes: tcpMonitorAssertionAttrTypes}

	values := make([]attr.Value, len(assertions))
	for i, assertion := range assertions {
		obj, d := types.ObjectValue(tcpMonitorAssertionAttrTypes, map[string]attr.Value{
			"source":     types.StringValue(assertion.Source),
			"property":   types.StringValue(assertion.Property),
			"comparison": types.StringValue(assertion.Comparison),
			"target":     types.StringValue(assertion.Target),
		})
		diags.Append(d...)
		values[i] = obj
	}

	set, d := types.SetValue(elementType, values)
	diags.Append(d...)

	return set
}
