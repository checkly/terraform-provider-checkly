package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

const playwrightCheckSuiteBase = `
	resource "checkly_playwright_code_bundle" "test" {
		prebuilt_archive = {
			file = "../../fixtures/playwright-project-pnpm.tar.gz"
		}
	}
`

func TestAccPlaywrightCheckSuiteWithEnvironmentVariable(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check with env vars"
					description               = "Playwright check description"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {
						steps = {
							install = {
								command = "pnpm i"
							}

							test = {
								command = "pnpm playwright test"
							}
						}

						playwright = {
							version = "1.56.1"
							devices = [
								{
									type = "chromium"
								},
							]
						}
					}

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
					"checkly_playwright_check_suite.test",
					"description",
					"Playwright check description",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"environment_variables.#",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"environment_variables.0.key",
					"FOO",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"environment_variables.0.value",
					"bar",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"environment_variables.0.locked",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"environment_variables.1.key",
					"SECRET",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"environment_variables.1.value",
					"s3cr3t",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"environment_variables.1.locked",
					"true",
				),
			),
		},
		{
			ResourceName:      "checkly_playwright_check_suite.test",
			ImportState:       true,
			ImportStateVerify: true,
			// The bundle attachment exists only in Terraform state: its
			// metadata is computed from the local archive and the API never
			// returns it, so an imported check has no bundle until the next
			// apply attaches the configured one.
			ImportStateVerifyIgnore: []string{"bundle"},
		},
	})
}

func TestAccPlaywrightCheckSuiteDescriptionRemoval(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check description removal"
					description               = "Playwright check description"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}
				}
			`,
			Check: resource.TestCheckResourceAttr(
				"checkly_playwright_check_suite.test",
				"description",
				"Playwright check description",
			),
		},
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check description removal"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}
				}
			`,
			Check: resource.TestCheckResourceAttr(
				"checkly_playwright_check_suite.test",
				"description",
				"",
			),
		},
	})
}

// TestAccPlaywrightCheckSuitePrivateLocationsOnly verifies that a suite
// running only in a private location, with locations omitted, applies
// cleanly and leaves no diff after refresh.
func TestAccPlaywrightCheckSuitePrivateLocationsOnly(t *testing.T) {
	// Private location slugs are unique per account and limited to 30
	// characters, so a short random suffix avoids collisions between runs.
	slug := fmt.Sprintf("tf-pw-private-%d", acctest.RandInt()%100000)
	accTestCase(t, []resource.TestStep{
		{
			Config: playwrightCheckSuiteBase + fmt.Sprintf(`
				resource "checkly_private_location" "test" {
					name      = %[1]q
					slug_name = %[1]q
				}

				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check private locations only"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					private_locations         = [checkly_private_location.test.slug_name]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}
				}
			`, slug),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_playwright_check_suite.test", "private_locations.#", "1"),
			),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PostApplyPostRefresh: []plancheck.PlanCheck{
					plancheck.ExpectEmptyPlan(),
				},
			},
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue("checkly_playwright_check_suite.test", tfjsonpath.New("locations"), emptySet),
			},
		},
	})
}

func TestAccPlaywrightCheckSuiteBundleChange(t *testing.T) {
	pnpmBundle := `
		resource "checkly_playwright_code_bundle" "test" {
			prebuilt_archive = {
				file = "../../fixtures/playwright-project-pnpm.tar.gz"
			}
		}
	`

	pnpmNextBundle := `
		resource "checkly_playwright_code_bundle" "test" {
			prebuilt_archive = {
				file = "../../fixtures/playwright-project-pnpm-playwright-next.tar.gz"
			}
		}
	`

	checkSuite := `
		resource "checkly_playwright_check_suite" "test" {
			name                      = "PW Check bundle change"
			activated                 = true
			frequency                 = 720
			use_global_alert_settings = true
			locations                 = ["us-east-1"]

			bundle = {
				id       = checkly_playwright_code_bundle.test.id
				metadata = checkly_playwright_code_bundle.test.metadata
			}
		}
	`

	accTestCase(t, []resource.TestStep{
		// Step 1: Use the pnpm fixture.
		{
			Config: pnpmBundle + checkSuite,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.playwright.version",
					"1.58.2",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.steps.test.command",
					playwrightCheckSuiteDefaultTestCommands["pnpm"],
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.working_dir",
					".",
				),
			),
		},
		// Step 2: Swap to a different archive — auto-detected values should
		// update. A new archive replaces the code bundle, while the check
		// suite that references it is updated in place.
		{
			Config: pnpmNextBundle + checkSuite,
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction("checkly_playwright_code_bundle.test", plancheck.ResourceActionReplace),
					plancheck.ExpectResourceAction("checkly_playwright_check_suite.test", plancheck.ResourceActionUpdate),
				},
			},
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.playwright.version",
					"1.59.0-alpha-1774287265000",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.steps.test.command",
					playwrightCheckSuiteDefaultTestCommands["pnpm"],
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.working_dir",
					".",
				),
			),
		},
	})
}

