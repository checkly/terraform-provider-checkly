package provider

import (
	"context"

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

var (
	_ resource.Resource                     = (*TracerouteMonitorResource)(nil)
	_ resource.ResourceWithConfigure        = (*TracerouteMonitorResource)(nil)
	_ resource.ResourceWithUpgradeState     = (*TracerouteMonitorResource)(nil)
	_ resource.ResourceWithConfigValidators = (*TracerouteMonitorResource)(nil)
	_ resource.ResourceWithValidateConfig   = (*TracerouteMonitorResource)(nil)
	_ resource.ResourceWithImportState      = (*TracerouteMonitorResource)(nil)
	_ resource.ResourceWithModifyPlan       = (*TracerouteMonitorResource)(nil)
)

// NewTracerouteMonitorResource returns a new checkly_traceroute_monitor
// resource.
func NewTracerouteMonitorResource() resource.Resource {
	return &TracerouteMonitorResource{}
}

// TracerouteMonitorResource implements the checkly_traceroute_monitor
// resource.
type TracerouteMonitorResource struct {
	client checkly.Client
}

// TracerouteMonitorResourceModel describes the resource data model.
type TracerouteMonitorResourceModel struct {
	ID                        types.String `tfsdk:"id"`
	Name                      types.String `tfsdk:"name"`
	Description               types.String `tfsdk:"description"`
	Frequency                 types.Int64  `tfsdk:"frequency"`
	FrequencyOffset           types.Int64  `tfsdk:"frequency_offset"`
	Activated                 types.Bool   `tfsdk:"activated"`
	Muted                     types.Bool   `tfsdk:"muted"`
	RunParallel               types.Bool   `tfsdk:"run_parallel"`
	Locations                 types.Set    `tfsdk:"locations"`
	DegradedResponseTime      types.Int64  `tfsdk:"degraded_response_time"`
	MaxResponseTime           types.Int64  `tfsdk:"max_response_time"`
	Tags                      types.Set    `tfsdk:"tags"`
	PrivateLocations          types.Set    `tfsdk:"private_locations"`
	RuntimeID                 types.String `tfsdk:"runtime_id"`
	AlertSettings             types.Object `tfsdk:"alert_settings"`
	UseGlobalAlertSettings    types.Bool   `tfsdk:"use_global_alert_settings"`
	GroupID                   types.Int64  `tfsdk:"group_id"`
	GroupOrder                types.Int64  `tfsdk:"group_order"`
	RetryStrategy             types.Object `tfsdk:"retry_strategy"`
	Request                   types.Object `tfsdk:"request"`
	AlertChannelSubscriptions types.Set    `tfsdk:"alert_channel_subscriptions"`
	TriggerIncident           types.Object `tfsdk:"trigger_incident"`
}

// TracerouteMonitorRequestModel describes the request attribute data model.
type TracerouteMonitorRequestModel struct {
	URL            types.String `tfsdk:"url"`
	Protocol       types.String `tfsdk:"protocol"`
	Port           types.Int64  `tfsdk:"port"`
	IPFamily       types.String `tfsdk:"ip_family"`
	MaxHops        types.Int64  `tfsdk:"max_hops"`
	MaxUnknownHops types.Int64  `tfsdk:"max_unknown_hops"`
	PtrLookup      types.Bool   `tfsdk:"ptr_lookup"`
	Timeout        types.Int64  `tfsdk:"timeout"`
	Assertions     types.Set    `tfsdk:"assertions"`
}

var tracerouteMonitorRequestAttrTypes = map[string]attr.Type{
	"url":              types.StringType,
	"protocol":         types.StringType,
	"port":             types.Int64Type,
	"ip_family":        types.StringType,
	"max_hops":         types.Int64Type,
	"max_unknown_hops": types.Int64Type,
	"ptr_lookup":       types.BoolType,
	"timeout":          types.Int64Type,
	"assertions":       types.SetType{ElemType: types.ObjectType{AttrTypes: assertionAttrTypes}},
}

func (r *TracerouteMonitorResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_traceroute_monitor"
}

func (r *TracerouteMonitorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     sdkv2SchemaVersion + 1,
		Description: "Traceroute monitors trace the network path to a host and measure per-hop latency and packet loss.",
		Attributes: withCommonCheckAttributes(map[string]schema.Attribute{
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
				Subject:            subjectMonitor,
				AllowHighFrequency: true,
			}),
			frequencyOffsetAttributeName: makeFrequencyOffsetAttributeSchema(FrequencyOffsetAttributeSchemaOptions{
				Subject: subjectMonitor,
			}),
			"degraded_response_time": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(10000),
				Description: "The response time in milliseconds where the monitor should be considered degraded. Possible values are between `1` and `30000`. (Default `10000`).",
				Validators: []validator.Int64{
					int64validator.Between(1, 30000),
				},
			},
			"max_response_time": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(20000),
				Description: "The response time in milliseconds where the monitor should be considered failing. Possible values are between `0` and `30000`. `0` means the monitor never fails on response time; any other value must be greater than or equal to `degraded_response_time`. (Default `20000`).",
				Validators: []validator.Int64{
					int64validator.Between(0, 30000),
				},
			},
			triggerIncidentAttributeName: makeTriggerIncidentSchema(),
			alertSettingsAttributeName: makeAlertSettingsAttributeSchema(AlertSettingsAttributeSchemaOptions{
				Subject: subjectMonitor,
			}),
			retryStrategyAttributeName: makeRetryStrategyAttributeSchema(RetryStrategyAttributeSchemaOptions{
				SupportsOnlyOnNetworkError: false,
			}),
			"request": schema.SingleNestedAttribute{
				Required:    true,
				Description: "The parameters for the traceroute probe.",
				Attributes: map[string]schema.Attribute{
					"url": schema.StringAttribute{
						Required:    true,
						Description: "The host to trace the network path to (1 to 2048 characters). Do not include a scheme or a port in this value.",
						Validators: []validator.String{
							utf16LengthBetween(1, 2048),
							noControlCharacters(),
							hostWithoutSchemeOrPortValidator{},
						},
					},
					"protocol": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("TCP"),
						Description: "The probe protocol. `TCP` sends SYN probes (default), `UDP` sends datagrams to a high port, `ICMP` sends Echo Requests, `SCTP` sends INIT chunks. (Default `TCP`).",
						Validators: []validator.String{
							stringvalidator.OneOf("TCP", "ICMP", "UDP", "SCTP"),
						},
					},
					// port and max_unknown_hops are derived by the server
					// from other attributes when unset, so they are left
					// unknown in the plan rather than keeping their prior
					// state value; see tracerouteMonitorRequestFromModel.
					"port": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Description: "The destination port for TCP/UDP/SCTP probes. Possible values are between 1 and 65535. Must not be set when `protocol = \"ICMP\"`, which has no port. The default depends on the protocol: `443` for `TCP`, `33434` for `UDP` and `SCTP`.",
						Validators: []validator.Int64{
							int64validator.Between(1, 65535),
						},
					},
					"ip_family": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("IPv4"),
						Description: "The IP family to use when executing the traceroute. The value can be either `IPv4` or `IPv6`. (Default `IPv4`).",
						Validators: []validator.String{
							stringvalidator.OneOf("IPv4", "IPv6"),
						},
					},
					"max_hops": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Default:     int64default.StaticInt64(30),
						Description: "The maximum number of network hops to probe before stopping. Possible values are between 1 and 64. (Default `30`).",
						Validators: []validator.Int64{
							int64validator.Between(1, 64),
						},
					},
					"max_unknown_hops": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Description: "The maximum number of consecutive unresponsive hops to tolerate before stopping the trace. Possible values are between 1 and 30, and the value must not exceed `max_hops`. (Default `min(15, max_hops)`).",
						Validators: []validator.Int64{
							int64validator.Between(1, 30),
						},
					},
					"ptr_lookup": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(true),
						Description: "Whether to perform reverse-DNS (PTR) lookups on each hop's IP address. (Default `true`).",
					},
					// timeout has the API's default rather than keeping
					// its prior state value: the update endpoint keeps the
					// stored value of an omitted field, so removing
					// timeout from the configuration must send the
					// default explicitly.
					"timeout": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Default:     int64default.StaticInt64(10),
						Description: "The number of seconds to wait for the traceroute to complete before timing out. Possible values are between 1 and 30. (Default `10`).",
						Validators: []validator.Int64{
							int64validator.Between(1, 30),
						},
					},
					// The full per-source assertion rules (allowed
					// comparisons, property usage, target formats) are
					// documented in the "Assertion Reference" section of
					// templates/resources/traceroute_monitor.md.tmpl.
					// When assertion sources or their rules change here
					// or in the API, update that template too — it is
					// not generated from this schema.
					"assertions": schema.SetNestedAttribute{
						Optional:    true,
						Computed:    true,
						Default:     emptySetDefault(types.ObjectType{AttrTypes: assertionAttrTypes}),
						Description: "A request can have multiple assertions. The allowed comparisons, properties, and target formats depend on the assertion source — see the [Assertion Reference](#assertion-reference) below.",
						NestedObject: schema.NestedAttributeObject{
							Validators: []validator.Object{
								tracerouteMonitorAssertionRules,
							},
							// Attributes nested in this set attribute must
							// not use schema defaults. The framework looks
							// up configuration values for set elements by
							// full element value; once computed attributes
							// resolve, the lookup misses, the configuration
							// is treated as null, and defaults would
							// override values the practitioner actually
							// configured, producing a perpetual diff.
							// Default values are instead applied in
							// ModifyPlan.
							Attributes: map[string]schema.Attribute{
								"source": schema.StringAttribute{
									Required:    true,
									Description: "The source of the asserted value. Possible values are `RESPONSE_TIME`, `HOP_COUNT`, and `PACKET_LOSS`.",
									Validators: []validator.String{
										stringvalidator.OneOf("RESPONSE_TIME", "HOP_COUNT", "PACKET_LOSS"),
									},
								},
								"property": schema.StringAttribute{
									Optional:    true,
									Computed:    true,
									Description: "The statistic to assert on. Required for `RESPONSE_TIME`, where possible values are `avg`, `min`, `max`, and `stdDev`. Not used for `HOP_COUNT` and `PACKET_LOSS`.",
								},
								"comparison": schema.StringAttribute{
									Required:    true,
									Description: "The type of comparison to be executed between expected and actual value of the assertion. For `RESPONSE_TIME`, possible values are `EQUALS`, `NOT_EQUALS`, `GREATER_THAN`, and `LESS_THAN`. For `HOP_COUNT` and `PACKET_LOSS`, possible values are `EQUALS`, `GREATER_THAN`, and `LESS_THAN`.",
								},
								"target": schema.StringAttribute{
									Optional:    true,
									Computed:    true,
									Description: "The value to compare against. Required, and must be numeric without surrounding whitespace: a non-negative number of milliseconds for `RESPONSE_TIME`, a non-negative integer for `HOP_COUNT`, or a number between 0 and 100 for `PACKET_LOSS`.",
								},
							},
						},
					},
				},
			},
			alertChannelSubscriptionAttributeName: makeAlertChannelSubscriptionSchema(AlertChannelSubscriptionAttributeSchemaOptions{
				Subject: subjectMonitor,
			}),
		}, CommonCheckAttributeSchemaOptions{Subject: subjectMonitor, RunParallel: true, Locations: true, GroupMembership: true, RuntimeID: true, PrivateLocationsNote: "A private location runs traceroute monitors with agent v8.4.0 or later and the `NET_RAW` capability."}),
	}
}

