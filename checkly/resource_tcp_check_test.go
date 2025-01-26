package checkly

import (
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/checkly/checkly-go-sdk"
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
				testCheckResourceAttrExpr(
					"checkly_tcp_check.test",
					"locations.*",
					"eu-central-1",
				),
				testCheckResourceAttrExpr(
					"checkly_tcp_check.test",
					"locations.*",
					"us-east-1",
				),
				testCheckResourceAttrExpr(
					"checkly_tcp_check.test",
					"request.*.hostname",
					"api.checklyhq.com",
				),
				testCheckResourceAttrExpr(
					"checkly_tcp_check.test",
					"request.*.port",
					"80",
				),
				testCheckResourceAttrExpr(
					"checkly_tcp_check.test",
					"request.*.assertion.*.comparison",
					"EQUALS",
				),
				testCheckResourceAttrExpr(
					"checkly_tcp_check.test",
					"request.*.assertion.*.property",
					"",
				),
				testCheckResourceAttrExpr(
					"checkly_tcp_check.test",
					"request.*.assertion.*.source",
					"RESPONSE_DATA",
				),
				testCheckResourceAttrExpr(
					"checkly_tcp_check.test",
					"request.*.assertion.*.target",
					"hello",
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
					"15000",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_check.test",
					"max_response_time",
					"30000",
				),
				testCheckResourceAttrExpr(
					"checkly_tcp_check.test",
					`"locations.#"`,
					"3",
				),
				testCheckResourceAttrExpr(
					"checkly_tcp_check.test",
					"request.*.assertion.#",
					"2",
				),
				testCheckResourceAttrExpr(
					"checkly_tcp_check.test",
					"request.*.assertion.*.comparison",
					"LESS_THAN",
				),
				testCheckResourceAttrExpr(
					"checkly_tcp_check.test",
					"request.*.assertion.*.comparison",
					"CONTAINS",
				),
				testCheckResourceAttrExpr(
					"checkly_tcp_check.test",
					"request.*.assertion.*.target",
					"2000",
				),
				testCheckResourceAttrExpr(
					"checkly_tcp_check.test",
					"request.*.assertion.*.target",
					"Foo",
				),
			),
		},
	})
}

var wantTCPCheck = checkly.TCPCheck{
	Name:                 "My test check",
	Frequency:            1,
	Activated:            true,
	Muted:                false,
	ShouldFail:           false,
	Locations:            []string{"eu-west-1"},
	PrivateLocations:     &[]string{},
	DegradedResponseTime: 15000,
	MaxResponseTime:      30000,
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

func TestEncodeDecodeTCPCheckResource(t *testing.T) {
	res := resourceTCPCheck()
	data := res.TestResourceData()
	wantTCPCheck.AlertChannelSubscriptions = []checkly.AlertChannelSubscription{}
	resourceDataFromTCPCheck(&wantTCPCheck, data)
	got, err := tcpCheckFromResourceData(data)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(wantTCPCheck, got) {
		t.Error(cmp.Diff(wantTCPCheck, got))
	}
}

const tcpCheck_basic = `
	resource "checkly_tcp_check" "test" {
	  name                      = "TCP Check 1"
	  frequency                 = 60
	  activated                 = true
	  muted                     = true
	  max_response_time         = 18000
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

const tcpCheck_full = `
  resource "checkly_tcp_check" "test" {
	name                   = "tcpCheck_full"
	frequency              = 120
	activated              = true
	muted                  = true
	degraded_response_time = 15000
	max_response_time      = 30000
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
