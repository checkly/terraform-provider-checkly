package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBrowserCheckResourceBasic(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `resource "checkly_browser_check" "test" {
				name      = "Browser Check Resource Defaults"
				locations = ["us-east-1"]
				script    = "console.log('test')"
			}`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"name",
					"Browser Check Resource Defaults",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"activated",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"frequency",
					"10",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"muted",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"description",
					"",
				),
			),
		},
		{
			Config: browserCheckResource_basic,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"name",
					"Browser Check Resource",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"description",
					"Browser check description",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"activated",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"script",
					"console.log('test')",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"locations.#",
					"2",
				),
				testCheckResourceAttrExpr(
					"checkly_browser_check.test",
					"locations.*",
					"us-east-1",
				),
				testCheckResourceAttrExpr(
					"checkly_browser_check.test",
					"locations.*",
					"eu-central-1",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"tags.#",
					"2",
				),
				testCheckResourceAttrExpr(
					"checkly_browser_check.test",
					"tags.*",
					"browser",
				),
				testCheckResourceAttrExpr(
					"checkly_browser_check.test",
					"tags.*",
					"e2e",
				),
			),
		},
		{
			// Removing the tags attribute must clear the tags remotely;
			// otherwise the post-apply refresh would report the remote tags
			// as drift and fail this step.
			Config: browserCheckResource_basic_withoutTags,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(
					"checkly_browser_check.test",
					"tags.#",
				),
			),
		},
	})
}

func TestAccBrowserCheckResourceFull(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: browserCheckResource_full,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"name",
					"Browser Check Resource Full",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"ssl_check_domain",
					"checklyhq.com",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"run_parallel",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"alert_settings.escalation_type",
					"RUN_BASED",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"alert_settings.run_based_escalation.failed_run_threshold",
					"1",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"retry_strategy.type",
					"FIXED",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"retry_strategy.max_retries",
					"3",
				),
			),
		},
	})
}

func TestAccBrowserCheckResourceInvalidFrequency(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `resource "checkly_browser_check" "test" {
				name      = "Browser Check Invalid Frequency"
				frequency = 0
				locations = ["us-east-1"]
				script    = "console.log('test')"
			}`,
			ExpectError: regexp.MustCompile(`value must be one of`),
		},
	})
}

func TestAccBrowserCheckResourceEnvironmentVariables(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `resource "checkly_browser_check" "test" {
				name                      = "Browser Check Resource Env Vars"
				activated                 = true
				frequency                 = 720
				use_global_alert_settings = true
				locations                 = ["us-east-1"]
				script                    = "console.log('test')"

				environment_variable {
					key    = "FOO"
					value  = "bar"
					locked = false
				}

				environment_variable {
					key    = "SECRET"
					value  = "s3cr3t"
					secret = true
				}
			}`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"environment_variable.#",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"environment_variable.0.key",
					"FOO",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"environment_variable.0.value",
					"bar",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"environment_variable.1.key",
					"SECRET",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"environment_variable.1.value",
					"s3cr3t",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"environment_variable.1.secret",
					"true",
				),
			),
		},
		{
			Config: `resource "checkly_browser_check" "test" {
				name                      = "Browser Check Resource Env Vars"
				activated                 = true
				frequency                 = 720
				use_global_alert_settings = true
				locations                 = ["us-east-1"]
				script                    = "console.log('test')"
			}`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"environment_variable.#",
					"0",
				),
			),
		},
	})
}

func TestAccBrowserCheckResourceDependencies(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `resource "checkly_browser_check" "test" {
				name                      = "Browser Check Resource Dependencies"
				activated                 = true
				frequency                 = 720
				use_global_alert_settings = true
				locations                 = ["us-east-1"]
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
					"checkly_browser_check.test",
					"script_path",
					"__checks__/main.spec.ts",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"dependency.#",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"dependency.0.path",
					"__checks__/util.ts",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"dependency.0.content",
					"export const foo = 'bar'",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"dependency.1.path",
					"__checks__/data.json",
				),
			),
		},
		{
			Config: `resource "checkly_browser_check" "test" {
				name                      = "Browser Check Resource Dependencies"
				activated                 = true
				frequency                 = 720
				use_global_alert_settings = true
				locations                 = ["us-east-1"]
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
					"checkly_browser_check.test",
					"dependency.#",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"dependency.0.content",
					"export const foo = 'baz'",
				),
			),
		},
		{
			ResourceName:      "checkly_browser_check.test",
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
			Config: `resource "checkly_browser_check" "test" {
				name                      = "Browser Check Resource Dependencies"
				activated                 = true
				frequency                 = 720
				use_global_alert_settings = true
				locations                 = ["us-east-1"]
				script                    = "console.log('main')"
			}`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"dependency.#",
					"0",
				),
			),
		},
	})
}

// TestAccBrowserCheckResourceMovedFromCheck exercises the MoveState
// implementation of checkly_browser_check via a `moved` block from
// checkly_check.
func TestAccBrowserCheckResourceMovedFromCheck(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `resource "checkly_check" "test" {
				name                      = "Browser Check Resource Moved"
				type                      = "BROWSER"
				activated                 = true
				frequency                 = 10
				use_global_alert_settings = true
				locations                 = ["us-east-1"]
				script                    = "console.log('test')"
			}`,
		},
		{
			Config: `
				resource "checkly_browser_check" "test" {
					name                      = "Browser Check Resource Moved"
					activated                 = true
					frequency                 = 10
					use_global_alert_settings = true
					locations                 = ["us-east-1"]
					script                    = "console.log('test')"
				}

				moved {
					from = checkly_check.test
					to   = checkly_browser_check.test
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"name",
					"Browser Check Resource Moved",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"activated",
					"true",
				),
			),
		},
	})
}

const browserCheckResource_basic = `
	resource "checkly_browser_check" "test" {
		name                      = "Browser Check Resource"
		description               = "Browser check description"
		activated                 = true
		should_fail               = false
		frequency                 = 720
		use_global_alert_settings = true
		locations                 = [ "us-east-1", "eu-central-1" ]
		tags                      = [ "browser", "e2e" ]
		script                    = "console.log('test')"
	}
`

const browserCheckResource_basic_withoutTags = `
	resource "checkly_browser_check" "test" {
		name                      = "Browser Check Resource"
		description               = "Browser check description"
		activated                 = true
		should_fail               = false
		frequency                 = 720
		use_global_alert_settings = true
		locations                 = [ "us-east-1", "eu-central-1" ]
		script                    = "console.log('test')"
	}
`

const browserCheckResource_full = `
	resource "checkly_browser_check" "test" {
		name             = "Browser Check Resource Full"
		description      = "Browser check description"
		activated        = true
		muted            = true
		should_fail      = false
		run_parallel     = true
		frequency        = 720
		locations        = [ "us-east-1", "eu-central-1" ]
		tags             = [ "browser", "e2e" ]
		ssl_check_domain = "checklyhq.com"
		script           = "console.log('test')"

		alert_settings = {
			escalation_type = "RUN_BASED"
			reminders = {
				amount   = 0
				interval = 5
			}
			run_based_escalation = {
				failed_run_threshold = 1
			}
			parallel_run_failure_threshold = {
				enabled    = false
				percentage = 10
			}
		}

		retry_strategy = {
			type                 = "FIXED"
			base_backoff_seconds = 60
			max_duration_seconds = 600
			max_retries          = 3
			same_region          = false
		}
	}
`