func (r *TracerouteMonitorResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		newFrequencyOffsetConfigValidator(),
		// A max_response_time of 0 is no failing limit, so any
		// degraded_response_time is within it.
		orderedThresholdsConfigValidator{
			lower:              path.Root("degraded_response_time"),
			lowerDefault:       types.Int64Value(10000),
			upper:              path.Root("max_response_time"),
			upperDefault:       types.Int64Value(20000),
			zeroUpperUnbounded: true,
		},
		// An unset max_unknown_hops is derived from max_hops, within it.
		orderedThresholdsConfigValidator{
			lower:        path.Root("request").AtName("max_unknown_hops"),
			lowerDefault: types.Int64Null(),
			upper:        path.Root("request").AtName("max_hops"),
			upperDefault: types.Int64Value(30),
		},
	}
}

// ValidateConfig rejects a port on ICMP probes. The API drops the port of an
// ICMP request, so a configured one would read back as unset.
func (r *TracerouteMonitorResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var protocol types.String
	var port types.Int64
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("request").AtName("protocol"), &protocol)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("request").AtName("port"), &port)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if protocol.ValueString() == "ICMP" && !port.IsNull() && !port.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("request").AtName("port"),
			"Invalid Attribute Combination",
			`request.port cannot be set when request.protocol is "ICMP": ICMP probes have no port. Remove request.port.`,
		)
	}
}

