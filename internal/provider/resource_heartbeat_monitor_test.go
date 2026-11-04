package provider

import (
	"context"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccHeartbeatMonitorRequiredFields(t *testing.T) {
	config := `resource "checkly_heartbeat_monitor" "test" {}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "name" is required, but no definition was found.`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "activated" is required, but no definition was found.`),
		},
	})
}

func TestAccHeartbeatMonitorInvalidInputs(t *testing.T) {
	config := `resource "checkly_heartbeat_monitor" "test" {
		name                      = 1
		activated                 = "invalid"
		use_global_alert_settings = "invalid"
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`Inappropriate value for attribute "activated"`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`Inappropriate value for attribute "use_global_alert_settings"`),
		},
	})
}

func TestAccHeartbeatMonitorMissingHeartbeat(t *testing.T) {
	config := `resource "checkly_heartbeat_monitor" "test" {
		activated = true
		name = "heartbeat monitor"
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "heartbeat" is required, but no definition was found.`),
		},
	})
}

func TestAccHeartbeatMonitorMissingHeartbeatFields(t *testing.T) {
	config := `resource "checkly_heartbeat_monitor" "test" {
		activated = true
		name = "heartbeat monitor"
		heartbeat = {

		}
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`attributes "grace",\s+"grace_unit", "period", and "period_unit" are required`),
		},
	})
}

func TestAccHeartbeatMonitorPeriodTooBig(t *testing.T) {
	config := `resource "checkly_heartbeat_monitor" "test" {
		activated = true
		name = "heartbeat monitor"
		heartbeat = {
			period = 366
			period_unit = "days"
			grace = 0
			grace_unit = "seconds"
		}
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`period must be between 30 seconds and 365 days`),
		},
	})
}

func TestAccHeartbeatMonitorPeriodTooSmall(t *testing.T) {
	config := `resource "checkly_heartbeat_monitor" "test" {
		activated = true
		name = "heartbeat monitor"
		heartbeat = {
			period = 5
			period_unit = "seconds"
			grace = 0
			grace_unit = "seconds"
		}
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`period must be between 30 seconds and 365 days`),
		},
	})
}

func TestAccHeartbeatMonitorGraceTooBig(t *testing.T) {
	config := `resource "checkly_heartbeat_monitor" "test" {
		activated = true
		name = "heartbeat monitor"
		heartbeat = {
			period = 5
			period_unit = "days"
			grace = 366
			grace_unit = "days"
		}
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`grace must be less than 365 days`),
		},
	})
}

func TestAccHeartbeatMonitorInvalidPeriodUnit(t *testing.T) {
	config := `resource "checkly_heartbeat_monitor" "test" {
		activated = true
		name = "heartbeat monitor"
		heartbeat = {
			period = 5
			period_unit = "lightyear"
			grace = 0
			grace_unit = "seconds"
		}
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`Attribute heartbeat\.period_unit value must be one of`),
		},
	})
}

func TestAccHeartbeatMonitorInvalidGraceUnit(t *testing.T) {
	config := `resource "checkly_heartbeat_monitor" "test" {
		activated = true
		name = "heartbeat monitor"
		heartbeat = {
			period = 5
			period_unit = "days"
			grace = 0
			grace_unit = "lightyear"
		}
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`Attribute heartbeat\.grace_unit value must be one of`),
		},
	})
}

