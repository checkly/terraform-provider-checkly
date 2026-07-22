package checkly

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// sslBaselineSeveritySchema returns the severity attribute for a security
// baseline rule. The default mirrors the server's: enforceable rules default
// to "fail", advisory rules to "ignore". Because every leaf carries a default,
// a rule block that is present in the config always produces a fully-populated
// payload, and removing a leaf from the config falls back to the default with
// a visible plan diff.
func sslBaselineSeveritySchema(defaultSeverity string) *schema.Schema {
	return &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		Default:      defaultSeverity,
		ValidateFunc: validateOneOf([]string{"fail", "degrade", "ignore"}),
		Description:  fmt.Sprintf("What happens when the rule is violated: `fail` fails the monitor, `degrade` marks it degraded, `ignore` disables the rule. (Default `%s`).", defaultSeverity),
	}
}

func sslBaselineTLSRuleSchema(defaultValue, defaultSeverity, description string) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		MaxItems:    1,
		Description: description,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"value": {
					Type:         schema.TypeString,
					Optional:     true,
					Default:      defaultValue,
					ValidateFunc: validateOneOf([]string{"TLS1.2", "TLS1.3"}),
					Description:  fmt.Sprintf("The TLS version. Possible values are `TLS1.2` and `TLS1.3`. (Default `%s`).", defaultValue),
				},
				"severity": sslBaselineSeveritySchema(defaultSeverity),
			},
		},
	}
}

func sslBaselineKeySizeRuleSchema(defaultValue int, defaultSeverity, description string) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		MaxItems:    1,
		Description: description,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"value": {
					Type:         schema.TypeInt,
					Optional:     true,
					Default:      defaultValue,
					ValidateFunc: validateBetween(1024, 16384),
					Description:  fmt.Sprintf("The key size in bits. Possible values are between 1024 and 16384. (Default `%d`).", defaultValue),
				},
				"severity": sslBaselineSeveritySchema(defaultSeverity),
			},
		},
	}
}

func sslBaselineSeverityRuleSchema(defaultSeverity, description string) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		MaxItems:    1,
		Description: description,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"severity": sslBaselineSeveritySchema(defaultSeverity),
			},
		},
	}
}

