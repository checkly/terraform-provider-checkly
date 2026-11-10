package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/checkly/checkly-go-sdk/v2"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccCheckRequiredFields(t *testing.T) {
	config := `resource "checkly_check" "test" {}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "type" is required, but no definition was found.`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "name" is required, but no definition was found.`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "activated" is required, but no definition was found.`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "frequency" is required, but no definition was found.`),
		},
	})
}

func TestAccBrowserCheckInvalidInputs(t *testing.T) {
	config := `resource "checkly_check" "test" {
		name                      = 1
		type                      = "BROWSER"
		activated                 = "invalid"
		should_fail               = "invalid"
		use_global_alert_settings = "invalid"
		locations = "invalid"
		script = 4
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`Inappropriate value for attribute "activated"`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "frequency" is required`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`Inappropriate value for attribute "should_fail"`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`Inappropriate value for attribute "use_global_alert_settings"`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`Inappropriate value for attribute "locations"`),
		},
	})
}

func TestAccBrowserCheckMissingScript(t *testing.T) {
	config := `resource "checkly_check" "test" {
		type = "BROWSER"
		activated = true
		frequency = 10
		name = "browser check"
		locations = [ "us-west-1" ]
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`Missing\s+Check\s+Script`),
		},
	})
}

// TestAccCheckEmptyScript covers every way that a browser check's planned
// script ends up empty, on create and on update: the plan reports an omitted
// or null script, and the apply one known only then. The API would accept the
// update and clear the stored script.
func TestAccCheckEmptyScript(t *testing.T) {
	// Replacing the terraform_data resource makes its output unknown in the
	// plan of an update as well.
	knownOnlyAtApply := func(revision int) string {
		return fmt.Sprintf(`
			resource "terraform_data" "empty" {
				input            = ""
				triggers_replace = %d
			}
		`, revision)
	}

	cases := map[string]struct {
		extra  func(revision int) string
		script string
	}{
		"omitted": {extra: func(int) string { return "" }},
		"null":    {extra: func(int) string { return "" }, script: `script = null`},
		"known only at apply": {
			extra:  knownOnlyAtApply,
			script: `script = terraform_data.empty.output`,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			config := func(revision int, script string) string {
				return c.extra(revision) + fmt.Sprintf(`
					resource "checkly_check" "test" {
						name      = "tf-acc-empty-script"
						type      = "BROWSER"
						activated = false
						frequency = 10
						locations = ["eu-central-1"]
						use_global_alert_settings = true
						%s
					}
				`, script)
			}
			missingScript := regexp.MustCompile(`Missing\s+Check\s+Script`)

			accTestCase(t, []resource.TestStep{
				{
					Config:      config(1, c.script),
					ExpectError: missingScript,
				},
				{
					Config: config(1, `script = "console.log('hi')"`),
				},
				{
					Config:      config(2, c.script),
					ExpectError: missingScript,
				},
			})
		})
	}
}

func TestAccBrowserCheckBasic(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: browserCheck_basic,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"name",
					"Browser Check",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"description",
					"Browser check description",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"type",
					"BROWSER",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"activated",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"script",
					"console.log('test')",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"locations.#",
					"2",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"locations.*",
					"us-east-1",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"locations.*",
					"eu-central-1",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"tags.#",
					"2",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"tags.*",
					"browser",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"tags.*",
					"e2e",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_check.test",
					"request",
				),
			),
		},
	})
}

// TestAccApiCheckRequestUnknownValues exercises a request whose values are
// unknown at plan time because they reference an attribute of a resource
// that has not been created yet. Plan-time request defaulting must leave
// unknown configurations untouched so that the apply-time plan can resolve
// them without producing an inconsistent final plan.
func TestAccApiCheckRequestUnknownValues(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_snippet" "test" {
					name   = "request-unknown-values"
					script = "console.log('hi')"
				}

				resource "checkly_check" "test" {
					name      = "API Check With Unknown Request Values"
					type      = "API"
					activated = false
					frequency = 60

					locations = ["eu-central-1"]

					request = {
						url = "https://api.checklyhq.com/public-stats?marker=${checkly_snippet.test.id}"
						assertions = [{
							source     = "STATUS_CODE"
							comparison = "EQUALS"
							target     = checkly_snippet.test.id
						}]
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"request.method",
					"GET",
				),
			),
		},
	})
}

func TestAccApiCheckBasic(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: apiCheck_basic,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"name",
					"API Check 1",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"description",
					"API check description",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"activated",
					"true",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"locations.*",
					"eu-central-1",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"locations.*",
					"us-east-1",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"request.method",
					"GET",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"request.url",
					"https://api.checklyhq.com/public-stats",
				),
				resource.TestCheckTypeSetElemNestedAttrs(
					"checkly_check.test",
					"request.assertions.*",
					map[string]string{
						"source":     "STATUS_CODE",
						"property":   "",
						"comparison": "EQUALS",
						"target":     "200",
					},
				),
			),
		},
	})
}

