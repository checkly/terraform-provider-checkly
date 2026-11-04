package provider

import (
	"context"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

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
			ExpectError: regexp.MustCompile(`The argument "request" is required, but no definition was found.`),
		},
	})
}

func TestAccDNSMonitorBasic(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name                      = "DNS Monitor 1"
					description               = "DNS monitor description"
					frequency                 = 60
					activated                 = true
					muted                     = true
					max_response_time         = 3000
					locations                 = ["us-east-1", "eu-central-1"]
					use_global_alert_settings = true

					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"

						assertion = [{
							source     = "RESPONSE_CODE"
							comparison = "EQUALS"
							target     = "NOERROR"
						}]
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
					"description",
					"DNS monitor description",
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
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"request.record_type",
					"A",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"request.query",
					"welcome.checklyhq.com",
				),
				resource.TestCheckTypeSetElemNestedAttrs(
					"checkly_dns_monitor.test",
					"request.assertion.*",
					map[string]string{
						"source":     "RESPONSE_CODE",
						"property":   "",
						"comparison": "EQUALS",
						"target":     "NOERROR",
					},
				),
			),
		},
	})
}

func TestAccDNSMonitorDescriptionRemoval(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name                      = "DNS Monitor 1"
					description               = "DNS monitor description"
					frequency                 = 60
					activated                 = true
					muted                     = true
					max_response_time         = 3000
					locations                 = ["us-east-1", "eu-central-1"]
					use_global_alert_settings = true

					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"

						assertion = [{
							source     = "RESPONSE_CODE"
							comparison = "EQUALS"
							target     = "NOERROR"
						}]
					}
				}
			`,
			Check: resource.TestCheckResourceAttr(
				"checkly_dns_monitor.test",
				"description",
				"DNS monitor description",
			),
		},
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

					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"

						assertion = [{
							source     = "RESPONSE_CODE"
							comparison = "EQUALS"
							target     = "NOERROR"
						}]
					}
				}
			`,
			Check: resource.TestCheckResourceAttr(
				"checkly_dns_monitor.test",
				"description",
				"",
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

					request = {
						record_type  = "AAAA"
						query        = "api.checklyhq.com"
						protocol     = "TCP"

						name_server = {
							host = "1.1.1.1"
							port = 53
						}

						assertion = [{
							source     = "RESPONSE_CODE"
							comparison = "EQUALS"
							target     = "NOERROR"
						}, {
							source     = "RESPONSE_TIME"
							comparison = "LESS_THAN"
							target     = "300"
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
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"request.record_type",
					"AAAA",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"request.query",
					"api.checklyhq.com",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"request.name_server.host",
					"1.1.1.1",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"request.name_server.port",
					"53",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"request.protocol",
					"TCP",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"request.assertion.#",
					"2",
				),
				resource.TestCheckTypeSetElemNestedAttrs(
					"checkly_dns_monitor.test",
					"request.assertion.*",
					map[string]string{
						"source":     "RESPONSE_CODE",
						"comparison": "EQUALS",
						"target":     "NOERROR",
					},
				),
				resource.TestCheckTypeSetElemNestedAttrs(
					"checkly_dns_monitor.test",
					"request.assertion.*",
					map[string]string{
						"source":     "RESPONSE_TIME",
						"comparison": "LESS_THAN",
						"target":     "300",
					},
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

	request = {
		record_type = "A"
		query       = "welcome.checklyhq.com"

		assertion = [{
			source     = "RESPONSE_CODE"
			comparison = "EQUALS"
			target     = "NOERROR"
		}]
	}

	trigger_incident = {
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
					"trigger_incident.severity",
					"CRITICAL",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test_trigger_incident",
					"trigger_incident.name",
					"Missing DNS Record",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test_trigger_incident",
					"trigger_incident.description",
					"The DNS monitor has detected that a record may be missing",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test_trigger_incident",
					"trigger_incident.notify_subscribers",
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
	RetryStrategy:          &checkly.RetryStrategy{Type: "NO_RETRIES"},
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
	ctx := context.Background()

	var diags diag.Diagnostics

	// Start from a model whose prior-aware values are typed nulls, as they
	// would be when decoded from an empty plan or state.
	model := DNSMonitorResourceModel{
		RetryStrategy: types.ObjectNull(retryStrategyAttrTypes(false)),
		AlertSettings: types.ObjectNull(alertSettingsAttrTypes),
	}

	want := wantDNSMonitor
	want.AlertChannelSubscriptions = []checkly.AlertChannelSubscription{}

	dnsMonitorToModel(ctx, &want, &model, &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}

	got := dnsMonitorFromModel(ctx, &model, &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}

	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
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
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
					retry_strategy = {
						type                 = "SINGLE_RETRY"
						base_backoff_seconds = 30
						same_region          = true
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.type",
					"SINGLE_RETRY",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.base_backoff_seconds",
					"30",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.max_retries",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.max_duration_seconds",
					"600",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.same_region",
					"true",
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
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
					retry_strategy = {
						type = "NO_RETRIES"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.type",
					"NO_RETRIES",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.base_backoff_seconds",
					"60",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.max_retries",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.max_duration_seconds",
					"600",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.same_region",
					"true",
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
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.type",
					"NO_RETRIES",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.base_backoff_seconds",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.max_retries",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.max_duration_seconds",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.same_region",
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
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
					retry_strategy = {
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
					"retry_strategy.type",
					"FIXED",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.base_backoff_seconds",
					"30",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.max_retries",
					"3",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.max_duration_seconds",
					"300",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.same_region",
					"false",
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
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
					retry_strategy = {
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
					"retry_strategy.type",
					"EXPONENTIAL",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.base_backoff_seconds",
					"60",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.max_retries",
					"5",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.max_duration_seconds",
					"600",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.same_region",
					"true",
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
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
					retry_strategy = {
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
					"retry_strategy.type",
					"LINEAR",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.base_backoff_seconds",
					"45",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.max_retries",
					"4",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.max_duration_seconds",
					"450",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.same_region",
					"false",
				),
			),
		},
	})
}

// DNS monitors do not support retry conditions, so their retry_strategy has
// no only_on attribute. Terraform's object conversion rules silently drop
// unexpected attributes from attribute values, so a configured only_on is
// ignored rather than rejected.
func TestAccDNSMonitorWithOnlyOnNetworkErrorIgnored(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name      = "test-only-on-ignored"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
					retry_strategy = {
						type = "SINGLE_RETRY"

						only_on = {
							network_error = true
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.type",
					"SINGLE_RETRY",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.only_on.network_error",
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
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
					retry_strategy = {
						type = "LINEAR"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.type",
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
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"retry_strategy.type",
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
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}

				resource "checkly_dns_monitor" "test2" {
					name      = "test-group-assignment"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
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
					request = {
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
					request = {
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
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}

				resource "checkly_dns_monitor" "test2" {
					name      = "test-group-assignment"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
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

func TestAccDNSMonitorNameServerRemoval(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name      = "test-name-server-removal"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"

						name_server = {
							host = "1.1.1.1"
							port = 53
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"request.name_server.%",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"request.name_server.host",
					"1.1.1.1",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"request.name_server.port",
					"53",
				),
			),
		},
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name      = "test-name-server-removal"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(
					"checkly_dns_monitor.test",
					"request.name_server",
				),
			),
		},
	})
}

func TestAccDNSMonitorNameServerEmptyObject(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name      = "test-name-server-empty"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"

						name_server = {}
					}
				}
			`,
			ExpectError: regexp.MustCompile(`Invalid Name Server`),
		},
	})
}

func TestAccDNSMonitorFrequencyValidation(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_dns_monitor" "test-missing-frequency_offset" {
					name      = "test-missing-frequency_offset"
					activated = true
					frequency = 0
					locations = ["eu-central-1"]
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`"frequency_offset" is required when "frequency" is 0`),
		},
		{
			Config: `
				resource "checkly_dns_monitor" "test-frequency_offset-ok-10" {
					name             = "test-frequency_offset-ok-10"
					activated        = true
					frequency        = 0
					frequency_offset = 10
					locations        = ["eu-central-1"]
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}

				resource "checkly_dns_monitor" "test-frequency_offset-ok-20" {
					name             = "test-frequency_offset-ok-20"
					activated        = true
					frequency        = 0
					frequency_offset = 20
					locations        = ["eu-central-1"]
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}

				resource "checkly_dns_monitor" "test-frequency_offset-ok-30" {
					name             = "test-frequency_offset-ok-30"
					activated        = true
					frequency        = 0
					frequency_offset = 30
					locations        = ["eu-central-1"]
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test-frequency_offset-ok-10",
					"frequency",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test-frequency_offset-ok-10",
					"frequency_offset",
					"10",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test-frequency_offset-ok-20",
					"frequency",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test-frequency_offset-ok-20",
					"frequency_offset",
					"20",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test-frequency_offset-ok-30",
					"frequency",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test-frequency_offset-ok-30",
					"frequency_offset",
					"30",
				),
			),
		},
		{
			Config: `
				resource "checkly_dns_monitor" "test-extra-frequency_offset" {
					name             = "test-extra-frequency_offset"
					activated        = true
					frequency        = 60
					frequency_offset = 30
					locations        = ["eu-central-1"]
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`"frequency_offset" can only be set when "frequency" is 0`),
		},
		{
			Config: `
				resource "checkly_dns_monitor" "test-bad-frequency" {
					name      = "test-bad-frequency"
					activated = true
					frequency = 9999
					locations = ["eu-central-1"]
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`Attribute frequency value must be one of`),
		},
		{
			Config: `
				resource "checkly_dns_monitor" "test-bad-frequency_offset" {
					name             = "test-bad-frequency_offset"
					activated        = true
					frequency        = 60
					frequency_offset = 99
					locations        = ["eu-central-1"]
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`Attribute frequency_offset value must be one of`),
		},
		{
			Config: `
				resource "checkly_dns_monitor" "test-frequency-ok-60" {
					name             = "test-frequency-ok-60"
					activated        = true
					frequency        = 60
					frequency_offset = 0
					locations        = ["eu-central-1"]
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test-frequency-ok-60",
					"frequency",
					"60",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test-frequency-ok-60",
					"frequency_offset",
					"0",
				),
			),
		},
	})
}

