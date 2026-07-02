package provider

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// checkGroupTestCheckAlertChannelSubscription checks that an alert channel
// subscription collection contains a subscription for the channel created by
// channelResourceName with the expected activation state.
func checkGroupTestCheckAlertChannelSubscription(resourceName, collectionPath, channelResourceName, expectedActivated string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		// Resolve the channel_id from another resource's attribute
		channelRS, ok := s.RootModule().Resources[channelResourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", channelResourceName)
		}
		wantChannelID := channelRS.Primary.Attributes["id"]

		count := rs.Primary.Attributes[collectionPath+".#"]
		n, _ := strconv.Atoi(count)
		for i := range n {
			prefix := fmt.Sprintf("%s.%d", collectionPath, i)
			if rs.Primary.Attributes[prefix+".channel_id"] == wantChannelID {
				got := rs.Primary.Attributes[prefix+".activated"]
				if got != expectedActivated {
					return fmt.Errorf("channel %s: expected activated=%s, got %s", wantChannelID, expectedActivated, got)
				}
				return nil
			}
		}

		return fmt.Errorf("no alert channel subscription with channel_id=%s found", wantChannelID)
	}
}

func TestAccCheckGroupEmptyConfig(t *testing.T) {
	config := `resource "checkly_check_group" "test" {}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "name" is required`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "concurrency" is required`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "activated" is required`),
		},
	})
}

func TestAccCheckGroupInvalid(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config:      testCheckGroup_invalid,
			ExpectError: regexp.MustCompile(`Inappropriate value for attribute "locations"`),
		},
		{
			Config:      testCheckGroup_invalid,
			ExpectError: regexp.MustCompile(`Inappropriate value for attribute "muted"`),
		},
		{
			Config:      testCheckGroup_invalid,
			ExpectError: regexp.MustCompile(`Inappropriate value for attribute "activated"`),
		},
		{
			Config:      testCheckGroup_invalid,
			ExpectError: regexp.MustCompile(`The argument "concurrency" is required`),
		},
		{
			Config:      testCheckGroup_invalid,
			ExpectError: regexp.MustCompile(`Missing required argument`),
		},
	})
}

func TestAccCheckGroupBasic(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: testCheckGroup_basic,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"name",
					"test",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"activated",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"muted",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"concurrency",
					"3",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"locations.#",
					"2",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"locations.*",
					"us-east-1",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"locations.*",
					"eu-central-1",
				),
			),
		},
	})
}

func TestAccCheckGroupWithApiDefaults(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: testCheckGroup_withApiDefaults,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"name",
					"test",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"api_check_defaults.url",
					"http://api.example.com/",
				),
			),
		},
	})
}

func TestAccCheckGroupFull(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: testCheckGroup_full,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"name",
					"test",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"activated",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"muted",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"concurrency",
					"3",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"use_global_alert_settings",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"locations.#",
					"2",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"locations.*",
					"us-east-1",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"locations.*",
					"eu-central-1",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"alert_settings.escalation_type",
					"RUN_BASED",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"alert_settings.reminders.amount",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"alert_settings.reminders.interval",
					"5",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"alert_settings.run_based_escalation.failed_run_threshold",
					"1",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"api_check_defaults.assertion.#",
					"1",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"api_check_defaults.assertion.*.comparison",
					"EQUALS",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"api_check_defaults.assertion.*.property",
					"",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"api_check_defaults.assertion.*.source",
					"STATUS_CODE",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"api_check_defaults.assertion.*.target",
					"200",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"api_check_defaults.basic_auth.password",
					"pass",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"api_check_defaults.basic_auth.username",
					"user",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"api_check_defaults.headers.X-Test",
					"foo",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"api_check_defaults.query_parameters.query",
					"foo",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"api_check_defaults.url",
					"http://example.com/",
				),
			),
		},
	})
}

const testCheckGroup_invalid = `
	resource "checkly_check_group" "test" {
		name = "test"
		activated = "invalid"
		muted = "invalid"
		locations = "invalid"
	}
`

