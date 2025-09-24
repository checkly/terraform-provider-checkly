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

func TestAccTCPMonitorWithTriggerIncident(t *testing.T) {
	tcpMonitorWithTriggerIncident := `
resource "checkly_status_page_service" "test_tcp_service" {
	name = "TCP Test Service"
}

resource "checkly_tcp_monitor" "test_trigger_incident" {
	name          = "TCP Monitor with Trigger Incident"
	activated     = true
	frequency     = 60
	locations     = ["us-west-1", "ap-south-1"]

	request {
		hostname = "example.com"
		port     = 443
		assertion {
			source     = "RESPONSE_TIME"
			comparison = "LESS_THAN"
			target     = "3000"
		}
	}

	trigger_incident {
		service_id         = checkly_status_page_service.test_tcp_service.id
		severity           = "MINOR"
		name               = "TCP Monitor Connection Issue"
		description        = "The TCP monitor has detected connectivity issues"
		notify_subscribers = false
	}

	use_global_alert_settings = true
}
`
	accTestCase(t, []resource.TestStep{
		{
			Config: tcpMonitorWithTriggerIncident,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test_trigger_incident",
					"name",
					"TCP Monitor with Trigger Incident",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test_trigger_incident",
					"activated",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test_trigger_incident",
					"frequency",
					"60",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test_trigger_incident",
					"trigger_incident.0.severity",
					"MINOR",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test_trigger_incident",
					"trigger_incident.0.name",
					"TCP Monitor Connection Issue",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test_trigger_incident",
					"trigger_incident.0.description",
					"The TCP monitor has detected connectivity issues",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test_trigger_incident",
					"trigger_incident.0.notify_subscribers",
					"false",
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

func TestAccTCPMonitorWithSingleRetry(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_tcp_monitor" "test" {
					name      = "test-single-retry"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						port     = 443
						hostname = "checkly.com"
					}
					retry_strategy {
						type                 = "SINGLE_RETRY"
						base_backoff_seconds = 30
						same_region          = true
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.type",
					"SINGLE_RETRY",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.base_backoff_seconds",
					"30",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.max_retries",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.max_duration_seconds",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.same_region",
					"true",
				),
			),
		},
	})
}

func TestAccTCPMonitorWithNoRetries(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_tcp_monitor" "test" {
					name      = "test-no-retries"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						port     = 443
						hostname = "checkly.com"
					}
					retry_strategy {
						type = "NO_RETRIES"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.type",
					"NO_RETRIES",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.base_backoff_seconds",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.max_retries",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.max_duration_seconds",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.same_region",
					"false",
				),
			),
		},
	})
}

func TestAccTCPMonitorWithDefaultNoRetries(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_tcp_monitor" "test" {
					name      = "test-no-retries"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						port     = 443
						hostname = "checkly.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.type",
					"NO_RETRIES",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.base_backoff_seconds",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.max_retries",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.max_duration_seconds",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.same_region",
					"false",
				),
			),
		},
	})
}

func TestAccTCPMonitorWithFixedRetry(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_tcp_monitor" "test" {
					name      = "test-fixed-retry"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						port     = 443
						hostname = "checkly.com"
					}
					retry_strategy {
						type                 = "FIXED"
						base_backoff_seconds = 30
						max_retries          = 3
						max_duration_seconds = 300
						same_region          = false
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.type",
					"FIXED",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.base_backoff_seconds",
					"30",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.max_retries",
					"3",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.max_duration_seconds",
					"300",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.same_region",
					"false",
				),
			),
		},
	})
}

func TestAccTCPMonitorWithExponentialRetry(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_tcp_monitor" "test" {
					name      = "test-exponential-retry"
					activated = true
					frequency = 720
					locations = ["eu-central-1"]
					request {
						port     = 443
						hostname = "checkly.com"
					}
					retry_strategy {
						type                 = "EXPONENTIAL"
						base_backoff_seconds = 60
						max_retries          = 5
						max_duration_seconds = 600
						same_region          = true
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.type",
					"EXPONENTIAL",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.base_backoff_seconds",
					"60",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.max_retries",
					"5",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.max_duration_seconds",
					"600",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.same_region",
					"true",
				),
			),
		},
	})
}

func TestAccTCPMonitorWithLinearRetry(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_tcp_monitor" "test" {
					name      = "test-linear-retry"
					activated = true
					frequency = 720
					locations = ["eu-central-1"]
					request {
						port     = 443
						hostname = "checkly.com"
					}
					retry_strategy {
						type                 = "LINEAR"
						base_backoff_seconds = 45
						max_retries          = 4
						max_duration_seconds = 450
						same_region          = false
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.type",
					"LINEAR",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.base_backoff_seconds",
					"45",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.max_retries",
					"4",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.max_duration_seconds",
					"450",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.same_region",
					"false",
				),
			),
		},
	})
}

func TestAccTCPMonitorRetryStrategyRemoval(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_tcp_monitor" "test" {
					name      = "test-linear-retry"
					activated = true
					frequency = 720
					locations = ["eu-central-1"]
					request {
						port     = 443
						hostname = "checkly.com"
					}
					retry_strategy {
						type = "LINEAR"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.type",
					"LINEAR",
				),
			),
		},
		{
			Config: `
				resource "checkly_tcp_monitor" "test" {
					name      = "test-linear-retry"
					activated = true
					frequency = 720
					locations = ["eu-central-1"]
					request {
						port     = 443
						hostname = "checkly.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.0.type",
					"NO_RETRIES",
				),
			),
		},
	})
}
