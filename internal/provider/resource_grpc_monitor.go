package provider

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
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
	_ resource.Resource                     = (*GRPCMonitorResource)(nil)
	_ resource.ResourceWithConfigure        = (*GRPCMonitorResource)(nil)
	_ resource.ResourceWithConfigValidators = (*GRPCMonitorResource)(nil)
	_ resource.ResourceWithImportState      = (*GRPCMonitorResource)(nil)
	_ resource.ResourceWithModifyPlan       = (*GRPCMonitorResource)(nil)
	_ resource.ResourceWithValidateConfig   = (*GRPCMonitorResource)(nil)
)

// NewGRPCMonitorResource returns a new checkly_grpc_monitor resource.
func NewGRPCMonitorResource() resource.Resource {
	return &GRPCMonitorResource{}
}

// GRPCMonitorResource implements the checkly_grpc_monitor resource.
type GRPCMonitorResource struct {
	client checkly.Client
}

// GRPCMonitorResourceModel describes the resource data model.
type GRPCMonitorResourceModel struct {
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
	Request                  types.List   `tfsdk:"request"`
	AlertChannelSubscription types.Set    `tfsdk:"alert_channel_subscription"`
	TriggerIncident          types.Set    `tfsdk:"trigger_incident"`
}

// GRPCMonitorRequestModel describes the request block data model.
type GRPCMonitorRequestModel struct {
	Host              types.String `tfsdk:"host"`
	Port              types.Int64  `tfsdk:"port"`
	IPFamily          types.String `tfsdk:"ip_family"`
	GRPCMode          types.String `tfsdk:"grpc_mode"`
	TLS               types.Bool   `tfsdk:"tls"`
	SkipSSL           types.Bool   `tfsdk:"skip_ssl"`
	Timeout           types.Int64  `tfsdk:"timeout"`
	Encoding          types.String `tfsdk:"encoding"`
	Service           types.String `tfsdk:"service"`
	ServiceDefinition types.String `tfsdk:"service_definition"`
	ProtoContent      types.String `tfsdk:"proto_content"`
	BfbsContent       types.String `tfsdk:"bfbs_content"`
	Method            types.String `tfsdk:"method"`
	Message           types.String `tfsdk:"message"`
	Metadata          types.Set    `tfsdk:"metadata"`
	Assertion         types.Set    `tfsdk:"assertion"`
}

// GRPCMonitorMetadataModel describes the request.metadata block data model.
type GRPCMonitorMetadataModel struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

// GRPCMonitorAssertionModel describes the request.assertion block data model.
type GRPCMonitorAssertionModel struct {
	Source     types.String `tfsdk:"source"`
	Property   types.String `tfsdk:"property"`
	Comparison types.String `tfsdk:"comparison"`
	Target     types.String `tfsdk:"target"`
}

var grpcMonitorMetadataAttrTypes = map[string]attr.Type{
	"key":   types.StringType,
	"value": types.StringType,
}

var grpcMonitorAssertionAttrTypes = map[string]attr.Type{
	"source":     types.StringType,
	"property":   types.StringType,
	"comparison": types.StringType,
	"target":     types.StringType,
}

var grpcMonitorRequestAttrTypes = map[string]attr.Type{
	"host":               types.StringType,
	"port":               types.Int64Type,
	"ip_family":          types.StringType,
	"grpc_mode":          types.StringType,
	"tls":                types.BoolType,
	"skip_ssl":           types.BoolType,
	"timeout":            types.Int64Type,
	"encoding":           types.StringType,
	"service":            types.StringType,
	"service_definition": types.StringType,
	"proto_content":      types.StringType,
	"bfbs_content":       types.StringType,
	"method":             types.StringType,
	"message":            types.StringType,
	"metadata":           types.SetType{ElemType: types.ObjectType{AttrTypes: grpcMonitorMetadataAttrTypes}},
	"assertion":          types.SetType{ElemType: types.ObjectType{AttrTypes: grpcMonitorAssertionAttrTypes}},
}

func (r *GRPCMonitorResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_grpc_monitor"
}