const testCheckGroup_basic = `
	resource "checkly_check_group" "test" {
		name        = "test"
		activated   = true
		muted       = false
		concurrency = 3
		locations   = [
			"us-east-1",
			"eu-central-1",
		]
	}
`

const testCheckGroup_withApiDefaults = `
	resource "checkly_check_group" "test" {
		name        = "test"
		activated   = true
		muted       = false
		concurrency = 3
		locations   = [
			"eu-west-1",
			"eu-west-2"
		]
		api_check_defaults = {
			url = "http://api.example.com/"
		}
	}
`

const testCheckGroup_full = `
  resource "checkly_check_group" "test" {
	name                      = "test"
	activated                 = true
	muted                     = false
	concurrency               = 3
	use_global_alert_settings = false
	locations = [ "us-east-1", "eu-central-1" ]
	api_check_defaults = {
	  url = "http://example.com/"
	  headers = {
		X-Test = "foo"
	  }
	  query_parameters = {
		query = "foo"
	  }
	  assertion = [
		{
		  source     = "STATUS_CODE"
		  property   = ""
		  comparison = "EQUALS"
		  target     = "200"
		}
	  ]
	  basic_auth = {
		username = "user"
		password = "pass"
	  }
	}
	alert_settings = {
	  escalation_type = "RUN_BASED"
	  run_based_escalation = {
		failed_run_threshold = 1
	  }
	  reminders = {
		amount   = 2
		interval = 5
	  }
	  parallel_run_failure_threshold = {
		enabled = false
		percentage = 10
	  }
	}
	local_setup_script    = "setup-test"
	local_teardown_script = "teardown-test"
  }
`

func TestAccCheckGroupWithEnvironmentVariable(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "test-env-var"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]

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
					"checkly_check_group.test",
					"environment_variable.#",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"environment_variable.0.key",
					"FOO",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"environment_variable.0.value",
					"bar",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"environment_variable.0.locked",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"environment_variable.1.key",
					"SECRET",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"environment_variable.1.value",
					"s3cr3t",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"environment_variable.1.locked",
					"true",
				),
			),
		},
	})
}

func TestAccCheckGroupEnvironmentVariableRemoval(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "test-env-var-removal"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]

					environment_variable {
						key   = "FOO"
						value = "bar"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"environment_variable.#",
					"1",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "test-env-var-removal"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"environment_variable.#",
					"0",
				),
			),
		},
	})
}

func TestAccCheckGroupWithSingleRetry(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "test-single-retry"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
					retry_strategy = {
						type                 = "SINGLE_RETRY"
						base_backoff_seconds = 30
						same_region          = true
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.type",
					"SINGLE_RETRY",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.base_backoff_seconds",
					"30",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.max_retries",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.max_duration_seconds",
					"600",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.same_region",
					"true",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
	})
}

func TestAccCheckGroupWithSingleRetryOnlyOnNetworkError(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "test-single-retry"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
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
					"checkly_check_group.test",
					"retry_strategy.only_on.network_error",
					"true",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "test-single-retry"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
					retry_strategy = {
						type = "SINGLE_RETRY"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
	})
}

func TestAccCheckGroupWithNoRetries(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "test-no-retries"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
					retry_strategy = {
						type = "NO_RETRIES"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.type",
					"NO_RETRIES",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.base_backoff_seconds",
					"60",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.max_retries",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.max_duration_seconds",
					"600",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.same_region",
					"true",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
	})
}

func TestAccCheckGroupWithNoRetriesOnlyOnNetworkError(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "test-no-retries"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
					retry_strategy = {
						type = "NO_RETRIES"

						only_on = {
							network_error = true
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				// The API tracks no retry conditions for the NO_RETRIES
				// strategy; the configured value is preserved as-is.
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.only_on.network_error",
					"true",
				),
			),
		},
	})
}

func TestAccCheckGroupWithDefaultNoRetries(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "test-no-retries"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.type",
					"NO_RETRIES",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.base_backoff_seconds",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.max_retries",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.max_duration_seconds",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.same_region",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
	})
}