func TestAccMultiStepCheckRuntimeValidation(t *testing.T) {
	unsupportedRuntime := `resource "checkly_check" "test" {
		name = "test"
		type = "MULTI_STEP"
		activated = true
		frequency = 5
		locations = ["eu-central-1"]
		script = "console.log('test')"
		runtime_id = "2023.02"
	}`
	noSpecifiedRuntime := `resource "checkly_check" "test" {
		name = "test"
		type = "MULTI_STEP"
		activated = true
		frequency = 5
		locations = ["eu-central-1"]
		script = "console.log('test')"
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      unsupportedRuntime,
			ExpectError: regexp.MustCompile("runtime 2023.02 does not support MULTI_STEP checks"),
		},
		{
			Config: noSpecifiedRuntime,
			Check: resource.TestCheckNoResourceAttr(
				"checkly_check.test",
				"runtime_id",
			),
		},
	})
}

func TestAccMultiStepCheckBasic(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: multiStepCheck_basic,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"name",
					"MultiStep Check",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"type",
					"MULTI_STEP",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"runtime_id",
					"2023.09",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"activated",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"script",
					"console.log('test')",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"locations.#",
					"2",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"locations.*",
					"us-east-1",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"locations.*",
					"eu-central-1",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"tags.#",
					"2",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"tags.*",
					"api",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"tags.*",
					"multi-step",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_check.test",
					"request",
				),
			),
		},
	})
}

func TestAccApiCheckFull(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: apiCheck_full,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"degraded_response_time",
					"15000",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"max_response_time",
					"30000",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					`"locations.#"`,
					"3",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"request.headers.X-CUSTOM-1",
					"1",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"request.headers.X-CUSTOM-2",
					"foo",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"request.query_parameters.param1",
					"123",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"request.query_parameters.param2",
					"bar",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"request.basic_auth.username",
					"user",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"request.basic_auth.password",
					"pass",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"request.assertions.#",
					"3",
				),
				resource.TestCheckTypeSetElemNestedAttrs(
					"checkly_check.test",
					"request.assertions.*",
					map[string]string{
						"source":     "STATUS_CODE",
						"property":   "",
						"comparison": "EQUALS",
						"target":     "200",
					},
				),
				resource.TestCheckTypeSetElemNestedAttrs(
					"checkly_check.test",
					"request.assertions.*",
					map[string]string{
						"source":     "HEADERS",
						"property":   "cache-control",
						"comparison": "EQUALS",
						"target":     "no-cache",
					},
				),
				resource.TestCheckTypeSetElemNestedAttrs(
					"checkly_check.test",
					"request.assertions.*",
					map[string]string{
						"source":     "JSON_BODY",
						"property":   "$.apiCheckResults",
						"comparison": "GREATER_THAN",
						"target":     "100",
					},
				),
			),
		},
		{
			ResourceName:      "checkly_check.test",
			ImportState:       true,
			ImportStateVerify: true,
		},
	})
}

func TestAccApiCheckMore(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: apiCheck_post,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"request.method",
					"POST",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"request.body",
					`{"message":"hello checkly","messageId":1}`,
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"request.body_type",
					"JSON",
				),
			),
		},
		{
			Config: apiCheck_withEmptyBasicAuth,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"name",
					"api check with empty basic_auth",
				),
			),
		},
	})
}

func TestAccApiCheckWithTriggerIncident(t *testing.T) {
	apiCheckWithTriggerIncident := `
resource "checkly_status_page_service" "test_service" {
	name = "API Test Service"
}

resource "checkly_check" "test_trigger_incident" {
	name         = "API Check with Trigger Incident"
	type         = "API"
	activated    = true
	frequency    = 10
	locations    = ["us-east-1"]

	trigger_incident = {
		service_id         = checkly_status_page_service.test_service.id
		severity           = "MAJOR"
		name               = "API Check Failure Incident"
		description        = "The API health check has failed"
		notify_subscribers = true
	}

	request = {
		url    = "https://api.example.com/health"
		method = "GET"
		assertions = [{
			source     = "STATUS_CODE"
			comparison = "EQUALS"
			target     = "200"
		}]
	}

	use_global_alert_settings = true
}
`
	accTestCase(t, []resource.TestStep{
		{
			Config: apiCheckWithTriggerIncident,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test_trigger_incident",
					"name",
					"API Check with Trigger Incident",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test_trigger_incident",
					"type",
					"API",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test_trigger_incident",
					"trigger_incident.severity",
					"MAJOR",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test_trigger_incident",
					"trigger_incident.name",
					"API Check Failure Incident",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test_trigger_incident",
					"trigger_incident.description",
					"The API health check has failed",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test_trigger_incident",
					"trigger_incident.notify_subscribers",
					"true",
				),
			),
		},
	})
}

func TestAccBrowserCheckWithTriggerIncident(t *testing.T) {
	browserCheckWithTriggerIncident := `
resource "checkly_status_page_service" "test_browser_service" {
	name = "Browser Test Service"
}

resource "checkly_check" "test_browser_trigger" {
	name         = "Browser Check with Trigger Incident"
	type         = "BROWSER"
	activated    = true
	frequency    = 15
	locations    = ["eu-west-1"]

	trigger_incident = {
		service_id         = checkly_status_page_service.test_browser_service.id
		severity           = "CRITICAL"
		name               = "Browser Check Critical Failure"
		description        = "The browser check has detected a critical failure"
		notify_subscribers = false
	}

	script = <<EOT
const { chromium } = require('playwright');
(async () => {
	const browser = await chromium.launch();
	const page = await browser.newPage();
	await page.goto('https://example.com');
	await browser.close();
})();
EOT

	use_global_alert_settings = true
}
`
	accTestCase(t, []resource.TestStep{
		{
			Config: browserCheckWithTriggerIncident,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test_browser_trigger",
					"name",
					"Browser Check with Trigger Incident",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test_browser_trigger",
					"type",
					"BROWSER",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test_browser_trigger",
					"trigger_incident.severity",
					"CRITICAL",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test_browser_trigger",
					"trigger_incident.name",
					"Browser Check Critical Failure",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test_browser_trigger",
					"trigger_incident.description",
					"The browser check has detected a critical failure",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test_browser_trigger",
					"trigger_incident.notify_subscribers",
					"false",
				),
			),
		},
	})
}

func TestAccCheckWithSingleRetry(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check" "test" {
					name      = "test-single-retry"
					type      = "API"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						method = "GET"
						url    = "https://api.checklyhq.com/public-stats"
					}
					retry_strategy = {
						type                 = "SINGLE_RETRY"
						base_backoff_seconds = 30
						same_region          = true
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.type",
					"SINGLE_RETRY",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.base_backoff_seconds",
					"30",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.max_retries",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.max_duration_seconds",
					"600",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.same_region",
					"true",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_check.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
	})
}

func TestAccCheckWithSingleRetryOnlyOnNetworkError(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check" "test" {
					name      = "test-single-retry"
					type      = "API"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						method = "GET"
						url    = "https://api.checklyhq.com/public-stats"
					}
					retry_strategy = {
						type = "SINGLE_RETRY"

						only_on = {
							network_error = true
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.only_on.network_error",
					"true",
				),
			),
		},
		{
			Config: `
				resource "checkly_check" "test" {
					name      = "test-single-retry"
					type      = "API"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						method = "GET"
						url    = "https://api.checklyhq.com/public-stats"
					}
					retry_strategy = {
						type = "SINGLE_RETRY"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(
					"checkly_check.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
	})
}

