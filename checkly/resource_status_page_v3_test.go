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
					"allow_indexing",
					"false",
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
