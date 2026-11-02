package provider

import (
	"context"
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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	checkly "github.com/checkly/checkly-go-sdk"
)

var (
	_ resource.Resource                     = (*SSLMonitorResource)(nil)
	_ resource.ResourceWithConfigure        = (*SSLMonitorResource)(nil)
	_ resource.ResourceWithConfigValidators = (*SSLMonitorResource)(nil)
	_ resource.ResourceWithImportState      = (*SSLMonitorResource)(nil)
	_ resource.ResourceWithModifyPlan       = (*SSLMonitorResource)(nil)
)

// NewSSLMonitorResource returns a new checkly_ssl_monitor resource.
func NewSSLMonitorResource() resource.Resource {
	return &SSLMonitorResource{}
}

// SSLMonitorResource implements the checkly_ssl_monitor resource.
type SSLMonitorResource struct {
	client checkly.Client
}

// SSLMonitorResourceModel describes the resource data model.
type SSLMonitorResourceModel struct {
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

// SSLMonitorRequestModel describes the request block data model.
type SSLMonitorRequestModel struct {
	Hostname              types.String `tfsdk:"hostname"`
	Port                  types.Int64  `tfsdk:"port"`
	ServerName            types.String `tfsdk:"server_name"`
	IPFamily              types.String `tfsdk:"ip_family"`
	SkipChainValidation   types.Bool   `tfsdk:"skip_chain_validation"`
	HandshakeTimeoutMs    types.Int64  `tfsdk:"handshake_timeout_ms"`
	AlertDaysBeforeExpiry types.Int64  `tfsdk:"alert_days_before_expiry"`
	SecurityBaseline      types.List   `tfsdk:"security_baseline"`
	ClientCertificate     types.Object `tfsdk:"client_certificate"`
	Assertion             types.Set    `tfsdk:"assertion"`
}

// SSLMonitorSecurityBaselineModel describes the request.security_baseline
// block data model. Each rule is a list block holding at most one element.
type SSLMonitorSecurityBaselineModel struct {
	Enabled                 types.Bool `tfsdk:"enabled"`
	MinTLSVersion           types.List `tfsdk:"min_tls_version"`
	MinKeySizeBits          types.List `tfsdk:"min_key_size_bits"`
	WeakSignatureAlgorithm  types.List `tfsdk:"weak_signature_algorithm"`
	WeakCipherSuite         types.List `tfsdk:"weak_cipher_suite"`
	KnownBadCA              types.List `tfsdk:"known_bad_ca"`
	RecommendedTLSVersion   types.List `tfsdk:"recommended_tls_version"`
	RecommendedKeySizeBits  types.List `tfsdk:"recommended_key_size_bits"`
	OCSPMustStapleRespected types.List `tfsdk:"ocsp_must_staple_respected"`
	SCTPresent              types.List `tfsdk:"sct_present"`
}

// SSLMonitorBaselineTLSRuleModel describes a security baseline rule whose
// value is a TLS version.
type SSLMonitorBaselineTLSRuleModel struct {
	Value    types.String `tfsdk:"value"`
	Severity types.String `tfsdk:"severity"`
}

// SSLMonitorBaselineKeySizeRuleModel describes a security baseline rule whose
// value is a key size in bits.
type SSLMonitorBaselineKeySizeRuleModel struct {
	Value    types.Int64  `tfsdk:"value"`
	Severity types.String `tfsdk:"severity"`
}

// SSLMonitorBaselineSeverityRuleModel describes a security baseline rule that
// only carries a severity.
type SSLMonitorBaselineSeverityRuleModel struct {
	Severity types.String `tfsdk:"severity"`
}

// SSLMonitorClientCertificateModel describes the request.client_certificate
// attribute data model.
type SSLMonitorClientCertificateModel struct {
	Mode                types.String `tfsdk:"mode"`
	ClientCertificateID types.String `tfsdk:"client_certificate_id"`
}

// SSLMonitorAssertionModel describes the request.assertion block data model.
type SSLMonitorAssertionModel struct {
	Source     types.String `tfsdk:"source"`
	Property   types.String `tfsdk:"property"`
	Comparison types.String `tfsdk:"comparison"`
	Target     types.String `tfsdk:"target"`
}

var sslMonitorBaselineTLSRuleAttrTypes = map[string]attr.Type{
	"value":    types.StringType,
	"severity": types.StringType,
}

var sslMonitorBaselineKeySizeRuleAttrTypes = map[string]attr.Type{
	"value":    types.Int64Type,
	"severity": types.StringType,
}

var sslMonitorBaselineSeverityRuleAttrTypes = map[string]attr.Type{
	"severity": types.StringType,
}

var sslMonitorSecurityBaselineAttrTypes = map[string]attr.Type{
	"enabled":                    types.BoolType,
	"min_tls_version":            types.ListType{ElemType: types.ObjectType{AttrTypes: sslMonitorBaselineTLSRuleAttrTypes}},
	"min_key_size_bits":          types.ListType{ElemType: types.ObjectType{AttrTypes: sslMonitorBaselineKeySizeRuleAttrTypes}},
	"weak_signature_algorithm":   types.ListType{ElemType: types.ObjectType{AttrTypes: sslMonitorBaselineSeverityRuleAttrTypes}},
	"weak_cipher_suite":          types.ListType{ElemType: types.ObjectType{AttrTypes: sslMonitorBaselineSeverityRuleAttrTypes}},
	"known_bad_ca":               types.ListType{ElemType: types.ObjectType{AttrTypes: sslMonitorBaselineSeverityRuleAttrTypes}},
	"recommended_tls_version":    types.ListType{ElemType: types.ObjectType{AttrTypes: sslMonitorBaselineTLSRuleAttrTypes}},
	"recommended_key_size_bits":  types.ListType{ElemType: types.ObjectType{AttrTypes: sslMonitorBaselineKeySizeRuleAttrTypes}},
	"ocsp_must_staple_respected": types.ListType{ElemType: types.ObjectType{AttrTypes: sslMonitorBaselineSeverityRuleAttrTypes}},
	"sct_present":                types.ListType{ElemType: types.ObjectType{AttrTypes: sslMonitorBaselineSeverityRuleAttrTypes}},
}

var sslMonitorClientCertificateAttrTypes = map[string]attr.Type{
	"mode":                  types.StringType,
	"client_certificate_id": types.StringType,
}

var sslMonitorAssertionAttrTypes = map[string]attr.Type{
	"source":     types.StringType,
	"property":   types.StringType,
	"comparison": types.StringType,
	"target":     types.StringType,
}

var sslMonitorRequestAttrTypes = map[string]attr.Type{
	"hostname":                 types.StringType,
	"port":                     types.Int64Type,
	"server_name":              types.StringType,
	"ip_family":                types.StringType,
	"skip_chain_validation":    types.BoolType,
	"handshake_timeout_ms":     types.Int64Type,
	"alert_days_before_expiry": types.Int64Type,
	"security_baseline":        types.ListType{ElemType: types.ObjectType{AttrTypes: sslMonitorSecurityBaselineAttrTypes}},
	"client_certificate":       types.ObjectType{AttrTypes: sslMonitorClientCertificateAttrTypes},
	"assertion":                types.SetType{ElemType: types.ObjectType{AttrTypes: sslMonitorAssertionAttrTypes}},
}

// sslMonitorBaselineSeverityAttribute returns the severity attribute for a
// security baseline rule. The default mirrors the server's: enforceable rules
// default to "fail", advisory rules to "ignore". Because every leaf carries a
// default, a rule block that is present in the configuration always produces
// a fully-populated payload, and removing a leaf from the configuration falls
// back to the default with a visible plan diff.
func sslMonitorBaselineSeverityAttribute(defaultSeverity string) schema.StringAttribute {
	return schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Default:     stringdefault.StaticString(defaultSeverity),
		Description: fmt.Sprintf("What happens when the rule is violated: `fail` fails the monitor, `degrade` marks it degraded, `ignore` disables the rule. (Default `%s`).", defaultSeverity),
		Validators: []validator.String{
			stringvalidator.OneOf("fail", "degrade", "ignore"),
		},
	}
}

