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

// sslMonitorBaselineSeverityAttribute returns the severity attribute for a
// security baseline rule. The default mirrors the server's: enforceable rules
// default to "fail", advisory rules to "ignore". Because every leaf carries a
// default, a rule that is present in the configuration always produces
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

func sslMonitorBaselineTLSRuleAttribute(defaultValue, defaultSeverity, description string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
		Description: description,
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
	}
}

func sslMonitorBaselineKeySizeRuleAttribute(defaultValue int64, defaultSeverity, description string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
		Description: description,
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
	}
}

func sslMonitorBaselineSeverityRuleAttribute(defaultSeverity, description string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
		Description: description,
		Attributes: map[string]schema.Attribute{
			"severity": sslMonitorBaselineSeverityAttribute(defaultSeverity),
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
				Monitor:            true,
				AllowHighFrequency: true,
			}),
			frequencyOffsetAttributeName: makeFrequencyOffsetAttributeSchema(FrequencyOffsetAttributeSchemaOptions{
				Monitor: true,
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
				Monitor: true,
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
					"security_baseline": schema.SingleNestedAttribute{
						Optional:    true,
						Description: "The SSL security baseline — a set of enforceable and advisory rules. Omit the attribute to inherit the account default baseline. Rules that are not listed keep their server defaults; removing a rule (or the whole attribute) resets it to its default on the next apply. Only listed rules are drift-checked: an external change to an unlisted rule is not shown by `terraform plan` and is reset on the next apply.",
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Default:     booldefault.StaticBool(true),
								Description: "Whether the security baseline is enforced. (Default `true`).",
							},
							"min_tls_version":            sslMonitorBaselineTLSRuleAttribute("TLS1.2", "fail", "Enforceable rule: the minimum TLS version the server must accept."),
							"min_key_size_bits":          sslMonitorBaselineKeySizeRuleAttribute(2048, "fail", "Enforceable rule: the minimum public key size in bits."),
							"weak_signature_algorithm":   sslMonitorBaselineSeverityRuleAttribute("fail", "Enforceable rule: the certificate must not use a weak signature algorithm."),
							"weak_cipher_suite":          sslMonitorBaselineSeverityRuleAttribute("fail", "Enforceable rule: the connection must not negotiate a weak cipher suite."),
							"known_bad_ca":               sslMonitorBaselineSeverityRuleAttribute("fail", "Enforceable rule: the certificate chain must not include a known-bad CA."),
							"recommended_tls_version":    sslMonitorBaselineTLSRuleAttribute("TLS1.3", "ignore", "Advisory rule: the recommended TLS version."),
							"recommended_key_size_bits":  sslMonitorBaselineKeySizeRuleAttribute(3072, "ignore", "Advisory rule: the recommended public key size in bits."),
							"ocsp_must_staple_respected": sslMonitorBaselineSeverityRuleAttribute("ignore", "Advisory rule: an OCSP Must-Staple extension, when present, must be respected."),
							"sct_present":                sslMonitorBaselineSeverityRuleAttribute("ignore", "Advisory rule: the certificate should carry a Signed Certificate Timestamp."),
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
				Monitor: true,
			}),
		}, CommonCheckAttributeSchemaOptions{Subject: subjectMonitor, ShouldFail: true, RunParallel: true, Locations: true, GroupMembership: true, RuntimeID: true}),
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
	model.ShouldFail = types.BoolValue(c.ShouldFail)
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
	model.GroupID = types.Int64Value(c.GroupID)
	model.GroupOrder = types.Int64Value(int64(c.GroupOrder))
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
		// A nil baseline (attribute omitted) makes the server apply its
		// default baseline. The API replaces the stored baseline wholesale on
		// every write (rules missing from the payload are reset to their
		// defaults), so the payload carries exactly the rules present in the
		// configuration; no state values are mixed in.
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
	if !model.ClientCertificate.IsNull() && !model.ClientCertificate.IsUnknown() {
		var cert SSLMonitorClientCertificateModel
		diags.Append(model.ClientCertificate.As(ctx, &cert, basetypes.ObjectAsOptions{})...)

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
// drives the security baseline projection (see
// sslMonitorSecurityBaselineToObject) and preserves a configured empty client
// certificate ID.
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
		SecurityBaseline:      sslMonitorSecurityBaselineToObject(ctx, config.SecurityBaseline, priorRequest.SecurityBaseline, diags),
		ClientCertificate:     sslMonitorClientCertificateToObject(ctx, r, priorRequest.ClientCertificate, diags),
		Assertions:            assertionsToSet(ctx, r.Assertions, priorAssertions(prior), diags),
	})
	diags.Append(d...)

	return obj
}

