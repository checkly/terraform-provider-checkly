package checkly

import (
	"context"
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
		Description: "Creates a Traceroute Monitor to map network paths and monitor routing changes.",
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
				Description: "An array of one or more data center locations where to run this monitor.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"degraded_response_time": {
				Description:  "The response time in milliseconds where the monitor should be considered degraded. Possible values are between `0` and `30000`. (Default `15000`).",
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      15000,
				ValidateFunc: validateBetween(0, 30000),
			},
			"max_response_time": {
				Description:  "The response time in milliseconds where the monitor should be considered failing. Possible values are between `0` and `30000`. (Default `30000`).",
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      30000,
				ValidateFunc: validateBetween(0, 30000),
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
				Description: "The parameters of the traceroute request.",
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"hostname": {
							Description: "The hostname to trace.",
							Type:        schema.TypeString,
							Required:    true,
						},
						"port": {
							Description:  "The port to connect to. (Default `443`).",
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      443,
							ValidateFunc: validateBetween(1, 65535),
						},
						"ip_family": {
							Description:  "The IP family to use. Possible values are `IPv4` and `IPv6`. (Default `IPv4`).",
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "IPv4",
							ValidateFunc: validateOneOf([]string{"IPv4", "IPv6"}),
						},
						"max_hops": {
							Description:  "The maximum number of hops to trace. (Default `30`).",
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      30,
							ValidateFunc: validateBetween(1, 64),
						},
						"max_unknown_hops": {
							Description:  "The maximum number of consecutive unknown hops before stopping. (Default `15`).",
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      15,
							ValidateFunc: validateBetween(1, 64),
						},
						"ptr_lookup": {
							Description: "Whether to perform PTR (reverse DNS) lookups for each hop. (Default `true`).",
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
						},
						"timeout": {
							Description:  "The timeout in seconds for each hop probe. (Default `10`).",
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      10,
							ValidateFunc: validateBetween(1, 60),
						},
						"assertion": {
							Description: "Assertions to validate the traceroute response.",
							Type:        schema.TypeSet,
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"source": {
										Description:  "The source of the asserted value. Possible values are `RESPONSE_TIME`, `HOP_COUNT` and `PACKET_LOSS`.",
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validateOneOf([]string{"RESPONSE_TIME", "HOP_COUNT", "PACKET_LOSS"}),
									},
									"property": {
										Description: "The property of the source to assert. For `RESPONSE_TIME` source, possible values are `avg`, `min`, `max` and `stdDev`.",
										Type:        schema.TypeString,
										Optional:    true,
									},
									"comparison": {
										Description:  "The type of comparison to be executed between expected and actual value of the assertion. Possible values are `EQUALS`, `NOT_EQUALS`, `GREATER_THAN` and `LESS_THAN`.",
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validateOneOf([]string{"EQUALS", "NOT_EQUALS", "GREATER_THAN", "LESS_THAN"}),
									},
									"target": {
										Description: "The target value for the assertion.",
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
			retryStrategyAttributeName: makeRetryStrategyAttributeSchema(RetryStrategyAttributeSchemaOptions{}),
			"trigger_incident":         triggerIncidentAttributeSchema,
		},
		CustomizeDiff: customdiff.Sequence(
			RetryStrategyCustomizeDiff,
			FrequencyOffsetCustomizeDiff,
		),
	}
}