func (r *GRPCMonitorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "gRPC monitors allow you to monitor gRPC services, either via a health check (HEALTH mode) or by invoking a unary method (BEHAVIOR mode).",
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
				Description: "The response time in milliseconds starting from which a monitor should be considered degraded. Possible values are between 0 and 180000. (Default `10000`).",
				Validators: []validator.Int64{
					int64validator.Between(0, 180000),
				},
			},
			"max_response_time": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(20000),
				Description: "The response time in milliseconds starting from which a monitor should be considered failing. Possible values are between 0 and 180000. (Default `20000`).",
				Validators: []validator.Int64{
					int64validator.Between(0, 180000),
				},
			},
			"tags": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "A list of tags for organizing and filtering checks and monitors.",
			},
			"runtime_id": schema.StringAttribute{
				Optional:    true,
				Description: "The ID of the runtime to use for this monitor.",
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
				Description: "When true, the account level alert settings will be used, not the alert setting defined on this monitor.",
			},
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
		},
		Blocks: map[string]schema.Block{
			"request": schema.ListNestedBlock{
				Description: "The parameters for the gRPC connection.",
				Validators: []validator.List{
					listvalidator.IsRequired(),
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"host": schema.StringAttribute{
							Required:    true,
							Description: "The host to connect to. Do not include a scheme or a port in this value.",
						},
						"port": schema.Int64Attribute{
							Required:    true,
							Description: "The port number to connect to. Possible values are between 1 and 65535.",
							Validators: []validator.Int64{
								int64validator.Between(1, 65535),
							},
						},
						"ip_family": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("IPv4"),
							Description: "The IP family to use when executing the gRPC check. The value can be either `IPv4` or `IPv6`. (Default `IPv4`).",
							Validators: []validator.String{
								stringvalidator.OneOf("IPv4", "IPv6"),
							},
						},
						"grpc_mode": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("BEHAVIOR"),
							Description: "The gRPC monitoring mode. `BEHAVIOR` invokes a unary method (requires `method`); `HEALTH` queries the standard gRPC health-check service (allows `service`). (Default `BEHAVIOR`).",
							Validators: []validator.String{
								stringvalidator.OneOf("BEHAVIOR", "HEALTH"),
							},
						},
						"tls": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(true),
							Description: "Whether to use a TLS-encrypted connection to the gRPC server. (Default `true`).",
						},
						"skip_ssl": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
							Description: "Whether to skip SSL certificate validation when `tls` is enabled. (Default `false`).",
						},
						// timeout has the API's default rather than keeping
						// its prior state value: the update endpoint keeps the
						// stored value of an omitted field, so removing
						// timeout from the configuration must send the
						// default explicitly.
						"timeout": schema.Int64Attribute{
							Optional:    true,
							Computed:    true,
							Default:     int64default.StaticInt64(60),
							Description: "The number of seconds to wait for the gRPC call to complete before timing out. Possible values are between 1 and 180. (Default `60`).",
							Validators: []validator.Int64{
								int64validator.Between(1, 180),
							},
						},
						"service": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString(""),
							Description: "The service name to query in `HEALTH` mode. An empty value queries overall server health. Forbidden in `BEHAVIOR` mode.",
						},
						// encoding and service_definition have schema
						// defaults rather than keeping their prior state
						// value: the update endpoint keeps stored values for
						// omitted fields, so removing either from the
						// configuration must send the default explicitly to
						// switch the monitor back.
						"encoding": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("PROTOBUF"),
							Description: "The wire encoding used in `BEHAVIOR` mode. Possible values are `PROTOBUF` and `FLATBUFFERS`. (Default `PROTOBUF`).",
							Validators: []validator.String{
								stringvalidator.OneOf("PROTOBUF", "FLATBUFFERS"),
							},
						},
						"service_definition": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("REFLECTION"),
							Description: "How the service definition is resolved in `BEHAVIOR` mode: `REFLECTION` uses server reflection; `PROTO_FILE` uses the inline `proto_content`. Forbidden when `encoding = \"FLATBUFFERS\"`. (Default `REFLECTION`).",
							Validators: []validator.String{
								stringvalidator.OneOf("REFLECTION", "PROTO_FILE"),
							},
						},
						"proto_content": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString(""),
							Description: "The inline `.proto` file source used when `service_definition = \"PROTO_FILE\"` in `BEHAVIOR` mode. Forbidden when `encoding = \"FLATBUFFERS\"`.",
						},
						"bfbs_content": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString(""),
							Description: "A base64-encoded binary FlatBuffers schema (`.bfbs`), at most 100 KiB once encoded. Required when `encoding = \"FLATBUFFERS\"` and forbidden otherwise. Generate it with `flatc -b --schema --bfbs-builtins schema.fbs` and load it with Terraform's `filebase64()` function.",
							Validators: []validator.String{
								grpcMonitorBfbsContentValidator{},
							},
						},
						"method": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString(""),
							Description: "The fully-qualified gRPC method to invoke in `BEHAVIOR` mode (e.g. `package.Service/Method`). Required in `BEHAVIOR` mode; forbidden in `HEALTH` mode.",
						},
						"message": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString(""),
							Description: "The JSON request message sent as the gRPC call payload in `BEHAVIOR` mode.",
						},
					},
					Blocks: map[string]schema.Block{
						"metadata": schema.SetNestedBlock{
							Description: "gRPC metadata (request headers) sent with the call.",
							NestedObject: schema.NestedBlockObject{
								// Attributes nested in this set block must not
								// use schema defaults. The framework looks up
								// configuration values for set elements by full
								// element value; once computed attributes
								// resolve, the lookup misses, the configuration
								// is treated as null, and defaults would
								// override values the practitioner actually
								// configured, producing a perpetual diff.
								// Default values are instead applied in
								// ModifyPlan.
								Attributes: map[string]schema.Attribute{
									"key": schema.StringAttribute{
										Required:    true,
										Description: "The gRPC metadata (header) key.",
									},
									"value": schema.StringAttribute{
										Optional:    true,
										Computed:    true,
										Description: "The gRPC metadata (header) value.",
									},
								},
							},
						},
						// The full per-source assertion rules (allowed
						// comparisons, property usage, target formats) are
						// documented in the "Assertion Reference" section of
						// templates/resources/grpc_monitor.md.tmpl. When
						// assertion sources or their rules change here or in
						// the API, update that template too — it is not
						// generated from this schema.
						"assertion": schema.SetNestedBlock{
							Description: "A request can have multiple assertions. The allowed comparisons, properties, and target formats depend on the assertion source — see the [Assertion Reference](#assertion-reference) below.",
							NestedObject: schema.NestedBlockObject{
								// Attributes nested in this set block must not
								// use schema defaults (see the metadata block).
								// Default values are instead applied in
								// ModifyPlan.
								Attributes: map[string]schema.Attribute{
									"source": schema.StringAttribute{
										Required:    true,
										Description: "The source of the asserted value. Possible values are `RESPONSE_TIME`, `GRPC_RESPONSE`, `TEXT_BODY`, `GRPC_METADATA`, `GRPC_HEALTHCHECK_STATUS`, and `GRPC_STATUS_CODE`.",
										Validators: []validator.String{
											stringvalidator.OneOf("RESPONSE_TIME", "GRPC_RESPONSE", "TEXT_BODY", "GRPC_METADATA", "GRPC_HEALTHCHECK_STATUS", "GRPC_STATUS_CODE"),
										},
									},
									"property": schema.StringAttribute{
										Optional:    true,
										Computed:    true,
										Description: "The property selecting the asserted value within the source, e.g. a JSONPath expression for `GRPC_RESPONSE` or a metadata key for `GRPC_METADATA`.",
									},
									"comparison": schema.StringAttribute{
										Required:    true,
										Description: "The type of comparison to be executed between expected and actual value of the assertion. For `GRPC_RESPONSE`, `TEXT_BODY` and `GRPC_METADATA`, possible values are `EQUALS`, `NOT_EQUALS`, `HAS_KEY`, `NOT_HAS_KEY`, `HAS_VALUE`, `NOT_HAS_VALUE`, `IS_EMPTY`, `NOT_EMPTY`, `GREATER_THAN`, `LESS_THAN`, `CONTAINS`, `NOT_CONTAINS`, `IS_NULL`, and `NOT_NULL`. For `RESPONSE_TIME` and `GRPC_STATUS_CODE`, possible values are `EQUALS`, `NOT_EQUALS`, `GREATER_THAN`, and `LESS_THAN`. For `GRPC_HEALTHCHECK_STATUS`, possible values are `EQUALS` and `NOT_EQUALS`.",
									},
									"target": schema.StringAttribute{
										Optional:    true,
										Computed:    true,
										Description: "The value to compare against. Must be numeric for `RESPONSE_TIME` (milliseconds), `GRPC_STATUS_CODE` (0-16), and `GRPC_HEALTHCHECK_STATUS` (0-3, where 0=UNKNOWN, 1=SERVING, 2=NOT_SERVING, 3=SERVICE_UNKNOWN).",
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

func (r *GRPCMonitorResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		newFrequencyOffsetConfigValidator(),
	}
}

// ValidateConfig rejects encoding/schema combinations that the API would
// otherwise silently discard or reject on apply. It reads the configuration
// rather than the planned values: service_definition holds its default
// `REFLECTION` even on FLATBUFFERS monitors, so only an explicitly configured
// value is a conflict.
func (r *GRPCMonitorResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var requests types.List
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("request"), &requests)...)
	if resp.Diagnostics.HasError() || requests.IsNull() || requests.IsUnknown() || len(requests.Elements()) == 0 {
		return
	}

	var models []GRPCMonitorRequestModel
	resp.Diagnostics.Append(requests.ElementsAs(ctx, &models, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := models[0]
	err := validateGRPCEncoding(grpcEncodingConfig{
		mode:              configuredString(request.GRPCMode, "BEHAVIOR"),
		encoding:          configuredString(request.Encoding, "PROTOBUF"),
		bfbsContent:       configuredPresence(request.BfbsContent),
		serviceDefinition: configuredPresence(request.ServiceDefinition),
		protoContent:      configuredPresence(request.ProtoContent),
	})
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("request").AtListIndex(0),
			"Invalid gRPC Encoding Configuration",
			err.Error(),
		)
	}
}

// grpcEncodingConfig holds the configured values validateGRPCEncoding needs.
// String values are "" when unknown at validation time; presence values are
// true when the attribute is set in the configuration, even to a value not
// yet known.
type grpcEncodingConfig struct {
	mode              string
	encoding          string
	bfbsContent       bool
	serviceDefinition bool
	protoContent      bool
}

func validateGRPCEncoding(c grpcEncodingConfig) error {
	switch {
	case c.mode == "HEALTH" && (c.encoding == "FLATBUFFERS" || c.bfbsContent):
		return errors.New("encoding = \"FLATBUFFERS\" and bfbs_content cannot be used when grpc_mode is HEALTH")
	case c.mode != "BEHAVIOR" || c.encoding == "":
		return nil
	case c.encoding == "PROTOBUF" && c.bfbsContent:
		return errors.New("bfbs_content can only be used when encoding is FLATBUFFERS")
	case c.encoding == "FLATBUFFERS" && !c.bfbsContent:
		return errors.New("bfbs_content is required when encoding is FLATBUFFERS")
	case c.encoding == "FLATBUFFERS" && c.serviceDefinition:
		return errors.New("service_definition cannot be used when encoding is FLATBUFFERS")
	case c.encoding == "FLATBUFFERS" && c.protoContent:
		return errors.New("proto_content cannot be used when encoding is FLATBUFFERS")
	}
	return nil
}

// configuredString returns a configured string, def when it is not set, or ""
// when its value is not known yet.
func configuredString(v types.String, def string) string {
	switch {
	case v.IsUnknown():
		return ""
	case v.IsNull():
		return def
	}
	return v.ValueString()
}

// configuredPresence reports whether a string attribute is set in the
// configuration. An empty string counts as unset, as it converts to the same
// API representation as an omitted attribute.
func configuredPresence(v types.String) bool {
	if v.IsUnknown() {
		return true
	}
	return !v.IsNull() && v.ValueString() != ""
}

// maxGRPCSchemaContentLength mirrors the API's limit on the encoded length of
// bfbs_content.
const maxGRPCSchemaContentLength = 100 * 1024

var _ validator.String = grpcMonitorBfbsContentValidator{}

// grpcMonitorBfbsContentValidator accepts an empty string (equivalent to an
// unset attribute) or a standard base64 string of at most
// maxGRPCSchemaContentLength characters.
type grpcMonitorBfbsContentValidator struct{}

func (v grpcMonitorBfbsContentValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("value must be empty or a base64 string of at most %d characters", maxGRPCSchemaContentLength)
}

func (v grpcMonitorBfbsContentValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v grpcMonitorBfbsContentValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	if value == "" {
		return
	}

	if _, err := base64.StdEncoding.DecodeString(value); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid FlatBuffers Schema",
			fmt.Sprintf("Attribute %s must be a base64 string: %s", req.Path, err),
		)
		return
	}

	if len(value) > maxGRPCSchemaContentLength {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid FlatBuffers Schema",
			fmt.Sprintf("Attribute %s must be at most %d characters long once encoded, got %d.", req.Path, maxGRPCSchemaContentLength, len(value)),
		)
	}
}

