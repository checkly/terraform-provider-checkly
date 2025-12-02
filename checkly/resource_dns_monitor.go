package checkly

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDNSMonitor() *schema.Resource {
	return &schema.Resource{
		Create: resourceDNSMonitorCreate,
		Read:   resourceDNSMonitorRead,
		Update: resourceDNSMonitorUpdate,
		Delete: resourceDNSMonitorDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "Creates a DNS Monitor to check DNS record availability and response times.",
		Schema: map[string]*schema.Schema{
			"name": {
				Description: "The name of the monitor.",
				Type:        schema.TypeString,
				Required:    true,
			},
			frequencyAttributeName: makeFrequencyAttributeSchema(FrequencyAttributeSchemaOptions{
				Monitor:            true,
				AllowHighFrequency: true,
			}),
			frequencyOffsetAttributeName: makeFrequencyOffsetAttributeSchema(FrequencyOffsetAttributeSchemaOptions{
				Monitor: true,
			}),
			"activated": {
				Description: "Determines whether the monitor will run periodically or not after being deployed.",
				Type:        schema.TypeBool,
				Required:    true,
			},
			"muted": {
				Description: "Determines if any notifications will be sent out when the monitor fails and/or recovers. (Default `false`).",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"run_parallel": {
				Description: "Determines whether the monitor should run on all selected locations in parallel or round-robin. (Default `false`).",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"locations": {
				Description: "An array of one or more data center locations where to run the this monitor.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"degraded_response_time": {
				Description:  "The response time in milliseconds where the monitor should be considered degraded. Possible values are between `0` and `5000`. (Default `500`).",
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      500,
				ValidateFunc: validateBetween(0, 5000),
			},
			"max_response_time": {
				Description:  "The response time in milliseconds where the monitor should be considered failing. Possible values are between `0` and `5000`. (Default `1000`).",
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      1000,
				ValidateFunc: validateBetween(0, 5000),
			},
			"tags": {
				Description: "A list of tags for organizing and filtering checks and monitors.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"alert_channel_subscription": {
				Description: "An array of channel IDs and whether they're activated or not. If you don't set at least one alert subscription for your monitor, we won't be able to alert you.",
				Type:        schema.TypeList,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"channel_id": {
							Description: "The ID of the alert channel.",
							Type:        schema.TypeInt,
							Required:    true,
						},
						"activated": {
							Description: "Whether an alert should be sent to this channel.",
							Type:        schema.TypeBool,
							Required:    true,
						},
					},
				},
			},
			alertSettingsAttributeName: makeAlertSettingsAttributeSchema(AlertSettingsAttributeSchemaOptions{
				Monitor: true,
			}),
			"use_global_alert_settings": {
				Description: "When true, the account level alert settings will be used, not the alert setting defined on this monitor. (Default `true`).",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
			},
			"request": {
				Description: "The parameters of the HTTP request.",
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"record_type": {
							Description:  "The DNS record type. Possible values are `A`, `AAAA`, `CNAME`, `MX`, `NS`, `TXT` and `SOA`.",
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateOneOf([]string{"A", "AAAA", "CNAME", "MX", "NS", "TXT", "SOA"}),
						},
						"query": {
							Description: "The DNS record to query.",
							Type:        schema.TypeString,
							Required:    true,
						},
						"protocol": {
							Description:  "The protocol used to communicate with the name server. Possible values are `UDP` and `TCP`. (Default `UDP`).",
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "UDP",
							ValidateFunc: validateOneOf([]string{"UDP", "TCP"}),
						},
						"name_server": {
							Description: "The name server to use.",
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"host": {
										Description: "The name server host.",
										Type:        schema.TypeString,
										Optional:    true,
										RequiredWith: []string{
											"request.0.name_server.0.port",
										},
									},
									"port": {
										Description: "The name server port.",
										Type:        schema.TypeInt,
										Optional:    true,
										RequiredWith: []string{
											"request.0.name_server.0.host",
										},
									},
								},
							},
						},
						"assertion": {
							Description: "Assertions to validate the HTTP response. DNS monitors only support status code assertions.",
							Type:        schema.TypeSet,
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"source": {
										Description:  "The source of the asserted value. Possible values are `RESPONSE_CODE`, `RESPONSE_TIME`, `TEXT_ANSWER` and `JSON_ANSWER`.",
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validateOneOf([]string{"RESPONSE_CODE", "RESPONSE_TIME", "TEXT_ANSWER", "JSON_ANSWER"}),
									},
									"property": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"comparison": {
										Description:  "The type of comparison to be executed between expected and actual value of the assertion. Possible values are `EQUALS`, `NOT_EQUALS`, `GREATER_THAN` and `LESS_THAN`.",
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validateOneOf([]string{"EQUALS", "NOT_EQUALS", "GREATER_THAN", "LESS_THAN"}),
									},
									"target": {
										Description: "The target value. Typically `NOERROR` when the source is `RESPONSE_CODE`.",
										Type:        schema.TypeString,
										Required:    true,
									},
								},
							},
						},
					},
				},
			},
			"group_id": {
				Description: "The ID of the check group that this monitor is part of.",
				Type:        schema.TypeInt,
				Optional:    true,
			},
			"group_order": {
				Description: "The position of the monitor in the check group. It determines in what order checks and monitors are run when a group is triggered from the API or from CI/CD.",
				Type:        schema.TypeInt,
				Optional:    true,
			},
			retryStrategyAttributeName: retryStrategyAttributeSchema,
			"trigger_incident":         triggerIncidentAttributeSchema,
		},
		CustomizeDiff: customdiff.Sequence(
			RetryStrategyCustomizeDiff,
		),
	}
}