// tracerouteMonitorAssertionRules are the API's rules for the comparisons,
// properties and targets of each assertion source.
var tracerouteMonitorAssertionRules = assertionRulesValidator{rules: map[string]assertionSourceRule{
	"RESPONSE_TIME": {
		comparisons:   []string{"EQUALS", "NOT_EQUALS", "GREATER_THAN", "LESS_THAN"},
		properties:    []string{"avg", "min", "max", "stdDev"},
		numericTarget: nonNegativeMillisecondsTarget,
	},
	"HOP_COUNT": {
		comparisons:   []string{"EQUALS", "GREATER_THAN", "LESS_THAN"},
		numericTarget: &numericAssertionTarget{pattern: nonNegativeIntegerRegex, label: "a non-negative integer"},
	},
	"PACKET_LOSS": {
		comparisons:   []string{"EQUALS", "GREATER_THAN", "LESS_THAN"},
		numericTarget: &numericAssertionTarget{pattern: nonNegativeNumberRegex, bounded: true, min: 0, max: 100, label: "a number between 0 and 100"},
	},
}}

// UpgradeState upgrades state written by the provider's SDK v2
// implementation; see sdkv2StateUpgraders.
func (r *TracerouteMonitorResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return sdkv2StateUpgraders(ctx, r)
}

