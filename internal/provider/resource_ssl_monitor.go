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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/checkly/checkly-go-sdk/v2"
)

var (
	_ resource.Resource                     = (*SSLMonitorResource)(nil)
	_ resource.ResourceWithConfigure        = (*SSLMonitorResource)(nil)
	_ resource.ResourceWithUpgradeState     = (*SSLMonitorResource)(nil)
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

// SSLMonitorRequestModel describes the request attribute data model.
type SSLMonitorRequestModel struct {
	Hostname              types.String `tfsdk:"hostname"`
	Port                  types.Int64  `tfsdk:"port"`
	ServerName            types.String `tfsdk:"server_name"`
	IPFamily              types.String `tfsdk:"ip_family"`
	SkipChainValidation   types.Bool   `tfsdk:"skip_chain_validation"`
	HandshakeTimeoutMs    types.Int64  `tfsdk:"handshake_timeout_ms"`
	AlertDaysBeforeExpiry types.Int64  `tfsdk:"alert_days_before_expiry"`
	SecurityBaseline      types.Object `tfsdk:"security_baseline"`
	ClientCertificate     types.Object `tfsdk:"client_certificate"`
	Assertions            types.Set    `tfsdk:"assertions"`
}

// SSLMonitorSecurityBaselineModel describes the request.security_baseline
// attribute data model. Each rule is an optional single nested attribute.
type SSLMonitorSecurityBaselineModel struct {
	Enabled                 types.Bool   `tfsdk:"enabled"`
	MinTLSVersion           types.Object `tfsdk:"min_tls_version"`
	MinKeySizeBits          types.Object `tfsdk:"min_key_size_bits"`
	WeakSignatureAlgorithm  types.Object `tfsdk:"weak_signature_algorithm"`
	WeakCipherSuite         types.Object `tfsdk:"weak_cipher_suite"`
	KnownBadCA              types.Object `tfsdk:"known_bad_ca"`
	RecommendedTLSVersion   types.Object `tfsdk:"recommended_tls_version"`
	RecommendedKeySizeBits  types.Object `tfsdk:"recommended_key_size_bits"`
	OCSPMustStapleRespected types.Object `tfsdk:"ocsp_must_staple_respected"`
	SCTPresent              types.Object `tfsdk:"sct_present"`
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
	"min_tls_version":            types.ObjectType{AttrTypes: sslMonitorBaselineTLSRuleAttrTypes},
	"min_key_size_bits":          types.ObjectType{AttrTypes: sslMonitorBaselineKeySizeRuleAttrTypes},
	"weak_signature_algorithm":   types.ObjectType{AttrTypes: sslMonitorBaselineSeverityRuleAttrTypes},
	"weak_cipher_suite":          types.ObjectType{AttrTypes: sslMonitorBaselineSeverityRuleAttrTypes},
	"known_bad_ca":               types.ObjectType{AttrTypes: sslMonitorBaselineSeverityRuleAttrTypes},
	"recommended_tls_version":    types.ObjectType{AttrTypes: sslMonitorBaselineTLSRuleAttrTypes},
	"recommended_key_size_bits":  types.ObjectType{AttrTypes: sslMonitorBaselineKeySizeRuleAttrTypes},
	"ocsp_must_staple_respected": types.ObjectType{AttrTypes: sslMonitorBaselineSeverityRuleAttrTypes},
	"sct_present":                types.ObjectType{AttrTypes: sslMonitorBaselineSeverityRuleAttrTypes},
}

var sslMonitorClientCertificateAttrTypes = map[string]attr.Type{
	"mode":                  types.StringType,
	"client_certificate_id": types.StringType,
}

var sslMonitorRequestAttrTypes = map[string]attr.Type{
	"hostname":                 types.StringType,
	"port":                     types.Int64Type,
	"server_name":              types.StringType,
	"ip_family":                types.StringType,
	"skip_chain_validation":    types.BoolType,
	"handshake_timeout_ms":     types.Int64Type,
	"alert_days_before_expiry": types.Int64Type,
	"security_baseline":        types.ObjectType{AttrTypes: sslMonitorSecurityBaselineAttrTypes},
	"client_certificate":       types.ObjectType{AttrTypes: sslMonitorClientCertificateAttrTypes},
	"assertions":               types.SetType{ElemType: types.ObjectType{AttrTypes: assertionAttrTypes}},
}

// sslMonitorDefaultSecurityBaseline is the security baseline a monitor gets
// for the rules its configuration leaves out. The schema defaults of the
// baseline, of each rule and of each rule's leaves are all derived from this
// value, and the provider sends the complete baseline on every write.
//
// The values must follow the Checkly API's default security baseline, so
// that a monitor whose configuration omits a rule plans no changes against
// what the API stores for it. Changing a value here plans an update for
// every monitor that leaves that rule unconfigured.
var sslMonitorDefaultSecurityBaseline = checkly.SecurityBaseline{
	Enabled:                 new(true),
	MinTLSVersion:           &checkly.SSLBaselineTLSRule{Value: checkly.TLSVersion12, Severity: checkly.SSLBaselineSeverityFail},
	MinKeySizeBits:          &checkly.SSLBaselineKeySizeRule{Value: 2048, Severity: checkly.SSLBaselineSeverityFail},
	WeakSignatureAlgorithm:  &checkly.SSLBaselineSeverityRule{Severity: checkly.SSLBaselineSeverityFail},
	WeakCipherSuite:         &checkly.SSLBaselineSeverityRule{Severity: checkly.SSLBaselineSeverityFail},
	KnownBadCA:              &checkly.SSLBaselineSeverityRule{Severity: checkly.SSLBaselineSeverityFail},
	RecommendedTLSVersion:   &checkly.SSLBaselineTLSRule{Value: checkly.TLSVersion13, Severity: checkly.SSLBaselineSeverityIgnore},
	RecommendedKeySizeBits:  &checkly.SSLBaselineKeySizeRule{Value: 3072, Severity: checkly.SSLBaselineSeverityIgnore},
	OCSPMustStapleRespected: &checkly.SSLBaselineSeverityRule{Severity: checkly.SSLBaselineSeverityIgnore},
	SCTPresent:              &checkly.SSLBaselineSeverityRule{Severity: checkly.SSLBaselineSeverityIgnore},
}

// sslMonitorBaselineSeverityAttribute returns the severity attribute for a
// security baseline rule. The default mirrors the server's: enforceable rules
// default to "fail", advisory rules to "ignore". Because every leaf carries a
// default, a rule given with only some of its leaves still produces a
// fully-populated payload, and removing a leaf from the configuration falls
// back to the default with a visible plan diff.
func sslMonitorBaselineSeverityAttribute(defaultSeverity checkly.SSLBaselineSeverity) schema.StringAttribute {
	return schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Default:     stringdefault.StaticString(string(defaultSeverity)),
		Description: fmt.Sprintf("What happens when the rule is violated: `fail` fails the monitor, `degrade` marks it degraded, `ignore` disables the rule. (Default `%s`).", defaultSeverity),
		Validators: []validator.String{
			stringvalidator.OneOf("fail", "degrade", "ignore"),
		},
	}
}

