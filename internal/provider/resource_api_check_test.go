package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccApiCheckResourceBasic(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: apiCheckResource_basic,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"name",
					"API Check Resource 1",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"description",
					"API check description",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"activated",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"degraded_response_time",
					"5000",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"max_response_time",
					"18000",
				),
				testCheckResourceAttrExpr(
					"checkly_api_check.test",
					"locations.*",
					"eu-central-1",
				),
				testCheckResourceAttrExpr(
					"checkly_api_check.test",
					"locations.*",
					"us-east-1",
				),
				testCheckResourceAttrExpr(
					"checkly_api_check.test",
					"request.*.method",
					"GET",
				),
				testCheckResourceAttrExpr(
					"checkly_api_check.test",
					"request.*.url",
					"https://api.checklyhq.com/public-stats",
				),
				testCheckResourceAttrExpr(
					"checkly_api_check.test",
					"request.*.assertion.*.comparison",
					"EQUALS",
				),
				testCheckResourceAttrExpr(
					"checkly_api_check.test",
					"request.*.assertion.*.property",
					"",
				),
				testCheckResourceAttrExpr(
					"checkly_api_check.test",
					"request.*.assertion.*.source",
					"STATUS_CODE",
				),
				testCheckResourceAttrExpr(
					"checkly_api_check.test",
					"request.*.assertion.*.target",
					"200",
				),
			),
		},
		{
			ResourceName:      "checkly_api_check.test",
			ImportState:       true,
			ImportStateVerify: true,
		},
	})
}

// TestAccApiCheckResourceDefaults verifies the attribute defaults that
// differ from the deprecated checkly_check resource: activated defaults to
// true, frequency to 10, degraded_response_time to 5000 and
// max_response_time to 20000.
func TestAccApiCheckResourceDefaults(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_api_check" "test" {
					name      = "API Check Resource Defaults"
					locations = ["eu-central-1"]
					request {
						url = "https://api.checklyhq.com/public-stats"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"activated",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"frequency",
					"10",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"degraded_response_time",
					"5000",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"max_response_time",
					"20000",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"use_global_alert_settings",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"description",
					"",
				),
				testCheckResourceAttrExpr(
					"checkly_api_check.test",
					"request.*.method",
					"GET",
				),
				testCheckResourceAttrExpr(
					"checkly_api_check.test",
					"request.*.body_type",
					"NONE",
				),
				testCheckResourceAttrExpr(
					"checkly_api_check.test",
					"request.*.ip_family",
					"IPv4",
				),
			),
		},
		// A degraded_response_time of 0 is a valid value and must round-trip
		// instead of being treated as unset.
		{
			Config: `
				resource "checkly_api_check" "test" {
					name                   = "API Check Resource Defaults"
					degraded_response_time = 0
					locations              = ["eu-central-1"]
					request {
						url = "https://api.checklyhq.com/public-stats"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"degraded_response_time",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"max_response_time",
					"20000",
				),
			),
		},
	})
}

// TestAccApiCheckResourceSnippetsAndScripts exercises the snippet and local
// script attributes: setting them, then clearing them back to their defaults
// (0 for snippet IDs, "" for scripts), which must unset them via the API.
func TestAccApiCheckResourceSnippetsAndScripts(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_snippet" "test" {
					name   = "api-check-resource-snippet"
					script = "console.log('snippet')"
				}

				resource "checkly_api_check" "test" {
					name      = "api-check-resource-snippets"
					activated = false
					frequency = 60
					locations = ["eu-central-1"]

					setup_snippet_id      = checkly_snippet.test.id
					teardown_snippet_id   = checkly_snippet.test.id
					local_setup_script    = "console.log('setup')"
					local_teardown_script = "console.log('teardown')"

					request {
						url = "https://api.checklyhq.com/public-stats"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrPair(
					"checkly_api_check.test",
					"setup_snippet_id",
					"checkly_snippet.test",
					"id",
				),
				resource.TestCheckResourceAttrPair(
					"checkly_api_check.test",
					"teardown_snippet_id",
					"checkly_snippet.test",
					"id",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"local_setup_script",
					"console.log('setup')",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"local_teardown_script",
					"console.log('teardown')",
				),
			),
		},
		{
			Config: `
				resource "checkly_snippet" "test" {
					name   = "api-check-resource-snippet"
					script = "console.log('snippet')"
				}

				resource "checkly_api_check" "test" {
					name      = "api-check-resource-snippets"
					activated = false
					frequency = 60
					locations = ["eu-central-1"]

					request {
						url = "https://api.checklyhq.com/public-stats"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"setup_snippet_id",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"teardown_snippet_id",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"local_setup_script",
					"",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"local_teardown_script",
					"",
				),
			),
		},
	})
}

