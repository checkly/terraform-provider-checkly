package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccStatusPageServiceCheckRequiredFields(t *testing.T) {
	config := `resource "checkly_status_page_service" "test" {}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "name" is required`),
		},
	})
}

func TestAccStatusPageServiceHappyPath(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_status_page_service" "test" {
					name = "foo"
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_status_page_service.test",
					"name",
					"foo",
				),
			),
		},
		{
			ResourceName:      "checkly_status_page_service.test",
			ImportState:       true,
			ImportStateVerify: true,
		},
	})
}
