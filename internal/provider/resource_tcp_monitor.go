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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
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
	RuntimeID                 types.String `tfsdk:"runtime_id"`
	PrivateLocations          types.Set    `tfsdk:"private_locations"`
	AlertSettings             types.Object `tfsdk:"alert_settings"`
	UseGlobalAlertSettings    types.Bool   `tfsdk:"use_global_alert_settings"`
	GroupID                   types.Int64  `tfsdk:"group_id"`
	GroupOrder                types.Int64  `tfsdk:"group_order"`
	RetryStrategy             types.Object `tfsdk:"retry_strategy"`
	Request                   types.Object `tfsdk:"request"`
	AlertChannelSubscriptions types.Set    `tfsdk:"alert_channel_subscriptions"`
	TriggerIncident           types.Object `tfsdk:"trigger_incident"`
}

// TCPMonitorRequestModel describes the request attribute data model.
type TCPMonitorRequestModel struct {
	Hostname   types.String `tfsdk:"hostname"`
	Port       types.Int64  `tfsdk:"port"`
	Data       types.String `tfsdk:"data"`
	Assertions types.Set    `tfsdk:"assertions"`
	IPFamily   types.String `tfsdk:"ip_family"`
}

var tcpMonitorRequestAttrTypes = map[string]attr.Type{
	"hostname":   types.StringType,
	"port":       types.Int64Type,
	"data":       types.StringType,
	"assertions": types.SetType{ElemType: types.ObjectType{AttrTypes: assertionAttrTypes}},
	"ip_family":  types.StringType,
}

func (r *TCPMonitorResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tcp_monitor"
}

func (r *TCPMonitorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     sdkv2SchemaVersion + 1,
		Description: "TCP checks allow you to monitor remote endpoints at a lower level.",
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
			triggerIncidentAttributeName: makeTriggerIncidentSchema(),
			alertSettingsAttributeName: makeAlertSettingsAttributeSchema(AlertSettingsAttributeSchemaOptions{
				Monitor: true,
			}),
			retryStrategyAttributeName: makeRetryStrategyAttributeSchema(RetryStrategyAttributeSchemaOptions{
				SupportsOnlyOnNetworkError: false,
			}),
			"request": schema.SingleNestedAttribute{
				Required:    true,
				Description: "The parameters for the TCP connection.",
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
						Default:     stringdefault.StaticString(""),
						Description: "The data to send to the target host.",
					},
					"ip_family": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("IPv4"),
						Description: "The IP family to use when executing the TCP check. The value can be either `IPv4` or `IPv6`. (Default `IPv4`).",
						Validators: []validator.String{
							stringvalidator.OneOf("IPv4", "IPv6"),
						},
					},
					"assertions": schema.SetNestedAttribute{
						Optional:    true,
						Description: "A request can have multiple assertions.",
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
									Description: "The source of the asserted value. Possible values are `RESPONSE_DATA` and `RESPONSE_TIME`.",
								},
								"property": schema.StringAttribute{
									Optional:    true,
									Computed:    true,
									Description: "A regular expression for `RESPONSE_DATA` whose first capture group is asserted instead of the whole response data. Not used for `RESPONSE_TIME`.",
								},
								"comparison": schema.StringAttribute{
									Required:    true,
									Description: "The type of comparison to be executed between expected and actual value of the assertion. Possible values are `EQUALS`, `NOT_EQUALS`, `HAS_KEY`, `NOT_HAS_KEY`, `HAS_VALUE`, `NOT_HAS_VALUE`, `IS_EMPTY`, `NOT_EMPTY`, `GREATER_THAN`, `LESS_THAN`, `CONTAINS`, `NOT_CONTAINS`, `IS_NULL`, and `NOT_NULL`.",
								},
								"target": schema.StringAttribute{
									Optional:    true,
									Computed:    true,
									Description: "The value to compare against, e.g. a number of milliseconds for `RESPONSE_TIME`. Leave empty for comparisons that take no value, such as `IS_EMPTY` and `NOT_NULL`.",
								},
							},
						},
					},
				},
			},
			alertChannelSubscriptionAttributeName: makeAlertChannelSubscriptionSchema(AlertChannelSubscriptionAttributeSchemaOptions{
				Monitor: true,
			}),
		}, CommonCheckAttributeSchemaOptions{Subject: subjectMonitor, ShouldFail: true, RunParallel: true, Locations: true, GroupMembership: true, RuntimeID: true}),
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

