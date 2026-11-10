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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

var (
	_ resource.Resource                     = (*ICMPMonitorResource)(nil)
	_ resource.ResourceWithConfigure        = (*ICMPMonitorResource)(nil)
	_ resource.ResourceWithUpgradeState     = (*ICMPMonitorResource)(nil)
	_ resource.ResourceWithConfigValidators = (*ICMPMonitorResource)(nil)
	_ resource.ResourceWithImportState      = (*ICMPMonitorResource)(nil)
	_ resource.ResourceWithModifyPlan       = (*ICMPMonitorResource)(nil)
)

// NewICMPMonitorResource returns a new checkly_icmp_monitor resource.
func NewICMPMonitorResource() resource.Resource {
	return &ICMPMonitorResource{}
}

// ICMPMonitorResource implements the checkly_icmp_monitor resource.
type ICMPMonitorResource struct {
	client checkly.Client
}

// ICMPMonitorResourceModel describes the resource data model.
type ICMPMonitorResourceModel struct {
	ID                          types.String `tfsdk:"id"`
	Name                        types.String `tfsdk:"name"`
	Description                 types.String `tfsdk:"description"`
	Frequency                   types.Int64  `tfsdk:"frequency"`
	FrequencyOffset             types.Int64  `tfsdk:"frequency_offset"`
	Activated                   types.Bool   `tfsdk:"activated"`
	Muted                       types.Bool   `tfsdk:"muted"`
	RunParallel                 types.Bool   `tfsdk:"run_parallel"`
	Locations                   types.Set    `tfsdk:"locations"`
	DegradedPacketLossThreshold types.Int64  `tfsdk:"degraded_packet_loss_threshold"`
	MaxPacketLossThreshold      types.Int64  `tfsdk:"max_packet_loss_threshold"`
	Tags                        types.Set    `tfsdk:"tags"`
	PrivateLocations            types.Set    `tfsdk:"private_locations"`
	AlertChannelSubscriptions   types.Set    `tfsdk:"alert_channel_subscriptions"`
	AlertSettings               types.Object `tfsdk:"alert_settings"`
	UseGlobalAlertSettings      types.Bool   `tfsdk:"use_global_alert_settings"`
	Request                     types.Object `tfsdk:"request"`
	GroupID                     types.Int64  `tfsdk:"group_id"`
	GroupOrder                  types.Int64  `tfsdk:"group_order"`
	RetryStrategy               types.Object `tfsdk:"retry_strategy"`
	TriggerIncident             types.Object `tfsdk:"trigger_incident"`
}

// ICMPMonitorRequestModel describes the request attribute data model.
type ICMPMonitorRequestModel struct {
	Hostname   types.String `tfsdk:"hostname"`
	IPFamily   types.String `tfsdk:"ip_family"`
	PingCount  types.Int64  `tfsdk:"ping_count"`
	Assertions types.Set    `tfsdk:"assertions"`
}

var icmpMonitorRequestAttrTypes = map[string]attr.Type{
	"hostname":   types.StringType,
	"ip_family":  types.StringType,
	"ping_count": types.Int64Type,
	"assertions": types.SetType{ElemType: types.ObjectType{AttrTypes: assertionAttrTypes}},
}

func (r *ICMPMonitorResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_icmp_monitor"
}