func TestAccDNSMonitorFrequencyUpdates(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_dns_monitor" "test1" {
					name      = "test-normal-frequency"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}

				resource "checkly_dns_monitor" "test2" {
					name             = "test-high-frequency"
					activated        = true
					frequency        = 0
					frequency_offset = 30
					locations        = ["eu-central-1"]
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test1",
					"frequency",
					"60",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_dns_monitor.test1",
					"frequency_offset",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test2",
					"frequency",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test2",
					"frequency_offset",
					"30",
				),
			),
		},
		{
			Config: `
				resource "checkly_dns_monitor" "test1" {
					name             = "test-normal-frequency-updated-to-high"
					activated        = true
					frequency        = 0
					frequency_offset = 10
					locations        = ["eu-central-1"]
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}

				resource "checkly_dns_monitor" "test2" {
					name      = "test-high-frequency-updated-to-normal"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test1",
					"frequency",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test1",
					"frequency_offset",
					"10",
				),
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test2",
					"frequency",
					"60",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_dns_monitor.test2",
					"frequency_offset",
				),
			),
		},
	})
}

func TestAccDNSMonitorAlertChannelSubscriptions(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		// Step 1: Add two alert channel subscriptions.
		{
			Config: `
				resource "checkly_alert_channel" "email_1" {
					email = {
						address = "dns-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email = {
						address = "dns-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email = {
						address = "dns-acs-3@example.com"
					}
				}

				resource "checkly_dns_monitor" "test" {
					name      = "dns-alert-channel-sub-test"
					frequency = 60
					activated = true
					locations = ["eu-central-1"]

					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"

						assertion = [{
							source     = "RESPONSE_CODE"
							comparison = "EQUALS"
							target     = "NOERROR"
						}]
					}

					alert_channel_subscription {
						channel_id = checkly_alert_channel.email_1.id
						activated  = true
					}

					alert_channel_subscription {
						channel_id = checkly_alert_channel.email_2.id
						activated  = false
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"alert_channel_subscription.#",
					"2",
				),
				testCheckAlertChannelSubscription(
					"checkly_dns_monitor.test",
					"alert_channel_subscription",
					"checkly_alert_channel.email_1",
					"true",
				),
				testCheckAlertChannelSubscription(
					"checkly_dns_monitor.test",
					"alert_channel_subscription",
					"checkly_alert_channel.email_2",
					"false",
				),
			),
		},
		// Step 2: Change activations and add a third subscription.
		{
			Config: `
				resource "checkly_alert_channel" "email_1" {
					email = {
						address = "dns-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email = {
						address = "dns-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email = {
						address = "dns-acs-3@example.com"
					}
				}

				resource "checkly_dns_monitor" "test" {
					name      = "dns-alert-channel-sub-test"
					frequency = 60
					activated = true
					locations = ["eu-central-1"]

					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"

						assertion = [{
							source     = "RESPONSE_CODE"
							comparison = "EQUALS"
							target     = "NOERROR"
						}]
					}

					alert_channel_subscription {
						channel_id = checkly_alert_channel.email_1.id
						activated  = false
					}

					alert_channel_subscription {
						channel_id = checkly_alert_channel.email_2.id
						activated  = true
					}

					alert_channel_subscription {
						channel_id = checkly_alert_channel.email_3.id
						activated  = true
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"alert_channel_subscription.#",
					"3",
				),
				testCheckAlertChannelSubscription(
					"checkly_dns_monitor.test",
					"alert_channel_subscription",
					"checkly_alert_channel.email_1",
					"false",
				),
				testCheckAlertChannelSubscription(
					"checkly_dns_monitor.test",
					"alert_channel_subscription",
					"checkly_alert_channel.email_2",
					"true",
				),
				testCheckAlertChannelSubscription(
					"checkly_dns_monitor.test",
					"alert_channel_subscription",
					"checkly_alert_channel.email_3",
					"true",
				),
			),
		},
		// Step 3: Remove all alert channel subscriptions.
		{
			Config: `
				resource "checkly_alert_channel" "email_1" {
					email = {
						address = "dns-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email = {
						address = "dns-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email = {
						address = "dns-acs-3@example.com"
					}
				}

				resource "checkly_dns_monitor" "test" {
					name      = "dns-alert-channel-sub-test"
					frequency = 60
					activated = true
					locations = ["eu-central-1"]

					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"

						assertion = [{
							source     = "RESPONSE_CODE"
							comparison = "EQUALS"
							target     = "NOERROR"
						}]
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dns_monitor.test",
					"alert_channel_subscription.#",
					"0",
				),
			),
		},
	})
}