func TestAccCheckGroupWithFixedRetry(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "test-fixed-retry"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
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
					"checkly_check_group.test",
					"retry_strategy.type",
					"FIXED",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.base_backoff_seconds",
					"30",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.max_retries",
					"3",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.max_duration_seconds",
					"300",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.same_region",
					"false",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
	})
}

func TestAccCheckGroupWithFixedRetryOnlyOnNetworkError(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "test-fixed-retry"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
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
					"checkly_check_group.test",
					"retry_strategy.only_on.network_error",
					"true",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "test-fixed-retry"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
					retry_strategy = {
						type = "FIXED"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
	})
}

func TestAccCheckGroupWithExponentialRetry(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "test-exponential-retry"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
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
					"checkly_check_group.test",
					"retry_strategy.type",
					"EXPONENTIAL",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.base_backoff_seconds",
					"60",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.max_retries",
					"5",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.max_duration_seconds",
					"600",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.same_region",
					"true",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
	})
}

func TestAccCheckGroupWithExponentialRetryOnlyOnNetworkError(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "test-exponential-retry"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
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
					"checkly_check_group.test",
					"retry_strategy.only_on.network_error",
					"true",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "test-exponential-retry"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
					retry_strategy = {
						type = "EXPONENTIAL"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
	})
}

func TestAccCheckGroupWithLinearRetry(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "test-linear-retry"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
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
					"checkly_check_group.test",
					"retry_strategy.type",
					"LINEAR",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.base_backoff_seconds",
					"45",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.max_retries",
					"4",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.max_duration_seconds",
					"450",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.same_region",
					"false",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
	})
}

func TestAccCheckGroupWithLinearRetryOnlyOnNetworkError(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "test-linear-retry"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
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
					"checkly_check_group.test",
					"retry_strategy.only_on.network_error",
					"true",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "test-linear-retry"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
					retry_strategy = {
						type = "LINEAR"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
	})
}

func TestAccCheckGroupRetryStrategyRemoval(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "test-linear-retry"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
					retry_strategy = {
						type = "LINEAR"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.type",
					"LINEAR",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "test-linear-retry"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.type",
					"NO_RETRIES",
				),
			),
		},
	})
}

func TestAccCheckGroupAlertChannelSubscriptions(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		// Step 1: Add two alert channel subscriptions.
		{
			Config: `
				resource "checkly_alert_channel" "email_1" {
					email {
						address = "group-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email {
						address = "group-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email {
						address = "group-acs-3@example.com"
					}
				}

				resource "checkly_check_group" "test" {
					name        = "group-alert-channel-sub-test"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]

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
					"checkly_check_group.test",
					"alert_channel_subscription.#",
					"2",
				),
				checkGroupTestCheckAlertChannelSubscription(
					"checkly_check_group.test",
					"alert_channel_subscription",
					"checkly_alert_channel.email_1",
					"true",
				),
				checkGroupTestCheckAlertChannelSubscription(
					"checkly_check_group.test",
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
						address = "group-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email {
						address = "group-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email {
						address = "group-acs-3@example.com"
					}
				}

				resource "checkly_check_group" "test" {
					name        = "group-alert-channel-sub-test"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]

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
					"checkly_check_group.test",
					"alert_channel_subscription.#",
					"3",
				),
				checkGroupTestCheckAlertChannelSubscription(
					"checkly_check_group.test",
					"alert_channel_subscription",
					"checkly_alert_channel.email_1",
					"false",
				),
				checkGroupTestCheckAlertChannelSubscription(
					"checkly_check_group.test",
					"alert_channel_subscription",
					"checkly_alert_channel.email_2",
					"true",
				),
				checkGroupTestCheckAlertChannelSubscription(
					"checkly_check_group.test",
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
						address = "group-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email {
						address = "group-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email {
						address = "group-acs-3@example.com"
					}
				}

				resource "checkly_check_group" "test" {
					name        = "group-alert-channel-sub-test"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"alert_channel_subscription.#",
					"0",
				),
			),
		},
	})
}