func TestAccHeartbeatMonitorCreate(t *testing.T) {
	config := `resource "checkly_heartbeat_monitor" "test" {
		activated = true
		name = "heartbeat monitor"
		description = "Heartbeat monitor description"
		heartbeat = {
			period = 5
			period_unit = "days"
			grace = 0
			grace_unit = "seconds"
		}
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config: config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_heartbeat_monitor.test",
					"name",
					"heartbeat monitor",
				),
				resource.TestCheckResourceAttr(
					"checkly_heartbeat_monitor.test",
					"description",
					"Heartbeat monitor description",
				),
				resource.TestCheckResourceAttr(
					"checkly_heartbeat_monitor.test",
					"heartbeat.period",
					"5",
				),
				resource.TestCheckResourceAttr(
					"checkly_heartbeat_monitor.test",
					"heartbeat.period_unit",
					"days",
				),
				resource.TestCheckResourceAttrSet(
					"checkly_heartbeat_monitor.test",
					"heartbeat.ping_token",
				),
			),
		},
	})
}

func TestAccHeartbeatMonitorDescriptionRemoval(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `resource "checkly_heartbeat_monitor" "test" {
				activated = true
				name = "heartbeat monitor"
				description = "Heartbeat monitor description"
				heartbeat = {
					period = 5
					period_unit = "days"
					grace = 0
					grace_unit = "seconds"
				}
			}`,
			Check: resource.TestCheckResourceAttr(
				"checkly_heartbeat_monitor.test",
				"description",
				"Heartbeat monitor description",
			),
		},
		{
			Config: `resource "checkly_heartbeat_monitor" "test" {
				activated = true
				name = "heartbeat monitor"
				heartbeat = {
					period = 5
					period_unit = "days"
					grace = 0
					grace_unit = "seconds"
				}
			}`,
			Check: resource.TestCheckResourceAttr(
				"checkly_heartbeat_monitor.test",
				"description",
				"",
			),
		},
	})
}

func TestAccHeartbeatMonitorWithTriggerIncident(t *testing.T) {
	heartbeatMonitorWithTriggerIncident := `
resource "checkly_status_page_service" "test_heartbeat_service" {
	name = "Heartbeat Test Service"
}

resource "checkly_heartbeat_monitor" "test_trigger_incident" {
	name      = "Heartbeat Monitor with Trigger Incident"
	activated = true

	heartbeat = {
		period      = 10
		period_unit = "minutes"
		grace       = 5
		grace_unit  = "minutes"
	}

	trigger_incident = {
		service_id         = checkly_status_page_service.test_heartbeat_service.id
		severity           = "MEDIUM"
		name               = "Heartbeat Monitor Failure"
		description        = "The heartbeat monitor has not received a ping within the grace period"
		notify_subscribers = true
	}

	use_global_alert_settings = true
}
`
	accTestCase(t, []resource.TestStep{
		{
			Config: heartbeatMonitorWithTriggerIncident,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_heartbeat_monitor.test_trigger_incident",
					"name",
					"Heartbeat Monitor with Trigger Incident",
				),
				resource.TestCheckResourceAttr(
					"checkly_heartbeat_monitor.test_trigger_incident",
					"activated",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_heartbeat_monitor.test_trigger_incident",
					"trigger_incident.severity",
					"MEDIUM",
				),
				resource.TestCheckResourceAttr(
					"checkly_heartbeat_monitor.test_trigger_incident",
					"trigger_incident.name",
					"Heartbeat Monitor Failure",
				),
				resource.TestCheckResourceAttr(
					"checkly_heartbeat_monitor.test_trigger_incident",
					"trigger_incident.description",
					"The heartbeat monitor has not received a ping within the grace period",
				),
				resource.TestCheckResourceAttr(
					"checkly_heartbeat_monitor.test_trigger_incident",
					"trigger_incident.notify_subscribers",
					"true",
				),
			),
		},
	})
}

// The Checkly PUT endpoints are PATCH-style and have a history of silently
// keeping properties that are omitted from update payloads. The unset sweep
// tests below set every feasibly testable optional property to a non-default
// value and then remove it from the configuration, verifying that the server
// actually clears (or resets) the value. The framework's implicit post-apply
// refresh and empty-plan checks detect any value the server kept.

func TestAccHeartbeatMonitorUnsetSweepBooleansAndTags(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_heartbeat_monitor" "test" {
					name                      = "heartbeat-unset-sweep-basics"
					activated                 = true
					muted                     = true
					use_global_alert_settings = true
					tags                      = ["sweep-a", "sweep-b"]
					heartbeat = {
						period      = 5
						period_unit = "days"
						grace       = 10
						grace_unit  = "minutes"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_heartbeat_monitor.test",
					"muted",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_heartbeat_monitor.test",
					"use_global_alert_settings",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_heartbeat_monitor.test",
					"tags.#",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_heartbeat_monitor.test",
					"heartbeat.grace",
					"10",
				),
			),
		},
		// The optional attributes are removed, and grace (a required
		// attribute) is reset to 0 to verify that an explicit zero value is
		// actually persisted.
		{
			Config: `
				resource "checkly_heartbeat_monitor" "test" {
					name      = "heartbeat-unset-sweep-basics"
					activated = true
					heartbeat = {
						period      = 5
						period_unit = "days"
						grace       = 0
						grace_unit  = "seconds"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_heartbeat_monitor.test",
					"muted",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_heartbeat_monitor.test",
					"use_global_alert_settings",
					"false",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_heartbeat_monitor.test",
					"tags.#",
				),
				resource.TestCheckResourceAttr(
					"checkly_heartbeat_monitor.test",
					"heartbeat.grace",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_heartbeat_monitor.test",
					"heartbeat.grace_unit",
					"seconds",
				),
			),
		},
	})
}

