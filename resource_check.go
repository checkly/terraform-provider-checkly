package main

import (
	"fmt"
	"sort"
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
			"degraded_response_time": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  15000,
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					// Response times are hard capped to 30 seconds according to the web UI.
					// Should/may require degraded response time to be less than max response time.
					// https://checklyhq.com/docs/api-checks/limits/
					v := val.(int)
					if !(v > 0) {
						errs = append(errs, fmt.Errorf("%q must be larger than 0 ms, got: %d", key, v))
					}
					if !(v <= 30000) {
						errs = append(errs, fmt.Errorf("%q must be at most 30000 ms, got: %d", key, v))
					}
					return warns, errs
				},
			},
			"max_response_time": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  30000,
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					v := val.(int)
					// Response times are hard capped to 30 seconds according to the web UI.
					// https://checklyhq.com/docs/api-checks/limits/
					if !(v > 0) {
						errs = append(errs, fmt.Errorf("%q must be larger than 0 ms, got: %d", key, v))
					}
					if !(v <= 30000) {
						errs = append(errs, fmt.Errorf("%q must be at most 30000 ms, got: %d", key, v))
					}
					return warns, errs
				},
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
						"basic_auth": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"username": {
										Type:     schema.TypeString,
										Required: true,
									},
									"password": {
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
	d.Set("degraded_response_time", c.DegradedResponseTime)
	d.Set("max_response_time", c.MaxResponseTime)
	d.Set("created_at", c.CreatedAt.Format(time.RFC3339))
	d.Set("updated_at", c.UpdatedAt.Format(time.RFC3339))
	if err := d.Set("environment_variables", setFromEnvVars(c.EnvironmentVariables)); err != nil {
		return fmt.Errorf("error setting environment variables for resource %s: %s", d.Id(), err)
	}
	d.Set("double_check", c.DoubleCheck)
	sort.Strings(c.Tags)
	d.Set("tags", c.Tags)
	d.Set("ssl_check", c.SSLCheck)
	d.Set("ssl_check_domain", c.SSLCheckDomain)
	d.Set("setup_snippet_id", c.SetupSnippetID)
	d.Set("teardown_snippet_id", c.TearDownSnippetID)
	d.Set("local_setup_script", c.LocalSetupScript)
	d.Set("local_teardown_script", c.LocalTearDownScript)
	if err := d.Set("alert_settings", setFromAlertSettings(c.AlertSettings)); err != nil {
		return fmt.Errorf("error setting alert settings for resource %s: %s", d.Id(), err)
	}
	d.Set("use_global_alert_settings", c.UseGlobalAlertSettings)
	if err := d.Set("request", setFromRequest(c.Request)); err != nil {
		return fmt.Errorf("error setting request for resource %s: %s", d.Id(), err)
	}
	d.SetId(d.Id())
	return nil
}

func setFromEnvVars(evs []checkly.EnvironmentVariable) tfMap {
	var s = tfMap{}
	for _, ev := range evs {
		s[ev.Key] = ev.Value
	}
	return s
}

func setFromAlertSettings(as checkly.AlertSettings) []tfMap {
	return []tfMap{
		{
			"escalation_type": as.EscalationType,
			"run_based_escalation": []tfMap{
				{
					"failed_run_threshold": as.RunBasedEscalation.FailedRunThreshold,
				},
			},
			"time_based_escalation": []tfMap{
				{
					"minutes_failing_threshold": as.TimeBasedEscalation.MinutesFailingThreshold,
				},
			},
			"reminders": []tfMap{
				{
					"amount":   as.Reminders.Amount,
					"interval": as.Reminders.Interval,
				},
			},
			"ssl_certificates": []tfMap{
				{
					"enabled":         as.SSLCertificates.Enabled,
					"alert_threshold": as.SSLCertificates.AlertThreshold,
				},
			},
		},
	}
}

func setFromRequest(r checkly.Request) []tfMap {
	s := tfMap{}
	s["method"] = r.Method
	s["url"] = r.URL
	s["follow_redirects"] = r.FollowRedirects
	s["body"] = r.Body
	s["body_type"] = r.BodyType
	s["headers"] = mapFromKeyValues(r.Headers)
	s["query_parameters"] = mapFromKeyValues(r.QueryParameters)
	s["assertion"] = setFromAssertions(r.Assertions)
	s["basic_auth"] = setFromBasicAuth(r.BasicAuth)
	return []tfMap{s}
}

func setFromAssertions(assertions []checkly.Assertion) []tfMap {
	s := make([]tfMap, len(assertions))
	for i, a := range assertions {
		as := tfMap{}
		as["source"] = a.Source
		as["property"] = a.Property
		as["comparison"] = a.Comparison
		as["target"] = a.Target
		s[i] = as
	}
	return s
}

func mapFromKeyValues(kvs []checkly.KeyValue) tfMap {
	var s = tfMap{}
	for _, item := range kvs {
		s[item.Key] = item.Value
	}
	return s
}

