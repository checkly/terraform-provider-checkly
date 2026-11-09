package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
					"enforce_locations.enabled",
					"false",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_retry_strategy.enabled",
					"false",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.enabled",
					"false",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_scheduling_strategy.enabled",
					"false",
				),
			),
		},
	})
}

func TestAccCheckGroupV2EnforceLocations(t *testing.T) {
	// Private location slugs are unique per account and limited to 30
	// characters, so a short random suffix avoids collisions between runs.
	slug := fmt.Sprintf("tf-enforce-locations-%d", acctest.RandInt()%100000)
	accTestCase(t, []resource.TestStep{
		{
			Config: fmt.Sprintf(`
				resource "checkly_private_location" "test" {
					name      = %[1]q
					slug_name = %[1]q
					icon      = "bell-fill"
				}

				resource "checkly_check_group_v2" "test" {
					name = "enforce-locations-test"

					enforce_locations = {
						enabled           = true
						locations         = ["us-east-1", "eu-west-1"]
						private_locations = [checkly_private_location.test.slug_name]
					}
				}
			`, slug),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_locations.enabled",
					"true",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_locations.locations.#",
					"2",
				),
				testCheckResourceAttrExpr(
					checkGroupV2Resource,
					"enforce_locations.locations.*",
					"us-east-1",
				),
				testCheckResourceAttrExpr(
					checkGroupV2Resource,
					"enforce_locations.locations.*",
					"eu-west-1",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_locations.private_locations.#",
					"1",
				),
				testCheckResourceAttrExpr(
					checkGroupV2Resource,
					"enforce_locations.private_locations.*",
					slug,
				),
			),
		},
		{
			Config: fmt.Sprintf(`
				resource "checkly_private_location" "test" {
					name      = %[1]q
					slug_name = %[1]q
					icon      = "bell-fill"
				}

				resource "checkly_check_group_v2" "test" {
					name = "enforce-locations-test"
				}
			`, slug),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_locations.enabled",
					"false",
				),
			),
		},
	})
}

func TestAccCheckGroupV2EnforceLocationsValidation(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "enforce-locations-validation-test"

					enforce_locations = {
						enabled = true
					}
				}
			`,
			ExpectError: regexp.MustCompile(`Missing Locations`),
		},
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "enforce-locations-validation-test"

					enforce_locations = {
						enabled           = true
						locations         = []
						private_locations = []
					}
				}
			`,
			ExpectError: regexp.MustCompile(`Missing Locations`),
		},
	})
}

