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
		Delete: resourceCheckDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"url": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"activated": &schema.Schema{
				Type:     schema.TypeBool,
				ForceNew: true,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"locations": &schema.Schema{
				Type:     schema.TypeSet,
				ForceNew: true,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"frequency": &schema.Schema{
				Type:     schema.TypeInt,
				ForceNew: true,
				Required: true,
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					v := val.(int)
					valid := false
					validFreqs := []int{5, 10, 15, 30, 60, 720, 1440}
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
	}
}

func resourceCheckCreate(d *schema.ResourceData, client interface{}) error {
	tfLocations := d.Get("locations").(*schema.Set).List()
	locations := make([]string, len(tfLocations))
	for i, tfLoc := range tfLocations {
		locations[i] = tfLoc.(string)
	}

	check := checkly.Check{
		Name:      d.Get("name").(string),
		Type:      d.Get("type").(string),
		Activated: d.Get("activated").(bool),
		Frequency: d.Get("frequency").(int),
		Locations: locations,
		Request: checkly.Request{
			Method: http.MethodGet,
			URL:    d.Get("url").(string),
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

func resourceCheckDelete(d *schema.ResourceData, client interface{}) error {
	if err := client.(*checkly.Client).Delete(d.Id()); err != nil {
		return fmt.Errorf("API error: %v", err)
	}
	return nil
}