func TestAccCheckWithNoRetries(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check" "test" {
					name      = "test-no-retries"
					type      = "API"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						method = "GET"
						url    = "https://api.checklyhq.com/public-stats"
					}
					retry_strategy = {
						type = "NO_RETRIES"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.type",
					"NO_RETRIES",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.base_backoff_seconds",
					"60",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.max_retries",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.max_duration_seconds",
					"600",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.same_region",
					"true",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_check.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
	})
}

func TestAccCheckWithNoRetriesOnlyOnNetworkError(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check" "test" {
					name      = "test-no-retries"
					type      = "API"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						method = "GET"
						url    = "https://api.checklyhq.com/public-stats"
					}
					retry_strategy = {
						type = "NO_RETRIES"

						only_on = {
							network_error = true
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				// The NO_RETRIES strategy sends no retry conditions to the
				// API, but the configured value is kept in state.
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.only_on.network_error",
					"true",
				),
			),
		},
	})
}

func TestAccCheckWithDefaultNoRetries(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check" "test" {
					name      = "test-no-retries"
					type      = "API"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						method = "GET"
						url    = "https://api.checklyhq.com/public-stats"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.type",
					"NO_RETRIES",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_check.test",
					"retry_strategy.base_backoff_seconds",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_check.test",
					"retry_strategy.max_retries",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_check.test",
					"retry_strategy.max_duration_seconds",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_check.test",
					"retry_strategy.same_region",
				),
			),
		},
	})
}

func TestAccCheckWithFixedRetry(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check" "test" {
					name      = "test-fixed-retry"
					type      = "API"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						method = "GET"
						url    = "https://api.checklyhq.com/public-stats"
					}
					retry_strategy = {
						type                 = "FIXED"
						base_backoff_seconds = 30
						max_retries          = 3
						max_duration_seconds = 300
						same_region          = false
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.type",
					"FIXED",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.base_backoff_seconds",
					"30",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.max_retries",
					"3",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.max_duration_seconds",
					"300",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.same_region",
					"false",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_check.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
	})
}

func TestAccCheckWithFixedRetryOnlyOnNetworkError(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check" "test" {
					name      = "test-fixed-retry"
					type      = "API"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						method = "GET"
						url    = "https://api.checklyhq.com/public-stats"
					}
					retry_strategy = {
						type = "FIXED"

						only_on = {
							network_error = true
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.only_on.network_error",
					"true",
				),
			),
		},
		{
			Config: `
				resource "checkly_check" "test" {
					name      = "test-fixed-retry"
					type      = "API"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						method = "GET"
						url    = "https://api.checklyhq.com/public-stats"
					}
					retry_strategy = {
						type = "FIXED"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(
					"checkly_check.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
	})
}

func TestAccCheckWithExponentialRetry(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check" "test" {
					name      = "test-exponential-retry"
					type      = "BROWSER"
					activated = true
					frequency = 720
					locations = ["eu-central-1"]
					script    = "console.log('test')"
					retry_strategy = {
						type                 = "EXPONENTIAL"
						base_backoff_seconds = 60
						max_retries          = 5
						max_duration_seconds = 600
						same_region          = true
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.type",
					"EXPONENTIAL",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.base_backoff_seconds",
					"60",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.max_retries",
					"5",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.max_duration_seconds",
					"600",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.same_region",
					"true",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_check.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
	})
}

func TestAccCheckWithExponentialRetryOnlyOnNetworkError(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check" "test" {
					name      = "test-exponential-retry"
					type      = "BROWSER"
					activated = true
					frequency = 720
					locations = ["eu-central-1"]
					script    = "console.log('test')"
					retry_strategy = {
						type = "EXPONENTIAL"

						only_on = {
							network_error = true
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.only_on.network_error",
					"true",
				),
			),
		},
		{
			Config: `
				resource "checkly_check" "test" {
					name      = "test-exponential-retry"
					type      = "BROWSER"
					activated = true
					frequency = 720
					locations = ["eu-central-1"]
					script    = "console.log('test')"
					retry_strategy = {
						type = "EXPONENTIAL"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(
					"checkly_check.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
	})
}

func TestAccCheckWithLinearRetry(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check" "test" {
					name       = "test-linear-retry"
					type       = "MULTI_STEP"
					activated  = true
					frequency  = 720
					locations  = ["eu-central-1"]
					runtime_id = "2023.09"
					script     = "console.log('test')"
					retry_strategy = {
						type                 = "LINEAR"
						base_backoff_seconds = 45
						max_retries          = 4
						max_duration_seconds = 450
						same_region          = false
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.type",
					"LINEAR",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.base_backoff_seconds",
					"45",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.max_retries",
					"4",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.max_duration_seconds",
					"450",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.same_region",
					"false",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_check.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
	})
}