func TestAccCheckGroupV2EnforceRetryStrategy(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "enforce-retry-strategy-test"

					enforce_retry_strategy = {
						enabled = true

						retry_strategy = {
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
					"enforce_retry_strategy.enabled",
					"true",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_retry_strategy.retry_strategy.type",
					"FIXED",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_retry_strategy.retry_strategy.base_backoff_seconds",
					"30",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_retry_strategy.retry_strategy.max_retries",
					"3",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_retry_strategy.retry_strategy.max_duration_seconds",
					"300",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_retry_strategy.retry_strategy.same_region",
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
					"enforce_retry_strategy.enabled",
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

					enforce_retry_strategy = {
						enabled = true

						retry_strategy = {
							type = "NO_RETRIES"
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_retry_strategy.enabled",
					"true",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_retry_strategy.retry_strategy.type",
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
					"enforce_retry_strategy.enabled",
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

					enforce_alert_settings = {
						enabled = true

						alert_settings = {
							escalation_type = "RUN_BASED"

							run_based_escalation = {
								failed_run_threshold = 2
							}

							reminders = {
								amount   = 3
								interval = 10
							}

							parallel_run_failure_threshold = {
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
					"enforce_alert_settings.enabled",
					"true",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.alert_settings.escalation_type",
					"RUN_BASED",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.alert_settings.run_based_escalation.failed_run_threshold",
					"2",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.alert_settings.reminders.amount",
					"3",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.alert_settings.reminders.interval",
					"10",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.alert_settings.parallel_run_failure_threshold.enabled",
					"true",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.alert_settings.parallel_run_failure_threshold.percentage",
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
					"enforce_alert_settings.enabled",
					"false",
				),
			),
		},
	})
}

func TestAccCheckGroupV2EnforceAlertSettingsDisabledPartial(t *testing.T) {
	// A disabled enforcement with a partial alert_settings object leaves the
	// unconfigured optional+computed members (run_based_escalation, reminders,
	// parallel_run_failure_threshold) unknown in the plan; they must resolve
	// to null in state rather than fail the apply.
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "enforce-alerts-disabled-partial-test"

					enforce_alert_settings = {
						enabled = false

						alert_settings = {
							escalation_type = "RUN_BASED"
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.enabled",
					"false",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.alert_settings.escalation_type",
					"RUN_BASED",
				),
				resource.TestCheckNoResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.alert_settings.run_based_escalation.%",
				),
				resource.TestCheckNoResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.alert_settings.reminders.%",
				),
				resource.TestCheckNoResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.alert_settings.parallel_run_failure_threshold.%",
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

					enforce_scheduling_strategy = {
						enabled      = true
						run_parallel = true
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_scheduling_strategy.enabled",
					"true",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_scheduling_strategy.run_parallel",
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
					"enforce_scheduling_strategy.enabled",
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

					default_runtime = {
						runtime_id = "2023.02"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"default_runtime.runtime_id",
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
					"default_runtime",
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

					enforce_alert_settings = {
						enabled = true

						alert_settings = {
							escalation_type = "RUN_BASED"

							run_based_escalation = {
								failed_run_threshold = 1
							}

							reminders = {
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
					"enforce_alert_settings.enabled",
					"true",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.alert_settings.escalation_type",
					"RUN_BASED",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.use_global_alert_settings",
					"false",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "enforce-alert-swap-test"

					enforce_alert_settings = {
						enabled                   = true
						use_global_alert_settings = true
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.enabled",
					"true",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.use_global_alert_settings",
					"true",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "enforce-alert-swap-test"

					enforce_alert_settings = {
						enabled = true

						alert_settings = {
							escalation_type = "TIME_BASED"

							time_based_escalation = {
								minutes_failing_threshold = 10
							}

							reminders = {
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
					"enforce_alert_settings.enabled",
					"true",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.alert_settings.escalation_type",
					"TIME_BASED",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.alert_settings.time_based_escalation.minutes_failing_threshold",
					"10",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.use_global_alert_settings",
					"false",
				),
			),
		},
	})
}

func TestAccCheckGroupV2AlertChannelSubscriptions(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		// Step 1: Add two alert channel subscriptions.
		{
			Config: `
				resource "checkly_alert_channel" "email_1" {
					email = {
						address = "alert-sub-test-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email = {
						address = "alert-sub-test-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email = {
						address = "alert-sub-test-3@example.com"
					}
				}

				resource "checkly_check_group_v2" "test" {
					name = "alert-channel-sub-test"

					enforce_alert_settings = {
						enabled = true

						alert_settings = {
							escalation_type = "RUN_BASED"

							run_based_escalation = {
								failed_run_threshold = 1
							}

							reminders = {
								amount   = 0
								interval = 5
							}
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
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.alert_channel_subscriptions.#",
					"2",
				),
				testCheckAlertChannelSubscription(
					checkGroupV2Resource,
					"enforce_alert_settings.alert_channel_subscriptions",
					"checkly_alert_channel.email_1",
					"true",
				),
				testCheckAlertChannelSubscription(
					checkGroupV2Resource,
					"enforce_alert_settings.alert_channel_subscriptions",
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
						address = "alert-sub-test-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email = {
						address = "alert-sub-test-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email = {
						address = "alert-sub-test-3@example.com"
					}
				}

				resource "checkly_check_group_v2" "test" {
					name = "alert-channel-sub-test"

					enforce_alert_settings = {
						enabled = true

						alert_settings = {
							escalation_type = "RUN_BASED"

							run_based_escalation = {
								failed_run_threshold = 1
							}

							reminders = {
								amount   = 0
								interval = 5
							}
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
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.alert_channel_subscriptions.#",
					"3",
				),
				testCheckAlertChannelSubscription(
					checkGroupV2Resource,
					"enforce_alert_settings.alert_channel_subscriptions",
					"checkly_alert_channel.email_1",
					"false",
				),
				testCheckAlertChannelSubscription(
					checkGroupV2Resource,
					"enforce_alert_settings.alert_channel_subscriptions",
					"checkly_alert_channel.email_2",
					"true",
				),
				testCheckAlertChannelSubscription(
					checkGroupV2Resource,
					"enforce_alert_settings.alert_channel_subscriptions",
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
						address = "alert-sub-test-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email = {
						address = "alert-sub-test-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email = {
						address = "alert-sub-test-3@example.com"
					}
				}

				resource "checkly_check_group_v2" "test" {
					name = "alert-channel-sub-test"

					enforce_alert_settings = {
						enabled = true

						alert_settings = {
							escalation_type = "RUN_BASED"

							run_based_escalation = {
								failed_run_threshold = 1
							}

							reminders = {
								amount   = 0
								interval = 5
							}
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.alert_channel_subscriptions.#",
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

					setup_script = {
						inline_script = "console.log('setup')"
					}

					teardown_script = {
						inline_script = "console.log('teardown')"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"setup_script.inline_script",
					"console.log('setup')",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"teardown_script.inline_script",
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
					"setup_script",
				),
				resource.TestCheckNoResourceAttr(
					checkGroupV2Resource,
					"teardown_script",
				),
			),
		},
	})
}

func TestAccCheckGroupV2ScriptsValidation(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "scripts-validation-test"

					setup_script = {
						snippet_id = 0
					}
				}
			`,
			ExpectError: regexp.MustCompile(`value must be at least 1`),
		},
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "scripts-validation-test"

					teardown_script = {
						snippet_id    = 1
						inline_script = "console.log('teardown')"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`Invalid Script`),
		},
	})
}

func TestCheckGroupV2ScriptValidator(t *testing.T) {
	script := func(snippetID types.Int64, inlineScript types.String) types.Object {
		return types.ObjectValueMust(checkGroupV2ScriptAttrTypes, map[string]attr.Value{
			"snippet_id":    snippetID,
			"inline_script": inlineScript,
		})
	}

	tests := map[string]struct {
		value     types.Object
		wantError bool
	}{
		"null object":     {value: types.ObjectNull(checkGroupV2ScriptAttrTypes)},
		"empty object":    {value: script(types.Int64Null(), types.StringNull()), wantError: true},
		"both configured": {value: script(types.Int64Value(1), types.StringValue("x")), wantError: true},
		"snippet only":    {value: script(types.Int64Value(1), types.StringNull())},
		"inline only":     {value: script(types.Int64Null(), types.StringValue("x"))},
		"unknown member":  {value: script(types.Int64Unknown(), types.StringNull())},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			req := validator.ObjectRequest{
				Path:        path.Root("setup_script"),
				ConfigValue: tt.value,
			}
			resp := &validator.ObjectResponse{}

			checkGroupV2ScriptValidator{}.ValidateObject(context.Background(), req, resp)

			if got := resp.Diagnostics.HasError(); got != tt.wantError {
				t.Fatalf("HasError() = %v, want %v: %v", got, tt.wantError, resp.Diagnostics)
			}
			if tt.wantError && resp.Diagnostics.Errors()[0].Summary() != "Invalid Script" {
				t.Fatalf("unexpected error summary %q", resp.Diagnostics.Errors()[0].Summary())
			}
		})
	}
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
		{
			ResourceName:      checkGroupV2Resource,
			ImportState:       true,
			ImportStateVerify: true,
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

					environment_variables = [
						{
							key    = "BASE_URL"
							value  = "https://api.example.com"
							locked = false
						},
						{
							key    = "API_TOKEN"
							value  = "secret"
							locked = true
						},
					]
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"environment_variables.#",
					"2",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"environment_variables.0.key",
					"BASE_URL",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"environment_variables.0.value",
					"https://api.example.com",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"environment_variables.0.locked",
					"false",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"environment_variables.1.key",
					"API_TOKEN",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"environment_variables.1.value",
					"secret",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"environment_variables.1.locked",
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
					"environment_variables.#",
					"0",
				),
			),
		},
	})
}