func TestAccPlaywrightCheckSuiteWithoutRuntime(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check without runtime"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.playwright.version",
					"1.58.2",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.playwright.devices.#",
					fmt.Sprintf("%d", len(playwrightCheckSuiteDefaultBrowsers)),
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.working_dir",
					".",
				),
			),
		},
	})
}

func TestAccPlaywrightCheckSuiteWithExplicitVersion(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check with explicit version"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {
						playwright = {
							version = "1.56.1"
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.playwright.version",
					"1.56.1",
				),
			),
		},
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check with explicit version"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {
						playwright = {}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.playwright.version",
					"1.58.2",
				),
			),
		},
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check with explicit version"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.playwright.version",
					"1.58.2",
				),
			),
		},
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check with explicit version"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.playwright.version",
					"1.58.2",
				),
			),
		},
	})
}

func TestAccPlaywrightCheckSuiteVersionRemovedWithDevicesKept(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check version removed devices kept"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {
						playwright = {
							version = "1.56.1"
							devices = [
								{
									type = "chromium"
								},
							]
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.playwright.version",
					"1.56.1",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.playwright.devices.#",
					"1",
				),
			),
		},
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check version removed devices kept"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {
						playwright = {
							devices = [
								{
									type = "chromium"
								},
							]
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.playwright.version",
					"1.58.2",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.playwright.devices.#",
					"1",
				),
			),
		},
	})
}

func TestAccPlaywrightCheckSuiteWithExplicitDevices(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check with explicit devices"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {
						playwright = {
							devices = [
								{
									type = "chromium"
								},
							]
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.playwright.devices.#",
					"1",
				),
			),
		},
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check with explicit devices"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {
						playwright = {}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.playwright.devices.#",
					fmt.Sprintf("%d", len(playwrightCheckSuiteDefaultBrowsers)),
				),
			),
		},
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check with explicit devices"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.playwright.devices.#",
					fmt.Sprintf("%d", len(playwrightCheckSuiteDefaultBrowsers)),
				),
			),
		},
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check with explicit devices"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.playwright.devices.#",
					fmt.Sprintf("%d", len(playwrightCheckSuiteDefaultBrowsers)),
				),
			),
		},
	})
}

func TestAccPlaywrightCheckSuiteTestCommand(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		// Step 1: No explicit test command — auto-detected from pnpm lockfile.
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check test command"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.steps.test.command",
					playwrightCheckSuiteDefaultTestCommands["pnpm"],
				),
			),
		},
		// Step 2: Explicit custom test command — should be preserved.
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check test command"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {
						steps = {
							test = {
								command = "pnpm playwright test --workers=4"
							}
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.steps.test.command",
					"pnpm playwright test --workers=4",
				),
			),
		},
		// Step 3: Remove explicit test command — falls back to auto-detected.
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check test command"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.steps.test.command",
					playwrightCheckSuiteDefaultTestCommands["pnpm"],
				),
			),
		},
	})
}

func TestAccPlaywrightCheckSuiteInstallCommand(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		// Step 1: Set an explicit install command.
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check install command"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {
						steps = {
							install = {
								command = "pnpm install --frozen-lockfile"
							}
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.steps.install.command",
					"pnpm install --frozen-lockfile",
				),
			),
		},
		// Step 2: Remove the install command — should be unset.
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check install command"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.steps.install.command",
				),
			),
		},
	})
}

