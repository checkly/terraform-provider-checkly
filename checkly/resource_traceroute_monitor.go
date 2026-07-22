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

func resourceTracerouteMonitor() *schema.Resource {
	return &schema.Resource{
		Create: resourceTracerouteMonitorCreate,
		Read:   resourceTracerouteMonitorRead,
		Update: resourceTracerouteMonitorUpdate,
		Delete: resourceTracerouteMonitorDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "Traceroute monitors trace the network path to a host and measure per-hop latency and packet loss. Traceroute monitors do not support private locations.",
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
				Description:  "The response time in milliseconds starting from which a monitor should be considered degraded. Possible values are between 0 and 30000. (Default `10000`).",
			},
			"max_response_time": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      20000,
				ValidateFunc: validateBetween(0, 30000),
				Description:  "The response time in milliseconds starting from which a monitor should be considered failing. Possible values are between 0 and 30000. (Default `20000`).",
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
						"url": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The host to trace the network path to. Do not include a scheme or a port in this value.",
						},
						"protocol": {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "TCP",
							ValidateFunc: validateOneOf([]string{"TCP", "ICMP", "UDP", "SCTP"}),
							Description:  "The probe protocol. `TCP` sends SYN probes (default), `UDP` sends datagrams to a high port, `ICMP` sends Echo Requests, `SCTP` sends INIT chunks. (Default `TCP`).",
						},
						"port": {
							Type:         schema.TypeInt,
							Optional:     true,
							Computed:     true,
							ValidateFunc: validateBetween(1, 65535),
							Description:  "The destination port for TCP/UDP/SCTP probes. Possible values are between 1 and 65535. Ignored (and not sent) when `protocol = \"ICMP\"`. The default depends on the protocol: `443` for `TCP`, `33434` for `UDP` and `SCTP`.",
						},
						"ip_family": {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "IPv4",
							ValidateFunc: validateOneOf([]string{"IPv4", "IPv6"}),
							Description:  "The IP family to use when executing the traceroute. The value can be either `IPv4` or `IPv6`. (Default `IPv4`).",
						},
						"max_hops": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      30,
							ValidateFunc: validateBetween(1, 64),
							Description:  "The maximum number of network hops to probe before stopping. Possible values are between 1 and 64. (Default `30`).",
						},
						"max_unknown_hops": {
							Type:         schema.TypeInt,
							Optional:     true,
							Computed:     true,
							ValidateFunc: validateBetween(1, 30),
							Description:  "The maximum number of consecutive unresponsive hops to tolerate before stopping the trace. Possible values are between 1 and 30, and the value must not exceed `max_hops`. (Default `min(15, max_hops)`).",
						},
						"ptr_lookup": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Whether to perform reverse-DNS (PTR) lookups on each hop's IP address. (Default `true`).",
						},
						"timeout": {
							Type:         schema.TypeInt,
							Optional:     true,
							Computed:     true,
							ValidateFunc: validateBetween(1, 30),
							Description:  "The number of seconds to wait for the traceroute to complete before timing out. Possible values are between 1 and 30. (Default `10`).",
						},
						// The full per-source assertion rules (allowed
						// comparisons, property usage, target formats) are
						// documented in the "Assertion Reference" section of
						// templates/resources/traceroute_monitor.md.tmpl.
						// When assertion sources or their rules change here
						// or in the API, update that template too — it is
						// not generated from this schema.
						"assertion": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"source": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validateOneOf([]string{"RESPONSE_TIME", "HOP_COUNT", "PACKET_LOSS"}),
										Description:  "The source of the asserted value. Possible values are `RESPONSE_TIME`, `HOP_COUNT`, and `PACKET_LOSS`.",
									},
									"property": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "The statistic to assert on. Required for `RESPONSE_TIME`, where possible values are `avg`, `min`, `max`, and `stdDev`. Must be empty for `HOP_COUNT` and `PACKET_LOSS`.",
									},
									"comparison": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "The type of comparison to be executed between expected and actual value of the assertion. For `RESPONSE_TIME`, possible values are `EQUALS`, `NOT_EQUALS`, `GREATER_THAN`, and `LESS_THAN`. For `HOP_COUNT` and `PACKET_LOSS`, possible values are `EQUALS`, `GREATER_THAN`, and `LESS_THAN`.",
									},
									"target": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "The value to compare against. Must be numeric: a non-negative number of milliseconds for `RESPONSE_TIME`, a non-negative integer for `HOP_COUNT`, or a number between 0 and 100 for `PACKET_LOSS`.",
									},
								},
							},
							Description: "A request can have multiple assertions. The allowed comparisons, properties, and target formats depend on the assertion source — see the [Assertion Reference](#assertion-reference) below.",
						},
					},
				},
				Description: "The parameters for the traceroute probe.",
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