func TestAccHeartbeatMonitorUnsetSweepTriggerIncident(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_status_page_service" "test" {
					name = "heartbeat-unset-sweep-service"
				}

				resource "checkly_heartbeat_monitor" "test" {
					name      = "heartbeat-unset-sweep-trigger-incident"
					activated = true
					heartbeat = {
						period      = 5
						period_unit = "days"
						grace       = 0
						grace_unit  = "seconds"
					}
					trigger_incident = {
						service_id         = checkly_status_page_service.test.id
						severity           = "MINOR"
						name               = "Heartbeat sweep incident"
						description        = "Incident trigger for the unset sweep"
						notify_subscribers = false
					}
				}
			`,
			Check: resource.TestCheckResourceAttr(
				"checkly_heartbeat_monitor.test",
				"trigger_incident.%",
				"5",
			),
		},
		{
			Config: `
				resource "checkly_status_page_service" "test" {
					name = "heartbeat-unset-sweep-service"
				}

				resource "checkly_heartbeat_monitor" "test" {
					name      = "heartbeat-unset-sweep-trigger-incident"
					activated = true
					heartbeat = {
						period      = 5
						period_unit = "days"
						grace       = 0
						grace_unit  = "seconds"
					}
				}
			`,
			Check: resource.TestCheckNoResourceAttr(
				"checkly_heartbeat_monitor.test",
				"trigger_incident",
			),
		},
	})
}

func TestAccHeartbeatMonitorAlertChannelSubscriptions(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		// Step 1: Add two alert channel subscriptions.
		{
			Config: `
				resource "checkly_alert_channel" "email_1" {
					email = {
						address = "heartbeat-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email = {
						address = "heartbeat-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email = {
						address = "heartbeat-acs-3@example.com"
					}
				}

				resource "checkly_heartbeat_monitor" "test" {
					name      = "heartbeat-alert-channel-sub-test"
					activated = true

					heartbeat = {
						period      = 5
						period_unit = "days"
						grace       = 0
						grace_unit  = "seconds"
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
					"checkly_heartbeat_monitor.test",
					"alert_channel_subscription.#",
					"2",
				),
				testCheckAlertChannelSubscription(
					"checkly_heartbeat_monitor.test",
					"alert_channel_subscription",
					"checkly_alert_channel.email_1",
					"true",
				),
				testCheckAlertChannelSubscription(
					"checkly_heartbeat_monitor.test",
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
						address = "heartbeat-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email = {
						address = "heartbeat-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email = {
						address = "heartbeat-acs-3@example.com"
					}
				}

				resource "checkly_heartbeat_monitor" "test" {
					name      = "heartbeat-alert-channel-sub-test"
					activated = true

					heartbeat = {
						period      = 5
						period_unit = "days"
						grace       = 0
						grace_unit  = "seconds"
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
					"checkly_heartbeat_monitor.test",
					"alert_channel_subscription.#",
					"3",
				),
				testCheckAlertChannelSubscription(
					"checkly_heartbeat_monitor.test",
					"alert_channel_subscription",
					"checkly_alert_channel.email_1",
					"false",
				),
				testCheckAlertChannelSubscription(
					"checkly_heartbeat_monitor.test",
					"alert_channel_subscription",
					"checkly_alert_channel.email_2",
					"true",
				),
				testCheckAlertChannelSubscription(
					"checkly_heartbeat_monitor.test",
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
						address = "heartbeat-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email = {
						address = "heartbeat-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email = {
						address = "heartbeat-acs-3@example.com"
					}
				}

				resource "checkly_heartbeat_monitor" "test" {
					name      = "heartbeat-alert-channel-sub-test"
					activated = true

					heartbeat = {
						period      = 5
						period_unit = "days"
						grace       = 0
						grace_unit  = "seconds"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_heartbeat_monitor.test",
					"alert_channel_subscription.#",
					"0",
				),
			),
		},
	})
}

