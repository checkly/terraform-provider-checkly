package main

import (
	"fmt"
	"net/http"
	"os"

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
			"follow_redirects": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"url": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceCheckCreate(d *schema.ResourceData, client interface{}) error {
	tfLocations := d.Get("locations").(*schema.Set).List()
	locations := make([]string, len(tfLocations))
	for i, tfLoc := range tfLocations {
		locations[i] = tfLoc.(string)
	}

	check := checkly.Check{
		Name:       d.Get("name").(string),
		Type:       d.Get("type").(string),
		Activated:  d.Get("activated").(bool),
		Frequency:  d.Get("frequency").(int),
		ShouldFail: d.Get("should_fail").(bool),
		Locations:  locations,
		Request: checkly.Request{
			Method:          http.MethodGet,
			URL:             d.Get("url").(string),
			FollowRedirects: d.Get("follow_redirects").(bool),
		},
	}
	debugFile, _ := os.Create("/tmp/checkly.log")
	client.(*checkly.Client).Debug = debugFile
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
	d.Set("name", check.Name)
	d.Set("url", check.Request.URL)
	if check.Request.URL == "" {
		panic(fmt.Sprintf("empty URL: %v", check))
	}
	d.Set("type", check.Type)
	d.Set("activated", check.Activated)
	d.Set("frequency", check.Frequency)
	d.Set("locations", check.Locations)
	d.SetId(d.Id())
	return nil
}

func resourceCheckUpdate(d *schema.ResourceData, client interface{}) error {
	return nil
}

func resourceCheckDelete(d *schema.ResourceData, client interface{}) error {
	if err := client.(*checkly.Client).Delete(d.Id()); err != nil {
		return fmt.Errorf("API error: %v", err)
	}
	return nil
}
