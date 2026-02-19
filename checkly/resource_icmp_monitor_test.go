package checkly

import (
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/checkly/checkly-go-sdk"
)

func TestAccICMPMonitorRequiredFields(t *testing.T) {
	config := `resource "checkly_icmp_monitor" "test" {}`
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

func TestAccICMPMonitorBasic(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_icmp_monitor" "test" {
					name                      = "ICMP Monitor 1"
					frequency                 = 60
					activated                 = true
					muted                     = true
					locations                 = ["us-east-1", "eu-central-1"]
					use_global_alert_settings = true

					request {
						hostname = "example.com"

						assertion {
							source     = "LATENCY"
							property   = "avg"
							comparison = "LESS_THAN"
							target     = "200"
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test",
					"name",
					"ICMP Monitor 1",
				),
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test",
					"activated",
					"true",
				),
				testCheckResourceAttrExpr(
					"checkly_icmp_monitor.test",
					"locations.*",
					"eu-central-1",
				),
				testCheckResourceAttrExpr(
					"checkly_icmp_monitor.test",
					"locations.*",
					"us-east-1",
				),
				testCheckResourceAttrExpr(
					"checkly_icmp_monitor.test",
					"request.*.hostname",
					"example.com",
				),
				testCheckResourceAttrExpr(
					"checkly_icmp_monitor.test",
					"request.*.assertion.*.source",
					"LATENCY",
				),
				testCheckResourceAttrExpr(
					"checkly_icmp_monitor.test",
					"request.*.assertion.*.property",
					"avg",
				),
				testCheckResourceAttrExpr(
					"checkly_icmp_monitor.test",
					"request.*.assertion.*.comparison",
					"LESS_THAN",
				),
				testCheckResourceAttrExpr(
					"checkly_icmp_monitor.test",
					"request.*.assertion.*.target",
					"200",
				),
			),
		},
	})
}

func TestAccICMPMonitorFull(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_icmp_monitor" "test" {
					name                           = "ICMP Monitor 2"
					frequency                      = 120
					activated                      = true
					muted                          = true
					degraded_packet_loss_threshold = 10
					max_packet_loss_threshold      = 60

					locations = [
						"eu-central-1",
						"us-east-1",
						"ap-northeast-1"
					]

					request {
						hostname   = "api.checklyhq.com"
						ip_family  = "IPv4"
						ping_count = 10

						assertion {
							source     = "LATENCY"
							property   = "avg"
							comparison = "LESS_THAN"
							target     = "200"
						}

						assertion {
							source     = "LATENCY"
							property   = "max"
							comparison = "LESS_THAN"
							target     = "500"
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
					"checkly_icmp_monitor.test",
					"degraded_packet_loss_threshold",
					"10",
				),
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test",
					"max_packet_loss_threshold",
					"60",
				),
				testCheckResourceAttrExpr(
					"checkly_icmp_monitor.test",
					`"locations.#"`,
					"3",
				),
				testCheckResourceAttrExpr(
					"checkly_icmp_monitor.test",
					"request.*.hostname",
					"api.checklyhq.com",
				),
				testCheckResourceAttrExpr(
					"checkly_icmp_monitor.test",
					"request.*.ip_family",
					"IPv4",
				),
				testCheckResourceAttrExpr(
					"checkly_icmp_monitor.test",
					"request.*.ping_count",
					"10",
				),
				testCheckResourceAttrExpr(
					"checkly_icmp_monitor.test",
					"request.*.assertion.#",
					"2",
				),
				testCheckResourceAttrExpr(
					"checkly_icmp_monitor.test",
					"request.*.assertion.*.comparison",
					"LESS_THAN",
				),
				testCheckResourceAttrExpr(
					"checkly_icmp_monitor.test",
					"request.*.assertion.*.target",
					"200",
				),
				testCheckResourceAttrExpr(
					"checkly_icmp_monitor.test",
					"request.*.assertion.*.target",
					"500",
				),
			),
		},
	})
}