func TestHeartbeatMonitorDurationValidator(t *testing.T) {
	heartbeat := func(period types.Int64, periodUnit types.String, grace types.Int64, graceUnit types.String) types.Object {
		return types.ObjectValueMust(heartbeatMonitorHeartbeatAttrTypes, map[string]attr.Value{
			"period":      period,
			"period_unit": periodUnit,
			"grace":       grace,
			"grace_unit":  graceUnit,
			"ping_token":  types.StringNull(),
		})
	}

	tests := map[string]struct {
		value     types.Object
		wantError string
	}{
		"null object": {value: types.ObjectNull(heartbeatMonitorHeartbeatAttrTypes)},
		// Terraform rejects an empty heartbeat object before provider
		// validation runs; with every member null there is nothing to check.
		"all members null": {value: heartbeat(types.Int64Null(), types.StringNull(), types.Int64Null(), types.StringNull())},
		"unknown period":   {value: heartbeat(types.Int64Unknown(), types.StringValue("seconds"), types.Int64Value(1), types.StringValue("hours"))},
		"valid":            {value: heartbeat(types.Int64Value(1), types.StringValue("days"), types.Int64Value(1), types.StringValue("hours"))},
		"period too short": {value: heartbeat(types.Int64Value(29), types.StringValue("seconds"), types.Int64Value(1), types.StringValue("hours")), wantError: "period must be between 30 seconds and 365 days"},
		"period too long":  {value: heartbeat(types.Int64Value(366), types.StringValue("days"), types.Int64Value(1), types.StringValue("hours")), wantError: "period must be between 30 seconds and 365 days"},
		"grace too long":   {value: heartbeat(types.Int64Value(1), types.StringValue("days"), types.Int64Value(366), types.StringValue("days")), wantError: "grace must be less than 365 days"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			req := validator.ObjectRequest{
				Path:        path.Root("heartbeat"),
				ConfigValue: tt.value,
			}
			resp := &validator.ObjectResponse{}

			heartbeatMonitorDurationValidator{}.ValidateObject(context.Background(), req, resp)

			if tt.wantError == "" {
				if resp.Diagnostics.HasError() {
					t.Fatalf("unexpected error: %v", resp.Diagnostics)
				}
				return
			}
			if resp.Diagnostics.ErrorsCount() != 1 || resp.Diagnostics.Errors()[0].Detail() != tt.wantError {
				t.Fatalf("expected error %q, got %v", tt.wantError, resp.Diagnostics)
			}
		})
	}
}