func TestAccApiCheckResourceFull(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: apiCheckResource_full,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"degraded_response_time",
					"15000",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"max_response_time",
					"30000",
				),
				testCheckResourceAttrExpr(
					"checkly_api_check.test",
					`"locations.#"`,
					"3",
				),
				testCheckResourceAttrExpr(
					"checkly_api_check.test",
					`"request.*.headers.X-CUSTOM-1"`,
					"1",
				),
				testCheckResourceAttrExpr(
					"checkly_api_check.test",
					"request.*.headers.X-CUSTOM-2",
					"FOO",
				),
				testCheckResourceAttrExpr(
					"checkly_api_check.test",
					"request.*.query_parameters.param1",
					"123",
				),
				testCheckResourceAttrExpr(
					"checkly_api_check.test",
					"request.*.query_parameters.param2",
					"bar",
				),
				testCheckResourceAttrExpr(
					"checkly_api_check.test",
					"request.*.basic_auth.username",
					"user",
				),
				testCheckResourceAttrExpr(
					"checkly_api_check.test",
					"request.*.basic_auth.password",
					"pass",
				),
				testCheckResourceAttrExpr(
					"checkly_api_check.test",
					"request.*.assertion.#",
					"3",
				),
				testCheckResourceAttrExpr(
					"checkly_api_check.test",
					"request.*.assertion.*.comparison",
					"EQUALS",
				),
				testCheckResourceAttrExpr(
					"checkly_api_check.test",
					"request.*.assertion.*.comparison",
					"GREATER_THAN",
				),
				testCheckResourceAttrExpr(
					"checkly_api_check.test",
					"request.*.assertion.*.target",
					"200",
				),
				testCheckResourceAttrExpr(
					"checkly_api_check.test",
					"request.*.assertion.*.target",
					"no-cache",
				),
				testCheckResourceAttrExpr(
					"checkly_api_check.test",
					"request.*.assertion.*.target",
					"100",
				),
			),
		},
	})
}

func TestAccApiCheckResourceMore(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: apiCheckResource_post,
			Check: resource.ComposeTestCheckFunc(
				testCheckResourceAttrExpr(
					"checkly_api_check.test",
					"request.*.method",
					"POST",
				),
				testCheckResourceAttrExpr(
					"checkly_api_check.test",
					"request.*.body",
					`{\"message\":\"hello checkly\",\"messageId\":1}`,
				),
				testCheckResourceAttrExpr(
					"checkly_api_check.test",
					"request.*.body_type",
					"JSON",
				),
			),
		},
		{
			Config: apiCheckResource_withEmptyBasicAuth,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"name",
					"api check resource with empty basic_auth",
				),
			),
		},
	})
}

func TestAccApiCheckResourceDescriptionRemoval(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: apiCheckResource_basic,
			Check: resource.TestCheckResourceAttr(
				"checkly_api_check.test",
				"description",
				"API check description",
			),
		},
		{
			Config: apiCheckResource_basicWithoutDescription,
			Check: resource.TestCheckResourceAttr(
				"checkly_api_check.test",
				"description",
				"",
			),
		},
	})
}

func TestAccApiCheckResourceRetryStrategies(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_api_check" "test" {
					name      = "api-check-resource-retries"
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
					"checkly_api_check.test",
					"retry_strategy.type",
					"SINGLE_RETRY",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"retry_strategy.base_backoff_seconds",
					"30",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"retry_strategy.same_region",
					"true",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_api_check.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
		{
			Config: `
				resource "checkly_api_check" "test" {
					name      = "api-check-resource-retries"
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

						only_on = {
							network_error = true
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"retry_strategy.type",
					"FIXED",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"retry_strategy.max_retries",
					"3",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"retry_strategy.max_duration_seconds",
					"300",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"retry_strategy.same_region",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"retry_strategy.only_on.network_error",
					"true",
				),
			),
		},
		{
			Config: `
				resource "checkly_api_check" "test" {
					name      = "api-check-resource-retries"
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
					"checkly_api_check.test",
					"retry_strategy.type",
					"NO_RETRIES",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_api_check.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
	})
}

