package main

import (
	"fmt"
	"time"

	"github.com/bitfield/checkly"
	"github.com/hashicorp/terraform/helper/schema"
)

// tfMap is a shorthand alias for convenience; Terraform uses this type a *lot*.
type tfMap = map[string]interface{}

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
			"alert_channels": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"email": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"webhook": {
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
						"slack": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"sms": {
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
					},
				},
			},
			"alert_settings": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"escalation_type": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"run_based_escalation": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"failed_run_threshold": {
										Type:     schema.TypeInt,
										Optional: true,
										Default:  1,
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
										Type:     schema.TypeInt,
										Optional: true,
										Default:  5,
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
										Type:     schema.TypeInt,
										Optional: true,
									},
									"interval": {
										Type:     schema.TypeInt,
										Optional: true,
										Default:  5,
									},
								},
							},
						},
						"ssl_certificates": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"enabled": {
										Type:     schema.TypeBool,
										Optional: true,
										Default:  false,
									},
									"alert_threshold": {
										Type:     schema.TypeInt,
										Optional: true,
										Default:  3,
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
									},
								},
							},
						},
					},
				},
			},
			"use_global_alert_settings": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"request": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"body": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"body_type": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "NONE",
						},
						"headers": {
							Type:     schema.TypeMap,
							Optional: true,
						},
						"query_parameters": {
							Type:     schema.TypeMap,
							Optional: true,
						},
						"follow_redirects": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"method": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "GET",
						},
						"url": {
							Type:     schema.TypeString,
							Required: true,
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
	check, err := checkFromResourceData(d)
	if err != nil {
		return fmt.Errorf("translation error: %v", err)
	}
	err = client.(*checkly.Client).Update(check.ID, check)
	if err != nil {
		return fmt.Errorf("API error: %v", err)
	}
	d.SetId(check.ID)
	return resourceCheckRead(d, client)
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
	d.Set("alert_channels", setFromAlertChannels(c.AlertChannels))
	d.Set("alert_settings", setFromAlertSettings(c.AlertSettings))
	d.Set("use_global_alert_settings", c.UseGlobalAlertSettings)
	d.Set("request", setFromRequest(c.Request))
	d.SetId(d.Id())
	return nil
}

func setFromAlertChannels(alertChannels checkly.AlertChannels) []tfMap {
	var emails = []string{}
	for _, e := range alertChannels.Email {
		emails = append(emails, e.Address)
	}
	var webhooks = []tfMap{}
	for _, w := range alertChannels.Webhook {
		webhooks = append(webhooks, tfMap{
			"name": w.Name,
			"url":  w.URL,
		})
	}
	var slacks = []string{}
	for _, s := range alertChannels.Slack {
		slacks = append(slacks, s.URL)
	}
	var smss = []tfMap{}
	for _, s := range alertChannels.SMS {
		smss = append(smss, tfMap{
			"number": s.Number,
			"name":   s.Name,
		})
	}
	return []tfMap{
		{
			"email":   emails,
			"webhook": webhooks,
			"slack":   slacks,
			"sms":     smss,
		},
	}
}

func setFromAlertSettings(as checkly.AlertSettings) []tfMap {
	var result = tfMap{}
	result["escalation_type"] = as.EscalationType
	result["run_based_escalation"] = setFromRunBasedEscalation(as.RunBasedEscalation)
	result["time_based_escalation"] = setFromTimeBasedEscalation(as.TimeBasedEscalation)
	result["reminders"] = setFromReminders(as.Reminders)
	result["ssl_certificates"] = setFromSSLCertificates(as.SSLCertificates)
	return []tfMap{result}
}

func setFromRunBasedEscalation(r checkly.RunBasedEscalation) []tfMap {
	return []tfMap{
		tfMap{
			"failed_run_threshold": r.FailedRunThreshold,
		},
	}
}

func setFromTimeBasedEscalation(t checkly.TimeBasedEscalation) []tfMap {
	return []tfMap{
		tfMap{
			"minutes_failing_threshold": t.MinutesFailingThreshold,
		},
	}
}

