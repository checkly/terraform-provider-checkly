package checkly

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const playwrightCheckSuiteBase = `
	resource "checkly_playwright_code_bundle" "test" {
		prebuilt_archive {
			file = "../fixtures/playwright-project-pnpm.tar.gz"
		}
	}
`

func TestAccPlaywrightCheckSuiteWithEnvironmentVariable(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check with env vars"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime {
						steps {
							install {
								command = "pnpm i"
							}

							test {
								command = "pnpm playwright test"
							}
						}

						playwright {
							version = "1.56.1"
							device {
								type = "chromium"
							}
						}
					}

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
					"checkly_playwright_check_suite.test",
					"environment_variable.#",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"environment_variable.0.key",
					"FOO",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"environment_variable.0.value",
					"bar",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"environment_variable.0.locked",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"environment_variable.1.key",
					"SECRET",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"environment_variable.1.value",
					"s3cr3t",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"environment_variable.1.locked",
					"true",
				),
			),
		},
	})
}

func TestAccPlaywrightCheckSuiteBundleChange(t *testing.T) {
	pnpmBundle := `
		resource "checkly_playwright_code_bundle" "test" {
			prebuilt_archive {
				file = "../fixtures/playwright-project-pnpm.tar.gz"
			}
		}
	`

	pnpmNextBundle := `
		resource "checkly_playwright_code_bundle" "test" {
			prebuilt_archive {
				file = "../fixtures/playwright-project-pnpm-playwright-next.tar.gz"
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

			bundle {
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
					"runtime.0.playwright.0.version",
					"1.58.2",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.steps.0.test.0.command",
					defaultTestCommand["pnpm"],
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.working_dir",
					".",
				),
			),
		},
		// Step 2: Swap to a different archive — auto-detected values should update.
		{
			Config: pnpmNextBundle + checkSuite,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.playwright.0.version",
					"1.59.0-alpha-1774287265000",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.steps.0.test.0.command",
					defaultTestCommand["pnpm"],
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.working_dir",
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.playwright.0.version",
					"1.58.2",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.playwright.0.device.#",
					fmt.Sprintf("%d", len(defaultPlaywrightBrowsers)),
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.working_dir",
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime {
						playwright {
							version = "1.56.1"
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.playwright.0.version",
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime {
						playwright {
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.playwright.0.version",
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime {
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.playwright.0.version",
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.playwright.0.version",
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime {
						playwright {
							version = "1.56.1"
							device {
								type = "chromium"
							}
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.playwright.0.version",
					"1.56.1",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.playwright.0.device.#",
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime {
						playwright {
							device {
								type = "chromium"
							}
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.playwright.0.version",
					"1.58.2",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.playwright.0.device.#",
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime {
						playwright {
							device {
								type = "chromium"
							}
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.playwright.0.device.#",
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime {
						playwright {
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.playwright.0.device.#",
					fmt.Sprintf("%d", len(defaultPlaywrightBrowsers)),
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime {
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.playwright.0.device.#",
					fmt.Sprintf("%d", len(defaultPlaywrightBrowsers)),
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.playwright.0.device.#",
					fmt.Sprintf("%d", len(defaultPlaywrightBrowsers)),
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.steps.0.test.0.command",
					defaultTestCommand["pnpm"],
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime {
						steps {
							test {
								command = "pnpm playwright test --workers=4"
							}
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.steps.0.test.0.command",
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.steps.0.test.0.command",
					defaultTestCommand["pnpm"],
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime {
						steps {
							install {
								command = "pnpm install --frozen-lockfile"
							}
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.steps.0.install.0.command",
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.steps.0.install.0.command",
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime {
						steps {
							install {
								command = "pnpm install --frozen-lockfile"
							}
						}

						playwright {
							version = "1.56.1"
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.steps.0.install.0.command",
					"pnpm install --frozen-lockfile",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.playwright.0.version",
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime {
						steps {
							install {
								command = "pnpm install --frozen-lockfile"
							}
						}

						playwright {
							version = "1.58.2"
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.steps.0.install.0.command",
					"pnpm install --frozen-lockfile",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.playwright.0.version",
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime {
						steps {
							install {
								command = "pnpm install --frozen-lockfile"
							}
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.steps.0.install.0.command",
					"pnpm install --frozen-lockfile",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.playwright.0.version",
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime {
						playwright {
							version = "1.56.1"
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.playwright.0.version",
					"1.56.1",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.playwright.0.device.#",
					fmt.Sprintf("%d", len(defaultPlaywrightBrowsers)),
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime {
						auto_detect = false

						steps {
							test {
								command = "npx playwright test"
							}
						}

						playwright {
							version = "1.56.1"
							device {
								type = "chromium"
							}
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.playwright.0.version",
					"1.56.1",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.playwright.0.device.#",
					"1",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.steps.0.test.0.command",
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime {
						playwright {
							version = "1.56.1"
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.playwright.0.version",
					"1.56.1",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.playwright.0.device.#",
					fmt.Sprintf("%d", len(defaultPlaywrightBrowsers)),
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.steps.0.test.0.command",
					defaultTestCommand["pnpm"],
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.playwright.0.version",
					"1.58.2",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.playwright.0.device.#",
					fmt.Sprintf("%d", len(defaultPlaywrightBrowsers)),
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.steps.0.test.0.command",
					defaultTestCommand["pnpm"],
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime {
						auto_detect = false

						steps {
							test {
								command = "npx playwright test"
							}
						}

						playwright {
							version = "1.56.1"
							device {
								type = "chromium"
							}
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.auto_detect",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.playwright.0.version",
					"1.56.1",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.playwright.0.device.#",
					"1",
				),
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.steps.0.test.0.command",
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime {
						auto_detect = false

						steps {
							test {
								command = "npx playwright test"
							}
						}

						playwright {
							device {
								type = "chromium"
							}
						}
					}
				}
			`,
			ExpectError: regexp.MustCompile(`"runtime.playwright.version" is required when "runtime.auto_detect" is false`),
		},
	})
}

