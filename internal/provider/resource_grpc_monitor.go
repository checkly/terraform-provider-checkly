package provider

import (
	"context"
	"encoding/base64"
	"errors"
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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/checkly/checkly-go-sdk/v2"
)

var (
	_ resource.Resource                     = (*GRPCMonitorResource)(nil)
	_ resource.ResourceWithConfigure        = (*GRPCMonitorResource)(nil)
	_ resource.ResourceWithUpgradeState     = (*GRPCMonitorResource)(nil)
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

// GRPCMonitorRequestModel describes the request attribute data model.
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
	Assertions        types.Set    `tfsdk:"assertions"`
}

// GRPCMonitorMetadataModel describes the data model of a request.metadata
// element.
type GRPCMonitorMetadataModel struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

var grpcMonitorMetadataAttrTypes = map[string]attr.Type{
	"key":   types.StringType,
	"value": types.StringType,
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
	"assertions":         types.SetType{ElemType: types.ObjectType{AttrTypes: assertionAttrTypes}},
}

func (r *GRPCMonitorResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_grpc_monitor"
}

func (r *GRPCMonitorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     sdkv2SchemaVersion + 1,
		Description: "gRPC monitors allow you to monitor gRPC services, either via a health check (HEALTH mode) or by invoking a unary method (BEHAVIOR mode).",
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
			// The SDK leaves a zero response time out of the request, so a 0
			// never reached the API, which kept its default or the stored
			// value. A 0 would only make the monitor always degraded or
			// always failing, so the minimum is 1.
			"degraded_response_time": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(10000),
				Description: "The response time in milliseconds where the monitor should be considered degraded. Possible values are between `1` and `180000`. (Default `10000`).",
				Validators: []validator.Int64{
					int64validator.Between(1, 180000),
				},
			},
			"max_response_time": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(20000),
				Description: "The response time in milliseconds where the monitor should be considered failing. Possible values are between `1` and `180000`. (Default `20000`).",
				Validators: []validator.Int64{
					int64validator.Between(1, 180000),
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
				Description: "The parameters for the gRPC connection.",
				Attributes: map[string]schema.Attribute{
					"host": schema.StringAttribute{
						Required:    true,
						Description: "The host to connect to (1 to 2048 characters). Do not include a scheme or a port in this value.",
						Validators: []validator.String{
							utf16LengthBetween(1, 2048),
							noControlCharacters(),
							hostWithoutSchemeOrPortValidator{},
						},
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
						Description: "The service name to query in `HEALTH` mode. An empty value queries overall server health. Must not be set in `BEHAVIOR` mode.",
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
						Description: "How the service definition is resolved in `BEHAVIOR` mode: `REFLECTION` uses server reflection; `PROTO_FILE` uses the inline `proto_content`, which it requires. Must not be set when `encoding = \"FLATBUFFERS\"`, and must not be `PROTO_FILE` in `HEALTH` mode. (Default `REFLECTION`).",
						Validators: []validator.String{
							stringvalidator.OneOf("REFLECTION", "PROTO_FILE"),
						},
					},
					"proto_content": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString(""),
						Description: "The inline `.proto` file source (at most 102400 characters) used when `service_definition = \"PROTO_FILE\"` in `BEHAVIOR` mode, where it is required. Must not be set in `HEALTH` mode or when `encoding = \"FLATBUFFERS\"`.",
						Validators: []validator.String{
							utf16LengthBetween(0, 102400),
							noNULCharacterValidator{},
						},
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
						Description: "The fully-qualified gRPC method to invoke in `BEHAVIOR` mode (e.g. `package.Service/Method`). Required in `BEHAVIOR` mode; must not be set in `HEALTH` mode.",
					},
					"message": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString(""),
						Description: "The JSON request message (at most 65536 characters) sent as the gRPC call payload in `BEHAVIOR` mode. Must not be set in `HEALTH` mode.",
						Validators: []validator.String{
							utf16LengthBetween(0, 65536),
							noNULCharacterValidator{},
						},
					},
					"metadata": schema.SetNestedAttribute{
						Optional:    true,
						Computed:    true,
						Default:     emptySetDefault(types.ObjectType{AttrTypes: grpcMonitorMetadataAttrTypes}),
						Description: "gRPC metadata (request headers) sent with the call. At most 50 elements.",
						Validators: []validator.Set{
							setvalidator.SizeAtMost(50),
							grpcMetadataUniqueKeysValidator{},
						},
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
								"key": schema.StringAttribute{
									Required:    true,
									Description: "The gRPC metadata (header) key (1 to 256 characters).",
									Validators: []validator.String{
										utf16LengthBetween(1, 256),
										noNULCharacterValidator{},
									},
								},
								"value": schema.StringAttribute{
									Optional:    true,
									Computed:    true,
									Description: "The gRPC metadata (header) value (at most 8192 characters). The value is not sensitive and shows in plan output. To keep a credential out of plan output, take it from a variable declared with `sensitive = true` or wrap it in `sensitive(...)`.",
									Validators: []validator.String{
										utf16LengthBetween(0, 8192),
										noNULCharacterValidator{},
									},
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
					"assertions": schema.SetNestedAttribute{
						Optional:    true,
						Computed:    true,
						Default:     emptySetDefault(types.ObjectType{AttrTypes: assertionAttrTypes}),
						Description: "A request can have multiple assertions. The allowed comparisons, properties, and target formats depend on the assertion source — see the [Assertion Reference](#assertion-reference) below.",
						NestedObject: schema.NestedAttributeObject{
							Validators: []validator.Object{
								grpcMonitorAssertionRules,
							},
							// Attributes nested in this set attribute must
							// not use schema defaults (see metadata).
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
									Description: "The value to compare against. Required, and must be numeric without surrounding whitespace, for `RESPONSE_TIME` (milliseconds), `GRPC_STATUS_CODE` (0-16), and `GRPC_HEALTHCHECK_STATUS` (0-3, where 0=UNKNOWN, 1=SERVING, 2=NOT_SERVING, 3=SERVICE_UNKNOWN).",
								},
							},
						},
					},
				},
			},
			alertChannelSubscriptionAttributeName: makeAlertChannelSubscriptionSchema(AlertChannelSubscriptionAttributeSchemaOptions{
				Subject: subjectMonitor,
			}),
		}, CommonCheckAttributeSchemaOptions{Subject: subjectMonitor, RunParallel: true, Locations: true, GroupMembership: true, RuntimeID: true}),
	}
}

