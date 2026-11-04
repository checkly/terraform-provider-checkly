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
	_ resource.Resource                     = (*DNSMonitorResource)(nil)
	_ resource.ResourceWithConfigure        = (*DNSMonitorResource)(nil)
	_ resource.ResourceWithUpgradeState     = (*DNSMonitorResource)(nil)
	_ resource.ResourceWithConfigValidators = (*DNSMonitorResource)(nil)
	_ resource.ResourceWithImportState      = (*DNSMonitorResource)(nil)
	_ resource.ResourceWithModifyPlan       = (*DNSMonitorResource)(nil)
)

// NewDNSMonitorResource returns a new checkly_dns_monitor resource.
func NewDNSMonitorResource() resource.Resource {
	return &DNSMonitorResource{}
}

// DNSMonitorResource implements the checkly_dns_monitor resource.
type DNSMonitorResource struct {
	client checkly.Client
}

// DNSMonitorResourceModel describes the resource data model.
type DNSMonitorResourceModel struct {
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
	AlertChannelSubscriptions types.Set    `tfsdk:"alert_channel_subscription"`
	AlertSettings             types.Object `tfsdk:"alert_settings"`
	UseGlobalAlertSettings    types.Bool   `tfsdk:"use_global_alert_settings"`
	Request                   types.Object `tfsdk:"request"`
	GroupID                   types.Int64  `tfsdk:"group_id"`
	GroupOrder                types.Int64  `tfsdk:"group_order"`
	RetryStrategy             types.Object `tfsdk:"retry_strategy"`
	TriggerIncident           types.Object `tfsdk:"trigger_incident"`
}

// DNSMonitorRequestModel describes the request attribute data model.
type DNSMonitorRequestModel struct {
	RecordType types.String `tfsdk:"record_type"`
	Query      types.String `tfsdk:"query"`
	Protocol   types.String `tfsdk:"protocol"`
	NameServer types.Object `tfsdk:"name_server"`
	Assertions types.Set    `tfsdk:"assertion"`
}

// DNSMonitorNameServerModel describes the request.name_server attribute
// data model.
type DNSMonitorNameServerModel struct {
	Host types.String `tfsdk:"host"`
	Port types.Int64  `tfsdk:"port"`
}

var dnsMonitorNameServerAttrTypes = map[string]attr.Type{
	"host": types.StringType,
	"port": types.Int64Type,
}

var dnsMonitorRequestAttrTypes = map[string]attr.Type{
	"record_type": types.StringType,
	"query":       types.StringType,
	"protocol":    types.StringType,
	"name_server": types.ObjectType{AttrTypes: dnsMonitorNameServerAttrTypes},
	"assertion":   types.SetType{ElemType: types.ObjectType{AttrTypes: assertionAttrTypes}},
}

func (r *DNSMonitorResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_monitor"
}

