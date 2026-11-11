package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccMultistepCheckResourceBasic(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: multistepCheckResource_basic,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"name",
					"Multistep Check Resource",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"runtime_id",
					"2023.09",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"activated",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"script",
					"console.log('test')",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"locations.#",
					"2",
				),
				resource.TestCheckTypeSetElemAttr(
					"checkly_multistep_check.test",
					"locations.*",
					"us-east-1",
				),
				resource.TestCheckTypeSetElemAttr(
					"checkly_multistep_check.test",
					"locations.*",
					"eu-central-1",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"tags.#",
					"2",
				),
				resource.TestCheckTypeSetElemAttr(
					"checkly_multistep_check.test",
					"tags.*",
					"api",
				),
				resource.TestCheckTypeSetElemAttr(
					"checkly_multistep_check.test",
					"tags.*",
					"multi-step",
				),
			),
		},
	})
}

func TestAccMultistepCheckResourceRuntimeValidation(t *testing.T) {
	unsupportedRuntime := `resource "checkly_multistep_check" "test" {
		name       = "Multistep Check Resource Runtime"
		activated  = true
		frequency  = 5
		locations  = ["eu-central-1"]
		script     = "console.log('test')"
		runtime_id = "2023.02"
	}`
	noSpecifiedRuntime := `resource "checkly_multistep_check" "test" {
		name      = "Multistep Check Resource Runtime"
		activated = true
		frequency = 5
		locations = ["eu-central-1"]
		script    = "console.log('test')"
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      unsupportedRuntime,
			ExpectError: regexp.MustCompile("runtime 2023.02 does not support MULTI_STEP checks"),
		},
		{
			Config: noSpecifiedRuntime,
			Check: resource.TestCheckNoResourceAttr(
				"checkly_multistep_check.test",
				"runtime_id",
			),
		},
	})
}

func TestAccMultistepCheckResourceDependencies(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `resource "checkly_multistep_check" "test" {
				name                      = "Multistep Check Resource Dependencies"
				activated                 = true
				frequency                 = 720
				use_global_alert_settings = true
				locations                 = ["us-east-1"]
				runtime_id                = "2023.09"
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
					"checkly_multistep_check.test",
					"script_path",
					"__checks__/main.spec.ts",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"dependencies.#",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"dependencies.0.path",
					"__checks__/util.ts",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"dependencies.0.content",
					"export const foo = 'bar'",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"dependencies.1.path",
					"__checks__/data.json",
				),
			),
		},
		{
			Config: `resource "checkly_multistep_check" "test" {
				name                      = "Multistep Check Resource Dependencies"
				activated                 = true
				frequency                 = 720
				use_global_alert_settings = true
				locations                 = ["us-east-1"]
				runtime_id                = "2023.09"
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
					"checkly_multistep_check.test",
					"dependencies.#",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"dependencies.0.content",
					"export const foo = 'baz'",
				),
			),
		},
		{
			ResourceName:      "checkly_multistep_check.test",
			ImportState:       true,
			ImportStateVerify: true,
			// The API returns dependencies ordered by path, while the
			// existing state keeps the configured order. An imported state
			// has no prior value to recover the configured order from, so
			// the element order may legitimately differ.
			ImportStateVerifyIgnore: []string{"dependencies"},
		},
		{
			// An empty list must clear the dependencies remotely and is
			// stored as an empty list, not null; otherwise the post-apply
			// refresh and plan would not be empty.
			Config: `resource "checkly_multistep_check" "test" {
				name                      = "Multistep Check Resource Dependencies"
				activated                 = true
				frequency                 = 720
				use_global_alert_settings = true
				locations                 = ["us-east-1"]
				runtime_id                = "2023.09"
				script                    = "console.log('main')"

				dependencies = []
			}`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"dependencies.#",
					"0",
				),
			),
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue(
					"checkly_multistep_check.test",
					tfjsonpath.New("dependencies"),
					knownvalue.ListExact([]knownvalue.Check{}),
				),
			},
		},
		{
			// Leaving the attribute out stores null.
			Config: `resource "checkly_multistep_check" "test" {
				name                      = "Multistep Check Resource Dependencies"
				activated                 = true
				frequency                 = 720
				use_global_alert_settings = true
				locations                 = ["us-east-1"]
				runtime_id                = "2023.09"
				script                    = "console.log('main')"
			}`,
			Check: resource.ComposeTestCheckFunc(
				// Unlike a check of the element count, which also passes
				// for an empty list, this fails unless the value is null.
				resource.TestCheckNoResourceAttr(
					"checkly_multistep_check.test",
					"dependencies",
				),
			),
		},
	})
}

// TestAccMultistepCheckResourceMovedFromCheck exercises the MoveState
// implementation of checkly_multistep_check via a `moved` block from
// checkly_check.
func TestAccMultistepCheckResourceMovedFromCheck(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `resource "checkly_check" "test" {
				name                      = "Multistep Check Resource Moved"
				type                      = "MULTI_STEP"
				activated                 = true
				frequency                 = 10
				use_global_alert_settings = true
				locations                 = ["us-east-1"]
				runtime_id                = "2023.09"
				script                    = "console.log('test')"
			}`,
		},
		{
			Config: `
				resource "checkly_multistep_check" "test" {
					name                      = "Multistep Check Resource Moved"
					activated                 = true
					frequency                 = 10
					use_global_alert_settings = true
					locations                 = ["us-east-1"]
					runtime_id                = "2023.09"
					script                    = "console.log('test')"
				}

				moved {
					from = checkly_check.test
					to   = checkly_multistep_check.test
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"name",
					"Multistep Check Resource Moved",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"activated",
					"true",
				),
			),
		},
	})
}