func resourceSSLMonitor() *schema.Resource {
	return &schema.Resource{
		Create: resourceSSLMonitorCreate,
		Read:   resourceSSLMonitorRead,
		Update: resourceSSLMonitorUpdate,
		Delete: resourceSSLMonitorDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "SSL monitors check the TLS certificate of a host and alert before it expires or violates a security baseline.",
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the monitor.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A description of the monitor.",
			},
			frequencyAttributeName: makeFrequencyAttributeSchema(FrequencyAttributeSchemaOptions{
				Monitor:            true,
				AllowHighFrequency: true,
			}),
			frequencyOffsetAttributeName: makeFrequencyOffsetAttributeSchema(FrequencyOffsetAttributeSchemaOptions{
				Monitor: true,
			}),
			"activated": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Determines if the monitor is running or not. Possible values `true`, and `false`.",
			},
			"muted": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Determines if any notifications will be sent out when a monitor fails/degrades/recovers.",
			},
			"should_fail": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Allows to invert the behaviour of when a monitor is considered to fail.",
			},
			"run_parallel": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Determines if the monitor should run in all selected locations in parallel or round-robin.",
			},
			"locations": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "An array of one or more data center locations where to run this monitor. (Default [\"us-east-1\"])",
			},
			"degraded_response_time": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      10000,
				ValidateFunc: validateBetween(0, 30000),
				Description:  "The handshake time in milliseconds above which the monitor is considered degraded. Possible values are between 0 and 30000. (Default `10000`).",
			},
			"max_response_time": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      20000,
				ValidateFunc: validateBetween(0, 30000),
				Description:  "The handshake time in milliseconds above which the monitor is considered failing. Must be greater than or equal to `degraded_response_time`. Possible values are between 0 and 30000. (Default `20000`).",
			},
			"tags": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "A list of tags for organizing and filtering checks and monitors.",
			},
			"runtime_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     nil,
				Description: "The ID of the runtime to use for this monitor.",
			},
			alertChannelSubscriptionAttributeName: makeAlertChannelSubscriptionAttributeSchema(AlertChannelSubscriptionAttributeSchemaOptions{
				Monitor: true,
			}),
			"private_locations": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				DefaultFunc: func() (interface{}, error) {
					return []tfMap{}, nil
				},
				Description: "An array of one or more private locations slugs.",
			},
			alertSettingsAttributeName: makeAlertSettingsAttributeSchema(AlertSettingsAttributeSchemaOptions{
				Monitor: true,
			}),
			"use_global_alert_settings": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "When true, the account level alert settings will be used, not the alert setting defined on this monitor.",
			},
			"request": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"hostname": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The hostname to connect to and validate the TLS certificate of. Do not include a scheme or a port in this value.",
						},
						"port": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      443,
							ValidateFunc: validateBetween(1, 65535),
							Description:  "The port number to connect to. Possible values are between 1 and 65535. (Default `443`).",
						},
						"server_name": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "An optional SNI server name to send in the TLS handshake. Defaults to `hostname` when unset.",
						},
						"ip_family": {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "IPv4",
							ValidateFunc: validateOneOf([]string{"IPv4", "IPv6"}),
							Description:  "The IP family to use when executing the check. The value can be either `IPv4` or `IPv6`. (Default `IPv4`).",
						},
						"skip_chain_validation": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "When true, the certificate chain is not validated against trusted roots (the certificate is still inspected for expiry and the security baseline). (Default `false`).",
						},
						"handshake_timeout_ms": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      10000,
							ValidateFunc: validateBetween(1000, 30000),
							Description:  "The number of milliseconds to wait for the TLS handshake to complete before timing out. Possible values are between 1000 and 30000. (Default `10000`).",
						},
						"alert_days_before_expiry": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      20,
							ValidateFunc: validateBetween(1, 365),
							Description:  "Raise an alert when the certificate is within this many days of expiry. Possible values are between 1 and 365. (Default `20`).",
						},
						"security_baseline": {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "The SSL security baseline — a set of enforceable and advisory rules. Omit the block to inherit the account default baseline. Rules that are not listed keep their server defaults; removing a rule (or the whole block) resets it to its default on the next apply. Only listed rules are drift-checked: an external change to an unlisted rule is not shown by `terraform plan` and is reset on the next apply.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"enabled": {
										Type:        schema.TypeBool,
										Optional:    true,
										Default:     true,
										Description: "Whether the security baseline is enforced. (Default `true`).",
									},
									"min_tls_version":            sslBaselineTLSRuleSchema("TLS1.2", "fail", "Enforceable rule: the minimum TLS version the server must accept."),
									"min_key_size_bits":          sslBaselineKeySizeRuleSchema(2048, "fail", "Enforceable rule: the minimum public key size in bits."),
									"weak_signature_algorithm":   sslBaselineSeverityRuleSchema("fail", "Enforceable rule: the certificate must not use a weak signature algorithm."),
									"weak_cipher_suite":          sslBaselineSeverityRuleSchema("fail", "Enforceable rule: the connection must not negotiate a weak cipher suite."),
									"known_bad_ca":               sslBaselineSeverityRuleSchema("fail", "Enforceable rule: the certificate chain must not include a known-bad CA."),
									"recommended_tls_version":    sslBaselineTLSRuleSchema("TLS1.3", "ignore", "Advisory rule: the recommended TLS version."),
									"recommended_key_size_bits":  sslBaselineKeySizeRuleSchema(3072, "ignore", "Advisory rule: the recommended public key size in bits."),
									"ocsp_must_staple_respected": sslBaselineSeverityRuleSchema("ignore", "Advisory rule: an OCSP Must-Staple extension, when present, must be respected."),
									"sct_present":                sslBaselineSeverityRuleSchema("ignore", "Advisory rule: the certificate should carry a Signed Certificate Timestamp."),
								},
							},
						},
						"client_certificate": {
							Type:     schema.TypeSet,
							Optional: true,
							Computed: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"mode": {
										Type:         schema.TypeString,
										Optional:     true,
										Default:      "account_default",
										ValidateFunc: validateOneOf([]string{"account_default", "auto", "explicit"}),
										Description:  "The mutual-TLS client-certificate mode. `account_default` inherits the account setting (no certificate sent), `auto` lets Checkly select a stored certificate, `explicit` uses the certificate referenced by `client_certificate_id`. (Default `account_default`).",
									},
									"client_certificate_id": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "The ID of the stored client certificate to present. Required when `mode = \"explicit\"`.",
									},
								},
							},
							Description: "The mutual-TLS client certificate configuration.",
						},
						// The full assertion grammar (per-source property
						// lists and the property-dependent comparison sets)
						// is documented in the "Assertion Reference" section
						// of templates/resources/ssl_monitor.md.tmpl. When
						// assertion sources, properties, or their rules
						// change here or in the API, update that template
						// too — it is not generated from this schema.
						"assertion": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"source": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validateOneOf([]string{"CERTIFICATE", "CONNECTION", "RESPONSE_TIME", "JSON_RESPONSE", "TEXT_RESPONSE"}),
										Description:  "The source of the asserted value. Possible values are `CERTIFICATE`, `CONNECTION`, `RESPONSE_TIME`, `JSON_RESPONSE`, and `TEXT_RESPONSE`.",
									},
									"property": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "The property selecting the asserted value within the source. For `CERTIFICATE`: `daysUntilExpiry`, `keySizeBits`, `subjectCN`, `issuerCN`, `serialNumber`, `fingerprintSha256`, `issuerFingerprintSha256`, `keyAlgorithm`, `signatureAlgorithm`, `sans`, `selfSigned`, or `isCA`. For `CONNECTION`: `tlsVersion`, `cipherSuite`, `hostnameVerified`, `chainTrusted`, `ocspStapled`, `ocspStatus`, or `resolvedIp`. For `JSON_RESPONSE`: a JSONPath expression. For `TEXT_RESPONSE`: a regular expression applied to the serialized response.",
									},
									"comparison": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "The type of comparison to be executed between expected and actual value of the assertion. Possible values are `EQUALS`, `NOT_EQUALS`, `IS_EMPTY`, `NOT_EMPTY`, `GREATER_THAN`, `LESS_THAN`, `CONTAINS`, `NOT_CONTAINS`, `IS_NULL`, and `NOT_NULL`. The allowed set depends on the asserted `source` and `property`; for example, boolean properties such as `chainTrusted` only allow `EQUALS`.",
									},
									"target": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
							Description: "A request can have multiple assertions.",
						},
					},
				},
				Description: "The parameters for the SSL connection.",
			},
			"group_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The id of the check group this monitor is part of.",
			},
			"group_order": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The position of this monitor in a check group. It determines in what order checks and monitors are run when a group is triggered from the API or from CI/CD.",
			},
			retryStrategyAttributeName: makeRetryStrategyAttributeSchema(RetryStrategyAttributeSchemaOptions{
				Required: false,
				Computed: true,
			}),
			"trigger_incident": triggerIncidentAttributeSchema,
		},
		CustomizeDiff: customdiff.Sequence(
			RetryStrategyCustomizeDiff,
			FrequencyOffsetCustomizeDiff,
		),
	}
}

