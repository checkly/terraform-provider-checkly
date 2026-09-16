package checkly

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const statusPageV3Resource = "checkly_status_page_v3.test"

func TestAccStatusPageV3CheckRequiredFields(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config:      `resource "checkly_status_page_v3" "test" {}`,
			ExpectError: regexp.MustCompile(`The argument "name" is required`),
		},
		{
			Config:      `resource "checkly_status_page_v3" "test" { name = "foo" }`,
			ExpectError: regexp.MustCompile(`The argument "url" is required`),
		},
	})
}

func TestAccStatusPageV3URLValidation(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_status_page_v3" "test" {
					name = "foo"
					url  = "Not A Slug"
				}
			`,
			ExpectError: regexp.MustCompile(`can only contain lowercase alphanumeric characters and dashes`),
		},
	})
}

// Kept apart from the happy path: custom theme colors are a separately
// entitled feature, so this only passes on an account whose plan has it.
func TestAccStatusPageV3ThemeColors(t *testing.T) {
	rInt := acctest.RandInt()
	accTestCase(t, []resource.TestStep{
		{
			Config: fmt.Sprintf(`
				resource "checkly_status_page_v3" "test" {
					name = "themed"
					url  = "status-page-v3-themed-%d"
					theme_colors {
						light {
							link_font_color = "not-a-color"
						}
					}
				}
			`, rInt),
			ExpectError: regexp.MustCompile(`must be a hex color`),
		},
		{
			Config: fmt.Sprintf(`
				resource "checkly_status_page_v3" "test" {
					name = "themed"
					url  = "status-page-v3-themed-%d"
					theme_colors {
						light {
							body_background_color           = "#F9FAFB"
							header_background_color         = "#FFFFFF"
							header_font_color               = "#151A1E"
							title_font_color                = "#212930"
							body_font_color                 = "#475766"
							body_font_color_muted           = "#60758A"
							navigation_font_color           = "#151A1E"
							link_font_color                 = "#FF0000"
							card_background_color           = "#FFFFFF"
							border_color                    = "#E0E5EB"
							primary_button_background_color = "#151A1E"
							primary_button_font_color       = "#FFFFFF"
						}
						dark {
							body_background_color           = "#14171C"
							header_background_color         = "#171B21"
							header_font_color               = "#FFFFFF"
							title_font_color                = "#ECEEF2"
							body_font_color                 = "#C6CDD7"
							body_font_color_muted           = "#A3B3C2"
							navigation_font_color           = "#FFFFFF"
							link_font_color                 = "#248AFF"
							card_background_color           = "#171B21"
							border_color                    = "#242B34"
							primary_button_background_color = "#242B34"
							primary_button_font_color       = "#FFFFFF"
						}
					}
				}
			`, rInt),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					statusPageV3Resource,
					"theme_colors.0.light.0.link_font_color",
					"#FF0000",
				),
				resource.TestCheckResourceAttr(
					statusPageV3Resource,
					"theme_colors.0.dark.0.link_font_color",
					"#248AFF",
				),
			),
		},
		{
			// Removing the block clears the custom colors remotely; the
			// default palette the API then reports must not show up as a
			// diff in this step's post-apply plan.
			Config: fmt.Sprintf(`
				resource "checkly_status_page_v3" "test" {
					name = "themed"
					url  = "status-page-v3-themed-%d"
				}
			`, rInt),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					statusPageV3Resource,
					"theme_colors.#",
					"0",
				),
			),
		},
	})
}

func TestAccStatusPageV3HappyPath(t *testing.T) {
	rInt := acctest.RandInt()
	accTestCase(t, []resource.TestStep{
		{
			Config: fmt.Sprintf(`
				resource "checkly_status_page_v3" "test" {
					name = "foo"
					url  = "status-page-v3-%d"
				}
			`, rInt),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					statusPageV3Resource,
					"name",
					"foo",
				),
				resource.TestCheckResourceAttr(
					statusPageV3Resource,
					"url",
					fmt.Sprintf("status-page-v3-%d", rInt),
				),
				resource.TestCheckResourceAttr(
					statusPageV3Resource,
					"default_theme",
					"AUTO",
				),
				resource.TestCheckResourceAttr(
					statusPageV3Resource,
					"allow_indexing",
					"true",
				),
			),
		},
		{
			Config: fmt.Sprintf(`
				resource "checkly_status_page_v3" "test" {
					name                  = "bar"
					url                   = "status-page-v3-%d"
					description           = "All systems"
					logo                  = "https://example.org/logo.png"
					logo_dark             = "https://example.org/logo-dark.png"
					redirect_to           = "https://example.org"
					favicon               = "https://example.org/favicon.png"
					default_theme         = "DARK"
					privacy_policy_link   = "https://example.org/privacy"
					terms_of_service_link = "https://example.org/terms"
					support_link          = "mailto:support@example.org"
					footer_text           = "Example Inc."
					google_analytics_tag  = "G-XXXXXXXXXX"
					allow_indexing        = false
				}
			`, rInt),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					statusPageV3Resource,
					"name",
					"bar",
				),
				resource.TestCheckResourceAttr(
					statusPageV3Resource,
					"description",
					"All systems",
				),
				resource.TestCheckResourceAttr(
					statusPageV3Resource,
					"default_theme",
					"DARK",
				),
				resource.TestCheckResourceAttr(
					statusPageV3Resource,
					"footer_text",
					"Example Inc.",
				),
				resource.TestCheckResourceAttr(
					statusPageV3Resource,
					"support_link",
					"mailto:support@example.org",
				),
				resource.TestCheckResourceAttr(
					statusPageV3Resource,
					"allow_indexing",
					"false",
				),
			),
		},
		{
			// Back to the minimal config: every removed optional must be
			// cleared remotely, or this step fails its post-apply plan.
			Config: fmt.Sprintf(`
				resource "checkly_status_page_v3" "test" {
					name = "bar"
					url  = "status-page-v3-%d"
				}
			`, rInt),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					statusPageV3Resource,
					"description",
					"",
				),
				resource.TestCheckResourceAttr(
					statusPageV3Resource,
					"footer_text",
					"",
				),
				resource.TestCheckResourceAttr(
					statusPageV3Resource,
					"support_link",
					"",
				),
				resource.TestCheckResourceAttr(
					statusPageV3Resource,
					"logo",
					"",
				),
				resource.TestCheckResourceAttr(
					statusPageV3Resource,
					"google_analytics_tag",
					"",
				),
				resource.TestCheckResourceAttr(
					statusPageV3Resource,
					"allow_indexing",
					"true",
				),
			),
		},
		{
			ResourceName:      statusPageV3Resource,
			ImportState:       true,
			ImportStateVerify: true,
		},
	})
}
