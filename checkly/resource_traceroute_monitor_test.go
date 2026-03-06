package checkly

import (
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/checkly/checkly-go-sdk"
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
			Config: `
				resource "checkly_traceroute_monitor" "test" {
					name                      = "Traceroute Monitor 1"
					frequency                 = 60
					activated                 = true
					muted                     = true
					locations                 = ["us-east-1", "eu-central-1"]
					use_global_alert_settings = true

					request {
						hostname = "example.com"

						assertion {
							source     = "RESPONSE_TIME"
							property   = "avg"
							comparison = "LESS_THAN"
							target     = "200"
						}
					}
				}
			`,
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
				testCheckResourceAttrExpr(
					"checkly_traceroute_monitor.test",
					"locations.*",
					"eu-central-1",
				),
				testCheckResourceAttrExpr(
					"checkly_traceroute_monitor.test",
					"locations.*",
					"us-east-1",
				),
				testCheckResourceAttrExpr(
					"checkly_traceroute_monitor.test",
					"request.*.hostname",
					"example.com",
				),
				testCheckResourceAttrExpr(
					"checkly_traceroute_monitor.test",
					"request.*.assertion.*.source",
					"RESPONSE_TIME",
				),
				testCheckResourceAttrExpr(
					"checkly_traceroute_monitor.test",
					"request.*.assertion.*.property",
					"avg",
				),
				testCheckResourceAttrExpr(
					"checkly_traceroute_monitor.test",
					"request.*.assertion.*.comparison",
					"LESS_THAN",
				),
				testCheckResourceAttrExpr(
					"checkly_traceroute_monitor.test",
					"request.*.assertion.*.target",
					"200",
				),
			),
		},
	})
}

func TestAccTracerouteMonitorFull(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_traceroute_monitor" "test" {
					name                   = "Traceroute Monitor 2"
					frequency              = 120
					activated              = true
					muted                  = true
					degraded_response_time = 10000
					max_response_time      = 25000

					locations = [
						"eu-central-1",
						"us-east-1",
						"ap-northeast-1"
					]

					request {
						hostname        = "api.checklyhq.com"
						port            = 443
						ip_family       = "IPv4"
						max_hops        = 20
						max_unknown_hops = 10
						ptr_lookup      = true
						timeout         = 15

						assertion {
							source     = "RESPONSE_TIME"
							property   = "avg"
							comparison = "LESS_THAN"
							target     = "200"
						}

						assertion {
							source     = "HOP_COUNT"
							comparison = "LESS_THAN"
							target     = "15"
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
					"checkly_traceroute_monitor.test",
					"degraded_response_time",
					"10000",
				),
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test",
					"max_response_time",
					"25000",
				),
				testCheckResourceAttrExpr(
					"checkly_traceroute_monitor.test",
					`"locations.#"`,
					"3",
				),
				testCheckResourceAttrExpr(
					"checkly_traceroute_monitor.test",
					"request.*.hostname",
					"api.checklyhq.com",
				),
				testCheckResourceAttrExpr(
					"checkly_traceroute_monitor.test",
					"request.*.port",
					"443",
				),
				testCheckResourceAttrExpr(
					"checkly_traceroute_monitor.test",
					"request.*.ip_family",
					"IPv4",
				),
				testCheckResourceAttrExpr(
					"checkly_traceroute_monitor.test",
					"request.*.max_hops",
					"20",
				),
				testCheckResourceAttrExpr(
					"checkly_traceroute_monitor.test",
					"request.*.max_unknown_hops",
					"10",
				),
				testCheckResourceAttrExpr(
					"checkly_traceroute_monitor.test",
					"request.*.ptr_lookup",
					"true",
				),
				testCheckResourceAttrExpr(
					"checkly_traceroute_monitor.test",
					"request.*.timeout",
					"15",
				),
				testCheckResourceAttrExpr(
					"checkly_traceroute_monitor.test",
					"request.*.assertion.#",
					"2",
				),
			),
		},
	})
}

func TestAccTracerouteMonitorWithTriggerIncident(t *testing.T) {
	config := `
resource "checkly_status_page_service" "test_traceroute_service" {
	name = "Traceroute Test Service"
}

resource "checkly_traceroute_monitor" "test_trigger_incident" {
	name          = "Traceroute Monitor with Trigger Incident"
	activated     = true
	frequency     = 30
	locations     = ["eu-west-2", "us-east-1"]

	request {
		hostname = "example.com"

		assertion {
			source     = "RESPONSE_TIME"
			property   = "avg"
			comparison = "LESS_THAN"
			target     = "200"
		}
	}

	trigger_incident {
		service_id         = checkly_status_page_service.test_traceroute_service.id
		severity           = "CRITICAL"
		name               = "Route Unreachable"
		description        = "The traceroute monitor has detected that the route may be unreachable"
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
					"checkly_traceroute_monitor.test_trigger_incident",
					"name",
					"Traceroute Monitor with Trigger Incident",
				),
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test_trigger_incident",
					"trigger_incident.0.severity",
					"CRITICAL",
				),
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test_trigger_incident",
					"trigger_incident.0.name",
					"Route Unreachable",
				),
			),
		},
	})
}

var ptrLookupTrue = true

var wantTracerouteMonitor = checkly.TracerouteMonitor{
	Name:                 "My test traceroute monitor",
	Frequency:            1,
	Activated:            true,
	Muted:                false,
	Locations:            []string{"eu-west-1"},
	DegradedResponseTime: 15000,
	MaxResponseTime:      30000,
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
	Request: checkly.TracerouteRequest{
		Hostname:       "example.com",
		Port:           443,
		IPFamily:       "IPv4",
		MaxHops:        30,
		MaxUnknownHops: 15,
		PtrLookup:      &ptrLookupTrue,
		Timeout:        10,
		Assertions: []checkly.Assertion{
			{
				Source:     "RESPONSE_TIME",
				Property:  "avg",
				Comparison: checkly.LessThan,
				Target:    "200",
			},
		},
	},
}

func TestEncodeDecodeTracerouteMonitorResource(t *testing.T) {
	res := resourceTracerouteMonitor()
	data := res.TestResourceData()
	wantTracerouteMonitor.AlertChannelSubscriptions = []checkly.AlertChannelSubscription{}
	resourceDataFromTracerouteMonitor(&wantTracerouteMonitor, data)
	got, err := tracerouteMonitorFromResourceData(data)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(wantTracerouteMonitor, got) {
		t.Error(cmp.Diff(wantTracerouteMonitor, got))
	}
}

func TestAccTracerouteMonitorGroupAssignment(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group" "test1" {
					name        = "test-traceroute-group-assignment"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}

				resource "checkly_traceroute_monitor" "test1" {
					name      = "test-traceroute-group-assignment"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					group_id  = checkly_check_group.test1.id
					request {
						hostname = "example.com"
					}
				}

				resource "checkly_traceroute_monitor" "test2" {
					name      = "test-traceroute-group-assignment"
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
					"checkly_traceroute_monitor.test1",
					"group_id",
					"checkly_check_group.test1",
					"id",
				),
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test2",
					"group_id",
					"0",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group" "test1" {
					name        = "test-traceroute-group-assignment"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}

				resource "checkly_traceroute_monitor" "test1" {
					name      = "test-traceroute-group-assignment"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						hostname = "example.com"
					}
				}

				resource "checkly_traceroute_monitor" "test2" {
					name      = "test-traceroute-group-assignment"
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
					"checkly_traceroute_monitor.test1",
					"group_id",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test2",
					"group_id",
					"0",
				),
			),
		},
	})
}
