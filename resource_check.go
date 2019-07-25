package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/bitfield/checkly"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceCheck() *schema.Resource {
	return &schema.Resource{
		Create: resourceCheckCreate,
		Read:   resourceCheckRead,
		Update: resourceCheckUpdate,
		Delete: resourceCheckDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"frequency": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					v := val.(int)
					valid := false
					validFreqs := []int{1, 5, 10, 15, 30, 60, 720, 1440}
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
			},
			"activated": &schema.Schema{
				Type:     schema.TypeBool,
				Required: true,
			},
			"muted": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"should_fail": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"locations": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"script": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"created_at": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"updated_at": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"environment_variables": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			},
			"double_check": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"tags": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"ssl_check": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"ssl_check_domain": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"setup_snippet_id": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"teardown_snippet_id": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"local_setup_script": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"local_teardown_script": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"alert_email": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"alert_webhook": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"url": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"alert_slack": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"alert_sms": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"number": {
							Type:     schema.TypeString,
							Required: true,
						},
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"alert_escalation_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"failed_run_threshold": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  1,
			},
			"minutes_failing_threshold": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  5,
			},
			"reminders_amount": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"reminders_interval": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  5,
			},
			"ssl_alerts_enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"ssl_alerts_threshold": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  3,
			},
			"use_global_alert_settings": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"request_method": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "GET",
			},
			"request_url": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"follow_redirects": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"request_body": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"request_body_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "NONE",
			},
			"request_headers": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			},
			"query_parameters": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			},
			"assertion": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"source": {
							Type:     schema.TypeString,
							Required: true,
						},
						"property": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"comparison": {
							Type:     schema.TypeString,
							Required: true,
						},
						"target": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceCheckCreate(d *schema.ResourceData, client interface{}) error {
	check, err := checkFromResourceData(d)
	if err != nil {
		return fmt.Errorf("translation error: %v", err)
	}
	ID, err := client.(*checkly.Client).Create(check)
	if err != nil {
		return fmt.Errorf("API error: %v", err)
	}
	d.SetId(ID)
	return resourceCheckRead(d, client)
}

func resourceCheckRead(d *schema.ResourceData, client interface{}) error {
	check, err := client.(*checkly.Client).Get(d.Id())
	if err != nil {
		return fmt.Errorf("API error: %v", err)
	}
	resourceDataFromCheck(&check, d)
	return nil
}

func resourceCheckUpdate(d *schema.ResourceData, client interface{}) error {
	// needs checkly.Update to be implemented
	return nil
}

func resourceCheckDelete(d *schema.ResourceData, client interface{}) error {
	if err := client.(*checkly.Client).Delete(d.Id()); err != nil {
		return fmt.Errorf("API error: %v", err)
	}
	return nil
}

