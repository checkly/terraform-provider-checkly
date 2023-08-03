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
			"api_url": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CHECKLY_API_URL", nil),
			},
			"account_id": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CHECKLY_ACCOUNT_ID", nil),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"checkly_check":                resourceCheck(),
			"checkly_heartbeat":            resourceHeartbeat(),
			"checkly_check_group":          resourceCheckGroup(),
			"checkly_snippet":              resourceSnippet(),
			"checkly_dashboard":            resourceDashboard(),
			"checkly_maintenance_windows":  resourceMaintenanceWindow(),
			"checkly_alert_channel":        resourceAlertChannel(),
			"checkly_trigger_check":        resourceTriggerCheck(),
			"checkly_trigger_group":        resourceTriggerGroup(),
			"checkly_environment_variable": resourceEnvironmentVariable(),
			"checkly_private_location":     resourcePrivateLocation(),
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

			apiUrl := ""
			switch v := r.Get("api_url").(type) {
			case string:
				apiUrl = v
			}

			if apiUrl == "" {
				apiUrl = "https://api.checklyhq.com"
			}

			client := checkly.NewClient(
				apiUrl,
				apiKey,
				nil,
				debugOutput,
			)

			accountId := ""
			switch v := r.Get("account_id").(type) {
			case string:
				accountId = v
			}
			if accountId != "" {
				client.SetAccountId(accountId)
			}

			checklyApiSource := os.Getenv("CHECKLY_API_SOURCE")
			if checklyApiSource != "" {
				client.SetChecklySource(checklyApiSource)
			} else {
				client.SetChecklySource("TF")
			}

			return client, nil
		},
	}
}
