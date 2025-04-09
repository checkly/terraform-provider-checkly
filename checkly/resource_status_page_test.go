package checkly

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
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
			Config:      config,
			ExpectError: regexp.MustCompile(`At least 1 "card" blocks are required`),
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
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_status_page_service" "test" {
					name = "qux"
				}

				resource "checkly_status_page" "test" {
					name = "foo"
					url  = "bar"

					card {
						name = "baz"

						service_attachment {
							service_id = checkly_status_page_service.test.id
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_status_page.test",
					"name",
					"foo",
				),
				resource.TestCheckResourceAttr(
					"checkly_status_page.test",
					"url",
					"bar",
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
			Config: `
				resource "checkly_status_page_service" "test" {
					name = "qux"
				}

				resource "checkly_status_page" "test" {
					name          = "foo"
					url           = "bar"
					custom_domain = "my-example-status-page-248234834.checklyhq.com"
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
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_status_page.test",
					"name",
					"foo",
				),
				resource.TestCheckResourceAttr(
					"checkly_status_page.test",
					"url",
					"bar",
				),
				resource.TestCheckResourceAttr(
					"checkly_status_page.test",
					"custom_domain",
					"my-example-status-page-248234834.checklyhq.com",
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
	})
}

func TestAccStatusPageUnsupportedCustomDomains(t *testing.T) {
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
					url           = "bar"
					custom_domain = "%s"

					card {
						name = "baz"

						service_attachment {
							service_id = checkly_status_page_service.test.id
						}
					}
				}
			`, domain),

			ExpectError: regexp.MustCompile(`custom domains ending in .+ are not supported`),
		})
	}

	accTestCase(t, steps)
}
