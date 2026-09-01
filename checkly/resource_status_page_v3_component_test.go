package checkly

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// statusPageV3CompositeImportID builds the "<status_page_id>/<id>" import ID
// for nested v3 status page resources from the resource's state.
func statusPageV3CompositeImportID(resourceName string) resource.ImportStateIdFunc {
	return func(state *terraform.State) (string, error) {
		rs, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource %q not found in state", resourceName)
		}
		return rs.Primary.Attributes["status_page_id"] + "/" + rs.Primary.ID, nil
	}
}

func TestAccStatusPageV3ComponentCheckRequiredFields(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config:      `resource "checkly_status_page_v3_component" "test" {}`,
			ExpectError: regexp.MustCompile(`The argument "status_page_id" is required`),
		},
		{
			Config: `
				resource "checkly_status_page_v3_component" "test" {
					status_page_id = "6a11caa1-b6cb-4e46-8bd3-e70a90c5add8"
					name           = "foo"
				}
			`,
			ExpectError: regexp.MustCompile(`The argument "display_order" is required`),
		},
	})
}

func TestAccStatusPageV3ComponentHappyPath(t *testing.T) {
	rInt := acctest.RandInt()
	page := fmt.Sprintf(`
		resource "checkly_status_page_v3" "test" {
			name = "foo"
			url  = "status-page-v3-comp-%d"
		}

		resource "checkly_status_page_v3_component" "group" {
			status_page_id = checkly_status_page_v3.test.id
			type           = "GROUP"
			name           = "Foo group"
			display_order  = 0
		}
	`, rInt)
	accTestCase(t, []resource.TestStep{
		{
			Config: page + `
				resource "checkly_status_page_v3_component" "api" {
					status_page_id = checkly_status_page_v3.test.id
					name           = "Foo API"
					description    = "The public API"
					display_order  = 1
					parent_id      = checkly_status_page_v3_component.group.id
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_status_page_v3_component.group",
					"type",
					"GROUP",
				),
				resource.TestCheckResourceAttr(
					"checkly_status_page_v3_component.api",
					"type",
					"SERVICE",
				),
				resource.TestCheckResourceAttr(
					"checkly_status_page_v3_component.api",
					"hidden",
					"false",
				),
				resource.TestCheckResourceAttrPair(
					"checkly_status_page_v3_component.api",
					"status_page_id",
					"checkly_status_page_v3.test",
					"id",
				),
				resource.TestCheckResourceAttrPair(
					"checkly_status_page_v3_component.api",
					"parent_id",
					"checkly_status_page_v3_component.group",
					"id",
				),
			),
		},
		{
			Config: page + `
				resource "checkly_status_page_v3_component" "api" {
					status_page_id = checkly_status_page_v3.test.id
					name           = "Bar API"
					display_order  = 2
					hidden         = true
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_status_page_v3_component.api",
					"name",
					"Bar API",
				),
				resource.TestCheckResourceAttr(
					"checkly_status_page_v3_component.api",
					"display_order",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_status_page_v3_component.api",
					"hidden",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_status_page_v3_component.api",
					"parent_id",
					"",
				),
			),
		},
		{
			ResourceName:      "checkly_status_page_v3_component.api",
			ImportState:       true,
			ImportStateVerify: true,
			// The import ID is composite: "<status_page_id>/<component_id>".
			ImportStateIdFunc: statusPageV3CompositeImportID("checkly_status_page_v3_component.api"),
		},
	})
}