// TestAccDNSMonitorUnsetSweepBasics sets every plain optional attribute to a
// non-default value and then removes them all, verifying the server actually
// clears (or resets to the default) each value. The Checkly PUT endpoint is
// PATCH-style, so a key omitted from the update payload keeps its server-side
// value; the implicit post-apply refresh would surface that as a non-empty
// plan and fail the test.
func TestAccDNSMonitorUnsetSweepBasics(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name                      = "dns-unset-sweep-basics"
					activated                 = true
					frequency                 = 60
					locations                 = ["eu-central-1"]
					muted                     = true
					run_parallel              = true
					use_global_alert_settings = false
					degraded_response_time    = 400
					max_response_time         = 900
					tags                      = ["sweep-a", "sweep-b"]
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_dns_monitor.test", "muted", "true"),
				resource.TestCheckResourceAttr("checkly_dns_monitor.test", "run_parallel", "true"),
				resource.TestCheckResourceAttr("checkly_dns_monitor.test", "use_global_alert_settings", "false"),
				resource.TestCheckResourceAttr("checkly_dns_monitor.test", "degraded_response_time", "400"),
				resource.TestCheckResourceAttr("checkly_dns_monitor.test", "max_response_time", "900"),
				resource.TestCheckResourceAttr("checkly_dns_monitor.test", "tags.#", "2"),
			),
		},
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name      = "dns-unset-sweep-basics"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_dns_monitor.test", "muted", "false"),
				resource.TestCheckResourceAttr("checkly_dns_monitor.test", "run_parallel", "false"),
				resource.TestCheckResourceAttr("checkly_dns_monitor.test", "use_global_alert_settings", "true"),
				resource.TestCheckResourceAttr("checkly_dns_monitor.test", "degraded_response_time", "500"),
				resource.TestCheckResourceAttr("checkly_dns_monitor.test", "max_response_time", "1000"),
				resource.TestCheckNoResourceAttr("checkly_dns_monitor.test", "tags.#"),
			),
		},
	})
}