func TestAccCheckWithLinearRetryOnlyOnNetworkError(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check" "test" {
					name       = "test-linear-retry"
					type       = "MULTI_STEP"
					activated  = true
					frequency  = 720
					locations  = ["eu-central-1"]
					runtime_id = "2023.09"
					script     = "console.log('test')"
					retry_strategy = {
						type = "LINEAR"

						only_on = {
							network_error = true
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.only_on.network_error",
					"true",
				),
			),
		},
		{
			Config: `
				resource "checkly_check" "test" {
					name       = "test-linear-retry"
					type       = "MULTI_STEP"
					activated  = true
					frequency  = 720
					locations  = ["eu-central-1"]
					runtime_id = "2023.09"
					script     = "console.log('test')"
					retry_strategy = {
						type = "LINEAR"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(
					"checkly_check.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
	})
}

func TestAccCheckRetryStrategyRemoval(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check" "test" {
					name       = "test-linear-retry"
					type       = "MULTI_STEP"
					activated  = true
					frequency  = 720
					locations  = ["eu-central-1"]
					runtime_id = "2023.09"
					script     = "console.log('test')"
					retry_strategy = {
						type = "LINEAR"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.type",
					"LINEAR",
				),
			),
		},
		{
			Config: `
				resource "checkly_check" "test" {
					name       = "test-linear-retry"
					type       = "MULTI_STEP"
					activated  = true
					frequency  = 720
					locations  = ["eu-central-1"]
					runtime_id = "2023.09"
					script     = "console.log('test')"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"retry_strategy.type",
					"NO_RETRIES",
				),
			),
		},
	})
}

func TestAccCheckDescriptionRemoval(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: apiCheck_basic,
			Check: resource.TestCheckResourceAttr(
				"checkly_check.test",
				"description",
				"API check description",
			),
		},
		{
			Config: apiCheck_basic_withoutDescription,
			Check: resource.TestCheckResourceAttr(
				"checkly_check.test",
				"description",
				"",
			),
		},
	})
}

const browserCheck_basic = `
	resource "checkly_check" "test" {
		name                      = "Browser Check"
		description               = "Browser check description"
		type                      = "BROWSER"
		activated                 = true
		should_fail               = false
		frequency                 = 720
		use_global_alert_settings = true
		locations                 = [ "us-east-1", "eu-central-1" ]
		tags                      = [ "browser", "e2e" ]
		script                    = "console.log('test')"
	}
`
const multiStepCheck_basic = `
	resource "checkly_check" "test" {
		name                      = "MultiStep Check"
		type                      = "MULTI_STEP"
		activated                 = true
		should_fail               = false
		frequency                 = 720
		use_global_alert_settings = true
		locations                 = [ "us-east-1", "eu-central-1" ]
		tags                      = [ "api", "multi-step" ]
		runtime_id				  = "2023.09"
		script                    = "console.log('test')"
	}
`

const apiCheck_basic = `
	resource "checkly_check" "test" {
	  name                      = "API Check 1"
	  description               = "API check description"
	  type                      = "API"
	  frequency                 = 60
	  activated                 = true
	  muted                     = true
	  max_response_time         = 18000
	  locations                 = [ "us-east-1", "eu-central-1" ]
	  use_global_alert_settings = true
	  request = {
		method     = "GET"
		url        = "https://api.checklyhq.com/public-stats"
		assertions = [{
		  comparison = "EQUALS"
		  property   = ""
		  source     = "STATUS_CODE"
		  target     = "200"
		}]
	  }
	}
`

const apiCheck_basic_withoutDescription = `
	resource "checkly_check" "test" {
	  name                      = "API Check 1"
	  type                      = "API"
	  frequency                 = 60
	  activated                 = true
	  muted                     = true
	  max_response_time         = 18000
	  locations                 = [ "us-east-1", "eu-central-1" ]
	  use_global_alert_settings = true
	  request = {
		method     = "GET"
		url        = "https://api.checklyhq.com/public-stats"
		assertions = [{
		  comparison = "EQUALS"
		  property   = ""
		  source     = "STATUS_CODE"
		  target     = "200"
		}]
	  }
	}
`