func setFromReminders(r checkly.Reminders) []tfMap {
	return []tfMap{
		tfMap{
			"amount":   r.Amount,
			"interval": r.Interval,
		},
	}
}

func setFromSSLCertificates(s checkly.SSLCertificates) []tfMap {
	return []tfMap{
		tfMap{
			"enabled":         s.Enabled,
			"alert_threshold": s.AlertThreshold,
		},
	}
}

func setFromRequest(r checkly.Request) []tfMap {
	rs := tfMap{}
	rs["url"] = r.URL
	rs["body"] = r.Body
	rs["body_type"] = r.BodyType
	rs["method"] = r.Method
	rs["follow_redirects"] = r.FollowRedirects
	rs["assertion"] = setFromAssertions(r.Assertions)
	return []tfMap{rs}
}

func setFromAssertions(assertions []checkly.Assertion) []tfMap {
	var result = []tfMap{}
	for _, a := range assertions {
		as := tfMap{}
		as["source"] = a.Source
		as["property"] = a.Property
		as["comparison"] = a.Comparison
		as["target"] = a.Target
		result = append(result, as)
	}
	return result
}

func checkFromResourceData(d *schema.ResourceData) (checkly.Check, error) {
	check := checkly.Check{
		ID:                     d.Id(),
		Name:                   d.Get("name").(string),
		Type:                   d.Get("type").(string),
		Frequency:              d.Get("frequency").(int),
		Activated:              d.Get("activated").(bool),
		Muted:                  d.Get("muted").(bool),
		ShouldFail:             d.Get("should_fail").(bool),
		Locations:              stringsFromSet(d.Get("locations").(*schema.Set)),
		Script:                 d.Get("script").(string),
		CreatedAt:              mustParseRFC3339Time(d.Get("created_at").(string)),
		UpdatedAt:              mustParseRFC3339Time(d.Get("created_at").(string)),
		EnvironmentVariables:   envVarsFromMap(d.Get("environment_variables").(map[string]interface{})),
		DoubleCheck:            d.Get("double_check").(bool),
		Tags:                   stringsFromSet(d.Get("tags").(*schema.Set)),
		SSLCheck:               d.Get("ssl_check").(bool),
		SSLCheckDomain:         d.Get("ssl_check_domain").(string),
		SetupSnippetID:         int64(d.Get("setup_snippet_id").(int)),
		TearDownSnippetID:      int64(d.Get("teardown_snippet_id").(int)),
		LocalSetupScript:       d.Get("local_setup_script").(string),
		LocalTearDownScript:    d.Get("local_teardown_script").(string),
		AlertChannels:          alertChannelsFromSet(d.Get("alert_channels").(*schema.Set)),
		AlertSettings:          alertSettingsFromSet(d.Get("alert_settings").(*schema.Set)),
		UseGlobalAlertSettings: d.Get("use_global_alert_settings").(bool),
		Request:                requestFromSet(d.Get("request").(*schema.Set)),
	}
	return check, nil
}

func stringsFromSet(s *schema.Set) []string {
	result := make([]string, s.Len())
	for i, item := range s.List() {
		result[i] = item.(string)
	}
	return result
}

func assertionsFromSet(s *schema.Set) []checkly.Assertion {
	result := make([]checkly.Assertion, s.Len())
	for i, item := range s.List() {
		data := item.(tfMap)
		result[i] = checkly.Assertion{
			Source:     data["source"].(string),
			Property:   data["property"].(string),
			Comparison: data["comparison"].(string),
			Target:     data["target"].(string),
		}
	}
	return result
}

func alertChannelsFromSet(s *schema.Set) checkly.AlertChannels {
	if s.Len() == 0 {
		return checkly.AlertChannels{}
	}
	ac := s.List()[0].(tfMap)
	return checkly.AlertChannels{
		Email:   emailsFromSet(ac["email"].(*schema.Set)),
		Webhook: webhooksFromSet(ac["webhook"].(*schema.Set)),
		Slack:   slacksFromSet(ac["slack"].(*schema.Set)),
		SMS:     smssFromSet(ac["sms"].(*schema.Set)),
	}
}