// TestAccDNSMonitorUnsetSweepResponseTimeZero verifies that the response time
// limits accept an explicit 0 (a schema-valid value that omitempty-style
// serialization would drop) and reset to their documented defaults when
// removed from the configuration.
func TestAccDNSMonitorUnsetSweepResponseTimeZero(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name                   = "dns-unset-sweep-response-time-zero"
					activated              = true
					frequency              = 60
					locations              = ["eu-central-1"]
					degraded_response_time = 0
					max_response_time      = 0
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_dns_monitor.test", "degraded_response_time", "0"),
				resource.TestCheckResourceAttr("checkly_dns_monitor.test", "max_response_time", "0"),
			),
		},
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name      = "dns-unset-sweep-response-time-zero"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_dns_monitor.test", "degraded_response_time", "500"),
				resource.TestCheckResourceAttr("checkly_dns_monitor.test", "max_response_time", "1000"),
			),
		},
	})
}

// TestAccDNSMonitorUnsetSweepLocations verifies that removing the locations
// attribute entirely clears the locations on the server.
func TestAccDNSMonitorUnsetSweepLocations(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name      = "dns-unset-sweep-locations"
					activated = true
					frequency = 60
					locations = ["eu-central-1", "us-east-1"]
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_dns_monitor.test", "locations.#", "2"),
			),
		},
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name      = "dns-unset-sweep-locations"
					activated = true
					frequency = 60
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr("checkly_dns_monitor.test", "locations.#"),
			),
		},
	})
}

