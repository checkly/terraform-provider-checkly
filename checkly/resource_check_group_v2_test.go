package checkly

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const checkGroupV2Resource = "checkly_check_group_v2.test"

func TestAccCheckGroupV2Folder(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "folder-test"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"name",
					"folder-test",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"activated",
					"true",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"muted",
					"false",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"concurrency",
					"1",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_locations.0.enabled",
					"false",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_retry_strategy.0.enabled",
					"false",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.0.enabled",
					"false",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_scheduling_strategy.0.enabled",
					"false",
				),
			),
		},
	})
}

func TestAccCheckGroupV2EnforceLocations(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_private_location" "test" {
					name      = "enforce-locations-pl"
					slug_name = "enforce-locations-pl"
					icon      = "bell-fill"
				}

				resource "checkly_check_group_v2" "test" {
					name = "enforce-locations-test"

					enforce_locations {
						enabled           = true
						locations         = ["us-east-1", "eu-west-1"]
						private_locations = [checkly_private_location.test.slug_name]
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_locations.0.enabled",
					"true",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_locations.0.locations.#",
					"2",
				),
				testCheckResourceAttrExpr(
					checkGroupV2Resource,
					"enforce_locations.0.locations.*",
					"us-east-1",
				),
				testCheckResourceAttrExpr(
					checkGroupV2Resource,
					"enforce_locations.0.locations.*",
					"eu-west-1",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_locations.0.private_locations.#",
					"1",
				),
				testCheckResourceAttrExpr(
					checkGroupV2Resource,
					"enforce_locations.0.private_locations.*",
					"enforce-locations-pl",
				),
			),
		},
		{
			Config: `
				resource "checkly_private_location" "test" {
					name      = "enforce-locations-pl"
					slug_name = "enforce-locations-pl"
					icon      = "bell-fill"
				}

				resource "checkly_check_group_v2" "test" {
					name = "enforce-locations-test"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_locations.0.enabled",
					"false",
				),
			),
		},
	})
}

func TestAccCheckGroupV2EnforceRetryStrategy(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "enforce-retry-strategy-test"

					enforce_retry_strategy {
						enabled = true

						retry_strategy {
							type                 = "FIXED"
							base_backoff_seconds = 30
							max_retries          = 3
							max_duration_seconds = 300
							same_region          = false
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_retry_strategy.0.enabled",
					"true",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_retry_strategy.0.retry_strategy.0.type",
					"FIXED",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_retry_strategy.0.retry_strategy.0.base_backoff_seconds",
					"30",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_retry_strategy.0.retry_strategy.0.max_retries",
					"3",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_retry_strategy.0.retry_strategy.0.max_duration_seconds",
					"300",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_retry_strategy.0.retry_strategy.0.same_region",
					"false",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "enforce-retry-strategy-test"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_retry_strategy.0.enabled",
					"false",
				),
			),
		},
	})
}

func TestAccCheckGroupV2EnforceRetryStrategyNoRetries(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "enforce-retry-strategy-no-retries-test"

					enforce_retry_strategy {
						enabled = true

						retry_strategy {
							type = "NO_RETRIES"
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_retry_strategy.0.enabled",
					"true",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_retry_strategy.0.retry_strategy.0.type",
					"NO_RETRIES",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "enforce-retry-strategy-no-retries-test"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_retry_strategy.0.enabled",
					"false",
				),
			),
		},
	})
}

func TestAccCheckGroupV2EnforceAlertSettings(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "enforce-alert-settings-test"

					enforce_alert_settings {
						enabled = true

						alert_settings {
							escalation_type = "RUN_BASED"

							run_based_escalation {
								failed_run_threshold = 2
							}

							reminders {
								amount   = 3
								interval = 10
							}

							parallel_run_failure_threshold {
								enabled    = true
								percentage = 50
							}
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.0.enabled",
					"true",
				),
				testCheckResourceAttrExpr(
					checkGroupV2Resource,
					"enforce_alert_settings.0.alert_settings.*.escalation_type",
					"RUN_BASED",
				),
				testCheckResourceAttrExpr(
					checkGroupV2Resource,
					"enforce_alert_settings.0.alert_settings.*.run_based_escalation.*.failed_run_threshold",
					"2",
				),
				testCheckResourceAttrExpr(
					checkGroupV2Resource,
					"enforce_alert_settings.0.alert_settings.*.reminders.*.amount",
					"3",
				),
				testCheckResourceAttrExpr(
					checkGroupV2Resource,
					"enforce_alert_settings.0.alert_settings.*.reminders.*.interval",
					"10",
				),
				testCheckResourceAttrExpr(
					checkGroupV2Resource,
					"enforce_alert_settings.0.alert_settings.*.parallel_run_failure_threshold.*.enabled",
					"true",
				),
				testCheckResourceAttrExpr(
					checkGroupV2Resource,
					"enforce_alert_settings.0.alert_settings.*.parallel_run_failure_threshold.*.percentage",
					"50",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "enforce-alert-settings-test"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.0.enabled",
					"false",
				),
			),
		},
	})
}

func TestAccCheckGroupV2EnforceSchedulingStrategy(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "enforce-scheduling-strategy-test"

					enforce_scheduling_strategy {
						enabled      = true
						run_parallel = true
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_scheduling_strategy.0.enabled",
					"true",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_scheduling_strategy.0.run_parallel",
					"true",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "enforce-scheduling-strategy-test"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_scheduling_strategy.0.enabled",
					"false",
				),
			),
		},
	})
}

func TestAccCheckGroupV2DefaultRuntime(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "default-runtime-test"

					default_runtime {
						runtime_id = "2023.02"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"default_runtime.0.runtime_id",
					"2023.02",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "default-runtime-test"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(
					checkGroupV2Resource,
					"default_runtime.#",
				),
			),
		},
	})
}

func TestAccCheckGroupV2EnforceAlertSettingsSwap(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "enforce-alert-swap-test"

					enforce_alert_settings {
						enabled = true

						alert_settings {
							escalation_type = "RUN_BASED"

							run_based_escalation {
								failed_run_threshold = 1
							}

							reminders {
								amount   = 0
								interval = 5
							}
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.0.enabled",
					"true",
				),
				testCheckResourceAttrExpr(
					checkGroupV2Resource,
					"enforce_alert_settings.0.alert_settings.*.escalation_type",
					"RUN_BASED",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.0.use_global_alert_settings",
					"false",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "enforce-alert-swap-test"

					enforce_alert_settings {
						enabled                   = true
						use_global_alert_settings = true
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.0.enabled",
					"true",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.0.use_global_alert_settings",
					"true",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "enforce-alert-swap-test"

					enforce_alert_settings {
						enabled = true

						alert_settings {
							escalation_type = "TIME_BASED"

							time_based_escalation {
								minutes_failing_threshold = 10
							}

							reminders {
								amount   = 1
								interval = 5
							}
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.0.enabled",
					"true",
				),
				testCheckResourceAttrExpr(
					checkGroupV2Resource,
					"enforce_alert_settings.0.alert_settings.*.escalation_type",
					"TIME_BASED",
				),
				testCheckResourceAttrExpr(
					checkGroupV2Resource,
					"enforce_alert_settings.0.alert_settings.*.time_based_escalation.*.minutes_failing_threshold",
					"10",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.0.use_global_alert_settings",
					"false",
				),
			),
		},
	})
}

func TestAccCheckGroupV2Scripts(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "scripts-test"

					setup_script {
						inline_script = "console.log('setup')"
					}

					teardown_script {
						inline_script = "console.log('teardown')"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"setup_script.0.inline_script",
					"console.log('setup')",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"teardown_script.0.inline_script",
					"console.log('teardown')",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "scripts-test"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(
					checkGroupV2Resource,
					"setup_script.#",
				),
				resource.TestCheckNoResourceAttr(
					checkGroupV2Resource,
					"teardown_script.#",
				),
			),
		},
	})
}

func TestAccCheckGroupV2TopLevelAttributes(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name        = "top-level-attrs-test"
					activated   = false
					muted       = true
					concurrency = 5
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"activated",
					"false",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"muted",
					"true",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"concurrency",
					"5",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "top-level-attrs-test"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"activated",
					"true",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"muted",
					"false",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"concurrency",
					"1",
				),
			),
		},
	})
}

func TestAccCheckGroupV2Tags(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "tags-test"
					tags = ["production", "api"]
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"tags.#",
					"2",
				),
				testCheckResourceAttrExpr(
					checkGroupV2Resource,
					"tags.*",
					"production",
				),
				testCheckResourceAttrExpr(
					checkGroupV2Resource,
					"tags.*",
					"api",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "tags-test"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"tags.#",
					"0",
				),
			),
		},
	})
}

func TestAccCheckGroupV2EnvironmentVariable(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "env-var-test"

					environment_variable {
						key    = "BASE_URL"
						value  = "https://api.example.com"
						locked = false
					}

					environment_variable {
						key    = "API_TOKEN"
						value  = "secret"
						locked = true
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"environment_variable.#",
					"2",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"environment_variable.0.key",
					"BASE_URL",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"environment_variable.0.value",
					"https://api.example.com",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"environment_variable.0.locked",
					"false",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"environment_variable.1.key",
					"API_TOKEN",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"environment_variable.1.value",
					"secret",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"environment_variable.1.locked",
					"true",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "env-var-test"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"environment_variable.#",
					"0",
				),
			),
		},
	})
}

func TestAccCheckGroupV2ApiCheckDefaults(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "api-defaults-test"

					api_check_defaults {
						url = "https://api.example.com/"

						headers = {
							"X-Api-Key" = "test-key"
						}

						query_parameters = {
							"version" = "v2"
						}

						assertion {
							source     = "STATUS_CODE"
							comparison = "EQUALS"
							target     = "200"
						}

						basic_auth {
							username = "admin"
							password = "pass"
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				testCheckResourceAttrExpr(
					checkGroupV2Resource,
					"api_check_defaults.*.url",
					"https://api.example.com/",
				),
				testCheckResourceAttrExpr(
					checkGroupV2Resource,
					"api_check_defaults.*.headers.X-Api-Key",
					"test-key",
				),
				testCheckResourceAttrExpr(
					checkGroupV2Resource,
					"api_check_defaults.*.query_parameters.version",
					"v2",
				),
				testCheckResourceAttrExpr(
					checkGroupV2Resource,
					"api_check_defaults.*.assertion.*.source",
					"STATUS_CODE",
				),
				testCheckResourceAttrExpr(
					checkGroupV2Resource,
					"api_check_defaults.*.assertion.*.comparison",
					"EQUALS",
				),
				testCheckResourceAttrExpr(
					checkGroupV2Resource,
					"api_check_defaults.*.assertion.*.target",
					"200",
				),
				testCheckResourceAttrExpr(
					checkGroupV2Resource,
					"api_check_defaults.*.basic_auth.*.username",
					"admin",
				),
				testCheckResourceAttrExpr(
					checkGroupV2Resource,
					"api_check_defaults.*.basic_auth.*.password",
					"pass",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "api-defaults-test"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				testCheckResourceAttrExpr(
					checkGroupV2Resource,
					"api_check_defaults.*.url",
					"",
				),
			),
		},
	})
}