const apiCheck_full = `
  resource "checkly_check" "test" {
	name                   = "apiCheck_full"
	type                   = "API"
	frequency              = 120
	activated              = true
	muted                  = true
	degraded_response_time = 15000
	max_response_time      = 30000
	locations = [
	  "eu-central-1",
	  "us-east-1",
	  "ap-northeast-1"
	]
	request = {
	  method           = "GET"
	  url              = "https://api.checklyhq.com/public-stats"
	  follow_redirects = true
	  headers = {
		X-CUSTOM-1 = 1
		X-CUSTOM-2 = "foo"
	  }
	  query_parameters = {
		param1 = 123
		param2 = "bar"
	  }
	  basic_auth = {
		username = "user"
		password = "pass"
	  }
	  assertions = [{
		comparison = "EQUALS"
		property   = ""
		source     = "STATUS_CODE"
		target     = "200"
	  }, {
		comparison = "EQUALS"
		property   = "cache-control"
		source     = "HEADERS"
		target     = "no-cache"
	  }, {
		comparison = "GREATER_THAN"
		property   = "$.apiCheckResults"
		source     = "JSON_BODY"
		target     = "100"
	  }]
	}

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
  }
`

const apiCheck_post = `
  resource "checkly_check" "test" {
	name         = "apiCheck_post"
	type         = "API"
	activated    = true
	frequency    = 720
	locations = [ "eu-central-1", "us-east-2" ]
	max_response_time     = 18000
	muted                 = true
	request = {
	  method           = "POST"
	  url              = "https://jsonplaceholder.typicode.com/posts"
	  headers = {
		Content-type = "application/json; charset=UTF-8"
	  }
	  body      = "{\"message\":\"hello checkly\",\"messageId\":1}"
	  body_type = "JSON"
	}
	use_global_alert_settings = true
  }
`

const apiCheck_withEmptyBasicAuth = `
  resource "checkly_check" "test" {
	name                   = "api check with empty basic_auth"
	type                   = "API"
	activated              = true
	should_fail            = false
	frequency              = 1
	degraded_response_time = 3000
	max_response_time      = 6000
	tags = [
	  "testing",
	  "bug"
	]
	locations = [ "eu-central-1" ]
	request = {
	  follow_redirects = false
	  url              = "https://api.checklyhq.com/public-stats"
	  basic_auth = {
		username = ""
		password = ""
	  }
	  assertions = [{
		source     = "STATUS_CODE"
		property   = ""
		comparison = "EQUALS"
		target     = "200"
	  }]
	}
  }
`

func TestAccCheckGroupAssignment(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group" "test1" {
					name        = "test-group-assignment"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}

				resource "checkly_check_group" "test2" {
					name        = "test-group-assignment"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}

				resource "checkly_check" "test1" {
					name      = "test-group-assignment"
					type      = "API"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					group_id  = checkly_check_group.test1.id
					request = {
						method = "GET"
						url    = "https://welcome.checklyhq.com"
					}
				}

				resource "checkly_check" "test2" {
					name      = "test-group-assignment"
					type      = "API"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						method = "GET"
						url    = "https://welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrPair(
					"checkly_check.test1",
					"group_id",
					"checkly_check_group.test1",
					"id",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test2",
					"group_id",
					"0",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group" "test1" {
					name        = "test-group-assignment"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}

				resource "checkly_check_group" "test2" {
					name        = "test-group-assignment"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}

				resource "checkly_check" "test1" {
					name      = "test-group-assignment"
					type      = "API"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					group_id  = checkly_check_group.test2.id
					request = {
						method = "GET"
						url    = "https://welcome.checklyhq.com"
					}
				}

				resource "checkly_check" "test2" {
					name      = "test-group-assignment"
					type      = "API"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					group_id  = checkly_check_group.test2.id
					request = {
						method = "GET"
						url    = "https://welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrPair(
					"checkly_check.test1",
					"group_id",
					"checkly_check_group.test2",
					"id",
				),
				resource.TestCheckResourceAttrPair(
					"checkly_check.test2",
					"group_id",
					"checkly_check_group.test2",
					"id",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group" "test1" {
					name        = "test-group-assignment"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}

				resource "checkly_check_group" "test2" {
					name        = "test-group-assignment"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}

				resource "checkly_check" "test1" {
					name      = "test-group-assignment"
					type      = "API"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						method = "GET"
						url    = "https://welcome.checklyhq.com"
					}
				}

				resource "checkly_check" "test2" {
					name      = "test-group-assignment"
					type      = "API"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						method = "GET"
						url    = "https://welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test1",
					"group_id",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test2",
					"group_id",
					"0",
				),
			),
		},
	})
}

func TestAccCheckWithEnvironmentVariable(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check" "test" {
					name                      = "Check with env vars"
					type                      = "BROWSER"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]
					script                    = "console.log('test')"

					environment_variables = [
						{
							key    = "FOO"
							value  = "bar"
							locked = false
						},
						{
							key    = "SECRET"
							value  = "s3cr3t"
							locked = true
						},
					]
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"environment_variables.#",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"environment_variables.0.key",
					"FOO",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"environment_variables.0.value",
					"bar",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"environment_variables.0.locked",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"environment_variables.1.key",
					"SECRET",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"environment_variables.1.value",
					"s3cr3t",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"environment_variables.1.locked",
					"true",
				),
			),
		},
	})
}