// TestAccDNSMonitorUnsetSweepRequest sets the optional request attributes to
// non-default values plus an assertion, then removes them, verifying the
// attributes reset to their defaults and the assertions are cleared on the
// server. The name_server removal is covered separately by
// TestAccDNSMonitorNameServerRemoval.
func TestAccDNSMonitorUnsetSweepRequest(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name      = "dns-unset-sweep-request"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
						protocol    = "TCP"
						assertion = [{
							source     = "RESPONSE_CODE"
							comparison = "EQUALS"
							target     = "NOERROR"
						}]
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_dns_monitor.test", "request.protocol", "TCP"),
				resource.TestCheckResourceAttr("checkly_dns_monitor.test", "request.assertion.#", "1"),
			),
		},
		{
			Config: `
				resource "checkly_dns_monitor" "test" {
					name      = "dns-unset-sweep-request"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_dns_monitor.test", "request.protocol", "UDP"),
				resource.TestCheckResourceAttr("checkly_dns_monitor.test", "request.assertion.#", "0"),
			),
		},
	})
}

// TestAccDNSMonitorUnsetSweepTriggerIncident verifies that removing the
// trigger_incident attribute clears the incident trigger on the server.
func TestAccDNSMonitorUnsetSweepTriggerIncident(t *testing.T) {
	service := `
		resource "checkly_status_page_service" "dns_sweep" {
			name = "DNS Monitor Unset Sweep Service"
		}
	`
	accTestCase(t, []resource.TestStep{
		{
			Config: service + `
				resource "checkly_dns_monitor" "test" {
					name      = "dns-unset-sweep-trigger-incident"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
					trigger_incident = {
						service_id         = checkly_status_page_service.dns_sweep.id
						severity           = "MEDIUM"
						name               = "DNS sweep incident"
						description        = "Incident created by the unset sweep test"
						notify_subscribers = false
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_dns_monitor.test", "trigger_incident.%", "5"),
			),
		},
		{
			Config: service + `
				resource "checkly_dns_monitor" "test" {
					name      = "dns-unset-sweep-trigger-incident"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr("checkly_dns_monitor.test", "trigger_incident"),
			),
		},
	})
}

// TestAccDNSMonitorUnsetSweepGroup verifies that removing group_id and
// group_order detaches the monitor from the group and resets the order.
func TestAccDNSMonitorUnsetSweepGroup(t *testing.T) {
	group := `
		resource "checkly_check_group" "dns_sweep" {
			name        = "dns-unset-sweep-group"
			activated   = true
			concurrency = 1
			locations   = ["eu-central-1"]
		}
	`
	accTestCase(t, []resource.TestStep{
		{
			Config: group + `
				resource "checkly_dns_monitor" "test" {
					name        = "dns-unset-sweep-group"
					activated   = true
					frequency   = 60
					locations   = ["eu-central-1"]
					group_id    = checkly_check_group.dns_sweep.id
					group_order = 2
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrPair(
					"checkly_dns_monitor.test",
					"group_id",
					"checkly_check_group.dns_sweep",
					"id",
				),
				resource.TestCheckResourceAttr("checkly_dns_monitor.test", "group_order", "2"),
			),
		},
		{
			Config: group + `
				resource "checkly_dns_monitor" "test" {
					name      = "dns-unset-sweep-group"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_dns_monitor.test", "group_id", "0"),
				resource.TestCheckResourceAttr("checkly_dns_monitor.test", "group_order", "0"),
			),
		},
	})
}
