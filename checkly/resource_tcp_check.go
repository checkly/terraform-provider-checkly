package checkly

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceTCPCheck() *schema.Resource {
	return &schema.Resource{
		Create: resourceTCPCheckCreate,
		Read:   resourceTCPCheckRead,
		Update: resourceTCPCheckUpdate,
		Delete: resourceTCPCheckDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "TCP checks allow you to monitor remote endpoints at a lower level.",
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the check.",
			},
			"frequency": {
				Type:     schema.TypeInt,
				Required: true,
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					v := val.(int)
					valid := false
					validFreqs := []int{0, 1, 2, 5, 10, 15, 30, 60, 120, 180, 360, 720, 1440}
					for _, i := range validFreqs {
						if v == i {
							valid = true
						}
					}
					if !valid {
						errs = append(errs, fmt.Errorf("%q must be one of %v, got %d", key, validFreqs, v))
					}
					return warns, errs
				},
				Description: "The frequency in minutes to run the check. Possible values are `0`, `1`, `2`, `5`, `10`, `15`, `30`, `60`, `120`, `180`, `360`, `720`, and `1440`.",
			},
			"frequency_offset": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "To create a high frequency check, the property `frequency` must be `0` and `frequency_offset` can be `10`, `20` or `30`.",
			},
			"activated": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Determines if the check is running or not. Possible values `true`, and `false`.",
			},
			"muted": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Determines if any notifications will be sent out when a check fails/degrades/recovers.",
			},
			"should_fail": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Allows to invert the behaviour of when a check is considered to fail.",
			},
			"run_parallel": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Determines if the check should run in all selected locations in parallel or round-robin.",
			},
			"locations": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "An array of one or more data center locations where to run the this check. (Default [\"us-east-1\"])",
			},
			"degraded_response_time": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  4000,
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					v := val.(int)
					if v < 0 || v > 5000 {
						errs = append(errs, fmt.Errorf("%q must be 0-5000 ms, got %d", key, v))
					}
					return warns, errs
				},
				Description: "The response time in milliseconds starting from which a check should be considered degraded. Possible values are between 0 and 5000. (Default `4000`).",
			},
			"max_response_time": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  5000,
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					v := val.(int)
					if v < 0 || v > 5000 {
						errs = append(errs, fmt.Errorf("%q must be 0-5000 ms, got: %d", key, v))
					}
					return warns, errs
				},
				Description: "The response time in milliseconds starting from which a check should be considered failing. Possible values are between 0 and 5000. (Default `5000`).",
			},
			"tags": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "A list of tags for organizing and filtering checks.",
			},
			"runtime_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     nil,
				Description: "The ID of the runtime to use for this check.",
			},
			"alert_channel_subscription": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "An array of channel IDs and whether they're activated or not. If you don't set at least one alert subscription for your check, we won't be able to alert you in case something goes wrong with it.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"channel_id": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"activated": {
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
			},
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
			"alert_settings": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"escalation_type": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Determines what type of escalation to use. Possible values are `RUN_BASED` or `TIME_BASED`.",
						},
						"run_based_escalation": {
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"failed_run_threshold": {
										Type:        schema.TypeInt,
										Optional:    true,
										Description: "After how many failed consecutive check runs an alert notification should be sent. Possible values are between 1 and 5. (Default `1`).",
									},
								},
							},
						},
						"time_based_escalation": {
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"minutes_failing_threshold": {
										Type:        schema.TypeInt,
										Optional:    true,
										Description: "After how many minutes after a check starts failing an alert should be sent. Possible values are `5`, `10`, `15`, and `30`. (Default `5`).",
									},
								},
							},
						},
						"reminders": {
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"amount": {
										Type:        schema.TypeInt,
										Optional:    true,
										Description: "How many reminders to send out after the initial alert notification. Possible values are `0`, `1`, `2`, `3`, `4`, `5`, and `100000`",
									},
									"interval": {
										Type:        schema.TypeInt,
										Optional:    true,
										Default:     5,
										Description: "Possible values are `5`, `10`, `15`, and `30`. (Default `5`).",
									},
								},
							},
						},
						"parallel_run_failure_threshold": {
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"enabled": {
										Type:        schema.TypeBool,
										Optional:    true,
										Default:     false,
										Description: "Applicable only for checks scheduled in parallel in multiple locations.",
									},
									"percentage": {
										Type:        schema.TypeInt,
										Optional:    true,
										Default:     10,
										Description: "Possible values are `10`, `20`, `30`, `40`, `50`, `60`, `70`, `80`, `100`, and `100`. (Default `10`).",
									},
								},
							},
						},
					},
				},
			},
			"use_global_alert_settings": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "When true, the account level alert settings will be used, not the alert setting defined on this check.",
			},
			"request": {
				Type:     schema.TypeSet,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"hostname": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The hostname or IP to connect to. Do not include a scheme or a port in this value.",
						},
						"port": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "The port number to connect to.",
						},
						"data": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The data to send to the target host.",
						},
						"assertion": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"source": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "The source of the asserted value. Possible values are `RESPONSE_DATA` and `RESPONSE_TIME`.",
									},
									"property": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"comparison": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "The type of comparison to be executed between expected and actual value of the assertion. Possible values are `EQUALS`, `NOT_EQUALS`, `HAS_KEY`, `NOT_HAS_KEY`, `HAS_VALUE`, `NOT_HAS_VALUE`, `IS_EMPTY`, `NOT_EMPTY`, `GREATER_THAN`, `LESS_THAN`, `CONTAINS`, `NOT_CONTAINS`, `IS_NULL`, and `NOT_NULL`.",
									},
									"target": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
							Description: "A request can have multiple assertions.",
						},
						"ip_family": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "IPv4",
							Description: "The IP family to use when executing the TCP check. The value can be either `IPv4` or `IPv6`.",
							ValidateFunc: func(value interface{}, key string) (warns []string, errs []error) {
								v := value.(string)
								isValid := false
								options := []string{"IPv4", "IPv6"}
								for _, option := range options {
									if v == option {
										isValid = true
									}
								}
								if !isValid {
									errs = append(errs, fmt.Errorf("%q must be one of %v, got %s", key, options, v))
								}
								return warns, errs
							},
						},
					},
				},
				Description: "The parameters for the TCP connection.",
			},
			"group_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The id of the check group this check is part of.",
			},
			"group_order": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The position of this check in a check group. It determines in what order checks are run when a group is triggered from the API or from CI/CD.",
			},
			"retry_strategy": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				DefaultFunc: func() (interface{}, error) {
					return []tfMap{}, nil
				},
				Description: "A strategy for retrying failed check runs.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Determines which type of retry strategy to use. Possible values are `FIXED`, `LINEAR`, or `EXPONENTIAL`.",
						},
						"base_backoff_seconds": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     60,
							Description: "The number of seconds to wait before the first retry attempt.",
						},
						"max_retries": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     2,
							Description: "The maximum number of times to retry the check. Value must be between 1 and 10.",
						},
						"max_duration_seconds": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     600,
							Description: "The total amount of time to continue retrying the check (maximum 600 seconds).",
						},
						"same_region": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Whether retries should be run in the same region as the initial check run.",
						},
					},
				},
			},
		},
	}
}