func (r *ICMPMonitorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     sdkv2SchemaVersion + 1,
		Description: "Creates an ICMP Monitor to check host availability using ping.",
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
				Description: "Determines whether the monitor will run periodically or not after being deployed.",
			},
			"muted": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Determines if any notifications will be sent out when the monitor fails and/or recovers. (Default `false`).",
			},
			"run_parallel": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Determines whether the monitor should run on all selected locations in parallel or round-robin. (Default `false`).",
			},
			"locations": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "An array of one or more data center locations where to run this monitor.",
			},
			"degraded_packet_loss_threshold": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(10),
				Description: "The packet loss percentage where the monitor should be considered degraded. Possible values are between `1` and `100`. (Default `10`).",
				// The API rejects an explicit 0 even though its public schema
				// documents it as valid.
				Validators: []validator.Int64{
					int64validator.Between(1, 100),
				},
			},
			"max_packet_loss_threshold": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(20),
				Description: "The packet loss percentage where the monitor should be considered failing. Possible values are between `1` and `100`. (Default `20`).",
				// The API rejects an explicit 0 even though its public schema
				// documents it as valid.
				Validators: []validator.Int64{
					int64validator.Between(1, 100),
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
				Description: "The slugs of the private locations to run the monitor from. A private location runs ICMP monitors with agent v7.0.0 or later and the `NET_RAW` capability.",
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
				Description: "The ID of the check group that this monitor is part of.",
			},
			"group_order": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "The position of the monitor in the check group. It determines in what order checks and monitors are run when a group is triggered from the API or from CI/CD.",
			},
			retryStrategyAttributeName: makeRetryStrategyAttributeSchema(RetryStrategyAttributeSchemaOptions{}),
			"request": schema.SingleNestedAttribute{
				Required:    true,
				Description: "The parameters of the ICMP request.",
				Attributes: map[string]schema.Attribute{
					"hostname": schema.StringAttribute{
						Required:    true,
						Description: "The hostname to ping.",
					},
					"ip_family": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("IPv4"),
						Description: "The IP family to use. Possible values are `IPv4` and `IPv6`. (Default `IPv4`).",
						Validators: []validator.String{
							stringvalidator.OneOf("IPv4", "IPv6"),
						},
					},
					"ping_count": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Default:     int64default.StaticInt64(10),
						Description: "The number of ping packets to send. (Default `10`).",
						Validators: []validator.Int64{
							int64validator.Between(1, 50),
						},
					},
					"assertions": schema.SetNestedAttribute{
						Optional:    true,
						Description: "Assertions to validate the ICMP response.",
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
									Description: "The source of the asserted value. Possible values are `LATENCY` and `JSON_RESPONSE`.",
									Validators: []validator.String{
										stringvalidator.OneOf("LATENCY", "JSON_RESPONSE"),
									},
								},
								"property": schema.StringAttribute{
									Optional:    true,
									Computed:    true,
									Description: "The property of the source to assert. For `LATENCY` source, possible values are `avg`, `min`, `max` and `stdDev`.",
								},
								"comparison": schema.StringAttribute{
									Required:    true,
									Description: "The type of comparison to be executed between expected and actual value of the assertion. Possible values are `EQUALS`, `NOT_EQUALS`, `GREATER_THAN` and `LESS_THAN`.",
									Validators: []validator.String{
										stringvalidator.OneOf("EQUALS", "NOT_EQUALS", "GREATER_THAN", "LESS_THAN"),
									},
								},
								"target": schema.StringAttribute{
									Required:    true,
									Description: "The target value for the assertion.",
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

func (r *ICMPMonitorResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		newFrequencyOffsetConfigValidator(),
	}
}

// UpgradeState upgrades state written by the provider's SDK v2
// implementation; see sdkv2StateUpgraders.
func (r *ICMPMonitorResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return sdkv2StateUpgraders(ctx, r)
}

