package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccEnvVarCheckRequiredFields(t *testing.T) {
	config := `resource "checkly_environment_variable" "test" {}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "key" is required`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "value" is required`),
		},
	})
}

func TestAccEnvVarSuccess(t *testing.T) {
	config := `resource "checkly_environment_variable" "test" {
		key     = "API_URL"
		value   = "https://api.checklyhq.com"
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config: config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_environment_variable.test",
					"key",
					"API_URL",
				),
				resource.TestCheckResourceAttr(
					"checkly_environment_variable.test",
					"value",
					"https://api.checklyhq.com",
				),
			),
		},
		{
			ResourceName:      "checkly_environment_variable.test",
			ImportState:       true,
			ImportStateVerify: true,
		},
	})
}

func TestAccSecretEnvVarSuccess(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `resource "checkly_environment_variable" "test" {
				key     = "SECRET"
				value   = "https://api.checklyhq.com"
				secret  = true
			}`,
		},
		{
			ResourceName:      "checkly_environment_variable.test",
			ImportState:       true,
			ImportStateVerify: true,
			// The API never returns the value of a secret environment variable.
			ImportStateVerifyIgnore: []string{"value"},
		},
	})
}