func TestAccCheckGroupV2EnforceLocationsDisableEquivalence(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group_v2" "removed" {
					name = "enforce-locations-equiv-removed"

					enforce_locations = {
						enabled   = true
						locations = ["us-east-1"]
					}
				}

				resource "checkly_check_group_v2" "disabled" {
					name = "enforce-locations-equiv-disabled"

					enforce_locations = {
						enabled   = true
						locations = ["eu-west-1"]
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check_group_v2.removed",
					"enforce_locations.enabled",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group_v2.disabled",
					"enforce_locations.enabled",
					"true",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group_v2" "removed" {
					name = "enforce-locations-equiv-removed"
				}

				resource "checkly_check_group_v2" "disabled" {
					name = "enforce-locations-equiv-disabled"

					enforce_locations = {
						enabled   = false
						locations = ["eu-west-1"]
					}
				}

				resource "checkly_check_group_v2" "fresh" {
					name = "enforce-locations-equiv-fresh"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check_group_v2.removed",
					"enforce_locations.enabled",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group_v2.disabled",
					"enforce_locations.enabled",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group_v2.fresh",
					"enforce_locations.enabled",
					"false",
				),
			),
		},
	})
}

func TestAccCheckGroupV2EnforceRetryStrategyDisableEquivalence(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group_v2" "removed" {
					name = "enforce-retry-equiv-removed"

					enforce_retry_strategy = {
						enabled = true

						retry_strategy = {
							type                 = "FIXED"
							base_backoff_seconds = 30
							max_retries          = 3
							max_duration_seconds = 300
							same_region          = false
						}
					}
				}

				resource "checkly_check_group_v2" "disabled" {
					name = "enforce-retry-equiv-disabled"

					enforce_retry_strategy = {
						enabled = true

						retry_strategy = {
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
					"checkly_check_group_v2.removed",
					"enforce_retry_strategy.enabled",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group_v2.disabled",
					"enforce_retry_strategy.enabled",
					"true",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group_v2" "removed" {
					name = "enforce-retry-equiv-removed"
				}

				resource "checkly_check_group_v2" "disabled" {
					name = "enforce-retry-equiv-disabled"

					enforce_retry_strategy = {
						enabled = false

						retry_strategy = {
							type                 = "FIXED"
							base_backoff_seconds = 30
							max_retries          = 3
							max_duration_seconds = 300
							same_region          = false
						}
					}
				}

				resource "checkly_check_group_v2" "fresh" {
					name = "enforce-retry-equiv-fresh"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check_group_v2.removed",
					"enforce_retry_strategy.enabled",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group_v2.disabled",
					"enforce_retry_strategy.enabled",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group_v2.fresh",
					"enforce_retry_strategy.enabled",
					"false",
				),
			),
		},
	})
}

