package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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

// A resource deleted outside Terraform must drop out of state on refresh, so
// that the next plan recreates it, rather than failing the refresh with the
// API's 404.
func TestAccEnvVarDeletedOutsideTerraform(t *testing.T) {
	const name = "checkly_environment_variable.test"
	key := fmt.Sprintf("TF_ACC_DELETED_%d", acctest.RandInt())
	config := fmt.Sprintf(`resource "checkly_environment_variable" "test" {
		key   = %q
		value = "deleted outside Terraform"
	}`, key)
	accTestCase(t, []resource.TestStep{
		{
			Config: config,
			Check:  resource.TestCheckResourceAttr(name, "key", key),
		},
		{
			PreConfig: func() {
				if err := testAccClient().DeleteEnvironmentVariable(context.Background(), key); err != nil {
					t.Fatalf("deleting environment variable %s outside Terraform: %v", key, err)
				}
			},
			RefreshState:       true,
			ExpectNonEmptyPlan: true,
			Check: func(s *terraform.State) error {
				if _, ok := s.RootModule().Resources[name]; ok {
					return fmt.Errorf("%s is still in state after a refresh", name)
				}
				return nil
			},
		},
	})
}
