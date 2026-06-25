package checkly

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccTracerouteMonitorRequiredFields(t *testing.T) {
	config := `resource "checkly_traceroute_monitor" "test" {}`
	accTestCase(t, []resource.TestStep{
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
			ExpectError: regexp.MustCompile(`The argument "frequency" is required, but no definition was found.`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`At least 1 "request" blocks are required.`),
		},
	})
}

func TestAccTracerouteMonitorBasic(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: tracerouteMonitor_basic,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test",
					"name",
					"Traceroute Monitor 1",
				),
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test",
					"activated",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test",
					"frequency",
					"1",
				),
				testCheckResourceAttrExpr(
					"checkly_traceroute_monitor.test",
					"locations.*",
					"us-east-1",
				),
				testCheckResourceAttrExpr(
					"checkly_traceroute_monitor.test",
					"request.*.url",
					"api.checklyhq.com",
				),
			),
		},
	})
}

func TestAccTracerouteMonitorFull(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: tracerouteMonitor_full,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test",
					"name",
					"tracerouteMonitor_full",
				),
				testCheckResourceAttrExpr(
					"checkly_traceroute_monitor.test",
					`"locations.#"`,
					"2",
				),
				testCheckResourceAttrExpr(
					"checkly_traceroute_monitor.test",
					"request.*.protocol",
					"TCP",
				),
				testCheckResourceAttrExpr(
					"checkly_traceroute_monitor.test",
					"request.*.max_hops",
					"30",
				),
				testCheckResourceAttrExpr(
					"checkly_traceroute_monitor.test",
					"request.*.max_unknown_hops",
					"15",
				),
				testCheckResourceAttrExpr(
					"checkly_traceroute_monitor.test",
					"request.*.ptr_lookup",
					"true",
				),
				testCheckResourceAttrExpr(
					"checkly_traceroute_monitor.test",
					"request.*.assertion.#",
					"2",
				),
				testCheckResourceAttrExpr(
					"checkly_traceroute_monitor.test",
					"request.*.assertion.*.source",
					"PACKET_LOSS",
				),
			),
		},
	})
}

// TestAccTracerouteMonitorMinimalCleanReplan asserts anti-pattern B is avoided:
// a config omitting every optional field applies, then re-plans with no diff.
func TestAccTracerouteMonitorMinimalCleanReplan(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: tracerouteMonitor_minimal,
		},
		{
			Config:             tracerouteMonitor_minimal,
			PlanOnly:           true,
			ExpectNonEmptyPlan: false,
		},
	})
}

const tracerouteMonitor_basic = `
	resource "checkly_traceroute_monitor" "test" {
	  name      = "Traceroute Monitor 1"
	  activated = true
	  frequency = 1
	  locations = ["us-east-1"]
	  request {
		url = "api.checklyhq.com"
	  }
	}
`

const tracerouteMonitor_full = `
	resource "checkly_traceroute_monitor" "test" {
	  name      = "tracerouteMonitor_full"
	  activated = true
	  frequency = 5
	  muted     = true
	  locations = [
		"us-east-1",
		"ap-northeast-1",
	  ]
	  request {
		url              = "api.checklyhq.com"
		protocol         = "TCP"
		port             = 443
		ip_family        = "IPv4"
		max_hops         = 30
		max_unknown_hops = 15
		ptr_lookup       = true
		timeout          = 10

		assertion {
		  source     = "PACKET_LOSS"
		  property   = ""
		  comparison = "LESS_THAN"
		  target     = "10"
		}

		assertion {
		  source     = "RESPONSE_TIME"
		  property   = "avg"
		  comparison = "LESS_THAN"
		  target     = "2000"
		}
	  }

	  alert_settings {
		escalation_type = "RUN_BASED"
		reminders {
		  amount   = 1
		  interval = 5
		}
		run_based_escalation {
		  failed_run_threshold = 1
		}
	  }
	}
`

// tracerouteMonitor_minimal omits every optional attribute; only the required
// name/activated/frequency/locations and request.url are set.
const tracerouteMonitor_minimal = `
	resource "checkly_traceroute_monitor" "test" {
	  name      = "traceroute-minimal"
	  activated = true
	  frequency = 1
	  locations = ["us-east-1"]
	  request {
		url = "api.checklyhq.com"
	  }
	}
`