func setFromBasicAuth(b checkly.BasicAuth) []tfMap {
	return []tfMap{
		{
			"username": b.Username,
			"password": b.Password,
		},
	}
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
		DegradedResponseTime:   d.Get("degraded_response_time").(int),
		MaxResponseTime:        d.Get("max_response_time").(int),
		CreatedAt:              mustParseRFC3339Time(d.Get("created_at").(string)),
		UpdatedAt:              mustParseRFC3339Time(d.Get("created_at").(string)),
		EnvironmentVariables:   envVarsFromMap(d.Get("environment_variables").(tfMap)),
		DoubleCheck:            d.Get("double_check").(bool),
		Tags:                   stringsFromSet(d.Get("tags").(*schema.Set)),
		SSLCheck:               d.Get("ssl_check").(bool),
		SSLCheckDomain:         d.Get("ssl_check_domain").(string),
		SetupSnippetID:         int64(d.Get("setup_snippet_id").(int)),
		TearDownSnippetID:      int64(d.Get("teardown_snippet_id").(int)),
		LocalSetupScript:       d.Get("local_setup_script").(string),
		LocalTearDownScript:    d.Get("local_teardown_script").(string),
		AlertSettings:          alertSettingsFromSet(d.Get("alert_settings").(*schema.Set)),
		UseGlobalAlertSettings: d.Get("use_global_alert_settings").(bool),
		Request:                requestFromSet(d.Get("request").(*schema.Set)),
	}
	return check, nil
}

func stringsFromSet(s *schema.Set) []string {
	r := make([]string, s.Len())
	for i, item := range s.List() {
		r[i] = item.(string)
	}
	return r
}

func assertionsFromSet(s *schema.Set) []checkly.Assertion {
	r := make([]checkly.Assertion, s.Len())
	for i, item := range s.List() {
		res := item.(tfMap)
		r[i] = checkly.Assertion{
			Source:     res["source"].(string),
			Property:   res["property"].(string),
			Comparison: res["comparison"].(string),
			Target:     res["target"].(string),
		}
	}
	return r
}

func basicAuthFromSet(s *schema.Set) checkly.BasicAuth {
	if s.Len() == 0 {
		return checkly.BasicAuth{}
	}
	res := s.List()[0].(tfMap)
	return checkly.BasicAuth{
		Username: res["username"].(string),
		Password: res["password"].(string),
	}
}

func alertSettingsFromSet(s *schema.Set) checkly.AlertSettings {
	if s.Len() == 0 {
		return checkly.AlertSettings{
			SSLCertificates: checkly.SSLCertificates{
				AlertThreshold: 3,
			},
		}
	}
	res := s.List()[0].(tfMap)
	return checkly.AlertSettings{
		EscalationType:      res["escalation_type"].(string),
		RunBasedEscalation:  runBasedEscalationFromSet(res["run_based_escalation"].(*schema.Set)),
		TimeBasedEscalation: timeBasedEscalationFromSet(res["time_based_escalation"].(*schema.Set)),
		Reminders:           remindersFromSet(res["reminders"].(*schema.Set)),
		SSLCertificates:     sslCertificatesFromSet(res["ssl_certificates"].(*schema.Set)),
	}
}

func runBasedEscalationFromSet(s *schema.Set) checkly.RunBasedEscalation {
	if s.Len() == 0 {
		return checkly.RunBasedEscalation{}
	}
	res := s.List()[0].(tfMap)
	return checkly.RunBasedEscalation{
		FailedRunThreshold: res["failed_run_threshold"].(int),
	}
}

func timeBasedEscalationFromSet(s *schema.Set) checkly.TimeBasedEscalation {
	if s.Len() == 0 {
		return checkly.TimeBasedEscalation{}
	}
	res := s.List()[0].(tfMap)
	return checkly.TimeBasedEscalation{
		MinutesFailingThreshold: res["minutes_failing_threshold"].(int),
	}
}

func remindersFromSet(s *schema.Set) checkly.Reminders {
	if s.Len() == 0 {
		return checkly.Reminders{}
	}
	res := s.List()[0].(tfMap)
	return checkly.Reminders{
		Amount:   res["amount"].(int),
		Interval: res["interval"].(int),
	}
}

func sslCertificatesFromSet(s *schema.Set) checkly.SSLCertificates {
	if s.Len() == 0 {
		return checkly.SSLCertificates{}
	}
	res := s.List()[0].(tfMap)
	return checkly.SSLCertificates{
		Enabled:        res["enabled"].(bool),
		AlertThreshold: res["alert_threshold"].(int),
	}
}

func requestFromSet(s *schema.Set) checkly.Request {
	if s.Len() == 0 {
		return checkly.Request{}
	}
	res := s.List()[0].(tfMap)
	return checkly.Request{
		Method:          res["method"].(string),
		URL:             res["url"].(string),
		FollowRedirects: res["follow_redirects"].(bool),
		Body:            res["body"].(string),
		BodyType:        res["body_type"].(string),
		Headers:         keyValuesFromMap(res["headers"].(tfMap)),
		QueryParameters: keyValuesFromMap(res["query_parameters"].(tfMap)),
		Assertions:      assertionsFromSet(res["assertion"].(*schema.Set)),
		BasicAuth:       basicAuthFromSet(res["basic_auth"].(*schema.Set)),
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
	r := make([]checkly.EnvironmentVariable, 0, len(m))
	for k, v := range m {
		s, ok := v.(string)
		if !ok {
			panic(fmt.Errorf("could not convert environment variable value %v to string", v))
		}
		r = append(r, checkly.EnvironmentVariable{
			Key:   k,
			Value: s,
		})
	}
	return r
}

func keyValuesFromMap(m map[string]interface{}) []checkly.KeyValue {
	r := make([]checkly.KeyValue, 0, len(m))
	for k, v := range m {
		s, ok := v.(string)
		if !ok {
			panic(fmt.Errorf("could not convert environment variable value %v to string", v))
		}
		r = append(r, checkly.KeyValue{
			Key:   k,
			Value: s,
		})
	}
	return r
}
