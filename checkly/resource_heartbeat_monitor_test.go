package checkly

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccHeartbeatMonitorRequiredFields(t *testing.T) {
	config := `resource "checkly_heartbeat_monitor" "test" {}`
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

func TestAccHeartbeatMonitorInvalidInputs(t *testing.T) {
	config := `resource "checkly_heartbeat_monitor" "test" {
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

func TestAccHeartbeatMonitorMissingHeartbeatBlock(t *testing.T) {
	config := `resource "checkly_heartbeat_monitor" "test" {
		activated = true
		name = "heartbeat monitor"
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`At least 1 "heartbeat" blocks are required.`),
		},
	})
}

func TestAccHeartbeatMonitorMissingHeartbeatFields(t *testing.T) {
	config := `resource "checkly_heartbeat_monitor" "test" {
		activated = true
		name = "heartbeat monitor"
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

func TestAccHeartbeatMonitorPeriodTooBig(t *testing.T) {
	config := `resource "checkly_heartbeat_monitor" "test" {
		activated = true
		name = "heartbeat monitor"
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

func TestAccHeartbeatMonitorPeriodTooSmall(t *testing.T) {
	config := `resource "checkly_heartbeat_monitor" "test" {
		activated = true
		name = "heartbeat monitor"
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

func TestAccHeartbeatMonitorInvalidPeriodUnit(t *testing.T) {
	config := `resource "checkly_heartbeat_monitor" "test" {
		activated = true
		name = "heartbeat monitor"
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
			ExpectError: regexp.MustCompile(`"heartbeat\.0\.period_unit" must be one of \[seconds minutes hours days\], got lightyear`),
		},
	})
}

func TestAccHeartbeatMonitorInvalidGraceUnit(t *testing.T) {
	config := `resource "checkly_heartbeat_monitor" "test" {
		activated = true
		name = "heartbeat monitor"
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
			ExpectError: regexp.MustCompile(`"heartbeat\.0\.grace_unit" must be one of \[seconds minutes hours days\], got lightyear`),
		},
	})
}

func TestAccHeartbeatMonitorCreate(t *testing.T) {
	config := `resource "checkly_heartbeat_monitor" "test" {
		activated = true
		name = "heartbeat monitor"
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
					"checkly_heartbeat_monitor.test",
					"name",
					"heartbeat monitor",
				),
				testCheckResourceAttrExpr(
					"checkly_heartbeat_monitor.test",
					"heartbeat.*.period",
					"5",
				),
				testCheckResourceAttrExpr(
					"checkly_heartbeat_monitor.test",
					"heartbeat.*.period_unit",
					"days",
				),
			),
		},
	})
}

func TestAccHeartbeatMonitorWithTriggerIncident(t *testing.T) {
	heartbeatMonitorWithTriggerIncident := `
resource "checkly_status_page_service" "test_heartbeat_service" {
	name = "Heartbeat Test Service"
}

resource "checkly_heartbeat_monitor" "test_trigger_incident" {
	name      = "Heartbeat Monitor with Trigger Incident"
	activated = true

	heartbeat {
		period      = 10
		period_unit = "minutes"
		grace       = 5
		grace_unit  = "minutes"
	}

	trigger_incident {
		service_id         = checkly_status_page_service.test_heartbeat_service.id
		severity           = "MEDIUM"
		name               = "Heartbeat Monitor Failure"
		description        = "The heartbeat monitor has not received a ping within the grace period"
		notify_subscribers = true
	}

	use_global_alert_settings = true
}
`
	accTestCase(t, []resource.TestStep{
		{
			Config: heartbeatMonitorWithTriggerIncident,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_heartbeat_monitor.test_trigger_incident",
					"name",
					"Heartbeat Monitor with Trigger Incident",
				),
				resource.TestCheckResourceAttr(
					"checkly_heartbeat_monitor.test_trigger_incident",
					"activated",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_heartbeat_monitor.test_trigger_incident",
					"trigger_incident.0.severity",
					"MEDIUM",
				),
				resource.TestCheckResourceAttr(
					"checkly_heartbeat_monitor.test_trigger_incident",
					"trigger_incident.0.name",
					"Heartbeat Monitor Failure",
				),
				resource.TestCheckResourceAttr(
					"checkly_heartbeat_monitor.test_trigger_incident",
					"trigger_incident.0.description",
					"The heartbeat monitor has not received a ping within the grace period",
				),
				resource.TestCheckResourceAttr(
					"checkly_heartbeat_monitor.test_trigger_incident",
					"trigger_incident.0.notify_subscribers",
					"true",
				),
			),
		},
	})
}