// The rule attributes are computed with the rule's default from
// sslMonitorDefaultSecurityBaseline, so that a rule left out of the
// configuration (or removed from it) plans the default and state can hold
// the full baseline the server returns.

func sslMonitorBaselineTLSRuleAttribute(def *checkly.SSLBaselineTLSRule, description string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
		Computed:    true,
		Description: description,
		Default:     objectdefault.StaticValue(sslMonitorTLSRuleToObject(def, def)),
		Attributes: map[string]schema.Attribute{
			"value": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(string(def.Value)),
				Description: fmt.Sprintf("The TLS version. Possible values are `TLS1.2` and `TLS1.3`. (Default `%s`).", def.Value),
				Validators: []validator.String{
					stringvalidator.OneOf("TLS1.2", "TLS1.3"),
				},
			},
			"severity": sslMonitorBaselineSeverityAttribute(def.Severity),
		},
	}
}

func sslMonitorBaselineKeySizeRuleAttribute(def *checkly.SSLBaselineKeySizeRule, description string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
		Computed:    true,
		Description: description,
		Default:     objectdefault.StaticValue(sslMonitorKeySizeRuleToObject(def, def)),
		Attributes: map[string]schema.Attribute{
			"value": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(int64(def.Value)),
				Description: fmt.Sprintf("The key size in bits. Possible values are between 1024 and 16384. (Default `%d`).", def.Value),
				Validators: []validator.Int64{
					int64validator.Between(1024, 16384),
				},
			},
			"severity": sslMonitorBaselineSeverityAttribute(def.Severity),
		},
	}
}