func TestAccPlaywrightCheckSuiteInstallCommandPreservedAcrossChanges(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		// Step 1: Set install command alongside an explicit playwright version.
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check install preserved"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {
						steps = {
							install = {
								command = "pnpm install --frozen-lockfile"
							}
						}

						playwright = {
							version = "1.56.1"
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.steps.install.command",
					"pnpm install --frozen-lockfile",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.playwright.version",
					"1.56.1",
				),
			),
		},
		// Step 2: Change the playwright version — install command should be preserved.
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check install preserved"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {
						steps = {
							install = {
								command = "pnpm install --frozen-lockfile"
							}
						}

						playwright = {
							version = "1.58.2"
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.steps.install.command",
					"pnpm install --frozen-lockfile",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.playwright.version",
					"1.58.2",
				),
			),
		},
		// Step 3: Remove the playwright version (fall back to auto-detect) — install command should still be preserved.
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check install preserved"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {
						steps = {
							install = {
								command = "pnpm install --frozen-lockfile"
							}
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.steps.install.command",
					"pnpm install --frozen-lockfile",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.playwright.version",
					"1.58.2",
				),
			),
		},
	})
}

func TestAccPlaywrightCheckSuiteWithoutDevicesShouldNotCrash(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check without devices"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {
						playwright = {
							version = "1.56.1"
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.playwright.version",
					"1.56.1",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.playwright.devices.#",
					fmt.Sprintf("%d", len(playwrightCheckSuiteDefaultBrowsers)),
				),
			),
		},
	})
}

func TestAccPlaywrightCheckSuiteAutoDetectTransition(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		// Step 1: auto_detect = false with all explicit values.
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check auto_detect transition"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {
						auto_detect = false

						steps = {
							test = {
								command = "npx playwright test"
							}
						}

						playwright = {
							version = "1.56.1"
							devices = [
								{
									type = "chromium"
								},
							]
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.playwright.version",
					"1.56.1",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.playwright.devices.#",
					"1",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.steps.test.command",
					"npx playwright test",
				),
			),
		},
		// Step 2: Switch to auto_detect = true, keep explicit version but
		// remove devices and test command — those should be auto-detected.
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check auto_detect transition"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {
						playwright = {
							version = "1.56.1"
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.playwright.version",
					"1.56.1",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.playwright.devices.#",
					fmt.Sprintf("%d", len(playwrightCheckSuiteDefaultBrowsers)),
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.steps.test.command",
					playwrightCheckSuiteDefaultTestCommands["pnpm"],
				),
			),
		},
		// Step 3: Remove explicit version too — everything auto-detected.
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check auto_detect transition"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.playwright.version",
					"1.58.2",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.playwright.devices.#",
					fmt.Sprintf("%d", len(playwrightCheckSuiteDefaultBrowsers)),
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.steps.test.command",
					playwrightCheckSuiteDefaultTestCommands["pnpm"],
				),
			),
		},
	})
}

func TestAccPlaywrightCheckSuiteAutoDetectFalse(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check auto_detect false"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {
						auto_detect = false

						steps = {
							test = {
								command = "npx playwright test"
							}
						}

						playwright = {
							version = "1.56.1"
							devices = [
								{
									type = "chromium"
								},
							]
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.auto_detect",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.playwright.version",
					"1.56.1",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.playwright.devices.#",
					"1",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.steps.test.command",
					"npx playwright test",
				),
			),
		},
	})
}