func resourceTCPCheckCreate(d *schema.ResourceData, client interface{}) error {
	check, err := tcpCheckFromResourceData(d)
	if err != nil {
		return fmt.Errorf("translation error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	newCheck, err := client.(checkly.Client).CreateTCPCheck(ctx, check)

	if err != nil {
		checkJSON, _ := json.Marshal(check)
		return fmt.Errorf("API error 1: %w, Check: %s", err, string(checkJSON))
	}
	d.SetId(newCheck.ID)
	return resourceTCPCheckRead(d, client)
}

func resourceTCPCheckRead(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	check, err := client.(checkly.Client).GetTCPCheck(ctx, d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			//if resource is deleted remotely, then mark it as
			//successfully gone by unsetting it's ID
			d.SetId("")
			return nil
		}
		return fmt.Errorf("API error 2: %w", err)
	}
	return resourceDataFromTCPCheck(check, d)
}

func resourceTCPCheckUpdate(d *schema.ResourceData, client interface{}) error {
	check, err := tcpCheckFromResourceData(d)

	if err != nil {
		return fmt.Errorf("translation error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	_, err = client.(checkly.Client).UpdateTCPCheck(ctx, check.ID, check)
	if err != nil {
		checkJSON, _ := json.Marshal(check)
		return fmt.Errorf("API error 3: Couldn't update check, Error: %w, \nCheck: %s", err, checkJSON)
	}
	d.SetId(check.ID)
	return resourceTCPCheckRead(d, client)
}

func resourceTCPCheckDelete(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	if err := client.(checkly.Client).DeleteCheck(ctx, d.Id()); err != nil {
		return fmt.Errorf("API error 4: Couldn't delete Check %s, Error: %w", d.Id(), err)
	}
	return nil
}

func resourceDataFromTCPCheck(c *checkly.TCPCheck, d *schema.ResourceData) error {
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

	if c.RuntimeID != nil {
		d.Set("runtime_id", *c.RuntimeID)
	}

	if err := d.Set("alert_settings", setFromAlertSettings(*c.AlertSettings)); err != nil {
		return fmt.Errorf("error setting alert settings for resource %s: %w", d.Id(), err)
	}
	d.Set("use_global_alert_settings", c.UseGlobalAlertSettings)

	err := d.Set("request", setFromTCPRequest(c.Request))
	if err != nil {
		return fmt.Errorf("error setting request for resource %s: %w", d.Id(), err)
	}
	d.Set("group_id", c.GroupID)
	d.Set("group_order", c.GroupOrder)
	d.Set("private_locations", c.PrivateLocations)
	d.Set("alert_channel_subscription", c.AlertChannelSubscriptions)
	d.Set("retry_strategy", setFromRetryStrategy(c.RetryStrategy))
	d.SetId(d.Id())
	return nil
}

func setFromTCPRequest(r checkly.TCPRequest) []tfMap {
	s := tfMap{}
	s["hostname"] = r.Hostname
	s["port"] = r.Port
	s["data"] = r.Data
	s["assertion"] = setFromAssertions(r.Assertions)
	s["ip_family"] = r.IPFamily
	return []tfMap{s}
}

func tcpCheckFromResourceData(d *schema.ResourceData) (checkly.TCPCheck, error) {
	check := checkly.TCPCheck{
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
		RetryStrategy:             retryStrategyFromSet(d.Get("retry_strategy").(*schema.Set)),
	}

	alertSettings := alertSettingsFromSet(d.Get("alert_settings").([]interface{}))
	check.AlertSettings = &alertSettings

	runtimeId := d.Get("runtime_id").(string)
	if runtimeId == "" {
		check.RuntimeID = nil
	} else {
		check.RuntimeID = &runtimeId
	}

	privateLocations := stringsFromSet(d.Get("private_locations").(*schema.Set))
	check.PrivateLocations = &privateLocations

	check.Request = tcpRequestFromSet(d.Get("request").(*schema.Set))

	check.FrequencyOffset = d.Get("frequency_offset").(int)

	if check.Frequency == 0 && (check.FrequencyOffset != 10 && check.FrequencyOffset != 20 && check.FrequencyOffset != 30) {
		return check, errors.New("when property frequency is 0, frequency_offset must be 10, 20 or 30")
	}

	return check, nil
}

func tcpRequestFromSet(s *schema.Set) checkly.TCPRequest {
	if s.Len() == 0 {
		return checkly.TCPRequest{}
	}
	res := s.List()[0].(tfMap)
	return checkly.TCPRequest{
		Hostname:   res["hostname"].(string),
		Port:       uint16(res["port"].(int)),
		Data:       res["data"].(string),
		Assertions: assertionsFromSet(res["assertion"].(*schema.Set)),
		IPFamily:   res["ip_family"].(string),
	}
}