func sslMonitorBaselineSeverityRuleAttribute(def *checkly.SSLBaselineSeverityRule, description string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
		Computed:    true,
		Description: description,
		Default:     objectdefault.StaticValue(sslMonitorSeverityRuleToObject(def, def)),
		Attributes: map[string]schema.Attribute{
			"severity": sslMonitorBaselineSeverityAttribute(def.Severity),
		},
	}
}

func (r *SSLMonitorResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssl_monitor"
}

func (r *SSLMonitorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     sdkv2SchemaVersion + 1,
		Description: "SSL monitors check the TLS certificate of a host and alert before it expires or violates a security baseline.",
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
			triggerIncidentAttributeName: makeTriggerIncidentSchema(),
			alertSettingsAttributeName: makeAlertSettingsAttributeSchema(AlertSettingsAttributeSchemaOptions{
				Subject: subjectMonitor,
			}),
			retryStrategyAttributeName: makeRetryStrategyAttributeSchema(RetryStrategyAttributeSchemaOptions{
				SupportsOnlyOnNetworkError: false,
			}),
			"request": schema.SingleNestedAttribute{
				Required:    true,
				Description: "The parameters for the SSL connection.",
				Attributes: map[string]schema.Attribute{
					"hostname": schema.StringAttribute{
						Required:    true,
						Description: "The hostname (1 to 2048 characters) to connect to and validate the TLS certificate of. Do not include a scheme or a port in this value.",
						Validators: []validator.String{
							utf16LengthBetween(1, 2048),
							noControlCharacters(),
						},
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
						Description: "An optional SNI server name to send in the TLS handshake (at most 2048 characters). Defaults to `hostname` when unset.",
						Validators: []validator.String{
							utf16LengthBetween(0, 2048),
							noControlCharacters(),
						},
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
					// The server stores a complete baseline on every monitor and
					// returns all rules, so the attribute is computed with the
					// full default baseline and state holds every rule.
					"security_baseline": schema.SingleNestedAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The SSL security baseline — a set of enforceable and advisory rules. Omit the attribute, or a rule, to use its default. Removing a rule (or the whole attribute) resets it to its default on the next apply.",
						Default:     objectdefault.StaticValue(sslMonitorSecurityBaselineToObject(ctx, nil, &resp.Diagnostics)),
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Default:     booldefault.StaticBool(*sslMonitorDefaultSecurityBaseline.Enabled),
								Description: "Whether the security baseline is enforced. (Default `true`).",
							},
							"min_tls_version":            sslMonitorBaselineTLSRuleAttribute(sslMonitorDefaultSecurityBaseline.MinTLSVersion, "Enforceable rule: the minimum TLS version the server must accept."),
							"min_key_size_bits":          sslMonitorBaselineKeySizeRuleAttribute(sslMonitorDefaultSecurityBaseline.MinKeySizeBits, "Enforceable rule: the minimum public key size in bits."),
							"weak_signature_algorithm":   sslMonitorBaselineSeverityRuleAttribute(sslMonitorDefaultSecurityBaseline.WeakSignatureAlgorithm, "Enforceable rule: the certificate must not use a weak signature algorithm."),
							"weak_cipher_suite":          sslMonitorBaselineSeverityRuleAttribute(sslMonitorDefaultSecurityBaseline.WeakCipherSuite, "Enforceable rule: the connection must not negotiate a weak cipher suite."),
							"known_bad_ca":               sslMonitorBaselineSeverityRuleAttribute(sslMonitorDefaultSecurityBaseline.KnownBadCA, "Enforceable rule: the certificate chain must not include a known-bad CA."),
							"recommended_tls_version":    sslMonitorBaselineTLSRuleAttribute(sslMonitorDefaultSecurityBaseline.RecommendedTLSVersion, "Advisory rule: the recommended TLS version."),
							"recommended_key_size_bits":  sslMonitorBaselineKeySizeRuleAttribute(sslMonitorDefaultSecurityBaseline.RecommendedKeySizeBits, "Advisory rule: the recommended public key size in bits."),
							"ocsp_must_staple_respected": sslMonitorBaselineSeverityRuleAttribute(sslMonitorDefaultSecurityBaseline.OCSPMustStapleRespected, "Advisory rule: an OCSP Must-Staple extension, when present, must be respected."),
							"sct_present":                sslMonitorBaselineSeverityRuleAttribute(sslMonitorDefaultSecurityBaseline.SCTPresent, "Advisory rule: the certificate should carry a Signed Certificate Timestamp."),
						},
					},
					// The full assertion grammar (per-source property
					// lists and the property-dependent comparison sets)
					// is documented in the "Assertion Reference" section
					// of templates/resources/ssl_monitor.md.tmpl. When
					// assertion sources, properties, or their rules
					// change here or in the API, update that template
					// too — it is not generated from this schema.
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
									Optional:    true,
									Computed:    true,
									Description: "The value to compare against, e.g. a number of days for the `daysUntilExpiry` property. Leave empty for comparisons that take no value, such as `IS_EMPTY` and `NOT_NULL`.",
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

func (r *SSLMonitorResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		newFrequencyOffsetConfigValidator(),
	}
}