func TestAccCheckGroupV2EnforceAlertSettingsDisableEquivalence(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group_v2" "removed" {
					name = "enforce-alerts-equiv-removed"

					enforce_alert_settings = {
						enabled = true

						alert_settings = {
							escalation_type = "RUN_BASED"

							run_based_escalation = {
								failed_run_threshold = 2
							}

							reminders = {
								amount   = 3
								interval = 10
							}
						}
					}
				}

				resource "checkly_check_group_v2" "disabled" {
					name = "enforce-alerts-equiv-disabled"

					enforce_alert_settings = {
						enabled = true

						alert_settings = {
							escalation_type = "RUN_BASED"

							run_based_escalation = {
								failed_run_threshold = 2
							}

							reminders = {
								amount   = 3
								interval = 10
							}
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check_group_v2.removed",
					"enforce_alert_settings.enabled",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group_v2.disabled",
					"enforce_alert_settings.enabled",
					"true",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group_v2" "removed" {
					name = "enforce-alerts-equiv-removed"
				}

				resource "checkly_check_group_v2" "disabled" {
					name = "enforce-alerts-equiv-disabled"

					enforce_alert_settings = {
						enabled = false
					}
				}

				resource "checkly_check_group_v2" "fresh" {
					name = "enforce-alerts-equiv-fresh"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check_group_v2.removed",
					"enforce_alert_settings.enabled",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group_v2.disabled",
					"enforce_alert_settings.enabled",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group_v2.fresh",
					"enforce_alert_settings.enabled",
					"false",
				),
			),
		},
	})
}