func TestAccICMPMonitorWithTriggerIncident(t *testing.T) {
	config := `
resource "checkly_status_page_service" "test_icmp_service" {
	name = "ICMP Test Service"
}

resource "checkly_icmp_monitor" "test_trigger_incident" {
	name          = "ICMP Monitor with Trigger Incident"
	activated     = true
	frequency     = 30
	locations     = ["eu-west-2", "us-east-1"]

	request {
		hostname = "example.com"

		assertion {
			source     = "LATENCY"
			property   = "avg"
			comparison = "LESS_THAN"
			target     = "200"
		}
	}

	trigger_incident {
		service_id         = checkly_status_page_service.test_icmp_service.id
		severity           = "CRITICAL"
		name               = "Host Unreachable"
		description        = "The ICMP monitor has detected that the host may be unreachable"
		notify_subscribers = true
	}

	use_global_alert_settings = true
}
`
	accTestCase(t, []resource.TestStep{
		{
			Config: config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test_trigger_incident",
					"name",
					"ICMP Monitor with Trigger Incident",
				),
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test_trigger_incident",
					"activated",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test_trigger_incident",
					"frequency",
					"30",
				),
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test_trigger_incident",
					"trigger_incident.0.severity",
					"CRITICAL",
				),
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test_trigger_incident",
					"trigger_incident.0.name",
					"Host Unreachable",
				),
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test_trigger_incident",
					"trigger_incident.0.description",
					"The ICMP monitor has detected that the host may be unreachable",
				),
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test_trigger_incident",
					"trigger_incident.0.notify_subscribers",
					"true",
				),
			),
		},
	})
}

var wantICMPMonitor = checkly.ICMPMonitor{
	Name:                        "My test ICMP monitor",
	Frequency:                   1,
	Activated:                   true,
	Muted:                       false,
	Locations:                   []string{"eu-west-1"},
	DegradedPacketLossThreshold: 10,
	MaxPacketLossThreshold:      20,
	Tags: []string{
		"bar",
		"foo",
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
	Request: checkly.ICMPRequest{
		Hostname:  "example.com",
		IPFamily:  "IPv4",
		PingCount: 10,
		Assertions: []checkly.Assertion{
			{
				Source:     "LATENCY",
				Property:  "avg",
				Comparison: checkly.LessThan,
				Target:    "200",
			},
		},
	},
}

func TestEncodeDecodeICMPMonitorResource(t *testing.T) {
	res := resourceICMPMonitor()
	data := res.TestResourceData()
	wantICMPMonitor.AlertChannelSubscriptions = []checkly.AlertChannelSubscription{}
	resourceDataFromICMPMonitor(&wantICMPMonitor, data)
	got, err := icmpMonitorFromResourceData(data)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(wantICMPMonitor, got) {
		t.Error(cmp.Diff(wantICMPMonitor, got))
	}
}

func TestAccICMPMonitorWithSingleRetry(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_icmp_monitor" "test" {
					name      = "test-single-retry"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						hostname = "example.com"
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
					"checkly_icmp_monitor.test",
					"retry_strategy.0.type",
					"SINGLE_RETRY",
				),
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test",
					"retry_strategy.0.base_backoff_seconds",
					"30",
				),
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test",
					"retry_strategy.0.same_region",
					"true",
				),
			),
		},
	})
}

func TestAccICMPMonitorWithNoRetries(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_icmp_monitor" "test" {
					name      = "test-no-retries"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						hostname = "example.com"
					}
					retry_strategy {
						type = "NO_RETRIES"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test",
					"retry_strategy.0.type",
					"NO_RETRIES",
				),
			),
		},
	})
}

func TestAccICMPMonitorWithDefaultNoRetries(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_icmp_monitor" "test" {
					name      = "test-no-retries"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						hostname = "example.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test",
					"retry_strategy.0.type",
					"NO_RETRIES",
				),
			),
		},
	})
}

