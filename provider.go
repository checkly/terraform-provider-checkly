package main

import (
	"github.com/bitfield/checkly"
	"github.com/hashicorp/terraform/helper/schema"
)

// Provider makes the provider available to Terraform.
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("CHECKLY_API_KEY", nil),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"checkly_check": resourceCheck(),
		},
		ConfigureFunc: func(r *schema.ResourceData) (interface{}, error) {
			client := checkly.NewClient(r.Get("api_key").(string))
			return &client, nil
		},
	}
}
