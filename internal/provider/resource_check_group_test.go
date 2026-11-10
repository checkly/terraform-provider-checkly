package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

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
					"api_check_defaults.assertions.#",
					"1",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"api_check_defaults.assertions.*.comparison",
					"EQUALS",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"api_check_defaults.assertions.*.property",
					"",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"api_check_defaults.assertions.*.source",
					"STATUS_CODE",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"api_check_defaults.assertions.*.target",
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
		{
			ResourceName:      "checkly_check_group.test",
			ImportState:       true,
			ImportStateVerify: true,
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
	  assertions = [
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
					"checkly_check_group.test",
					"environment_variables.#",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"environment_variables.0.key",
					"FOO",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"environment_variables.0.value",
					"bar",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"environment_variables.0.locked",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"environment_variables.1.key",
					"SECRET",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"environment_variables.1.value",
					"s3cr3t",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"environment_variables.1.locked",
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
					"checkly_check_group.test",
					"environment_variables.#",
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
					"environment_variables.#",
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

func TestAccCheckGroupUnsetSweepScalars(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		// Step 1: set every scalar/collection property to a non-default value.
		{
			Config: `
				resource "checkly_check_group" "test" {
					name         = "group-sweep-scalars"
					activated    = true
					concurrency  = 2
					muted        = true
					run_parallel = true
					locations    = ["eu-central-1", "eu-west-1"]
					tags         = ["sweep", "group"]
					runtime_id   = "2023.02"

					local_setup_script    = "console.log('setup')"
					local_teardown_script = "console.log('teardown')"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_check_group.test", "concurrency", "2"),
				resource.TestCheckResourceAttr("checkly_check_group.test", "muted", "true"),
				resource.TestCheckResourceAttr("checkly_check_group.test", "run_parallel", "true"),
				resource.TestCheckResourceAttr("checkly_check_group.test", "locations.#", "2"),
				resource.TestCheckResourceAttr("checkly_check_group.test", "tags.#", "2"),
				resource.TestCheckResourceAttr("checkly_check_group.test", "runtime_id", "2023.02"),
				resource.TestCheckResourceAttr("checkly_check_group.test", "local_setup_script", "console.log('setup')"),
				resource.TestCheckResourceAttr("checkly_check_group.test", "local_teardown_script", "console.log('teardown')"),
			),
		},
		// Step 2: remove all optional properties and change concurrency. The
		// post-apply refresh verifies the server actually cleared them.
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "group-sweep-scalars"
					activated   = true
					concurrency = 5
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_check_group.test", "concurrency", "5"),
				resource.TestCheckResourceAttr("checkly_check_group.test", "muted", "false"),
				resource.TestCheckResourceAttr("checkly_check_group.test", "run_parallel", "false"),
				resource.TestCheckResourceAttr("checkly_check_group.test", "locations.#", "0"),
				resource.TestCheckResourceAttr("checkly_check_group.test", "tags.#", "0"),
				resource.TestCheckNoResourceAttr("checkly_check_group.test", "runtime_id"),
				resource.TestCheckResourceAttr("checkly_check_group.test", "local_setup_script", ""),
				resource.TestCheckResourceAttr("checkly_check_group.test", "local_teardown_script", ""),
			),
		},
	})
}

func TestAccCheckGroupUnsetSweepSnippets(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		// Step 1: reference setup and teardown snippets.
		{
			Config: `
				resource "checkly_snippet" "setup" {
					name   = "group-sweep-setup"
					script = "console.log('setup')"
				}

				resource "checkly_snippet" "teardown" {
					name   = "group-sweep-teardown"
					script = "console.log('teardown')"
				}

				resource "checkly_check_group" "test" {
					name        = "group-sweep-snippets"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]

					setup_snippet_id    = checkly_snippet.setup.id
					teardown_snippet_id = checkly_snippet.teardown.id
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrPair(
					"checkly_check_group.test", "setup_snippet_id",
					"checkly_snippet.setup", "id",
				),
				resource.TestCheckResourceAttrPair(
					"checkly_check_group.test", "teardown_snippet_id",
					"checkly_snippet.teardown", "id",
				),
			),
		},
		// Step 2: remove the snippet references; they must reset to 0.
		{
			Config: `
				resource "checkly_snippet" "setup" {
					name   = "group-sweep-setup"
					script = "console.log('setup')"
				}

				resource "checkly_snippet" "teardown" {
					name   = "group-sweep-teardown"
					script = "console.log('teardown')"
				}

				resource "checkly_check_group" "test" {
					name        = "group-sweep-snippets"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_check_group.test", "setup_snippet_id", "0"),
				resource.TestCheckResourceAttr("checkly_check_group.test", "teardown_snippet_id", "0"),
			),
		},
	})
}

func TestAccCheckGroupUnsetSweepPrivateLocations(t *testing.T) {
	// Private location slugs are unique per account and limited to 30
	// characters, so a short random suffix avoids collisions between runs.
	slug := fmt.Sprintf("tf-group-v1-sweep-%d", acctest.RandInt()%100000)
	accTestCase(t, []resource.TestStep{
		// Step 1: assign a private location.
		{
			Config: fmt.Sprintf(`
				resource "checkly_private_location" "test" {
					name      = %[1]q
					slug_name = %[1]q
					icon      = "bell-fill"
				}

				resource "checkly_check_group" "test" {
					name        = "group-sweep-private-locations"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]

					private_locations = [checkly_private_location.test.slug_name]
				}
			`, slug),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_check_group.test", "private_locations.#", "1"),
				testCheckResourceAttrExpr("checkly_check_group.test", "private_locations.*", slug),
			),
		},
		// Step 2: remove the private location assignment.
		{
			Config: fmt.Sprintf(`
				resource "checkly_private_location" "test" {
					name      = %[1]q
					slug_name = %[1]q
					icon      = "bell-fill"
				}

				resource "checkly_check_group" "test" {
					name        = "group-sweep-private-locations"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}
			`, slug),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_check_group.test", "private_locations.#", "0"),
			),
		},
	})
}

func TestAccCheckGroupUnsetSweepAPICheckDefaults(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		// Step 1: set every api_check_defaults sub-property.
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "group-sweep-api-defaults"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]

					api_check_defaults = {
						url = "http://sweep1.example.com/"

						headers = {
							X-Sweep = "one"
						}

						query_parameters = {
							sweep = "one"
						}

						assertions = [
							{
								source     = "STATUS_CODE"
								comparison = "EQUALS"
								target     = "200"
							}
						]

						basic_auth = {
							username = "user"
							password = "pass"
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_check_group.test", "api_check_defaults.url", "http://sweep1.example.com/"),
				resource.TestCheckResourceAttr("checkly_check_group.test", "api_check_defaults.headers.X-Sweep", "one"),
				resource.TestCheckResourceAttr("checkly_check_group.test", "api_check_defaults.query_parameters.sweep", "one"),
				resource.TestCheckResourceAttr("checkly_check_group.test", "api_check_defaults.assertions.#", "1"),
				resource.TestCheckResourceAttr("checkly_check_group.test", "api_check_defaults.basic_auth.username", "user"),
			),
		},
		// Step 2: keep the object but remove every sub-property and change the
		// url; the sub-properties must clear server-side.
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "group-sweep-api-defaults"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]

					api_check_defaults = {
						url = "http://sweep2.example.com/"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_check_group.test", "api_check_defaults.url", "http://sweep2.example.com/"),
				resource.TestCheckResourceAttr("checkly_check_group.test", "api_check_defaults.headers.%", "0"),
				resource.TestCheckResourceAttr("checkly_check_group.test", "api_check_defaults.query_parameters.%", "0"),
				resource.TestCheckResourceAttr("checkly_check_group.test", "api_check_defaults.assertions.#", "0"),
				resource.TestCheckNoResourceAttr("checkly_check_group.test", "api_check_defaults.basic_auth.%"),
			),
		},
		// Step 3: remove the whole object; it must reset to the empty default.
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "group-sweep-api-defaults"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_check_group.test", "api_check_defaults.url", ""),
				resource.TestCheckResourceAttr("checkly_check_group.test", "api_check_defaults.headers.%", "0"),
				resource.TestCheckResourceAttr("checkly_check_group.test", "api_check_defaults.query_parameters.%", "0"),
				resource.TestCheckResourceAttr("checkly_check_group.test", "api_check_defaults.assertions.#", "0"),
				resource.TestCheckNoResourceAttr("checkly_check_group.test", "api_check_defaults.basic_auth.%"),
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
					email = {
						address = "group-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email = {
						address = "group-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email = {
						address = "group-acs-3@example.com"
					}
				}

				resource "checkly_check_group" "test" {
					name        = "group-alert-channel-sub-test"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]

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
					"checkly_check_group.test",
					"alert_channel_subscriptions.#",
					"2",
				),
				testCheckAlertChannelSubscription(
					"checkly_check_group.test",
					"alert_channel_subscriptions",
					"checkly_alert_channel.email_1",
					"true",
				),
				testCheckAlertChannelSubscription(
					"checkly_check_group.test",
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
						address = "group-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email = {
						address = "group-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email = {
						address = "group-acs-3@example.com"
					}
				}

				resource "checkly_check_group" "test" {
					name        = "group-alert-channel-sub-test"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]

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
					"checkly_check_group.test",
					"alert_channel_subscriptions.#",
					"3",
				),
				testCheckAlertChannelSubscription(
					"checkly_check_group.test",
					"alert_channel_subscriptions",
					"checkly_alert_channel.email_1",
					"false",
				),
				testCheckAlertChannelSubscription(
					"checkly_check_group.test",
					"alert_channel_subscriptions",
					"checkly_alert_channel.email_2",
					"true",
				),
				testCheckAlertChannelSubscription(
					"checkly_check_group.test",
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
						address = "group-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email = {
						address = "group-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email = {
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
					"alert_channel_subscriptions.#",
					"0",
				),
			),
		},
	})
}

// TestAccCheckGroupSwitchToV2 follows the documented way of switching an
// existing group from the deprecated checkly_check_group to
// checkly_check_group_v2: import the group as checkly_check_group_v2, then
// remove the checkly_check_group resource from state without destroying it.
// Both resource types manage the same API object, so the imported group must
// match its equivalent configuration without changes. That configuration
// enforces the settings checkly_check_group always applies to its checks.
func TestAccCheckGroupSwitchToV2(t *testing.T) {
	name := fmt.Sprintf("tf-acc-group-switch-%d", acctest.RandInt())
	v1 := fmt.Sprintf(`
		resource "checkly_check_group" "test" {
			name        = %q
			activated   = true
			concurrency = 1
			locations   = ["eu-central-1"]
		}
	`, name)
	v2 := fmt.Sprintf(`
		resource "checkly_check_group_v2" "test" {
			name = %q

			enforce_locations = {
				enabled   = true
				locations = ["eu-central-1"]
			}

			enforce_alert_settings = {
				enabled                   = true
				use_global_alert_settings = true
			}

			enforce_retry_strategy = {
				enabled = true
				retry_strategy = {
					type = "NO_RETRIES"
				}
			}

			enforce_scheduling_strategy = {
				enabled      = true
				run_parallel = false
			}
		}
	`, name)
	accTestCase(t, []resource.TestStep{
		{
			Config: v1,
		},
		{
			Config: v1 + v2 + `
				import {
					to = checkly_check_group_v2.test
					id = checkly_check_group.test.id
				}
			`,
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction("checkly_check_group_v2.test", plancheck.ResourceActionNoop),
				},
			},
		},
		{
			Config: v2 + `
				removed {
					from = checkly_check_group.test

					lifecycle {
						destroy = false
					}
				}
			`,
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction("checkly_check_group_v2.test", plancheck.ResourceActionNoop),
				},
			},
		},
	})
}

// TestAccCheckGroupDoubleCheck covers the deprecated double_check attribute,
// which implies a single retry in another region when retry_strategy is not
// configured, and switches back to no retries when removed. The SDK does not
// read the flag, so importing a group always reads it as false.
func TestAccCheckGroupDoubleCheck(t *testing.T) {
	config := func(doubleCheck string) string {
		return fmt.Sprintf(`
			resource "checkly_check_group" "test" {
				name        = "tf-acc-group-double-check"
				activated   = true
				concurrency = 1
				locations   = ["eu-central-1"]
				%s
			}
		`, doubleCheck)
	}
	const r = "checkly_check_group.test"
	accTestCase(t, []resource.TestStep{
		{
			Config: config("double_check = true"),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(r, "double_check", "true"),
				resource.TestCheckResourceAttr(r, "retry_strategy.type", "FIXED"),
				resource.TestCheckResourceAttr(r, "retry_strategy.max_retries", "1"),
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
		{
			Config: config("double_check = true"),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(r, "double_check", "true"),
				resource.TestCheckResourceAttr(r, "retry_strategy.type", "FIXED"),
				resource.TestCheckResourceAttr(r, "retry_strategy.max_retries", "1"),
			),
		},
	})
}

// TestAccCheckGroupWithoutAlertSettings manages a group that has no alert
// settings, which is how checkly_check_group_v2 creates one without
// enforce_alert_settings: an unrelated change keeps them unset, and configuring
// alert_settings afterwards results in the API's values and an empty plan.
func TestAccCheckGroupWithoutAlertSettings(t *testing.T) {
	cases := map[string]struct {
		alertSettings string
		check         resource.TestCheckFunc
	}{
		"run based": {
			alertSettings: `alert_settings = { escalation_type = "RUN_BASED" }`,
			check:         resource.TestCheckResourceAttr("checkly_check_group.test", "alert_settings.run_based_escalation.failed_run_threshold", "1"),
		},
		"time based": {
			alertSettings: `alert_settings = { escalation_type = "TIME_BASED" }`,
			check:         resource.TestCheckResourceAttr("checkly_check_group.test", "alert_settings.time_based_escalation.minutes_failing_threshold", "5"),
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			groupName := fmt.Sprintf("tf-acc-group-no-alerts-%d", acctest.RandInt())
			v2 := fmt.Sprintf(`
				resource "checkly_check_group_v2" "test" {
					name = %q

					enforce_locations = {
						enabled   = true
						locations = ["eu-central-1"]
					}

					enforce_retry_strategy = {
						enabled = true
						retry_strategy = {
							type = "NO_RETRIES"
						}
					}

					enforce_scheduling_strategy = {
						enabled      = true
						run_parallel = false
					}
				}
			`, groupName)
			v1 := func(concurrency int, alertSettings string) string {
				return fmt.Sprintf(`
					resource "checkly_check_group" "test" {
						name        = %q
						activated   = true
						concurrency = %d
						locations   = ["eu-central-1"]
						%s
					}
				`, groupName, concurrency, alertSettings)
			}
			// Without alert_settings, checkly_check_group plans
			// use_global_alert_settings = true, which would switch the group
			// to the account level alert settings.
			const keepOwnAlertSettings = "use_global_alert_settings = false"
			const removeV2 = `
				removed {
					from = checkly_check_group_v2.test

					lifecycle {
						destroy = false
					}
				}
			`

			accTestCase(t, []resource.TestStep{
				{
					Config: v2,
				},
				{
					Config: v2 + v1(1, keepOwnAlertSettings) + `
						import {
							to = checkly_check_group.test
							id = checkly_check_group_v2.test.id
						}
					`,
					Check: resource.TestCheckResourceAttr("checkly_check_group.test", "alert_settings.escalation_type", ""),
				},
				{
					Config: removeV2 + v1(2, keepOwnAlertSettings),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("checkly_check_group.test", "concurrency", "2"),
						resource.TestCheckResourceAttr("checkly_check_group.test", "alert_settings.escalation_type", ""),
					),
				},
				{
					Config: v1(2, c.alertSettings),
					Check: resource.ComposeTestCheckFunc(
						c.check,
						resource.TestCheckResourceAttr("checkly_check_group.test", "alert_settings.reminders.amount", "0"),
						resource.TestCheckResourceAttr("checkly_check_group.test", "alert_settings.reminders.interval", "5"),
						resource.TestCheckResourceAttr("checkly_check_group.test", "alert_settings.parallel_run_failure_threshold.percentage", "10"),
					),
				},
			})
		})
	}
}
