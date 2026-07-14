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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

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
				Default:      3000,
				ValidateFunc: validateBetween(0, 30000),
				Description:  "The handshake time in milliseconds above which the monitor is considered degraded. Possible values are between 0 and 30000. (Default `3000`).",
			},
			"max_response_time": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      10000,
				ValidateFunc: validateBetween(0, 30000),
				Description:  "The handshake time in milliseconds above which the monitor is considered failing. Must be greater than or equal to `degraded_response_time`. Possible values are between 0 and 30000. (Default `10000`).",
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
							Type:             schema.TypeString,
							Optional:         true,
							Computed:         true,
							ValidateFunc:     validation.StringIsJSON,
							DiffSuppressFunc: suppressEquivalentJSON,
							Description:      "The SSL security baseline as a `jsonencode`d object of enforceable/advisory rules. Omit to inherit the account default baseline.",
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
						"assertion": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"source": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validateOneOf([]string{"CERTIFICATE", "CONNECTION", "RESPONSE_TIME", "JSON_RESPONSE", "TEXT_RESPONSE"}),
										Description:  "The source of the asserted value. Possible values are `CERTIFICATE`, `CONNECTION`, `RESPONSE_TIME`, `JSON_RESPONSE`, and `TEXT_RESPONSE`. For `CERTIFICATE` and `CONNECTION`, `property` selects the field to assert on. `RESPONSE_TIME` takes no `property` and asserts on the handshake time in milliseconds. `JSON_RESPONSE` uses `property` as a JSONPath expression, and `TEXT_RESPONSE` uses `property` as a regular expression.",
									},
									"property": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "The property to assert on. For `CERTIFICATE`, a certificate field selector: one of `daysUntilExpiry`, `subjectCN`, `issuerCN`, `serialNumber`, `fingerprintSha256`, `issuerFingerprintSha256`, `keySizeBits`, `keyAlgorithm`, `signatureAlgorithm`, `sans`, `selfSigned`, or `isCA`. For `CONNECTION`, a connection field selector: one of `tlsVersion`, `cipherSuite`, `hostnameVerified`, `chainTrusted`, `ocspStapled`, `ocspStatus`, or `resolvedIp`. For `JSON_RESPONSE`, a JSONPath expression. For `TEXT_RESPONSE`, a regular expression. Not used for `RESPONSE_TIME`.",
									},
									"comparison": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validateOneOf([]string{"EQUALS", "NOT_EQUALS", "CONTAINS", "NOT_CONTAINS", "GREATER_THAN", "LESS_THAN", "IS_EMPTY", "NOT_EMPTY", "IS_NULL", "NOT_NULL"}),
										Description:  "The type of comparison to be executed between expected and actual value of the assertion. Possible values are `EQUALS`, `NOT_EQUALS`, `CONTAINS`, `NOT_CONTAINS`, `GREATER_THAN`, `LESS_THAN`, `IS_EMPTY`, `NOT_EMPTY`, `IS_NULL`, and `NOT_NULL`.",
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

	request, err := setFromSSLRequest(c.Request)
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

func setFromSSLRequest(r checkly.SSLRequest) ([]tfMap, error) {
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

	// SecurityBaseline round-trips as a jsonencode'd string. A nil baseline
	// (server inheriting its default) flattens to an empty string; the schema's
	// Computed flag then absorbs the server-applied default without a perpetual
	// diff. json.Marshal emits keys in struct-declaration order, and the
	// DiffSuppressFunc normalises key order on both sides.
	if cfg.SecurityBaseline != nil {
		baselineJSON, err := json.Marshal(cfg.SecurityBaseline)
		if err != nil {
			return nil, fmt.Errorf("marshalling security_baseline: %w", err)
		}
		s["security_baseline"] = string(baselineJSON)
	} else {
		s["security_baseline"] = ""
	}

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

	// security_baseline is an optional jsonencode'd object. Decode it into a
	// *SecurityBaseline; leave nil when empty so the server applies its default.
	if baseline, ok := res["security_baseline"].(string); ok && strings.TrimSpace(baseline) != "" {
		var parsed checkly.SecurityBaseline
		if err := json.Unmarshal([]byte(baseline), &parsed); err != nil {
			return checkly.SSLRequest{}, fmt.Errorf("decoding security_baseline JSON: %w", err)
		}
		cfg.SecurityBaseline = &parsed
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

// suppressEquivalentJSON treats two JSON strings as equal when they are
// semantically identical, ignoring key order and whitespace differences (the
// jsonencode round-trip anti-pattern).
func suppressEquivalentJSON(_, old, new string, _ *schema.ResourceData) bool {
	oldNorm, err := structure.NormalizeJsonString(old)
	if err != nil {
		return false
	}
	newNorm, err := structure.NormalizeJsonString(new)
	if err != nil {
		return false
	}
	return oldNorm == newNorm
}
