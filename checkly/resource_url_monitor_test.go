package checkly

import (
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/checkly/checkly-go-sdk"
)

func TestAccURLMonitorRequiredFields(t *testing.T) {
	config := `resource "checkly_url_monitor" "test" {}`
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

func TestAccURLMonitorBasic(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: urlMonitor_basic,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"name",
					"URL Monitor 1",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"activated",
					"true",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					"locations.*",
					"eu-central-1",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					"locations.*",
					"us-east-1",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					"request.*.url",
					"https://api.checklyhq.com",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					"request.*.assertion.*.comparison",
					"EQUALS",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					"request.*.assertion.*.property",
					"",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					"request.*.assertion.*.source",
					"STATUS_CODE",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					"request.*.assertion.*.target",
					"200",
				),
			),
		},
	})
}

func TestAccURLMonitorFull(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: urlMonitor_full,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"degraded_response_time",
					"4000",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"max_response_time",
					"5000",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					`"locations.#"`,
					"3",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					"request.*.follow_redirects",
					"true",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					"request.*.skip_ssl",
					"false",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					"request.*.ip_family",
					"IPv6",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					"request.*.assertion.#",
					"2",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					"request.*.assertion.*.comparison",
					"LESS_THAN",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					"request.*.assertion.*.comparison",
					"GREATER_THAN",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					"request.*.assertion.*.target",
					"199",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					"request.*.assertion.*.target",
					"300",
				),
			),
		},
	})
}

var wantURLMonitor = checkly.URLMonitor{
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
	Request: checkly.URLRequest{
		URL:             "https://example.com",
		FollowRedirects: true,
		SkipSSL:         false,
		IPFamily:        "IPv6",
		Assertions: []checkly.Assertion{
			{
				Source:     checkly.StatusCode,
				Comparison: checkly.Equals,
				Target:     "200",
			},
		},
	},
}

func TestEncodeDecodeURLMonitorResource(t *testing.T) {
	res := resourceURLMonitor()
	data := res.TestResourceData()
	wantURLMonitor.AlertChannelSubscriptions = []checkly.AlertChannelSubscription{}
	resourceDataFromURLMonitor(&wantURLMonitor, data)
	got, err := urlMonitorFromResourceData(data)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(wantURLMonitor, got) {
		t.Error(cmp.Diff(wantURLMonitor, got))
	}
}

const urlMonitor_basic = `
resource "checkly_url_monitor" "test" {
  name                      = "URL Monitor 1"
  frequency                 = 60
  activated                 = true
  muted                     = true
  max_response_time         = 3000
  locations                 = ["us-east-1", "eu-central-1"]
  use_global_alert_settings = true
  request {
    url = "https://api.checklyhq.com"
    assertion {
      comparison = "EQUALS"
      property   = ""
      source     = "STATUS_CODE"
      target     = "200"
    }
  }
}
`

const urlMonitor_full = `
resource "checkly_url_monitor" "test" {
  name                   = "URL Monitor 2"
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
    url              = "https://api.checklyhq.com"
    follow_redirects = true
    skip_ssl         = false
    ip_family        = "IPv6"
    assertion {
      comparison = "GREATER_THAN"
      property   = ""
      source     = "STATUS_CODE"
      target     = "199"
    }
    assertion {
      comparison = "LESS_THAN"
      property   = ""
      source     = "STATUS_CODE"
      target     = "300"
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
      enabled    = false
      percentage = 10
    }
  }
}
`
