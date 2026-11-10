package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/checkly/checkly-go-sdk/v2"
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
			ExpectError: regexp.MustCompile(`The argument "request" is required, but no definition was found.`),
		},
	})
}

func TestAccICMPMonitorBasic(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_icmp_monitor" "test" {
					name                      = "ICMP Monitor 1"
					description               = "ICMP monitor description"
					frequency                 = 60
					activated                 = true
					muted                     = true
					locations                 = ["us-east-1", "eu-central-1"]
					use_global_alert_settings = true

					request = {
						hostname = "example.com"

						assertions = [{
							source     = "LATENCY"
							property   = "avg"
							comparison = "LESS_THAN"
							target     = "200"
						}]
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
					"description",
					"ICMP monitor description",
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
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test",
					"request.hostname",
					"example.com",
				),
				resource.TestCheckTypeSetElemNestedAttrs(
					"checkly_icmp_monitor.test",
					"request.assertions.*",
					map[string]string{
						"source":     "LATENCY",
						"property":   "avg",
						"comparison": "LESS_THAN",
						"target":     "200",
					},
				),
			),
		},
	})
}

func TestAccICMPMonitorDescriptionRemoval(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_icmp_monitor" "test" {
					name                      = "ICMP Monitor 1"
					description               = "ICMP monitor description"
					frequency                 = 60
					activated                 = true
					muted                     = true
					locations                 = ["us-east-1", "eu-central-1"]
					use_global_alert_settings = true

					request = {
						hostname = "example.com"

						assertions = [{
							source     = "LATENCY"
							property   = "avg"
							comparison = "LESS_THAN"
							target     = "200"
						}]
					}
				}
			`,
			Check: resource.TestCheckResourceAttr(
				"checkly_icmp_monitor.test",
				"description",
				"ICMP monitor description",
			),
		},
		{
			Config: `
				resource "checkly_icmp_monitor" "test" {
					name                      = "ICMP Monitor 1"
					frequency                 = 60
					activated                 = true
					muted                     = true
					locations                 = ["us-east-1", "eu-central-1"]
					use_global_alert_settings = true

					request = {
						hostname = "example.com"

						assertions = [{
							source     = "LATENCY"
							property   = "avg"
							comparison = "LESS_THAN"
							target     = "200"
						}]
					}
				}
			`,
			Check: resource.TestCheckResourceAttr(
				"checkly_icmp_monitor.test",
				"description",
				"",
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

					request = {
						hostname   = "api.checklyhq.com"
						ip_family  = "IPv4"
						ping_count = 10

						assertions = [{
							source     = "LATENCY"
							property   = "avg"
							comparison = "LESS_THAN"
							target     = "200"
						}, {
							source     = "LATENCY"
							property   = "max"
							comparison = "LESS_THAN"
							target     = "500"
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
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test",
					"request.hostname",
					"api.checklyhq.com",
				),
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test",
					"request.ip_family",
					"IPv4",
				),
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test",
					"request.ping_count",
					"10",
				),
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test",
					"request.assertions.#",
					"2",
				),
				resource.TestCheckTypeSetElemNestedAttrs(
					"checkly_icmp_monitor.test",
					"request.assertions.*",
					map[string]string{
						"source":     "LATENCY",
						"property":   "avg",
						"comparison": "LESS_THAN",
						"target":     "200",
					},
				),
				resource.TestCheckTypeSetElemNestedAttrs(
					"checkly_icmp_monitor.test",
					"request.assertions.*",
					map[string]string{
						"source":     "LATENCY",
						"property":   "max",
						"comparison": "LESS_THAN",
						"target":     "500",
					},
				),
			),
		},
		{
			ResourceName:      "checkly_icmp_monitor.test",
			ImportState:       true,
			ImportStateVerify: true,
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

	request = {
		hostname = "example.com"

		assertions = [{
			source     = "LATENCY"
			property   = "avg"
			comparison = "LESS_THAN"
			target     = "200"
		}]
	}

	trigger_incident = {
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
					"trigger_incident.severity",
					"CRITICAL",
				),
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test_trigger_incident",
					"trigger_incident.name",
					"Host Unreachable",
				),
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test_trigger_incident",
					"trigger_incident.description",
					"The ICMP monitor has detected that the host may be unreachable",
				),
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test_trigger_incident",
					"trigger_incident.notify_subscribers",
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
	PrivateLocations: &[]string{"my-private-location"},
	AlertSettings: &checkly.AlertSettings{
		EscalationType: checkly.EscalationTypeRunBased,
		RunBasedEscalation: checkly.RunBasedEscalation{
			FailedRunThreshold: 1,
		},
		Reminders: checkly.Reminders{
			Interval: 5,
		},
	},
	UseGlobalAlertSettings: false,
	RetryStrategy:          &checkly.RetryStrategy{Type: "NO_RETRIES"},
	Request: checkly.ICMPRequest{
		Hostname:  "example.com",
		IPFamily:  "IPv4",
		PingCount: 10,
		Assertions: []checkly.Assertion{
			{
				Source:     "LATENCY",
				Property:   "avg",
				Comparison: checkly.AssertionComparisonLessThan,
				Target:     "200",
			},
		},
	},
}

func TestEncodeDecodeICMPMonitorResource(t *testing.T) {
	ctx := context.Background()

	var diags diag.Diagnostics

	// Start from a model whose prior-aware values are typed nulls, as they
	// would be when decoded from an empty plan or state.
	model := ICMPMonitorResourceModel{
		RetryStrategy: types.ObjectNull(retryStrategyAttrTypes(false)),
		AlertSettings: types.ObjectNull(alertSettingsAttrTypes),
	}

	want := wantICMPMonitor
	want.AlertChannelSubscriptions = []checkly.AlertChannelSubscription{}

	icmpMonitorToModel(ctx, &want, &model, &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}

	got := icmpMonitorFromModel(ctx, &model, &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}

	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
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
					request = {
						hostname = "example.com"
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
					"checkly_icmp_monitor.test",
					"retry_strategy.type",
					"SINGLE_RETRY",
				),
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test",
					"retry_strategy.base_backoff_seconds",
					"30",
				),
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test",
					"retry_strategy.same_region",
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
					request = {
						hostname = "example.com"
					}
					retry_strategy = {
						type = "NO_RETRIES"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test",
					"retry_strategy.type",
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
					request = {
						hostname = "example.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test",
					"retry_strategy.type",
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
					request = {
						hostname = "example.com"
					}
				}

				resource "checkly_icmp_monitor" "test2" {
					name      = "test-icmp-group-assignment"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
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
					request = {
						hostname = "example.com"
					}
				}

				resource "checkly_icmp_monitor" "test2" {
					name      = "test-icmp-group-assignment"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
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
					request = {
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
					request = {
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
					request = {
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
					request = {
						hostname = "example.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`Attribute frequency value must be one of`),
		},
		{
			Config: `
				resource "checkly_icmp_monitor" "test-frequency-ok-60" {
					name             = "test-frequency-ok-60"
					activated        = true
					frequency        = 60
					frequency_offset = 0
					locations        = ["eu-central-1"]
					request = {
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
					request = {
						hostname = "example.com"
					}
				}

				resource "checkly_icmp_monitor" "test2" {
					name             = "test-high-frequency"
					activated        = true
					frequency        = 0
					frequency_offset = 30
					locations        = ["eu-central-1"]
					request = {
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
				resource.TestCheckNoResourceAttr(
					"checkly_icmp_monitor.test1",
					"frequency_offset",
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
					request = {
						hostname = "example.com"
					}
				}

				resource "checkly_icmp_monitor" "test2" {
					name      = "test-high-frequency-updated-to-normal"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
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
				resource.TestCheckNoResourceAttr(
					"checkly_icmp_monitor.test2",
					"frequency_offset",
				),
			),
		},
	})
}

