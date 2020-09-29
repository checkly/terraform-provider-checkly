package checkly

import (
	"fmt"
	"os"

	"github.com/checkly/checkly-go-sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Provider makes the provider available to Terraform.
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_key": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("CHECKLY_API_KEY", nil),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"checkly_check":       resourceCheck(),
			"checkly_check_group": resourceCheckGroup(),
			"checkly_snippet":     resourceSnippet(),
			//"checkly_environment_variable": resourceEnvironmentVariable(),
		},
		ConfigureFunc: func(r *schema.ResourceData) (interface{}, error) {
			client := checkly.NewClient(r.Get("api_key").(string))
			debugLog := os.Getenv("CHECKLY_DEBUG_LOG")
			if debugLog != "" {
				debugFile, err := os.OpenFile(debugLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
				if err != nil {
					panic(fmt.Sprintf("can't write to debug log file: %v", err))
				}
				client.Debug = debugFile
			}
			return &client, nil
		},
	}
}