func (r *DNSMonitorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     sdkv2SchemaVersion + 1,
		Description: "Creates a DNS Monitor to check DNS record availability and response times.",
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
				Description: "An array of one or more data center locations where to run the this monitor.",
			},
			"degraded_response_time": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(500),
				Description: "The response time in milliseconds where the monitor should be considered degraded. Possible values are between `0` and `5000`. (Default `500`).",
				Validators: []validator.Int64{
					int64validator.Between(0, 5000),
				},
			},
			"max_response_time": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1000),
				Description: "The response time in milliseconds where the monitor should be considered failing. Possible values are between `0` and `5000`. (Default `1000`).",
				Validators: []validator.Int64{
					int64validator.Between(0, 5000),
				},
			},
			"tags": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "A list of tags for organizing and filtering checks and monitors.",
			},
			triggerIncidentAttributeName: makeTriggerIncidentSchema(),
			alertSettingsAttributeName: makeAlertSettingsAttributeSchema(AlertSettingsAttributeSchemaOptions{
				Monitor: true,
			}),
			"use_global_alert_settings": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "When true, the account level alert settings will be used, not the alert setting defined on this monitor. (Default `true`).",
			},
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
				Description: "The parameters of the DNS request.",
				Attributes: map[string]schema.Attribute{
					"record_type": schema.StringAttribute{
						Required:    true,
						Description: "The DNS record type. Possible values are `A`, `AAAA`, `CNAME`, `MX`, `NS`, `TXT` and `SOA`.",
						Validators: []validator.String{
							stringvalidator.OneOf("A", "AAAA", "CNAME", "MX", "NS", "TXT", "SOA"),
						},
					},
					"query": schema.StringAttribute{
						Required:    true,
						Description: "The DNS record to query.",
					},
					"protocol": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("UDP"),
						Description: "The protocol used to communicate with the name server. Possible values are `UDP` and `TCP`. (Default `UDP`).",
						Validators: []validator.String{
							stringvalidator.OneOf("UDP", "TCP"),
						},
					},
					"name_server": schema.SingleNestedAttribute{
						Optional:    true,
						Description: "The name server to use.",
						Validators: []validator.Object{
							dnsMonitorNameServerValidator{},
						},
						Attributes: map[string]schema.Attribute{
							"host": schema.StringAttribute{
								Optional:    true,
								Description: "The name server host.",
								Validators: []validator.String{
									stringvalidator.AlsoRequires(
										path.MatchRelative().AtParent().AtName("port"),
									),
								},
							},
							"port": schema.Int64Attribute{
								Optional:    true,
								Description: "The name server port.",
								Validators: []validator.Int64{
									int64validator.AlsoRequires(
										path.MatchRelative().AtParent().AtName("host"),
									),
								},
							},
						},
					},
					"assertion": schema.SetNestedAttribute{
						Optional:    true,
						Computed:    true,
						Default:     setdefault.StaticValue(types.SetValueMust(types.ObjectType{AttrTypes: assertionAttrTypes}, []attr.Value{})),
						Description: "Assertions to validate the DNS response.",
						NestedObject: schema.NestedAttributeObject{
							// Attributes nested in this set attribute must not use
							// schema defaults. The framework looks up configuration
							// values for set elements by full element value; once
							// computed attributes resolve, the lookup misses, the
							// configuration is treated as null, and defaults would
							// override values the practitioner actually configured,
							// producing a perpetual diff. Default values are instead
							// applied in ModifyPlan.
							Attributes: map[string]schema.Attribute{
								"source": schema.StringAttribute{
									Required:    true,
									Description: "The source of the asserted value. Possible values are `RESPONSE_CODE`, `RESPONSE_TIME`, `TEXT_ANSWER` and `JSON_ANSWER`.",
									Validators: []validator.String{
										stringvalidator.OneOf("RESPONSE_CODE", "RESPONSE_TIME", "TEXT_ANSWER", "JSON_ANSWER"),
									},
								},
								"property": schema.StringAttribute{
									Optional: true,
									Computed: true,
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
									Description: "The target value. Typically `NOERROR` when the source is `RESPONSE_CODE`.",
								},
							},
						},
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			alertChannelSubscriptionAttributeName: makeAlertChannelSubscriptionSchema(AlertChannelSubscriptionAttributeSchemaOptions{
				Monitor: true,
			}),
		},
	}
}

func (r *DNSMonitorResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		newFrequencyOffsetConfigValidator(),
	}
}

// UpgradeState upgrades state written by the provider's SDK v2
// implementation; see sdkv2StateUpgraders.
func (r *DNSMonitorResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return sdkv2StateUpgraders(ctx, r)
}

