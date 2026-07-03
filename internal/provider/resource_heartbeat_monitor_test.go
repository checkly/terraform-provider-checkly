package provider

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// heartbeatMonitorTestCheckAlertChannelSubscription checks that the resource
// has an alert channel subscription for the channel created by the resource
// named by channelResourceName, with the expected activation state.
func heartbeatMonitorTestCheckAlertChannelSubscription(resourceName, channelResourceName, expectedActivated string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		channelRS, ok := s.RootModule().Resources[channelResourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", channelResourceName)
		}
		wantChannelID := channelRS.Primary.Attributes["id"]

		count := rs.Primary.Attributes["alert_channel_subscription.#"]
		n, _ := strconv.Atoi(count)
		for i := range n {
			prefix := fmt.Sprintf("alert_channel_subscription.%d", i)
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

func TestAccHeartbeatMonitorMissingHeartbeatBlock(t *testing.T) {
	config := `resource "checkly_heartbeat_monitor" "test" {
		activated = true
		name = "heartbeat monitor"
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`Block heartbeat must have a configuration value`),
		},
	})
}

func TestAccHeartbeatMonitorMissingHeartbeatFields(t *testing.T) {
	config := `resource "checkly_heartbeat_monitor" "test" {
		activated = true
		name = "heartbeat monitor"
		heartbeat {

		}
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "grace" is required, but no definition was found.`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "grace_unit" is required, but no definition was found.`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "period" is required, but no definition was found.`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "period_unit" is required, but no definition was found.`),
		},
	})
}

func TestAccHeartbeatMonitorPeriodTooBig(t *testing.T) {
	config := `resource "checkly_heartbeat_monitor" "test" {
		activated = true
		name = "heartbeat monitor"
		heartbeat {
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
		heartbeat {
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
		heartbeat {
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
		heartbeat {
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
		heartbeat {
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
		heartbeat {
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
				heartbeat {
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
				heartbeat {
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

	heartbeat {
		period      = 10
		period_unit = "minutes"
		grace       = 5
		grace_unit  = "minutes"
	}

	trigger_incident {
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
					"trigger_incident.0.severity",
					"MEDIUM",
				),
				resource.TestCheckResourceAttr(
					"checkly_heartbeat_monitor.test_trigger_incident",
					"trigger_incident.0.name",
					"Heartbeat Monitor Failure",
				),
				resource.TestCheckResourceAttr(
					"checkly_heartbeat_monitor.test_trigger_incident",
					"trigger_incident.0.description",
					"The heartbeat monitor has not received a ping within the grace period",
				),
				resource.TestCheckResourceAttr(
					"checkly_heartbeat_monitor.test_trigger_incident",
					"trigger_incident.0.notify_subscribers",
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
					heartbeat {
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
					heartbeat {
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
					heartbeat {
						period      = 5
						period_unit = "days"
						grace       = 0
						grace_unit  = "seconds"
					}
					trigger_incident {
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
				"trigger_incident.#",
				"1",
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
					heartbeat {
						period      = 5
						period_unit = "days"
						grace       = 0
						grace_unit  = "seconds"
					}
				}
			`,
			Check: resource.TestCheckResourceAttr(
				"checkly_heartbeat_monitor.test",
				"trigger_incident.#",
				"0",
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
					email {
						address = "heartbeat-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email {
						address = "heartbeat-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email {
						address = "heartbeat-acs-3@example.com"
					}
				}

				resource "checkly_heartbeat_monitor" "test" {
					name      = "heartbeat-alert-channel-sub-test"
					activated = true

					heartbeat {
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
				heartbeatMonitorTestCheckAlertChannelSubscription(
					"checkly_heartbeat_monitor.test",
					"checkly_alert_channel.email_1",
					"true",
				),
				heartbeatMonitorTestCheckAlertChannelSubscription(
					"checkly_heartbeat_monitor.test",
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
						address = "heartbeat-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email {
						address = "heartbeat-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email {
						address = "heartbeat-acs-3@example.com"
					}
				}

				resource "checkly_heartbeat_monitor" "test" {
					name      = "heartbeat-alert-channel-sub-test"
					activated = true

					heartbeat {
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
				heartbeatMonitorTestCheckAlertChannelSubscription(
					"checkly_heartbeat_monitor.test",
					"checkly_alert_channel.email_1",
					"false",
				),
				heartbeatMonitorTestCheckAlertChannelSubscription(
					"checkly_heartbeat_monitor.test",
					"checkly_alert_channel.email_2",
					"true",
				),
				heartbeatMonitorTestCheckAlertChannelSubscription(
					"checkly_heartbeat_monitor.test",
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
						address = "heartbeat-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email {
						address = "heartbeat-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email {
						address = "heartbeat-acs-3@example.com"
					}
				}

				resource "checkly_heartbeat_monitor" "test" {
					name      = "heartbeat-alert-channel-sub-test"
					activated = true

					heartbeat {
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
