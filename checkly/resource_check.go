package checkly

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/checkly/checkly-go-sdk"
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
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "Checks allows you to monitor key webapp flows, backend API's and set up alerting, so you get a notification when things break or slow down.",
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the check.",
			},
			"type": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The type of the check. Possible values are `API`, and `BROWSER`.",
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
				Description: "This property only valid for API high frequency checks. To create a hight frequency check, the property `frequency` must be `0` and `frequency_offset` could be `10`, `20` or `30`.",
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
				Description: "Allows to invert the behaviour of when a check is considered to fail. Allows for validating error status like 404.",
			},
			"locations": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "An array of one or more data center locations where to run the this check. (Default [\"us-east-1\"])",
			},
			"script": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "A valid piece of Node.js JavaScript code describing a browser interaction with the Puppeteer/Playwright framework or a reference to an external JavaScript file.",
			},
			"degraded_response_time": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  15000,
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					// https://checklyhq.com/docs/api-checks/limits/
					v := val.(int)
					if v < 0 || v > 30000 {
						errs = append(errs, fmt.Errorf("%q must be 0-30000 ms, got %d", key, v))
					}
					return warns, errs
				},
				Description: "The response time in milliseconds starting from which a check should be considered degraded. Possible values are between 0 and 30000. (Default `15000`).",
			},
			"max_response_time": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  30000,
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					v := val.(int)
					// https://checklyhq.com/docs/api-checks/limits/
					if v < 0 || v > 30000 {
						errs = append(errs, fmt.Errorf("%q must be 0-30000 ms, got: %d", key, v))
					}
					return warns, errs
				},
				Description: "The response time in milliseconds starting from which a check should be considered failing. Possible values are between 0 and 30000. (Default `30000`).",
			},
			"environment_variables": {
				Type:        schema.TypeMap,
				Optional:    true,
				Deprecated:  "The property `environment_variables` is deprecated and will be removed in a future version. Consider using the new `environment_variable` list.",
				Description: "Key/value pairs for setting environment variables during check execution. These are only relevant for browser checks. Use global environment variables whenever possible.",
			},
			"environment_variable": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Key/value pairs for setting environment variables during check execution, add locked = true to keep value hidden. These are only relevant for browser checks. Use global environment variables whenever possible.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:     schema.TypeString,
							Required: true,
						},
						"value": {
							Type:     schema.TypeString,
							Required: true,
						},
						"locked": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
			},
			"double_check": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Setting this to `true` will trigger a retry when a check fails from the failing region and another, randomly selected region before marking the check as failed.",
			},
			"tags": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "A list of tags for organizing and filtering checks.",
			},
			"ssl_check": {
				Type:        schema.TypeBool,
				Optional:    true,
				Deprecated:  "The property `ssl_check` is deprecated and it's ignored by the Checkly Public API. It will be removed in a future version.",
				Description: "Determines if the SSL certificate should be validated for expiry.",
			},
			"setup_snippet_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "An ID reference to a snippet to use in the setup phase of an API check.",
			},
			"teardown_snippet_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "An ID reference to a snippet to use in the teardown phase of an API check.",
			},
			"local_setup_script": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A valid piece of Node.js code to run in the setup phase.",
			},
			"local_teardown_script": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A valid piece of Node.js code to run in the teardown phase.",
			},
			"runtime_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     nil,
				Description: "The id of the runtime to use for this check.",
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
			"request": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"method": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "GET",
							Description: "The HTTP method to use for this API check. Possible values are `GET`, `POST`, `PUT`, `HEAD`, `DELETE`, `PATCH`. (Default `GET`).",
						},
						"url": {
							Type:     schema.TypeString,
							Required: true,
						},
						"follow_redirects": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"skip_ssl": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"headers": {
							Type:     schema.TypeMap,
							Optional: true,
							Computed: true,
							DefaultFunc: func() (interface{}, error) {
								return []tfMap{}, nil
							},
						},
						"query_parameters": {
							Type:     schema.TypeMap,
							Optional: true,
							Computed: true,
							DefaultFunc: func() (interface{}, error) {
								return []tfMap{}, nil
							},
						},
						"body": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The body of the request.",
						},
						"body_type": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "NONE",
							Description: "The `Content-Type` header of the request. Possible values `NONE`, `JSON`, `FORM`, `RAW`, and `GRAPHQL`.",
							ValidateFunc: func(value interface{}, key string) (warns []string, errs []error) {
								v := value.(string)
								isValid := false
								options := []string{"NONE", "JSON", "FORM", "RAW", "GRAPHQL"}
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
						"assertion": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"source": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "The source of the asserted value. Possible values `STATUS_CODE`, `JSON_BODY`, `HEADERS`, `TEXT_BODY`, and `RESPONSE_TIME`.",
									},
									"property": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"comparison": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "The type of comparison to be executed between expected and actual value of the assertion. Possible values `EQUALS`, `NOT_EQUALS`, `HAS_KEY`, `NOT_HAS_KEY`, `HAS_VALUE`, `NOT_HAS_VALUE`, `IS_EMPTY`, `NOT_EMPTY`, `GREATER_THAN`, `LESS_THAN`, `CONTAINS`, `NOT_CONTAINS`, `IS_NULL`, and `NOT_NULL`.",
									},
									"target": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
							Description: "A request can have multiple assertions.",
						},
						"basic_auth": {
							Type:     schema.TypeSet,
							MaxItems: 1,
							Optional: true,
							Computed: true,
							DefaultFunc: func() (interface{}, error) {
								return []tfMap{}, nil
							},
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
							Description: "Set up HTTP basic authentication (username & password).",
						},
					},
				},
				Description: "An API check might have one request config.",
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
		},
	}
}