func (r *ICMPMonitorResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

// ModifyPlan applies default values to the optional+computed attributes of
// the request assertions. Schema defaults cannot be used for attributes
// nested in set elements (see the schema for details), so the plan is
// adjusted here instead. Substituting static values keeps every set element
// fully known and stable across plans, which set element correlation
// requires.
func (r *ICMPMonitorResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planNestedDefaults(ctx, req, resp, path.Root("request"), objectWithSetDefaults("assertions", assertionsWithPropertyDefault))
}

func (r *ICMPMonitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ICMPMonitorResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	monitor := icmpMonitorFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.CreateICMPMonitor(apiCtx, monitor)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create ICMP Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	// State is written even when the conversion produces error diagnostics.
	// The monitor already exists remotely at this point, and returning
	// without saving state would leave it unmanaged.
	icmpMonitorToModel(ctx, result, &plan, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ICMPMonitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ICMPMonitorResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	monitor, err := r.client.GetICMPMonitor(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Read ICMP Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	icmpMonitorToModel(ctx, monitor, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ICMPMonitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ICMPMonitorResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	monitor := icmpMonitorFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.UpdateICMPMonitor(apiCtx, monitor.ID, monitor)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update ICMP Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	icmpMonitorToModel(ctx, result, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ICMPMonitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ICMPMonitorResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	err := r.client.DeleteICMPMonitor(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Delete ICMP Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}
}

func (r *ICMPMonitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// icmpMonitorFromModel converts the resource data model to its API
// representation.
func icmpMonitorFromModel(ctx context.Context, model *ICMPMonitorResourceModel, diags *diag.Diagnostics) checkly.ICMPMonitor {
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

	return checkly.ICMPMonitor{
		ID:                          model.ID.ValueString(),
		Name:                        model.Name.ValueString(),
		Description:                 description,
		Frequency:                   int(model.Frequency.ValueInt64()),
		FrequencyOffset:             int(model.FrequencyOffset.ValueInt64()),
		Activated:                   model.Activated.ValueBool(),
		Muted:                       model.Muted.ValueBool(),
		RunParallel:                 model.RunParallel.ValueBool(),
		Locations:                   locations,
		DegradedPacketLossThreshold: int(model.DegradedPacketLossThreshold.ValueInt64()),
		MaxPacketLossThreshold:      int(model.MaxPacketLossThreshold.ValueInt64()),
		Tags:                        stringsFromSet(ctx, model.Tags, diags),
		PrivateLocations:            &privateLocations,
		AlertSettings:               &alertSettings,
		UseGlobalAlertSettings:      model.UseGlobalAlertSettings.ValueBool(),
		Request:                     icmpMonitorRequestFromObject(ctx, model.Request, diags),
		GroupID:                     model.GroupID.ValueInt64(),
		GroupOrder:                  int(model.GroupOrder.ValueInt64()),
		AlertChannelSubscriptions:   alertChannelSubscriptionsFromSet(ctx, model.AlertChannelSubscriptions, diags),
		RetryStrategy:               retryStrategyFromObject(ctx, model.RetryStrategy, diags),
		TriggerIncident:             triggerIncidentFromObject(ctx, model.TriggerIncident, diags),
	}
}

// icmpMonitorToModel updates the model to match the API representation of
// the monitor. The values already present in the model (from the plan on
// create/update, from the state on read) are used to resolve fields that the
// API does not track faithfully.
func icmpMonitorToModel(ctx context.Context, c *checkly.ICMPMonitor, model *ICMPMonitorResourceModel, diags *diag.Diagnostics) {
	model.ID = types.StringValue(c.ID)
	model.Name = types.StringValue(c.Name)

	// The attribute defaults to "", so a missing description maps to "".
	if c.Description != nil {
		model.Description = types.StringValue(*c.Description)
	} else {
		model.Description = types.StringValue("")
	}

	model.Frequency = types.Int64Value(int64(c.Frequency))

	// The API only tracks an offset for high frequency scheduling
	// (frequency 0). For regular frequencies a configured value of 0
	// (equivalent to unset) is preserved; any other value is cleared.
	if c.Frequency == 0 {
		model.FrequencyOffset = types.Int64Value(int64(c.FrequencyOffset))
	} else if model.FrequencyOffset.IsUnknown() ||
		(!model.FrequencyOffset.IsNull() && model.FrequencyOffset.ValueInt64() != 0) {
		model.FrequencyOffset = types.Int64Null()
	}

	model.Activated = types.BoolValue(c.Activated)
	model.Muted = types.BoolValue(c.Muted)
	model.RunParallel = types.BoolValue(c.RunParallel)
	model.Locations = stringSetFromAPINullIfUnset(ctx, c.Locations, model.Locations, diags)
	model.DegradedPacketLossThreshold = types.Int64Value(int64(c.DegradedPacketLossThreshold))
	model.MaxPacketLossThreshold = types.Int64Value(int64(c.MaxPacketLossThreshold))
	model.Tags = stringSetFromAPINullIfUnset(ctx, c.Tags, model.Tags, diags)

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
	model.Request = icmpMonitorRequestToObject(ctx, c.Request, model.Request, diags)
	model.GroupID = types.Int64Value(c.GroupID)
	model.GroupOrder = types.Int64Value(int64(c.GroupOrder))
	model.AlertChannelSubscriptions = alertChannelSubscriptionsToSet(ctx, c.AlertChannelSubscriptions, model.AlertChannelSubscriptions, diags)
	model.RetryStrategy = retryStrategyToObject(ctx, c.RetryStrategy, model.RetryStrategy, diags)
	model.TriggerIncident = triggerIncidentToObject(ctx, c.TriggerIncident, diags)
}

// icmpMonitorRequestFromObject converts the request attribute value to its
// API representation.
func icmpMonitorRequestFromObject(ctx context.Context, obj types.Object, diags *diag.Diagnostics) checkly.ICMPRequest {
	model := objectAs[ICMPMonitorRequestModel](ctx, obj, diags)
	if model == nil {
		return checkly.ICMPRequest{}
	}

	return checkly.ICMPRequest{
		Hostname:   model.Hostname.ValueString(),
		IPFamily:   checkly.IPFamily(model.IPFamily.ValueString()),
		PingCount:  int(model.PingCount.ValueInt64()),
		Assertions: assertionsFromSet(ctx, model.Assertions, diags),
	}
}

// icmpMonitorRequestToObject converts the API representation of the request
// to the request attribute value.
func icmpMonitorRequestToObject(ctx context.Context, r checkly.ICMPRequest, prior types.Object, diags *diag.Diagnostics) types.Object {
	obj, d := types.ObjectValueFrom(ctx, icmpMonitorRequestAttrTypes, ICMPMonitorRequestModel{
		Hostname:   types.StringValue(r.Hostname),
		IPFamily:   types.StringValue(string(r.IPFamily)),
		PingCount:  types.Int64Value(int64(r.PingCount)),
		Assertions: assertionsToSet(ctx, r.Assertions, priorAssertions(prior), diags),
	})
	diags.Append(d...)

	return obj
}