func sslMonitorBaselineTLSRuleBlock(defaultValue, defaultSeverity, description string) schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: description,
		Validators: []validator.List{
			listvalidator.SizeAtMost(1),
		},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"value": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Default:     stringdefault.StaticString(defaultValue),
					Description: fmt.Sprintf("The TLS version. Possible values are `TLS1.2` and `TLS1.3`. (Default `%s`).", defaultValue),
					Validators: []validator.String{
						stringvalidator.OneOf("TLS1.2", "TLS1.3"),
					},
				},
				"severity": sslMonitorBaselineSeverityAttribute(defaultSeverity),
			},
		},
	}
}

func sslMonitorBaselineKeySizeRuleBlock(defaultValue int64, defaultSeverity, description string) schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: description,
		Validators: []validator.List{
			listvalidator.SizeAtMost(1),
		},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"value": schema.Int64Attribute{
					Optional:    true,
					Computed:    true,
					Default:     int64default.StaticInt64(defaultValue),
					Description: fmt.Sprintf("The key size in bits. Possible values are between 1024 and 16384. (Default `%d`).", defaultValue),
					Validators: []validator.Int64{
						int64validator.Between(1024, 16384),
					},
				},
				"severity": sslMonitorBaselineSeverityAttribute(defaultSeverity),
			},
		},
	}
}

func sslMonitorBaselineSeverityRuleBlock(defaultSeverity, description string) schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: description,
		Validators: []validator.List{
			listvalidator.SizeAtMost(1),
		},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"severity": sslMonitorBaselineSeverityAttribute(defaultSeverity),
			},
		},
	}
}

func (r *SSLMonitorResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssl_monitor"
}