// sslMonitorPriorRequest returns the model of a prior request value. Without
// a prior value (e.g. on import) the nested values are null.
func sslMonitorPriorRequest(ctx context.Context, prior types.Object, diags *diag.Diagnostics) SSLMonitorRequestModel {
	empty := SSLMonitorRequestModel{
		SecurityBaseline:  types.ObjectNull(sslMonitorSecurityBaselineAttrTypes),
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
// attribute value to its API representation. An absent baseline converts to
// nil, which omits securityBaseline from the payload so the server applies
// (or resets to) its default baseline. Rules present in the configuration
// always carry both leaves, as the schema defaults fill whatever the
// configuration leaves out.
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

// sslMonitorSecurityBaselineToObject projects the server-returned security
// baseline onto the shape of the prior request.security_baseline value
// (which mirrors the configuration as of the last apply). The server
// normalizes every baseline to the full rule set, so writing the raw response
// into state would make any configuration that omits a rule diff forever.
// Instead, only rules present in the prior value are written back, with the
// server's values: drift on configured rules stays visible, while
// server-filled defaults for unconfigured rules stay null. Without a prior
// baseline (never configured, or a fresh import) the baseline stays null;
// after an import the first plan therefore shows the configured baseline as
// an addition.
func sslMonitorSecurityBaselineToObject(ctx context.Context, remote *checkly.SecurityBaseline, prior types.Object, diags *diag.Diagnostics) types.Object {
	null := types.ObjectNull(sslMonitorSecurityBaselineAttrTypes)

	if remote == nil {
		return null
	}

	priorModel := objectAs[SSLMonitorSecurityBaselineModel](ctx, prior, diags)
	if priorModel == nil {
		return null
	}

	obj, d := types.ObjectValueFrom(ctx, sslMonitorSecurityBaselineAttrTypes, SSLMonitorSecurityBaselineModel{
		Enabled:                 types.BoolValue(remote.Enabled == nil || *remote.Enabled),
		MinTLSVersion:           sslMonitorTLSRuleToObject(remote.MinTLSVersion, priorModel.MinTLSVersion),
		MinKeySizeBits:          sslMonitorKeySizeRuleToObject(remote.MinKeySizeBits, priorModel.MinKeySizeBits),
		WeakSignatureAlgorithm:  sslMonitorSeverityRuleToObject(remote.WeakSignatureAlgorithm, priorModel.WeakSignatureAlgorithm),
		WeakCipherSuite:         sslMonitorSeverityRuleToObject(remote.WeakCipherSuite, priorModel.WeakCipherSuite),
		KnownBadCA:              sslMonitorSeverityRuleToObject(remote.KnownBadCA, priorModel.KnownBadCA),
		RecommendedTLSVersion:   sslMonitorTLSRuleToObject(remote.RecommendedTLSVersion, priorModel.RecommendedTLSVersion),
		RecommendedKeySizeBits:  sslMonitorKeySizeRuleToObject(remote.RecommendedKeySizeBits, priorModel.RecommendedKeySizeBits),
		OCSPMustStapleRespected: sslMonitorSeverityRuleToObject(remote.OCSPMustStapleRespected, priorModel.OCSPMustStapleRespected),
		SCTPresent:              sslMonitorSeverityRuleToObject(remote.SCTPresent, priorModel.SCTPresent),
	})
	diags.Append(d...)

	return obj
}

func sslMonitorTLSRuleToObject(remote *checkly.SSLBaselineTLSRule, prior types.Object) types.Object {
	if remote == nil || prior.IsNull() || prior.IsUnknown() {
		return types.ObjectNull(sslMonitorBaselineTLSRuleAttrTypes)
	}

	return types.ObjectValueMust(sslMonitorBaselineTLSRuleAttrTypes, map[string]attr.Value{
		"value":    types.StringValue(string(remote.Value)),
		"severity": types.StringValue(string(remote.Severity)),
	})
}

func sslMonitorKeySizeRuleToObject(remote *checkly.SSLBaselineKeySizeRule, prior types.Object) types.Object {
	if remote == nil || prior.IsNull() || prior.IsUnknown() {
		return types.ObjectNull(sslMonitorBaselineKeySizeRuleAttrTypes)
	}

	return types.ObjectValueMust(sslMonitorBaselineKeySizeRuleAttrTypes, map[string]attr.Value{
		"value":    types.Int64Value(int64(remote.Value)),
		"severity": types.StringValue(string(remote.Severity)),
	})
}

func sslMonitorSeverityRuleToObject(remote *checkly.SSLBaselineSeverityRule, prior types.Object) types.Object {
	if remote == nil || prior.IsNull() || prior.IsUnknown() {
		return types.ObjectNull(sslMonitorBaselineSeverityRuleAttrTypes)
	}

	return types.ObjectValueMust(sslMonitorBaselineSeverityRuleAttrTypes, map[string]attr.Value{
		"severity": types.StringValue(string(remote.Severity)),
	})
}
