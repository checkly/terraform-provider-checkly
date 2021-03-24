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
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"concurrency": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"activated": {
				Type:     schema.TypeBool,
				Required: true,
			},
			"muted": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"locations": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"environment_variables": {
				Type:     schema.TypeMap,
				Optional: true,
			},
			"double_check": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"tags": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"local_setup_script": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"local_teardown_script": {
				Type:     schema.TypeString,
				Optional: true,
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
			"use_global_alert_settings": {
				Type:     schema.TypeBool,
				Optional: true,
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
							Type:     schema.TypeString,
							Required: true,
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
			//todo setupSnippetId, tearDownSnippetId,
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
	if err := d.Set("environment_variables", setFromEnvVars(g.EnvironmentVariables)); err != nil {
		return fmt.Errorf("error setting environment variables for resource %s: %s", d.Id(), err)
	}
	d.Set("double_check", g.DoubleCheck)
	sort.Strings(g.Tags)
	d.Set("tags", g.Tags)
	d.Set("local_setup_script", g.LocalSetupScript)
	d.Set("local_teardown_script", g.LocalTearDownScript)
	if err := d.Set("alert_settings", setFromAlertSettings(g.AlertSettings)); err != nil {
		return fmt.Errorf("error setting alert settings for resource %s: %s", d.Id(), err)
	}
	d.Set("use_global_alert_settings", g.UseGlobalAlertSettings)
	if err := d.Set("api_check_defaults", setFromAPICheckDefaults(g.APICheckDefaults)); err != nil {
		return fmt.Errorf("error setting request for resource %s: %s", d.Id(), err)
	}
	d.Set("alert_channel_subscription", g.AlertChannelSubscriptions)
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
	return checkly.Group{
		ID:                        ID,
		Name:                      d.Get("name").(string),
		Concurrency:               d.Get("concurrency").(int),
		Activated:                 d.Get("activated").(bool),
		Muted:                     d.Get("muted").(bool),
		Locations:                 stringsFromSet(d.Get("locations").(*schema.Set)),
		EnvironmentVariables:      envVarsFromMap(d.Get("environment_variables").(tfMap)),
		DoubleCheck:               d.Get("double_check").(bool),
		Tags:                      stringsFromSet(d.Get("tags").(*schema.Set)),
		LocalSetupScript:          d.Get("local_setup_script").(string),
		LocalTearDownScript:       d.Get("local_teardown_script").(string),
		AlertSettings:             alertSettingsFromSet(d.Get("alert_settings").(*schema.Set)),
		UseGlobalAlertSettings:    d.Get("use_global_alert_settings").(bool),
		APICheckDefaults:          apiCheckDefaultsFromSet(d.Get("api_check_defaults").(*schema.Set)),
		AlertChannelSubscriptions: alertChannelSubscriptionsFromSet(d.Get("alert_channel_subscription").([]interface{})),
	}, nil
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