func TestAccPlaywrightCheckSuiteAutoDetectFalseMissingVersion(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check auto_detect false missing version"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {
						auto_detect = false

						steps = {
							test = {
								command = "npx playwright test"
							}
						}

						playwright = {
							devices = [
								{
									type = "chromium"
								},
							]
						}
					}
				}
			`,
			ExpectError: regexp.MustCompile(`"runtime.playwright.version" is required when "runtime.auto_detect" is false`),
		},
	})
}

// An explicitly empty devices set can't be planned as the default devices,
// so validation rejects it with or without auto-detection.
func TestAccPlaywrightCheckSuiteEmptyDevicesRejected(t *testing.T) {
	config := func(autoDetect bool) string {
		return playwrightCheckSuiteBase + fmt.Sprintf(`
			resource "checkly_playwright_check_suite" "test" {
				name                      = "PW Check empty devices"
				activated                 = true
				frequency                 = 720
				use_global_alert_settings = true
				locations                 = ["us-east-1"]

				bundle = {
					id       = checkly_playwright_code_bundle.test.id
					metadata = checkly_playwright_code_bundle.test.metadata
				}

				runtime = {
					auto_detect = %t

					steps = {
						test = {
							command = "npx playwright test"
						}
					}

					playwright = {
						version = "1.56.1"
						devices = []
					}
				}
			}
		`, autoDetect)
	}
	wantError := regexp.MustCompile(`runtime\.playwright\.devices\s+set\s+must\s+contain\s+at\s+least\s+1\s+elements`)

	accTestCase(t, []resource.TestStep{
		{Config: config(true), ExpectError: wantError},
		{Config: config(false), ExpectError: wantError},
	})
}

// With auto_detect = false and no devices configured, the suite gets the
// default devices, as with auto-detection, and plans no changes after the
// apply.
func TestAccPlaywrightCheckSuiteAutoDetectFalseDefaultDevices(t *testing.T) {
	config := playwrightCheckSuiteBase + `
		resource "checkly_playwright_check_suite" "test" {
			name                      = "PW Check auto_detect false default devices"
			activated                 = true
			frequency                 = 720
			use_global_alert_settings = true
			locations                 = ["us-east-1"]

			bundle = {
				id       = checkly_playwright_code_bundle.test.id
				metadata = checkly_playwright_code_bundle.test.metadata
			}

			runtime = {
				auto_detect = false

				steps = {
					test = {
						command = "npx playwright test"
					}
				}

				playwright = {
					version = "1.56.1"
				}
			}
		}
	`

	accTestCase(t, []resource.TestStep{
		{
			Config: config,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue(
					"checkly_playwright_check_suite.test",
					tfjsonpath.New("runtime").AtMapKey("playwright").AtMapKey("devices"),
					knownvalue.SetExact([]knownvalue.Check{
						knownvalue.ObjectExact(map[string]knownvalue.Check{"type": knownvalue.StringExact("chromium")}),
						knownvalue.ObjectExact(map[string]knownvalue.Check{"type": knownvalue.StringExact("firefox")}),
						knownvalue.ObjectExact(map[string]knownvalue.Check{"type": knownvalue.StringExact("webkit")}),
					}),
				),
			},
		},
		{
			// The defaults resolve the same way on every plan.
			Config:   config,
			PlanOnly: true,
		},
	})
}

func TestAccPlaywrightCheckSuiteAutoDetectFalseMissingTestCommand(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check auto_detect false missing test command"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {
						auto_detect = false

						playwright = {
							version = "1.56.1"
							devices = [
								{
									type = "chromium"
								},
							]
						}
					}
				}
			`,
			ExpectError: regexp.MustCompile(`"runtime.steps.test.command" is required when "runtime.auto_detect" is false`),
		},
	})
}

func TestAccPlaywrightCheckSuiteWorkingDirAutoDetect(t *testing.T) {
	monorepoBundle := `
		resource "checkly_playwright_code_bundle" "test" {
			prebuilt_archive = {
				file = "../../fixtures/playwright-project-monorepo-pnpm.tar.gz"
			}
		}
	`

	accTestCase(t, []resource.TestStep{
		// Step 1: Auto-detect working_dir from a monorepo archive.
		{
			Config: monorepoBundle + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check working dir auto"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.working_dir",
					"packages/e2e",
				),
			),
		},
		// Step 2: Explicit working_dir overrides auto-detection.
		{
			Config: monorepoBundle + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check working dir auto"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {
						working_dir = "custom/dir"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.working_dir",
					"custom/dir",
				),
			),
		},
		// Step 3: Remove explicit working_dir — falls back to auto-detected.
		{
			Config: monorepoBundle + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check working dir auto"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.working_dir",
					"packages/e2e",
				),
			),
		},
	})
}

