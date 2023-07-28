package checkly

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
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

func TestAccHeartbeatCheckInvalidInputs(t *testing.T) {
	config := `resource "checkly_check" "test" {
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

func TestAccHeartbeatCheckMissingHeartbeatBlock(t *testing.T) {
	config := `resource "checkly_heartbeat" "test" {
		activated = true
		name = "heartbeat check"
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`At least 1 "heartbeat" blocks are required.`),
		},
	})
}

func TestAccHeartbeatCheckMissingHeartbeatFields(t *testing.T) {
	config := `resource "checkly_heartbeat" "test" {
		activated = true
		name = "heartbeat check"
		heartbeat {

		}
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "grace" is required, but no definition was found.`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "grace_unit" is required, but no definition was found.`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "period" is required, but no definition was found.`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "period_unit" is required, but no definition was found.`),
		},
	})
}

func TestAccHeartbeatCheckPeriodTooBig(t *testing.T) {
	config := `resource "checkly_heartbeat" "test" {
		activated = true
		name = "heartbeat check"
		heartbeat {
			period = 366
			period_unit = "days"
			grace = 0
			grace_unit = "seconds"
		}
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`translation error: period must be between 30 seconds and 365 days`),
		},
	})
}

func TestAccHeartbeatCheckPeriodTooSmall(t *testing.T) {
	config := `resource "checkly_heartbeat" "test" {
		activated = true
		name = "heartbeat check"
		heartbeat {
			period = 5
			period_unit = "seconds"
			grace = 0
			grace_unit = "seconds"
		}
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`translation error: period must be between 30 seconds and 365 days`),
		},
	})
}

func TestAccHeartbeatCheckInvalidPeriodUnit(t *testing.T) {
	config := `resource "checkly_heartbeat" "test" {
		activated = true
		name = "heartbeat check"
		heartbeat {
			period = 5
			period_unit = "lightyear"
			grace = 0
			grace_unit = "seconds"
		}
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`"heartbeat\.0\.period_unit" must be one of \[seconds minutes days\], got lightyear`),
		},
	})
}

func TestAccHeartbeatCheckInvalidGraceUnit(t *testing.T) {
	config := `resource "checkly_heartbeat" "test" {
		activated = true
		name = "heartbeat check"
		heartbeat {
			period = 5
			period_unit = "days"
			grace = 0
			grace_unit = "lightyear"
		}
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`"heartbeat\.0\.grace_unit" must be one of \[seconds minutes days\], got lightyear`),
		},
	})
}

func TestAccHeartbeatCheckCreate(t *testing.T) {
	config := `resource "checkly_heartbeat" "test" {
		activated = true
		name = "heartbeat check"
		heartbeat {
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
					"heartbeat check",
				),
				testCheckResourceAttrExpr(
					"checkly_heartbeat.test",
					"heartbeat.*.period",
					"5",
				),
				testCheckResourceAttrExpr(
					"checkly_heartbeat.test",
					"heartbeat.*.period_unit",
					"days",
				),
			),
		},
	})
}