func (r *GRPCMonitorResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

// ModifyPlan applies default values to the optional+computed attributes
// nested in the request.metadata and request.assertion set blocks. Schema
// defaults cannot be used for these attributes (see the schema for details),
// so the plan is adjusted here instead. Substituting static values keeps
// every set element fully known and stable across plans, which set element
// correlation requires. The request block itself is a list, so its own
// attributes can keep using schema defaults.
func (r *GRPCMonitorResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// A destroy plan has no planned state to default values in.
	if req.Plan.Raw.IsNull() {
		return
	}

	var configRequest types.List
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

	var request types.List
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("request"), &request)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request = grpcMonitorRequestWithDefaults(ctx, request, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("request"), request)...)
}

// grpcMonitorRequestWithDefaults returns the request block value with default
// values substituted for null and unknown optional+computed attributes nested
// in the metadata and assertion set blocks.
func grpcMonitorRequestWithDefaults(ctx context.Context, list types.List, diags *diag.Diagnostics) types.List {
	if list.IsNull() || list.IsUnknown() {
		return list
	}

	var models []GRPCMonitorRequestModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return list
	}

	for i, model := range models {
		models[i].Metadata = grpcMonitorMetadataWithDefaults(ctx, model.Metadata, diags)
		models[i].Assertion = grpcMonitorAssertionsWithDefaults(ctx, model.Assertion, diags)
	}

	result, d := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: grpcMonitorRequestAttrTypes}, models)
	diags.Append(d...)

	return result
}