func resourceDataFromCheck(c *checkly.Check, d *schema.ResourceData) error {
	d.Set("name", c.Name)
	d.Set("type", c.Type)
	d.Set("frequency", c.Frequency)
	d.Set("activated", c.Activated)
	d.Set("muted", c.Muted)
	d.Set("should_fail", c.ShouldFail)
	d.Set("locations", c.Locations)
	d.Set("script", c.Script)
	d.Set("created_at", c.CreatedAt.Format(time.RFC3339))
	d.Set("updated_at", c.UpdatedAt.Format(time.RFC3339))
	d.Set("environment_variables", c.EnvironmentVariables)
	d.Set("double_check", c.DoubleCheck)
	d.Set("tags", c.Tags)
	d.Set("ssl_check", c.SSLCheck)
	d.Set("ssl_check_domain", c.SSLCheckDomain)
	d.Set("setup_snippet_id", c.SetupSnippetID)
	d.Set("teardown_snippet_id", c.TearDownSnippetID)
	d.Set("local_setup_script", c.LocalSetupScript)
	d.Set("local_teardown_script", c.LocalTearDownScript)
	d.Set("alert_email", c.AlertChannels.Email)
	d.Set("alert_webhook", c.AlertChannels.Webhook)
	d.Set("alert_slack", c.AlertChannels.Slack)
	d.Set("alert_sms", c.AlertChannels.SMS)
	d.Set("alert_escalation_type", c.AlertSettings.EscalationType)
	d.Set("failed_run_threshold", c.AlertSettings.RunBasedEscalation.FailedRunThreshold)
	d.Set("minutes_failing_threshold", c.AlertSettings.TimeBasedEscalation.MinutesFailingThreshold)
	d.Set("reminders_amount", c.AlertSettings.Reminders.Amount)
	d.Set("reminders_interval", c.AlertSettings.Reminders.Interval)
	d.Set("ssl_alerts_enabled", c.AlertSettings.SSLCertificates.Enabled)
	d.Set("ssl_alerts_threshold", c.AlertSettings.SSLCertificates.AlertThreshold)
	d.Set("use_global_alert_settings", c.UseGlobalAlertSettings)
	d.Set("request_method", c.Request.Method)
	d.Set("request_url", c.Request.URL)
	d.Set("follow_redirects", c.Request.FollowRedirects)
	d.Set("request_body", c.Request.Body)
	d.Set("request_body_type", c.Request.BodyType)
	d.Set("request_headers", c.Request.Headers)
	d.Set("query_parameters", c.Request.QueryParameters)
	d.Set("assertion", c.Request.Assertions)
	d.SetId(d.Id())
	return nil
}

func checkFromResourceData(d *schema.ResourceData) (checkly.Check, error) {
	check := checkly.Check{
		Name:                 d.Get("name").(string),
		Type:                 d.Get("type").(string),
		Frequency:            d.Get("frequency").(int),
		Activated:            d.Get("activated").(bool),
		Muted:                d.Get("muted").(bool),
		ShouldFail:           d.Get("should_fail").(bool),
		Locations:            stringsFromSet(d.Get("locations").(*schema.Set)),
		Script:               d.Get("script").(string),
		CreatedAt:            mustParseRFC3339Time(d.Get("created_at").(string)),
		UpdatedAt:            mustParseRFC3339Time(d.Get("created_at").(string)),
		EnvironmentVariables: envVarsFromMap(d.Get("environment_variables").(map[string]interface{})),
		DoubleCheck:          d.Get("double_check").(bool),
		Tags:                 stringsFromSet(d.Get("tags").(*schema.Set)),
		SSLCheck:             d.Get("ssl_check").(bool),
		SSLCheckDomain:       d.Get("ssl_check_domain").(string),
		SetupSnippetID:       d.Get("setup_snippet_id").(int64),
		TearDownSnippetID:    d.Get("teardown_snippet_id").(int64),
		LocalSetupScript:     d.Get("local_setup_script").(string),
		LocalTearDownScript:  d.Get("local_teardown_script").(string),
		AlertChannels: checkly.AlertChannels{
			Email:   emailsFromSet(d.Get("alert_email").(*schema.Set)),
			Webhook: webhooksFromSet(d.Get("alert_webhook").(*schema.Set)),
			Slack:   slacksFromSet(d.Get("alert_slack").(*schema.Set)),
			SMS:     smssFromSet(d.Get("alert_sms").(*schema.Set)),
		},
		AlertSettings: checkly.AlertSettings{
			EscalationType: d.Get("alert_escalation_type").(string),
			RunBasedEscalation: checkly.RunBasedEscalation{
				FailedRunThreshold: d.Get("failed_run_threshold").(int),
			},
			TimeBasedEscalation: checkly.TimeBasedEscalation{
				MinutesFailingThreshold: d.Get("minutes_failing_threshold").(int),
			},
			Reminders: checkly.Reminders{
				Amount:   d.Get("reminders_amount").(int),
				Interval: d.Get("reminders_interval").(int),
			},
			SSLCertificates: checkly.SSLCertificates{
				Enabled:        d.Get("ssl_alerts_enabled").(bool),
				AlertThreshold: d.Get("ssl_alerts_threshold").(int),
			},
		},
		UseGlobalAlertSettings: d.Get("use_global_alert_settings").(bool),
		Request: checkly.Request{
			Method:          http.MethodGet,
			URL:             d.Get("request_url").(string),
			FollowRedirects: d.Get("follow_redirects").(bool),
			Body:            d.Get("request_body").(string),
			BodyType:        d.Get("request_body_type").(string),
			Headers:         keyValuesFromMap(d.Get("request_headers").(map[string]interface{})),
			QueryParameters: keyValuesFromMap(d.Get("query_parameters").(map[string]interface{})),
			Assertions:      assertionsFromSet(d.Get("assertion").(*schema.Set)),
		},
	}

	return check, nil
}

