package checkly

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccMaintenanceWindowInvalidTimezone(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_maintenance_windows" "test" {
					name      = "foo"
					starts_at = "2014-08-24T00:00:00.000Z"
					ends_at   = "2014-08-25T00:00:00.000Z"
					timezone  = "+05:00"
				}
			`,
			ExpectError: regexp.MustCompile(`must be a valid IANA time zone`),
		},
	})
}

func TestAccMaintenanceWindowScopes(t *testing.T) {
	rInt := acctest.RandInt()
	accTestCase(t, []resource.TestStep{
		{
			Config: fmt.Sprintf(`
				resource "checkly_maintenance_windows" "test" {
					name           = "maintenance-%d"
					starts_at      = "2014-08-24T00:00:00.000Z"
					ends_at        = "2014-08-25T00:00:00.000Z"
					timezone       = "America/New_York"
					tags           = ["production"]

					silence_alerts_tags = ["production", "staging"]
				}
			`, rInt),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_maintenance_windows.test",
					"timezone",
					"America/New_York",
				),
				// Both booleans default to false and must round-trip as false
				// rather than being dropped from the payload.
				resource.TestCheckResourceAttr(
					"checkly_maintenance_windows.test",
					"pause_all_checks",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_maintenance_windows.test",
					"silence_all_alerts",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_maintenance_windows.test",
					"silence_alerts_tags.#",
					"2",
				),
			),
		},
		{
			// Account-wide scope: tags become irrelevant on both sides.
			Config: fmt.Sprintf(`
				resource "checkly_maintenance_windows" "test" {
					name               = "maintenance-%d"
					starts_at          = "2014-08-24T00:00:00.000Z"
					ends_at            = "2014-08-25T00:00:00.000Z"
					timezone           = "Europe/Berlin"
					pause_all_checks   = true
					silence_all_alerts = true
				}
			`, rInt),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_maintenance_windows.test",
					"timezone",
					"Europe/Berlin",
				),
				resource.TestCheckResourceAttr(
					"checkly_maintenance_windows.test",
					"pause_all_checks",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_maintenance_windows.test",
					"silence_all_alerts",
					"true",
				),
			),
		},
	})
}
