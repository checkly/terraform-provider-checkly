package checkly

import (
	"fmt"
	"io"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/checkly/checkly-go-sdk"
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
			// "checkly_environment_variable": resourceEnvironmentVariable(),
			"checkly_alert_channel": resourceAlertChannel(),
		},
		ConfigureFunc: func(r *schema.ResourceData) (interface{}, error) {
			debugLog := os.Getenv("CHECKLY_DEBUG_LOG")
			var debugOutput io.Writer
			if debugLog != "" {
				debugFile, err := os.OpenFile(debugLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
				if err != nil {
					panic(fmt.Sprintf("can't write to debug log file: %v", err))
				}
				debugOutput = debugFile
			}
			apiKey := ""
			switch v := r.Get("api_key").(type) {
			case string:
				apiKey = v
			}
			client := checkly.NewClient(
				"https://api.checklyhq.com",
				apiKey,
				nil,
				debugOutput,
			)
			return client, nil
		},
	}
}