func TestAccPlaywrightCheckSuiteWorkingDirAutoDetectFlat(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		// Flat project — auto-detected working_dir should be ".".
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check working dir flat"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.working_dir",
					".",
				),
			),
		},
	})
}

func TestAccPlaywrightCheckSuiteWorkingDirEmptyStringRejected(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check working dir empty"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {
						working_dir = ""
					}
				}
			`,
			ExpectError: regexp.MustCompile(`must not be empty`),
		},
	})
}

func TestAccPlaywrightCheckSuiteWorkingDir(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		// Step 1: Set an explicit working directory.
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check working dir"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {
						working_dir = "packages/e2e"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.working_dir",
					"packages/e2e",
				),
			),
		},
		// Step 2: Remove working_dir but keep the runtime attribute — falls back to auto-detect (".").
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check working dir"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.working_dir",
					".",
				),
			),
		},
		// Step 3: Set it again so we can test removal via dropping the runtime attribute.
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check working dir"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {
						working_dir = "apps/web"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.working_dir",
					"apps/web",
				),
			),
		},
		// Step 4: Remove the entire runtime attribute — falls back to auto-detect (".").
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check working dir"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.working_dir",
					".",
				),
			),
		},
	})
}

func TestAccPlaywrightCheckSuiteEnvironmentVariableRemoval(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check env var removal"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {
						steps = {
							install = {
								command = "pnpm i"
							}

							test = {
								command = "pnpm playwright test"
							}
						}

						playwright = {
							version = "1.56.1"
							devices = [
								{
									type = "chromium"
								},
							]
						}
					}

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
					"checkly_playwright_check_suite.test",
					"environment_variables.#",
					"1",
				),
			),
		},
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check env var removal"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {
						steps = {
							install = {
								command = "pnpm i"
							}

							test = {
								command = "pnpm playwright test"
							}
						}

						playwright = {
							version = "1.56.1"
							devices = [
								{
									type = "chromium"
								},
							]
						}
					}
				}
			`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue("checkly_playwright_check_suite.test", tfjsonpath.New("environment_variables"), emptyList),
			},
		},
	})
}

func TestAccPlaywrightCheckSuiteWithEngine(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check with engine"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {
						engine = {
							name    = "node"
							version = "24"
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.engine.name",
					"node",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.engine.version",
					"24",
				),
			),
		},
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check with engine"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {
						engine = {
							name    = "bun"
							version = "1.3"
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.engine.name",
					"bun",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.engine.version",
					"1.3",
				),
			),
		},
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check with engine"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				// The engine attribute is null when no engine is configured
				// or detected from the code bundle.
				resource.TestCheckNoResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.engine.name",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.engine.version",
				),
			),
		},
	})
}

func TestAccPlaywrightCheckSuiteEngineVersionValidation(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check engine validation"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {
						engine = {
							name    = "node"
							version = "18"
						}
					}
				}
			`,
			ExpectError: regexp.MustCompile(`"runtime.engine.version" is not valid`),
		},
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check engine validation"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {
						engine = {
							name    = "bun"
							version = "2.0"
						}
					}
				}
			`,
			ExpectError: regexp.MustCompile(`"runtime.engine.version" is not valid`),
		},
	})
}

