package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
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
				resource.TestCheckTypeSetElemAttr(
					checkGroupV2Resource,
					"enforce_locations.locations.*",
					"us-east-1",
				),
				resource.TestCheckTypeSetElemAttr(
					checkGroupV2Resource,
					"enforce_locations.locations.*",
					"eu-west-1",
				),
				resource.TestCheckResourceAttr(
					checkGroupV2Resource,
					"enforce_locations.private_locations.#",
					"1",
				),
				resource.TestCheckTypeSetElemAttr(
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

// checkGroupV2UseGlobalAlertSettingsStep returns a step that configures a
// group with enforcement as its enforce_alert_settings, or without the
// attribute if enforcement is empty, and expects the plan to take action
// with enforce_alert_settings.use_global_alert_settings planned and stored
// as want. The implicit empty-plan check after the step verifies that the
// API and Read keep the planned value.
func checkGroupV2UseGlobalAlertSettingsStep(name, enforcement string, action plancheck.ResourceActionType, want knownvalue.Check) resource.TestStep {
	value := tfjsonpath.New("enforce_alert_settings").AtMapKey(useGlobalAlertSettingsAttributeName)
	if enforcement != "" {
		enforcement = "enforce_alert_settings = {\n" + enforcement + "\n}"
	}
	return resource.TestStep{
		Config: fmt.Sprintf(`
			resource "checkly_check_group_v2" "test" {
				name = %q
				%s
			}
		`, name, enforcement),
		ConfigPlanChecks: resource.ConfigPlanChecks{
			PreApply: []plancheck.PlanCheck{
				plancheck.ExpectResourceAction(checkGroupV2Resource, action),
				plancheck.ExpectKnownValue(checkGroupV2Resource, value, want),
			},
		},
		ConfigStateChecks: []statecheck.StateCheck{
			statecheck.ExpectKnownValue(checkGroupV2Resource, value, want),
		},
	}
}

const checkGroupV2UseGlobalAlertSettingsAlertSettings = `
	alert_settings = {
		escalation_type = "RUN_BASED"
		run_based_escalation = {
			failed_run_threshold = 2
		}
	}
`

// TestAccCheckGroupV2EnforceAlertSettingsUseGlobalDerived walks an enabled
// alert settings enforcement through every way use_global_alert_settings is
// derived: from no alert_settings, to configured alert_settings, back to
// none, and finally to an explicit true alongside alert_settings, which wins
// over the derived value. Removing alert_settings keeps the stored alert
// settings, so the last step configures the stored ones again and, with
// use_global_alert_settings still true, plans no change at all.
func TestAccCheckGroupV2EnforceAlertSettingsUseGlobalDerived(t *testing.T) {
	name := fmt.Sprintf("group-v2-use-global-derived-%d", acctest.RandInt())
	step := func(enforcement string, action plancheck.ResourceActionType, want bool) resource.TestStep {
		return checkGroupV2UseGlobalAlertSettingsStep(name, "enabled = true\n"+enforcement, action, knownvalue.Bool(want))
	}

	accTestCase(t, []resource.TestStep{
		step("", plancheck.ResourceActionCreate, true),
		step(checkGroupV2UseGlobalAlertSettingsAlertSettings, plancheck.ResourceActionUpdate, false),
		step("", plancheck.ResourceActionUpdate, true),
		step(checkGroupV2UseGlobalAlertSettingsAlertSettings+"use_global_alert_settings = true", plancheck.ResourceActionNoop, true),
	})
}

// TestAccCheckGroupV2EnforceAlertSettingsUseGlobalDisabled covers
// use_global_alert_settings of an omitted or disabled alert settings
// enforcement, which the API ignores. It stays null for a group that never
// enforced alert settings, becomes null when an enforcement is disabled or
// removed, keeps a stored value while the enforcement stays disabled, and is
// derived again once the enforcement is enabled.
func TestAccCheckGroupV2EnforceAlertSettingsUseGlobalDisabled(t *testing.T) {
	name := fmt.Sprintf("group-v2-use-global-disabled-%d", acctest.RandInt())
	step := func(enforcement string, action plancheck.ResourceActionType, want knownvalue.Check) resource.TestStep {
		return checkGroupV2UseGlobalAlertSettingsStep(name, enforcement, action, want)
	}

	accTestCase(t, []resource.TestStep{
		// Omitted, then disabled: nothing to derive.
		step("", plancheck.ResourceActionCreate, knownvalue.Null()),
		step("enabled = false", plancheck.ResourceActionNoop, knownvalue.Null()),
		// Disabled to enabled derives the value.
		step("enabled = true", plancheck.ResourceActionUpdate, knownvalue.Bool(true)),
		// Enabled to disabled drops it.
		step("enabled = false", plancheck.ResourceActionUpdate, knownvalue.Null()),
		// A disabled enforcement keeps a stored value, also when the
		// enforcement is then removed.
		step("enabled = true\n"+checkGroupV2UseGlobalAlertSettingsAlertSettings, plancheck.ResourceActionUpdate, knownvalue.Bool(false)),
		step("enabled = false\nuse_global_alert_settings = false", plancheck.ResourceActionUpdate, knownvalue.Bool(false)),
		step("enabled = false", plancheck.ResourceActionNoop, knownvalue.Bool(false)),
		step("", plancheck.ResourceActionNoop, knownvalue.Bool(false)),
		// Enabled to removed drops it.
		step("enabled = true", plancheck.ResourceActionUpdate, knownvalue.Bool(true)),
		step("", plancheck.ResourceActionUpdate, knownvalue.Null()),
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
				resource.TestCheckTypeSetElemAttr(
					checkGroupV2Resource,
					"tags.*",
					"production",
				),
				resource.TestCheckTypeSetElemAttr(
					checkGroupV2Resource,
					"tags.*",
					"api",
				),
			),
		},
		// An explicitly empty tags is stored as an empty set.
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "tags-test"
					tags = []
				}
			`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue(checkGroupV2Resource, tfjsonpath.New("tags"), knownvalue.SetExact([]knownvalue.Check{})),
			},
		},
		// Leaving tags out stores null.
		{
			Config: `
				resource "checkly_check_group_v2" "test" {
					name = "tags-test"
				}
			`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue(checkGroupV2Resource, tfjsonpath.New("tags"), knownvalue.Null()),
			},
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
				resource.TestCheckTypeSetElemNestedAttrs(
					checkGroupV2Resource,
					"api_check_defaults.assertions.*",
					map[string]string{
						"source":     "STATUS_CODE",
						"comparison": "EQUALS",
						"target":     "200",
					},
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
				resource.TestCheckTypeSetElemAttr(
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
				resource.TestCheckNoResourceAttr(checkGroupV2Resource, "api_check_defaults.basic_auth.%"),
			),
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue(checkGroupV2Resource, tfjsonpath.New("api_check_defaults").AtMapKey("headers"), knownvalue.Null()),
				statecheck.ExpectKnownValue(checkGroupV2Resource, tfjsonpath.New("api_check_defaults").AtMapKey("query_parameters"), knownvalue.Null()),
				statecheck.ExpectKnownValue(checkGroupV2Resource, tfjsonpath.New("api_check_defaults").AtMapKey("assertions"), knownvalue.Null()),
			},
		},
	})
}

// checkGroupV2UseGlobalAlertSettingsTestObjectType is a minimal resource
// object type with an enforce_alert_settings object that holds the members
// the checkly_check_group_v2 plan modifier and warning read.
var checkGroupV2UseGlobalAlertSettingsTestObjectType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"enforce_alert_settings": tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"enabled":                           tftypes.Bool,
		useGlobalAlertSettingsAttributeName: tftypes.Bool,
		alertSettingsAttributeName:          useGlobalAlertSettingsTestObjectType.AttributeTypes[alertSettingsAttributeName],
	}},
}}

var checkGroupV2UseGlobalAlertSettingsTestSchema = schema.Schema{
	Attributes: map[string]schema.Attribute{
		"enforce_alert_settings": schema.SingleNestedAttribute{
			Optional: true,
			Computed: true,
			Attributes: map[string]schema.Attribute{
				"enabled":                           schema.BoolAttribute{Required: true},
				useGlobalAlertSettingsAttributeName: schema.BoolAttribute{Optional: true, Computed: true},
				alertSettingsAttributeName:          useGlobalAlertSettingsTestSchema.Attributes[alertSettingsAttributeName],
			},
		},
	},
}

var checkGroupV2UseGlobalAlertSettingsTestPath = path.Root("enforce_alert_settings").AtName(useGlobalAlertSettingsAttributeName)

// checkGroupV2UseGlobalAlertSettingsTestConfig returns a configuration with
// the given enforce_alert_settings members, or with enforcement, if it is
// non-nil, as the whole enforce_alert_settings value.
func checkGroupV2UseGlobalAlertSettingsTestConfig(enforcement *tftypes.Value, enabled, useGlobal, alertSettings tftypes.Value) tfsdk.Config {
	var value tftypes.Value
	if enforcement != nil {
		value = *enforcement
	} else {
		value = tftypes.NewValue(checkGroupV2UseGlobalAlertSettingsTestObjectType.AttributeTypes["enforce_alert_settings"], map[string]tftypes.Value{
			"enabled":                           enabled,
			useGlobalAlertSettingsAttributeName: useGlobal,
			alertSettingsAttributeName:          alertSettings,
		})
	}
	return tfsdk.Config{
		Schema: checkGroupV2UseGlobalAlertSettingsTestSchema,
		Raw: tftypes.NewValue(checkGroupV2UseGlobalAlertSettingsTestObjectType, map[string]tftypes.Value{
			"enforce_alert_settings": value,
		}),
	}
}

func TestCheckGroupV2UseGlobalAlertSettingsPlanModifier(t *testing.T) {
	ctx := context.Background()

	enforcementType := checkGroupV2UseGlobalAlertSettingsTestObjectType.AttributeTypes["enforce_alert_settings"]
	enforcementNull := tftypes.NewValue(enforcementType, nil)
	enforcementUnknown := tftypes.NewValue(enforcementType, tftypes.UnknownValue)
	unset := tftypes.NewValue(tftypes.Bool, nil)
	enabled := tftypes.NewValue(tftypes.Bool, true)
	disabled := tftypes.NewValue(tftypes.Bool, false)

	cases := map[string]struct {
		// enforcement, if set, replaces the whole enforce_alert_settings
		// configuration.
		enforcement   *tftypes.Value
		enabled       tftypes.Value
		useGlobal     tftypes.Value
		alertSettings tftypes.Value
		// planned is the value before this modifier runs: what the
		// enforcement default and plan modifiers produced, or unknown.
		planned types.Bool
		destroy bool
		want    types.Bool
	}{
		"enabled without alert_settings plans true": {
			enabled:       enabled,
			useGlobal:     unset,
			alertSettings: alertSettingsNull,
			planned:       types.BoolUnknown(),
			want:          types.BoolValue(true),
		},
		"enabled with alert_settings plans false": {
			enabled:       enabled,
			useGlobal:     unset,
			alertSettings: alertSettingsSet,
			planned:       types.BoolValue(true),
			want:          types.BoolValue(false),
		},
		"enabled with unknown alert_settings plans unknown": {
			enabled:       enabled,
			useGlobal:     unset,
			alertSettings: alertSettingsUnknown,
			planned:       types.BoolValue(false),
			want:          types.BoolUnknown(),
		},
		"enabled with configured false keeps it": {
			enabled:       enabled,
			useGlobal:     tftypes.NewValue(tftypes.Bool, false),
			alertSettings: alertSettingsNull,
			planned:       types.BoolValue(false),
			want:          types.BoolValue(false),
		},
		"enabled with configured true keeps it": {
			enabled:       enabled,
			useGlobal:     tftypes.NewValue(tftypes.Bool, true),
			alertSettings: alertSettingsSet,
			planned:       types.BoolValue(true),
			want:          types.BoolValue(true),
		},
		"unknown enabled plans unknown": {
			enabled:       tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue),
			useGlobal:     unset,
			alertSettings: alertSettingsNull,
			planned:       types.BoolValue(false),
			want:          types.BoolUnknown(),
		},
		"unknown enforcement plans unknown": {
			enforcement: &enforcementUnknown,
			planned:     types.BoolUnknown(),
			want:        types.BoolUnknown(),
		},
		"omitted enforcement keeps the default null": {
			enforcement: &enforcementNull,
			planned:     types.BoolNull(),
			want:        types.BoolNull(),
		},
		"omitted enforcement keeps a stored false": {
			enforcement: &enforcementNull,
			planned:     types.BoolValue(false),
			want:        types.BoolValue(false),
		},
		"disabled enforcement keeps a stored false": {
			enabled:       disabled,
			useGlobal:     unset,
			alertSettings: alertSettingsNull,
			planned:       types.BoolValue(false),
			want:          types.BoolValue(false),
		},
		"disabled enforcement keeps a stored null": {
			enabled:       disabled,
			useGlobal:     unset,
			alertSettings: alertSettingsNull,
			planned:       types.BoolNull(),
			want:          types.BoolNull(),
		},
		"disabled enforcement plans null for unknown": {
			enabled:       disabled,
			useGlobal:     unset,
			alertSettings: alertSettingsSet,
			planned:       types.BoolUnknown(),
			want:          types.BoolNull(),
		},
		"destroy is left alone": {
			enabled:       enabled,
			useGlobal:     unset,
			alertSettings: alertSettingsNull,
			planned:       types.BoolNull(),
			destroy:       true,
			want:          types.BoolNull(),
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			config := checkGroupV2UseGlobalAlertSettingsTestConfig(c.enforcement, c.enabled, c.useGlobal, c.alertSettings)

			// GetAttribute of a member of a null or unknown object returns
			// null, as the framework passes it to nested plan modifiers.
			configValue := types.BoolNull()
			if c.enforcement == nil {
				if diags := config.GetAttribute(ctx, checkGroupV2UseGlobalAlertSettingsTestPath, &configValue); diags.HasError() {
					t.Fatal(diags)
				}
			}

			plan := tfsdk.Plan{Schema: config.Schema, Raw: config.Raw}
			if c.destroy {
				plan.Raw = tftypes.NewValue(checkGroupV2UseGlobalAlertSettingsTestObjectType, nil)
			}

			resp := planmodifier.BoolResponse{PlanValue: c.planned}
			checkGroupV2UseGlobalAlertSettingsPlanModifier{}.PlanModifyBool(ctx, planmodifier.BoolRequest{
				Path:        checkGroupV2UseGlobalAlertSettingsTestPath,
				Config:      config,
				ConfigValue: configValue,
				Plan:        plan,
				PlanValue:   c.planned,
			}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			if !resp.PlanValue.Equal(c.want) {
				t.Errorf("planned %s, want %s", resp.PlanValue, c.want)
			}
		})
	}
}

func TestCheckGroupV2UseGlobalAlertSettingsValidator(t *testing.T) {
	ctx := context.Background()

	configuredTrue := tftypes.NewValue(tftypes.Bool, true)

	enforcementType := checkGroupV2UseGlobalAlertSettingsTestObjectType.AttributeTypes["enforce_alert_settings"]
	enforcementNull := tftypes.NewValue(enforcementType, nil)
	enforcementUnknown := tftypes.NewValue(enforcementType, tftypes.UnknownValue)

	cases := map[string]struct {
		// enforcement, if set, replaces the whole enforce_alert_settings
		// configuration.
		enforcement   *tftypes.Value
		enabled       tftypes.Value
		useGlobal     tftypes.Value
		alertSettings tftypes.Value
		wantWarning   bool
	}{
		"omitted enforcement": {
			enforcement: &enforcementNull,
		},
		"unknown enforcement": {
			enforcement: &enforcementUnknown,
		},
		"enabled, true with alert_settings": {
			enabled:       tftypes.NewValue(tftypes.Bool, true),
			useGlobal:     configuredTrue,
			alertSettings: alertSettingsSet,
			wantWarning:   true,
		},
		"enabled, true without alert_settings": {
			enabled:       tftypes.NewValue(tftypes.Bool, true),
			useGlobal:     configuredTrue,
			alertSettings: alertSettingsNull,
		},
		"enabled, false with alert_settings": {
			enabled:       tftypes.NewValue(tftypes.Bool, true),
			useGlobal:     tftypes.NewValue(tftypes.Bool, false),
			alertSettings: alertSettingsSet,
		},
		"disabled, true with alert_settings": {
			enabled:       tftypes.NewValue(tftypes.Bool, false),
			useGlobal:     configuredTrue,
			alertSettings: alertSettingsSet,
		},
		"unknown enabled, true with alert_settings": {
			enabled:       tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue),
			useGlobal:     configuredTrue,
			alertSettings: alertSettingsSet,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			config := checkGroupV2UseGlobalAlertSettingsTestConfig(c.enforcement, c.enabled, c.useGlobal, c.alertSettings)

			// The framework validates a member of a null or unknown object
			// with a null value.
			configValue := types.BoolNull()
			if c.enforcement == nil {
				if diags := config.GetAttribute(ctx, checkGroupV2UseGlobalAlertSettingsTestPath, &configValue); diags.HasError() {
					t.Fatal(diags)
				}
			}

			var resp validator.BoolResponse
			checkGroupV2UseGlobalAlertSettingsValidator{}.ValidateBool(ctx, validator.BoolRequest{
				Config:      config,
				ConfigValue: configValue,
				Path:        checkGroupV2UseGlobalAlertSettingsTestPath,
			}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			if got := resp.Diagnostics.WarningsCount() > 0; got != c.wantWarning {
				t.Errorf("warning reported = %v, want %v: %v", got, c.wantWarning, resp.Diagnostics)
			}
		})
	}
}

func TestCheckGroupV2UseGlobalAlertSettingsInRequest(t *testing.T) {
	ctx := context.Background()

	setAlertSettings := types.ObjectValueMust(alertSettingsAttrTypes, func() map[string]attr.Value {
		values := make(map[string]attr.Value, len(alertSettingsAttrTypes))
		for name, attrType := range alertSettingsAttrTypes {
			switch t := attrType.(type) {
			case types.ObjectType:
				values[name] = types.ObjectNull(t.AttrTypes)
			default:
				values[name] = types.StringValue("RUN_BASED")
			}
		}
		return values
	}())

	cases := map[string]struct {
		useGlobal     types.Bool
		alertSettings types.Object
		want          *bool
	}{
		"known true is sent": {
			useGlobal:     types.BoolValue(true),
			alertSettings: setAlertSettings,
			want:          new(true),
		},
		"known false is sent": {
			useGlobal:     types.BoolValue(false),
			alertSettings: types.ObjectNull(alertSettingsAttrTypes),
			want:          new(false),
		},
		"null is not sent": {
			useGlobal:     types.BoolNull(),
			alertSettings: types.ObjectNull(alertSettingsAttrTypes),
		},
		"unknown without alert_settings resolves to true": {
			useGlobal:     types.BoolUnknown(),
			alertSettings: types.ObjectNull(alertSettingsAttrTypes),
			want:          new(true),
		},
		"unknown with computed alert_settings resolves to true": {
			useGlobal:     types.BoolUnknown(),
			alertSettings: types.ObjectUnknown(alertSettingsAttrTypes),
			want:          new(true),
		},
		"unknown with alert_settings resolves to false": {
			useGlobal:     types.BoolUnknown(),
			alertSettings: setAlertSettings,
			want:          new(false),
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			model := CheckGroupV2ResourceModel{
				Name: types.StringValue("test"),
				EnforceAlertSettings: types.ObjectValueMust(checkGroupV2EnforceAlertSettingsAttrTypes, map[string]attr.Value{
					"enabled":                             types.BoolValue(true),
					alertSettingsAttributeName:            c.alertSettings,
					useGlobalAlertSettingsAttributeName:   c.useGlobal,
					alertChannelSubscriptionAttributeName: types.SetNull(checkGroupV2AlertChannelSubscriptionSetType.ElemType),
				}),
			}

			var diags diag.Diagnostics
			group := checkGroupV2FromModel(ctx, &model, &diags)
			if diags.HasError() {
				t.Fatal(diags)
			}

			got := group.UseGlobalAlertSettings
			switch {
			case c.want == nil && got != nil:
				t.Errorf("UseGlobalAlertSettings = %v, want nil", *got)
			case c.want != nil && (got == nil || *got != *c.want):
				t.Errorf("UseGlobalAlertSettings = %v, want %v", got, *c.want)
			}
		})
	}
}