func (r *TracerouteMonitorResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

// ModifyPlan applies default values to the optional+computed attributes of
// the request assertions. Schema defaults cannot be used for attributes
// nested in set elements (see the schema for details), so the plan is
// adjusted here instead. Substituting static values keeps every set element
// fully known and stable across plans, which set element correlation
// requires.
func (r *TracerouteMonitorResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planNestedDefaults(ctx, req, resp, path.Root("request"), objectWithSetDefaults("assertions", assertionsWithPropertyAndTargetDefaults))
}

func (r *TracerouteMonitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TracerouteMonitorResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	monitor := tracerouteMonitorFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.CreateTracerouteMonitor(apiCtx, monitor)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to Create Traceroute Monitor", err)
		return
	}

	// State is written even when the conversion produces error diagnostics.
	// The monitor already exists remotely at this point, and returning
	// without saving state would leave it unmanaged.
	plan.ID = types.StringValue(result.ID)
	tracerouteMonitorToModel(ctx, result, &plan, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TracerouteMonitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TracerouteMonitorResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	monitor, err := r.client.GetTracerouteMonitor(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		addAPIError(&resp.Diagnostics, "Unable to Read Traceroute Monitor", err)
		return
	}

	tracerouteMonitorToModel(ctx, monitor, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TracerouteMonitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TracerouteMonitorResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	monitor := tracerouteMonitorFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.UpdateTracerouteMonitor(apiCtx, plan.ID.ValueString(), monitor)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to Update Traceroute Monitor", err)
		return
	}

	tracerouteMonitorToModel(ctx, result, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TracerouteMonitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TracerouteMonitorResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	err := r.client.DeleteTracerouteMonitor(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			return
		}
		addAPIError(&resp.Diagnostics, "Unable to Delete Traceroute Monitor", err)
		return
	}
}

func (r *TracerouteMonitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// tracerouteMonitorFromModel converts the resource model to its API
// representation.
func tracerouteMonitorFromModel(ctx context.Context, model *TracerouteMonitorResourceModel, diags *diag.Diagnostics) checkly.TracerouteMonitor {
	// The locations property is serialized even when empty.
	locations := stringsFromSetOrEmpty(ctx, model.Locations, diags)

	privateLocations := stringsFromSetOrEmpty(ctx, model.PrivateLocations, diags)

	monitor := checkly.TracerouteMonitor{
		ID:                        model.ID.ValueString(),
		Name:                      model.Name.ValueString(),
		Frequency:                 int(model.Frequency.ValueInt64()),
		FrequencyOffset:           int(model.FrequencyOffset.ValueInt64()),
		Activated:                 model.Activated.ValueBool(),
		Muted:                     model.Muted.ValueBool(),
		RunParallel:               model.RunParallel.ValueBool(),
		Locations:                 locations,
		DegradedResponseTime:      int(model.DegradedResponseTime.ValueInt64()),
		Tags:                      stringsFromSet(ctx, model.Tags, diags),
		PrivateLocations:          &privateLocations,
		UseGlobalAlertSettings:    model.UseGlobalAlertSettings.ValueBool(),
		GroupID:                   model.GroupID.ValueInt64(),
		GroupOrder:                groupOrderToAPI(model.GroupOrder),
		Request:                   tracerouteMonitorRequestFromObject(ctx, model.Request, diags),
		AlertChannelSubscriptions: alertChannelSubscriptionsFromSet(ctx, model.AlertChannelSubscriptions, diags),
		RetryStrategy:             retryStrategyFromObject(ctx, model.RetryStrategy, diags),
		TriggerIncident:           triggerIncidentFromObject(ctx, model.TriggerIncident, diags),
	}

	// An empty description is not tracked by the Checkly API; it is
	// equivalent to no description at all.
	if description := model.Description.ValueString(); description != "" {
		monitor.Description = &description
	}

	if runtimeID := model.RuntimeID.ValueString(); runtimeID != "" {
		monitor.RuntimeID = &runtimeID
	}

	// A max_response_time of 0 means no failing response time limit, which
	// the SDK stores with a separate flag, as its zero MaxResponseTime
	// means the API's default.
	if maxResponseTime := model.MaxResponseTime.ValueInt64(); maxResponseTime == 0 {
		monitor.NoMaxResponseTime = true
	} else {
		monitor.MaxResponseTime = int(maxResponseTime)
	}

	alertSettings := alertSettingsFromObject(ctx, model.AlertSettings, diags)
	monitor.AlertSettings = &alertSettings

	return monitor
}

// tracerouteMonitorToModel updates the model to match the API representation
// of the monitor. The model must hold prior values (the plan on
// create/update, the state on read); they are preserved for attributes that
// the API does not track or echo faithfully.
func tracerouteMonitorToModel(ctx context.Context, c *checkly.TracerouteMonitor, model *TracerouteMonitorResourceModel, diags *diag.Diagnostics) {
	model.Name = types.StringValue(c.Name)

	if c.Description != nil {
		model.Description = types.StringValue(*c.Description)
	} else {
		model.Description = types.StringValue("")
	}

	model.Frequency = types.Int64Value(int64(c.Frequency))

	model.FrequencyOffset = frequencyOffsetFromAPI(c.Frequency, c.FrequencyOffset, model.FrequencyOffset)

	model.Activated = types.BoolValue(c.Activated)
	model.Muted = types.BoolValue(c.Muted)
	model.RunParallel = types.BoolValue(c.RunParallel)
	model.Locations = setFromStrings(ctx, c.Locations, diags)
	model.DegradedResponseTime = types.Int64Value(int64(c.DegradedResponseTime))
	// A monitor without a failing response time limit reads back as 0, the
	// value that configures it.
	if c.NoMaxResponseTime {
		model.MaxResponseTime = types.Int64Value(0)
	} else {
		model.MaxResponseTime = types.Int64Value(int64(c.MaxResponseTime))
	}
	model.Tags = setFromStrings(ctx, c.Tags, diags)

	var privateLocations []string
	if c.PrivateLocations != nil {
		privateLocations = *c.PrivateLocations
	}
	model.PrivateLocations = setFromStrings(ctx, privateLocations, diags)

	// The API does not track a runtime ID for monitors using the default
	// runtime. A prior empty string is preserved because it converts to the
	// same API representation as null.
	if c.RuntimeID != nil {
		model.RuntimeID = types.StringValue(*c.RuntimeID)
	} else if !model.RuntimeID.IsNull() && model.RuntimeID.ValueString() != "" {
		model.RuntimeID = types.StringNull()
	}

	model.AlertSettings = alertSettingsFromAPI(ctx, c.AlertSettings, model.AlertSettings, diags)

	model.UseGlobalAlertSettings = types.BoolValue(c.UseGlobalAlertSettings)
	model.GroupID = int64ValueOrNull(c.GroupID)
	model.GroupOrder = groupOrderFromAPI(c.GroupOrder, model.GroupOrder)
	model.RetryStrategy = retryStrategyToObject(ctx, c.RetryStrategy, model.RetryStrategy, diags)
	model.Request = tracerouteMonitorRequestToObject(ctx, c.Request, diags)
	model.AlertChannelSubscriptions = alertChannelSubscriptionsToSet(ctx, c.AlertChannelSubscriptions, diags)
	model.TriggerIncident = triggerIncidentToObject(ctx, c.TriggerIncident, diags)
}

// tracerouteMonitorRequestFromObject converts the request attribute value to
// its API representation.
func tracerouteMonitorRequestFromObject(ctx context.Context, obj types.Object, diags *diag.Diagnostics) checkly.TracerouteRequest {
	model := objectAs[TracerouteMonitorRequestModel](ctx, obj, diags)
	if model == nil {
		return checkly.TracerouteRequest{}
	}

	return tracerouteMonitorRequestFromModel(ctx, *model, diags)
}

// tracerouteMonitorRequestFromModel converts the request attribute model to
// its API representation.
//
// port and max_unknown_hops are server-derived when the configuration does
// not set them: the API derives port from protocol (443 for TCP, 33434 for
// UDP/SCTP, no port for ICMP) and max_unknown_hops from max_hops
// (min(15, max_hops)). The update endpoint keeps the stored value of an
// omitted field, so omitting them after a protocol or max_hops change would
// keep a port that does not match the new protocol, or violate the API's
// `max_unknown_hops <= max_hops` rule. An unset attribute is unknown in the
// planned values being applied, so in that case the server's derivation rules
// are mirrored here, and every write carries the value the server would have
// chosen for the current configuration.
func tracerouteMonitorRequestFromModel(ctx context.Context, model TracerouteMonitorRequestModel, diags *diag.Diagnostics) checkly.TracerouteRequest {
	// PtrLookup is sent as an explicit pointer so that a configured `false`
	// reaches the wire. The SDK drops `port` from the payload when the
	// protocol is ICMP.
	ptrLookup := model.PtrLookup.ValueBool()

	request := checkly.TracerouteRequest{
		URL:            model.URL.ValueString(),
		Protocol:       checkly.TracerouteProtocol(model.Protocol.ValueString()),
		Port:           int(model.Port.ValueInt64()),
		IPFamily:       checkly.IPFamily(model.IPFamily.ValueString()),
		MaxHops:        int(model.MaxHops.ValueInt64()),
		MaxUnknownHops: int(model.MaxUnknownHops.ValueInt64()),
		PtrLookup:      &ptrLookup,
		Timeout:        int(model.Timeout.ValueInt64()),
		Assertions:     assertionsFromSet(ctx, model.Assertions, diags),
	}

	if model.Port.IsNull() || model.Port.IsUnknown() {
		switch request.Protocol {
		case "UDP", "SCTP":
			request.Port = 33434
		case "ICMP":
			// ICMP probes have no port; the zero value is dropped from the
			// marshaled payload.
			request.Port = 0
		default:
			request.Port = 443
		}
	}

	if model.MaxUnknownHops.IsNull() || model.MaxUnknownHops.IsUnknown() {
		request.MaxUnknownHops = min(15, request.MaxHops)
	}

	return request
}

// tracerouteMonitorRequestToObject converts the API representation of a
// traceroute request to the request attribute value.
func tracerouteMonitorRequestToObject(ctx context.Context, r checkly.TracerouteRequest, diags *diag.Diagnostics) types.Object {
	// The API defaults ptrLookup to true, so a missing value reads as true.
	ptrLookup := true
	if r.PtrLookup != nil {
		ptrLookup = *r.PtrLookup
	}

	obj, d := types.ObjectValueFrom(ctx, tracerouteMonitorRequestAttrTypes, TracerouteMonitorRequestModel{
		URL:            types.StringValue(r.URL),
		Protocol:       types.StringValue(string(r.Protocol)),
		Port:           types.Int64Value(int64(r.Port)),
		IPFamily:       types.StringValue(string(r.IPFamily)),
		MaxHops:        types.Int64Value(int64(r.MaxHops)),
		MaxUnknownHops: types.Int64Value(int64(r.MaxUnknownHops)),
		PtrLookup:      types.BoolValue(ptrLookup),
		Timeout:        types.Int64Value(int64(r.Timeout)),
		Assertions:     assertionsToSet(ctx, r.Assertions, diags),
	})
	diags.Append(d...)

	return obj
}
