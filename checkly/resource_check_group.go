package checkly

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	checkly "github.com/checkly/checkly-go-sdk"
)

func resourceCheckGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceCheckGroupCreate,
		Read:   resourceCheckGroupRead,
		Update: resourceCheckGroupUpdate,
		Delete: resourceCheckGroupDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "Check groups allow  you to group together a set of related checks, which can also share default settings for various attributes.",
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the check group.",
			},
			"concurrency": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Determines how many checks are run concurrently when triggering a check group from CI/CD or through the API.",
			},
			"activated": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Determines if the checks in the group are running or not.",
			},
			"muted": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Determines if any notifications will be sent out when a check in this group fails and/or recovers.",
			},
			"locations": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "An array of one or more data center locations where to run the checks.",
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
				Description: "Tags for organizing and filtering checks.",
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
				Description: "A valid piece of Node.js code to run in the setup phase of an API check in this group.",
			},
			"local_teardown_script": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A valid piece of Node.js code to run in the teardown phase of an API check in this group.",
			},
			"runtime_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     nil,
				Description: "The id of the runtime to use for this group.",
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
							Default:     checkly.RunBased,
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
										Description: "Determines if alert notifications should be sent for expiring SSL certificates.",
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
										Description: "At what moment in time to start alerting on SSL certificates. Possible values `3`, `7`, `14`, `30`. (Default `3`).",
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
				Description: "When true, the account level alert settings will be used, not the alert setting defined on this check group.",
			},
			"api_check_defaults": {
				Type:     schema.TypeSet,
				MaxItems: 1,
				Optional: true,
				Computed: true,
				DefaultFunc: func() (interface{}, error) {
					return []tfMap{
						tfMap{
							"url":              "",
							"headers":          []tfMap{},
							"query_parameters": []tfMap{},
							"basic_auth":       tfMap{},
						}}, nil
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"url": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The base url for this group which you can reference with the `GROUP_BASE_URL` variable in all group checks.",
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
										Required: true,
									},
								},
							},
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
						},
					},
				},
			},
		},
	}
}