func (r *SSLMonitorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "SSL monitors check the TLS certificate of a host and alert before it expires or violates a security baseline.",
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
				Description: "The handshake time in milliseconds above which the monitor is considered degraded. Possible values are between 0 and 30000. (Default `10000`).",
				Validators: []validator.Int64{
					int64validator.Between(0, 30000),
				},
			},
			"max_response_time": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(20000),
				Description: "The handshake time in milliseconds above which the monitor is considered failing. Must be greater than or equal to `degraded_response_time`. Possible values are between 0 and 30000. (Default `20000`).",
				Validators: []validator.Int64{
					int64validator.Between(0, 30000),
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
				Description: "The parameters for the SSL connection.",
				Validators: []validator.List{
					listvalidator.IsRequired(),
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"hostname": schema.StringAttribute{
							Required:    true,
							Description: "The hostname to connect to and validate the TLS certificate of. Do not include a scheme or a port in this value.",
						},
						"port": schema.Int64Attribute{
							Optional:    true,
							Computed:    true,
							Default:     int64default.StaticInt64(443),
							Description: "The port number to connect to. Possible values are between 1 and 65535. (Default `443`).",
							Validators: []validator.Int64{
								int64validator.Between(1, 65535),
							},
						},
						"server_name": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString(""),
							Description: "An optional SNI server name to send in the TLS handshake. Defaults to `hostname` when unset.",
						},
						"ip_family": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("IPv4"),
							Description: "The IP family to use when executing the check. The value can be either `IPv4` or `IPv6`. (Default `IPv4`).",
							Validators: []validator.String{
								stringvalidator.OneOf("IPv4", "IPv6"),
							},
						},
						"skip_chain_validation": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
							Description: "When true, the certificate chain is not validated against trusted roots (the certificate is still inspected for expiry and the security baseline). (Default `false`).",
						},
						"handshake_timeout_ms": schema.Int64Attribute{
							Optional:    true,
							Computed:    true,
							Default:     int64default.StaticInt64(10000),
							Description: "The number of milliseconds to wait for the TLS handshake to complete before timing out. Possible values are between 1000 and 30000. (Default `10000`).",
							Validators: []validator.Int64{
								int64validator.Between(1000, 30000),
							},
						},
						"alert_days_before_expiry": schema.Int64Attribute{
							Optional:    true,
							Computed:    true,
							Default:     int64default.StaticInt64(20),
							Description: "Raise an alert when the certificate is within this many days of expiry. Possible values are between 1 and 365. (Default `20`).",
							Validators: []validator.Int64{
								int64validator.Between(1, 365),
							},
						},
						// The server always reports a client certificate mode,
						// so the attribute is computed. Its default is the
						// inherit mode, which makes removing the attribute
						// reset the monitor to the account default.
						"client_certificate": schema.SingleNestedAttribute{
							Optional:    true,
							Computed:    true,
							Description: "The mutual-TLS client certificate configuration.",
							Default: objectdefault.StaticValue(types.ObjectValueMust(sslMonitorClientCertificateAttrTypes, map[string]attr.Value{
								"mode":                  types.StringValue("account_default"),
								"client_certificate_id": types.StringNull(),
							})),
							Attributes: map[string]schema.Attribute{
								"mode": schema.StringAttribute{
									Optional:    true,
									Computed:    true,
									Default:     stringdefault.StaticString("account_default"),
									Description: "The mutual-TLS client-certificate mode. `account_default` inherits the account setting (no certificate sent), `auto` lets Checkly select a stored certificate, `explicit` uses the certificate referenced by `client_certificate_id`. (Default `account_default`).",
									Validators: []validator.String{
										stringvalidator.OneOf("account_default", "auto", "explicit"),
									},
								},
								"client_certificate_id": schema.StringAttribute{
									Optional:    true,
									Description: "The ID of the stored client certificate to present. Required when `mode = \"explicit\"`.",
								},
							},
						},
					},
					Blocks: map[string]schema.Block{
						"security_baseline": schema.ListNestedBlock{
							Description: "The SSL security baseline — a set of enforceable and advisory rules. Omit the block to inherit the account default baseline. Rules that are not listed keep their server defaults; removing a rule (or the whole block) resets it to its default on the next apply. Only listed rules are drift-checked: an external change to an unlisted rule is not shown by `terraform plan` and is reset on the next apply.",
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"enabled": schema.BoolAttribute{
										Optional:    true,
										Computed:    true,
										Default:     booldefault.StaticBool(true),
										Description: "Whether the security baseline is enforced. (Default `true`).",
									},
								},
								Blocks: map[string]schema.Block{
									"min_tls_version":            sslMonitorBaselineTLSRuleBlock("TLS1.2", "fail", "Enforceable rule: the minimum TLS version the server must accept."),
									"min_key_size_bits":          sslMonitorBaselineKeySizeRuleBlock(2048, "fail", "Enforceable rule: the minimum public key size in bits."),
									"weak_signature_algorithm":   sslMonitorBaselineSeverityRuleBlock("fail", "Enforceable rule: the certificate must not use a weak signature algorithm."),
									"weak_cipher_suite":          sslMonitorBaselineSeverityRuleBlock("fail", "Enforceable rule: the connection must not negotiate a weak cipher suite."),
									"known_bad_ca":               sslMonitorBaselineSeverityRuleBlock("fail", "Enforceable rule: the certificate chain must not include a known-bad CA."),
									"recommended_tls_version":    sslMonitorBaselineTLSRuleBlock("TLS1.3", "ignore", "Advisory rule: the recommended TLS version."),
									"recommended_key_size_bits":  sslMonitorBaselineKeySizeRuleBlock(3072, "ignore", "Advisory rule: the recommended public key size in bits."),
									"ocsp_must_staple_respected": sslMonitorBaselineSeverityRuleBlock("ignore", "Advisory rule: an OCSP Must-Staple extension, when present, must be respected."),
									"sct_present":                sslMonitorBaselineSeverityRuleBlock("ignore", "Advisory rule: the certificate should carry a Signed Certificate Timestamp."),
								},
							},
						},
						// The full assertion grammar (per-source property
						// lists and the property-dependent comparison sets)
						// is documented in the "Assertion Reference" section
						// of templates/resources/ssl_monitor.md.tmpl. When
						// assertion sources, properties, or their rules
						// change here or in the API, update that template
						// too — it is not generated from this schema.
						"assertion": schema.SetNestedBlock{
							Description: "A request can have multiple assertions. The allowed comparisons, properties, and target formats depend on the assertion source — see the [Assertion Reference](#assertion-reference) below.",
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
									"source": schema.StringAttribute{
										Required:    true,
										Description: "The source of the asserted value. Possible values are `CERTIFICATE`, `CONNECTION`, `RESPONSE_TIME`, `JSON_RESPONSE`, and `TEXT_RESPONSE`.",
										Validators: []validator.String{
											stringvalidator.OneOf("CERTIFICATE", "CONNECTION", "RESPONSE_TIME", "JSON_RESPONSE", "TEXT_RESPONSE"),
										},
									},
									"property": schema.StringAttribute{
										Optional:    true,
										Computed:    true,
										Description: "The property selecting the asserted value within the source. For `CERTIFICATE`: `daysUntilExpiry`, `keySizeBits`, `subjectCN`, `issuerCN`, `serialNumber`, `fingerprintSha256`, `issuerFingerprintSha256`, `keyAlgorithm`, `signatureAlgorithm`, `sans`, `selfSigned`, or `isCA`. For `CONNECTION`: `tlsVersion`, `cipherSuite`, `hostnameVerified`, `chainTrusted`, `ocspStapled`, `ocspStatus`, or `resolvedIp`. For `JSON_RESPONSE`: a JSONPath expression. For `TEXT_RESPONSE`: a regular expression applied to the serialized response.",
									},
									"comparison": schema.StringAttribute{
										Required:    true,
										Description: "The type of comparison to be executed between expected and actual value of the assertion. Possible values are `EQUALS`, `NOT_EQUALS`, `IS_EMPTY`, `NOT_EMPTY`, `GREATER_THAN`, `LESS_THAN`, `CONTAINS`, `NOT_CONTAINS`, `IS_NULL`, and `NOT_NULL`. The allowed set depends on the asserted `source` and `property`; for example, boolean properties such as `chainTrusted` only allow `EQUALS`.",
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

func (r *SSLMonitorResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		newFrequencyOffsetConfigValidator(),
	}
}

func (r *SSLMonitorResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

// ModifyPlan applies default values to the optional+computed attributes
// nested in the request.assertion set block. Schema defaults cannot be used
// for these attributes (see the schema for details), so the plan is adjusted
// here instead. Substituting static values keeps every set element fully
// known and stable across plans, which set element correlation requires. The
// request block and the security baseline blocks are lists, so their own
// attributes can keep using schema defaults.
func (r *SSLMonitorResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
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

	request = sslMonitorRequestWithDefaults(ctx, request, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("request"), request)...)
}

// sslMonitorRequestWithDefaults returns the request block value with default
// values substituted for null and unknown optional+computed attributes nested
// in the assertion set block.
func sslMonitorRequestWithDefaults(ctx context.Context, list types.List, diags *diag.Diagnostics) types.List {
	if list.IsNull() || list.IsUnknown() {
		return list
	}

	var models []SSLMonitorRequestModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return list
	}

	for i, model := range models {
		models[i].Assertion = sslMonitorAssertionsWithDefaults(ctx, model.Assertion, diags)
	}

	result, d := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: sslMonitorRequestAttrTypes}, models)
	diags.Append(d...)

	return result
}

