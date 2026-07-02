package provider

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/checkly/checkly-go-sdk"
)

// icmpMonitorTestCheckAlertChannelSubscription checks that the resource has
// an alert_channel_subscription for the channel created by channelIDAttr
// with the expected activation state.
func icmpMonitorTestCheckAlertChannelSubscription(resourceName, blockPath string, channelIDAttr string, expectedActivated string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		// Resolve the channel_id from another resource's attribute
		channelRS, ok := s.RootModule().Resources[channelIDAttr]
		if !ok {
			return fmt.Errorf("resource not found: %s", channelIDAttr)
		}
		wantChannelID := channelRS.Primary.Attributes["id"]

		count := rs.Primary.Attributes[blockPath+".#"]
		n, _ := strconv.Atoi(count)
		for i := range n {
			prefix := fmt.Sprintf("%s.%d", blockPath, i)
			if rs.Primary.Attributes[prefix+".channel_id"] == wantChannelID {
				got := rs.Primary.Attributes[prefix+".activated"]
				if got != expectedActivated {
					return fmt.Errorf("channel %s: expected activated=%s, got %s", wantChannelID, expectedActivated, got)
				}
				return nil
			}
		}

		return fmt.Errorf("no alert_channel_subscription with channel_id=%s found", wantChannelID)
	}
}

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
			// Terraform only reports the missing request block once all
			// required attributes are present.
			Config: `
				resource "checkly_icmp_monitor" "test" {
					name      = "test-required-fields"
					activated = true
					frequency = 60
				}
			`,
			ExpectError: regexp.MustCompile(`Block request must have a configuration value`),
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
	RetryStrategy:          &checkly.RetryStrategy{Type: "NO_RETRIES"},
	Request: checkly.ICMPRequest{
		Hostname:  "example.com",
		IPFamily:  "IPv4",
		PingCount: 10,
		Assertions: []checkly.Assertion{
			{
				Source:     "LATENCY",
				Property:   "avg",
				Comparison: checkly.LessThan,
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
					request {
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
					request {
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
					request {
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
					email {
						address = "icmp-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email {
						address = "icmp-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email {
						address = "icmp-acs-3@example.com"
					}
				}

				resource "checkly_icmp_monitor" "test" {
					name      = "icmp-alert-channel-sub-test"
					frequency = 60
					activated = true
					locations = ["eu-central-1"]

					request {
						hostname = "example.com"

						assertion {
							source     = "LATENCY"
							property   = "avg"
							comparison = "LESS_THAN"
							target     = "200"
						}
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
					"checkly_icmp_monitor.test",
					"alert_channel_subscription.#",
					"2",
				),
				icmpMonitorTestCheckAlertChannelSubscription(
					"checkly_icmp_monitor.test",
					"alert_channel_subscription",
					"checkly_alert_channel.email_1",
					"true",
				),
				icmpMonitorTestCheckAlertChannelSubscription(
					"checkly_icmp_monitor.test",
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
					email {
						address = "icmp-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email {
						address = "icmp-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email {
						address = "icmp-acs-3@example.com"
					}
				}

				resource "checkly_icmp_monitor" "test" {
					name      = "icmp-alert-channel-sub-test"
					frequency = 60
					activated = true
					locations = ["eu-central-1"]

					request {
						hostname = "example.com"

						assertion {
							source     = "LATENCY"
							property   = "avg"
							comparison = "LESS_THAN"
							target     = "200"
						}
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
					"checkly_icmp_monitor.test",
					"alert_channel_subscription.#",
					"3",
				),
				icmpMonitorTestCheckAlertChannelSubscription(
					"checkly_icmp_monitor.test",
					"alert_channel_subscription",
					"checkly_alert_channel.email_1",
					"false",
				),
				icmpMonitorTestCheckAlertChannelSubscription(
					"checkly_icmp_monitor.test",
					"alert_channel_subscription",
					"checkly_alert_channel.email_2",
					"true",
				),
				icmpMonitorTestCheckAlertChannelSubscription(
					"checkly_icmp_monitor.test",
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
					email {
						address = "icmp-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email {
						address = "icmp-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email {
						address = "icmp-acs-3@example.com"
					}
				}

				resource "checkly_icmp_monitor" "test" {
					name      = "icmp-alert-channel-sub-test"
					frequency = 60
					activated = true
					locations = ["eu-central-1"]

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
					"alert_channel_subscription.#",
					"0",
				),
			),
		},
	})
}