func TestAccCheckEnvironmentVariableRemoval(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check" "test" {
					name                      = "Check env var removal"
					type                      = "BROWSER"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]
					script                    = "console.log('test')"

					environment_variables = [
						{
							key   = "FOO"
							value = "bar"
						},
					]
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"environment_variables.#",
					"1",
				),
			),
		},
		{
			Config: `
				resource "checkly_check" "test" {
					name                      = "Check env var removal"
					type                      = "BROWSER"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]
					script                    = "console.log('test')"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"environment_variables.#",
					"0",
				),
			),
		},
	})
}

func TestAccCheckFrequencyValidation(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check" "test-missing-frequency_offset" {
					name      = "test-missing-frequency_offset"
					type      = "API"
					activated = true
					frequency = 0
					locations = ["eu-central-1"]
					request = {
						method = "GET"
						url    = "https://welcome.checklyhq.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`"frequency_offset" is required when "frequency" is 0`),
		},
		{
			Config: `
				resource "checkly_check" "test-frequency_offset-ok-10" {
					name             = "test-frequency_offset-ok-10"
					type             = "API"
					activated        = true
					frequency        = 0
					frequency_offset = 10
					locations        = ["eu-central-1"]
					request = {
						method = "GET"
						url    = "https://welcome.checklyhq.com"
					}
				}

				resource "checkly_check" "test-frequency_offset-ok-20" {
					name             = "test-frequency_offset-ok-20"
					type             = "API"
					activated        = true
					frequency        = 0
					frequency_offset = 20
					locations        = ["eu-central-1"]
					request = {
						method = "GET"
						url    = "https://welcome.checklyhq.com"
					}
				}

				resource "checkly_check" "test-frequency_offset-ok-30" {
					name             = "test-frequency_offset-ok-30"
					type             = "API"
					activated        = true
					frequency        = 0
					frequency_offset = 30
					locations        = ["eu-central-1"]
					request = {
						method = "GET"
						url    = "https://welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test-frequency_offset-ok-10",
					"frequency",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test-frequency_offset-ok-10",
					"frequency_offset",
					"10",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test-frequency_offset-ok-20",
					"frequency",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test-frequency_offset-ok-20",
					"frequency_offset",
					"20",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test-frequency_offset-ok-30",
					"frequency",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test-frequency_offset-ok-30",
					"frequency_offset",
					"30",
				),
			),
		},
		{
			Config: `
				resource "checkly_check" "test-extra-frequency_offset" {
					name             = "test-extra-frequency_offset"
					type             = "API"
					activated        = true
					frequency        = 60
					frequency_offset = 30
					locations        = ["eu-central-1"]
					request = {
						method = "GET"
						url    = "https://welcome.checklyhq.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`"frequency_offset" can only be set when "frequency" is 0`),
		},
		{
			Config: `
				resource "checkly_check" "test-bad-frequency" {
					name      = "test-bad-frequency"
					type      = "API"
					activated = true
					frequency = 9999
					locations = ["eu-central-1"]
					request = {
						method = "GET"
						url    = "https://welcome.checklyhq.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`value must be one of`),
		},
		{
			Config: `
				resource "checkly_check" "test-bad-frequency_offset" {
					name             = "test-bad-frequency_offset"
					type             = "API"
					activated        = true
					frequency        = 60
					frequency_offset = 99
					locations        = ["eu-central-1"]
					request = {
						method = "GET"
						url    = "https://welcome.checklyhq.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`value must be one of`),
		},
		{
			Config: `
				resource "checkly_check" "test-frequency-ok-60" {
					name             = "test-frequency-ok-60"
					type             = "API"
					activated        = true
					frequency        = 60
					frequency_offset = 0
					locations        = ["eu-central-1"]
					request = {
						method = "GET"
						url    = "https://welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test-frequency-ok-60",
					"frequency",
					"60",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test-frequency-ok-60",
					"frequency_offset",
					"0",
				),
			),
		},
	})
}

