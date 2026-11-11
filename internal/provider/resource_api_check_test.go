package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
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
				resource.TestCheckTypeSetElemAttr(
					"checkly_api_check.test",
					"locations.*",
					"eu-central-1",
				),
				resource.TestCheckTypeSetElemAttr(
					"checkly_api_check.test",
					"locations.*",
					"us-east-1",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"request.method",
					"GET",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"request.url",
					"https://api.checklyhq.com/public-stats",
				),
				resource.TestCheckTypeSetElemNestedAttrs(
					"checkly_api_check.test",
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
		{
			ResourceName:      "checkly_api_check.test",
			ImportState:       true,
			ImportStateVerify: true,
		},
	})
}

// TestAccApiCheckResourceDefaults verifies the attribute defaults:
// activated defaults to true and frequency to 10, as on every check, and
// degraded_response_time to 5000 and max_response_time to 20000, which
// differ from the deprecated checkly_check resource.
func TestAccApiCheckResourceDefaults(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_api_check" "test" {
					name      = "API Check Resource Defaults"
					locations = ["eu-central-1"]
					request = {
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
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"description",
					"",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"request.method",
					"GET",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"request.body_type",
					"NONE",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"request.ip_family",
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
					request = {
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

					request = {
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

					request = {
						url = "https://api.checklyhq.com/public-stats"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(
					"checkly_api_check.test",
					"setup_snippet_id",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_api_check.test",
					"teardown_snippet_id",
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
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"locations.#",
					"3",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"request.headers.X-CUSTOM-1",
					"1",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"request.headers.X-CUSTOM-2",
					"foo",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"request.query_parameters.param1",
					"123",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"request.query_parameters.param2",
					"bar",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"request.basic_auth.username",
					"user",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"request.basic_auth.password",
					"pass",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"request.assertions.#",
					"3",
				),
				resource.TestCheckTypeSetElemNestedAttrs(
					"checkly_api_check.test",
					"request.assertions.*",
					map[string]string{
						"source":     "STATUS_CODE",
						"property":   "",
						"comparison": "EQUALS",
						"target":     "200",
					},
				),
				resource.TestCheckTypeSetElemNestedAttrs(
					"checkly_api_check.test",
					"request.assertions.*",
					map[string]string{
						"source":     "HEADERS",
						"property":   "cache-control",
						"comparison": "EQUALS",
						"target":     "no-cache",
					},
				),
				resource.TestCheckTypeSetElemNestedAttrs(
					"checkly_api_check.test",
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
	})
}

func TestAccApiCheckResourceMore(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: apiCheckResource_post,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"request.method",
					"POST",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"request.body",
					`{"message":"hello checkly","messageId":1}`,
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"request.body_type",
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

// An empty runtime_id, such as a variable defaulting to "", leaves the
// runtime unset. The API then reports no runtime, and the configured empty
// string must survive a refresh instead of planning an update.
func TestAccApiCheckResourceEmptyRuntimeID(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_api_check" "test" {
					name       = "api-check-resource-empty-runtime-id"
					activated  = false
					frequency  = 60
					locations  = ["eu-central-1"]
					runtime_id = ""
					request = {
						url = "https://welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.TestCheckResourceAttr("checkly_api_check.test", "runtime_id", ""),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PostApplyPostRefresh: []plancheck.PlanCheck{
					plancheck.ExpectEmptyPlan(),
				},
			},
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
					request = {
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
					request = {
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
					request = {
						url = "https://welcome.checklyhq.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`"frequency_offset" is required when "frequency" is 0`),
		},
		{
			Config: `
				resource "checkly_api_check" "test" {
					name             = "api-check-resource-disabled-offset"
					activated        = false
					frequency        = 0
					frequency_offset = 0
					locations        = ["eu-central-1"]
					request = {
						url = "https://welcome.checklyhq.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`"frequency_offset" is required when "frequency" is 0`),
		},
		{
			Config: `
				resource "checkly_api_check" "test" {
					name             = "api-check-resource-extra-offset"
					activated        = false
					frequency        = 60
					frequency_offset = 30
					locations        = ["eu-central-1"]
					request = {
						url = "https://welcome.checklyhq.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`"frequency_offset" can only be set when "frequency" is 0`),
		},
		{
			Config: `
				resource "checkly_api_check" "test" {
					name             = "api-check-resource-bad-offset"
					activated        = false
					frequency        = 0
					frequency_offset = 15
					locations        = ["eu-central-1"]
					request = {
						url = "https://welcome.checklyhq.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`Attribute frequency_offset value must be one of`),
		},
		{
			Config: `
				resource "checkly_api_check" "test-offset-10" {
					name             = "api-check-resource-frequency-0-offset-10"
					activated        = false
					frequency        = 0
					frequency_offset = 10
					locations        = ["eu-central-1"]
					request = {
						url = "https://welcome.checklyhq.com"
					}
				}

				resource "checkly_api_check" "test-offset-20" {
					name             = "api-check-resource-frequency-0-offset-20"
					activated        = false
					frequency        = 0
					frequency_offset = 20
					locations        = ["eu-central-1"]
					request = {
						url = "https://welcome.checklyhq.com"
					}
				}

				resource "checkly_api_check" "test-offset-30" {
					name             = "api-check-resource-frequency-0-offset-30"
					activated        = false
					frequency        = 0
					frequency_offset = 30
					locations        = ["eu-central-1"]
					request = {
						url = "https://welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_api_check.test-offset-10", "frequency", "0"),
				resource.TestCheckResourceAttr("checkly_api_check.test-offset-10", "frequency_offset", "10"),
				resource.TestCheckResourceAttr("checkly_api_check.test-offset-20", "frequency", "0"),
				resource.TestCheckResourceAttr("checkly_api_check.test-offset-20", "frequency_offset", "20"),
				resource.TestCheckResourceAttr("checkly_api_check.test-offset-30", "frequency", "0"),
				resource.TestCheckResourceAttr("checkly_api_check.test-offset-30", "frequency_offset", "30"),
			),
		},
		{
			// A frequency_offset of 0 is equivalent to leaving it unset and is
			// kept in state even though the API reports its own offset for
			// regular frequencies.
			Config: `
				resource "checkly_api_check" "test-disabled-offset" {
					name             = "api-check-resource-frequency-60-offset-0"
					activated        = false
					frequency        = 60
					frequency_offset = 0
					locations        = ["eu-central-1"]
					request = {
						url = "https://welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_api_check.test-disabled-offset", "frequency", "60"),
				resource.TestCheckResourceAttr("checkly_api_check.test-disabled-offset", "frequency_offset", "0"),
			),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PostApplyPostRefresh: []plancheck.PlanCheck{
					plancheck.ExpectEmptyPlan(),
				},
			},
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

// TestAccApiCheckUnsetSweepScalars sets every optional scalar toggle to a
// non-default value and then removes them all, verifying the update endpoint
// resets them to their defaults (the endpoint is PATCH-style, so explicit
// values must be sent for the reset to take effect).
func TestAccApiCheckUnsetSweepScalars(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_api_check" "test" {
					name                      = "api-check-unset-sweep-scalars"
					frequency                 = 60
					activated                 = false
					muted                     = true
					should_fail               = true
					run_parallel              = true
					use_global_alert_settings = false
					runtime_id                = "2023.09"
					locations                 = ["eu-central-1"]
					request = {
						url = "https://api.checklyhq.com/public-stats"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"activated",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"muted",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"should_fail",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"run_parallel",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"use_global_alert_settings",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"runtime_id",
					"2023.09",
				),
			),
		},
		{
			Config: `
				resource "checkly_api_check" "test" {
					name      = "api-check-unset-sweep-scalars"
					frequency = 60
					locations = ["eu-central-1"]
					request = {
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
					"muted",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"should_fail",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"run_parallel",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"use_global_alert_settings",
					"true",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_api_check.test",
					"runtime_id",
				),
			),
		},
	})
}

// TestAccApiCheckUnsetSweepCollections sets the tags, locations and
// private_locations collections and then removes them, verifying that removal
// clears the values remotely (the post-apply refresh would report remote
// leftovers as drift and fail the step).
func TestAccApiCheckUnsetSweepCollections(t *testing.T) {
	// Private location slugs are unique per account and limited to 30
	// characters, so a short random suffix avoids collisions between runs.
	slug := fmt.Sprintf("tf-api-unset-%d", acctest.RandInt()%100000)
	privateLocation := fmt.Sprintf(`
		resource "checkly_private_location" "sweep" {
			name      = %[1]q
			slug_name = %[1]q
			icon      = "bell-fill"
		}
	`, slug)
	accTestCase(t, []resource.TestStep{
		{
			Config: privateLocation + `
				resource "checkly_api_check" "test" {
					name              = "api-check-unset-sweep-collections"
					activated         = false
					frequency         = 60
					locations         = ["eu-central-1", "eu-west-1"]
					private_locations = [checkly_private_location.sweep.slug_name]
					tags              = ["sweep-a", "sweep-b"]
					request = {
						url = "https://api.checklyhq.com/public-stats"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"locations.#",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"private_locations.#",
					"1",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"tags.#",
					"2",
				),
			),
		},
		// Removing tags and locations must clear them remotely. The check
		// keeps running from the private location.
		{
			Config: privateLocation + `
				resource "checkly_api_check" "test" {
					name              = "api-check-unset-sweep-collections"
					activated         = false
					frequency         = 60
					private_locations = [checkly_private_location.sweep.slug_name]
					request = {
						url = "https://api.checklyhq.com/public-stats"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(
					"checkly_api_check.test",
					"locations.#",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_api_check.test",
					"tags.#",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"private_locations.#",
					"1",
				),
			),
		},
		// An explicitly empty private_locations is stored as an empty set.
		// The check needs a public location again to remain valid.
		{
			Config: privateLocation + `
				resource "checkly_api_check" "test" {
					name              = "api-check-unset-sweep-collections"
					activated         = false
					frequency         = 60
					locations         = ["eu-central-1"]
					private_locations = []
					request = {
						url = "https://api.checklyhq.com/public-stats"
					}
				}
			`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue(
					"checkly_api_check.test",
					tfjsonpath.New("private_locations"),
					knownvalue.SetExact([]knownvalue.Check{}),
				),
			},
		},
		// Leaving private_locations out stores null.
		{
			Config: privateLocation + `
				resource "checkly_api_check" "test" {
					name      = "api-check-unset-sweep-collections"
					activated = false
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						url = "https://api.checklyhq.com/public-stats"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"locations.#",
					"1",
				),
			),
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue(
					"checkly_api_check.test",
					tfjsonpath.New("private_locations"),
					knownvalue.Null(),
				),
			},
		},
	})
}

const apiCheckUnsetSweepDeps = `
	resource "checkly_check_group" "sweep" {
		name        = "api-check-unset-sweep-group"
		activated   = true
		concurrency = 1
		locations   = ["eu-central-1"]
	}

	resource "checkly_alert_channel" "sweep" {
		email = {
			address = "api-check-unset-sweep@example.com"
		}
	}

	resource "checkly_status_page_service" "sweep" {
		name = "api-check-unset-sweep-service"
	}
`

// TestAccApiCheckUnsetSweepGroupAndAlerts assigns the check to a group (with
// an explicit group order), subscribes it to an alert channel and attaches an
// incident trigger, then removes them one by one, verifying the removals
// clear the values remotely.
func TestAccApiCheckUnsetSweepGroupAndAlerts(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: apiCheckUnsetSweepDeps + `
				resource "checkly_api_check" "test" {
					name        = "api-check-unset-sweep-group-alerts"
					activated   = false
					frequency   = 60
					locations   = ["eu-central-1"]
					group_id    = checkly_check_group.sweep.id
					group_order = 2

					alert_channel_subscriptions = [
						{
							channel_id = checkly_alert_channel.sweep.id
							activated  = true
						},
					]

					trigger_incident = {
						service_id         = checkly_status_page_service.sweep.id
						severity           = "MAJOR"
						name               = "API check down"
						description        = "The API check has detected a failure"
						notify_subscribers = false
					}

					request = {
						url = "https://api.checklyhq.com/public-stats"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrPair(
					"checkly_api_check.test",
					"group_id",
					"checkly_check_group.sweep",
					"id",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"group_order",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"alert_channel_subscriptions.#",
					"1",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"trigger_incident.%",
					"5",
				),
			),
		},
		// Removing group_order alone must clear the position remotely while the
		// check stays in the group.
		{
			Config: apiCheckUnsetSweepDeps + `
				resource "checkly_api_check" "test" {
					name      = "api-check-unset-sweep-group-alerts"
					activated = false
					frequency = 60
					locations = ["eu-central-1"]
					group_id  = checkly_check_group.sweep.id

					alert_channel_subscriptions = [
						{
							channel_id = checkly_alert_channel.sweep.id
							activated  = true
						},
					]

					trigger_incident = {
						service_id         = checkly_status_page_service.sweep.id
						severity           = "MAJOR"
						name               = "API check down"
						description        = "The API check has detected a failure"
						notify_subscribers = false
					}

					request = {
						url = "https://api.checklyhq.com/public-stats"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrPair(
					"checkly_api_check.test",
					"group_id",
					"checkly_check_group.sweep",
					"id",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_api_check.test",
					"group_order",
				),
			),
		},
		// Removing the group assignment, the alert channel subscription and
		// the incident trigger must clear them all remotely.
		{
			Config: apiCheckUnsetSweepDeps + `
				resource "checkly_api_check" "test" {
					name      = "api-check-unset-sweep-group-alerts"
					activated = false
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						url = "https://api.checklyhq.com/public-stats"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(
					"checkly_api_check.test",
					"group_id",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_api_check.test",
					"group_order",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"alert_channel_subscriptions.#",
					"0",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_api_check.test",
					"trigger_incident",
				),
			),
		},
	})
}

// TestAccApiCheckUnsetSweepRequest sets the optional request sub-properties
// (method, headers, query parameters, body, basic auth, extra assertions) and
// then removes them, verifying the update clears them remotely inside the
// nested request object.
func TestAccApiCheckUnsetSweepRequest(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_api_check" "test" {
					name      = "api-check-unset-sweep-request"
					activated = false
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						method = "POST"
						url    = "https://api.checklyhq.com/public-stats"
						headers = {
							X-Sweep-1 = "one"
							X-Sweep-2 = "two"
						}
						query_parameters = {
							p1 = "1"
							p2 = "2"
						}
						body      = "{\"hello\":\"world\"}"
						body_type = "JSON"
						basic_auth = {
							username = "user"
							password = "pass"
						}
						assertions = [{
							source     = "STATUS_CODE"
							comparison = "EQUALS"
							target     = "200"
						}, {
							source     = "RESPONSE_TIME"
							comparison = "LESS_THAN"
							target     = "5000"
						}]
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"request.method",
					"POST",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"request.headers.%",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"request.query_parameters.%",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"request.basic_auth.username",
					"user",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"request.assertions.#",
					"2",
				),
			),
		},
		{
			Config: `
				resource "checkly_api_check" "test" {
					name      = "api-check-unset-sweep-request"
					activated = false
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						url = "https://api.checklyhq.com/public-stats"
						assertions = [{
							source     = "STATUS_CODE"
							comparison = "EQUALS"
							target     = "200"
						}]
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"request.method",
					"GET",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"request.body",
					"",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"request.body_type",
					"NONE",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"request.assertions.#",
					"1",
				),
			),
		},
	})
}

// TestAccApiCheckUnsetSweepAlertSettingsReset verifies that an explicit
// change of the alert settings escalation type back to RUN_BASED is honored
// by the server. Removal of alert_settings deliberately keeps the prior state
// (UseStateForUnknown), so removal cannot be used to test clearing.
func TestAccApiCheckUnsetSweepAlertSettingsReset(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_api_check" "test" {
					name      = "api-check-unset-sweep-alert-settings"
					activated = false
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						url = "https://api.checklyhq.com/public-stats"
					}
					alert_settings = {
						escalation_type = "TIME_BASED"
						time_based_escalation = {
							minutes_failing_threshold = 10
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"alert_settings.escalation_type",
					"TIME_BASED",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"alert_settings.time_based_escalation.minutes_failing_threshold",
					"10",
				),
			),
		},
		{
			Config: `
				resource "checkly_api_check" "test" {
					name      = "api-check-unset-sweep-alert-settings"
					activated = false
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						url = "https://api.checklyhq.com/public-stats"
					}
					alert_settings = {
						escalation_type = "RUN_BASED"
						run_based_escalation = {
							failed_run_threshold = 2
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"alert_settings.escalation_type",
					"RUN_BASED",
				),
				resource.TestCheckResourceAttr(
					"checkly_api_check.test",
					"alert_settings.run_based_escalation.failed_run_threshold",
					"2",
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

const apiCheckResource_basicWithoutDescription = `
	resource "checkly_api_check" "test" {
	  name                      = "API Check Resource 1"
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

const apiCheckResource_post = `
  resource "checkly_api_check" "test" {
	name         = "apiCheckResource_post"
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