// grpcMonitorMetadataWithDefaults returns the request.metadata block value
// with default values substituted for null and unknown optional+computed
// attributes.
func grpcMonitorMetadataWithDefaults(ctx context.Context, set types.Set, diags *diag.Diagnostics) types.Set {
	if set.IsNull() || set.IsUnknown() {
		return set
	}

	var models []GRPCMonitorMetadataModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return set
	}

	for i, model := range models {
		models[i].Value = stringOrDefault(model.Value, "")
	}

	result, d := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: grpcMonitorMetadataAttrTypes}, models)
	diags.Append(d...)

	return result
}

// grpcMonitorAssertionsWithDefaults returns the request.assertion block value
// with default values substituted for null and unknown optional+computed
// attributes.
func grpcMonitorAssertionsWithDefaults(ctx context.Context, set types.Set, diags *diag.Diagnostics) types.Set {
	if set.IsNull() || set.IsUnknown() {
		return set
	}

	var models []GRPCMonitorAssertionModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return set
	}

	for i, model := range models {
		models[i].Property = stringOrDefault(model.Property, "")
		models[i].Target = stringOrDefault(model.Target, "")
	}

	result, d := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: grpcMonitorAssertionAttrTypes}, models)
	diags.Append(d...)

	return result
}