// TestAccMultistepCheckUnsetSweepScalars sets every optional scalar
// attribute to a non-default value and then removes them all, verifying the
// update endpoint clears or resets them (the endpoint is PATCH-style, so
// explicit values must be sent for the reset to take effect).
func TestAccMultistepCheckUnsetSweepScalars(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_multistep_check" "test" {
					name                      = "multistep-check-unset-sweep-scalars"
					description               = "sweep description"
					frequency                 = 720
					activated                 = false
					muted                     = true
					run_parallel              = true
					use_global_alert_settings = false
					runtime_id                = "2023.09"
					script_path               = "__checks__/sweep.spec.ts"
					locations                 = ["us-east-1"]
					script                    = "console.log('test')"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"description",
					"sweep description",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"activated",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"muted",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"run_parallel",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"use_global_alert_settings",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"runtime_id",
					"2023.09",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"script_path",
					"__checks__/sweep.spec.ts",
				),
			),
		},
		{
			Config: `
				resource "checkly_multistep_check" "test" {
					name      = "multistep-check-unset-sweep-scalars"
					frequency = 720
					locations = ["us-east-1"]
					script    = "console.log('test')"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"description",
					"",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"activated",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"muted",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"run_parallel",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"use_global_alert_settings",
					"true",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_multistep_check.test",
					"runtime_id",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_multistep_check.test",
					"script_path",
				),
			),
		},
	})
}

// TestAccMultistepCheckUnsetSweepCollections sets the tags, locations and
// private_locations collections and the environment_variables list, then
// removes them, verifying that removal clears the values remotely.
// Dependencies have dedicated removal coverage in another test.
func TestAccMultistepCheckUnsetSweepCollections(t *testing.T) {
	// Private location slugs are unique per account and limited to 30
	// characters, so a short random suffix avoids collisions between runs.
	slug := fmt.Sprintf("tf-multistep-unset-%d", acctest.RandInt()%100000)
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
				resource "checkly_multistep_check" "test" {
					name              = "multistep-check-unset-sweep-collections"
					activated         = false
					frequency         = 720
					locations         = ["us-east-1", "eu-central-1"]
					private_locations = [checkly_private_location.sweep.slug_name]
					tags              = ["sweep-a", "sweep-b"]
					script            = "console.log('test')"

					environment_variables = [
						{
							key    = "SWEEP_FOO"
							value  = "bar"
							locked = false
						},
						{
							key    = "SWEEP_SECRET"
							value  = "s3cr3t"
							secret = true
						},
					]
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"locations.#",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"private_locations.#",
					"1",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"tags.#",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"environment_variables.#",
					"2",
				),
			),
		},
		// Removing tags, locations and environment variables must clear them
		// remotely. The check keeps running from the private location. The
		// value of the secret environment variable is masked on reads, but
		// its removal (the count dropping to 0) is observable.
		{
			Config: privateLocation + `
				resource "checkly_multistep_check" "test" {
					name              = "multistep-check-unset-sweep-collections"
					activated         = false
					frequency         = 720
					private_locations = [checkly_private_location.sweep.slug_name]
					script            = "console.log('test')"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(
					"checkly_multistep_check.test",
					"locations.#",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_multistep_check.test",
					"tags.#",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"environment_variables.#",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"private_locations.#",
					"1",
				),
			),
		},
		// An explicitly empty private_locations is stored as an empty set.
		// The check needs a public location again to remain valid.
		{
			Config: privateLocation + `
				resource "checkly_multistep_check" "test" {
					name              = "multistep-check-unset-sweep-collections"
					activated         = false
					frequency         = 720
					locations         = ["us-east-1"]
					private_locations = []
					script            = "console.log('test')"
				}
			`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue(
					"checkly_multistep_check.test",
					tfjsonpath.New("private_locations"),
					knownvalue.SetExact([]knownvalue.Check{}),
				),
			},
		},
		// Leaving private_locations out stores null.
		{
			Config: privateLocation + `
				resource "checkly_multistep_check" "test" {
					name      = "multistep-check-unset-sweep-collections"
					activated = false
					frequency = 720
					locations = ["us-east-1"]
					script    = "console.log('test')"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"locations.#",
					"1",
				),
			),
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue(
					"checkly_multistep_check.test",
					tfjsonpath.New("private_locations"),
					knownvalue.Null(),
				),
			},
		},
	})
}

const multistepCheckUnsetSweepDeps = `
	resource "checkly_check_group" "sweep" {
		name        = "multistep-check-unset-sweep-group"
		activated   = true
		concurrency = 1
		locations   = ["eu-central-1"]
	}

	resource "checkly_alert_channel" "sweep" {
		email = {
			address = "multistep-check-unset-sweep@example.com"
		}
	}

	resource "checkly_status_page_service" "sweep" {
		name = "multistep-check-unset-sweep-service"
	}
`

// TestAccMultistepCheckUnsetSweepGroupAndAlerts assigns the check to a group
// (with an explicit group order), subscribes it to an alert channel, attaches
// an incident trigger and configures a retry strategy, then removes them,
// verifying the removals clear the values remotely.
func TestAccMultistepCheckUnsetSweepGroupAndAlerts(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: multistepCheckUnsetSweepDeps + `
				resource "checkly_multistep_check" "test" {
					name        = "multistep-check-unset-sweep-group-alerts"
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
						name               = "Multistep check down"
						description        = "The multistep check has detected a failure"
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
					"checkly_multistep_check.test",
					"group_id",
					"checkly_check_group.sweep",
					"id",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"group_order",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"alert_channel_subscriptions.#",
					"1",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"trigger_incident.%",
					"5",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"retry_strategy.type",
					"FIXED",
				),
			),
		},
		// Removing group_order alone must clear the position remotely while the
		// check stays in the group.
		{
			Config: multistepCheckUnsetSweepDeps + `
				resource "checkly_multistep_check" "test" {
					name      = "multistep-check-unset-sweep-group-alerts"
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
						name               = "Multistep check down"
						description        = "The multistep check has detected a failure"
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
					"checkly_multistep_check.test",
					"group_id",
					"checkly_check_group.sweep",
					"id",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_multistep_check.test",
					"group_order",
				),
			),
		},
		// Removing the group assignment, the alert channel subscription, the
		// incident trigger and the retry strategy must clear them all
		// remotely (retry_strategy resets to the NO_RETRIES default).
		{
			Config: multistepCheckUnsetSweepDeps + `
				resource "checkly_multistep_check" "test" {
					name      = "multistep-check-unset-sweep-group-alerts"
					activated = false
					frequency = 720
					locations = ["eu-central-1"]
					script    = "console.log('test')"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(
					"checkly_multistep_check.test",
					"group_id",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_multistep_check.test",
					"group_order",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"alert_channel_subscriptions.#",
					"0",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_multistep_check.test",
					"trigger_incident",
				),
				resource.TestCheckResourceAttr(
					"checkly_multistep_check.test",
					"retry_strategy.type",
					"NO_RETRIES",
				),
			),
		},
	})
}

const multistepCheckResource_basic = `
	resource "checkly_multistep_check" "test" {
		name                      = "Multistep Check Resource"
		activated                 = true
		frequency                 = 720
		use_global_alert_settings = true
		locations                 = [ "us-east-1", "eu-central-1" ]
		tags                      = [ "api", "multi-step" ]
		runtime_id                = "2023.09"
		script                    = "console.log('test')"
	}
`