func resourceCheckGroupCreate(d *schema.ResourceData, client interface{}) error {
	group, err := checkGroupFromResourceData(d)
	if err != nil {
		return fmt.Errorf("translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	gotGroup, err := client.(checkly.Client).CreateGroup(ctx, group)
	if err != nil {
		return fmt.Errorf("API error11: %w", err)
	}
	d.SetId(fmt.Sprintf("%d", gotGroup.ID))
	return resourceCheckGroupRead(d, client)
}

func resourceCheckGroupRead(d *schema.ResourceData, client interface{}) error {
	ID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return fmt.Errorf("ID %s is not numeric: %w", d.Id(), err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	group, err := client.(checkly.Client).GetGroup(ctx, ID)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			//if resource is deleted remotely, then mark it as
			//successfully gone by unsetting it's ID
			d.SetId("")
			return nil
		}
		return fmt.Errorf("API error12: %w", err)
	}
	return resourceDataFromCheckGroup(group, d)
}

func resourceCheckGroupUpdate(d *schema.ResourceData, client interface{}) error {
	group, err := checkGroupFromResourceData(d)
	if err != nil {
		return fmt.Errorf("translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	_, err = client.(checkly.Client).UpdateGroup(ctx, group.ID, group)
	if err != nil {
		return fmt.Errorf("API error13: %w", err)
	}
	d.SetId(fmt.Sprintf("%d", group.ID))
	return resourceCheckGroupRead(d, client)
}

func resourceCheckGroupDelete(d *schema.ResourceData, client interface{}) error {
	ID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return fmt.Errorf("ID %s is not numeric: %w", d.Id(), err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	if err := client.(checkly.Client).DeleteGroup(ctx, ID); err != nil {
		return fmt.Errorf("API error14: %w", err)
	}
	return nil
}

func resourceDataFromCheckGroup(g *checkly.Group, d *schema.ResourceData) error {
	d.Set("name", g.Name)
	d.Set("concurrency", g.Concurrency)
	d.Set("activated", g.Activated)
	d.Set("muted", g.Muted)
	d.Set("locations", g.Locations)
	d.Set("double_check", g.DoubleCheck)
	d.Set("setup_snippet_id", g.SetupSnippetID)
	d.Set("teardown_snippet_id", g.TearDownSnippetID)
	d.Set("local_setup_script", g.LocalSetupScript)
	d.Set("local_teardown_script", g.LocalTearDownScript)
	d.Set("alert_channel_subscription", g.AlertChannelSubscriptions)
	d.Set("private_locations", g.PrivateLocations)

	sort.Strings(g.Tags)
	d.Set("tags", g.Tags)

	environmentVariables := environmentVariablesFromSet(d.Get("environment_variable").([]interface{}))
	if len(environmentVariables) > 0 {
		d.Set("environment_variable", g.EnvironmentVariables)
	} else if err := d.Set("environment_variables", setFromEnvVars(g.EnvironmentVariables)); err != nil {
		return fmt.Errorf("error setting environment variables for resource %s: %s", d.Id(), err)
	}

	if g.RuntimeID != nil {
		d.Set("runtime_id", *g.RuntimeID)
	}

	if err := d.Set("alert_settings", setFromAlertSettings(g.AlertSettings)); err != nil {
		return fmt.Errorf("error setting alert settings for resource %s: %s", d.Id(), err)
	}
	d.Set("use_global_alert_settings", g.UseGlobalAlertSettings)

	if err := d.Set("api_check_defaults", setFromAPICheckDefaults(g.APICheckDefaults)); err != nil {
		return fmt.Errorf("error setting request for resource %s: %s", d.Id(), err)
	}

	d.SetId(d.Id())
	return nil
}

func checkGroupFromResourceData(d *schema.ResourceData) (checkly.Group, error) {
	ID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		if d.Id() != "" {
			return checkly.Group{}, err
		}
		ID = 0
	}

	group := checkly.Group{
		ID:                        ID,
		Name:                      d.Get("name").(string),
		Concurrency:               d.Get("concurrency").(int),
		Activated:                 d.Get("activated").(bool),
		Muted:                     d.Get("muted").(bool),
		Locations:                 stringsFromSet(d.Get("locations").(*schema.Set)),
		DoubleCheck:               d.Get("double_check").(bool),
		Tags:                      stringsFromSet(d.Get("tags").(*schema.Set)),
		SetupSnippetID:            int64(d.Get("setup_snippet_id").(int)),
		TearDownSnippetID:         int64(d.Get("teardown_snippet_id").(int)),
		LocalSetupScript:          d.Get("local_setup_script").(string),
		LocalTearDownScript:       d.Get("local_teardown_script").(string),
		AlertSettings:             alertSettingsFromSet(d.Get("alert_settings").(*schema.Set)),
		UseGlobalAlertSettings:    d.Get("use_global_alert_settings").(bool),
		APICheckDefaults:          apiCheckDefaultsFromSet(d.Get("api_check_defaults").(*schema.Set)),
		AlertChannelSubscriptions: alertChannelSubscriptionsFromSet(d.Get("alert_channel_subscription").([]interface{})),
	}

	runtimeId := d.Get("runtime_id").(string)
	if runtimeId == "" {
		group.RuntimeID = nil
	} else {
		group.RuntimeID = &runtimeId
	}

	environmentVariables, err := getResourceEnvironmentVariables(d)
	if err != nil {
		return checkly.Group{}, err
	}
	group.EnvironmentVariables = environmentVariables

	privateLocations := stringsFromSet(d.Get("private_locations").(*schema.Set))
	group.PrivateLocations = &privateLocations

	return group, nil
}

func setFromAPICheckDefaults(a checkly.APICheckDefaults) []tfMap {
	s := tfMap{}
	s["url"] = a.BaseURL
	s["headers"] = mapFromKeyValues(a.Headers)
	s["query_parameters"] = mapFromKeyValues(a.QueryParameters)
	s["assertion"] = setFromAssertions(a.Assertions)
	s["basic_auth"] = checkGroupSetFromBasicAuth(a.BasicAuth)
	return []tfMap{s}
}

func apiCheckDefaultsFromSet(s *schema.Set) checkly.APICheckDefaults {
	if s.Len() == 0 {
		return checkly.APICheckDefaults{}
	}
	res := s.List()[0].(tfMap)

	return checkly.APICheckDefaults{
		BaseURL:         res["url"].(string),
		Headers:         keyValuesFromMap(res["headers"].(tfMap)),
		QueryParameters: keyValuesFromMap(res["query_parameters"].(tfMap)),
		Assertions:      assertionsFromSet(res["assertion"].(*schema.Set)),
		BasicAuth:       checkGroupBasicAuthFromSet(res["basic_auth"].(*schema.Set)),
	}
}

func checkGroupSetFromBasicAuth(b checkly.BasicAuth) []tfMap {
	if b.Username == "" && b.Password == "" {
		return []tfMap{}
	}
	return []tfMap{
		{
			"username": b.Username,
			"password": b.Password,
		},
	}
}

func checkGroupBasicAuthFromSet(s *schema.Set) checkly.BasicAuth {
	if s.Len() == 0 {
		return checkly.BasicAuth{
			Username: "",
			Password: "",
		}
	}
	res := s.List()[0].(tfMap)
	return checkly.BasicAuth{
		Username: res["username"].(string),
		Password: res["password"].(string),
	}
}
