package provider

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// checkTestCheckAlertChannelSubscription verifies that the resource has an
// alert channel subscription for the alert channel resource identified by
// channelIDAttr, with the expected activation state.
func checkTestCheckAlertChannelSubscription(resourceName, blockPath string, channelIDAttr string, expectedActivated string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		// Resolve the channel_id from another resource's attribute
		channelRS, ok := s.RootModule().Resources[channelIDAttr]
		if !ok {
			return fmt.Errorf("resource not found: %s", channelIDAttr)
		}
		wantChannelID := channelRS.Primary.Attributes["id"]

		count := rs.Primary.Attributes[blockPath+".#"]
		n, _ := strconv.Atoi(count)
		for i := range n {
			prefix := fmt.Sprintf("%s.%d", blockPath, i)
			if rs.Primary.Attributes[prefix+".channel_id"] == wantChannelID {
				got := rs.Primary.Attributes[prefix+".activated"]
				if got != expectedActivated {
					return fmt.Errorf("channel %s: expected activated=%s, got %s", wantChannelID, expectedActivated, got)
				}
				return nil
			}
		}

		return fmt.Errorf("no alert_channel_subscription with channel_id=%s found", wantChannelID)
	}
}

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
			ExpectError: regexp.MustCompile(`unexpected response status 400`),
		},
	})
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
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"request.#",
					"0",
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
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.method",
					"GET",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.url",
					"https://api.checklyhq.com/public-stats",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.assertion.*.comparison",
					"EQUALS",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.assertion.*.property",
					"",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.assertion.*.source",
					"STATUS_CODE",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.assertion.*.target",
					"200",
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
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"request.#",
					"0",
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
				testCheckResourceAttrExpr(
					"checkly_check.test",
					`"request.*.headers.X-CUSTOM-1"`,
					"1",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.headers.X-CUSTOM-2",
					"FOO",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.query_parameters.param1",
					"123",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.query_parameters.param2",
					"bar",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.basic_auth.username",
					"user",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.basic_auth.password",
					"pass",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.assertion.#",
					"3",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.assertion.*.comparison",
					"EQUALS",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.assertion.*.comparison",
					"GREATER_THAN",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.assertion.*.target",
					"200",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.assertion.*.target",
					"no-cache",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.assertion.*.target",
					"100",
				),
			),
		},
	})
}