// Leaving devices out resolves to the default devices whatever auto_detect
// is, and validation rejects an explicitly empty devices set.
func TestPlaywrightCheckSuiteResolveRuntimeDevices(t *testing.T) {
	ctx := context.Background()

	runtime := func(autoDetect bool, devices types.Set) types.Object {
		return types.ObjectValueMust(playwrightCheckSuiteRuntimeAttrTypes, map[string]attr.Value{
			"auto_detect": types.BoolValue(autoDetect),
			"working_dir": types.StringNull(),
			"steps": types.ObjectValueMust(playwrightCheckSuiteRuntimeStepsAttrTypes, map[string]attr.Value{
				"install": types.ObjectNull(playwrightCheckSuiteRuntimeCommandAttrTypes),
				"test": types.ObjectValueMust(playwrightCheckSuiteRuntimeCommandAttrTypes, map[string]attr.Value{
					"command": types.StringValue("npx playwright test"),
				}),
			}),
			"playwright": types.ObjectValueMust(playwrightCheckSuiteRuntimePlaywrightAttrTypes, map[string]attr.Value{
				"version": types.StringValue("1.56.1"),
				"devices": devices,
			}),
			"engine": types.ObjectNull(playwrightCheckSuiteRuntimeEngineAttrTypes),
		})
	}

	var diags diag.Diagnostics
	defaults := playwrightCheckSuiteDeviceSet(ctx, playwrightCheckSuiteDefaultBrowsers, &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}

	for _, autoDetect := range []bool{true, false} {
		t.Run(fmt.Sprintf("auto_detect=%v", autoDetect), func(t *testing.T) {
			var diags diag.Diagnostics
			got := playwrightCheckSuiteResolveRuntime(ctx, runtime(autoDetect, types.SetNull(playwrightCheckSuiteRuntimeDeviceElementType)), &PlaywrightCodeBundleMetadata{}, &diags)
			if diags.HasError() {
				t.Fatal(diags)
			}
			devices := got.Attributes()["playwright"].(types.Object).Attributes()["devices"]
			if !devices.Equal(defaults) {
				t.Errorf("devices = %s, want %s", devices, defaults)
			}
		})
	}

	// A devices value that was unknown during validation can still resolve
	// to an empty set; the resolver rejects it too.
	t.Run("empty in resolver", func(t *testing.T) {
		var diags diag.Diagnostics
		empty := types.SetValueMust(playwrightCheckSuiteRuntimeDeviceElementType, []attr.Value{})
		playwrightCheckSuiteResolveRuntime(ctx, runtime(false, empty), &PlaywrightCodeBundleMetadata{}, &diags)
		if !diags.HasError() {
			t.Error("resolving an empty devices set succeeded, want an error")
		}
	})

	t.Run("empty", func(t *testing.T) {
		var resp fwresource.SchemaResponse
		NewPlaywrightCheckSuiteResource().Schema(ctx, fwresource.SchemaRequest{}, &resp)
		runtimeAttr := resp.Schema.Attributes["runtime"].(schema.SingleNestedAttribute)
		playwrightAttr := runtimeAttr.Attributes["playwright"].(schema.SingleNestedAttribute)
		devicesAttr := playwrightAttr.Attributes["devices"].(schema.SetNestedAttribute)

		req := validator.SetRequest{
			Path:        path.Root("runtime").AtName("playwright").AtName("devices"),
			ConfigValue: types.SetValueMust(playwrightCheckSuiteRuntimeDeviceElementType, []attr.Value{}),
		}
		var vresp validator.SetResponse
		for _, v := range devicesAttr.Validators {
			v.ValidateSet(ctx, req, &vresp)
		}
		if !vresp.Diagnostics.HasError() {
			t.Error("expected a validation error for an empty devices set")
		}
	})
}

func TestAccPlaywrightCheckSuiteRuntimeValidation(t *testing.T) {
	config := func(runtime string) string {
		return playwrightCheckSuiteBase + fmt.Sprintf(`
			resource "checkly_playwright_check_suite" "test" {
				name      = "PW Check runtime validation"
				activated = false
				frequency = 720
				locations = ["us-east-1"]

				bundle = {
					id       = checkly_playwright_code_bundle.test.id
					metadata = checkly_playwright_code_bundle.test.metadata
				}

				runtime = %s
			}
		`, runtime)
	}

	accTestCase(t, []resource.TestStep{
		{
			Config:      config(`{ playwright = { devices = [{ type = "chrome-beta" }] } }`),
			ExpectError: regexp.MustCompile(`value must be one of`),
		},
		{
			Config:      config(`{ steps = { install = { command = "" } } }`),
			ExpectError: regexp.MustCompile(`runtime.steps.install.command string length must be at least 1`),
		},
		{
			Config:      config(`{ steps = { test = { command = "" } } }`),
			ExpectError: regexp.MustCompile(`runtime.steps.test.command string length must be at least 1`),
		},
	})
}