func TestAccCheckGroupV2EnforceSchedulingStrategyDisableEquivalence(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group_v2" "removed" {
					name = "enforce-sched-equiv-removed"

					enforce_scheduling_strategy = {
						enabled      = true
						run_parallel = true
					}
				}

				resource "checkly_check_group_v2" "disabled" {
					name = "enforce-sched-equiv-disabled"

					enforce_scheduling_strategy = {
						enabled      = true
						run_parallel = true
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check_group_v2.removed",
					"enforce_scheduling_strategy.enabled",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group_v2.disabled",
					"enforce_scheduling_strategy.enabled",
					"true",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group_v2" "removed" {
					name = "enforce-sched-equiv-removed"
				}

				resource "checkly_check_group_v2" "disabled" {
					name = "enforce-sched-equiv-disabled"

					enforce_scheduling_strategy = {
						enabled      = false
						run_parallel = true
					}
				}

				resource "checkly_check_group_v2" "fresh" {
					name = "enforce-sched-equiv-fresh"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check_group_v2.removed",
					"enforce_scheduling_strategy.enabled",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group_v2.disabled",
					"enforce_scheduling_strategy.enabled",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group_v2.fresh",
					"enforce_scheduling_strategy.enabled",
					"false",
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

					api_check_defaults = {
						url = "https://api.example.com/"

						headers = {
							"X-Api-Key" = "test-key"
						}

						query_parameters = {
							"version" = "v2"
						}

						assertions = [
							{
								source     = "STATUS_CODE"
								comparison = "EQUALS"
								target     = "200"
							}
						]

						basic_auth = {
							username = "admin"
							password = "pass"
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"api_check_defaults.url",
					"https://api.example.com/",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"api_check_defaults.headers.X-Api-Key",
					"test-key",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"api_check_defaults.query_parameters.version",
					"v2",
				),
				testCheckResourceAttrExpr(
					checkGroupV2Resource,
					"api_check_defaults.assertions.*.source",
					"STATUS_CODE",
				),
				testCheckResourceAttrExpr(
					checkGroupV2Resource,
					"api_check_defaults.assertions.*.comparison",
					"EQUALS",
				),
				testCheckResourceAttrExpr(
					checkGroupV2Resource,
					"api_check_defaults.assertions.*.target",
					"200",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"api_check_defaults.basic_auth.username",
					"admin",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"api_check_defaults.basic_auth.password",
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
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"api_check_defaults.url",
					"",
				),
			),
		},
	})
}

