package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTCPCheckRequiredFields(t *testing.T) {
	config := `resource "checkly_tcp_check" "test" {}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "name" is required, but no definition was found.`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "request" is required, but no definition was found.`),
		},
	})
}

func TestAccTCPCheckBasic(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: tcpCheck_basic,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_tcp_check.test",
					"name",
					"TCP Check 1",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_check.test",
					"activated",
					"true",
				),
				resource.TestCheckTypeSetElemAttr(
					"checkly_tcp_check.test",
					"locations.*",
					"eu-central-1",
				),
				resource.TestCheckTypeSetElemAttr(
					"checkly_tcp_check.test",
					"locations.*",
					"us-east-1",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_check.test",
					"request.hostname",
					"api.checklyhq.com",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_check.test",
					"request.port",
					"80",
				),
				resource.TestCheckTypeSetElemNestedAttrs(
					"checkly_tcp_check.test",
					"request.assertions.*",
					map[string]string{
						"source":     "RESPONSE_TIME",
						"property":   "",
						"comparison": "LESS_THAN",
						"target":     "2000",
					},
				),
			),
		},
	})
}

func TestAccTCPCheckFull(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: tcpCheck_full,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_tcp_check.test",
					"degraded_response_time",
					"4000",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_check.test",
					"max_response_time",
					"5000",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_check.test",
					"locations.#",
					"3",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_check.test",
					"request.assertions.#",
					"2",
				),
				resource.TestCheckTypeSetElemNestedAttrs(
					"checkly_tcp_check.test",
					"request.assertions.*",
					map[string]string{
						"source":     "RESPONSE_TIME",
						"property":   "",
						"comparison": "LESS_THAN",
						"target":     "2000",
					},
				),
				resource.TestCheckTypeSetElemNestedAttrs(
					"checkly_tcp_check.test",
					"request.assertions.*",
					map[string]string{
						"source":     "RESPONSE_DATA",
						"property":   "",
						"comparison": "CONTAINS",
						"target":     "Foo",
					},
				),
			),
		},
	})
}

// TestAccTCPCheckMovedToTCPMonitor exercises the MoveState implementation of
// checkly_tcp_monitor via a `moved` block from checkly_tcp_check.
func TestAccTCPCheckMovedToTCPMonitor(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: tcpCheck_basic,
		},
		{
			Config: `
				resource "checkly_tcp_monitor" "test" {
				  name                      = "TCP Check 1"
				  frequency                 = 60
				  activated                 = true
				  muted                     = true
				  max_response_time         = 3000
				  locations                 = [ "us-east-1", "eu-central-1" ]
				  use_global_alert_settings = true
				  request = {
					hostname     = "api.checklyhq.com"
					port         = 80
					assertions = [{
					  comparison = "LESS_THAN"
					  property   = ""
					  source     = "RESPONSE_TIME"
					  target     = "2000"
					}]
				  }
				}

				moved {
					from = checkly_tcp_check.test
					to   = checkly_tcp_monitor.test
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"name",
					"TCP Check 1",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"activated",
					"true",
				),
			),
		},
	})
}

const tcpCheck_basic = `
	resource "checkly_tcp_check" "test" {
	  name                      = "TCP Check 1"
	  frequency                 = 60
	  activated                 = true
	  muted                     = true
	  max_response_time         = 3000
	  locations                 = [ "us-east-1", "eu-central-1" ]
	  use_global_alert_settings = true
	  request = {
		hostname     = "api.checklyhq.com"
		port         = 80
		assertions = [{
		  comparison = "LESS_THAN"
		  property   = ""
		  source     = "RESPONSE_TIME"
		  target     = "2000"
		}]
	  }
	}
`

const tcpCheck_full = `
  resource "checkly_tcp_check" "test" {
	name                   = "tcpCheck_full"
	frequency              = 120
	activated              = true
	muted                  = true
	degraded_response_time = 4000
	max_response_time      = 5000
	locations = [
	  "eu-central-1",
	  "us-east-1",
	  "ap-northeast-1"
	]
	request = {
	  hostname         = "api.checklyhq.com"
	  port             = 80
	  assertions = [{
		comparison = "LESS_THAN"
		property   = ""
		source     = "RESPONSE_TIME"
		target     = "2000"
	  }, {
		comparison = "CONTAINS"
		property   = ""
		source     = "RESPONSE_DATA"
		target     = "Foo"
	  }]
	}

	alert_settings = {
	  escalation_type = "RUN_BASED"
	  reminders = {
		amount   = 0
		interval = 5
	  }
	  run_based_escalation = {
		failed_run_threshold = 1
	  }
	  parallel_run_failure_threshold = {
		enabled    = false
		percentage = 10
	  }
	}
  }
`