func TestAccICMPMonitorAlertChannelSubscriptions(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		// Step 1: Add two alert channel subscriptions.
		{
			Config: `
				resource "checkly_alert_channel" "email_1" {
					email = {
						address = "icmp-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email = {
						address = "icmp-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email = {
						address = "icmp-acs-3@example.com"
					}
				}

				resource "checkly_icmp_monitor" "test" {
					name      = "icmp-alert-channel-sub-test"
					frequency = 60
					activated = true
					locations = ["eu-central-1"]

					request = {
						hostname = "example.com"

						assertions = [{
							source     = "LATENCY"
							property   = "avg"
							comparison = "LESS_THAN"
							target     = "200"
						}]
					}

					alert_channel_subscriptions = [
						{
							channel_id = checkly_alert_channel.email_1.id
							activated  = true
						},
						{
							channel_id = checkly_alert_channel.email_2.id
							activated  = false
						},
					]
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test",
					"alert_channel_subscriptions.#",
					"2",
				),
				testCheckAlertChannelSubscription(
					"checkly_icmp_monitor.test",
					"alert_channel_subscriptions",
					"checkly_alert_channel.email_1",
					"true",
				),
				testCheckAlertChannelSubscription(
					"checkly_icmp_monitor.test",
					"alert_channel_subscriptions",
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
						address = "icmp-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email = {
						address = "icmp-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email = {
						address = "icmp-acs-3@example.com"
					}
				}

				resource "checkly_icmp_monitor" "test" {
					name      = "icmp-alert-channel-sub-test"
					frequency = 60
					activated = true
					locations = ["eu-central-1"]

					request = {
						hostname = "example.com"

						assertions = [{
							source     = "LATENCY"
							property   = "avg"
							comparison = "LESS_THAN"
							target     = "200"
						}]
					}

					alert_channel_subscriptions = [
						{
							channel_id = checkly_alert_channel.email_1.id
							activated  = false
						},
						{
							channel_id = checkly_alert_channel.email_2.id
							activated  = true
						},
						{
							channel_id = checkly_alert_channel.email_3.id
							activated  = true
						},
					]
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test",
					"alert_channel_subscriptions.#",
					"3",
				),
				testCheckAlertChannelSubscription(
					"checkly_icmp_monitor.test",
					"alert_channel_subscriptions",
					"checkly_alert_channel.email_1",
					"false",
				),
				testCheckAlertChannelSubscription(
					"checkly_icmp_monitor.test",
					"alert_channel_subscriptions",
					"checkly_alert_channel.email_2",
					"true",
				),
				testCheckAlertChannelSubscription(
					"checkly_icmp_monitor.test",
					"alert_channel_subscriptions",
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
						address = "icmp-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email = {
						address = "icmp-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email = {
						address = "icmp-acs-3@example.com"
					}
				}

				resource "checkly_icmp_monitor" "test" {
					name      = "icmp-alert-channel-sub-test"
					frequency = 60
					activated = true
					locations = ["eu-central-1"]

					request = {
						hostname = "example.com"

						assertions = [{
							source     = "LATENCY"
							property   = "avg"
							comparison = "LESS_THAN"
							target     = "200"
						}]
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test",
					"alert_channel_subscriptions.#",
					"0",
				),
			),
		},
	})
}

// TestAccICMPMonitorUnsetSweepBasics sets every plain optional attribute to a
// non-default value and then removes them all, verifying the server actually
// clears (or resets to the default) each value. The Checkly PUT endpoint is
// PATCH-style, so a key omitted from the update payload keeps its server-side
// value; the implicit post-apply refresh would surface that as a non-empty
// plan and fail the test.
func TestAccICMPMonitorUnsetSweepBasics(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_icmp_monitor" "test" {
					name                           = "icmp-unset-sweep-basics"
					activated                      = true
					frequency                      = 60
					locations                      = ["eu-central-1"]
					muted                          = true
					run_parallel                   = true
					use_global_alert_settings      = false
					degraded_packet_loss_threshold = 35
					max_packet_loss_threshold      = 70
					tags                           = ["sweep-a", "sweep-b"]
					request = {
						hostname = "api.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_icmp_monitor.test", "muted", "true"),
				resource.TestCheckResourceAttr("checkly_icmp_monitor.test", "run_parallel", "true"),
				resource.TestCheckResourceAttr("checkly_icmp_monitor.test", "use_global_alert_settings", "false"),
				resource.TestCheckResourceAttr("checkly_icmp_monitor.test", "degraded_packet_loss_threshold", "35"),
				resource.TestCheckResourceAttr("checkly_icmp_monitor.test", "max_packet_loss_threshold", "70"),
				resource.TestCheckResourceAttr("checkly_icmp_monitor.test", "tags.#", "2"),
			),
		},
		{
			Config: `
				resource "checkly_icmp_monitor" "test" {
					name      = "icmp-unset-sweep-basics"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						hostname = "api.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_icmp_monitor.test", "muted", "false"),
				resource.TestCheckResourceAttr("checkly_icmp_monitor.test", "run_parallel", "false"),
				resource.TestCheckResourceAttr("checkly_icmp_monitor.test", "use_global_alert_settings", "true"),
				resource.TestCheckResourceAttr("checkly_icmp_monitor.test", "degraded_packet_loss_threshold", "10"),
				resource.TestCheckResourceAttr("checkly_icmp_monitor.test", "max_packet_loss_threshold", "20"),
				resource.TestCheckNoResourceAttr("checkly_icmp_monitor.test", "tags.#"),
			),
		},
	})
}