// ModifyPlan applies default values to the optional+computed attributes of
// the request assertions. Schema defaults cannot be used for attributes
// nested in set elements (see the schema for details), so the plan is
// adjusted here instead. Substituting static values keeps every set element
// fully known and stable across plans, which set element correlation
// requires.
func (r *TCPMonitorResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planNestedDefaults(ctx, req, resp, path.Root("request"), objectWithSetDefaults("assertions", assertionsWithPropertyAndTargetDefaults))
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
	return renamedResourceStateMovers(ctx, r, "checkly_tcp_check", "checkly_tcp_monitor")
}

// tcpMonitorFromModel converts the resource model to its API representation.
func tcpMonitorFromModel(ctx context.Context, model *TCPMonitorResourceModel, diags *diag.Diagnostics) checkly.TCPMonitor {
	// The locations property is serialized even when empty.
	locations := stringsFromSetOrEmpty(ctx, model.Locations, diags)

	privateLocations := stringsFromSetOrEmpty(ctx, model.PrivateLocations, diags)

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
		Request:                   tcpMonitorRequestFromObject(ctx, model.Request, diags),
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
	model.Locations = stringSetFromAPINullIfUnset(ctx, c.Locations, model.Locations, diags)

	model.DegradedResponseTime = types.Int64Value(int64(c.DegradedResponseTime))
	model.MaxResponseTime = types.Int64Value(int64(c.MaxResponseTime))

	// The API reports no tags as an empty array. Preserve a prior null value
	// so that an unset attribute stays unset.
	model.Tags = stringSetFromAPINullIfUnset(ctx, c.Tags, model.Tags, diags)

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
	model.PrivateLocations = stringSetFromAPINullIfUnset(ctx, privateLocations, model.PrivateLocations, diags)

	if c.AlertSettings != nil {
		model.AlertSettings = alertSettingsToObject(ctx, *c.AlertSettings, model.AlertSettings, diags)
	} else if model.AlertSettings.IsUnknown() {
		model.AlertSettings = types.ObjectNull(alertSettingsAttrTypes)
	}

	model.UseGlobalAlertSettings = types.BoolValue(c.UseGlobalAlertSettings)
	model.GroupID = types.Int64Value(c.GroupID)
	model.GroupOrder = types.Int64Value(int64(c.GroupOrder))
	model.RetryStrategy = retryStrategyToObject(ctx, c.RetryStrategy, model.RetryStrategy, diags)
	model.Request = tcpMonitorRequestToObject(ctx, c.Request, model.Request, diags)
	model.AlertChannelSubscriptions = alertChannelSubscriptionsToSet(ctx, c.AlertChannelSubscriptions, model.AlertChannelSubscriptions, diags)
	model.TriggerIncident = triggerIncidentToObject(ctx, c.TriggerIncident, diags)
}

// tcpMonitorRequestFromObject converts the request attribute value to its
// API representation.
func tcpMonitorRequestFromObject(ctx context.Context, obj types.Object, diags *diag.Diagnostics) checkly.TCPRequest {
	model := objectAs[TCPMonitorRequestModel](ctx, obj, diags)
	if model == nil {
		return checkly.TCPRequest{}
	}

	return checkly.TCPRequest{
		Hostname:   model.Hostname.ValueString(),
		Port:       uint16(model.Port.ValueInt64()),
		Data:       model.Data.ValueString(),
		Assertions: assertionsFromSet(ctx, model.Assertions, diags),
		IPFamily:   checkly.IPFamily(model.IPFamily.ValueString()),
	}
}

// tcpMonitorRequestToObject converts the API representation of a TCP request
// to the request attribute value.
func tcpMonitorRequestToObject(ctx context.Context, request checkly.TCPRequest, prior types.Object, diags *diag.Diagnostics) types.Object {
	obj, d := types.ObjectValue(tcpMonitorRequestAttrTypes, map[string]attr.Value{
		"hostname":   types.StringValue(request.Hostname),
		"port":       types.Int64Value(int64(request.Port)),
		"data":       types.StringValue(request.Data),
		"assertions": assertionsToSet(ctx, request.Assertions, priorAssertions(prior), diags),
		"ip_family":  types.StringValue(string(request.IPFamily)),
	})
	diags.Append(d...)

	return obj
}
