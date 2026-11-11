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
				resource.TestCheckTypeSetElemAttr(
					"checkly_browser_check.test",
					"locations.*",
					"us-east-1",
				),
				resource.TestCheckTypeSetElemAttr(
					"checkly_browser_check.test",
					"locations.*",
					"eu-central-1",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"tags.#",
					"2",
				),
				resource.TestCheckTypeSetElemAttr(
					"checkly_browser_check.test",
					"tags.*",
					"browser",
				),
				resource.TestCheckTypeSetElemAttr(
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

				environment_variables = [
					{
						key    = "FOO"
						value  = "bar"
						locked = false
					},
					{
						key    = "SECRET"
						value  = "s3cr3t"
						secret = true
					},
				]
			}`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"environment_variables.#",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"environment_variables.0.key",
					"FOO",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"environment_variables.0.value",
					"bar",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"environment_variables.1.key",
					"SECRET",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"environment_variables.1.value",
					"s3cr3t",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"environment_variables.1.secret",
					"true",
				),
			),
		},
		// An empty list removes all variables and is stored as an empty
		// list, not null; otherwise the post-apply plan would not be empty.
		{
			Config: `resource "checkly_browser_check" "test" {
				name                      = "Browser Check Resource Env Vars"
				activated                 = true
				frequency                 = 720
				use_global_alert_settings = true
				locations                 = ["us-east-1"]
				script                    = "console.log('test')"

				environment_variables = []
			}`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"environment_variables.#",
					"0",
				),
			),
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue(
					"checkly_browser_check.test",
					tfjsonpath.New("environment_variables"),
					knownvalue.ListExact([]knownvalue.Check{}),
				),
			},
		},
		// Leaving the attribute out stores null.
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
				// Unlike a check of the element count, which also passes
				// for an empty list, this fails unless the value is null.
				resource.TestCheckNoResourceAttr(
					"checkly_browser_check.test",
					"environment_variables",
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

				dependencies = [
					{
						path    = "__checks__/util.ts"
						content = "export const foo = 'bar'"
					},
					{
						path    = "__checks__/data.json"
						content = "{\"foo\": \"bar\"}"
					},
				]
			}`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"script_path",
					"__checks__/main.spec.ts",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"dependencies.#",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"dependencies.0.path",
					"__checks__/util.ts",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"dependencies.0.content",
					"export const foo = 'bar'",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"dependencies.1.path",
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

				dependencies = [
					{
						path    = "__checks__/util.ts"
						content = "export const foo = 'baz'"
					},
					{
						path    = "__checks__/data.json"
						content = "{\"foo\": \"bar\"}"
					},
				]
			}`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"dependencies.#",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"dependencies.0.content",
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
			ImportStateVerifyIgnore: []string{"dependencies"},
		},
		{
			// Removing every dependency must clear the dependencies
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
					"dependencies.#",
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
				ai_auto_repair_enabled    = false
				intent = {
					goal = "tf-acc: the page keeps loading"
				}
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
					ai_auto_repair_enabled    = false
					intent = {
						goal = "tf-acc: the page keeps loading"
					}
				}

				moved {
					from = checkly_check.test
					to   = checkly_browser_check.test
				}
			`,
			// The move carries the intent and the automatic repair override
			// over, so it changes nothing.
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction("checkly_browser_check.test", plancheck.ResourceActionNoop),
				},
			},
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_browser_check.test", "intent.goal", "tf-acc: the page keeps loading"),
				resource.TestCheckResourceAttr("checkly_browser_check.test", "ai_auto_repair_enabled", "false"),
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