func resourceCheckCreate(d *schema.ResourceData, client interface{}) error {
	check, err := checkFromResourceData(d)
	if err != nil {
		return fmt.Errorf("translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	newCheck, err := client.(checkly.Client).Create(ctx, check)

	if err != nil {
		checkJSON, _ := json.Marshal(check)
		return fmt.Errorf("API error 1: %w, Check: %s", err, string(checkJSON))
	}
	d.SetId(newCheck.ID)
	return resourceCheckRead(d, client)
}

func resourceCheckRead(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	check, err := client.(checkly.Client).Get(ctx, d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			//if resource is deleted remotely, then mark it as
			//successfully gone by unsetting it's ID
			d.SetId("")
			return nil
		}
		return fmt.Errorf("API error 2: %w", err)
	}
	return resourceDataFromCheck(check, d)
}

func resourceCheckUpdate(d *schema.ResourceData, client interface{}) error {
	check, err := checkFromResourceData(d)

	if err != nil {
		return fmt.Errorf("translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	_, err = client.(checkly.Client).Update(ctx, check.ID, check)
	if err != nil {
		checkJSON, _ := json.Marshal(check)
		return fmt.Errorf("API error 3: Couldn't update check, Error: %w, \nCheck: %s", err, checkJSON)
	}
	d.SetId(check.ID)
	return resourceCheckRead(d, client)
}

func resourceCheckDelete(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	if err := client.(checkly.Client).Delete(ctx, d.Id()); err != nil {
		return fmt.Errorf("API error 4: Couldn't delete Check %s, Error: %w", d.Id(), err)
	}
	return nil
}

func resourceDataFromCheck(c *checkly.Check, d *schema.ResourceData) error {
	d.Set("name", c.Name)
	d.Set("type", c.Type)
	d.Set("activated", c.Activated)
	d.Set("muted", c.Muted)
	d.Set("should_fail", c.ShouldFail)
	d.Set("locations", c.Locations)
	d.Set("script", c.Script)
	d.Set("degraded_response_time", c.DegradedResponseTime)
	d.Set("max_response_time", c.MaxResponseTime)
	d.Set("double_check", c.DoubleCheck)
	d.Set("setup_snippet_id", c.SetupSnippetID)
	d.Set("teardown_snippet_id", c.TearDownSnippetID)
	d.Set("local_setup_script", c.LocalSetupScript)
	d.Set("local_teardown_script", c.LocalTearDownScript)

	sort.Strings(c.Tags)
	d.Set("tags", c.Tags)

	d.Set("frequency", c.Frequency)
	if c.Frequency == 0 {
		d.Set("frequency_offset", c.FrequencyOffset)
	}

	if c.RuntimeID != nil {
		d.Set("runtime_id", *c.RuntimeID)
	}

	environmentVariables := environmentVariablesFromSet(d.Get("environment_variable").([]interface{}))
	if len(environmentVariables) > 0 {
		d.Set("environment_variable", c.EnvironmentVariables)
	} else if err := d.Set("environment_variables", setFromEnvVars(c.EnvironmentVariables)); err != nil {
		return fmt.Errorf("error setting environment variables for resource %s: %s", d.Id(), err)
	}

	if err := d.Set("alert_settings", setFromAlertSettings(c.AlertSettings)); err != nil {
		return fmt.Errorf("error setting alert settings for resource %s: %w", d.Id(), err)
	}
	d.Set("use_global_alert_settings", c.UseGlobalAlertSettings)

	if c.Type == checkly.TypeAPI {
		err := d.Set("request", setFromRequest(c.Request))
		if err != nil {
			return fmt.Errorf("error setting request for resource %s: %w", d.Id(), err)
		}
	}
	d.Set("group_id", c.GroupID)
	d.Set("group_order", c.GroupOrder)
	d.Set("private_locations", c.PrivateLocations)
	d.Set("alert_channel_subscription", c.AlertChannelSubscriptions)
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
	if as.EscalationType == checkly.RunBased {
		return []tfMap{
			{
				"escalation_type": as.EscalationType,
				"run_based_escalation": []tfMap{
					{
						"failed_run_threshold": as.RunBasedEscalation.FailedRunThreshold,
					},
				},
				"reminders": []tfMap{
					{
						"amount":   as.Reminders.Amount,
						"interval": as.Reminders.Interval,
					},
				},
			},
		}
	} else {
		return []tfMap{
			{
				"escalation_type": as.EscalationType,
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
			},
		}
	}
}

func setFromRequest(r checkly.Request) []tfMap {
	s := tfMap{}
	s["method"] = r.Method
	s["url"] = r.URL
	s["follow_redirects"] = r.FollowRedirects
	s["skip_ssl"] = r.SkipSSL
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

func setFromBasicAuth(b *checkly.BasicAuth) []tfMap {
	if b == nil {
		return []tfMap{}
	}
	return []tfMap{
		{
			"username": b.Username,
			"password": b.Password,
		},
	}
}

func checkFromResourceData(d *schema.ResourceData) (checkly.Check, error) {
	check := checkly.Check{
		ID:                        d.Id(),
		Name:                      d.Get("name").(string),
		Type:                      d.Get("type").(string),
		Frequency:                 d.Get("frequency").(int),
		Activated:                 d.Get("activated").(bool),
		Muted:                     d.Get("muted").(bool),
		ShouldFail:                d.Get("should_fail").(bool),
		Locations:                 stringsFromSet(d.Get("locations").(*schema.Set)),
		Script:                    d.Get("script").(string),
		DegradedResponseTime:      d.Get("degraded_response_time").(int),
		MaxResponseTime:           d.Get("max_response_time").(int),
		DoubleCheck:               d.Get("double_check").(bool),
		Tags:                      stringsFromSet(d.Get("tags").(*schema.Set)),
		SSLCheck:                  d.Get("ssl_check").(bool),
		SetupSnippetID:            int64(d.Get("setup_snippet_id").(int)),
		TearDownSnippetID:         int64(d.Get("teardown_snippet_id").(int)),
		LocalSetupScript:          d.Get("local_setup_script").(string),
		LocalTearDownScript:       d.Get("local_teardown_script").(string),
		AlertSettings:             alertSettingsFromSet(d.Get("alert_settings").(*schema.Set)),
		UseGlobalAlertSettings:    d.Get("use_global_alert_settings").(bool),
		GroupID:                   int64(d.Get("group_id").(int)),
		GroupOrder:                d.Get("group_order").(int),
		AlertChannelSubscriptions: alertChannelSubscriptionsFromSet(d.Get("alert_channel_subscription").([]interface{})),
	}

	runtimeId := d.Get("runtime_id").(string)
	if runtimeId == "" {
		check.RuntimeID = nil
	} else {
		check.RuntimeID = &runtimeId
	}

	environmentVariables, err := getResourceEnvironmentVariables(d)
	if err != nil {
		return checkly.Check{}, err
	}
	check.EnvironmentVariables = environmentVariables

	privateLocations := stringsFromSet(d.Get("private_locations").(*schema.Set))
	check.PrivateLocations = &privateLocations

	if check.Type == checkly.TypeAPI {
		// this will prevent subsequent apply from causing a tf config change in browser checks
		check.Request = requestFromSet(d.Get("request").(*schema.Set))
		check.FrequencyOffset = d.Get("frequency_offset").(int)

		if check.Frequency == 0 && (check.FrequencyOffset != 10 && check.FrequencyOffset != 20 && check.FrequencyOffset != 30) {
			return check, errors.New("when property frequency is 0, frequency_offset must be 10, 20 or 30")
		}
	}

	if check.Type == checkly.TypeBrowser && check.Frequency == 0 {
		return check, errors.New("property frequency could only be 0 for API checks")
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

func basicAuthFromSet(s *schema.Set) *checkly.BasicAuth {
	if s.Len() == 0 {
		return nil
	}
	res := s.List()[0].(tfMap)
	return &checkly.BasicAuth{
		Username: res["username"].(string),
		Password: res["password"].(string),
	}
}

func alertSettingsFromSet(s *schema.Set) checkly.AlertSettings {
	if s.Len() == 0 {
		return checkly.AlertSettings{
			EscalationType: checkly.RunBased,
			RunBasedEscalation: checkly.RunBasedEscalation{
				FailedRunThreshold: 1,
			},
		}
	}
	res := s.List()[0].(tfMap)
	alertSettings := checkly.AlertSettings{
		EscalationType: res["escalation_type"].(string),
		Reminders:      remindersFromSet(res["reminders"].(*schema.Set)),
	}

	if alertSettings.EscalationType == checkly.RunBased {
		alertSettings.RunBasedEscalation = runBasedEscalationFromSet(res["run_based_escalation"].(*schema.Set))
	} else {
		alertSettings.TimeBasedEscalation = timeBasedEscalationFromSet(res["time_based_escalation"].(*schema.Set))
	}

	return alertSettings
}

func alertChannelSubscriptionsFromSet(s []interface{}) []checkly.AlertChannelSubscription {
	res := []checkly.AlertChannelSubscription{}
	if len(s) == 0 {
		return res
	}
	for _, it := range s {
		tm := it.(tfMap)
		chid := tm["channel_id"].(int)
		activated := tm["activated"].(bool)
		res = append(res, checkly.AlertChannelSubscription{
			Activated: activated,
			ChannelID: int64(chid),
		})
	}
	return res
}

func environmentVariablesFromSet(s []interface{}) []checkly.EnvironmentVariable {
	res := []checkly.EnvironmentVariable{}
	if len(s) == 0 {
		return res
	}
	for _, it := range s {
		tm := it.(tfMap)
		key := tm["key"].(string)
		value := tm["value"].(string)
		locked := tm["locked"].(bool)
		res = append(res, checkly.EnvironmentVariable{
			Key:    key,
			Value:  value,
			Locked: locked,
		})
	}

	return res
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

func requestFromSet(s *schema.Set) checkly.Request {
	if s.Len() == 0 {
		return checkly.Request{}
	}
	res := s.List()[0].(tfMap)
	return checkly.Request{
		Method:          res["method"].(string),
		URL:             res["url"].(string),
		FollowRedirects: res["follow_redirects"].(bool),
		SkipSSL:         res["skip_ssl"].(bool),
		Body:            res["body"].(string),
		BodyType:        res["body_type"].(string),
		Headers:         keyValuesFromMap(res["headers"].(tfMap)),
		QueryParameters: keyValuesFromMap(res["query_parameters"].(tfMap)),
		Assertions:      assertionsFromSet(res["assertion"].(*schema.Set)),
		BasicAuth:       basicAuthFromSet(res["basic_auth"].(*schema.Set)),
	}
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

func getResourceEnvironmentVariables(d *schema.ResourceData) ([]checkly.EnvironmentVariable, error) {
	deprecatedEnvironmentVariables := envVarsFromMap(d.Get("environment_variables").(tfMap))
	environmentVariables := environmentVariablesFromSet(d.Get("environment_variable").([]interface{}))

	if len(environmentVariables) > 0 && len(deprecatedEnvironmentVariables) > 0 {
		return nil, errors.New("can't use both \"environment_variables\" and \"environment_variable\" on checkly_check_group resource")
	}

	if len(environmentVariables) > 0 {
		return environmentVariables, nil
	}

	return deprecatedEnvironmentVariables, nil
}
