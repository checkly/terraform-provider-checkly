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

func resourceHeartbeatMonitor() *schema.Resource {
	return &schema.Resource{
		Create: resourceHeartbeatMonitorCreate,
		Read:   resourceHeartbeatMonitorRead,
		Update: resourceHeartbeatMonitorUpdate,
		Delete: resourceHeartbeatMonitorDelete,
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
			alertSettingsAttributeName: makeAlertSettingsAttributeSchema(AlertSettingsAttributeSchemaOptions{
				Monitor:               true,
				EnableSSLCertificates: true,
			}),
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
							Type:        schema.TypeInt,
							Required:    true,
							Description: "How often you expect a ping to the ping URL.",
						},
						"period_unit": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: func(value interface{}, key string) (warns []string, errs []error) {
								v := value.(string)
								isValid := false
								options := []string{"seconds", "minutes", "hours", "days"}
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
							Description: "Possible values `seconds`, `minutes`, `hours` and `days`.",
						},
						"grace": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "How long Checkly should wait before triggering any alerts when a ping does not arrive within the set period.",
						},
						"grace_unit": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: func(value interface{}, key string) (warns []string, errs []error) {
								v := value.(string)
								isValid := false
								options := []string{"seconds", "minutes", "hours", "days"}
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
							Description: "Possible values `seconds`, `minutes`, `hours` and `days`.",
						},
						"ping_token": {
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "Custom token to generate your ping URL. Checkly will expect a ping to `https://ping.checklyhq.com/[PING_TOKEN]`.",
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
			"trigger_incident": triggerIncidentAttributeSchema,
		},
	}
}

func resourceHeartbeatMonitorCreate(d *schema.ResourceData, client interface{}) error {
	monitor, err := heartbeatMonitorFromResourceData(d)

	if err != nil {
		return fmt.Errorf("translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	newMonitor, err := client.(checkly.Client).CreateHeartbeatMonitor(ctx, monitor)

	if err != nil {
		checkJSON, _ := json.Marshal(monitor)
		return fmt.Errorf("API error 1: %w, Check: %s", err, string(checkJSON))
	}
	d.SetId(newMonitor.ID)
	return resourceHeartbeatMonitorRead(d, client)
}

func resourceHeartbeatMonitorRead(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	monitor, err := client.(checkly.Client).GetHeartbeatMonitor(ctx, d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			//if resource is deleted remotely, then mark it as
			//successfully gone by unsetting it's ID
			d.SetId("")
			return nil
		}
		return fmt.Errorf("API error 2: %w", err)
	}
	return resourceDataFromHeartbeatMonitor(monitor, d)
}

func resourceHeartbeatMonitorUpdate(d *schema.ResourceData, client interface{}) error {
	monitor, err := heartbeatMonitorFromResourceData(d)

	if err != nil {
		return fmt.Errorf("translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	_, err = client.(checkly.Client).UpdateHeartbeatMonitor(ctx, monitor.ID, monitor)
	if err != nil {
		checkJSON, _ := json.Marshal(monitor)
		return fmt.Errorf("API error 3: Couldn't update check, Error: %w, \nCheck: %s", err, checkJSON)
	}
	d.SetId(monitor.ID)
	return resourceHeartbeatMonitorRead(d, client)
}

func resourceHeartbeatMonitorDelete(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	if err := client.(checkly.Client).DeleteHeartbeatMonitor(ctx, d.Id()); err != nil {
		return fmt.Errorf("API error 4: Couldn't delete Check %s, Error: %w", d.Id(), err)
	}
	return nil
}

func heartbeatMonitorFromResourceData(d *schema.ResourceData) (checkly.HeartbeatMonitor, error) {
	monitor := checkly.HeartbeatMonitor{
		ID:                        d.Id(),
		Name:                      d.Get("name").(string),
		Activated:                 d.Get("activated").(bool),
		Muted:                     d.Get("muted").(bool),
		Tags:                      stringsFromSet(d.Get("tags").(*schema.Set)),
		AlertSettings:             alertSettingsFromSet(d.Get("alert_settings").([]interface{})),
		UseGlobalAlertSettings:    d.Get("use_global_alert_settings").(bool),
		AlertChannelSubscriptions: alertChannelSubscriptionsFromSet(d.Get("alert_channel_subscription").([]interface{})),
		TriggerIncident:           triggerIncidentFromSet(d.Get("trigger_incident").(*schema.Set)),
	}

	// this will prevent subsequent apply from causing a tf config change in browser checks
	monitor.Heartbeat = heartbeatFromSet(d.Get("heartbeat").(*schema.Set))

	// Period / Grace validation
	periodDaysInHours := 0
	periodHours := 0
	periodMinutes := 0
	periodSseconds := 0
	graceDaysInHours := 0
	graceHours := 0
	graceMinutes := 0
	graceSseconds := 0

	if monitor.Heartbeat.PeriodUnit == "days" {
		periodDaysInHours = monitor.Heartbeat.Period * 24
	} else if monitor.Heartbeat.PeriodUnit == "hours" {
		periodHours = monitor.Heartbeat.Period
	} else if monitor.Heartbeat.PeriodUnit == "minutes" {
		periodMinutes = monitor.Heartbeat.Period
	} else {
		periodSseconds = monitor.Heartbeat.Period
	}

	if monitor.Heartbeat.GraceUnit == "days" {
		graceDaysInHours = monitor.Heartbeat.Grace * 24
	} else if monitor.Heartbeat.GraceUnit == "hours" {
		graceHours = monitor.Heartbeat.Grace
	} else if monitor.Heartbeat.GraceUnit == "minutes" {
		graceMinutes = monitor.Heartbeat.Grace
	} else {
		graceSseconds = monitor.Heartbeat.Grace
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
		return monitor, errors.New(fmt.Sprintf("period must be between 30 seconds and 365 days"))
	}

	if addedTimeGrace.Sub(now).Hours()/float64(24) > 365 {
		return monitor, errors.New("grace must be less than 365 days")
	}

	return monitor, nil
}

func resourceDataFromHeartbeatMonitor(c *checkly.HeartbeatMonitor, d *schema.ResourceData) error {
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
	d.Set("trigger_incident", setFromTriggerIncident(c.TriggerIncident))
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