func (r *GRPCMonitorResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		newFrequencyOffsetConfigValidator(),
	}
}

// ValidateConfig rejects mode and encoding/schema combinations that the API
// would otherwise silently discard or reject on apply. It reads the
// configuration rather than the planned values: service_definition holds its
// default `REFLECTION` even on FLATBUFFERS monitors, so only an explicitly
// configured value is a conflict.
func (r *GRPCMonitorResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var obj types.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("request"), &obj)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := objectAs[GRPCMonitorRequestModel](ctx, obj, &resp.Diagnostics)
	if request == nil {
		return
	}

	err := validateGRPCEncoding(grpcEncodingConfig{
		mode:              configuredString(request.GRPCMode, "BEHAVIOR"),
		encoding:          configuredString(request.Encoding, "PROTOBUF"),
		bfbsContent:       configuredPresence(request.BfbsContent),
		serviceDefinition: configuredPresence(request.ServiceDefinition),
		protoContent:      configuredPresence(request.ProtoContent),
	})
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("request"),
			"Invalid gRPC Encoding Configuration",
			err.Error(),
		)
	}

	err = validateGRPCMode(grpcModeConfig{
		mode:              configuredString(request.GRPCMode, "BEHAVIOR"),
		encoding:          configuredString(request.Encoding, "PROTOBUF"),
		serviceDefinition: configuredString(request.ServiceDefinition, "REFLECTION"),
		service:           configuredState(request.Service),
		method:            configuredState(request.Method),
		message:           configuredState(request.Message),
		protoContent:      configuredState(request.ProtoContent),
	})
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("request"),
			"Invalid gRPC Mode Configuration",
			err.Error(),
		)
	}
}

// valueState says whether a string attribute is set in the configuration.
type valueState int

const (
	// valueUnset is a null or empty value, which converts to the same API
	// representation as an omitted attribute.
	valueUnset valueState = iota
	// valueUnknown is a value that is not known yet. It may still turn out
	// empty.
	valueUnknown
	// valueSet is a known, non-empty value.
	valueSet
)

// configuredState returns the valueState of a configured string attribute.
func configuredState(v types.String) valueState {
	switch {
	case v.IsUnknown():
		return valueUnknown
	case v.IsNull() || v.ValueString() == "":
		return valueUnset
	}
	return valueSet
}

// grpcModeConfig holds the configured values validateGRPCMode needs. String
// values are "" when unknown at validation time, as in grpcEncodingConfig.
type grpcModeConfig struct {
	mode              string
	encoding          string
	serviceDefinition string
	service           valueState
	method            valueState
	message           valueState
	protoContent      valueState
}