func (r *DNSMonitorResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

// ModifyPlan applies default values to the optional+computed attributes of
// the request assertions. Schema defaults cannot be used for attributes
// nested in set elements (see the schema for details), so the plan is
// adjusted here instead. Substituting static values keeps every set element
// fully known and stable across plans, which set element correlation
// requires.
func (r *DNSMonitorResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planNestedDefaults(ctx, req, resp, path.Root("request"), objectWithSetDefaults("assertion", assertionsWithPropertyDefault))
}

func (r *DNSMonitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DNSMonitorResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	monitor := dnsMonitorFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.CreateDNSMonitor(apiCtx, monitor)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create DNS Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	// State is written even when the conversion produces error diagnostics.
	// The monitor already exists remotely at this point, and returning
	// without saving state would leave it unmanaged.
	dnsMonitorToModel(ctx, result, &plan, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DNSMonitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DNSMonitorResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	monitor, err := r.client.GetDNSMonitor(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Read DNS Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	dnsMonitorToModel(ctx, monitor, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DNSMonitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DNSMonitorResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	monitor := dnsMonitorFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.UpdateDNSMonitor(apiCtx, monitor.ID, monitor)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update DNS Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	dnsMonitorToModel(ctx, result, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DNSMonitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DNSMonitorResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	err := r.client.DeleteDNSMonitor(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Delete DNS Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}
}

func (r *DNSMonitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// dnsMonitorFromModel converts the resource data model to its API
// representation.
func dnsMonitorFromModel(ctx context.Context, model *DNSMonitorResourceModel, diags *diag.Diagnostics) checkly.DNSMonitor {
	// An empty description is not tracked by the Checkly API; send null
	// instead so that updates clear a previously configured description.
	var description *string
	if value := model.Description.ValueString(); value != "" {
		description = &value
	}

	// The locations property is serialized even when empty.
	locations := stringsFromSetOrEmpty(ctx, model.Locations, diags)

	alertSettings := alertSettingsFromObject(ctx, model.AlertSettings, diags)

	return checkly.DNSMonitor{
		ID:                        model.ID.ValueString(),
		Name:                      model.Name.ValueString(),
		Description:               description,
		Frequency:                 int(model.Frequency.ValueInt64()),
		FrequencyOffset:           int(model.FrequencyOffset.ValueInt64()),
		Activated:                 model.Activated.ValueBool(),
		Muted:                     model.Muted.ValueBool(),
		RunParallel:               model.RunParallel.ValueBool(),
		Locations:                 locations,
		DegradedResponseTime:      int(model.DegradedResponseTime.ValueInt64()),
		MaxResponseTime:           int(model.MaxResponseTime.ValueInt64()),
		Tags:                      stringsFromSet(ctx, model.Tags, diags),
		AlertSettings:             &alertSettings,
		UseGlobalAlertSettings:    model.UseGlobalAlertSettings.ValueBool(),
		Request:                   dnsMonitorRequestFromObject(ctx, model.Request, diags),
		GroupID:                   model.GroupID.ValueInt64(),
		GroupOrder:                int(model.GroupOrder.ValueInt64()),
		AlertChannelSubscriptions: alertChannelSubscriptionsFromSet(ctx, model.AlertChannelSubscriptions, diags),
		RetryStrategy:             retryStrategyFromObject(ctx, model.RetryStrategy, diags),
		TriggerIncident:           triggerIncidentFromObject(ctx, model.TriggerIncident, diags),
	}
}

// dnsMonitorToModel updates the model to match the API representation of the
// monitor. The values already present in the model (from the plan on
// create/update, from the state on read) are used to resolve fields that the
// API does not track faithfully.
func dnsMonitorToModel(ctx context.Context, c *checkly.DNSMonitor, model *DNSMonitorResourceModel, diags *diag.Diagnostics) {
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
	model.DegradedResponseTime = types.Int64Value(int64(c.DegradedResponseTime))
	model.MaxResponseTime = types.Int64Value(int64(c.MaxResponseTime))
	model.Tags = stringSetFromAPINullIfUnset(ctx, c.Tags, model.Tags, diags)

	if c.AlertSettings != nil {
		model.AlertSettings = alertSettingsToObject(ctx, *c.AlertSettings, model.AlertSettings, diags)
	} else if model.AlertSettings.IsUnknown() {
		model.AlertSettings = types.ObjectNull(alertSettingsAttrTypes)
	}

	model.UseGlobalAlertSettings = types.BoolValue(c.UseGlobalAlertSettings)
	model.Request = dnsMonitorRequestToObject(ctx, c.Request, diags)
	model.GroupID = types.Int64Value(c.GroupID)
	model.GroupOrder = types.Int64Value(int64(c.GroupOrder))
	model.AlertChannelSubscriptions = alertChannelSubscriptionsToSet(ctx, c.AlertChannelSubscriptions, diags)
	model.RetryStrategy = retryStrategyToObject(ctx, c.RetryStrategy, model.RetryStrategy, diags)
	model.TriggerIncident = triggerIncidentToObject(ctx, c.TriggerIncident, diags)
}

// dnsMonitorRequestFromObject converts the request attribute value to its
// API representation.
func dnsMonitorRequestFromObject(ctx context.Context, obj types.Object, diags *diag.Diagnostics) checkly.DNSRequest {
	model := objectAs[DNSMonitorRequestModel](ctx, obj, diags)
	if model == nil {
		return checkly.DNSRequest{}
	}

	nameServer := dnsMonitorNameServerFromObject(ctx, model.NameServer, diags)

	return checkly.DNSRequest{
		RecordType: model.RecordType.ValueString(),
		Query:      model.Query.ValueString(),
		NameServer: nameServer.Host.ValueString(),
		Port:       int(nameServer.Port.ValueInt64()),
		Protocol:   model.Protocol.ValueString(),
		Assertions: assertionsFromSet(ctx, model.Assertions, diags),
	}
}

// dnsMonitorRequestToObject converts the API representation of the request
// to the request attribute value.
func dnsMonitorRequestToObject(ctx context.Context, r checkly.DNSRequest, diags *diag.Diagnostics) types.Object {
	obj, d := types.ObjectValueFrom(ctx, dnsMonitorRequestAttrTypes, DNSMonitorRequestModel{
		RecordType: types.StringValue(r.RecordType),
		Query:      types.StringValue(r.Query),
		Protocol:   types.StringValue(r.Protocol),
		NameServer: dnsMonitorNameServerToObject(ctx, r.NameServer, r.Port, diags),
		Assertions: assertionsToSet(ctx, r.Assertions, diags),
	})
	diags.Append(d...)

	return obj
}

// dnsMonitorNameServerFromObject converts the request.name_server attribute
// value to its host and port components. An unset name server converts to
// zero values.
func dnsMonitorNameServerFromObject(ctx context.Context, obj types.Object, diags *diag.Diagnostics) DNSMonitorNameServerModel {
	model := objectAs[DNSMonitorNameServerModel](ctx, obj, diags)
	if model == nil {
		return DNSMonitorNameServerModel{}
	}

	return *model
}

// dnsMonitorNameServerToObject converts the API representation of the name
// server to the request.name_server attribute value. An unconfigured name
// server converts to a null object.
func dnsMonitorNameServerToObject(ctx context.Context, host string, port int, diags *diag.Diagnostics) types.Object {
	if host == "" && port == 0 {
		return types.ObjectNull(dnsMonitorNameServerAttrTypes)
	}

	obj, d := types.ObjectValueFrom(ctx, dnsMonitorNameServerAttrTypes, DNSMonitorNameServerModel{
		Host: types.StringValue(host),
		Port: types.Int64Value(int64(port)),
	})
	diags.Append(d...)

	return obj
}

var _ validator.Object = dnsMonitorNameServerValidator{}

// dnsMonitorNameServerValidator rejects an empty name_server object. An empty
// object would plan a name server with a null host and port, but the Checkly
// API does not track such a name server and would report none back,
// resulting in an inconsistent result after apply.
type dnsMonitorNameServerValidator struct{}

func (v dnsMonitorNameServerValidator) Description(ctx context.Context) string {
	return "at least one of host and port must be configured"
}

func (v dnsMonitorNameServerValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v dnsMonitorNameServerValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	attrs := req.ConfigValue.Attributes()
	host := attrs["host"]
	port := attrs["port"]

	if host.IsUnknown() || port.IsUnknown() {
		return
	}

	if host.IsNull() && port.IsNull() {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Name Server",
			"The name_server attribute must configure the host and port attributes. "+
				"Remove the empty name_server object to use the default name server.",
		)
	}
}
