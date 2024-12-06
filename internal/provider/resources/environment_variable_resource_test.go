package resources_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccEnvVarCheckRequiredFields(t *testing.T) {
	config := `resource "checkly_environment_variable" "test" {}`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),

		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`The argument "key" is required`),
			},
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`The argument "value" is required`),
			},
		},
	})
}

func TestAccEnvVarSuccess(t *testing.T) {
	config := `resource "checkly_environment_variable" "test" {
		key     = "API_URL"
		value   = "https://api.checklyhq.com"
	}`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),

		Steps: []resource.TestStep{
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
		},
	})
}

func TestAccSecretEnvVarSuccess(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),

		Steps: []resource.TestStep{
			{
				Config: `resource "checkly_environment_variable" "test" {
				key     = "SECRET"
				value   = "https://api.checklyhq.com"
				secret  = true
			}`,
			},
		},
	})
}
