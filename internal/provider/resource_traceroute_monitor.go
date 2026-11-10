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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
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
	ShouldFail                types.Bool   `tfsdk:"should_fail"`
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
				Description: "The name of the monitor.",
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
				Description: "Determines if the monitor is running or not. Possible values `true`, and `false`.",
			},
			"muted": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Determines if any notifications will be sent out when a monitor fails/degrades/recovers.",
			},
			"should_fail": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Allows to invert the behaviour of when a monitor is considered to fail.",
			},
			"run_parallel": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Determines if the monitor should run in all selected locations in parallel or round-robin.",
			},
			"locations": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "An array of one or more data center locations where to run this monitor. (Default [\"us-east-1\"])",
			},
			"degraded_response_time": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(10000),
				Description: "The response time in milliseconds starting from which a monitor should be considered degraded. Possible values are between 0 and 30000. (Default `10000`).",
				Validators: []validator.Int64{
					int64validator.Between(0, 30000),
				},
			},
			"max_response_time": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(20000),
				Description: "The response time in milliseconds starting from which a monitor should be considered failing. Possible values are between 0 and 30000. (Default `20000`).",
				Validators: []validator.Int64{
					int64validator.Between(0, 30000),
				},
			},
			"tags": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "A list of tags for organizing and filtering checks and monitors.",
			},
			"private_locations": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "The slugs of the private locations to run the monitor from. A private location runs traceroute monitors with agent v8.4.0 or later and the `NET_RAW` capability.",
			},
			"runtime_id": schema.StringAttribute{
				Optional:    true,
				Description: "The ID of the runtime to use for this monitor.",
			},
			triggerIncidentAttributeName: makeTriggerIncidentSchema(),
			alertSettingsAttributeName: makeAlertSettingsAttributeSchema(AlertSettingsAttributeSchemaOptions{
				Monitor: true,
			}),
			useGlobalAlertSettingsAttributeName: makeUseGlobalAlertSettingsAttributeSchema("monitor"),
			"group_id": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "The id of the check group this monitor is part of.",
			},
			"group_order": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "The position of this monitor in a check group. It determines in what order checks and monitors are run when a group is triggered from the API or from CI/CD.",
			},
			retryStrategyAttributeName: makeRetryStrategyAttributeSchema(RetryStrategyAttributeSchemaOptions{
				SupportsOnlyOnNetworkError: false,
			}),
			"request": schema.SingleNestedAttribute{
				Required:    true,
				Description: "The parameters for the traceroute probe.",
				Attributes: map[string]schema.Attribute{
					"url": schema.StringAttribute{
						Required:    true,
						Description: "The host to trace the network path to. Do not include a scheme or a port in this value.",
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
						Description: "The destination port for TCP/UDP/SCTP probes. Possible values are between 1 and 65535. Ignored (and not sent) when `protocol = \"ICMP\"`. The default depends on the protocol: `443` for `TCP`, `33434` for `UDP` and `SCTP`.",
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
					"timeout": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Description: "The number of seconds to wait for the traceroute to complete before timing out. Possible values are between 1 and 30. (Default `10`).",
						Validators: []validator.Int64{
							int64validator.Between(1, 30),
						},
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
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
						Description: "A request can have multiple assertions. The allowed comparisons, properties, and target formats depend on the assertion source — see the [Assertion Reference](#assertion-reference) below.",
						NestedObject: schema.NestedAttributeObject{
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
									Description: "The statistic to assert on. Required for `RESPONSE_TIME`, where possible values are `avg`, `min`, `max`, and `stdDev`. Must be empty for `HOP_COUNT` and `PACKET_LOSS`.",
								},
								"comparison": schema.StringAttribute{
									Required:    true,
									Description: "The type of comparison to be executed between expected and actual value of the assertion. For `RESPONSE_TIME`, possible values are `EQUALS`, `NOT_EQUALS`, `GREATER_THAN`, and `LESS_THAN`. For `HOP_COUNT` and `PACKET_LOSS`, possible values are `EQUALS`, `GREATER_THAN`, and `LESS_THAN`.",
								},
								"target": schema.StringAttribute{
									Optional:    true,
									Computed:    true,
									Description: "The value to compare against. Must be numeric: a non-negative number of milliseconds for `RESPONSE_TIME`, a non-negative integer for `HOP_COUNT`, or a number between 0 and 100 for `PACKET_LOSS`.",
								},
							},
						},
					},
				},
			},
			alertChannelSubscriptionAttributeName: makeAlertChannelSubscriptionSchema(AlertChannelSubscriptionAttributeSchemaOptions{
				Monitor: true,
			}),
		},
	}
}

func (r *TracerouteMonitorResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		newFrequencyOffsetConfigValidator(),
	}
}

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
		resp.Diagnostics.AddError(
			"Unable to Create Traceroute Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
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
		resp.Diagnostics.AddError(
			"Unable to Read Traceroute Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
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
		resp.Diagnostics.AddError(
			"Unable to Update Traceroute Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
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
		resp.Diagnostics.AddError(
			"Unable to Delete Traceroute Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
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
		ShouldFail:                model.ShouldFail.ValueBool(),
		RunParallel:               model.RunParallel.ValueBool(),
		Locations:                 locations,
		DegradedResponseTime:      int(model.DegradedResponseTime.ValueInt64()),
		MaxResponseTime:           int(model.MaxResponseTime.ValueInt64()),
		Tags:                      stringsFromSet(ctx, model.Tags, diags),
		PrivateLocations:          &privateLocations,
		UseGlobalAlertSettings:    model.UseGlobalAlertSettings.ValueBool(),
		GroupID:                   model.GroupID.ValueInt64(),
		GroupOrder:                int(model.GroupOrder.ValueInt64()),
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
	model.Locations = stringSetFromAPINullIfUnset(ctx, c.Locations, model.Locations, diags)
	model.DegradedResponseTime = types.Int64Value(int64(c.DegradedResponseTime))
	model.MaxResponseTime = types.Int64Value(int64(c.MaxResponseTime))
	model.Tags = stringSetFromAPINullIfUnset(ctx, c.Tags, model.Tags, diags)

	var privateLocations []string
	if c.PrivateLocations != nil {
		privateLocations = *c.PrivateLocations
	}
	model.PrivateLocations = stringSetFromAPINullIfUnset(ctx, privateLocations, model.PrivateLocations, diags)

	// The API does not track a runtime ID for monitors using the default
	// runtime. A prior empty string is preserved because it converts to the
	// same API representation as null.
	if c.RuntimeID != nil {
		model.RuntimeID = types.StringValue(*c.RuntimeID)
	} else if !model.RuntimeID.IsNull() && model.RuntimeID.ValueString() != "" {
		model.RuntimeID = types.StringNull()
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
	model.Request = tracerouteMonitorRequestToObject(ctx, c.Request, model.Request, diags)
	model.AlertChannelSubscriptions = alertChannelSubscriptionsToSet(ctx, c.AlertChannelSubscriptions, model.AlertChannelSubscriptions, diags)
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
func tracerouteMonitorRequestToObject(ctx context.Context, r checkly.TracerouteRequest, prior types.Object, diags *diag.Diagnostics) types.Object {
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
		Assertions:     assertionsToSet(ctx, r.Assertions, priorAssertions(prior), diags),
	})
	diags.Append(d...)

	return obj
}
