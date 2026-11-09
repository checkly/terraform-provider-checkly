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

func TestAccEnvVarValueTooLong(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `resource "checkly_environment_variable" "test" {
				key    = "TF_ACC_VALUE_TOO_LONG"
				value  = format("%031745d", 0)
				secret = true
			}`,
			PlanOnly:    true,
			ExpectError: regexp.MustCompile(`at\s+most\s+31744\s+bytes`),
		},
	})
}

func TestAccEnvVarSuccess(t *testing.T) {
	// Environment variable keys are unique per account, so a random suffix
	// avoids collisions between runs.
	key := fmt.Sprintf("TF_ACC_API_URL_%d", acctest.RandInt())
	config := fmt.Sprintf(`resource "checkly_environment_variable" "test" {
		key     = %q
		value   = "https://api.checklyhq.com"
	}`, key)
	accTestCase(t, []resource.TestStep{
		{
			Config: config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_environment_variable.test",
					"key",
					key,
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
	key := fmt.Sprintf("TF_ACC_SECRET_%d", acctest.RandInt())
	accTestCase(t, []resource.TestStep{
		{
			Config: fmt.Sprintf(`resource "checkly_environment_variable" "test" {
				key     = %q
				value   = "https://api.checklyhq.com"
				secret  = true
			}`, key),
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
