package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMultistepCheckResourceBasic(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: multistepCheckResource_basic,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"name",
					"Multistep Check Resource",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"runtime_id",
					"2023.09",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"activated",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"script",
					"console.log('test')",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"locations.#",
					"2",
				),
				testCheckResourceAttrExpr(
					"checkly_multistep_check.test",
					"locations.*",
					"us-east-1",
				),
				testCheckResourceAttrExpr(
					"checkly_multistep_check.test",
					"locations.*",
					"eu-central-1",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"tags.#",
					"2",
				),
				testCheckResourceAttrExpr(
					"checkly_multistep_check.test",
					"tags.*",
					"api",
				),
				testCheckResourceAttrExpr(
					"checkly_multistep_check.test",
					"tags.*",
					"multi-step",
				),
			),
		},
	})
}

func TestAccMultistepCheckResourceRuntimeValidation(t *testing.T) {
	unsupportedRuntime := `resource "checkly_multistep_check" "test" {
		name       = "Multistep Check Resource Runtime"
		activated  = true
		frequency  = 5
		locations  = ["eu-central-1"]
		script     = "console.log('test')"
		runtime_id = "2023.02"
	}`
	noSpecifiedRuntime := `resource "checkly_multistep_check" "test" {
		name      = "Multistep Check Resource Runtime"
		activated = true
		frequency = 5
		locations = ["eu-central-1"]
		script    = "console.log('test')"
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      unsupportedRuntime,
			ExpectError: regexp.MustCompile("runtime 2023.02 does not support MULTI_STEP checks"),
		},
		{
			Config: noSpecifiedRuntime,
			Check: resource.TestCheckNoResourceAttr(
				"checkly_multistep_check.test",
				"runtime_id",
			),
		},
	})
}

func TestAccMultistepCheckResourceDependencies(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `resource "checkly_multistep_check" "test" {
				name                      = "Multistep Check Resource Dependencies"
				activated                 = true
				frequency                 = 720
				use_global_alert_settings = true
				locations                 = ["us-east-1"]
				runtime_id                = "2023.09"
				script_path               = "__checks__/main.spec.ts"
				script                    = "console.log('main')"

				dependency {
					path    = "__checks__/util.ts"
					content = "export const foo = 'bar'"
				}

				dependency {
					path    = "__checks__/data.json"
					content = "{\"foo\": \"bar\"}"
				}
			}`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"script_path",
					"__checks__/main.spec.ts",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"dependency.#",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"dependency.0.path",
					"__checks__/util.ts",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"dependency.0.content",
					"export const foo = 'bar'",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"dependency.1.path",
					"__checks__/data.json",
				),
			),
		},
		{
			Config: `resource "checkly_multistep_check" "test" {
				name                      = "Multistep Check Resource Dependencies"
				activated                 = true
				frequency                 = 720
				use_global_alert_settings = true
				locations                 = ["us-east-1"]
				runtime_id                = "2023.09"
				script_path               = "__checks__/main.spec.ts"
				script                    = "console.log('main')"

				dependency {
					path    = "__checks__/util.ts"
					content = "export const foo = 'baz'"
				}

				dependency {
					path    = "__checks__/data.json"
					content = "{\"foo\": \"bar\"}"
				}
			}`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"dependency.#",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"dependency.0.content",
					"export const foo = 'baz'",
				),
			),
		},
		{
			ResourceName:      "checkly_multistep_check.test",
			ImportState:       true,
			ImportStateVerify: true,
			// The API returns dependencies ordered by path, while the
			// existing state keeps the configured order. An imported state
			// has no prior value to recover the configured order from, so
			// the element order may legitimately differ.
			ImportStateVerifyIgnore: []string{"dependency"},
		},
		{
			// Removing every dependency block must clear the dependencies
			// remotely; otherwise the post-apply refresh would report the
			// remote dependencies as drift and fail this step.
			Config: `resource "checkly_multistep_check" "test" {
				name                      = "Multistep Check Resource Dependencies"
				activated                 = true
				frequency                 = 720
				use_global_alert_settings = true
				locations                 = ["us-east-1"]
				runtime_id                = "2023.09"
				script                    = "console.log('main')"
			}`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"dependency.#",
					"0",
				),
			),
		},
	})
}

// TestAccMultistepCheckResourceMovedFromCheck exercises the MoveState
// implementation of checkly_multistep_check via a `moved` block from
// checkly_check.
func TestAccMultistepCheckResourceMovedFromCheck(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `resource "checkly_check" "test" {
				name                      = "Multistep Check Resource Moved"
				type                      = "MULTI_STEP"
				activated                 = true
				frequency                 = 10
				use_global_alert_settings = true
				locations                 = ["us-east-1"]
				runtime_id                = "2023.09"
				script                    = "console.log('test')"
			}`,
		},
		{
			Config: `
				resource "checkly_multistep_check" "test" {
					name                      = "Multistep Check Resource Moved"
					activated                 = true
					frequency                 = 10
					use_global_alert_settings = true
					locations                 = ["us-east-1"]
					runtime_id                = "2023.09"
					script                    = "console.log('test')"
				}

				moved {
					from = checkly_check.test
					to   = checkly_multistep_check.test
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"name",
					"Multistep Check Resource Moved",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"activated",
					"true",
				),
			),
		},
	})
}

const multistepCheckResource_basic = `
	resource "checkly_multistep_check" "test" {
		name                      = "Multistep Check Resource"
		activated                 = true
		should_fail               = false
		frequency                 = 720
		use_global_alert_settings = true
		locations                 = [ "us-east-1", "eu-central-1" ]
		tags                      = [ "api", "multi-step" ]
		runtime_id                = "2023.09"
		script                    = "console.log('test')"
	}
`
