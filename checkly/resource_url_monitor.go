package checkly

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceURLMonitor() *schema.Resource {
	return &schema.Resource{
		Create: resourceURLMonitorCreate,
		Read:   resourceURLMonitorRead,
		Update: resourceURLMonitorUpdate,
		Delete: resourceURLMonitorDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "Creates a URL Monitor to check HTTP endpoint availability and response times.",
		Schema: map[string]*schema.Schema{
			"name": {
				Description: "The name of the monitor.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"frequency": {
				Description:  "How often the monitor should run in minutes. Possible values are `0`, `1`, `2`, `5`, `10`, `15`, `30`, `60`, `120`, `180`, `360`, `720`, and `1440`.",
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validateOneOf([]int{0, 1, 2, 5, 10, 15, 30, 60, 120, 180, 360, 720, 1440}),
			},
			"frequency_offset": {
				Description:  "To create a high frequency monitor, set `frequency` to `0` and `frequency_offset` to `10`, `20`, or `30`.",
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validateOneOf([]int{10, 20, 30}),
			},
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
			"should_fail": {
				Description: "Allows to invert the behaviour of when the monitor is considered to fail. (Default `false`).",
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
				Description:  "The response time in milliseconds where the monitor should be considered degraded. Possible values are between `0` and `30000`. (Default `3000`).",
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      3000,
				ValidateFunc: validateBetween(0, 30000),
			},
			"max_response_time": {
				Description:  "The response time in milliseconds where the monitor should be considered failing. Possible values are between `0` and `30000`. (Default `5000`).",
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      5000,
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
			"private_locations": {
				Description: "An array of one or more private locations slugs.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				DefaultFunc: func() (interface{}, error) {
					return []tfMap{}, nil
				},
			},
			"alert_settings": {
				Description: "Determines the alert escalation policy for the monitor.",
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"escalation_type": {
							Description:  "Determines what type of escalation to use. Possible values are `RUN_BASED` and `TIME_BASED`.",
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateOneOf([]string{"RUN_BASED", "TIME_BASED"}),
						},
						"run_based_escalation": {
							Description: "Configuration for run-based escalation.",
							Type:        schema.TypeList,
							Optional:    true,
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"failed_run_threshold": {
										Description:  "After how many failed consecutive check runs an alert notification should be sent. Possible values are between `1` and `5`. (Default `1`).",
										Type:         schema.TypeInt,
										Optional:     true,
										Default:      1,
										ValidateFunc: validateBetween(1, 5),
									},
								},
							},
						},
						"time_based_escalation": {
							Description: "Configuration for time-based escalation.",
							Type:        schema.TypeList,
							Optional:    true,
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"minutes_failing_threshold": {
										Description:  "After how many minutes after a monitor starts failing an alert should be sent. Possible values are `5`, `10`, `15`, and `30`. (Default `5`).",
										Type:         schema.TypeInt,
										Optional:     true,
										Default:      5,
										ValidateFunc: validateOneOf([]int{5, 10, 15, 30}),
									},
								},
							},
						},
						"reminders": {
							Description: "Defines how often to send reminder notifications after initial alert.",
							Type:        schema.TypeList,
							Optional:    true,
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"amount": {
										Description:  "Number of reminder notifications to send. Possible values are `0`, `1`, `2`, `3`, `4`, `5`, and `100000` (`0` to disable, `100000` for unlimited). (Default `0`).",
										Type:         schema.TypeInt,
										Optional:     true,
										Default:      0,
										ValidateFunc: validateOneOf([]int{0, 1, 2, 3, 4, 5, 100000}),
									},
									"interval": {
										Description:  "Interval between reminder notifications in minutes. Possible values are `5`, `10`, `15`, and `30`. (Default `5`).",
										Type:         schema.TypeInt,
										Optional:     true,
										Default:      5,
										ValidateFunc: validateOneOf([]int{5, 10, 15, 30}),
									},
								},
							},
						},
						"parallel_run_failure_threshold": {
							Description: "Configuration for parallel run failure threshold.",
							Type:        schema.TypeList,
							Optional:    true,
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"enabled": {
										Description: "Whether parallel run failure threshold is enabled. Applicable only for monitors scheduled in parallel in multiple locations. (Default `false`).",
										Type:        schema.TypeBool,
										Optional:    true,
										Default:     false,
									},
									"percentage": {
										Description:  "Percentage of runs that must fail to trigger alert. Possible values are `10`, `20`, `30`, `40`, `50`, `60`, `70`, `80`, `90`, and `100`. (Default `10`).",
										Type:         schema.TypeInt,
										Optional:     true,
										Default:      10,
										ValidateFunc: validateOneOf([]int{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}),
									},
								},
							},
						},
					},
				},
			},
			"use_global_alert_settings": {
				Description: "When true, the account level alert settings will be used, not the alert setting defined on this monitor. (Default `true`).",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
			},
			"request": {
				Description: "The parameters of the HTTP request.",
				Type:        schema.TypeSet,
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"url": {
							Description: "The URL to monitor. Must be a valid HTTP or HTTPS URL.",
							Type:        schema.TypeString,
							Required:    true,
						},
						"follow_redirects": {
							Description: "Whether to follow HTTP redirects automatically. (Default `true`).",
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
						},
						"skip_ssl": {
							Description: "Whether to skip SSL certificate verification. (Default `false`).",
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
						},
						"assertion": {
							Description: "Assertions to validate the HTTP response. URL monitors only support status code assertions.",
							Type:        schema.TypeSet,
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"source": {
										Description:  "The source of the asserted value. The only allowed value is `STATUS_CODE`.",
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validateOneOf([]string{"STATUS_CODE"}),
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
										Description: "The target value. Typically `200` when the source is `STATUS_CODE`.",
										Type:        schema.TypeString,
										Required:    true,
									},
								},
							},
						},
						"ip_family": {
							Description:  "IP family version to use for the connection. The value can be either `IPv4` or `IPv6`. (Default `IPv4`).",
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "IPv4",
							ValidateFunc: validateOneOf([]string{"IPv4", "IPv6"}),
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

func resourceURLMonitorCreate(d *schema.ResourceData, client interface{}) error {
	check, err := urlMonitorFromResourceData(d)
	if err != nil {
		return fmt.Errorf("translation error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	newCheck, err := client.(checkly.Client).CreateURLMonitor(ctx, check)

	if err != nil {
		checkJSON, _ := json.Marshal(check)
		return fmt.Errorf("API error 1: %w, Check: %s", err, string(checkJSON))
	}
	d.SetId(newCheck.ID)
	return resourceURLMonitorRead(d, client)
}

func resourceURLMonitorRead(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	check, err := client.(checkly.Client).GetURLMonitor(ctx, d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			//if resource is deleted remotely, then mark it as
			//successfully gone by unsetting it's ID
			d.SetId("")
			return nil
		}
		return fmt.Errorf("API error 2: %w", err)
	}
	return resourceDataFromURLMonitor(check, d)
}

func resourceURLMonitorUpdate(d *schema.ResourceData, client interface{}) error {
	check, err := urlMonitorFromResourceData(d)

	if err != nil {
		return fmt.Errorf("translation error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	_, err = client.(checkly.Client).UpdateURLMonitor(ctx, check.ID, check)
	if err != nil {
		checkJSON, _ := json.Marshal(check)
		return fmt.Errorf("API error 3: Couldn't update check, Error: %w, \nCheck: %s", err, checkJSON)
	}
	d.SetId(check.ID)
	return resourceURLMonitorRead(d, client)
}

func resourceURLMonitorDelete(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	if err := client.(checkly.Client).DeleteCheck(ctx, d.Id()); err != nil {
		return fmt.Errorf("API error 4: Couldn't delete Check %s, Error: %w", d.Id(), err)
	}
	return nil
}

func resourceDataFromURLMonitor(c *checkly.URLMonitor, d *schema.ResourceData) error {
	d.Set("name", c.Name)
	d.Set("activated", c.Activated)
	d.Set("muted", c.Muted)
	d.Set("should_fail", c.ShouldFail)
	d.Set("run_parallel", c.RunParallel)
	d.Set("locations", c.Locations)
	d.Set("degraded_response_time", c.DegradedResponseTime)
	d.Set("max_response_time", c.MaxResponseTime)

	sort.Strings(c.Tags)
	d.Set("tags", c.Tags)

	d.Set("frequency", c.Frequency)
	if c.Frequency == 0 {
		d.Set("frequency_offset", c.FrequencyOffset)
	}

	if err := d.Set("alert_settings", setFromAlertSettings(*c.AlertSettings)); err != nil {
		return fmt.Errorf("error setting alert settings for resource %s: %w", d.Id(), err)
	}
	d.Set("use_global_alert_settings", c.UseGlobalAlertSettings)

	err := d.Set("request", setFromURLRequest(c.Request))
	if err != nil {
		return fmt.Errorf("error setting request for resource %s: %w", d.Id(), err)
	}
	d.Set("group_id", c.GroupID)
	d.Set("group_order", c.GroupOrder)
	d.Set("private_locations", c.PrivateLocations)
	d.Set("alert_channel_subscription", c.AlertChannelSubscriptions)
	d.Set(retryStrategyAttributeName, listFromRetryStrategy(c.RetryStrategy))
	d.Set("trigger_incident", setFromTriggerIncident(c.TriggerIncident))
	d.SetId(d.Id())
	return nil
}

func urlMonitorFromResourceData(d *schema.ResourceData) (checkly.URLMonitor, error) {
	check := checkly.URLMonitor{
		ID:                        d.Id(),
		Name:                      d.Get("name").(string),
		Frequency:                 d.Get("frequency").(int),
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
		AlertChannelSubscriptions: alertChannelSubscriptionsFromSet(d.Get("alert_channel_subscription").([]interface{})),
		RetryStrategy:             retryStrategyFromList(d.Get(retryStrategyAttributeName).([]any)),
		TriggerIncident:           triggerIncidentFromSet(d.Get("trigger_incident").(*schema.Set)),
	}

	alertSettings := alertSettingsFromSet(d.Get("alert_settings").([]interface{}))
	check.AlertSettings = &alertSettings

	privateLocations := stringsFromSet(d.Get("private_locations").(*schema.Set))
	check.PrivateLocations = &privateLocations

	check.Request = urlRequestFromSet(d.Get("request").(*schema.Set))

	check.FrequencyOffset = d.Get("frequency_offset").(int)

	if check.Frequency == 0 && (check.FrequencyOffset != 10 && check.FrequencyOffset != 20 && check.FrequencyOffset != 30) {
		return check, errors.New("when property frequency is 0, frequency_offset must be 10, 20 or 30")
	}

	return check, nil
}

func urlRequestFromSet(s *schema.Set) checkly.URLRequest {
	if s.Len() == 0 {
		return checkly.URLRequest{}
	}
	res := s.List()[0].(tfMap)
	return checkly.URLRequest{
		URL:             res["url"].(string),
		FollowRedirects: res["follow_redirects"].(bool),
		SkipSSL:         res["skip_ssl"].(bool),
		Assertions:      assertionsFromSet(res["assertion"].(*schema.Set)),
		IPFamily:        res["ip_family"].(string),
	}
}

func setFromURLRequest(r checkly.URLRequest) []tfMap {
	s := tfMap{}
	s["url"] = r.URL
	s["follow_redirects"] = r.FollowRedirects
	s["skip_ssl"] = r.SkipSSL
	s["assertion"] = setFromAssertions(r.Assertions)
	s["ip_family"] = r.IPFamily
	return []tfMap{s}
}