// TestAccICMPMonitorUnsetSweepThresholdZero verifies that a packet loss
// threshold of 0 is rejected at plan time. The Checkly API rejects an
// explicit 0 for both thresholds even though its public schema allows it.
func TestAccICMPMonitorUnsetSweepThresholdZero(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_icmp_monitor" "test" {
					name                           = "icmp-unset-sweep-threshold-zero"
					activated                      = true
					frequency                      = 60
					locations                      = ["eu-central-1"]
					degraded_packet_loss_threshold = 0
					max_packet_loss_threshold      = 0
					request = {
						hostname = "api.checklyhq.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`value must be between\s+1\s+and\s+100`),
		},
	})
}

// TestAccICMPMonitorUnsetSweepThresholdReset verifies that the packet loss
// thresholds reset to their documented defaults when removed from the
// configuration.
func TestAccICMPMonitorUnsetSweepThresholdReset(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_icmp_monitor" "test" {
					name                           = "icmp-unset-sweep-threshold-reset"
					activated                      = true
					frequency                      = 60
					locations                      = ["eu-central-1"]
					degraded_packet_loss_threshold = 5
					max_packet_loss_threshold      = 15
					request = {
						hostname = "api.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_icmp_monitor.test", "degraded_packet_loss_threshold", "5"),
				resource.TestCheckResourceAttr("checkly_icmp_monitor.test", "max_packet_loss_threshold", "15"),
			),
		},
		{
			Config: `
				resource "checkly_icmp_monitor" "test" {
					name      = "icmp-unset-sweep-threshold-reset"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						hostname = "api.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_icmp_monitor.test", "degraded_packet_loss_threshold", "10"),
				resource.TestCheckResourceAttr("checkly_icmp_monitor.test", "max_packet_loss_threshold", "20"),
			),
		},
	})
}

// TestAccICMPMonitorUnsetSweepLocations verifies that removing the locations
// attribute entirely clears the locations on the server.
func TestAccICMPMonitorUnsetSweepLocations(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_icmp_monitor" "test" {
					name      = "icmp-unset-sweep-locations"
					activated = true
					frequency = 60
					locations = ["eu-central-1", "us-east-1"]
					request = {
						hostname = "api.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_icmp_monitor.test", "locations.#", "2"),
			),
		},
		{
			Config: `
				resource "checkly_icmp_monitor" "test" {
					name      = "icmp-unset-sweep-locations"
					activated = true
					frequency = 60
					request = {
						hostname = "api.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr("checkly_icmp_monitor.test", "locations.#"),
			),
		},
	})
}