func resourceTracerouteMonitorCreate(d *schema.ResourceData, client interface{}) error {
	monitor, err := tracerouteMonitorFromResourceData(d)
	if err != nil {
		return fmt.Errorf("translation error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()

	newMonitor, err := client.(checkly.Client).CreateTracerouteMonitor(ctx, monitor)
	if err != nil {
		return fmt.Errorf("failed to create traceroute monitor: %w", err)
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
			d.SetId("")
			return nil
		}
		return fmt.Errorf("failed to retrieve traceroute monitor '%s': %w", d.Id(), err)
	}

	return resourceDataFromTracerouteMonitor(monitor, d)
}

func resourceTracerouteMonitorUpdate(d *schema.ResourceData, client interface{}) error {
	monitor, err := tracerouteMonitorFromResourceData(d)
	if err != nil {
		return fmt.Errorf("translation error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()

	_, err = client.(checkly.Client).UpdateTracerouteMonitor(ctx, monitor.ID, monitor)
	if err != nil {
		return fmt.Errorf("failed to update traceroute monitor '%s': %w", d.Id(), err)
	}

	d.SetId(monitor.ID)

	return resourceTracerouteMonitorRead(d, client)
}

func resourceTracerouteMonitorDelete(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()

	if err := client.(checkly.Client).DeleteTracerouteMonitor(ctx, d.Id()); err != nil {
		return fmt.Errorf("failed to delete traceroute monitor '%s': %w", d.Id(), err)
	}

	return nil
}

func resourceDataFromTracerouteMonitor(c *checkly.TracerouteMonitor, d *schema.ResourceData) error {
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

	err := d.Set("request", listFromTracerouteRequest(c.Request))
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

func tracerouteMonitorFromResourceData(d *schema.ResourceData) (checkly.TracerouteMonitor, error) {
	monitor := checkly.TracerouteMonitor{
		ID:                     d.Id(),
		Name:                   d.Get("name").(string),
		Frequency:              d.Get(frequencyAttributeName).(int),
		Activated:              d.Get("activated").(bool),
		Muted:                  d.Get("muted").(bool),
		RunParallel:            d.Get("run_parallel").(bool),
		Locations:              stringsFromSet(d.Get("locations").(*schema.Set)),
		DegradedResponseTime:   d.Get("degraded_response_time").(int),
		MaxResponseTime:        d.Get("max_response_time").(int),
		Tags:                   stringsFromSet(d.Get("tags").(*schema.Set)),
		UseGlobalAlertSettings: d.Get("use_global_alert_settings").(bool),
		GroupID:                int64(d.Get("group_id").(int)),
		GroupOrder:             d.Get("group_order").(int),
		AlertChannelSubscriptions: alertChannelSubscriptionsFromSet(d.Get("alert_channel_subscription").([]interface{})),
		RetryStrategy:          retryStrategyFromList(d.Get(retryStrategyAttributeName).([]any)),
		TriggerIncident:        triggerIncidentFromSet(d.Get("trigger_incident").(*schema.Set)),
	}

	alertSettings := alertSettingsFromSet(d.Get("alert_settings").([]interface{}))
	monitor.AlertSettings = &alertSettings

	monitor.Request = tracerouteRequestFromList(d.Get("request").([]any))

	monitor.FrequencyOffset = d.Get(frequencyOffsetAttributeName).(int)

	return monitor, nil
}

func tracerouteRequestFromList(s []any) checkly.TracerouteRequest {
	if len(s) == 0 {
		return checkly.TracerouteRequest{}
	}
	res := s[0].(tfMap)
	ptrLookup := res["ptr_lookup"].(bool)
	return checkly.TracerouteRequest{
		Hostname:       res["hostname"].(string),
		Port:           res["port"].(int),
		IPFamily:       res["ip_family"].(string),
		MaxHops:        res["max_hops"].(int),
		MaxUnknownHops: res["max_unknown_hops"].(int),
		PtrLookup:      &ptrLookup,
		Timeout:        res["timeout"].(int),
		Assertions:     assertionsFromSet(res["assertion"].(*schema.Set)),
	}
}

func listFromTracerouteRequest(r checkly.TracerouteRequest) []tfMap {
	s := tfMap{}
	s["hostname"] = r.Hostname
	s["port"] = r.Port
	s["ip_family"] = r.IPFamily
	s["max_hops"] = r.MaxHops
	s["max_unknown_hops"] = r.MaxUnknownHops
	if r.PtrLookup != nil {
		s["ptr_lookup"] = *r.PtrLookup
	} else {
		s["ptr_lookup"] = true
	}
	s["timeout"] = r.Timeout
	s["assertion"] = setFromAssertions(r.Assertions)
	return []tfMap{s}
}
