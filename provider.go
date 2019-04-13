package main

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// Provider makes the provider available to Terraform.
func Provider() *schema.Provider {
	return &schema.Provider{
		ResourcesMap: map[string]*schema.Resource{
			"uptimerobot_monitor": resourceMonitor(),
		},
	}
}