func resourceTracerouteMonitorCreate(d *schema.ResourceData, client interface{}) error {
	monitor, err := tracerouteCheckFromResourceData(d)
	if err != nil {
		return fmt.Errorf("translation error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	newMonitor, err := client.(checkly.Client).CreateTracerouteMonitor(ctx, monitor)

	if err != nil {
		checkJSON, _ := json.Marshal(monitor)
		return fmt.Errorf("API error 1: %w, Check: %s", err, string(checkJSON))
	}
	d.SetId(newMonitor.ID)
	return resourceTracerouteMonitorRead(d, client)
}

func resourceTracerouteMonitorRead(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	monitor, err := client.(checkly.Client).GetTracerouteMonitor(ctx, d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			//if resource is deleted remotely, then mark it as
			//successfully gone by unsetting it's ID
			d.SetId("")
			return nil
		}
		return fmt.Errorf("API error 2: %w", err)
	}
	return resourceDataFromTracerouteMonitor(monitor, d)
}

func resourceTracerouteMonitorUpdate(d *schema.ResourceData, client interface{}) error {
	monitor, err := tracerouteCheckFromResourceData(d)

	if err != nil {
		return fmt.Errorf("translation error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	_, err = client.(checkly.Client).UpdateTracerouteMonitor(ctx, monitor.ID, monitor)
	if err != nil {
		checkJSON, _ := json.Marshal(monitor)
		return fmt.Errorf("API error 3: Couldn't update check, Error: %w, \nCheck: %s", err, checkJSON)
	}
	d.SetId(monitor.ID)
	return resourceTracerouteMonitorRead(d, client)
}

func resourceTracerouteMonitorDelete(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	if err := client.(checkly.Client).DeleteTracerouteMonitor(ctx, d.Id()); err != nil {
		return fmt.Errorf("API error 4: Couldn't delete Check %s, Error: %w", d.Id(), err)
	}
	return nil
}

func resourceDataFromTracerouteMonitor(c *checkly.TracerouteMonitor, d *schema.ResourceData) error {
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

	err := d.Set("request", setFromTracerouteRequest(c.Request))
	if err != nil {
		return fmt.Errorf("error setting request for resource %s: %w", d.Id(), err)
	}
	d.Set("group_id", c.GroupID)
	d.Set("group_order", c.GroupOrder)
	// Traceroute monitors do not support private locations, so none is set here.
	d.Set(alertChannelSubscriptionAttributeName, setFromAlertChannelSubscriptions(c.AlertChannelSubscriptions))
	d.Set(retryStrategyAttributeName, listFromRetryStrategy(c.RetryStrategy))
	d.Set("trigger_incident", setFromTriggerIncident(c.TriggerIncident))
	d.SetId(d.Id())
	return nil
}

func setFromTracerouteRequest(r checkly.TracerouteRequest) []tfMap {
	s := tfMap{}
	s["url"] = r.URL
	s["protocol"] = r.Protocol
	s["port"] = r.Port
	s["ip_family"] = r.IPFamily
	s["max_hops"] = r.MaxHops
	s["max_unknown_hops"] = r.MaxUnknownHops
	// PtrLookup is a pointer in the SDK so an explicit false can be sent; the
	// API defaults it to true, so a nil response flattens back to true.
	if r.PtrLookup != nil {
		s["ptr_lookup"] = *r.PtrLookup
	} else {
		s["ptr_lookup"] = true
	}
	s["timeout"] = r.Timeout
	s["assertion"] = setFromAssertions(r.Assertions)
	return []tfMap{s}
}

func tracerouteCheckFromResourceData(d *schema.ResourceData) (checkly.TracerouteMonitor, error) {
	monitor := checkly.TracerouteMonitor{
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

	// Traceroute monitors do not support private locations: the public API
	// marks `privateLocations` forbidden, so the field is omitted entirely.

	monitor.Request = tracerouteRequestFromList(d.Get("request").([]any))

	// port and max_unknown_hops are server-derived when the config does not
	// set them: the API derives port from protocol (443 for TCP, 33434 for
	// UDP/SCTP, no port for ICMP) and max_unknown_hops from max_hops
	// (min(15, max_hops)). Both attributes are Optional+Computed, so on update
	// d.Get returns the value the server derived for the *previous*
	// configuration. Re-sending it after a protocol or max_hops change would
	// probe a port that does not match the new protocol, or violate the API's
	// `max_unknown_hops <= max_hops` rule. The update endpoint is PATCH-style
	// (omitted fields keep their stored value), so omitting the stale value is
	// not enough — instead, when the field is absent from the config, mirror
	// the server's derivation rules here so every write carries the value the
	// server would have chosen for the *current* configuration.
	rawConfig := d.GetRawConfig()
	if !rawConfig.IsNull() {
		if requestIt := rawConfig.GetAttr("request").ElementIterator(); requestIt.Next() {
			_, requestAttr := requestIt.Element()
			if requestAttr.GetAttr("port").IsNull() {
				switch monitor.Request.Protocol {
				case "UDP", "SCTP":
					monitor.Request.Port = 33434
				case "ICMP":
					// ICMP probes have no port; the zero value is dropped
					// from the marshaled payload.
					monitor.Request.Port = 0
				default:
					monitor.Request.Port = 443
				}
			}
			if requestAttr.GetAttr("max_unknown_hops").IsNull() {
				monitor.Request.MaxUnknownHops = 15
				if monitor.Request.MaxHops < 15 {
					monitor.Request.MaxUnknownHops = monitor.Request.MaxHops
				}
			}
		}
	}

	monitor.FrequencyOffset = d.Get(frequencyOffsetAttributeName).(int)

	return monitor, nil
}

func tracerouteRequestFromList(s []any) checkly.TracerouteRequest {
	if len(s) == 0 || s[0] == nil {
		return checkly.TracerouteRequest{}
	}
	res := s[0].(tfMap)
	// PtrLookup is sent as an explicit pointer so the schema's `false` reaches the
	// wire; the SDK's MarshalJSON drops `port` when protocol is "ICMP".
	ptrLookup := res["ptr_lookup"].(bool)
	return checkly.TracerouteRequest{
		URL:            res["url"].(string),
		Protocol:       res["protocol"].(string),
		Port:           res["port"].(int),
		IPFamily:       res["ip_family"].(string),
		MaxHops:        res["max_hops"].(int),
		MaxUnknownHops: res["max_unknown_hops"].(int),
		PtrLookup:      &ptrLookup,
		Timeout:        res["timeout"].(int),
		Assertions:     assertionsFromSet(res["assertion"].(*schema.Set)),
	}
}