// validateGRPCMode rejects attributes that don't apply to the mode, which the
// API drops from the request, so they would read back empty, and attributes
// that the mode or service definition requires, which the API rejects.
// FLATBUFFERS combinations are left to validateGRPCEncoding. A value that is
// not known yet may turn out empty, so it only fails a requirement, which it
// may meet, when it is not set at all: an unknown value is not rejected as
// an attribute that doesn't apply, and it counts as set for a required one.
func validateGRPCMode(c grpcModeConfig) error {
	switch c.mode {
	case "HEALTH":
		switch {
		case c.method == valueSet:
			return errors.New("method cannot be used when grpc_mode is HEALTH")
		case c.message == valueSet:
			return errors.New("message cannot be used when grpc_mode is HEALTH")
		case c.protoContent == valueSet:
			return errors.New("proto_content cannot be used when grpc_mode is HEALTH")
		case c.serviceDefinition == "PROTO_FILE":
			return errors.New("service_definition = \"PROTO_FILE\" cannot be used when grpc_mode is HEALTH")
		}
	case "BEHAVIOR":
		switch {
		case c.service == valueSet:
			return errors.New("service can only be used when grpc_mode is HEALTH")
		case c.method == valueUnset:
			return errors.New("method is required when grpc_mode is BEHAVIOR")
		case c.encoding == "PROTOBUF" && c.serviceDefinition == "PROTO_FILE" && c.protoContent == valueUnset:
			return errors.New("proto_content is required when service_definition is PROTO_FILE")
		}
	}
	return nil
}

// grpcMonitorAssertionRules are the API's rules for the comparisons and
// targets of each assertion source.
var grpcMonitorAssertionRules = assertionRulesValidator{rules: map[string]assertionSourceRule{
	"RESPONSE_TIME": {
		comparisons:   []string{"EQUALS", "NOT_EQUALS", "GREATER_THAN", "LESS_THAN"},
		numericTarget: nonNegativeMillisecondsTarget,
	},
	"GRPC_STATUS_CODE": {
		comparisons:   []string{"EQUALS", "NOT_EQUALS", "GREATER_THAN", "LESS_THAN"},
		numericTarget: &numericAssertionTarget{pattern: nonNegativeIntegerRegex, bounded: true, min: 0, max: 16, label: "an integer between 0 and 16"},
	},
	"GRPC_HEALTHCHECK_STATUS": {
		comparisons:   []string{"EQUALS", "NOT_EQUALS"},
		numericTarget: &numericAssertionTarget{pattern: nonNegativeIntegerRegex, bounded: true, min: 0, max: 3, label: "an integer between 0 and 3"},
	},
	"GRPC_RESPONSE": {comparisons: allAssertionComparisons},
	"TEXT_BODY":     {comparisons: allAssertionComparisons},
	"GRPC_METADATA": {comparisons: allAssertionComparisons},
}}

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

var _ validator.Set = grpcMetadataUniqueKeysValidator{}

// grpcMetadataUniqueKeysValidator rejects request.metadata elements whose
// keys are equal ignoring case, as the API does: gRPC metadata keys are
// case-insensitive. The set itself only removes identical elements, so it
// keeps two elements with the same key and different values, or keys that
// differ in case. The API compares keys lowercased by JavaScript's
// toLowerCase, which cases.Lower(language.Und) matches and strings.ToLower
// does not, for example for U+0130 and a final sigma. Elements and keys that
// are not known yet are skipped. The error names the keys but never the
// values, which may be secret.
type grpcMetadataUniqueKeysValidator struct{}

func (v grpcMetadataUniqueKeysValidator) Description(ctx context.Context) string {
	return "metadata keys must be unique ignoring case"
}

func (v grpcMetadataUniqueKeysValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v grpcMetadataUniqueKeysValidator) ValidateSet(ctx context.Context, req validator.SetRequest, resp *validator.SetResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	// A Caser is stateful, so each call gets its own.
	lower := cases.Lower(language.Und)
	seen := make(map[string]string)
	for _, element := range req.ConfigValue.Elements() {
		object, ok := element.(types.Object)
		if !ok || object.IsNull() || object.IsUnknown() {
			continue
		}
		key, ok := object.Attributes()["key"].(types.String)
		if !ok || key.IsNull() || key.IsUnknown() {
			continue
		}

		lowered := lower.String(key.ValueString())
		first, duplicate := seen[lowered]
		if !duplicate {
			seen[lowered] = key.ValueString()
			continue
		}

		detail := fmt.Sprintf("%s has the keys %q and %q, which differ only in case. gRPC metadata keys are case-insensitive, and the API requires each key to be unique ignoring case.", req.Path, first, key.ValueString())
		if first == key.ValueString() {
			detail = fmt.Sprintf("%s has more than one element with the key %q. The API requires each metadata key to be unique, ignoring case.", req.Path, first)
		}
		resp.Diagnostics.AddAttributeError(req.Path, "Duplicate gRPC Metadata Key", detail)
	}
}

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