// sslMonitorAssertionsWithDefaults returns the request.assertion block value
// with default values substituted for null and unknown optional+computed
// attributes.
func sslMonitorAssertionsWithDefaults(ctx context.Context, set types.Set, diags *diag.Diagnostics) types.Set {
	if set.IsNull() || set.IsUnknown() {
		return set
	}

	var models []SSLMonitorAssertionModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return set
	}

	for i, model := range models {
		models[i].Property = stringOrDefault(model.Property, "")
		models[i].Target = stringOrDefault(model.Target, "")
	}

	result, d := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: sslMonitorAssertionAttrTypes}, models)
	diags.Append(d...)

	return result
}

func (r *SSLMonitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SSLMonitorResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	monitor := sslMonitorFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.CreateSSLMonitor(apiCtx, monitor)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create SSL Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	// State is written even when the conversion produces error diagnostics.
	// The monitor already exists remotely at this point, and returning
	// without saving state would leave it unmanaged.
	plan.ID = types.StringValue(result.ID)
	sslMonitorToModel(ctx, result, &plan, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SSLMonitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SSLMonitorResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	monitor, err := r.client.GetSSLMonitor(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Read SSL Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	sslMonitorToModel(ctx, monitor, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SSLMonitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SSLMonitorResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	monitor := sslMonitorFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.UpdateSSLMonitor(apiCtx, plan.ID.ValueString(), monitor)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update SSL Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	sslMonitorToModel(ctx, result, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SSLMonitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SSLMonitorResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	err := r.client.DeleteSSLMonitor(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Delete SSL Monitor",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}
}

func (r *SSLMonitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// sslMonitorFromModel converts the resource model to its API representation.
func sslMonitorFromModel(ctx context.Context, model *SSLMonitorResourceModel, diags *diag.Diagnostics) checkly.SSLMonitor {
	// The locations property is serialized even when empty.
	locations := stringsFromSet(ctx, model.Locations, diags)
	if locations == nil {
		locations = []string{}
	}

	privateLocations := stringsFromSet(ctx, model.PrivateLocations, diags)
	if privateLocations == nil {
		privateLocations = []string{}
	}

	monitor := checkly.SSLMonitor{
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
		Request:                   sslMonitorRequestFromList(ctx, model.Request, diags),
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

// sslMonitorToModel updates the model to match the API representation of the
// monitor. The model must hold prior values (the plan on create/update, the
// state on read); they are preserved for attributes that the API does not
// track or echo faithfully.
func sslMonitorToModel(ctx context.Context, c *checkly.SSLMonitor, model *SSLMonitorResourceModel, diags *diag.Diagnostics) {
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
	model.Locations = sslMonitorStringSet(ctx, c.Locations, model.Locations, diags)
	model.DegradedResponseTime = types.Int64Value(int64(c.DegradedResponseTime))
	model.MaxResponseTime = types.Int64Value(int64(c.MaxResponseTime))
	model.Tags = sslMonitorStringSet(ctx, c.Tags, model.Tags, diags)

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
	model.PrivateLocations = sslMonitorStringSet(ctx, privateLocations, model.PrivateLocations, diags)

	if c.AlertSettings != nil {
		model.AlertSettings = alertSettingsToObject(ctx, *c.AlertSettings, model.AlertSettings, diags)
	} else if model.AlertSettings.IsUnknown() {
		model.AlertSettings = types.ObjectNull(alertSettingsAttrTypes)
	}

	model.UseGlobalAlertSettings = types.BoolValue(c.UseGlobalAlertSettings)
	model.GroupID = types.Int64Value(c.GroupID)
	model.GroupOrder = types.Int64Value(int64(c.GroupOrder))
	model.RetryStrategy = retryStrategyToObject(ctx, c.RetryStrategy, model.RetryStrategy, diags)
	model.Request = sslMonitorRequestToList(ctx, c.Request, model.Request, diags)
	model.AlertChannelSubscription = alertChannelSubscriptionsToSet(ctx, c.AlertChannelSubscriptions, diags)
	model.TriggerIncident = triggerIncidentToSet(ctx, c.TriggerIncident, diags)
}

// sslMonitorStringSet converts a string slice from the API to a set value. An
// empty slice resolves to null when the prior value is null, so that an
// optional set that is absent from the configuration stays null in state.
func sslMonitorStringSet(ctx context.Context, values []string, prior types.Set, diags *diag.Diagnostics) types.Set {
	if len(values) == 0 && (prior.IsNull() || prior.IsUnknown()) {
		return types.SetNull(types.StringType)
	}

	return setFromStrings(ctx, values, diags)
}

// sslMonitorRequestFromList converts the request block value to its API
// representation.
func sslMonitorRequestFromList(ctx context.Context, list types.List, diags *diag.Diagnostics) checkly.SSLRequest {
	if list.IsNull() || list.IsUnknown() {
		return checkly.SSLRequest{}
	}

	var models []SSLMonitorRequestModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	if diags.HasError() || len(models) == 0 {
		return checkly.SSLRequest{}
	}

	model := models[0]

	config := checkly.SSLConfig{
		Hostname:              model.Hostname.ValueString(),
		Port:                  int(model.Port.ValueInt64()),
		IPFamily:              model.IPFamily.ValueString(),
		SkipChainValidation:   model.SkipChainValidation.ValueBool(),
		HandshakeTimeoutMs:    int(model.HandshakeTimeoutMs.ValueInt64()),
		AlertDaysBeforeExpiry: int(model.AlertDaysBeforeExpiry.ValueInt64()),
		// A nil baseline (block omitted) makes the server apply its default
		// baseline. The API replaces the stored baseline wholesale on every
		// write (rules missing from the payload are reset to their
		// defaults), so the payload carries exactly the rules present in the
		// configuration; no state values are mixed in.
		SecurityBaseline: sslMonitorSecurityBaselineFromList(ctx, model.SecurityBaseline, diags),
	}

	// serverName is nullable in the API; send a pointer only when set.
	if serverName := model.ServerName.ValueString(); serverName != "" {
		config.ServerName = &serverName
	}

	request := checkly.SSLRequest{
		SSLConfig:  config,
		Assertions: sslMonitorAssertionsFromSet(ctx, model.Assertion, diags),
	}

	// client_certificate.mode maps to sslConfig.clientCertificateMode
	// (auto|explicit); "account_default" is the inherit case and is sent as
	// an empty (omitted) mode. The certificate ID maps to the top-level
	// sslClientCertificateId foreign key and is required when the mode is
	// "explicit".
	if !model.ClientCertificate.IsNull() && !model.ClientCertificate.IsUnknown() {
		var cert SSLMonitorClientCertificateModel
		diags.Append(model.ClientCertificate.As(ctx, &cert, basetypes.ObjectAsOptions{})...)

		if mode := cert.Mode.ValueString(); mode != "" && mode != "account_default" {
			request.SSLConfig.ClientCertificateMode = mode
		}

		if certID := cert.ClientCertificateID.ValueString(); certID != "" {
			request.SSLClientCertificateId = &certID
		}
	}

	return request
}

// sslMonitorRequestToList converts the API representation of an SSL request
// to the request block value. prior is the request block value held before
// the conversion (the plan on create/update, the state on read); it drives
// the security baseline projection (see sslMonitorSecurityBaselineToList)
// and preserves a configured empty client certificate ID.
func sslMonitorRequestToList(ctx context.Context, r checkly.SSLRequest, prior types.List, diags *diag.Diagnostics) types.List {
	priorRequest := sslMonitorPriorRequest(ctx, prior, diags)

	config := r.SSLConfig

	serverName := ""
	if config.ServerName != nil {
		serverName = *config.ServerName
	}

	list, d := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: sslMonitorRequestAttrTypes}, []SSLMonitorRequestModel{
		{
			Hostname:              types.StringValue(config.Hostname),
			Port:                  types.Int64Value(int64(config.Port)),
			ServerName:            types.StringValue(serverName),
			IPFamily:              types.StringValue(config.IPFamily),
			SkipChainValidation:   types.BoolValue(config.SkipChainValidation),
			HandshakeTimeoutMs:    types.Int64Value(int64(config.HandshakeTimeoutMs)),
			AlertDaysBeforeExpiry: types.Int64Value(int64(config.AlertDaysBeforeExpiry)),
			SecurityBaseline:      sslMonitorSecurityBaselineToList(ctx, config.SecurityBaseline, priorRequest.SecurityBaseline, diags),
			ClientCertificate:     sslMonitorClientCertificateToObject(ctx, r, priorRequest.ClientCertificate, diags),
			Assertion:             sslMonitorAssertionsToSet(ctx, r.Assertions, diags),
		},
	})
	diags.Append(d...)

	return list
}

// sslMonitorPriorRequest returns the single element of a prior request block
// value. Without a prior element (e.g. on import) the nested values are null.
func sslMonitorPriorRequest(ctx context.Context, prior types.List, diags *diag.Diagnostics) SSLMonitorRequestModel {
	empty := SSLMonitorRequestModel{
		SecurityBaseline:  types.ListNull(types.ObjectType{AttrTypes: sslMonitorSecurityBaselineAttrTypes}),
		ClientCertificate: types.ObjectNull(sslMonitorClientCertificateAttrTypes),
	}

	if prior.IsNull() || prior.IsUnknown() {
		return empty
	}

	var models []SSLMonitorRequestModel
	diags.Append(prior.ElementsAs(ctx, &models, false)...)
	if diags.HasError() || len(models) == 0 {
		return empty
	}

	return models[0]
}

// sslMonitorClientCertificateToObject converts the API representation of the
// client certificate configuration to the request.client_certificate value.
// The value is always set so that a configuration spelling out the inherit
// mode re-plans clean; an empty API mode maps back to "account_default".
func sslMonitorClientCertificateToObject(ctx context.Context, r checkly.SSLRequest, prior types.Object, diags *diag.Diagnostics) types.Object {
	mode := r.SSLConfig.ClientCertificateMode
	if mode == "" {
		mode = "account_default"
	}

	certID := types.StringNull()
	if r.SSLClientCertificateId != nil {
		certID = types.StringValue(*r.SSLClientCertificateId)
	} else if !prior.IsNull() && !prior.IsUnknown() {
		// A configured empty ID converts to the same API representation as
		// no ID at all, so it is preserved.
		var priorCert SSLMonitorClientCertificateModel
		diags.Append(prior.As(ctx, &priorCert, basetypes.ObjectAsOptions{})...)
		if !priorCert.ClientCertificateID.IsUnknown() && !priorCert.ClientCertificateID.IsNull() && priorCert.ClientCertificateID.ValueString() == "" {
			certID = types.StringValue("")
		}
	}

	obj, d := types.ObjectValue(sslMonitorClientCertificateAttrTypes, map[string]attr.Value{
		"mode":                  types.StringValue(mode),
		"client_certificate_id": certID,
	})
	diags.Append(d...)

	return obj
}

// sslMonitorSecurityBaselineFromList converts the request.security_baseline
// block value to its API representation. An absent block converts to nil,
// which omits securityBaseline from the payload so the server applies (or
// resets to) its default baseline. Rules present in the configuration always
// carry both leaves, as the schema defaults fill whatever the configuration
// leaves out.
func sslMonitorSecurityBaselineFromList(ctx context.Context, list types.List, diags *diag.Diagnostics) *checkly.SecurityBaseline {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}

	var models []SSLMonitorSecurityBaselineModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	if diags.HasError() || len(models) == 0 {
		return nil
	}

	model := models[0]

	enabled := true
	if !model.Enabled.IsNull() && !model.Enabled.IsUnknown() {
		enabled = model.Enabled.ValueBool()
	}

	return &checkly.SecurityBaseline{
		Enabled:                 &enabled,
		MinTLSVersion:           sslMonitorTLSRuleFromList(ctx, model.MinTLSVersion, diags),
		MinKeySizeBits:          sslMonitorKeySizeRuleFromList(ctx, model.MinKeySizeBits, diags),
		WeakSignatureAlgorithm:  sslMonitorSeverityRuleFromList(ctx, model.WeakSignatureAlgorithm, diags),
		WeakCipherSuite:         sslMonitorSeverityRuleFromList(ctx, model.WeakCipherSuite, diags),
		KnownBadCA:              sslMonitorSeverityRuleFromList(ctx, model.KnownBadCA, diags),
		RecommendedTLSVersion:   sslMonitorTLSRuleFromList(ctx, model.RecommendedTLSVersion, diags),
		RecommendedKeySizeBits:  sslMonitorKeySizeRuleFromList(ctx, model.RecommendedKeySizeBits, diags),
		OCSPMustStapleRespected: sslMonitorSeverityRuleFromList(ctx, model.OCSPMustStapleRespected, diags),
		SCTPresent:              sslMonitorSeverityRuleFromList(ctx, model.SCTPresent, diags),
	}
}

func sslMonitorTLSRuleFromList(ctx context.Context, list types.List, diags *diag.Diagnostics) *checkly.SSLBaselineTLSRule {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}

	var models []SSLMonitorBaselineTLSRuleModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	if diags.HasError() || len(models) == 0 {
		return nil
	}

	return &checkly.SSLBaselineTLSRule{
		Value:    models[0].Value.ValueString(),
		Severity: models[0].Severity.ValueString(),
	}
}

func sslMonitorKeySizeRuleFromList(ctx context.Context, list types.List, diags *diag.Diagnostics) *checkly.SSLBaselineKeySizeRule {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}

	var models []SSLMonitorBaselineKeySizeRuleModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	if diags.HasError() || len(models) == 0 {
		return nil
	}

	return &checkly.SSLBaselineKeySizeRule{
		Value:    int(models[0].Value.ValueInt64()),
		Severity: models[0].Severity.ValueString(),
	}
}

func sslMonitorSeverityRuleFromList(ctx context.Context, list types.List, diags *diag.Diagnostics) *checkly.SSLBaselineSeverityRule {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}

	var models []SSLMonitorBaselineSeverityRuleModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	if diags.HasError() || len(models) == 0 {
		return nil
	}

	return &checkly.SSLBaselineSeverityRule{
		Severity: models[0].Severity.ValueString(),
	}
}

// sslMonitorSecurityBaselineToList projects the server-returned security
// baseline onto the shape of the prior request.security_baseline value
// (which mirrors the configuration as of the last apply). The server
// normalizes every baseline to the full rule set, so writing the raw response
// into state would make any configuration that omits a rule diff forever.
// Instead, only rules present in the prior value are written back, with the
// server's values: drift on configured rules stays visible, while
// server-filled defaults for unconfigured rules stay out of state. Without a
// prior baseline (never configured, or a fresh import) nothing is written;
// after an import the first plan therefore shows the configured baseline as
// an addition.
func sslMonitorSecurityBaselineToList(ctx context.Context, remote *checkly.SecurityBaseline, prior types.List, diags *diag.Diagnostics) types.List {
	elementType := types.ObjectType{AttrTypes: sslMonitorSecurityBaselineAttrTypes}
	empty := types.ListValueMust(elementType, []attr.Value{})

	if remote == nil || prior.IsNull() || prior.IsUnknown() {
		return empty
	}

	var priorModels []SSLMonitorSecurityBaselineModel
	diags.Append(prior.ElementsAs(ctx, &priorModels, false)...)
	if diags.HasError() || len(priorModels) == 0 {
		return empty
	}

	priorModel := priorModels[0]

	model := SSLMonitorSecurityBaselineModel{
		Enabled:                 types.BoolValue(remote.Enabled == nil || *remote.Enabled),
		MinTLSVersion:           sslMonitorTLSRuleToList(remote.MinTLSVersion, priorModel.MinTLSVersion),
		MinKeySizeBits:          sslMonitorKeySizeRuleToList(remote.MinKeySizeBits, priorModel.MinKeySizeBits),
		WeakSignatureAlgorithm:  sslMonitorSeverityRuleToList(remote.WeakSignatureAlgorithm, priorModel.WeakSignatureAlgorithm),
		WeakCipherSuite:         sslMonitorSeverityRuleToList(remote.WeakCipherSuite, priorModel.WeakCipherSuite),
		KnownBadCA:              sslMonitorSeverityRuleToList(remote.KnownBadCA, priorModel.KnownBadCA),
		RecommendedTLSVersion:   sslMonitorTLSRuleToList(remote.RecommendedTLSVersion, priorModel.RecommendedTLSVersion),
		RecommendedKeySizeBits:  sslMonitorKeySizeRuleToList(remote.RecommendedKeySizeBits, priorModel.RecommendedKeySizeBits),
		OCSPMustStapleRespected: sslMonitorSeverityRuleToList(remote.OCSPMustStapleRespected, priorModel.OCSPMustStapleRespected),
		SCTPresent:              sslMonitorSeverityRuleToList(remote.SCTPresent, priorModel.SCTPresent),
	}

	list, d := types.ListValueFrom(ctx, elementType, []SSLMonitorSecurityBaselineModel{model})
	diags.Append(d...)

	return list
}

// sslMonitorPriorHasRule reports whether the prior value of a security
// baseline rule block holds an element.
func sslMonitorPriorHasRule(prior types.List) bool {
	return !prior.IsNull() && !prior.IsUnknown() && len(prior.Elements()) > 0
}

func sslMonitorTLSRuleToList(remote *checkly.SSLBaselineTLSRule, prior types.List) types.List {
	elementType := types.ObjectType{AttrTypes: sslMonitorBaselineTLSRuleAttrTypes}

	if remote == nil || !sslMonitorPriorHasRule(prior) {
		return types.ListValueMust(elementType, []attr.Value{})
	}

	return types.ListValueMust(elementType, []attr.Value{
		types.ObjectValueMust(sslMonitorBaselineTLSRuleAttrTypes, map[string]attr.Value{
			"value":    types.StringValue(remote.Value),
			"severity": types.StringValue(remote.Severity),
		}),
	})
}

func sslMonitorKeySizeRuleToList(remote *checkly.SSLBaselineKeySizeRule, prior types.List) types.List {
	elementType := types.ObjectType{AttrTypes: sslMonitorBaselineKeySizeRuleAttrTypes}

	if remote == nil || !sslMonitorPriorHasRule(prior) {
		return types.ListValueMust(elementType, []attr.Value{})
	}

	return types.ListValueMust(elementType, []attr.Value{
		types.ObjectValueMust(sslMonitorBaselineKeySizeRuleAttrTypes, map[string]attr.Value{
			"value":    types.Int64Value(int64(remote.Value)),
			"severity": types.StringValue(remote.Severity),
		}),
	})
}

func sslMonitorSeverityRuleToList(remote *checkly.SSLBaselineSeverityRule, prior types.List) types.List {
	elementType := types.ObjectType{AttrTypes: sslMonitorBaselineSeverityRuleAttrTypes}

	if remote == nil || !sslMonitorPriorHasRule(prior) {
		return types.ListValueMust(elementType, []attr.Value{})
	}

	return types.ListValueMust(elementType, []attr.Value{
		types.ObjectValueMust(sslMonitorBaselineSeverityRuleAttrTypes, map[string]attr.Value{
			"severity": types.StringValue(remote.Severity),
		}),
	})
}

// sslMonitorAssertionsFromSet converts the request.assertion block value to
// its API representation.
func sslMonitorAssertionsFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) []checkly.Assertion {
	assertions := []checkly.Assertion{}

	if set.IsNull() || set.IsUnknown() {
		return assertions
	}

	var models []SSLMonitorAssertionModel
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

// sslMonitorAssertionsToSet converts the API representation of assertions to
// the request.assertion block value.
func sslMonitorAssertionsToSet(ctx context.Context, assertions []checkly.Assertion, diags *diag.Diagnostics) types.Set {
	models := make([]SSLMonitorAssertionModel, len(assertions))
	for i, assertion := range assertions {
		models[i] = SSLMonitorAssertionModel{
			Source:     types.StringValue(assertion.Source),
			Property:   types.StringValue(assertion.Property),
			Comparison: types.StringValue(assertion.Comparison),
			Target:     types.StringValue(assertion.Target),
		}
	}

	set, d := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: sslMonitorAssertionAttrTypes}, models)
	diags.Append(d...)

	return set
}