func (r *GRPCMonitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GRPCMonitorResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	monitor := grpcMonitorFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.CreateGRPCMonitor(apiCtx, monitor)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create gRPC Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	// State is written even when the conversion produces error diagnostics.
	// The monitor already exists remotely at this point, and returning
	// without saving state would leave it unmanaged.
	plan.ID = types.StringValue(result.ID)
	grpcMonitorToModel(ctx, result, &plan, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GRPCMonitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GRPCMonitorResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	monitor, err := r.client.GetGRPCMonitor(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Read gRPC Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	grpcMonitorToModel(ctx, monitor, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GRPCMonitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan GRPCMonitorResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	monitor := grpcMonitorFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.UpdateGRPCMonitor(apiCtx, plan.ID.ValueString(), monitor)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update gRPC Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	grpcMonitorToModel(ctx, result, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GRPCMonitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GRPCMonitorResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	err := r.client.DeleteGRPCMonitor(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Delete gRPC Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}
}

func (r *GRPCMonitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// grpcMonitorFromModel converts the resource model to its API representation.
func grpcMonitorFromModel(ctx context.Context, model *GRPCMonitorResourceModel, diags *diag.Diagnostics) checkly.GRPCMonitor {
	// The locations property is serialized even when empty.
	locations := stringsFromSet(ctx, model.Locations, diags)
	if locations == nil {
		locations = []string{}
	}

	privateLocations := stringsFromSet(ctx, model.PrivateLocations, diags)
	if privateLocations == nil {
		privateLocations = []string{}
	}

	monitor := checkly.GRPCMonitor{
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
		Request:                   grpcMonitorRequestFromList(ctx, model.Request, diags),
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

// grpcMonitorToModel updates the model to match the API representation of the
// monitor. The model must hold prior values (the plan on create/update, the
// state on read); they are preserved for attributes that the API does not
// track or echo faithfully.
func grpcMonitorToModel(ctx context.Context, c *checkly.GRPCMonitor, model *GRPCMonitorResourceModel, diags *diag.Diagnostics) {
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
	model.Locations = grpcMonitorStringSet(ctx, c.Locations, model.Locations, diags)
	model.DegradedResponseTime = types.Int64Value(int64(c.DegradedResponseTime))
	model.MaxResponseTime = types.Int64Value(int64(c.MaxResponseTime))
	model.Tags = grpcMonitorStringSet(ctx, c.Tags, model.Tags, diags)

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
	model.PrivateLocations = grpcMonitorStringSet(ctx, privateLocations, model.PrivateLocations, diags)

	if c.AlertSettings != nil {
		model.AlertSettings = alertSettingsToObject(ctx, *c.AlertSettings, model.AlertSettings, diags)
	} else if model.AlertSettings.IsUnknown() {
		model.AlertSettings = types.ObjectNull(alertSettingsAttrTypes)
	}

	model.UseGlobalAlertSettings = types.BoolValue(c.UseGlobalAlertSettings)
	model.GroupID = types.Int64Value(c.GroupID)
	model.GroupOrder = types.Int64Value(int64(c.GroupOrder))
	model.RetryStrategy = retryStrategyToObject(ctx, c.RetryStrategy, model.RetryStrategy, diags)
	model.Request = grpcMonitorRequestToList(ctx, c.Request, diags)
	model.AlertChannelSubscription = alertChannelSubscriptionsToSet(ctx, c.AlertChannelSubscriptions, diags)
	model.TriggerIncident = triggerIncidentToSet(ctx, c.TriggerIncident, diags)
}

// grpcMonitorStringSet converts a string slice from the API to a set value.
// An empty slice resolves to null when the prior value is null, so that an
// optional set that is absent from the configuration stays null in state.
func grpcMonitorStringSet(ctx context.Context, values []string, prior types.Set, diags *diag.Diagnostics) types.Set {
	if len(values) == 0 && (prior.IsNull() || prior.IsUnknown()) {
		return types.SetNull(types.StringType)
	}

	return setFromStrings(ctx, values, diags)
}

// grpcMonitorRequestFromList converts the request block value to its API
// representation.
func grpcMonitorRequestFromList(ctx context.Context, list types.List, diags *diag.Diagnostics) checkly.GRPCRequest {
	if list.IsNull() || list.IsUnknown() {
		return checkly.GRPCRequest{}
	}

	var models []GRPCMonitorRequestModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	if diags.HasError() || len(models) == 0 {
		return checkly.GRPCRequest{}
	}

	return grpcMonitorRequestFromModel(ctx, models[0], diags)
}

// grpcMonitorRequestFromModel converts a request block element to its API
// representation.
func grpcMonitorRequestFromModel(ctx context.Context, model GRPCMonitorRequestModel, diags *diag.Diagnostics) checkly.GRPCRequest {
	config := checkly.GRPCConfig{
		Mode:              model.GRPCMode.ValueString(),
		TLS:               model.TLS.ValueBool(),
		Encoding:          model.Encoding.ValueString(),
		Service:           model.Service.ValueString(),
		ServiceDefinition: model.ServiceDefinition.ValueString(),
		ProtoContent:      model.ProtoContent.ValueString(),
		BfbsContent:       model.BfbsContent.ValueString(),
		Method:            model.Method.ValueString(),
		Message:           model.Message.ValueString(),
		Metadata:          grpcMonitorMetadataFromSet(ctx, model.Metadata, diags),
	}

	// encoding and service_definition always hold at least their defaults.
	// Leave them out where they do not apply; the API ignores them there
	// anyway.
	if config.Mode == "HEALTH" {
		config.Encoding = ""
	}
	if config.Encoding == "FLATBUFFERS" {
		config.ServiceDefinition = ""
	}

	return checkly.GRPCRequest{
		URL:        model.Host.ValueString(),
		Port:       int(model.Port.ValueInt64()),
		IPFamily:   model.IPFamily.ValueString(),
		SkipSSL:    model.SkipSSL.ValueBool(),
		Timeout:    int(model.Timeout.ValueInt64()),
		Assertions: grpcMonitorAssertionsFromSet(ctx, model.Assertion, diags),
		GRPCConfig: config,
	}
}

// grpcMonitorRequestToList converts the API representation of a gRPC request
// to the request block value.
func grpcMonitorRequestToList(ctx context.Context, r checkly.GRPCRequest, diags *diag.Diagnostics) types.List {
	list, d := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: grpcMonitorRequestAttrTypes}, []GRPCMonitorRequestModel{
		{
			Host:              types.StringValue(r.URL),
			Port:              types.Int64Value(int64(grpcPortToInt(r.Port))),
			IPFamily:          types.StringValue(r.IPFamily),
			GRPCMode:          types.StringValue(r.GRPCConfig.Mode),
			TLS:               types.BoolValue(r.GRPCConfig.TLS),
			SkipSSL:           types.BoolValue(r.SkipSSL),
			Timeout:           types.Int64Value(int64(r.Timeout)),
			Encoding:          types.StringValue(r.GRPCConfig.Encoding),
			Service:           types.StringValue(r.GRPCConfig.Service),
			ServiceDefinition: types.StringValue(r.GRPCConfig.ServiceDefinition),
			ProtoContent:      types.StringValue(r.GRPCConfig.ProtoContent),
			BfbsContent:       types.StringValue(r.GRPCConfig.BfbsContent),
			Method:            types.StringValue(r.GRPCConfig.Method),
			Message:           types.StringValue(r.GRPCConfig.Message),
			Metadata:          grpcMonitorMetadataToSet(ctx, r.GRPCConfig.Metadata, diags),
			Assertion:         grpcMonitorAssertionsToSet(ctx, r.Assertions, diags),
		},
	})
	diags.Append(d...)

	return list
}

// grpcPortToInt coerces the SDK's open `interface{}` port (a JSON number
// decodes to float64, a templated port stays a string) back into an integer.
// A non-numeric template value yields 0, which the configuration overrides.
func grpcPortToInt(p any) int {
	switch v := p.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}

// grpcMonitorMetadataFromSet converts the request.metadata block value to its
// API representation.
func grpcMonitorMetadataFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) []checkly.GRPCMetadata {
	metadata := []checkly.GRPCMetadata{}

	if set.IsNull() || set.IsUnknown() {
		return metadata
	}

	var models []GRPCMonitorMetadataModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return metadata
	}

	for _, model := range models {
		metadata = append(metadata, checkly.GRPCMetadata{
			Key:   model.Key.ValueString(),
			Value: model.Value.ValueString(),
		})
	}

	return metadata
}

// grpcMonitorMetadataToSet converts the API representation of gRPC metadata
// to the request.metadata block value.
func grpcMonitorMetadataToSet(ctx context.Context, metadata []checkly.GRPCMetadata, diags *diag.Diagnostics) types.Set {
	models := make([]GRPCMonitorMetadataModel, len(metadata))
	for i, m := range metadata {
		models[i] = GRPCMonitorMetadataModel{
			Key:   types.StringValue(m.Key),
			Value: types.StringValue(m.Value),
		}
	}

	set, d := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: grpcMonitorMetadataAttrTypes}, models)
	diags.Append(d...)

	return set
}

// grpcMonitorAssertionsFromSet converts the request.assertion block value to
// its API representation.
func grpcMonitorAssertionsFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) []checkly.Assertion {
	assertions := []checkly.Assertion{}

	if set.IsNull() || set.IsUnknown() {
		return assertions
	}

	var models []GRPCMonitorAssertionModel
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

// grpcMonitorAssertionsToSet converts the API representation of assertions to
// the request.assertion block value.
func grpcMonitorAssertionsToSet(ctx context.Context, assertions []checkly.Assertion, diags *diag.Diagnostics) types.Set {
	models := make([]GRPCMonitorAssertionModel, len(assertions))
	for i, assertion := range assertions {
		models[i] = GRPCMonitorAssertionModel{
			Source:     types.StringValue(assertion.Source),
			Property:   types.StringValue(assertion.Property),
			Comparison: types.StringValue(assertion.Comparison),
			Target:     types.StringValue(assertion.Target),
		}
	}

	set, d := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: grpcMonitorAssertionAttrTypes}, models)
	diags.Append(d...)

	return set
}