func emailsFromSet(s *schema.Set) []checkly.AlertEmail {
	result := make([]checkly.AlertEmail, s.Len())
	for i, item := range s.List() {
		data := item.(tfMap)
		result[i] = checkly.AlertEmail{
			Address: data["address"].(string),
		}
	}
	return result
}

func webhooksFromSet(s *schema.Set) []checkly.AlertWebhook {
	result := make([]checkly.AlertWebhook, s.Len())
	for i, item := range s.List() {
		data := item.(tfMap)
		result[i] = checkly.AlertWebhook{
			Name: data["name"].(string),
			URL:  data["url"].(string),
		}
	}
	return result
}

func slacksFromSet(s *schema.Set) []checkly.AlertSlack {
	result := make([]checkly.AlertSlack, s.Len())
	for i, item := range s.List() {
		data := item.(tfMap)
		result[i] = checkly.AlertSlack{
			URL: data["url"].(string),
		}
	}
	return result
}

func smssFromSet(s *schema.Set) []checkly.AlertSMS {
	result := make([]checkly.AlertSMS, s.Len())
	for i, item := range s.List() {
		data := item.(tfMap)
		result[i] = checkly.AlertSMS{
			Number: data["number"].(string),
			Name:   data["name"].(string),
		}
	}
	return result
}

func alertSettingsFromSet(s *schema.Set) checkly.AlertSettings {
	if s.Len() < 1 {
		return checkly.AlertSettings{
			SSLCertificates: checkly.SSLCertificates{
				AlertThreshold: 3,
			},
		}
	}
	asm := s.List()[0].(tfMap)
	return checkly.AlertSettings{
		EscalationType:     asm["escalation_type"].(string),
		RunBasedEscalation: runBasedEscalationFromSet(asm["run_based_escalation"].(*schema.Set)),

		TimeBasedEscalation: timeBasedEscalationFromSet(asm["time_based_escalation"].(*schema.Set)),
		Reminders:           remindersFromSet(asm["reminders"].(*schema.Set)),
		SSLCertificates:     sslCertificatesFromSet(asm["ssl_certificates"].(*schema.Set)),
	}
}

func runBasedEscalationFromSet(s *schema.Set) checkly.RunBasedEscalation {
	if s.Len() < 1 {
		return checkly.RunBasedEscalation{}
	}
	m := s.List()[0].(tfMap)
	return checkly.RunBasedEscalation{
		FailedRunThreshold: m["failed_run_threshold"].(int),
	}
}

func timeBasedEscalationFromSet(s *schema.Set) checkly.TimeBasedEscalation {
	if s.Len() < 1 {
		return checkly.TimeBasedEscalation{}
	}
	m := s.List()[0].(tfMap)
	return checkly.TimeBasedEscalation{
		MinutesFailingThreshold: m["minutes_failing_threshold"].(int),
	}
}

func remindersFromSet(s *schema.Set) checkly.Reminders {
	if s.Len() < 1 {
		return checkly.Reminders{}
	}
	m := s.List()[0].(tfMap)
	return checkly.Reminders{
		Amount:   m["amount"].(int),
		Interval: m["interval"].(int),
	}
}

func sslCertificatesFromSet(s *schema.Set) checkly.SSLCertificates {
	if s.Len() < 1 {
		return checkly.SSLCertificates{}
	}
	m := s.List()[0].(tfMap)
	return checkly.SSLCertificates{
		Enabled:        m["enabled"].(bool),
		AlertThreshold: m["alert_threshold"].(int),
	}
}

func requestFromSet(s *schema.Set) checkly.Request {
	if s.Len() < 1 {
		return checkly.Request{}
	}
	rm := s.List()[0].(tfMap)
	return checkly.Request{
		Method:          rm["method"].(string),
		URL:             rm["url"].(string),
		FollowRedirects: rm["follow_redirects"].(bool),
		Body:            rm["body"].(string),
		BodyType:        rm["body_type"].(string),
		Headers:         keyValuesFromMap(rm["headers"].(tfMap)),
		QueryParameters: keyValuesFromMap(rm["query_parameters"].(tfMap)),
		Assertions:      assertionsFromSet(rm["assertion"].(*schema.Set)),
	}
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
