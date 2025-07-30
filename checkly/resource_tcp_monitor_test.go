package checkly

import (
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/checkly/checkly-go-sdk"
)

func TestAccTCPMonitorRequiredFields(t *testing.T) {
	config := `resource "checkly_tcp_monitor" "test" {}`
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

func TestAccTCPMonitorBasic(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: tcpMonitor_basic,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"name",
					"TCP Monitor 1",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"activated",
					"true",
				),
				testCheckResourceAttrExpr(
					"checkly_tcp_monitor.test",
					"locations.*",
					"eu-central-1",
				),
				testCheckResourceAttrExpr(
					"checkly_tcp_monitor.test",
					"locations.*",
					"us-east-1",
				),
				testCheckResourceAttrExpr(
					"checkly_tcp_monitor.test",
					"request.*.hostname",
					"api.checklyhq.com",
				),
				testCheckResourceAttrExpr(
					"checkly_tcp_monitor.test",
					"request.*.port",
					"80",
				),
				testCheckResourceAttrExpr(
					"checkly_tcp_monitor.test",
					"request.*.assertion.*.comparison",
					"EQUALS",
				),
				testCheckResourceAttrExpr(
					"checkly_tcp_monitor.test",
					"request.*.assertion.*.property",
					"",
				),
				testCheckResourceAttrExpr(
					"checkly_tcp_monitor.test",
					"request.*.assertion.*.source",
					"RESPONSE_DATA",
				),
				testCheckResourceAttrExpr(
					"checkly_tcp_monitor.test",
					"request.*.assertion.*.target",
					"hello",
				),
			),
		},
	})
}

func TestAccTCPMonitorFull(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: tcpMonitor_full,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"degraded_response_time",
					"4000",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"max_response_time",
					"5000",
				),
				testCheckResourceAttrExpr(
					"checkly_tcp_monitor.test",
					`"locations.#"`,
					"3",
				),
				testCheckResourceAttrExpr(
					"checkly_tcp_monitor.test",
					"request.*.assertion.#",
					"2",
				),
				testCheckResourceAttrExpr(
					"checkly_tcp_monitor.test",
					"request.*.assertion.*.comparison",
					"LESS_THAN",
				),
				testCheckResourceAttrExpr(
					"checkly_tcp_monitor.test",
					"request.*.assertion.*.comparison",
					"CONTAINS",
				),
				testCheckResourceAttrExpr(
					"checkly_tcp_monitor.test",
					"request.*.assertion.*.target",
					"2000",
				),
				testCheckResourceAttrExpr(
					"checkly_tcp_monitor.test",
					"request.*.assertion.*.target",
					"Foo",
				),
			),
		},
	})
}

var wantTCPMonitor = checkly.TCPMonitor{
	Name:                 "My test check",
	Frequency:            1,
	Activated:            true,
	Muted:                false,
	ShouldFail:           false,
	Locations:            []string{"eu-west-1"},
	PrivateLocations:     &[]string{},
	DegradedResponseTime: 4000,
	MaxResponseTime:      5000,
	Tags: []string{
		"foo",
		"bar",
	},
	AlertSettings: &checkly.AlertSettings{
		EscalationType: checkly.RunBased,
		RunBasedEscalation: checkly.RunBasedEscalation{
			FailedRunThreshold: 1,
		},
		Reminders: checkly.Reminders{
			Interval: 5,
		},
	},
	UseGlobalAlertSettings: false,
	Request: checkly.TCPRequest{
		Hostname: "example.com",
		Port:     80,
		Assertions: []checkly.Assertion{
			{
				Source:     checkly.ResponseData,
				Comparison: checkly.Equals,
				Target:     "html",
			},
		},
	},
}

func TestEncodeDecodeTCPMonitorResource(t *testing.T) {
	res := resourceTCPMonitor()
	data := res.TestResourceData()
	wantTCPMonitor.AlertChannelSubscriptions = []checkly.AlertChannelSubscription{}
	resourceDataFromTCPMonitor(&wantTCPMonitor, data)
	got, err := tcpCheckFromResourceData(data)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(wantTCPMonitor, got) {
		t.Error(cmp.Diff(wantTCPMonitor, got))
	}
}

const tcpMonitor_basic = `
	resource "checkly_tcp_monitor" "test" {
	  name                      = "TCP Monitor 1"
	  frequency                 = 60
	  activated                 = true
	  muted                     = true
	  max_response_time         = 3000
	  locations                 = [ "us-east-1", "eu-central-1" ]
	  use_global_alert_settings = true
	  request {
		hostname     = "api.checklyhq.com"
		port         = 80
		assertion {
		  comparison = "LESS_THAN"
		  property   = ""
		  source     = "RESPONSE_TIME"
		  target     = "2000"
		}
	  }
	}
`

const tcpMonitor_full = `
  resource "checkly_tcp_monitor" "test" {
	name                   = "tcpMonitor_full"
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
	request {
	  hostname         = "api.checklyhq.com"
	  port             = 80
	  assertion {
		comparison = "LESS_THAN"
		property   = ""
		source     = "RESPONSE_TIME"
		target     = "2000"
	  }
	  assertion {
		comparison = "CONTAINS"
		property   = ""
		source     = "RESPONSE_DATA"
		target     = "Foo"
	  }
	}

	alert_settings {
	  escalation_type = "RUN_BASED"
	  reminders {
		amount   = 0
		interval = 5
	  }
	  run_based_escalation {
		failed_run_threshold = 1
	  }
		parallel_run_failure_threshold {
		enabled = false
		percentage = 10
		}
	}
  }
`