func TestAccCheckFrequencyUpdates(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check" "test1" {
					name      = "test-normal-frequency"
					type      = "API"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						method = "GET"
						url    = "https://welcome.checklyhq.com"
					}
				}

				resource "checkly_check" "test2" {
					name             = "test-high-frequency"
					type             = "API"
					activated        = true
					frequency        = 0
					frequency_offset = 30
					locations        = ["eu-central-1"]
					request = {
						method = "GET"
						url    = "https://welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test1",
					"frequency",
					"60",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_check.test1",
					"frequency_offset",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test2",
					"frequency",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test2",
					"frequency_offset",
					"30",
				),
			),
		},
		{
			Config: `
				resource "checkly_check" "test1" {
					name             = "test-normal-frequency-updated-to-high"
					type             = "API"
					activated        = true
					frequency        = 0
					frequency_offset = 10
					locations        = ["eu-central-1"]
					request = {
						method = "GET"
						url    = "https://welcome.checklyhq.com"
					}
				}

				resource "checkly_check" "test2" {
					name      = "test-high-frequency-updated-to-normal"
					type      = "API"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						method = "GET"
						url    = "https://welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test1",
					"frequency",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test1",
					"frequency_offset",
					"10",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test2",
					"frequency",
					"60",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_check.test2",
					"frequency_offset",
				),
			),
		},
	})
}

func TestAccCheckAlertChannelSubscriptions(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		// Step 1: Add two alert channel subscriptions.
		{
			Config: `
				resource "checkly_alert_channel" "email_1" {
					email = {
						address = "check-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email = {
						address = "check-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email = {
						address = "check-acs-3@example.com"
					}
				}

				resource "checkly_check" "test" {
					name      = "check-alert-channel-sub-test"
					type      = "API"
					frequency = 60
					activated = true
					locations = ["eu-central-1"]

					request = {
						method = "GET"
						url    = "https://api.checklyhq.com/public-stats"
					}

					alert_channel_subscriptions = [
						{
							channel_id = checkly_alert_channel.email_1.id
							activated  = true
						},
						{
							channel_id = checkly_alert_channel.email_2.id
							activated  = false
						},
					]
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"alert_channel_subscriptions.#",
					"2",
				),
				testCheckAlertChannelSubscription(
					"checkly_check.test",
					"alert_channel_subscriptions",
					"checkly_alert_channel.email_1",
					"true",
				),
				testCheckAlertChannelSubscription(
					"checkly_check.test",
					"alert_channel_subscriptions",
					"checkly_alert_channel.email_2",
					"false",
				),
			),
		},
		// Step 2: Change activations and add a third subscription.
		{
			Config: `
				resource "checkly_alert_channel" "email_1" {
					email = {
						address = "check-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email = {
						address = "check-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email = {
						address = "check-acs-3@example.com"
					}
				}

				resource "checkly_check" "test" {
					name      = "check-alert-channel-sub-test"
					type      = "API"
					frequency = 60
					activated = true
					locations = ["eu-central-1"]

					request = {
						method = "GET"
						url    = "https://api.checklyhq.com/public-stats"
					}

					alert_channel_subscriptions = [
						{
							channel_id = checkly_alert_channel.email_1.id
							activated  = false
						},
						{
							channel_id = checkly_alert_channel.email_2.id
							activated  = true
						},
						{
							channel_id = checkly_alert_channel.email_3.id
							activated  = true
						},
					]
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"alert_channel_subscriptions.#",
					"3",
				),
				testCheckAlertChannelSubscription(
					"checkly_check.test",
					"alert_channel_subscriptions",
					"checkly_alert_channel.email_1",
					"false",
				),
				testCheckAlertChannelSubscription(
					"checkly_check.test",
					"alert_channel_subscriptions",
					"checkly_alert_channel.email_2",
					"true",
				),
				testCheckAlertChannelSubscription(
					"checkly_check.test",
					"alert_channel_subscriptions",
					"checkly_alert_channel.email_3",
					"true",
				),
			),
		},
		// Step 3: Remove all alert channel subscriptions.
		{
			Config: `
				resource "checkly_alert_channel" "email_1" {
					email = {
						address = "check-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email = {
						address = "check-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email = {
						address = "check-acs-3@example.com"
					}
				}

				resource "checkly_check" "test" {
					name      = "check-alert-channel-sub-test"
					type      = "API"
					frequency = 60
					activated = true
					locations = ["eu-central-1"]

					request = {
						method = "GET"
						url    = "https://api.checklyhq.com/public-stats"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"alert_channel_subscriptions.#",
					"0",
				),
			),
		},
	})
}

// TestAccCheckDoubleCheck covers the deprecated double_check attribute, which
// implies a single retry in another region when retry_strategy is not
// configured, and switches back to no retries when removed.
func TestAccCheckDoubleCheck(t *testing.T) {
	config := func(doubleCheck string) string {
		return fmt.Sprintf(`
			resource "checkly_check" "test" {
				name                      = "tf-acc-double-check"
				type                      = "API"
				frequency                 = 60
				activated                 = false
				locations                 = ["eu-central-1"]
				use_global_alert_settings = true
				%s

				request = {
					url = "https://api.checklyhq.com/public-stats"
				}
			}
		`, doubleCheck)
	}
	const r = "checkly_check.test"
	accTestCase(t, []resource.TestStep{
		{
			// Fails validation before anything is created.
			Config: config(`
				double_check = true
				retry_strategy = {
					type = "FIXED"
				}
			`),
			ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
		},
		{
			Config: config("double_check = true"),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(r, "double_check", "true"),
				resource.TestCheckResourceAttr(r, "retry_strategy.type", "FIXED"),
				resource.TestCheckResourceAttr(r, "retry_strategy.max_retries", "1"),
				resource.TestCheckResourceAttr(r, "retry_strategy.same_region", "false"),
			),
		},
		{
			ResourceName:            r,
			ImportState:             true,
			ImportStateVerify:       true,
			ImportStateVerifyIgnore: []string{"double_check"},
		},
		{
			Config: config(""),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(r, "double_check", "false"),
				resource.TestCheckResourceAttr(r, "retry_strategy.type", "NO_RETRIES"),
			),
		},
	})
}

// TestAccCheckTypeChangeInPlace changes the type of a stored check with an
// update, which keeps its ID.
func TestAccCheckTypeChangeInPlace(t *testing.T) {
	config := func(checkType, attributes string) string {
		return fmt.Sprintf(`
			resource "checkly_check" "test" {
				name                      = "tf-acc-type-change"
				type                      = %q
				activated                 = false
				frequency                 = 10
				locations                 = ["eu-central-1"]
				runtime_id                = "2023.09"
				use_global_alert_settings = true
				%s
			}
		`, checkType, attributes)
	}
	const r = "checkly_check.test"
	const script = `script = "console.log('hi')"`

	var id string
	sameID := func(s *terraform.State) error {
		got := s.RootModule().Resources[r].Primary.ID
		if id == "" {
			id = got
		}
		if got != id {
			return fmt.Errorf("id = %s, want %s", got, id)
		}
		return nil
	}
	updatesInPlace := resource.ConfigPlanChecks{
		PreApply: []plancheck.PlanCheck{
			plancheck.ExpectResourceAction(r, plancheck.ResourceActionUpdate),
		},
	}

	accTestCase(t, []resource.TestStep{
		{
			Config: config("API", `
				request = {
					url = "https://api.checklyhq.com/public-stats"
				}
			`),
			Check: sameID,
		},
		{
			Config:           config("BROWSER", script),
			ConfigPlanChecks: updatesInPlace,
			Check: resource.ComposeTestCheckFunc(
				sameID,
				resource.TestCheckResourceAttr(r, "type", "BROWSER"),
			),
		},
		{
			Config:           config("MULTI_STEP", script),
			ConfigPlanChecks: updatesInPlace,
			Check: resource.ComposeTestCheckFunc(
				sameID,
				resource.TestCheckResourceAttr(r, "type", "MULTI_STEP"),
			),
		},
	})
}