func TestAccPlaywrightCheckSuiteAutoDetectFalseMissingDevices(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check auto_detect false missing devices"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime {
						auto_detect = false

						steps {
							test {
								command = "npx playwright test"
							}
						}

						playwright {
							version = "1.56.1"
						}
					}
				}
			`,
			ExpectError: regexp.MustCompile(`at least one "runtime.playwright.device" block is required when "runtime.auto_detect" is false`),
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime {
						auto_detect = false

						playwright {
							version = "1.56.1"
							device {
								type = "chromium"
							}
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
			prebuilt_archive {
				file = "../fixtures/playwright-project-monorepo-pnpm.tar.gz"
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.working_dir",
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime {
						working_dir = "custom/dir"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.working_dir",
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.working_dir",
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.working_dir",
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime {
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime {
						working_dir = "packages/e2e"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.working_dir",
					"packages/e2e",
				),
			),
		},
		// Step 2: Remove working_dir but keep the runtime block — falls back to auto-detect (".").
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check working dir"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime {
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.working_dir",
					".",
				),
			),
		},
		// Step 3: Set it again so we can test removal via dropping the runtime block.
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check working dir"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime {
						working_dir = "apps/web"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.working_dir",
					"apps/web",
				),
			),
		},
		// Step 4: Remove the entire runtime block — falls back to auto-detect (".").
		{
			Config: playwrightCheckSuiteBase + `
				resource "checkly_playwright_check_suite" "test" {
					name                      = "PW Check working dir"
					activated                 = true
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"runtime.0.working_dir",
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime {
						steps {
							install {
								command = "pnpm i"
							}

							test {
								command = "pnpm playwright test"
							}
						}

						playwright {
							version = "1.56.1"
							device {
								type = "chromium"
							}
						}
					}

					environment_variable {
						key   = "FOO"
						value = "bar"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"environment_variable.#",
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

					bundle {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime {
						steps {
							install {
								command = "pnpm i"
							}

							test {
								command = "pnpm playwright test"
							}
						}

						playwright {
							version = "1.56.1"
							device {
								type = "chromium"
							}
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_playwright_check_suite.test",
					"environment_variable.#",
					"0",
				),
			),
		},
	})
}