func resourceSSLMonitorCreate(d *schema.ResourceData, client interface{}) error {
	monitor, err := sslCheckFromResourceData(d)
	if err != nil {
		return fmt.Errorf("translation error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	newMonitor, err := client.(checkly.Client).CreateSSLMonitor(ctx, monitor)

	if err != nil {
		checkJSON, _ := json.Marshal(monitor)
		return fmt.Errorf("API error 1: %w, Check: %s", err, string(checkJSON))
	}
	d.SetId(newMonitor.ID)
	return resourceSSLMonitorRead(d, client)
}

func resourceSSLMonitorRead(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	monitor, err := client.(checkly.Client).GetSSLMonitor(ctx, d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			//if resource is deleted remotely, then mark it as
			//successfully gone by unsetting it's ID
			d.SetId("")
			return nil
		}
		return fmt.Errorf("API error 2: %w", err)
	}
	return resourceDataFromSSLMonitor(monitor, d)
}

func resourceSSLMonitorUpdate(d *schema.ResourceData, client interface{}) error {
	monitor, err := sslCheckFromResourceData(d)

	if err != nil {
		return fmt.Errorf("translation error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	_, err = client.(checkly.Client).UpdateSSLMonitor(ctx, monitor.ID, monitor)
	if err != nil {
		checkJSON, _ := json.Marshal(monitor)
		return fmt.Errorf("API error 3: Couldn't update check, Error: %w, \nCheck: %s", err, checkJSON)
	}
	d.SetId(monitor.ID)
	return resourceSSLMonitorRead(d, client)
}

func resourceSSLMonitorDelete(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	if err := client.(checkly.Client).DeleteSSLMonitor(ctx, d.Id()); err != nil {
		return fmt.Errorf("API error 4: Couldn't delete Check %s, Error: %w", d.Id(), err)
	}
	return nil
}

func resourceDataFromSSLMonitor(c *checkly.SSLMonitor, d *schema.ResourceData) error {
	d.Set("name", c.Name)
	d.Set("description", c.Description)
	d.Set("activated", c.Activated)
	d.Set("muted", c.Muted)
	d.Set("should_fail", c.ShouldFail)
	d.Set("run_parallel", c.RunParallel)
	d.Set("locations", c.Locations)
	d.Set("degraded_response_time", c.DegradedResponseTime)
	d.Set("max_response_time", c.MaxResponseTime)

	sort.Strings(c.Tags)
	d.Set("tags", c.Tags)

	d.Set(frequencyAttributeName, c.Frequency)
	if c.Frequency == 0 {
		d.Set(frequencyOffsetAttributeName, c.FrequencyOffset)
	} else {
		d.Set(frequencyOffsetAttributeName, nil)
	}

	if c.RuntimeID != nil {
		d.Set("runtime_id", *c.RuntimeID)
	}

	if err := d.Set("alert_settings", setFromAlertSettings(*c.AlertSettings)); err != nil {
		return fmt.Errorf("error setting alert settings for resource %s: %w", d.Id(), err)
	}
	d.Set("use_global_alert_settings", c.UseGlobalAlertSettings)

	var priorRequest tfMap
	if priorList, ok := d.Get("request").([]any); ok && len(priorList) > 0 && priorList[0] != nil {
		priorRequest = priorList[0].(tfMap)
	}
	request, err := setFromSSLRequest(c.Request, priorRequest)
	if err != nil {
		return fmt.Errorf("error encoding request for resource %s: %w", d.Id(), err)
	}
	if err := d.Set("request", request); err != nil {
		return fmt.Errorf("error setting request for resource %s: %w", d.Id(), err)
	}
	d.Set("group_id", c.GroupID)
	d.Set("group_order", c.GroupOrder)
	d.Set("private_locations", c.PrivateLocations)
	d.Set(alertChannelSubscriptionAttributeName, setFromAlertChannelSubscriptions(c.AlertChannelSubscriptions))
	d.Set(retryStrategyAttributeName, listFromRetryStrategy(c.RetryStrategy))
	d.Set("trigger_incident", setFromTriggerIncident(c.TriggerIncident))
	d.SetId(d.Id())
	return nil
}

// setFromSSLRequest flattens an API-returned SSL request into resource data.
// priorRequest is the request map from the state prior to this read (nil when
// there is none, e.g. on import); it drives the security-baseline projection —
// see setFromSecurityBaseline.
func setFromSSLRequest(r checkly.SSLRequest, priorRequest tfMap) ([]tfMap, error) {
	cfg := r.SSLConfig
	s := tfMap{}
	s["hostname"] = cfg.Hostname
	s["port"] = cfg.Port
	if cfg.ServerName != nil {
		s["server_name"] = *cfg.ServerName
	} else {
		s["server_name"] = ""
	}
	s["ip_family"] = cfg.IPFamily
	s["skip_chain_validation"] = cfg.SkipChainValidation
	s["handshake_timeout_ms"] = cfg.HandshakeTimeoutMs
	s["alert_days_before_expiry"] = cfg.AlertDaysBeforeExpiry

	s["security_baseline"] = setFromSecurityBaseline(cfg.SecurityBaseline, priorRequest)

	// client_certificate is always emitted so applying exported HCL (which writes
	// `mode = "account_default"`) re-plans clean. An empty API mode maps back to
	// "account_default".
	cert := tfMap{}
	if cfg.ClientCertificateMode == "" {
		cert["mode"] = "account_default"
	} else {
		cert["mode"] = cfg.ClientCertificateMode
	}
	if r.SSLClientCertificateId != nil {
		cert["client_certificate_id"] = *r.SSLClientCertificateId
	} else {
		cert["client_certificate_id"] = ""
	}
	s["client_certificate"] = []tfMap{cert}

	s["assertion"] = setFromAssertions(r.Assertions)
	return []tfMap{s}, nil
}

func sslCheckFromResourceData(d *schema.ResourceData) (checkly.SSLMonitor, error) {
	monitor := checkly.SSLMonitor{
		ID:                        d.Id(),
		Name:                      d.Get("name").(string),
		Description:               optionalStringPointerFromResourceData(d, "description"),
		Frequency:                 d.Get(frequencyAttributeName).(int),
		Activated:                 d.Get("activated").(bool),
		Muted:                     d.Get("muted").(bool),
		ShouldFail:                d.Get("should_fail").(bool),
		RunParallel:               d.Get("run_parallel").(bool),
		Locations:                 stringsFromSet(d.Get("locations").(*schema.Set)),
		DegradedResponseTime:      d.Get("degraded_response_time").(int),
		MaxResponseTime:           d.Get("max_response_time").(int),
		Tags:                      stringsFromSet(d.Get("tags").(*schema.Set)),
		UseGlobalAlertSettings:    d.Get("use_global_alert_settings").(bool),
		GroupID:                   int64(d.Get("group_id").(int)),
		GroupOrder:                d.Get("group_order").(int),
		AlertChannelSubscriptions: alertChannelSubscriptionsFromSet(d.Get(alertChannelSubscriptionAttributeName).(*schema.Set)),
		RetryStrategy:             retryStrategyFromList(d.Get(retryStrategyAttributeName).([]any)),
		TriggerIncident:           triggerIncidentFromSet(d.Get("trigger_incident").(*schema.Set)),
	}

	alertSettings := alertSettingsFromSet(d.Get("alert_settings").([]interface{}))
	monitor.AlertSettings = &alertSettings

	runtimeId := d.Get("runtime_id").(string)
	if runtimeId == "" {
		monitor.RuntimeID = nil
	} else {
		monitor.RuntimeID = &runtimeId
	}

	privateLocations := stringsFromSet(d.Get("private_locations").(*schema.Set))
	monitor.PrivateLocations = &privateLocations

	request, err := sslRequestFromList(d.Get("request").([]any))
	if err != nil {
		return checkly.SSLMonitor{}, err
	}
	monitor.Request = request

	monitor.FrequencyOffset = d.Get(frequencyOffsetAttributeName).(int)

	return monitor, nil
}

func sslRequestFromList(s []any) (checkly.SSLRequest, error) {
	if len(s) == 0 || s[0] == nil {
		return checkly.SSLRequest{}, nil
	}
	res := s[0].(tfMap)

	cfg := checkly.SSLConfig{
		Hostname:              res["hostname"].(string),
		Port:                  res["port"].(int),
		IPFamily:              res["ip_family"].(string),
		SkipChainValidation:   res["skip_chain_validation"].(bool),
		HandshakeTimeoutMs:    res["handshake_timeout_ms"].(int),
		AlertDaysBeforeExpiry: res["alert_days_before_expiry"].(int),
	}

	// serverName is nullable in the API; send a pointer only when set.
	if serverName, ok := res["server_name"].(string); ok && serverName != "" {
		cfg.ServerName = &serverName
	}

	// A nil baseline (block omitted) makes the server apply its default
	// baseline. The API replaces the stored baseline wholesale on every write
	// (rules missing from the payload are reset to their defaults), so the
	// payload carries exactly the rules present in the config — no state
	// values are mixed in.
	if baselineList, ok := res["security_baseline"].([]any); ok {
		cfg.SecurityBaseline = securityBaselineFromList(baselineList)
	}

	request := checkly.SSLRequest{
		SSLConfig:  cfg,
		Assertions: assertionsFromSet(res["assertion"].(*schema.Set)),
	}

	// client_certificate.mode maps to sslConfig.clientCertificateMode (auto|
	// explicit); "account_default" is the inherit case and is sent as an empty
	// (omitted) mode. The certificate id maps to the top-level
	// sslClientCertificateId FK and is required when mode is "explicit".
	if certSet, ok := res["client_certificate"].(*schema.Set); ok && certSet.Len() > 0 {
		cert := certSet.List()[0].(tfMap)
		mode := cert["mode"].(string)
		if mode != "" && mode != "account_default" {
			request.SSLConfig.ClientCertificateMode = mode
		}
		if certID, ok := cert["client_certificate_id"].(string); ok && certID != "" {
			request.SSLClientCertificateId = &certID
		}
	}

	return request, nil
}

// securityBaselineFromList expands the security_baseline block into the API
// payload. A missing block returns nil, which omits securityBaseline from the
// payload so the server applies (or resets to) its default baseline. Rules
// present in the config always carry both leaves — the schema defaults fill
// whatever the user left out — so the payload never depends on state.
func securityBaselineFromList(s []any) *checkly.SecurityBaseline {
	if len(s) == 0 {
		return nil
	}
	enabled := true
	baseline := &checkly.SecurityBaseline{}
	if s[0] != nil {
		res := s[0].(tfMap)
		enabled = res["enabled"].(bool)
		baseline.MinTLSVersion = sslTLSRuleFromList(res["min_tls_version"])
		baseline.MinKeySizeBits = sslKeySizeRuleFromList(res["min_key_size_bits"])
		baseline.WeakSignatureAlgorithm = sslSeverityRuleFromList(res["weak_signature_algorithm"])
		baseline.WeakCipherSuite = sslSeverityRuleFromList(res["weak_cipher_suite"])
		baseline.KnownBadCA = sslSeverityRuleFromList(res["known_bad_ca"])
		baseline.RecommendedTLSVersion = sslTLSRuleFromList(res["recommended_tls_version"])
		baseline.RecommendedKeySizeBits = sslKeySizeRuleFromList(res["recommended_key_size_bits"])
		baseline.OCSPMustStapleRespected = sslSeverityRuleFromList(res["ocsp_must_staple_respected"])
		baseline.SCTPresent = sslSeverityRuleFromList(res["sct_present"])
	}
	baseline.Enabled = &enabled
	return baseline
}

func sslTLSRuleFromList(v any) *checkly.SSLBaselineTLSRule {
	l, ok := v.([]any)
	if !ok || len(l) == 0 {
		return nil
	}
	if l[0] == nil {
		return &checkly.SSLBaselineTLSRule{}
	}
	m := l[0].(tfMap)
	return &checkly.SSLBaselineTLSRule{
		Value:    m["value"].(string),
		Severity: m["severity"].(string),
	}
}

func sslKeySizeRuleFromList(v any) *checkly.SSLBaselineKeySizeRule {
	l, ok := v.([]any)
	if !ok || len(l) == 0 {
		return nil
	}
	if l[0] == nil {
		return &checkly.SSLBaselineKeySizeRule{}
	}
	m := l[0].(tfMap)
	return &checkly.SSLBaselineKeySizeRule{
		Value:    m["value"].(int),
		Severity: m["severity"].(string),
	}
}

func sslSeverityRuleFromList(v any) *checkly.SSLBaselineSeverityRule {
	l, ok := v.([]any)
	if !ok || len(l) == 0 {
		return nil
	}
	if l[0] == nil {
		return &checkly.SSLBaselineSeverityRule{}
	}
	m := l[0].(tfMap)
	return &checkly.SSLBaselineSeverityRule{
		Severity: m["severity"].(string),
	}
}

// setFromSecurityBaseline projects the server-returned security baseline onto
// the shape held in prior state (which mirrors the config as of the last
// apply). The server normalizes every baseline to the full rule set, so
// writing the raw response into state would make any config that omits a rule
// diff forever. Instead, only rules present in prior state are written back,
// with the server's values — drift on configured rules stays visible, while
// server-filled defaults for unconfigured rules stay out of state. When prior
// state has no baseline (never configured, or a fresh import), nothing is
// written; after an import the first plan therefore shows the configured
// baseline as an addition.
func setFromSecurityBaseline(remote *checkly.SecurityBaseline, priorRequest tfMap) []tfMap {
	var prior tfMap
	if priorRequest != nil {
		if priorList, ok := priorRequest["security_baseline"].([]any); ok && len(priorList) > 0 && priorList[0] != nil {
			prior = priorList[0].(tfMap)
		}
	}
	if prior == nil || remote == nil {
		return nil
	}

	priorHasRule := func(key string) bool {
		l, ok := prior[key].([]any)
		return ok && len(l) > 0
	}

	s := tfMap{}
	s["enabled"] = remote.Enabled == nil || *remote.Enabled
	if priorHasRule("min_tls_version") && remote.MinTLSVersion != nil {
		s["min_tls_version"] = []tfMap{{"value": remote.MinTLSVersion.Value, "severity": remote.MinTLSVersion.Severity}}
	}
	if priorHasRule("min_key_size_bits") && remote.MinKeySizeBits != nil {
		s["min_key_size_bits"] = []tfMap{{"value": remote.MinKeySizeBits.Value, "severity": remote.MinKeySizeBits.Severity}}
	}
	if priorHasRule("weak_signature_algorithm") && remote.WeakSignatureAlgorithm != nil {
		s["weak_signature_algorithm"] = []tfMap{{"severity": remote.WeakSignatureAlgorithm.Severity}}
	}
	if priorHasRule("weak_cipher_suite") && remote.WeakCipherSuite != nil {
		s["weak_cipher_suite"] = []tfMap{{"severity": remote.WeakCipherSuite.Severity}}
	}
	if priorHasRule("known_bad_ca") && remote.KnownBadCA != nil {
		s["known_bad_ca"] = []tfMap{{"severity": remote.KnownBadCA.Severity}}
	}
	if priorHasRule("recommended_tls_version") && remote.RecommendedTLSVersion != nil {
		s["recommended_tls_version"] = []tfMap{{"value": remote.RecommendedTLSVersion.Value, "severity": remote.RecommendedTLSVersion.Severity}}
	}
	if priorHasRule("recommended_key_size_bits") && remote.RecommendedKeySizeBits != nil {
		s["recommended_key_size_bits"] = []tfMap{{"value": remote.RecommendedKeySizeBits.Value, "severity": remote.RecommendedKeySizeBits.Severity}}
	}
	if priorHasRule("ocsp_must_staple_respected") && remote.OCSPMustStapleRespected != nil {
		s["ocsp_must_staple_respected"] = []tfMap{{"severity": remote.OCSPMustStapleRespected.Severity}}
	}
	if priorHasRule("sct_present") && remote.SCTPresent != nil {
		s["sct_present"] = []tfMap{{"severity": remote.SCTPresent.Severity}}
	}
	return []tfMap{s}
}