func TestAccCheckGroupV2UnsetSweepAlertChannelSubscriptions(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		// Step 1: enforce alert settings with a real alert channel
		// subscription.
		{
			Config: `
				resource "checkly_alert_channel" "email" {
					email = {
						address = "group-v2-sweep-acs@example.com"
					}
				}

				resource "checkly_check_group_v2" "test" {
					name = "group-v2-sweep-acs"

					enforce_alert_settings = {
						enabled = true

						alert_settings = {
							escalation_type = "RUN_BASED"

							run_based_escalation = {
								failed_run_threshold = 1
							}

							reminders = {
								amount   = 0
								interval = 5
							}
						}

						alert_channel_subscriptions = [
							{
								channel_id = checkly_alert_channel.email.id
								activated  = true
							},
						]
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.enabled",
					"true",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.alert_channel_subscriptions.#",
					"1",
				),
				testCheckAlertChannelSubscription(
					checkGroupV2Resource,
					"enforce_alert_settings.alert_channel_subscriptions",
					"checkly_alert_channel.email",
					"true",
				),
			),
		},
		// Step 2: remove all subscriptions but keep the enforcement enabled.
		// The post-apply refresh verifies the server actually cleared them.
		{
			Config: `
				resource "checkly_alert_channel" "email" {
					email = {
						address = "group-v2-sweep-acs@example.com"
					}
				}

				resource "checkly_check_group_v2" "test" {
					name = "group-v2-sweep-acs"

					enforce_alert_settings = {
						enabled = true

						alert_settings = {
							escalation_type = "RUN_BASED"

							run_based_escalation = {
								failed_run_threshold = 1
							}

							reminders = {
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
					"enforce_alert_settings.enabled",
					"true",
				),
				resource.TestCheckNoResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.alert_channel_subscriptions.#",
				),
			),
		},
		// Step 3: remove the whole enforcement; it must plan back to the
		// disabled default and the server must disable it.
		{
			Config: `
				resource "checkly_alert_channel" "email" {
					email = {
						address = "group-v2-sweep-acs@example.com"
					}
				}

				resource "checkly_check_group_v2" "test" {
					name = "group-v2-sweep-acs"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.enabled",
					"false",
				),
				resource.TestCheckNoResourceAttr(
					checkGroupV2Resource,
					"enforce_alert_settings.alert_channel_subscriptions.#",
				),
			),
		},
	})
}

func TestAccCheckGroupV2UnsetSweepSnippetScripts(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		// Step 1: reference setup and teardown snippets.
		{
			Config: `
				resource "checkly_snippet" "setup" {
					name   = "group-v2-sweep-setup"
					script = "console.log('setup')"
				}

				resource "checkly_snippet" "teardown" {
					name   = "group-v2-sweep-teardown"
					script = "console.log('teardown')"
				}

				resource "checkly_check_group_v2" "test" {
					name = "group-v2-sweep-snippets"

					setup_script = {
						snippet_id = checkly_snippet.setup.id
					}

					teardown_script = {
						snippet_id = checkly_snippet.teardown.id
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrPair(
					checkGroupV2Resource, "setup_script.snippet_id",
					"checkly_snippet.setup", "id",
				),
				resource.TestCheckResourceAttrPair(
					checkGroupV2Resource, "teardown_script.snippet_id",
					"checkly_snippet.teardown", "id",
				),
			),
		},
		// Step 2: remove the script blocks; the snippet references must clear
		// server-side.
		{
			Config: `
				resource "checkly_snippet" "setup" {
					name   = "group-v2-sweep-setup"
					script = "console.log('setup')"
				}

				resource "checkly_snippet" "teardown" {
					name   = "group-v2-sweep-teardown"
					script = "console.log('teardown')"
				}

				resource "checkly_check_group_v2" "test" {
					name = "group-v2-sweep-snippets"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(
					checkGroupV2Resource,
					"setup_script",
				),
				resource.TestCheckNoResourceAttr(
					checkGroupV2Resource,
					"teardown_script",
				),
			),
		},
	})
}

func TestAccCheckGroupV2UnsetSweepEnforceLocationsChange(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		// Step 1: enforce two public locations.
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "group-v2-sweep-locations"

					enforce_locations = {
						enabled   = true
						locations = ["us-east-1", "eu-west-1"]
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_locations.enabled",
					"true",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_locations.locations.#",
					"2",
				),
			),
		},
		// Step 2: change the enforced locations to a different single one.
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "group-v2-sweep-locations"

					enforce_locations = {
						enabled   = true
						locations = ["eu-central-1"]
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_locations.locations.#",
					"1",
				),
				testCheckResourceAttrExpr(
					checkGroupV2Resource,
					"enforce_locations.locations.*",
					"eu-central-1",
				),
			),
		},
		// Step 3: remove the enforcement; it must plan back to the disabled
		// default and the server must disable it.
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "group-v2-sweep-locations"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_locations.enabled",
					"false",
				),
			),
		},
	})
}

func TestAccCheckGroupV2UnsetSweepAPICheckDefaults(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		// Step 1: set every api_check_defaults sub-property.
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "group-v2-sweep-api-defaults"

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
				resource.TestCheckResourceAttr(checkGroupV2Resource, "api_check_defaults.url", "http://sweep1.example.com/"),
				resource.TestCheckResourceAttr(checkGroupV2Resource, "api_check_defaults.headers.X-Sweep", "one"),
				resource.TestCheckResourceAttr(checkGroupV2Resource, "api_check_defaults.query_parameters.sweep", "one"),
				resource.TestCheckResourceAttr(checkGroupV2Resource, "api_check_defaults.assertions.#", "1"),
				resource.TestCheckResourceAttr(checkGroupV2Resource, "api_check_defaults.basic_auth.username", "user"),
			),
		},
		// Step 2: keep the object but remove every sub-property and change the
		// url; the sub-properties must clear server-side.
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "group-v2-sweep-api-defaults"

					api_check_defaults = {
						url = "http://sweep2.example.com/"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(checkGroupV2Resource, "api_check_defaults.url", "http://sweep2.example.com/"),
				resource.TestCheckResourceAttr(checkGroupV2Resource, "api_check_defaults.headers.%", "0"),
				resource.TestCheckResourceAttr(checkGroupV2Resource, "api_check_defaults.query_parameters.%", "0"),
				resource.TestCheckResourceAttr(checkGroupV2Resource, "api_check_defaults.assertions.#", "0"),
				resource.TestCheckNoResourceAttr(checkGroupV2Resource, "api_check_defaults.basic_auth.%"),
			),
		},
	})
}