// TestAccICMPMonitorUnsetSweepRequest sets the optional request attributes to
// non-default values plus an assertion, then removes them, verifying the
// attributes reset to their defaults and the assertions are cleared on the
// server.
func TestAccICMPMonitorUnsetSweepRequest(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_icmp_monitor" "test" {
					name      = "icmp-unset-sweep-request"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						hostname   = "api.checklyhq.com"
						ip_family  = "IPv6"
						ping_count = 5
						assertions = [{
							source     = "LATENCY"
							property   = "avg"
							comparison = "LESS_THAN"
							target     = "250"
						}]
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_icmp_monitor.test", "request.ip_family", "IPv6"),
				resource.TestCheckResourceAttr("checkly_icmp_monitor.test", "request.ping_count", "5"),
				resource.TestCheckResourceAttr("checkly_icmp_monitor.test", "request.assertions.#", "1"),
			),
		},
		{
			Config: `
				resource "checkly_icmp_monitor" "test" {
					name      = "icmp-unset-sweep-request"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						hostname = "api.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_icmp_monitor.test", "request.ip_family", "IPv4"),
				resource.TestCheckResourceAttr("checkly_icmp_monitor.test", "request.ping_count", "10"),
				resource.TestCheckNoResourceAttr("checkly_icmp_monitor.test", "request.assertions"),
			),
		},
	})
}

// TestAccICMPMonitorUnsetSweepTriggerIncident verifies that removing the
// trigger_incident attribute clears the incident trigger on the server.
func TestAccICMPMonitorUnsetSweepTriggerIncident(t *testing.T) {
	service := `
		resource "checkly_status_page_service" "icmp_sweep" {
			name = "ICMP Monitor Unset Sweep Service"
		}
	`
	accTestCase(t, []resource.TestStep{
		{
			Config: service + `
				resource "checkly_icmp_monitor" "test" {
					name      = "icmp-unset-sweep-trigger-incident"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						hostname = "api.checklyhq.com"
					}
					trigger_incident = {
						service_id         = checkly_status_page_service.icmp_sweep.id
						severity           = "MEDIUM"
						name               = "ICMP sweep incident"
						description        = "Incident created by the unset sweep test"
						notify_subscribers = false
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_icmp_monitor.test", "trigger_incident.%", "5"),
			),
		},
		{
			Config: service + `
				resource "checkly_icmp_monitor" "test" {
					name      = "icmp-unset-sweep-trigger-incident"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						hostname = "api.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr("checkly_icmp_monitor.test", "trigger_incident"),
			),
		},
	})
}

// TestAccICMPMonitorUnsetSweepGroup verifies that removing group_id and
// group_order detaches the monitor from the group and resets the order.
func TestAccICMPMonitorUnsetSweepGroup(t *testing.T) {
	group := `
		resource "checkly_check_group" "icmp_sweep" {
			name        = "icmp-unset-sweep-group"
			activated   = true
			concurrency = 1
			locations   = ["eu-central-1"]
		}
	`
	accTestCase(t, []resource.TestStep{
		{
			Config: group + `
				resource "checkly_icmp_monitor" "test" {
					name        = "icmp-unset-sweep-group"
					activated   = true
					frequency   = 60
					locations   = ["eu-central-1"]
					group_id    = checkly_check_group.icmp_sweep.id
					group_order = 2
					request = {
						hostname = "api.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrPair(
					"checkly_icmp_monitor.test",
					"group_id",
					"checkly_check_group.icmp_sweep",
					"id",
				),
				resource.TestCheckResourceAttr("checkly_icmp_monitor.test", "group_order", "2"),
			),
		},
		{
			Config: group + `
				resource "checkly_icmp_monitor" "test" {
					name      = "icmp-unset-sweep-group"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						hostname = "api.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_icmp_monitor.test", "group_id", "0"),
				resource.TestCheckResourceAttr("checkly_icmp_monitor.test", "group_order", "0"),
			),
		},
	})
}

// TestAccICMPMonitorRetryStrategyRemoval verifies that removing the
// retry_strategy attribute resets the strategy to NO_RETRIES on the server.
func TestAccICMPMonitorRetryStrategyRemoval(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_icmp_monitor" "test" {
					name      = "icmp-retry-strategy-removal"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						hostname = "api.checklyhq.com"
					}
					retry_strategy = {
						type = "LINEAR"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test",
					"retry_strategy.type",
					"LINEAR",
				),
			),
		},
		{
			Config: `
				resource "checkly_icmp_monitor" "test" {
					name      = "icmp-retry-strategy-removal"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						hostname = "api.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_icmp_monitor.test",
					"retry_strategy.type",
					"NO_RETRIES",
				),
			),
		},
	})
}

func TestAccICMPMonitorPrivateLocations(t *testing.T) {
	testAccMonitorPrivateLocations(t, monitorPrivateLocationsCase{
		resource: "checkly_icmp_monitor.test",
		config: func(name, attributes string) string {
			return fmt.Sprintf(`
				resource "checkly_icmp_monitor" "test" {
					name                      = %q
					frequency                 = 60
					activated                 = false
					locations                 = ["eu-central-1"]
					use_global_alert_settings = true

					request = {
						hostname = "example.com"
					}
					%s
				}
			`, name, attributes)
		},
		set: func(ctx context.Context, client checkly.Client, id string, privateLocations []string) error {
			monitor, err := client.GetICMPMonitor(ctx, id)
			if err != nil {
				return err
			}
			monitor.PrivateLocations = &privateLocations
			_, err = client.UpdateICMPMonitor(ctx, id, *monitor)
			return err
		},
		read: func(ctx context.Context, client checkly.Client, id string) (*[]string, error) {
			monitor, err := client.GetICMPMonitor(ctx, id)
			if err != nil {
				return nil, err
			}
			return monitor.PrivateLocations, nil
		},
	})
}
