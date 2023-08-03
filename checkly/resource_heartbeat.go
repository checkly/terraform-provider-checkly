package checkly

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/checkly/checkly-go-sdk"
)

func resourceHeartbeat() *schema.Resource {
	return &schema.Resource{
		Create: resourceHeartbeatCreate,
		Read:   resourceHeartbeatRead,
		Update: resourceHeartbeatUpdate,
		Delete: resourceHeartbeatDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "Heartbeats allows you to monitor your cron jobs and set up alerting, so you get a notification when things break or slow down.",
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the check.",
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
			"tags": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "A list of tags for organizing and filtering checks.",
			},
			"alert_settings": {
				Type:     schema.TypeSet,
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
							Type:     schema.TypeSet,
							Optional: true,
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
							Type:     schema.TypeSet,
							Optional: true,
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
							Type:     schema.TypeSet,
							Optional: true,
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
						"ssl_certificates": {
							Type:       schema.TypeSet,
							Optional:   true,
							Deprecated: "This property is deprecated and it's ignored by the Checkly Public API. It will be removed in a future version.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"enabled": {
										Type:        schema.TypeBool,
										Optional:    true,
										Description: "Determines if alert notifications should be sent for expiring SSL certificates. Possible values `true`, and `false`. (Default `false`).",
									},
									"alert_threshold": {
										Type:     schema.TypeInt,
										Optional: true,
										ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
											v := val.(int)
											valid := false
											validFreqs := []int{3, 7, 14, 30}
											for _, i := range validFreqs {
												if v == i {
													valid = true
												}
											}
											if !valid {
												errs = append(errs, fmt.Errorf("%q must be one of %v, got: %d", key, validFreqs, v))
											}
											return warns, errs
										},
										Description: "How long before SSL certificate expiry to send alerts. Possible values `3`, `7`, `14`, `30`. (Default `3`).",
									},
								},
								Description: "At what interval the reminders should be sent.",
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
			"heartbeat": {
				Type:     schema.TypeSet,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"period": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"period_unit": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: func(value interface{}, key string) (warns []string, errs []error) {
								v := value.(string)
								isValid := false
								options := []string{"seconds", "minutes", "days"}
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
						"grace": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"grace_unit": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: func(value interface{}, key string) (warns []string, errs []error) {
								v := value.(string)
								isValid := false
								options := []string{"seconds", "minutes", "days"}
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
						"ping_token": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
			},
			"alert_channel_subscription": {
				Type:     schema.TypeList,
				Optional: true,
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
		},
	}
}

