package resources_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccHeartbeatRequiredFields(t *testing.T) {
	config := `resource "checkly_heartbeat" "test" {}`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),

		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`The argument "name" is required, but no definition was found.`),
			},
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`The argument "activated" is required, but no definition was found.`),
			},
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`The argument "heartbeat" is required, but no definition was found.`),
			},
		},
	})
}

func TestAccHeartbeatCheckInvalidInputs(t *testing.T) {
	config := `resource "checkly_check" "test" {
		name                      = 1
		activated                 = "invalid"
		use_global_alert_settings = "invalid"
	}`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),

		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`Inappropriate value for attribute "activated"`),
			},
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`Inappropriate value for attribute "use_global_alert_settings"`),
			},
		},
	})
}

func TestAccHeartbeatCheckMissingHeartbeatFields(t *testing.T) {
	config := `resource "checkly_heartbeat" "test" {
		activated = true
		name = "heartbeat check"
		heartbeat = {

		}
	}`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),

		Steps: []resource.TestStep{
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
		},
	})
}

func TestAccHeartbeatCheckPeriodTooBig(t *testing.T) {
	config := `resource "checkly_heartbeat" "test" {
		activated = true
		name = "heartbeat check"
		heartbeat = {
			period = 366
			period_unit = "days"
			grace = 0
			grace_unit = "seconds"
		}
	}`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),

		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`Attribute heartbeat\.period value [\s\S]*?must be less than or equal to 365 days`),
			},
		},
	})
}

func TestAccHeartbeatCheckPeriodTooSmall(t *testing.T) {
	config := `resource "checkly_heartbeat" "test" {
		activated = true
		name = "heartbeat check"
		heartbeat = {
			period = 5
			period_unit = "seconds"
			grace = 0
			grace_unit = "seconds"
		}
	}`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),

		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`Attribute heartbeat.period value[\s\S]*?must be greater than or equal to 30s`),
			},
		},
	})
}

func TestAccHeartbeatCheckInvalidPeriodUnit(t *testing.T) {
	config := `resource "checkly_heartbeat" "test" {
		activated = true
		name = "heartbeat check"
		heartbeat = {
			period = 5
			period_unit = "lightyear"
			grace = 0
			grace_unit = "seconds"
		}
	}`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),

		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`Attribute heartbeat\.period_unit value must be one of`),
			},
		},
	})
}

func TestAccHeartbeatCheckInvalidGraceUnit(t *testing.T) {
	config := `resource "checkly_heartbeat" "test" {
		activated = true
		name = "heartbeat check"
		heartbeat = {
			period = 5
			period_unit = "days"
			grace = 0
			grace_unit = "lightyear"
		}
	}`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),

		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`Attribute heartbeat\.grace_unit value must be one of`),
			},
		},
	})
}

func TestAccHeartbeatCheckCreate(t *testing.T) {
	config := `resource "checkly_heartbeat" "test" {
		activated = true
		name = "heartbeat check"
		heartbeat = {
			period = 5
			period_unit = "days"
			grace = 0
			grace_unit = "seconds"
		}
	}`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),

		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"checkly_heartbeat.test",
						"name",
						"heartbeat check",
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
		},
	})
}
