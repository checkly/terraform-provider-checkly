package checkly

import (
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/checkly/checkly-go-sdk"
)

func TestAccDNSMonitorRequiredFields(t *testing.T) {
	config := `resource "checkly_dns_monitor" "test" {}`
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

func TestAccDNSMonitorBasic(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name                      = "DNS Monitor 1"
					frequency                 = 60
					activated                 = true
					muted                     = true
					max_response_time         = 3000
					locations                 = ["us-east-1", "eu-central-1"]
					use_global_alert_settings = true

					request {
						record_type = "A"
						query       = "welcome.checklyhq.com"

						assertion {
							source     = "RESPONSE_CODE"
							comparison = "EQUALS"
							target     = "NOERROR"
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"name",
					"DNS Monitor 1",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"activated",
					"true",
				),
				testCheckResourceAttrExpr(
					"checkly_dns_monitor.test",
					"locations.*",
					"eu-central-1",
				),
				testCheckResourceAttrExpr(
					"checkly_dns_monitor.test",
					"locations.*",
					"us-east-1",
				),
				testCheckResourceAttrExpr(
					"checkly_dns_monitor.test",
					"request.*.record_type",
					"A",
				),
				testCheckResourceAttrExpr(
					"checkly_dns_monitor.test",
					"request.*.query",
					"welcome.checklyhq.com",
				),
				testCheckResourceAttrExpr(
					"checkly_dns_monitor.test",
					"request.*.assertion.*.comparison",
					"EQUALS",
				),
				testCheckResourceAttrExpr(
					"checkly_dns_monitor.test",
					"request.*.assertion.*.property",
					"",
				),
				testCheckResourceAttrExpr(
					"checkly_dns_monitor.test",
					"request.*.assertion.*.source",
					"RESPONSE_CODE",
				),
				testCheckResourceAttrExpr(
					"checkly_dns_monitor.test",
					"request.*.assertion.*.target",
					"NOERROR",
				),
			),
		},
	})
}

func TestAccDNSMonitorFull(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name                   = "DNS Monitor 2"
					frequency              = 120
					activated              = true
					muted                  = true
					degraded_response_time = 400
					max_response_time      = 500

					locations = [
						"eu-central-1",
						"us-east-1",
						"ap-northeast-1"
					]

					request {
						record_type  = "AAAA"
						query        = "api.checklyhq.com"
						protocol     = "TCP"

						name_server {
							host = "1.1.1.1"
							port = 53
						}

						assertion {
							source     = "RESPONSE_CODE"
							comparison = "EQUALS"
							target     = "NOERROR"
						}

						assertion {
							source     = "RESPONSE_TIME"
							comparison = "LESS_THAN"
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
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"degraded_response_time",
					"400",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"max_response_time",
					"500",
				),
				testCheckResourceAttrExpr(
					"checkly_dns_monitor.test",
					`"locations.#"`,
					"3",
				),
				testCheckResourceAttrExpr(
					"checkly_dns_monitor.test",
					"request.*.record_type",
					"AAAA",
				),
				testCheckResourceAttrExpr(
					"checkly_dns_monitor.test",
					"request.*.query",
					"api.checklyhq.com",
				),
				testCheckResourceAttrExpr(
					"checkly_dns_monitor.test",
					"request.*.name_server.*.host",
					"1.1.1.1",
				),
				testCheckResourceAttrExpr(
					"checkly_dns_monitor.test",
					"request.*.name_server.*.port",
					"53",
				),
				testCheckResourceAttrExpr(
					"checkly_dns_monitor.test",
					"request.*.protocol",
					"TCP",
				),
				testCheckResourceAttrExpr(
					"checkly_dns_monitor.test",
					"request.*.assertion.#",
					"2",
				),
				testCheckResourceAttrExpr(
					"checkly_dns_monitor.test",
					"request.*.assertion.*.comparison",
					"EQUALS",
				),
				testCheckResourceAttrExpr(
					"checkly_dns_monitor.test",
					"request.*.assertion.*.comparison",
					"LESS_THAN",
				),
				testCheckResourceAttrExpr(
					"checkly_dns_monitor.test",
					"request.*.assertion.*.target",
					"NOERROR",
				),
				testCheckResourceAttrExpr(
					"checkly_dns_monitor.test",
					"request.*.assertion.*.target",
					"300",
				),
			),
		},
	})
}