// UpgradeState upgrades state written by the provider's SDK v2
// implementation; see sdkv2StateUpgraders.
func (r *GRPCMonitorResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return sdkv2StateUpgraders(ctx, r)
}

func (r *GRPCMonitorResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

// ModifyPlan applies default values to the optional+computed attributes of
// the request metadata and assertions. Schema defaults cannot be used for
// attributes nested in set elements (see the schema for details), so the
// plan is adjusted here instead. Substituting static values keeps every set
// element fully known and stable across plans, which set element correlation
// requires.
func (r *GRPCMonitorResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planNestedDefaults(ctx, req, resp, path.Root("request"), grpcMonitorRequestWithDefaults)
}

// grpcMonitorRequestWithDefaults returns the request attribute value with
// default values substituted for null and unknown optional+computed
// attributes nested in the metadata and assertions sets.
func grpcMonitorRequestWithDefaults(ctx context.Context, obj types.Object, diags *diag.Diagnostics) types.Object {
	obj = objectWithSetDefaults("metadata", grpcMonitorMetadataWithDefaults)(ctx, obj, diags)
	return objectWithSetDefaults("assertions", assertionsWithPropertyAndTargetDefaults)(ctx, obj, diags)
}

// grpcMonitorMetadataWithDefaults returns the request.metadata set value
// with default values substituted for null and unknown optional+computed
// attributes.
func grpcMonitorMetadataWithDefaults(ctx context.Context, set types.Set, diags *diag.Diagnostics) types.Set {
	return setElementsWithDefaults(ctx, set, func(m *GRPCMonitorMetadataModel) {
		m.Value = stringOrDefault(m.Value, "")
	}, diags)
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
		addAPIError(&resp.Diagnostics, "Unable to Create gRPC Monitor", err)
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
		addAPIError(&resp.Diagnostics, "Unable to Read gRPC Monitor", err)
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
		addAPIError(&resp.Diagnostics, "Unable to Update gRPC Monitor", err)
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
		addAPIError(&resp.Diagnostics, "Unable to Delete gRPC Monitor", err)
		return
	}
}