func TestAccApiCheckResourceFrequencyValidation(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_api_check" "test" {
					name      = "api-check-resource-bad-frequency"
					activated = false
					frequency = 9999
					locations = ["eu-central-1"]
					request {
						url = "https://welcome.checklyhq.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`value must be one of`),
		},
		{
			Config: `
				resource "checkly_api_check" "test" {
					name      = "api-check-resource-missing-offset"
					activated = false
					frequency = 0
					locations = ["eu-central-1"]
					request {
						url = "https://welcome.checklyhq.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`"frequency_offset" is required when "frequency" is 0`),
		},
		{
			Config: `
				resource "checkly_api_check" "test" {
					name             = "api-check-resource-bad-high-frequency-offset"
					activated        = false
					frequency        = 0
					frequency_offset = 15
					locations        = ["eu-central-1"]
					request {
						url = "https://welcome.checklyhq.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`must be 10, 20 or 30 when "frequency" is 0`),
		},
		{
			Config: `
				resource "checkly_api_check" "test" {
					name             = "api-check-resource-offset-too-small"
					activated        = false
					frequency        = 10
					frequency_offset = 0
					locations        = ["eu-central-1"]
					request {
						url = "https://welcome.checklyhq.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`value must be at least 1`),
		},
		{
			Config: `
				resource "checkly_api_check" "test" {
					name             = "api-check-resource-offset-too-large-low-frequency"
					activated        = false
					frequency        = 10
					frequency_offset = 101
					locations        = ["eu-central-1"]
					request {
						url = "https://welcome.checklyhq.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`must be between 1 and 100`),
		},
		{
			Config: `
				resource "checkly_api_check" "test" {
					name             = "api-check-resource-offset-too-large-high-frequency"
					activated        = false
					frequency        = 120
					frequency_offset = 25
					locations        = ["eu-central-1"]
					request {
						url = "https://welcome.checklyhq.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`must be between 1 and 24`),
		},
		{
			Config: `
				resource "checkly_api_check" "test-high-frequency" {
					name             = "api-check-resource-frequency-0-offset-20"
					activated        = false
					frequency        = 0
					frequency_offset = 20
					locations        = ["eu-central-1"]
					request {
						url = "https://welcome.checklyhq.com"
					}
				}

				resource "checkly_api_check" "test-low-frequency" {
					name             = "api-check-resource-frequency-10-offset-100"
					activated        = false
					frequency        = 10
					frequency_offset = 100
					locations        = ["eu-central-1"]
					request {
						url = "https://welcome.checklyhq.com"
					}
				}

				resource "checkly_api_check" "test-slow-frequency" {
					name             = "api-check-resource-frequency-120-offset-24"
					activated        = false
					frequency        = 120
					frequency_offset = 24
					locations        = ["eu-central-1"]
					request {
						url = "https://welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_api_check.test-high-frequency",
					"frequency",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test-high-frequency",
					"frequency_offset",
					"20",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test-low-frequency",
					"frequency",
					"10",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test-low-frequency",
					"frequency_offset",
					"100",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test-slow-frequency",
					"frequency",
					"120",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test-slow-frequency",
					"frequency_offset",
					"24",
				),
			),
		},
	})
}

// TestAccApiCheckResourceMovedFromCheck exercises the MoveState
// implementation of checkly_api_check via a `moved` block from
// checkly_check.
func TestAccApiCheckResourceMovedFromCheck(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check" "test" {
				  name                      = "API Check To Be Moved"
				  type                      = "API"
				  frequency                 = 60
				  activated                 = true
				  muted                     = true
				  degraded_response_time    = 15000
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
			`,
		},
		{
			Config: `
				resource "checkly_api_check" "test" {
				  name                      = "API Check To Be Moved"
				  frequency                 = 60
				  activated                 = true
				  muted                     = true
				  degraded_response_time    = 15000
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

				moved {
					from = checkly_check.test
					to   = checkly_api_check.test
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"name",
					"API Check To Be Moved",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"activated",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"muted",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"degraded_response_time",
					"15000",
				),
			),
		},
	})
}

const apiCheckResource_basic = `
	resource "checkly_api_check" "test" {
	  name                      = "API Check Resource 1"
	  description               = "API check description"
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

const apiCheckResource_basicWithoutDescription = `
	resource "checkly_api_check" "test" {
	  name                      = "API Check Resource 1"
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

const apiCheckResource_full = `
  resource "checkly_api_check" "test" {
	name                   = "apiCheckResource_full"
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

const apiCheckResource_post = `
  resource "checkly_api_check" "test" {
	name         = "apiCheckResource_post"
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

const apiCheckResource_withEmptyBasicAuth = `
  resource "checkly_api_check" "test" {
	name                   = "api check resource with empty basic_auth"
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