func TestAccICMPMonitorGroupAssignment(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group" "test1" {
					name        = "test-icmp-group-assignment"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}

				resource "checkly_icmp_monitor" "test1" {
					name      = "test-icmp-group-assignment"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					group_id  = checkly_check_group.test1.id
					request {
						hostname = "example.com"
					}
				}

				resource "checkly_icmp_monitor" "test2" {
					name      = "test-icmp-group-assignment"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						hostname = "example.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrPair(
					"checkly_icmp_monitor.test1",
					"group_id",
					"checkly_check_group.test1",
					"id",
				),
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test2",
					"group_id",
					"0",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group" "test1" {
					name        = "test-icmp-group-assignment"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}

				resource "checkly_icmp_monitor" "test1" {
					name      = "test-icmp-group-assignment"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						hostname = "example.com"
					}
				}

				resource "checkly_icmp_monitor" "test2" {
					name      = "test-icmp-group-assignment"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						hostname = "example.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test1",
					"group_id",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test2",
					"group_id",
					"0",
				),
			),
		},
	})
}

func TestAccICMPMonitorFrequencyValidation(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_icmp_monitor" "test-missing-frequency_offset" {
					name      = "test-missing-frequency_offset"
					activated = true
					frequency = 0
					locations = ["eu-central-1"]
					request {
						hostname = "example.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`"frequency_offset" is required when "frequency" is 0`),
		},
		{
			Config: `
				resource "checkly_icmp_monitor" "test-frequency_offset-ok-10" {
					name             = "test-frequency_offset-ok-10"
					activated        = true
					frequency        = 0
					frequency_offset = 10
					locations        = ["eu-central-1"]
					request {
						hostname = "example.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test-frequency_offset-ok-10",
					"frequency",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test-frequency_offset-ok-10",
					"frequency_offset",
					"10",
				),
			),
		},
		{
			Config: `
				resource "checkly_icmp_monitor" "test-extra-frequency_offset" {
					name             = "test-extra-frequency_offset"
					activated        = true
					frequency        = 60
					frequency_offset = 30
					locations        = ["eu-central-1"]
					request {
						hostname = "example.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`"frequency_offset" can only be set when "frequency" is 0`),
		},
		{
			Config: `
				resource "checkly_icmp_monitor" "test-bad-frequency" {
					name      = "test-bad-frequency"
					activated = true
					frequency = 9999
					locations = ["eu-central-1"]
					request {
						hostname = "example.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`"frequency" must be one of \[0 1 2 5 10 15 30 60 120 180 360 720 1440\]`),
		},
		{
			Config: `
				resource "checkly_icmp_monitor" "test-frequency-ok-60" {
					name             = "test-frequency-ok-60"
					activated        = true
					frequency        = 60
					frequency_offset = 0
					locations        = ["eu-central-1"]
					request {
						hostname = "example.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test-frequency-ok-60",
					"frequency",
					"60",
				),
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test-frequency-ok-60",
					"frequency_offset",
					"0",
				),
			),
		},
	})
}

func TestAccICMPMonitorFrequencyUpdates(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_icmp_monitor" "test1" {
					name      = "test-normal-frequency"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						hostname = "example.com"
					}
				}

				resource "checkly_icmp_monitor" "test2" {
					name             = "test-high-frequency"
					activated        = true
					frequency        = 0
					frequency_offset = 30
					locations        = ["eu-central-1"]
					request {
						hostname = "example.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test1",
					"frequency",
					"60",
				),
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test1",
					"frequency_offset",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test2",
					"frequency",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test2",
					"frequency_offset",
					"30",
				),
			),
		},
		{
			Config: `
				resource "checkly_icmp_monitor" "test1" {
					name             = "test-normal-frequency-updated-to-high"
					activated        = true
					frequency        = 0
					frequency_offset = 10
					locations        = ["eu-central-1"]
					request {
						hostname = "example.com"
					}
				}

				resource "checkly_icmp_monitor" "test2" {
					name      = "test-high-frequency-updated-to-normal"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						hostname = "example.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test1",
					"frequency",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test1",
					"frequency_offset",
					"10",
				),
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test2",
					"frequency",
					"60",
				),
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test2",
					"frequency_offset",
					"0",
				),
			),
		},
	})
}