func TestAccDNSMonitorWithTriggerIncident(t *testing.T) {
	dnsMonitorWithTriggerIncident := `
resource "checkly_status_page_service" "test_dns_service" {
	name = "DNS Test Service"
}

resource "checkly_dns_monitor" "test_trigger_incident" {
	name          = "DNS Monitor with Trigger Incident"
	activated     = true
	frequency     = 30
	locations     = ["eu-west-2", "us-east-1"]

	request {
		record_type = "A"
		query       = "welcome.checklyhq.com"

		assertion {
			source     = "RESPONSE_CODE"
			comparison = "EQUALS"
			target     = "NOERROR"
		}
	}

	trigger_incident {
		service_id         = checkly_status_page_service.test_dns_service.id
		severity           = "CRITICAL"
		name               = "Missing DNS Record"
		description        = "The DNS monitor has detected that a record may be missing"
		notify_subscribers = true
	}

	use_global_alert_settings = true
}
`
	accTestCase(t, []resource.TestStep{
		{
			Config: dnsMonitorWithTriggerIncident,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test_trigger_incident",
					"name",
					"DNS Monitor with Trigger Incident",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test_trigger_incident",
					"activated",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test_trigger_incident",
					"frequency",
					"30",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test_trigger_incident",
					"trigger_incident.0.severity",
					"CRITICAL",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test_trigger_incident",
					"trigger_incident.0.name",
					"Missing DNS Record",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test_trigger_incident",
					"trigger_incident.0.description",
					"The DNS monitor has detected that a record may be missing",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test_trigger_incident",
					"trigger_incident.0.notify_subscribers",
					"true",
				),
			),
		},
	})
}

var wantDNSMonitor = checkly.DNSMonitor{
	Name:                 "My test check",
	Frequency:            1,
	Activated:            true,
	Muted:                false,
	Locations:            []string{"eu-west-1"},
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
	Request: checkly.DNSRequest{
		RecordType: "AAAA",
		Query:      "example.com",
		NameServer: "1.1.1.1",
		Port:       53,
		Protocol:   "UDP",
		Assertions: []checkly.Assertion{
			{
				Source:     "RESPONSE_CODE",
				Comparison: checkly.Equals,
				Target:     "NOERROR",
			},
		},
	},
}

func TestEncodeDecodeDNSMonitorResource(t *testing.T) {
	res := resourceDNSMonitor()
	data := res.TestResourceData()
	wantDNSMonitor.AlertChannelSubscriptions = []checkly.AlertChannelSubscription{}
	resourceDataFromDNSMonitor(&wantDNSMonitor, data)
	got, err := dnsMonitorFromResourceData(data)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(wantDNSMonitor, got) {
		t.Error(cmp.Diff(wantDNSMonitor, got))
	}
}

func TestAccDNSMonitorWithSingleRetry(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name      = "test-single-retry"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						record_type = "A"
						query       = "welcome.checklyhq.com"
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
					"checkly_dns_monitor.test",
					"retry_strategy.0.type",
					"SINGLE_RETRY",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.base_backoff_seconds",
					"30",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.max_retries",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.max_duration_seconds",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.same_region",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.only_on.#",
					"0",
				),
			),
		},
	})

}

func TestAccDNSMonitorWithSingleRetryOnlyOnNetworkError(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name      = "test-single-retry"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
					retry_strategy {
						type = "SINGLE_RETRY"

						only_on {
							network_error = true
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.only_on.#",
					"1",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.only_on.0.network_error",
					"true",
				),
			),
		},
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name      = "test-single-retry"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
					retry_strategy {
						type = "SINGLE_RETRY"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.only_on.#",
					"0",
				),
			),
		},
	})
}

func TestAccDNSMonitorWithNoRetries(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name      = "test-no-retries"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
					retry_strategy {
						type = "NO_RETRIES"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.type",
					"NO_RETRIES",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.base_backoff_seconds",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.max_retries",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.max_duration_seconds",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.same_region",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.only_on.#",
					"0",
				),
			),
		},
	})
}

func TestAccDNSMonitorWithNoRetriesOnlyOnNetworkError(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name      = "test-no-retries"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
					retry_strategy {
						type = "NO_RETRIES"

						only_on {
							network_error = true
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.only_on.#",
					"0",
				),
			),
		},
	})
}

func TestAccDNSMonitorWithDefaultNoRetries(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name      = "test-no-retries"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.type",
					"NO_RETRIES",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.base_backoff_seconds",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.max_retries",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.max_duration_seconds",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.same_region",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.only_on.#",
					"0",
				),
			),
		},
	})
}

func TestAccDNSMonitorWithFixedRetry(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name      = "test-fixed-retry"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						record_type = "A"
						query       = "welcome.checklyhq.com"
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
					"checkly_dns_monitor.test",
					"retry_strategy.0.type",
					"FIXED",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.base_backoff_seconds",
					"30",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.max_retries",
					"3",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.max_duration_seconds",
					"300",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.same_region",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.only_on.#",
					"0",
				),
			),
		},
	})
}