func resourceHeartbeatCreate(d *schema.ResourceData, client interface{}) error {
	check, err := heartbeatCheckFromResourceData(d)

	if err != nil {
		return fmt.Errorf("translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	newCheck, err := client.(checkly.Client).CreateHeartbeat(ctx, check)

	if err != nil {
		checkJSON, _ := json.Marshal(check)
		return fmt.Errorf("API error 1: %w, Check: %s", err, string(checkJSON))
	}
	d.SetId(newCheck.ID)
	return resourceHeartbeatRead(d, client)
}

func resourceHeartbeatRead(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	check, err := client.(checkly.Client).GetHeartbeatCheck(ctx, d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			//if resource is deleted remotely, then mark it as
			//successfully gone by unsetting it's ID
			d.SetId("")
			return nil
		}
		return fmt.Errorf("API error 2: %w", err)
	}
	return resourceDataFromHeartbeat(check, d)
}

func resourceHeartbeatUpdate(d *schema.ResourceData, client interface{}) error {
	check, err := heartbeatCheckFromResourceData(d)

	if err != nil {
		return fmt.Errorf("translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	_, err = client.(checkly.Client).UpdateHeartbeat(ctx, check.ID, check)
	if err != nil {
		checkJSON, _ := json.Marshal(check)
		return fmt.Errorf("API error 3: Couldn't update check, Error: %w, \nCheck: %s", err, checkJSON)
	}
	d.SetId(check.ID)
	return resourceHeartbeatRead(d, client)
}

func resourceHeartbeatDelete(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	if err := client.(checkly.Client).Delete(ctx, d.Id()); err != nil {
		return fmt.Errorf("API error 4: Couldn't delete Check %s, Error: %w", d.Id(), err)
	}
	return nil
}

func heartbeatCheckFromResourceData(d *schema.ResourceData) (checkly.HeartbeatCheck, error) {
	check := checkly.HeartbeatCheck{
		ID:                        d.Id(),
		Name:                      d.Get("name").(string),
		Activated:                 d.Get("activated").(bool),
		Muted:                     d.Get("muted").(bool),
		Tags:                      stringsFromSet(d.Get("tags").(*schema.Set)),
		AlertSettings:             alertSettingsFromSet(d.Get("alert_settings").(*schema.Set)),
		UseGlobalAlertSettings:    d.Get("use_global_alert_settings").(bool),
		AlertChannelSubscriptions: alertChannelSubscriptionsFromSet(d.Get("alert_channel_subscription").([]interface{})),
	}

	// this will prevent subsequent apply from causing a tf config change in browser checks
	check.Heartbeat = heartbeatFromSet(d.Get("heartbeat").(*schema.Set))

	// Period / Grace validation
	periodDaysInHours := 0
	periodHours := 0
	periodMinutes := 0
	periodSseconds := 0
	graceDaysInHours := 0
	graceHours := 0
	graceMinutes := 0
	graceSseconds := 0

	if check.Heartbeat.PeriodUnit == "days" {
		periodDaysInHours = check.Heartbeat.Period * 24
	} else if check.Heartbeat.PeriodUnit == "hours" {
		periodHours = check.Heartbeat.Period
	} else if check.Heartbeat.PeriodUnit == "minutes" {
		periodMinutes = check.Heartbeat.Period
	} else {
		periodSseconds = check.Heartbeat.Period
	}

	if check.Heartbeat.GraceUnit == "days" {
		graceDaysInHours = check.Heartbeat.Grace * 24
	} else if check.Heartbeat.GraceUnit == "hours" {
		graceHours = check.Heartbeat.Grace
	} else if check.Heartbeat.GraceUnit == "minutes" {
		graceMinutes = check.Heartbeat.Grace
	} else {
		graceSseconds = check.Heartbeat.Grace
	}

	now := time.Now().Local()
	addedTimePeriod := time.Now().Local().Add(
		time.Hour*time.Duration(periodDaysInHours+periodHours) +
			time.Minute*time.Duration(periodMinutes) +
			time.Second*time.Duration(periodSseconds))
	addedTimeGrace := time.Now().Local().Add(
		time.Hour*time.Duration(graceDaysInHours+graceHours) +
			time.Minute*time.Duration(graceMinutes) +
			time.Second*time.Duration(graceSseconds))

	if addedTimePeriod.Sub(now).Hours()/float64(24) > 365 || addedTimePeriod.Sub(now).Seconds() < 30 {
		return check, errors.New(fmt.Sprintf("period must be between 30 seconds and 365 days"))
	}

	if addedTimeGrace.Sub(now).Hours()/float64(24) > 365 {
		return check, errors.New("grace must be less than 365 days")
	}

	return check, nil
}

func resourceDataFromHeartbeat(c *checkly.HeartbeatCheck, d *schema.ResourceData) error {
	d.Set("name", c.Name)
	d.Set("activated", c.Activated)
	d.Set("muted", c.Muted)

	sort.Strings(c.Tags)
	d.Set("tags", c.Tags)
	if err := d.Set("alert_settings", setFromAlertSettings(c.AlertSettings)); err != nil {
		return fmt.Errorf("error setting alert settings for resource %s: %w", d.Id(), err)
	}
	d.Set("use_global_alert_settings", c.UseGlobalAlertSettings)

	err := d.Set("heartbeat", setFromHeartbeat(c.Heartbeat))
	if err != nil {
		return fmt.Errorf("error setting heartbeat for resource %s: %w %v", d.Id(), err, c.Heartbeat)
	}

	d.Set("alert_channel_subscription", c.AlertChannelSubscriptions)
	d.SetId(d.Id())

	return nil
}

func setFromHeartbeat(r checkly.Heartbeat) []tfMap {
	s := tfMap{}
	s["period"] = r.Period
	s["period_unit"] = r.PeriodUnit
	s["grace_unit"] = r.GraceUnit
	s["grace"] = r.Grace
	s["ping_token"] = r.PingToken
	return []tfMap{s}
}

func heartbeatFromSet(s *schema.Set) checkly.Heartbeat {
	if s.Len() == 0 {
		return checkly.Heartbeat{}
	}
	res := s.List()[0].(tfMap)
	return checkly.Heartbeat{
		Period:     res["period"].(int),
		PeriodUnit: res["period_unit"].(string),
		Grace:      res["grace"].(int),
		GraceUnit:  res["grace_unit"].(string),
		PingToken:  res["ping_token"].(string),
	}
}
