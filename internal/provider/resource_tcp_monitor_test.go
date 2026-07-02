package provider

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// tcpMonitorTestCheckAlertChannelSubscription checks that the resource has an
// alert channel subscription for the channel created by the resource named by
// channelResourceName, with the expected activation state.
func tcpMonitorTestCheckAlertChannelSubscription(resourceName, channelResourceName, expectedActivated string) resource.TestCheckFunc {
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
			Config: `resource "checkly_tcp_monitor" "test" {
				name      = "tcp monitor"
				activated = true
				frequency = 60
			}`,
			ExpectError: regexp.MustCompile(`Block request must have a configuration value`),
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
					"description",
					"TCP monitor description",
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

func TestAccTCPMonitorDescriptionRemoval(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: tcpMonitor_basic,
			Check: resource.TestCheckResourceAttr(
				"checkly_tcp_monitor.test",
				"description",
				"TCP monitor description",
			),
		},
		{
			Config: tcpMonitor_basic_withoutDescription,
			Check: resource.TestCheckResourceAttr(
				"checkly_tcp_monitor.test",
				"description",
				"",
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
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"alert_settings.escalation_type",
					"RUN_BASED",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"alert_settings.run_based_escalation.failed_run_threshold",
					"1",
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

const tcpMonitor_basic = `
	resource "checkly_tcp_monitor" "test" {
	  name                      = "TCP Monitor 1"
	  description               = "TCP monitor description"
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

const tcpMonitor_basic_withoutDescription = `
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
					retry_strategy = {
						type                 = "SINGLE_RETRY"
						base_backoff_seconds = 30
						same_region          = true
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.type",
					"SINGLE_RETRY",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.base_backoff_seconds",
					"30",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.max_retries",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.max_duration_seconds",
					"600",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.same_region",
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
					retry_strategy = {
						type = "NO_RETRIES"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.type",
					"NO_RETRIES",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.base_backoff_seconds",
					"60",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.max_retries",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.max_duration_seconds",
					"600",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.same_region",
					"true",
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
					"retry_strategy.type",
					"NO_RETRIES",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.base_backoff_seconds",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.max_retries",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.max_duration_seconds",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.same_region",
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
					"checkly_tcp_monitor.test",
					"retry_strategy.type",
					"FIXED",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.base_backoff_seconds",
					"30",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.max_retries",
					"3",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.max_duration_seconds",
					"300",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.same_region",
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
					"checkly_tcp_monitor.test",
					"retry_strategy.type",
					"EXPONENTIAL",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.base_backoff_seconds",
					"60",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.max_retries",
					"5",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.max_duration_seconds",
					"600",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.same_region",
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
					"checkly_tcp_monitor.test",
					"retry_strategy.type",
					"LINEAR",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.base_backoff_seconds",
					"45",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.max_retries",
					"4",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.max_duration_seconds",
					"450",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.same_region",
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
					retry_strategy = {
						type = "LINEAR"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"retry_strategy.type",
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
					"retry_strategy.type",
					"NO_RETRIES",
				),
			),
		},
	})
}

func TestAccTCPMonitorGroupAssignment(t *testing.T) {
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

				resource "checkly_tcp_monitor" "test1" {
					name      = "test-group-assignment"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					group_id  = checkly_check_group.test1.id
					request {
						port     = 443
						hostname = "checkly.com"
					}
				}

				resource "checkly_tcp_monitor" "test2" {
					name      = "test-group-assignment"
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
				resource.TestCheckResourceAttrPair(
					"checkly_tcp_monitor.test1",
					"group_id",
					"checkly_check_group.test1",
					"id",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test2",
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

				resource "checkly_tcp_monitor" "test1" {
					name      = "test-group-assignment"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					group_id  = checkly_check_group.test2.id
					request {
						port     = 443
						hostname = "checkly.com"
					}
				}

				resource "checkly_tcp_monitor" "test2" {
					name      = "test-group-assignment"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					group_id  = checkly_check_group.test2.id
					request {
						port     = 443
						hostname = "checkly.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrPair(
					"checkly_tcp_monitor.test1",
					"group_id",
					"checkly_check_group.test2",
					"id",
				),
				resource.TestCheckResourceAttrPair(
					"checkly_tcp_monitor.test2",
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

				resource "checkly_tcp_monitor" "test1" {
					name      = "test-group-assignment"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						port     = 443
						hostname = "checkly.com"
					}
				}

				resource "checkly_tcp_monitor" "test2" {
					name      = "test-group-assignment"
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
					"checkly_tcp_monitor.test1",
					"group_id",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test2",
					"group_id",
					"0",
				),
			),
		},
	})
}