func resourceDNSMonitorCreate(d *schema.ResourceData, client interface{}) error {
	check, err := dnsMonitorFromResourceData(d)
	if err != nil {
		return fmt.Errorf("translation error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()

	newCheck, err := client.(checkly.Client).CreateDNSMonitor(ctx, check)
	if err != nil {
		return fmt.Errorf("failed to create DNS monitor: %w", err)
	}

	d.SetId(newCheck.ID)

	return resourceDNSMonitorRead(d, client)
}

func resourceDNSMonitorRead(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()

	check, err := client.(checkly.Client).GetDNSMonitor(ctx, d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			//if resource is deleted remotely, then mark it as
			//successfully gone by unsetting it's ID
			d.SetId("")
			return nil
		}
		return fmt.Errorf("failed to retrieve DNS monitor '%s': %w", d.Id(), err)
	}

	return resourceDataFromDNSMonitor(check, d)
}

func resourceDNSMonitorUpdate(d *schema.ResourceData, client interface{}) error {
	check, err := dnsMonitorFromResourceData(d)
	if err != nil {
		return fmt.Errorf("translation error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()

	_, err = client.(checkly.Client).UpdateDNSMonitor(ctx, check.ID, check)
	if err != nil {
		return fmt.Errorf("failed to update DNS monitor '%s': %w", d.Id(), err)
	}

	d.SetId(check.ID)

	return resourceDNSMonitorRead(d, client)
}

func resourceDNSMonitorDelete(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()

	if err := client.(checkly.Client).DeleteDNSMonitor(ctx, d.Id()); err != nil {
		return fmt.Errorf("failed to delete DNS monitor '%s': %w", d.Id(), err)
	}

	return nil
}

func resourceDataFromDNSMonitor(c *checkly.DNSMonitor, d *schema.ResourceData) error {
	d.Set("name", c.Name)
	d.Set("activated", c.Activated)
	d.Set("muted", c.Muted)
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

	if err := d.Set("alert_settings", setFromAlertSettings(*c.AlertSettings)); err != nil {
		return fmt.Errorf("error setting alert settings for resource %s: %w", d.Id(), err)
	}
	d.Set("use_global_alert_settings", c.UseGlobalAlertSettings)

	err := d.Set("request", listFromDNSRequest(c.Request))
	if err != nil {
		return fmt.Errorf("error setting request for resource %s: %w", d.Id(), err)
	}
	d.Set("group_id", c.GroupID)
	d.Set("group_order", c.GroupOrder)
	d.Set("alert_channel_subscription", c.AlertChannelSubscriptions)
	d.Set(retryStrategyAttributeName, listFromRetryStrategy(c.RetryStrategy))
	d.Set("trigger_incident", setFromTriggerIncident(c.TriggerIncident))
	d.SetId(d.Id())
	return nil
}

func dnsMonitorFromResourceData(d *schema.ResourceData) (checkly.DNSMonitor, error) {
	check := checkly.DNSMonitor{
		ID:                        d.Id(),
		Name:                      d.Get("name").(string),
		Frequency:                 d.Get(frequencyAttributeName).(int),
		Activated:                 d.Get("activated").(bool),
		Muted:                     d.Get("muted").(bool),
		RunParallel:               d.Get("run_parallel").(bool),
		Locations:                 stringsFromSet(d.Get("locations").(*schema.Set)),
		DegradedResponseTime:      d.Get("degraded_response_time").(int),
		MaxResponseTime:           d.Get("max_response_time").(int),
		Tags:                      stringsFromSet(d.Get("tags").(*schema.Set)),
		UseGlobalAlertSettings:    d.Get("use_global_alert_settings").(bool),
		GroupID:                   int64(d.Get("group_id").(int)),
		GroupOrder:                d.Get("group_order").(int),
		AlertChannelSubscriptions: alertChannelSubscriptionsFromSet(d.Get("alert_channel_subscription").([]interface{})),
		RetryStrategy:             retryStrategyFromList(d.Get(retryStrategyAttributeName).([]any)),
		TriggerIncident:           triggerIncidentFromSet(d.Get("trigger_incident").(*schema.Set)),
	}

	alertSettings := alertSettingsFromSet(d.Get("alert_settings").([]interface{}))
	check.AlertSettings = &alertSettings

	check.Request = dnsRequestFromList(d.Get("request").([]any))

	check.FrequencyOffset = d.Get(frequencyOffsetAttributeName).(int)

	if check.Frequency == 0 && (check.FrequencyOffset != 10 && check.FrequencyOffset != 20 && check.FrequencyOffset != 30) {
		return check, errors.New("when `frequency` is 0, `frequency_offset` must be 10, 20 or 30")
	}

	return check, nil
}

func dnsRequestFromList(s []any) checkly.DNSRequest {
	if len(s) == 0 {
		return checkly.DNSRequest{}
	}
	res := s[0].(tfMap)
	ns := dnsRequestNameServerFromList(res["name_server"].([]any))
	return checkly.DNSRequest{
		RecordType: res["record_type"].(string),
		Query:      res["query"].(string),
		NameServer: ns.Host,
		Port:       ns.Port,
		Protocol:   res["protocol"].(string),
		Assertions: assertionsFromSet(res["assertion"].(*schema.Set)),
	}
}

func listFromDNSRequest(r checkly.DNSRequest) []tfMap {
	s := tfMap{}
	s["record_type"] = r.RecordType
	s["query"] = r.Query
	s["name_server"] = listFromDNSRequestNameServer(dnsRequestNameServer{
		Host: r.NameServer,
		Port: r.Port,
	})
	s["protocol"] = r.Protocol
	s["assertion"] = setFromAssertions(r.Assertions)
	return []tfMap{s}
}

type dnsRequestNameServer struct {
	Host string
	Port int
}

func dnsRequestNameServerFromList(s []any) dnsRequestNameServer {
	if len(s) == 0 {
		return dnsRequestNameServer{}
	}

	res := s[0].(tfMap)

	return dnsRequestNameServer{
		Host: res["host"].(string),
		Port: res["port"].(int),
	}
}

func listFromDNSRequestNameServer(r dnsRequestNameServer) []tfMap {
	if r.Host == "" && r.Port == 0 {
		return []tfMap{}
	}

	s := tfMap{}
	s["host"] = r.Host
	s["port"] = r.Port

	return []tfMap{s}
}
