package checkly

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const playwrightCheckSuiteBase = `
	resource "checkly_playwright_code_bundle" "test" {
		prebuilt_archive {
			file = "../fixtures/playwright-project.tar.gz"
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
			ExpectError: regexp.MustCompile(`browsers.+does not contain 1 required value`),
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