// UpgradeState upgrades state written by the provider's SDK v2
// implementation; see sdkv2StateUpgraders.
func (r *SSLMonitorResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return sdkv2StateUpgraders(ctx, r)
}

func (r *SSLMonitorResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

// ModifyPlan applies default values to the optional+computed attributes of
// the request assertions. Schema defaults cannot be used for attributes
// nested in set elements (see the schema for details), so the plan is
// adjusted here instead. Substituting static values keeps every set element
// fully known and stable across plans, which set element correlation
// requires. The request, its security baseline and the baseline rules are
// single nested attributes, so their own attributes use schema defaults.
func (r *SSLMonitorResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planNestedDefaults(ctx, req, resp, path.Root("request"), objectWithSetDefaults("assertions", assertionsWithPropertyAndTargetDefaults))
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
		addAPIError(&resp.Diagnostics, "Unable to Create SSL Monitor", err)
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
		addAPIError(&resp.Diagnostics, "Unable to Read SSL Monitor", err)
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
		addAPIError(&resp.Diagnostics, "Unable to Update SSL Monitor", err)
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
		addAPIError(&resp.Diagnostics, "Unable to Delete SSL Monitor", err)
		return
	}
}

func (r *SSLMonitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// sslMonitorFromModel converts the resource model to its API representation.
func sslMonitorFromModel(ctx context.Context, model *SSLMonitorResourceModel, diags *diag.Diagnostics) checkly.SSLMonitor {
	// The locations property is serialized even when empty.
	locations := stringsFromSetOrEmpty(ctx, model.Locations, diags)

	privateLocations := stringsFromSetOrEmpty(ctx, model.PrivateLocations, diags)

	monitor := checkly.SSLMonitor{
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
		Request:                   sslMonitorRequestFromObject(ctx, model.Request, diags),
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

	model.FrequencyOffset = frequencyOffsetFromAPI(c.Frequency, c.FrequencyOffset, model.FrequencyOffset)

	model.Activated = types.BoolValue(c.Activated)
	model.Muted = types.BoolValue(c.Muted)
	model.RunParallel = types.BoolValue(c.RunParallel)
	model.Locations = stringSetFromAPINullIfUnset(ctx, c.Locations, model.Locations, diags)
	model.DegradedResponseTime = types.Int64Value(int64(c.DegradedResponseTime))
	model.MaxResponseTime = types.Int64Value(int64(c.MaxResponseTime))
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

	model.AlertSettings = alertSettingsFromAPI(ctx, c.AlertSettings, model.AlertSettings, diags)

	model.UseGlobalAlertSettings = types.BoolValue(c.UseGlobalAlertSettings)
	model.GroupID = int64ValueOrNull(c.GroupID)
	model.GroupOrder = groupOrderFromAPI(c.GroupOrder, model.GroupOrder)
	model.RetryStrategy = retryStrategyToObject(ctx, c.RetryStrategy, model.RetryStrategy, diags)
	model.Request = sslMonitorRequestToObject(ctx, c.Request, model.Request, diags)
	model.AlertChannelSubscriptions = alertChannelSubscriptionsToSet(ctx, c.AlertChannelSubscriptions, model.AlertChannelSubscriptions, diags)
	model.TriggerIncident = triggerIncidentToObject(ctx, c.TriggerIncident, diags)
}

// sslMonitorRequestFromObject converts the request attribute value to its
// API representation.
func sslMonitorRequestFromObject(ctx context.Context, obj types.Object, diags *diag.Diagnostics) checkly.SSLRequest {
	model := objectAs[SSLMonitorRequestModel](ctx, obj, diags)
	if model == nil {
		return checkly.SSLRequest{}
	}

	config := checkly.SSLConfig{
		Hostname:              model.Hostname.ValueString(),
		Port:                  int(model.Port.ValueInt64()),
		IPFamily:              checkly.IPFamily(model.IPFamily.ValueString()),
		SkipChainValidation:   model.SkipChainValidation.ValueBool(),
		HandshakeTimeoutMs:    int(model.HandshakeTimeoutMs.ValueInt64()),
		AlertDaysBeforeExpiry: int(model.AlertDaysBeforeExpiry.ValueInt64()),
		// The schema defaults fill every rule the configuration leaves out,
		// so the payload always carries the complete baseline and the
		// server stores exactly what the plan shows.
		SecurityBaseline: sslMonitorSecurityBaselineFromObject(ctx, model.SecurityBaseline, diags),
	}

	// serverName is nullable in the API; send a pointer only when set.
	if serverName := model.ServerName.ValueString(); serverName != "" {
		config.ServerName = &serverName
	}

	request := checkly.SSLRequest{
		SSLConfig:  config,
		Assertions: assertionsFromSet(ctx, model.Assertions, diags),
	}

	// client_certificate.mode maps to sslConfig.clientCertificateMode
	// (auto|explicit); "account_default" is the inherit case and is sent as
	// an empty (omitted) mode. The certificate ID maps to the top-level
	// sslClientCertificateId foreign key and is required when the mode is
	// "explicit".
	if cert := objectAs[SSLMonitorClientCertificateModel](ctx, model.ClientCertificate, diags); cert != nil {
		if mode := cert.Mode.ValueString(); mode != "" && mode != "account_default" {
			request.SSLConfig.ClientCertificateMode = checkly.SSLClientCertificateMode(mode)
		}

		if certID := cert.ClientCertificateID.ValueString(); certID != "" {
			request.SSLClientCertificateID = &certID
		}
	}

	return request
}

// sslMonitorRequestToObject converts the API representation of an SSL
// request to the request attribute value. prior is the request value held
// before the conversion (the plan on create/update, the state on read); it
// preserves a configured empty client certificate ID and keeps an
// unconfigured assertion set null.
func sslMonitorRequestToObject(ctx context.Context, r checkly.SSLRequest, prior types.Object, diags *diag.Diagnostics) types.Object {
	priorRequest := sslMonitorPriorRequest(ctx, prior, diags)

	config := r.SSLConfig

	serverName := ""
	if config.ServerName != nil {
		serverName = *config.ServerName
	}

	obj, d := types.ObjectValueFrom(ctx, sslMonitorRequestAttrTypes, SSLMonitorRequestModel{
		Hostname:              types.StringValue(config.Hostname),
		Port:                  types.Int64Value(int64(config.Port)),
		ServerName:            types.StringValue(serverName),
		IPFamily:              types.StringValue(string(config.IPFamily)),
		SkipChainValidation:   types.BoolValue(config.SkipChainValidation),
		HandshakeTimeoutMs:    types.Int64Value(int64(config.HandshakeTimeoutMs)),
		AlertDaysBeforeExpiry: types.Int64Value(int64(config.AlertDaysBeforeExpiry)),
		SecurityBaseline:      sslMonitorSecurityBaselineToObject(ctx, config.SecurityBaseline, diags),
		ClientCertificate:     sslMonitorClientCertificateToObject(ctx, r, priorRequest.ClientCertificate, diags),
		Assertions:            assertionsToSet(ctx, r.Assertions, priorAssertions(prior), diags),
	})
	diags.Append(d...)

	return obj
}

// sslMonitorPriorRequest returns the model of a prior request value. Without
// a prior value (e.g. on import) the client certificate is null.
func sslMonitorPriorRequest(ctx context.Context, prior types.Object, diags *diag.Diagnostics) SSLMonitorRequestModel {
	empty := SSLMonitorRequestModel{
		ClientCertificate: types.ObjectNull(sslMonitorClientCertificateAttrTypes),
	}

	model := objectAs[SSLMonitorRequestModel](ctx, prior, diags)
	if model == nil {
		return empty
	}

	return *model
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
	if r.SSLClientCertificateID != nil {
		certID = types.StringValue(*r.SSLClientCertificateID)
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
		"mode":                  types.StringValue(string(mode)),
		"client_certificate_id": certID,
	})
	diags.Append(d...)

	return obj
}

// sslMonitorSecurityBaselineFromObject converts the request.security_baseline
// attribute value to its API representation. After the schema defaults are
// applied the value holds every rule with all leaves, so the payload is the
// complete baseline. A null value (not expected once defaults apply)
// converts to nil, which leaves securityBaseline out of the payload.
func sslMonitorSecurityBaselineFromObject(ctx context.Context, obj types.Object, diags *diag.Diagnostics) *checkly.SecurityBaseline {
	model := objectAs[SSLMonitorSecurityBaselineModel](ctx, obj, diags)
	if model == nil {
		return nil
	}

	enabled := true
	if !model.Enabled.IsNull() && !model.Enabled.IsUnknown() {
		enabled = model.Enabled.ValueBool()
	}

	return &checkly.SecurityBaseline{
		Enabled:                 &enabled,
		MinTLSVersion:           sslMonitorTLSRuleFromObject(ctx, model.MinTLSVersion, diags),
		MinKeySizeBits:          sslMonitorKeySizeRuleFromObject(ctx, model.MinKeySizeBits, diags),
		WeakSignatureAlgorithm:  sslMonitorSeverityRuleFromObject(ctx, model.WeakSignatureAlgorithm, diags),
		WeakCipherSuite:         sslMonitorSeverityRuleFromObject(ctx, model.WeakCipherSuite, diags),
		KnownBadCA:              sslMonitorSeverityRuleFromObject(ctx, model.KnownBadCA, diags),
		RecommendedTLSVersion:   sslMonitorTLSRuleFromObject(ctx, model.RecommendedTLSVersion, diags),
		RecommendedKeySizeBits:  sslMonitorKeySizeRuleFromObject(ctx, model.RecommendedKeySizeBits, diags),
		OCSPMustStapleRespected: sslMonitorSeverityRuleFromObject(ctx, model.OCSPMustStapleRespected, diags),
		SCTPresent:              sslMonitorSeverityRuleFromObject(ctx, model.SCTPresent, diags),
	}
}

func sslMonitorTLSRuleFromObject(ctx context.Context, obj types.Object, diags *diag.Diagnostics) *checkly.SSLBaselineTLSRule {
	model := objectAs[SSLMonitorBaselineTLSRuleModel](ctx, obj, diags)
	if model == nil {
		return nil
	}

	return &checkly.SSLBaselineTLSRule{
		Value:    checkly.TLSVersion(model.Value.ValueString()),
		Severity: checkly.SSLBaselineSeverity(model.Severity.ValueString()),
	}
}

func sslMonitorKeySizeRuleFromObject(ctx context.Context, obj types.Object, diags *diag.Diagnostics) *checkly.SSLBaselineKeySizeRule {
	model := objectAs[SSLMonitorBaselineKeySizeRuleModel](ctx, obj, diags)
	if model == nil {
		return nil
	}

	return &checkly.SSLBaselineKeySizeRule{
		Value:    int(model.Value.ValueInt64()),
		Severity: checkly.SSLBaselineSeverity(model.Severity.ValueString()),
	}
}

func sslMonitorSeverityRuleFromObject(ctx context.Context, obj types.Object, diags *diag.Diagnostics) *checkly.SSLBaselineSeverityRule {
	model := objectAs[SSLMonitorBaselineSeverityRuleModel](ctx, obj, diags)
	if model == nil {
		return nil
	}

	return &checkly.SSLBaselineSeverityRule{
		Severity: checkly.SSLBaselineSeverity(model.Severity.ValueString()),
	}
}

// sslMonitorSecurityBaselineToObject converts the server-returned security
// baseline to the request.security_baseline value. The server returns every
// rule and state holds them all; the schema defaults make an omitted rule plan
// its default, which the provider sends and the server then returns. A missing baseline or rule (not expected from
// the API) converts to its default.
func sslMonitorSecurityBaselineToObject(ctx context.Context, remote *checkly.SecurityBaseline, diags *diag.Diagnostics) types.Object {
	def := &sslMonitorDefaultSecurityBaseline
	if remote == nil {
		remote = def
	}

	enabled := *def.Enabled
	if remote.Enabled != nil {
		enabled = *remote.Enabled
	}

	obj, d := types.ObjectValueFrom(ctx, sslMonitorSecurityBaselineAttrTypes, SSLMonitorSecurityBaselineModel{
		Enabled:                 types.BoolValue(enabled),
		MinTLSVersion:           sslMonitorTLSRuleToObject(remote.MinTLSVersion, def.MinTLSVersion),
		MinKeySizeBits:          sslMonitorKeySizeRuleToObject(remote.MinKeySizeBits, def.MinKeySizeBits),
		WeakSignatureAlgorithm:  sslMonitorSeverityRuleToObject(remote.WeakSignatureAlgorithm, def.WeakSignatureAlgorithm),
		WeakCipherSuite:         sslMonitorSeverityRuleToObject(remote.WeakCipherSuite, def.WeakCipherSuite),
		KnownBadCA:              sslMonitorSeverityRuleToObject(remote.KnownBadCA, def.KnownBadCA),
		RecommendedTLSVersion:   sslMonitorTLSRuleToObject(remote.RecommendedTLSVersion, def.RecommendedTLSVersion),
		RecommendedKeySizeBits:  sslMonitorKeySizeRuleToObject(remote.RecommendedKeySizeBits, def.RecommendedKeySizeBits),
		OCSPMustStapleRespected: sslMonitorSeverityRuleToObject(remote.OCSPMustStapleRespected, def.OCSPMustStapleRespected),
		SCTPresent:              sslMonitorSeverityRuleToObject(remote.SCTPresent, def.SCTPresent),
	})
	diags.Append(d...)

	return obj
}

// The rule converters below convert a rule to its attribute value, falling
// back to def when the rule is missing.

func sslMonitorTLSRuleToObject(remote, def *checkly.SSLBaselineTLSRule) types.Object {
	if remote == nil {
		remote = def
	}

	return types.ObjectValueMust(sslMonitorBaselineTLSRuleAttrTypes, map[string]attr.Value{
		"value":    types.StringValue(string(remote.Value)),
		"severity": types.StringValue(string(remote.Severity)),
	})
}

func sslMonitorKeySizeRuleToObject(remote, def *checkly.SSLBaselineKeySizeRule) types.Object {
	if remote == nil {
		remote = def
	}

	return types.ObjectValueMust(sslMonitorBaselineKeySizeRuleAttrTypes, map[string]attr.Value{
		"value":    types.Int64Value(int64(remote.Value)),
		"severity": types.StringValue(string(remote.Severity)),
	})
}

func sslMonitorSeverityRuleToObject(remote, def *checkly.SSLBaselineSeverityRule) types.Object {
	if remote == nil {
		remote = def
	}

	return types.ObjectValueMust(sslMonitorBaselineSeverityRuleAttrTypes, map[string]attr.Value{
		"severity": types.StringValue(string(remote.Severity)),
	})
}
