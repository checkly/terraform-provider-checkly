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

func resourceICMPMonitor() *schema.Resource {
	return &schema.Resource{
		Create: resourceICMPMonitorCreate,
		Read:   resourceICMPMonitorRead,
		Update: resourceICMPMonitorUpdate,
		Delete: resourceICMPMonitorDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "Creates an ICMP Monitor to check host availability using ping.",
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
			"degraded_packet_loss_threshold": {
				Description:  "The packet loss percentage where the monitor should be considered degraded. Possible values are between `0` and `100`. (Default `10`).",
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      10,
				ValidateFunc: validateBetween(0, 100),
			},
			"max_packet_loss_threshold": {
				Description:  "The packet loss percentage where the monitor should be considered failing. Possible values are between `0` and `100`. (Default `20`).",
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      20,
				ValidateFunc: validateBetween(0, 100),
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
				Description: "The parameters of the ICMP request.",
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"hostname": {
							Description: "The hostname to ping.",
							Type:        schema.TypeString,
							Required:    true,
						},
						"ip_family": {
							Description:  "The IP family to use. Possible values are `IPv4` and `IPv6`. (Default `IPv4`).",
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "IPv4",
							ValidateFunc: validateOneOf([]string{"IPv4", "IPv6"}),
						},
						"ping_count": {
							Description:  "The number of ping packets to send. (Default `10`).",
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      10,
							ValidateFunc: validateBetween(1, 50),
						},
						"assertion": {
							Description: "Assertions to validate the ICMP response.",
							Type:        schema.TypeSet,
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"source": {
										Description:  "The source of the asserted value. Possible values are `LATENCY` and `JSON_RESPONSE`.",
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validateOneOf([]string{"LATENCY", "JSON_RESPONSE"}),
									},
									"property": {
										Description: "The property of the source to assert. For `LATENCY` source, possible values are `avg`, `min`, `max` and `stdDev`.",
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
			retryStrategyAttributeName: retryStrategyAttributeSchema,
			"trigger_incident":         triggerIncidentAttributeSchema,
		},
		CustomizeDiff: customdiff.Sequence(
			RetryStrategyCustomizeDiff,
			FrequencyOffsetCustomizeDiff,
		),
	}
}

func resourceICMPMonitorCreate(d *schema.ResourceData, client interface{}) error {
	monitor, err := icmpMonitorFromResourceData(d)
	if err != nil {
		return fmt.Errorf("translation error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()

	newMonitor, err := client.(checkly.Client).CreateICMPMonitor(ctx, monitor)
	if err != nil {
		return fmt.Errorf("failed to create ICMP monitor: %w", err)
	}

	d.SetId(newMonitor.ID)

	return resourceICMPMonitorRead(d, client)
}

func resourceICMPMonitorRead(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()

	monitor, err := client.(checkly.Client).GetICMPMonitor(ctx, d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("failed to retrieve ICMP monitor '%s': %w", d.Id(), err)
	}

	return resourceDataFromICMPMonitor(monitor, d)
}

func resourceICMPMonitorUpdate(d *schema.ResourceData, client interface{}) error {
	monitor, err := icmpMonitorFromResourceData(d)
	if err != nil {
		return fmt.Errorf("translation error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()

	_, err = client.(checkly.Client).UpdateICMPMonitor(ctx, monitor.ID, monitor)
	if err != nil {
		return fmt.Errorf("failed to update ICMP monitor '%s': %w", d.Id(), err)
	}

	d.SetId(monitor.ID)

	return resourceICMPMonitorRead(d, client)
}

func resourceICMPMonitorDelete(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()

	if err := client.(checkly.Client).DeleteICMPMonitor(ctx, d.Id()); err != nil {
		return fmt.Errorf("failed to delete ICMP monitor '%s': %w", d.Id(), err)
	}

	return nil
}

func resourceDataFromICMPMonitor(c *checkly.ICMPMonitor, d *schema.ResourceData) error {
	d.Set("name", c.Name)
	d.Set("activated", c.Activated)
	d.Set("muted", c.Muted)
	d.Set("run_parallel", c.RunParallel)
	d.Set("locations", c.Locations)
	d.Set("degraded_packet_loss_threshold", c.DegradedPacketLossThreshold)
	d.Set("max_packet_loss_threshold", c.MaxPacketLossThreshold)

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

	err := d.Set("request", listFromICMPRequest(c.Request))
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

func icmpMonitorFromResourceData(d *schema.ResourceData) (checkly.ICMPMonitor, error) {
	monitor := checkly.ICMPMonitor{
		ID:                          d.Id(),
		Name:                        d.Get("name").(string),
		Frequency:                   d.Get(frequencyAttributeName).(int),
		Activated:                   d.Get("activated").(bool),
		Muted:                       d.Get("muted").(bool),
		RunParallel:                 d.Get("run_parallel").(bool),
		Locations:                   stringsFromSet(d.Get("locations").(*schema.Set)),
		DegradedPacketLossThreshold: d.Get("degraded_packet_loss_threshold").(int),
		MaxPacketLossThreshold:      d.Get("max_packet_loss_threshold").(int),
		Tags:                        stringsFromSet(d.Get("tags").(*schema.Set)),
		UseGlobalAlertSettings:      d.Get("use_global_alert_settings").(bool),
		GroupID:                     int64(d.Get("group_id").(int)),
		GroupOrder:                  d.Get("group_order").(int),
		AlertChannelSubscriptions:   alertChannelSubscriptionsFromSet(d.Get("alert_channel_subscription").([]interface{})),
		RetryStrategy:               retryStrategyFromList(d.Get(retryStrategyAttributeName).([]any)),
		TriggerIncident:             triggerIncidentFromSet(d.Get("trigger_incident").(*schema.Set)),
	}

	alertSettings := alertSettingsFromSet(d.Get("alert_settings").([]interface{}))
	monitor.AlertSettings = &alertSettings

	monitor.Request = icmpRequestFromList(d.Get("request").([]any))

	monitor.FrequencyOffset = d.Get(frequencyOffsetAttributeName).(int)

	return monitor, nil
}

func icmpRequestFromList(s []any) checkly.ICMPRequest {
	if len(s) == 0 {
		return checkly.ICMPRequest{}
	}
	res := s[0].(tfMap)
	return checkly.ICMPRequest{
		Hostname:   res["hostname"].(string),
		IPFamily:   res["ip_family"].(string),
		PingCount:  res["ping_count"].(int),
		Assertions: assertionsFromSet(res["assertion"].(*schema.Set)),
	}
}

func listFromICMPRequest(r checkly.ICMPRequest) []tfMap {
	s := tfMap{}
	s["hostname"] = r.Hostname
	s["ip_family"] = r.IPFamily
	s["ping_count"] = r.PingCount
	s["assertion"] = setFromAssertions(r.Assertions)
	return []tfMap{s}
}