func TestAccApiCheckMore(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: apiCheck_post,
			Check: resource.ComposeTestCheckFunc(
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.method",
					"POST",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.body",
					`{\"message\":\"hello checkly\",\"messageId\":1}`,
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.body_type",
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

	trigger_incident {
		service_id         = checkly_status_page_service.test_service.id
		severity           = "MAJOR"
		name               = "API Check Failure Incident"
		description        = "The API health check has failed"
		notify_subscribers = true
	}

	request {
		url    = "https://api.example.com/health"
		method = "GET"
		assertion {
			source     = "STATUS_CODE"
			comparison = "EQUALS"
			target     = "200"
		}
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
					"trigger_incident.0.severity",
					"MAJOR",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test_trigger_incident",
					"trigger_incident.0.name",
					"API Check Failure Incident",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test_trigger_incident",
					"trigger_incident.0.description",
					"The API health check has failed",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test_trigger_incident",
					"trigger_incident.0.notify_subscribers",
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

	trigger_incident {
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
					"trigger_incident.0.severity",
					"CRITICAL",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test_browser_trigger",
					"trigger_incident.0.name",
					"Browser Check Critical Failure",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test_browser_trigger",
					"trigger_incident.0.description",
					"The browser check has detected a critical failure",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test_browser_trigger",
					"trigger_incident.0.notify_subscribers",
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
					request {
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
					request {
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
					request {
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
					request {
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
					request {
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
					request {
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
					request {
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
					request {
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
					request {
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
	  request {
		method     = "GET"
		url        = "https://api.checklyhq.com/public-stats"
		assertion {
		  comparison = "EQUALS"
		  property   = ""
		  source     = "STATUS_CODE"
		  target     = "200"
		}
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
	  request {
		method     = "GET"
		url        = "https://api.checklyhq.com/public-stats"
		assertion {
		  comparison = "EQUALS"
		  property   = ""
		  source     = "STATUS_CODE"
		  target     = "200"
		}
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
	request {
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
	  assertion {
		comparison = "EQUALS"
		property   = ""
		source     = "STATUS_CODE"
		target     = "200"
	  }
	  assertion {
		comparison = "EQUALS"
		property   = "cache-control"
		source     = "HEADERS"
		target     = "no-cache"
	  }
	  assertion {
		comparison = "GREATER_THAN"
		property   = "$.apiCheckResults"
		source     = "JSON_BODY"
		target     = "100"
	  }
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
	request {
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
	request {
	  follow_redirects = false
	  url              = "https://api.checklyhq.com/public-stats"
	  basic_auth = {
		username = ""
		password = ""
	  }
	  assertion {
		source     = "STATUS_CODE"
		property   = ""
		comparison = "EQUALS"
		target     = "200"
	  }
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
					request {
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
					request {
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
					request {
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
					request {
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
					request {
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
					request {
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

					environment_variable {
						key    = "FOO"
						value  = "bar"
						locked = false
					}

					environment_variable {
						key    = "SECRET"
						value  = "s3cr3t"
						locked = true
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"environment_variable.#",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"environment_variable.0.key",
					"FOO",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"environment_variable.0.value",
					"bar",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"environment_variable.0.locked",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"environment_variable.1.key",
					"SECRET",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"environment_variable.1.value",
					"s3cr3t",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"environment_variable.1.locked",
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

					environment_variable {
						key   = "FOO"
						value = "bar"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"environment_variable.#",
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
					"environment_variable.#",
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
					request {
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
					request {
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
					request {
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
					request {
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
					request {
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
					request {
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
					request {
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
					request {
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
					request {
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
					request {
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
					request {
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
					request {
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
					email {
						address = "check-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email {
						address = "check-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email {
						address = "check-acs-3@example.com"
					}
				}

				resource "checkly_check" "test" {
					name      = "check-alert-channel-sub-test"
					type      = "API"
					frequency = 60
					activated = true
					locations = ["eu-central-1"]

					request {
						method = "GET"
						url    = "https://api.checklyhq.com/public-stats"
					}

					alert_channel_subscription {
						channel_id = checkly_alert_channel.email_1.id
						activated  = true
					}

					alert_channel_subscription {
						channel_id = checkly_alert_channel.email_2.id
						activated  = false
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"alert_channel_subscription.#",
					"2",
				),
				checkTestCheckAlertChannelSubscription(
					"checkly_check.test",
					"alert_channel_subscription",
					"checkly_alert_channel.email_1",
					"true",
				),
				checkTestCheckAlertChannelSubscription(
					"checkly_check.test",
					"alert_channel_subscription",
					"checkly_alert_channel.email_2",
					"false",
				),
			),
		},
		// Step 2: Change activations and add a third subscription.
		{
			Config: `
				resource "checkly_alert_channel" "email_1" {
					email {
						address = "check-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email {
						address = "check-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email {
						address = "check-acs-3@example.com"
					}
				}

				resource "checkly_check" "test" {
					name      = "check-alert-channel-sub-test"
					type      = "API"
					frequency = 60
					activated = true
					locations = ["eu-central-1"]

					request {
						method = "GET"
						url    = "https://api.checklyhq.com/public-stats"
					}

					alert_channel_subscription {
						channel_id = checkly_alert_channel.email_1.id
						activated  = false
					}

					alert_channel_subscription {
						channel_id = checkly_alert_channel.email_2.id
						activated  = true
					}

					alert_channel_subscription {
						channel_id = checkly_alert_channel.email_3.id
						activated  = true
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"alert_channel_subscription.#",
					"3",
				),
				checkTestCheckAlertChannelSubscription(
					"checkly_check.test",
					"alert_channel_subscription",
					"checkly_alert_channel.email_1",
					"false",
				),
				checkTestCheckAlertChannelSubscription(
					"checkly_check.test",
					"alert_channel_subscription",
					"checkly_alert_channel.email_2",
					"true",
				),
				checkTestCheckAlertChannelSubscription(
					"checkly_check.test",
					"alert_channel_subscription",
					"checkly_alert_channel.email_3",
					"true",
				),
			),
		},
		// Step 3: Remove all alert channel subscriptions.
		{
			Config: `
				resource "checkly_alert_channel" "email_1" {
					email {
						address = "check-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email {
						address = "check-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email {
						address = "check-acs-3@example.com"
					}
				}

				resource "checkly_check" "test" {
					name      = "check-alert-channel-sub-test"
					type      = "API"
					frequency = 60
					activated = true
					locations = ["eu-central-1"]

					request {
						method = "GET"
						url    = "https://api.checklyhq.com/public-stats"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"alert_channel_subscription.#",
					"0",
				),
			),
		},
	})
}
