package main

import (
	uptimerobot "github.com/bitfield/uptimerobot/pkg"
	"github.com/hashicorp/terraform/helper/schema"
)

// Provider makes the provider available to Terraform.
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("UPTIMEROBOT_API_KEY", nil),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"uptimerobot_monitor": resourceMonitor(),
		},
		ConfigureFunc: func(r *schema.ResourceData) (interface{}, error) {
			client := uptimerobot.New(r.Get("api_key").(string))
			return &client, nil
		},
	}
}
