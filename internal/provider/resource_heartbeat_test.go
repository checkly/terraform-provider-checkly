package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccHeartbeatRequiredFields(t *testing.T) {
	config := `resource "checkly_heartbeat" "test" {}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "name" is required, but no definition was found.`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "activated" is required, but no definition was found.`),
		},
	})
}

func TestAccHeartbeatInvalidInputs(t *testing.T) {
	config := `resource "checkly_heartbeat" "test" {
		name                      = 1
		activated                 = "invalid"
		use_global_alert_settings = "invalid"
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`Inappropriate value for attribute "activated"`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`Inappropriate value for attribute "use_global_alert_settings"`),
		},
	})
}

func TestAccHeartbeatMissingHeartbeat(t *testing.T) {
	config := `resource "checkly_heartbeat" "test" {
		activated = true
		name = "heartbeat monitor"
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "heartbeat" is required, but no definition was found.`),
		},
	})
}

func TestAccHeartbeatMissingHeartbeatFields(t *testing.T) {
	config := `resource "checkly_heartbeat" "test" {
		activated = true
		name = "heartbeat monitor"
		heartbeat = {

		}
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`attributes "grace",\s+"grace_unit", "period", and "period_unit" are required`),
		},
	})
}

func TestAccHeartbeatPeriodTooBig(t *testing.T) {
	config := `resource "checkly_heartbeat" "test" {
		activated = true
		name = "heartbeat monitor"
		heartbeat = {
			period = 366
			period_unit = "days"
			grace = 0
			grace_unit = "seconds"
		}
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`period must be between 30 seconds and 365 days`),
		},
	})
}

func TestAccHeartbeatPeriodTooSmall(t *testing.T) {
	config := `resource "checkly_heartbeat" "test" {
		activated = true
		name = "heartbeat monitor"
		heartbeat = {
			period = 5
			period_unit = "seconds"
			grace = 0
			grace_unit = "seconds"
		}
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`period must be between 30 seconds and 365 days`),
		},
	})
}

func TestAccHeartbeatInvalidPeriodUnit(t *testing.T) {
	config := `resource "checkly_heartbeat" "test" {
		activated = true
		name = "heartbeat monitor"
		heartbeat = {
			period = 5
			period_unit = "lightyear"
			grace = 0
			grace_unit = "seconds"
		}
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`Attribute heartbeat\.period_unit value must be one of`),
		},
	})
}

func TestAccHeartbeatInvalidGraceUnit(t *testing.T) {
	config := `resource "checkly_heartbeat" "test" {
		activated = true
		name = "heartbeat monitor"
		heartbeat = {
			period = 5
			period_unit = "days"
			grace = 0
			grace_unit = "lightyear"
		}
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`Attribute heartbeat\.grace_unit value must be one of`),
		},
	})
}

func TestAccHeartbeatCreate(t *testing.T) {
	config := `resource "checkly_heartbeat" "test" {
		activated = true
		name = "heartbeat monitor"
		heartbeat = {
			period = 5
			period_unit = "days"
			grace = 0
			grace_unit = "seconds"
		}
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config: config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_heartbeat.test",
					"name",
					"heartbeat monitor",
				),
				resource.TestCheckResourceAttr(
					"checkly_heartbeat.test",
					"heartbeat.period",
					"5",
				),
				resource.TestCheckResourceAttr(
					"checkly_heartbeat.test",
					"heartbeat.period_unit",
					"days",
				),
			),
		},
	})
}

// TestAccHeartbeatMovedToHeartbeatMonitor exercises the MoveState
// implementation of checkly_heartbeat_monitor via a `moved` block from
// checkly_heartbeat.
func TestAccHeartbeatMovedToHeartbeatMonitor(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `resource "checkly_heartbeat" "test" {
				activated = true
				name = "heartbeat monitor"
				heartbeat = {
					period = 5
					period_unit = "days"
					grace = 0
					grace_unit = "seconds"
				}
			}`,
		},
		{
			Config: `
				resource "checkly_heartbeat_monitor" "test" {
					activated = true
					name = "heartbeat monitor"
					heartbeat = {
						period = 5
						period_unit = "days"
						grace = 0
						grace_unit = "seconds"
					}
				}

				moved {
					from = checkly_heartbeat.test
					to   = checkly_heartbeat_monitor.test
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_heartbeat_monitor.test",
					"name",
					"heartbeat monitor",
				),
				resource.TestCheckResourceAttr(
					"checkly_heartbeat_monitor.test",
					"heartbeat.period",
					"5",
				),
			),
		},
	})
}
