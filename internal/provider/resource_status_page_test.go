package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccStatusPageCheckRequiredFields(t *testing.T) {
	config := `resource "checkly_status_page" "test" {}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "name" is required`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "url" is required`),
		},
		{
			// name and url must be present for validation to proceed far
			// enough to surface the missing-card error.
			Config: `
				resource "checkly_status_page" "test" {
					name = "foo"
					url  = "foo"
				}
			`,
			ExpectError: regexp.MustCompile(`Block card must have a configuration value`),
		},
	})
}

func TestAccStatusPageCardServiceAttachmentCheckRequiredFields(t *testing.T) {
	config := `
		resource "checkly_status_page" "test" {
			name = "foo"
			url  = "foo"

			card {
				name = "foo"

				service_attachment {
				}
			}
		}
	`

	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "service_id" is required`),
		},
	})
}

func TestAccStatusPageHappyPath(t *testing.T) {
	rInt := acctest.RandInt()
	accTestCase(t, []resource.TestStep{
		{
			Config: fmt.Sprintf(`
				resource "checkly_status_page_service" "test" {
					name = "qux"
				}

				resource "checkly_status_page" "test" {
					name = "foo"
					url  = "status-page-%d"

					card {
						name = "baz"

						service_attachment {
							service_id = checkly_status_page_service.test.id
						}
					}
				}
			`, rInt),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_status_page.test",
					"name",
					"foo",
				),
				resource.TestCheckResourceAttr(
					"checkly_status_page.test",
					"url",
					fmt.Sprintf("status-page-%d", rInt),
				),
				resource.TestCheckResourceAttr(
					"checkly_status_page.test",
					"card.#",
					"1",
				),
				resource.TestCheckResourceAttr(
					"checkly_status_page.test",
					"card.0.name",
					"baz",
				),
				resource.TestCheckResourceAttr(
					"checkly_status_page.test",
					"card.0.service_attachment.#",
					"1",
				),
				resource.TestCheckResourceAttrPair(
					"checkly_status_page.test",
					"card.0.service_attachment.0.service_id",
					"checkly_status_page_service.test",
					"id",
				),
			),
		},
		{
			Config: fmt.Sprintf(`
				resource "checkly_status_page_service" "test" {
					name = "qux"
				}

				resource "checkly_status_page" "test" {
					name          = "foo"
					url           = "status-page-%d"
					custom_domain = "status-page-%d.checklyhq.com"
					logo          = "https://example.org/logo.png"
					redirect_to   = "https://example.org"
					favicon       = "https://example.org/favicon.png"
					default_theme = "DARK"

					card {
						name = "baz"

						service_attachment {
							service_id = checkly_status_page_service.test.id
						}
					}
				}
			`, rInt, rInt),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_status_page.test",
					"name",
					"foo",
				),
				resource.TestCheckResourceAttr(
					"checkly_status_page.test",
					"url",
					fmt.Sprintf("status-page-%d", rInt),
				),
				resource.TestCheckResourceAttr(
					"checkly_status_page.test",
					"custom_domain",
					fmt.Sprintf("status-page-%d.checklyhq.com", rInt),
				),
				resource.TestCheckResourceAttr(
					"checkly_status_page.test",
					"logo",
					"https://example.org/logo.png",
				),
				resource.TestCheckResourceAttr(
					"checkly_status_page.test",
					"redirect_to",
					"https://example.org",
				),
				resource.TestCheckResourceAttr(
					"checkly_status_page.test",
					"favicon",
					"https://example.org/favicon.png",
				),
				resource.TestCheckResourceAttr(
					"checkly_status_page.test",
					"default_theme",
					"DARK",
				),
				resource.TestCheckResourceAttr(
					"checkly_status_page.test",
					"card.#",
					"1",
				),
				resource.TestCheckResourceAttr(
					"checkly_status_page.test",
					"card.0.name",
					"baz",
				),
				resource.TestCheckResourceAttr(
					"checkly_status_page.test",
					"card.0.service_attachment.#",
					"1",
				),
				resource.TestCheckResourceAttrPair(
					"checkly_status_page.test",
					"card.0.service_attachment.0.service_id",
					"checkly_status_page_service.test",
					"id",
				),
			),
		},
		{
			ResourceName:      "checkly_status_page.test",
			ImportState:       true,
			ImportStateVerify: true,
		},
	})
}

func TestAccStatusPageUnsupportedCustomDomains(t *testing.T) {
	rInt := acctest.RandInt()
	badDomains := []string{
		"example.com",
		"example.net",
		"example.org",
		"status.example.com",
		"status.example.net",
		"status.example.org",
	}

	var steps []resource.TestStep

	for _, domain := range badDomains {
		steps = append(steps, resource.TestStep{
			Config: fmt.Sprintf(`
				resource "checkly_status_page_service" "test" {
					name = "qux"
				}

				resource "checkly_status_page" "test" {
					name          = "foo"
					url           = "status-page-%d"
					custom_domain = "%s"

					card {
						name = "baz"

						service_attachment {
							service_id = checkly_status_page_service.test.id
						}
					}
				}
			`, rInt, domain),

			ExpectError: regexp.MustCompile(`custom domains ending in .+ are not supported`),
		})
	}

	accTestCase(t, steps)
}

// TestAccStatusPageUpdateClearsRemovedValues removes values that the API keeps
// when an update leaves them out. The empty plan after the last step shows
// that they are gone.
func TestAccStatusPageUpdateClearsRemovedValues(t *testing.T) {
	rInt := acctest.RandInt()
	config := func(values string) string {
		return fmt.Sprintf(`
			resource "checkly_status_page_service" "test" {
				name = "tf-acc-clear"
			}

			resource "checkly_status_page" "test" {
				name = "tf-acc-clear"
				url  = "status-page-clear-%d"
				%s

				card {
					name = "baz"

					service_attachment {
						service_id = checkly_status_page_service.test.id
					}
				}
			}
		`, rInt, values)
	}
	const r = "checkly_status_page.test"

	accTestCase(t, []resource.TestStep{
		{
			Config: config(fmt.Sprintf(`
				custom_domain = "status-page-clear-%d.checklyhq.com"
				logo          = "https://example.com/logo.png"
				redirect_to   = "https://example.com"
				favicon       = "https://example.com/favicon.ico"
			`, rInt)),
			Check: resource.TestCheckResourceAttr(r, "logo", "https://example.com/logo.png"),
		},
		{
			Config: config(""),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(r, "custom_domain", ""),
				resource.TestCheckResourceAttr(r, "logo", ""),
				resource.TestCheckResourceAttr(r, "redirect_to", ""),
				resource.TestCheckResourceAttr(r, "favicon", ""),
			),
		},
	})
}