func stringsFromSet(s *schema.Set) []string {
	rawSlice := s.List()
	result := make([]string, len(rawSlice))
	for i, item := range rawSlice {
		result[i] = item.(string)
	}
	return result
}

func assertionsFromSet(s *schema.Set) []checkly.Assertion {
	rawSlice := s.List()
	result := make([]checkly.Assertion, len(rawSlice))
	for i, item := range rawSlice {
		data := item.(map[string]interface{})
		result[i] = checkly.Assertion{
			Source:     data["source"].(string),
			Property:   data["property"].(string),
			Comparison: data["comparison"].(string),
			Target:     data["target"].(string),
		}
	}
	return result
}

func emailsFromSet(s *schema.Set) []checkly.AlertEmail {
	rawSlice := s.List()
	result := make([]checkly.AlertEmail, len(rawSlice))
	for i, item := range rawSlice {
		data := item.(map[string]interface{})
		result[i] = checkly.AlertEmail{
			Address: data["address"].(string),
		}
	}
	return result
}

func webhooksFromSet(s *schema.Set) []checkly.AlertWebhook {
	rawSlice := s.List()
	result := make([]checkly.AlertWebhook, len(rawSlice))
	for i, item := range rawSlice {
		data := item.(map[string]interface{})
		result[i] = checkly.AlertWebhook{
			Name: data["name"].(string),
			URL:  data["url"].(string),
		}
	}
	return result
}

func slacksFromSet(s *schema.Set) []checkly.AlertSlack {
	rawSlice := s.List()
	result := make([]checkly.AlertSlack, len(rawSlice))
	for i, item := range rawSlice {
		data := item.(map[string]interface{})
		result[i] = checkly.AlertSlack{
			URL: data["url"].(string),
		}
	}
	return result
}

func smssFromSet(s *schema.Set) []checkly.AlertSMS {
	rawSlice := s.List()
	result := make([]checkly.AlertSMS, len(rawSlice))
	for i, item := range rawSlice {
		data := item.(map[string]interface{})
		result[i] = checkly.AlertSMS{
			Number: data["number"].(string),
			Name:   data["name"].(string),
		}
	}
	return result
}

func mustParseRFC3339Time(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func envVarsFromMap(m map[string]interface{}) []checkly.EnvironmentVariable {
	result := make([]checkly.EnvironmentVariable, 0, len(m))
	for k, v := range m {
		s, ok := v.(string)
		if !ok {
			panic(fmt.Errorf("could not convert environment variable value %v to string", v))
		}
		result = append(result, checkly.EnvironmentVariable{
			Key:   k,
			Value: s,
		})
	}
	return result
}

func keyValuesFromMap(m map[string]interface{}) []checkly.KeyValue {
	result := make([]checkly.KeyValue, 0, len(m))
	for k, v := range m {
		s, ok := v.(string)
		if !ok {
			panic(fmt.Errorf("could not convert environment variable value %v to string", v))
		}
		result = append(result, checkly.KeyValue{
			Key:   k,
			Value: s,
		})
	}
	return result
}