// TestAccCheckUpdateClearsRemovedValues removes values that the API only clears
// when an update sends them empty. The empty plan after the last step shows
// that they are gone.
func TestAccCheckUpdateClearsRemovedValues(t *testing.T) {
	config := func(values string) string {
		return fmt.Sprintf(`
			resource "checkly_snippet" "test" {
				name   = "tf-acc-check-clear"
				script = "console.log('hi')"
			}

			resource "checkly_check_group" "test" {
				name        = "tf-acc-check-clear"
				activated   = true
				concurrency = 1
				locations   = ["eu-central-1"]
			}

			resource "checkly_check" "test" {
				name                      = "tf-acc-check-clear"
				type                      = "API"
				activated                 = false
				frequency                 = 60
				locations                 = ["eu-central-1"]
				use_global_alert_settings = true
				%s

				request = {
					url = "https://api.checklyhq.com/public-stats"
				}
			}
		`, values)
	}
	const r = "checkly_check.test"

	accTestCase(t, []resource.TestStep{
		{
			Config: config(`
				tags                  = ["tf-acc"]
				setup_snippet_id      = checkly_snippet.test.id
				teardown_snippet_id   = checkly_snippet.test.id
				local_setup_script    = "console.log('setup')"
				local_teardown_script = "console.log('teardown')"
				group_id              = checkly_check_group.test.id
				group_order           = 3
			`),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(r, "tags.#", "1"),
				resource.TestCheckResourceAttr(r, "group_order", "3"),
			),
		},
		{
			Config: config(`group_id = checkly_check_group.test.id`),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(r, "tags.#"),
				resource.TestCheckResourceAttr(r, "setup_snippet_id", "0"),
				resource.TestCheckResourceAttr(r, "teardown_snippet_id", "0"),
				resource.TestCheckResourceAttr(r, "local_setup_script", ""),
				resource.TestCheckResourceAttr(r, "local_teardown_script", ""),
				resource.TestCheckResourceAttr(r, "group_order", "0"),
			),
		},
		{
			Config: config(""),
			Check:  resource.TestCheckResourceAttr(r, "group_id", "0"),
		},
	})
}

// TestAccCheckUpdateClearsScriptPath sets the script path and dependencies of
// a browser check outside Terraform. checkly_check has no attributes for them,
// so an update clears them.
// TestAccCheckEmptyPrivateLocations verifies that an explicitly empty
// private_locations is stored as an empty set and that leaving it out stores
// null, without drift in either case.
func TestAccCheckEmptyPrivateLocations(t *testing.T) {
	config := func(privateLocations string) string {
		return fmt.Sprintf(`
			resource "checkly_check" "test" {
				name                      = "tf-acc-check-empty-private-locations"
				type                      = "API"
				activated                 = false
				frequency                 = 60
				locations                 = ["eu-central-1"]
				use_global_alert_settings = true
				%s

				request = {
					url = "https://api.checklyhq.com/public-stats"
				}
			}
		`, privateLocations)
	}
	const r = "checkly_check.test"

	accTestCase(t, []resource.TestStep{
		{
			Config: config(`private_locations = []`),
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue(r, tfjsonpath.New("private_locations"), knownvalue.SetExact([]knownvalue.Check{})),
			},
		},
		{
			Config: config(""),
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue(r, tfjsonpath.New("private_locations"), knownvalue.Null()),
			},
		},
	})
}

func TestAccCheckUpdateClearsScriptPath(t *testing.T) {
	config := func(name string) string {
		return fmt.Sprintf(`
			resource "checkly_check" "test" {
				name                      = %q
				type                      = "BROWSER"
				activated                 = false
				frequency                 = 10
				locations                 = ["eu-central-1"]
				use_global_alert_settings = true
				script                    = "console.log('hi')"
			}
		`, name)
	}
	const r = "checkly_check.test"

	var id string
	accTestCase(t, []resource.TestStep{
		{
			Config: config("tf-acc-clear-script-path"),
			Check: func(s *terraform.State) error {
				id = s.RootModule().Resources[r].Primary.ID
				return nil
			},
		},
		{
			PreConfig: func() {
				ctx := context.Background()
				client := testAccClient()
				check, err := client.GetBrowserCheck(ctx, id)
				if err != nil {
					t.Fatal(err)
				}
				scriptPath := "tests/main.spec.js"
				check.ScriptPath = &scriptPath
				check.Dependencies = []checkly.ScriptDependency{{Path: "tests/helper.js", Content: "module.exports = {}"}}
				if _, err := client.UpdateBrowserCheck(ctx, id, *check); err != nil {
					t.Fatal(err)
				}
			},
			Config: config("tf-acc-clear-script-path-renamed"),
			Check: func(s *terraform.State) error {
				check, err := testAccClient().GetBrowserCheck(context.Background(), id)
				if err != nil {
					return err
				}
				if check.ScriptPath != nil {
					return fmt.Errorf("scriptPath = %q, want it cleared", *check.ScriptPath)
				}
				if len(check.Dependencies) != 0 {
					return fmt.Errorf("dependencies = %+v, want them cleared", check.Dependencies)
				}
				return nil
			},
		},
	})
}
