package main

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceMonitor() *schema.Resource {
	return &schema.Resource{
		Create: resourceMonitorCreate,
		Read:   resourceMonitorRead,
		Update: resourceMonitorUpdate,
		Delete: resourceMonitorDelete,

		Schema: map[string]*schema.Schema{
			"friendly_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceMonitorCreate(d *schema.ResourceData, m interface{}) error {
        friendlyName := d.Get("friendly_name").(string)
        d.SetId(friendlyName)
        return resourceMonitorRead(d, m)
}

func resourceMonitorRead(d *schema.ResourceData, m interface{}) error {
        return nil
}

func resourceMonitorUpdate(d *schema.ResourceData, m interface{}) error {
        return resourceMonitorRead(d, m)
}

func resourceMonitorDelete(d *schema.ResourceData, m interface{}) error {
        return nil
}