func TestAccTCPMonitorFrequencyValidation(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_tcp_monitor" "test-missing-frequency_offset" {
					name      = "test-missing-frequency_offset"
					activated = true
					frequency = 0
					locations = ["eu-central-1"]
					request {
						port     = 443
						hostname = "checkly.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`"frequency_offset" is required when "frequency" is 0`),
		},
		{
			Config: `
				resource "checkly_tcp_monitor" "test-frequency_offset-ok-10" {
					name             = "test-frequency_offset-ok-10"
					activated        = true
					frequency        = 0
					frequency_offset = 10
					locations        = ["eu-central-1"]
					request {
						port     = 443
						hostname = "checkly.com"
					}
				}

				resource "checkly_tcp_monitor" "test-frequency_offset-ok-20" {
					name             = "test-frequency_offset-ok-20"
					activated        = true
					frequency        = 0
					frequency_offset = 20
					locations        = ["eu-central-1"]
					request {
						port     = 443
						hostname = "checkly.com"
					}
				}

				resource "checkly_tcp_monitor" "test-frequency_offset-ok-30" {
					name             = "test-frequency_offset-ok-30"
					activated        = true
					frequency        = 0
					frequency_offset = 30
					locations        = ["eu-central-1"]
					request {
						port     = 443
						hostname = "checkly.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test-frequency_offset-ok-10",
					"frequency",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test-frequency_offset-ok-10",
					"frequency_offset",
					"10",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test-frequency_offset-ok-20",
					"frequency",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test-frequency_offset-ok-20",
					"frequency_offset",
					"20",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test-frequency_offset-ok-30",
					"frequency",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test-frequency_offset-ok-30",
					"frequency_offset",
					"30",
				),
			),
		},
		{
			Config: `
				resource "checkly_tcp_monitor" "test-extra-frequency_offset" {
					name             = "test-extra-frequency_offset"
					activated        = true
					frequency        = 60
					frequency_offset = 30
					locations        = ["eu-central-1"]
					request {
						port     = 443
						hostname = "checkly.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`"frequency_offset" can only be set when "frequency" is 0`),
		},
		{
			Config: `
				resource "checkly_tcp_monitor" "test-bad-frequency" {
					name      = "test-bad-frequency"
					activated = true
					frequency = 9999
					locations = ["eu-central-1"]
					request {
						port     = 443
						hostname = "checkly.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`Attribute frequency value must be one of`),
		},
		{
			Config: `
				resource "checkly_tcp_monitor" "test-bad-frequency_offset" {
					name             = "test-bad-frequency_offset"
					activated        = true
					frequency        = 60
					frequency_offset = 99
					locations        = ["eu-central-1"]
					request {
						port     = 443
						hostname = "checkly.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`Attribute frequency_offset value must be one of`),
		},
		{
			Config: `
				resource "checkly_tcp_monitor" "test-frequency-ok-60" {
					name             = "test-frequency-ok-60"
					activated        = true
					frequency        = 60
					frequency_offset = 0
					locations        = ["eu-central-1"]
					request {
						port     = 443
						hostname = "checkly.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test-frequency-ok-60",
					"frequency",
					"60",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test-frequency-ok-60",
					"frequency_offset",
					"0",
				),
			),
		},
	})
}

func TestAccTCPMonitorFrequencyUpdates(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_tcp_monitor" "test1" {
					name      = "test-normal-frequency"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						port     = 443
						hostname = "checkly.com"
					}
				}

				resource "checkly_tcp_monitor" "test2" {
					name             = "test-high-frequency"
					activated        = true
					frequency        = 0
					frequency_offset = 30
					locations        = ["eu-central-1"]
					request {
						port     = 443
						hostname = "checkly.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test1",
					"frequency",
					"60",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_tcp_monitor.test1",
					"frequency_offset",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test2",
					"frequency",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test2",
					"frequency_offset",
					"30",
				),
			),
		},
		{
			Config: `
				resource "checkly_tcp_monitor" "test1" {
					name             = "test-normal-frequency-updated-to-high"
					activated        = true
					frequency        = 0
					frequency_offset = 10
					locations        = ["eu-central-1"]
					request {
						port     = 443
						hostname = "checkly.com"
					}
				}

				resource "checkly_tcp_monitor" "test2" {
					name      = "test-high-frequency-updated-to-normal"
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
					"checkly_tcp_monitor.test1",
					"frequency",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test1",
					"frequency_offset",
					"10",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test2",
					"frequency",
					"60",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_tcp_monitor.test2",
					"frequency_offset",
				),
			),
		},
	})
}

func TestAccTCPMonitorAlertChannelSubscriptions(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		// Step 1: Add two alert channel subscriptions.
		{
			Config: `
				resource "checkly_alert_channel" "email_1" {
					email {
						address = "tcp-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email {
						address = "tcp-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email {
						address = "tcp-acs-3@example.com"
					}
				}

				resource "checkly_tcp_monitor" "test" {
					name      = "tcp-alert-channel-sub-test"
					frequency = 60
					activated = true
					locations = ["eu-central-1"]

					request {
						hostname = "api.checklyhq.com"
						port     = 80

						assertion {
							source     = "RESPONSE_TIME"
							comparison = "LESS_THAN"
							target     = "2000"
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
					"checkly_tcp_monitor.test",
					"alert_channel_subscription.#",
					"2",
				),
				tcpMonitorTestCheckAlertChannelSubscription(
					"checkly_tcp_monitor.test",
					"checkly_alert_channel.email_1",
					"true",
				),
				tcpMonitorTestCheckAlertChannelSubscription(
					"checkly_tcp_monitor.test",
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
						address = "tcp-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email {
						address = "tcp-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email {
						address = "tcp-acs-3@example.com"
					}
				}

				resource "checkly_tcp_monitor" "test" {
					name      = "tcp-alert-channel-sub-test"
					frequency = 60
					activated = true
					locations = ["eu-central-1"]

					request {
						hostname = "api.checklyhq.com"
						port     = 80

						assertion {
							source     = "RESPONSE_TIME"
							comparison = "LESS_THAN"
							target     = "2000"
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
					"checkly_tcp_monitor.test",
					"alert_channel_subscription.#",
					"3",
				),
				tcpMonitorTestCheckAlertChannelSubscription(
					"checkly_tcp_monitor.test",
					"checkly_alert_channel.email_1",
					"false",
				),
				tcpMonitorTestCheckAlertChannelSubscription(
					"checkly_tcp_monitor.test",
					"checkly_alert_channel.email_2",
					"true",
				),
				tcpMonitorTestCheckAlertChannelSubscription(
					"checkly_tcp_monitor.test",
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
						address = "tcp-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email {
						address = "tcp-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email {
						address = "tcp-acs-3@example.com"
					}
				}

				resource "checkly_tcp_monitor" "test" {
					name      = "tcp-alert-channel-sub-test"
					frequency = 60
					activated = true
					locations = ["eu-central-1"]

					request {
						hostname = "api.checklyhq.com"
						port     = 80

						assertion {
							source     = "RESPONSE_TIME"
							comparison = "LESS_THAN"
							target     = "2000"
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"alert_channel_subscription.#",
					"0",
				),
			),
		},
	})
}