func (r *GRPCMonitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// grpcMonitorFromModel converts the resource model to its API representation.
func grpcMonitorFromModel(ctx context.Context, model *GRPCMonitorResourceModel, diags *diag.Diagnostics) checkly.GRPCMonitor {
	// The locations property is serialized even when empty.
	locations := stringsFromSetOrEmpty(ctx, model.Locations, diags)

	privateLocations := stringsFromSetOrEmpty(ctx, model.PrivateLocations, diags)

	monitor := checkly.GRPCMonitor{
		ID:                        model.ID.ValueString(),
		Name:                      model.Name.ValueString(),
		Frequency:                 int(model.Frequency.ValueInt64()),
		FrequencyOffset:           int(model.FrequencyOffset.ValueInt64()),
		Activated:                 model.Activated.ValueBool(),
		Muted:                     model.Muted.ValueBool(),
		RunParallel:               model.RunParallel.ValueBool(),
		Locations:                 locations,
		DegradedResponseTime:      int(model.DegradedResponseTime.ValueInt64()),
		MaxResponseTime:           int(model.MaxResponseTime.ValueInt64()),
		Tags:                      stringsFromSet(ctx, model.Tags, diags),
		UseGlobalAlertSettings:    model.UseGlobalAlertSettings.ValueBool(),
		GroupID:                   model.GroupID.ValueInt64(),
		GroupOrder:                groupOrderToAPI(model.GroupOrder),
		PrivateLocations:          &privateLocations,
		Request:                   grpcMonitorRequestFromObject(ctx, model.Request, diags),
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

	model.FrequencyOffset = frequencyOffsetFromAPI(c.Frequency, c.FrequencyOffset, model.FrequencyOffset)

	model.Activated = types.BoolValue(c.Activated)
	model.Muted = types.BoolValue(c.Muted)
	model.RunParallel = types.BoolValue(c.RunParallel)
	model.Locations = setFromStrings(ctx, c.Locations, diags)
	model.DegradedResponseTime = types.Int64Value(int64(c.DegradedResponseTime))
	model.MaxResponseTime = types.Int64Value(int64(c.MaxResponseTime))
	model.Tags = setFromStrings(ctx, c.Tags, diags)

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
	model.PrivateLocations = setFromStrings(ctx, privateLocations, diags)

	model.AlertSettings = alertSettingsFromAPI(ctx, c.AlertSettings, model.AlertSettings, diags)

	model.UseGlobalAlertSettings = types.BoolValue(c.UseGlobalAlertSettings)
	model.GroupID = int64ValueOrNull(c.GroupID)
	model.GroupOrder = groupOrderFromAPI(c.GroupOrder, model.GroupOrder)
	model.RetryStrategy = retryStrategyToObject(ctx, c.RetryStrategy, model.RetryStrategy, diags)
	model.Request = grpcMonitorRequestToObject(ctx, c.Request, diags)
	model.AlertChannelSubscriptions = alertChannelSubscriptionsToSet(ctx, c.AlertChannelSubscriptions, diags)
	model.TriggerIncident = triggerIncidentToObject(ctx, c.TriggerIncident, diags)
}

// grpcMonitorRequestFromObject converts the request attribute value to its
// API representation.
func grpcMonitorRequestFromObject(ctx context.Context, obj types.Object, diags *diag.Diagnostics) checkly.GRPCRequest {
	model := objectAs[GRPCMonitorRequestModel](ctx, obj, diags)
	if model == nil {
		return checkly.GRPCRequest{}
	}

	return grpcMonitorRequestFromModel(ctx, *model, diags)
}

// grpcMonitorRequestFromModel converts the request attribute model to its
// API representation.
func grpcMonitorRequestFromModel(ctx context.Context, model GRPCMonitorRequestModel, diags *diag.Diagnostics) checkly.GRPCRequest {
	config := checkly.GRPCConfig{
		Mode:              checkly.GRPCMode(model.GRPCMode.ValueString()),
		TLS:               model.TLS.ValueBool(),
		Encoding:          checkly.GRPCEncoding(model.Encoding.ValueString()),
		Service:           model.Service.ValueString(),
		ServiceDefinition: checkly.GRPCServiceDefinition(model.ServiceDefinition.ValueString()),
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
		IPFamily:   checkly.IPFamily(model.IPFamily.ValueString()),
		SkipSSL:    model.SkipSSL.ValueBool(),
		Timeout:    int(model.Timeout.ValueInt64()),
		Assertions: assertionsFromSet(ctx, model.Assertions, diags),
		GRPCConfig: config,
	}
}

// grpcMonitorRequestToObject converts the API representation of a gRPC
// request to the request attribute value.
func grpcMonitorRequestToObject(ctx context.Context, r checkly.GRPCRequest, diags *diag.Diagnostics) types.Object {
	obj, d := types.ObjectValueFrom(ctx, grpcMonitorRequestAttrTypes, GRPCMonitorRequestModel{
		Host:              types.StringValue(r.URL),
		Port:              types.Int64Value(int64(grpcPortToInt(r.Port))),
		IPFamily:          types.StringValue(string(r.IPFamily)),
		GRPCMode:          types.StringValue(string(r.GRPCConfig.Mode)),
		TLS:               types.BoolValue(r.GRPCConfig.TLS),
		SkipSSL:           types.BoolValue(r.SkipSSL),
		Timeout:           types.Int64Value(int64(r.Timeout)),
		Encoding:          types.StringValue(string(r.GRPCConfig.Encoding)),
		Service:           types.StringValue(r.GRPCConfig.Service),
		ServiceDefinition: types.StringValue(string(r.GRPCConfig.ServiceDefinition)),
		ProtoContent:      types.StringValue(r.GRPCConfig.ProtoContent),
		BfbsContent:       types.StringValue(r.GRPCConfig.BfbsContent),
		Method:            types.StringValue(r.GRPCConfig.Method),
		Message:           types.StringValue(r.GRPCConfig.Message),
		Metadata:          grpcMonitorMetadataToSet(ctx, r.GRPCConfig.Metadata, diags),
		Assertions:        assertionsToSet(ctx, r.Assertions, diags),
	})
	diags.Append(d...)

	return obj
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

// grpcMonitorMetadataFromSet converts the request.metadata set value to its
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
// to the request.metadata set value. No metadata converts to an empty set.
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
