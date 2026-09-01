package checkly

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const statusPageV3AutomationRuleResource = "checkly_status_page_v3_automation_rule.test"

func TestAccStatusPageV3AutomationRuleCheckRequiredFields(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config:      `resource "checkly_status_page_v3_automation_rule" "test" {}`,
			ExpectError: regexp.MustCompile(`The argument "first_update" is required`),
		},
		{
			Config: `
				resource "checkly_status_page_v3_automation_rule" "test" {
					status_page_id = "6a11caa1-b6cb-4e46-8bd3-e70a90c5add8"
					name           = "foo"
					first_update   = "Investigating."
					last_update    = "Resolved."
					tags           = []
				}
			`,
			ExpectError: regexp.MustCompile(`Attribute tags requires 1 item minimum`),
		},
		{
			Config: `
				resource "checkly_status_page_v3_automation_rule" "test" {
					status_page_id = "6a11caa1-b6cb-4e46-8bd3-e70a90c5add8"
					name           = "foo"
					first_update   = "Investigating."
					last_update    = "Resolved."
					tags           = ["foo"]

					component {
						component_id  = "0e4f5a72-6a5c-42a1-9a8a-5f7d38c8a9d1"
						target_impact = "OPERATIONAL"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`"component.0.target_impact" must be one of`),
		},
	})
}

func TestAccStatusPageV3AutomationRuleHappyPath(t *testing.T) {
	rInt := acctest.RandInt()
	base := fmt.Sprintf(`
		resource "checkly_status_page_v3" "test" {
			name = "foo"
			url  = "status-page-v3-rule-%d"
		}

		resource "checkly_status_page_v3_component" "api" {
			status_page_id = checkly_status_page_v3.test.id
			name           = "Foo API"
			display_order  = 0
		}
	`, rInt)
	accTestCase(t, []resource.TestStep{
		{
			Config: base + `
				resource "checkly_status_page_v3_automation_rule" "test" {
					status_page_id = checkly_status_page_v3.test.id
					name           = "Foo API outage"
					first_update   = "We are investigating an issue."
					last_update    = "The issue has been resolved."
					tags           = ["foo-api"]

					component {
						component_id  = checkly_status_page_v3_component.api.id
						target_impact = "MAJOR_OUTAGE"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					statusPageV3AutomationRuleResource,
					"name",
					"Foo API outage",
				),
				resource.TestCheckResourceAttr(
					statusPageV3AutomationRuleResource,
					"enabled",
					"true",
				),
				resource.TestCheckResourceAttr(
					statusPageV3AutomationRuleResource,
					"notify_subscribers",
					"true",
				),
				resource.TestCheckResourceAttr(
					statusPageV3AutomationRuleResource,
					"cool_down_window_minutes",
					"5",
				),
				resource.TestCheckResourceAttr(
					statusPageV3AutomationRuleResource,
					"tags.#",
					"1",
				),
				resource.TestCheckResourceAttr(
					statusPageV3AutomationRuleResource,
					"component.#",
					"1",
				),
				resource.TestCheckResourceAttrPair(
					statusPageV3AutomationRuleResource,
					"status_page_id",
					"checkly_status_page_v3.test",
					"id",
				),
			),
		},
		{
			Config: base + `
				resource "checkly_status_page_v3_automation_rule" "test" {
					status_page_id           = checkly_status_page_v3.test.id
					name                     = "Bar API outage"
					enabled                  = false
					first_update             = "We are investigating an issue."
					last_update              = "The issue has been resolved."
					notify_subscribers       = false
					cool_down_window_minutes = 30
					tags                     = ["foo-api", "bar-api"]

					component {
						component_id  = checkly_status_page_v3_component.api.id
						target_impact = "DEGRADED_PERFORMANCE"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					statusPageV3AutomationRuleResource,
					"name",
					"Bar API outage",
				),
				resource.TestCheckResourceAttr(
					statusPageV3AutomationRuleResource,
					"enabled",
					"false",
				),
				resource.TestCheckResourceAttr(
					statusPageV3AutomationRuleResource,
					"cool_down_window_minutes",
					"30",
				),
				resource.TestCheckResourceAttr(
					statusPageV3AutomationRuleResource,
					"tags.#",
					"2",
				),
			),
		},
		{
			ResourceName:      statusPageV3AutomationRuleResource,
			ImportState:       true,
			ImportStateVerify: true,
			// The import ID is composite: "<status_page_id>/<rule_id>".
			ImportStateIdFunc: statusPageV3CompositeImportID(statusPageV3AutomationRuleResource),
		},
	})
}