func TestAccDNSMonitorWithFixedRetryOnlyOnNetworkError(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name      = "test-fixed-retry"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
					retry_strategy {
						type = "FIXED"

						only_on {
							network_error = true
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.only_on.#",
					"1",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.only_on.0.network_error",
					"true",
				),
			),
		},
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name      = "test-fixed-retry"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
					retry_strategy {
						type = "FIXED"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.only_on.#",
					"0",
				),
			),
		},
	})
}

func TestAccDNSMonitorWithExponentialRetry(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name      = "test-exponential-retry"
					activated = true
					frequency = 720
					locations = ["eu-central-1"]
					request {
						record_type = "A"
						query       = "welcome.checklyhq.com"
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
					"checkly_dns_monitor.test",
					"retry_strategy.0.type",
					"EXPONENTIAL",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.base_backoff_seconds",
					"60",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.max_retries",
					"5",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.max_duration_seconds",
					"600",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.same_region",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.only_on.#",
					"0",
				),
			),
		},
	})
}

func TestAccDNSMonitorWithExponentialRetryOnlyOnNetworkError(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name      = "test-exponential-retry"
					activated = true
					frequency = 720
					locations = ["eu-central-1"]
					request {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
					retry_strategy {
						type = "EXPONENTIAL"

						only_on {
							network_error = true
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.only_on.#",
					"1",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.only_on.0.network_error",
					"true",
				),
			),
		},
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name      = "test-exponential-retry"
					activated = true
					frequency = 720
					locations = ["eu-central-1"]
					request {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
					retry_strategy {
						type = "EXPONENTIAL"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.only_on.#",
					"0",
				),
			),
		},
	})
}

func TestAccDNSMonitorWithLinearRetry(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name      = "test-linear-retry"
					activated = true
					frequency = 720
					locations = ["eu-central-1"]
					request {
						record_type = "A"
						query       = "welcome.checklyhq.com"
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
					"checkly_dns_monitor.test",
					"retry_strategy.0.type",
					"LINEAR",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.base_backoff_seconds",
					"45",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.max_retries",
					"4",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.max_duration_seconds",
					"450",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.same_region",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.only_on.#",
					"0",
				),
			),
		},
	})
}

func TestAccDNSMonitorWithLinearRetryOnlyOnNetworkError(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name      = "test-linear-retry"
					activated = true
					frequency = 720
					locations = ["eu-central-1"]
					request {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
					retry_strategy {
						type = "LINEAR"

						only_on {
							network_error = true
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.only_on.#",
					"1",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.only_on.0.network_error",
					"true",
				),
			),
		},
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name      = "test-linear-retry"
					activated = true
					frequency = 720
					locations = ["eu-central-1"]
					request {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
					retry_strategy {
						type = "LINEAR"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.only_on.#",
					"0",
				),
			),
		},
	})
}

func TestAccDNSMonitorRetryStrategyRemoval(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name      = "test-single-retry"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
					retry_strategy {
						type = "LINEAR"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.type",
					"LINEAR",
				),
			),
		},
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name      = "test-single-retry"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.0.type",
					"NO_RETRIES",
				),
			),
		},
	})
}

func TestAccDNSMonitorGroupAssignment(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group" "test1" {
					name        = "test-group-assignment"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}

				resource "checkly_check_group" "test2" {
					name        = "test-group-assignment"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}

				resource "checkly_dns_monitor" "test1" {
					name      = "test-group-assignment"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					group_id  = checkly_check_group.test1.id
					request {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}

				resource "checkly_dns_monitor" "test2" {
					name      = "test-group-assignment"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrPair(
					"checkly_dns_monitor.test1",
					"group_id",
					"checkly_check_group.test1",
					"id",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test2",
					"group_id",
					"0",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group" "test1" {
					name        = "test-group-assignment"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}

				resource "checkly_check_group" "test2" {
					name        = "test-group-assignment"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}

				resource "checkly_dns_monitor" "test1" {
					name      = "test-group-assignment"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					group_id  = checkly_check_group.test2.id
					request {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}

				resource "checkly_dns_monitor" "test2" {
					name      = "test-group-assignment"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					group_id  = checkly_check_group.test2.id
					request {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrPair(
					"checkly_dns_monitor.test1",
					"group_id",
					"checkly_check_group.test2",
					"id",
				),
				resource.TestCheckResourceAttrPair(
					"checkly_dns_monitor.test2",
					"group_id",
					"checkly_check_group.test2",
					"id",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group" "test1" {
					name        = "test-group-assignment"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}

				resource "checkly_check_group" "test2" {
					name        = "test-group-assignment"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}

				resource "checkly_dns_monitor" "test1" {
					name      = "test-group-assignment"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}

				resource "checkly_dns_monitor" "test2" {
					name      = "test-group-assignment"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test1",
					"group_id",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test2",
					"group_id",
					"0",
				),
			),
		},
	})
}