// TestAccBrowserCheckUnsetSweepScalars sets every optional scalar attribute
// to a non-default value and then removes them all, verifying the update
// endpoint clears or resets them (the endpoint is PATCH-style, so explicit
// values must be sent for the reset to take effect).
func TestAccBrowserCheckUnsetSweepScalars(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_browser_check" "test" {
					name                      = "browser-check-unset-sweep-scalars"
					description               = "sweep description"
					frequency                 = 720
					activated                 = false
					muted                     = true
					run_parallel              = true
					use_global_alert_settings = false
					ssl_check_domain          = "checklyhq.com"
					runtime_id                = "2023.09"
					script_path               = "__checks__/sweep.spec.ts"
					locations                 = ["us-east-1"]
					script                    = "console.log('test')"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"description",
					"sweep description",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"activated",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"muted",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"run_parallel",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"use_global_alert_settings",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"ssl_check_domain",
					"checklyhq.com",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"runtime_id",
					"2023.09",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"script_path",
					"__checks__/sweep.spec.ts",
				),
			),
		},
		{
			Config: `
				resource "checkly_browser_check" "test" {
					name      = "browser-check-unset-sweep-scalars"
					frequency = 720
					locations = ["us-east-1"]
					script    = "console.log('test')"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"description",
					"",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"activated",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"muted",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"run_parallel",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"use_global_alert_settings",
					"true",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_browser_check.test",
					"ssl_check_domain",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_browser_check.test",
					"runtime_id",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_browser_check.test",
					"script_path",
				),
			),
		},
	})
}

// TestAccBrowserCheckUnsetSweepCollections sets the locations and
// private_locations collections and then removes them, verifying that removal
// clears the values remotely. Tags, environment variables and dependencies
// have dedicated removal coverage in other tests.
func TestAccBrowserCheckUnsetSweepCollections(t *testing.T) {
	// Private location slugs are unique per account and limited to 30
	// characters, so a short random suffix avoids collisions between runs.
	slug := fmt.Sprintf("tf-browser-unset-%d", acctest.RandInt()%100000)
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
				resource "checkly_browser_check" "test" {
					name              = "browser-check-unset-sweep-collections"
					activated         = false
					frequency         = 720
					locations         = ["us-east-1", "eu-central-1"]
					private_locations = [checkly_private_location.sweep.slug_name]
					script            = "console.log('test')"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"locations.#",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"private_locations.#",
					"1",
				),
			),
		},
		// Removing locations must clear them remotely. The check keeps
		// running from the private location.
		{
			Config: privateLocation + `
				resource "checkly_browser_check" "test" {
					name              = "browser-check-unset-sweep-collections"
					activated         = false
					frequency         = 720
					private_locations = [checkly_private_location.sweep.slug_name]
					script            = "console.log('test')"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(
					"checkly_browser_check.test",
					"locations.#",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"private_locations.#",
					"1",
				),
			),
		},
		// An explicitly empty private_locations is stored as an empty set.
		// The check needs a public location again to remain valid.
		{
			Config: privateLocation + `
				resource "checkly_browser_check" "test" {
					name              = "browser-check-unset-sweep-collections"
					activated         = false
					frequency         = 720
					locations         = ["us-east-1"]
					private_locations = []
					script            = "console.log('test')"
				}
			`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue(
					"checkly_browser_check.test",
					tfjsonpath.New("private_locations"),
					knownvalue.SetExact([]knownvalue.Check{}),
				),
			},
		},
		// Leaving private_locations out stores null.
		{
			Config: privateLocation + `
				resource "checkly_browser_check" "test" {
					name      = "browser-check-unset-sweep-collections"
					activated = false
					frequency = 720
					locations = ["us-east-1"]
					script    = "console.log('test')"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"locations.#",
					"1",
				),
			),
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue(
					"checkly_browser_check.test",
					tfjsonpath.New("private_locations"),
					knownvalue.Null(),
				),
			},
		},
	})
}

const browserCheckUnsetSweepDeps = `
	resource "checkly_check_group" "sweep" {
		name        = "browser-check-unset-sweep-group"
		activated   = true
		concurrency = 1
		locations   = ["eu-central-1"]
	}

	resource "checkly_alert_channel" "sweep" {
		email = {
			address = "browser-check-unset-sweep@example.com"
		}
	}

	resource "checkly_status_page_service" "sweep" {
		name = "browser-check-unset-sweep-service"
	}
`

// TestAccBrowserCheckUnsetSweepGroupAndAlerts assigns the check to a group
// (with an explicit group order), subscribes it to an alert channel, attaches
// an incident trigger and configures a retry strategy, then removes them,
// verifying the removals clear the values remotely.
func TestAccBrowserCheckUnsetSweepGroupAndAlerts(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: browserCheckUnsetSweepDeps + `
				resource "checkly_browser_check" "test" {
					name        = "browser-check-unset-sweep-group-alerts"
					activated   = false
					frequency   = 720
					locations   = ["eu-central-1"]
					script      = "console.log('test')"
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
						name               = "Browser check down"
						description        = "The browser check has detected a failure"
						notify_subscribers = false
					}

					retry_strategy = {
						type        = "FIXED"
						max_retries = 3
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrPair(
					"checkly_browser_check.test",
					"group_id",
					"checkly_check_group.sweep",
					"id",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"group_order",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"alert_channel_subscriptions.#",
					"1",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"trigger_incident.%",
					"5",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"retry_strategy.type",
					"FIXED",
				),
			),
		},
		// Removing group_order alone must clear the position remotely while the
		// check stays in the group.
		{
			Config: browserCheckUnsetSweepDeps + `
				resource "checkly_browser_check" "test" {
					name      = "browser-check-unset-sweep-group-alerts"
					activated = false
					frequency = 720
					locations = ["eu-central-1"]
					script    = "console.log('test')"
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
						name               = "Browser check down"
						description        = "The browser check has detected a failure"
						notify_subscribers = false
					}

					retry_strategy = {
						type        = "FIXED"
						max_retries = 3
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrPair(
					"checkly_browser_check.test",
					"group_id",
					"checkly_check_group.sweep",
					"id",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_browser_check.test",
					"group_order",
				),
			),
		},
		// Removing the group assignment, the alert channel subscription, the
		// incident trigger and the retry strategy must clear them all
		// remotely (retry_strategy resets to the NO_RETRIES default).
		{
			Config: browserCheckUnsetSweepDeps + `
				resource "checkly_browser_check" "test" {
					name      = "browser-check-unset-sweep-group-alerts"
					activated = false
					frequency = 720
					locations = ["eu-central-1"]
					script    = "console.log('test')"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(
					"checkly_browser_check.test",
					"group_id",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_browser_check.test",
					"group_order",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"alert_channel_subscriptions.#",
					"0",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_browser_check.test",
					"trigger_incident",
				),
				resource.TestCheckResourceAttr(
					"checkly_browser_check.test",
					"retry_strategy.type",
					"NO_RETRIES",
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
