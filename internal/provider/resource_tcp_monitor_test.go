package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
			ExpectError: regexp.MustCompile(`The argument "request" is required, but no definition was found.`),
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
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"request.hostname",
					"api.checklyhq.com",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"request.port",
					"80",
				),
				resource.TestCheckTypeSetElemNestedAttrs(
					"checkly_tcp_monitor.test",
					"request.assertions.*",
					map[string]string{
						"source":     "RESPONSE_TIME",
						"property":   "",
						"comparison": "LESS_THAN",
						"target":     "2000",
					},
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
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"request.assertions.#",
					"2",
				),
				resource.TestCheckTypeSetElemNestedAttrs(
					"checkly_tcp_monitor.test",
					"request.assertions.*",
					map[string]string{
						"source":     "RESPONSE_TIME",
						"property":   "",
						"comparison": "LESS_THAN",
						"target":     "2000",
					},
				),
				resource.TestCheckTypeSetElemNestedAttrs(
					"checkly_tcp_monitor.test",
					"request.assertions.*",
					map[string]string{
						"source":     "RESPONSE_DATA",
						"property":   "",
						"comparison": "CONTAINS",
						"target":     "Foo",
					},
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
		{
			ResourceName:      "checkly_tcp_monitor.test",
			ImportState:       true,
			ImportStateVerify: true,
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

	request = {
		hostname = "example.com"
		port     = 443
		assertions = [{
			source     = "RESPONSE_TIME"
			comparison = "LESS_THAN"
			target     = "3000"
		}]
	}

	trigger_incident = {
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
					"trigger_incident.severity",
					"MINOR",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test_trigger_incident",
					"trigger_incident.name",
					"TCP Monitor Connection Issue",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test_trigger_incident",
					"trigger_incident.description",
					"The TCP monitor has detected connectivity issues",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test_trigger_incident",
					"trigger_incident.notify_subscribers",
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
	  request = {
		hostname     = "api.checklyhq.com"
		port         = 80
		assertions = [{
		  comparison = "LESS_THAN"
		  property   = ""
		  source     = "RESPONSE_TIME"
		  target     = "2000"
		}]
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
	  request = {
		hostname     = "api.checklyhq.com"
		port         = 80
		assertions = [{
		  comparison = "LESS_THAN"
		  property   = ""
		  source     = "RESPONSE_TIME"
		  target     = "2000"
		}]
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
	request = {
	  hostname         = "api.checklyhq.com"
	  port             = 80
	  assertions = [{
		comparison = "LESS_THAN"
		property   = ""
		source     = "RESPONSE_TIME"
		target     = "2000"
	  }, {
		comparison = "CONTAINS"
		property   = ""
		source     = "RESPONSE_DATA"
		target     = "Foo"
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
					request = {
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
					request = {
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
					request = {
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
					request = {
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
					request = {
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
					request = {
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
					request = {
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
					request = {
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
					request = {
						port     = 443
						hostname = "checkly.com"
					}
				}

				resource "checkly_tcp_monitor" "test2" {
					name      = "test-group-assignment"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
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
					request = {
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
					request = {
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
					request = {
						port     = 443
						hostname = "checkly.com"
					}
				}

				resource "checkly_tcp_monitor" "test2" {
					name      = "test-group-assignment"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
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
					request = {
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
					request = {
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
					request = {
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
					request = {
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
					request = {
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
					request = {
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
					request = {
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
					request = {
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
					request = {
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
					request = {
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
					request = {
						port     = 443
						hostname = "checkly.com"
					}
				}

				resource "checkly_tcp_monitor" "test2" {
					name      = "test-high-frequency-updated-to-normal"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
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

// The Checkly PUT endpoints are PATCH-style and have a history of silently
// keeping properties that are omitted from update payloads. The unset sweep
// tests below set every feasibly testable optional property to a non-default
// value and then remove it from the configuration, verifying that the server
// actually clears (or resets) the value. The framework's implicit post-apply
// refresh and empty-plan checks detect any value the server kept.

func TestAccTCPMonitorUnsetSweepBooleansAndTags(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_tcp_monitor" "test" {
					name                      = "tcp-unset-sweep-basics"
					activated                 = true
					frequency                 = 60
					locations                 = ["eu-central-1"]
					muted                     = true
					should_fail               = true
					run_parallel              = true
					use_global_alert_settings = false
					tags                      = ["sweep-a", "sweep-b"]
					request = {
						hostname = "api.checklyhq.com"
						port     = 443
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"muted",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"should_fail",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"run_parallel",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"use_global_alert_settings",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"tags.#",
					"2",
				),
			),
		},
		{
			Config: `
				resource "checkly_tcp_monitor" "test" {
					name      = "tcp-unset-sweep-basics"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						hostname = "api.checklyhq.com"
						port     = 443
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"muted",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"should_fail",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"run_parallel",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"use_global_alert_settings",
					"true",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_tcp_monitor.test",
					"tags.#",
				),
			),
		},
	})
}

func TestAccTCPMonitorUnsetSweepResponseTimes(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_tcp_monitor" "test" {
					name                   = "tcp-unset-sweep-response-times"
					activated              = true
					frequency              = 60
					locations              = ["eu-central-1"]
					degraded_response_time = 1000
					max_response_time      = 2000
					request = {
						hostname = "api.checklyhq.com"
						port     = 443
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"degraded_response_time",
					"1000",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"max_response_time",
					"2000",
				),
			),
		},
		// An explicit 0 is schema-valid and must actually be persisted; it is
		// omitted from the payload when the SDK serializes it with omitempty.
		{
			Config: `
				resource "checkly_tcp_monitor" "test" {
					name                   = "tcp-unset-sweep-response-times"
					activated              = true
					frequency              = 60
					locations              = ["eu-central-1"]
					degraded_response_time = 0
					max_response_time      = 0
					request = {
						hostname = "api.checklyhq.com"
						port     = 443
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"degraded_response_time",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"max_response_time",
					"0",
				),
			),
		},
		{
			Config: `
				resource "checkly_tcp_monitor" "test" {
					name      = "tcp-unset-sweep-response-times"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						hostname = "api.checklyhq.com"
						port     = 443
					}
				}
			`,
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
			),
		},
	})
}

func TestAccTCPMonitorUnsetSweepLocations(t *testing.T) {
	// Private location slugs are unique per account and limited to 30
	// characters, so a short random suffix avoids collisions between runs.
	slug := fmt.Sprintf("tf-tcp-unset-%d", acctest.RandInt()%100000)
	accTestCase(t, []resource.TestStep{
		{
			Config: fmt.Sprintf(`
				resource "checkly_private_location" "test" {
					name      = %[1]q
					slug_name = %[1]q
				}

				resource "checkly_tcp_monitor" "test" {
					name              = "tcp-unset-sweep-locations"
					activated         = true
					frequency         = 60
					locations         = ["eu-central-1", "eu-west-1"]
					private_locations = [checkly_private_location.test.slug_name]
					request = {
						hostname = "api.checklyhq.com"
						port     = 443
					}
				}
			`, slug),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"locations.#",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"private_locations.#",
					"1",
				),
			),
		},
		// Removing locations while the monitor still runs on a private
		// location must clear the public locations server-side.
		{
			Config: fmt.Sprintf(`
				resource "checkly_private_location" "test" {
					name      = %[1]q
					slug_name = %[1]q
				}

				resource "checkly_tcp_monitor" "test" {
					name              = "tcp-unset-sweep-locations"
					activated         = true
					frequency         = 60
					private_locations = [checkly_private_location.test.slug_name]
					request = {
						hostname = "api.checklyhq.com"
						port     = 443
					}
				}
			`, slug),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(
					"checkly_tcp_monitor.test",
					"locations.#",
				),
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"private_locations.#",
					"1",
				),
			),
		},
		// Removing private locations while public locations are configured
		// must clear the private locations server-side.
		{
			Config: fmt.Sprintf(`
				resource "checkly_private_location" "test" {
					name      = %[1]q
					slug_name = %[1]q
				}

				resource "checkly_tcp_monitor" "test" {
					name      = "tcp-unset-sweep-locations"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						hostname = "api.checklyhq.com"
						port     = 443
					}
				}
			`, slug),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"locations.#",
					"1",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_tcp_monitor.test",
					"private_locations.#",
				),
			),
		},
	})
}

func TestAccTCPMonitorUnsetSweepRequest(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_tcp_monitor" "test" {
					name      = "tcp-unset-sweep-request"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						hostname  = "api.checklyhq.com"
						port      = 443
						data      = "PING"
						ip_family = "IPv6"
						assertions = [{
							source     = "RESPONSE_TIME"
							comparison = "LESS_THAN"
							target     = "2000"
						}, {
							source     = "RESPONSE_DATA"
							comparison = "CONTAINS"
							target     = "PONG"
						}]
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_tcp_monitor.test", "request.data", "PING"),
				resource.TestCheckResourceAttr("checkly_tcp_monitor.test", "request.ip_family", "IPv6"),
				resource.TestCheckResourceAttr("checkly_tcp_monitor.test", "request.assertions.#", "2"),
			),
		},
		// Removing ip_family must reset it to the IPv4 default, and removing
		// all assertions must clear the assertions server-side.
		//
		// data is NOT removed here: the backend cannot clear it. It rejects
		// an empty string ("request.data is not allowed to be empty") and
		// silently keeps the previous value when the key is null or omitted,
		// so removing data from the configuration fails the apply with an
		// inconsistent result error. A set data value can only be changed to
		// another non-empty value, which this step also verifies.
		{
			Config: `
				resource "checkly_tcp_monitor" "test" {
					name      = "tcp-unset-sweep-request"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						hostname = "api.checklyhq.com"
						port     = 443
						data     = "PONG"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_tcp_monitor.test", "request.data", "PONG"),
				resource.TestCheckResourceAttr("checkly_tcp_monitor.test", "request.ip_family", "IPv4"),
				resource.TestCheckResourceAttr("checkly_tcp_monitor.test", "request.assertions.#", "0"),
			),
		},
	})
}

func TestAccTCPMonitorUnsetSweepTriggerIncident(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_status_page_service" "test" {
					name = "tcp-unset-sweep-service"
				}

				resource "checkly_tcp_monitor" "test" {
					name      = "tcp-unset-sweep-trigger-incident"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						hostname = "api.checklyhq.com"
						port     = 443
					}
					trigger_incident = {
						service_id         = checkly_status_page_service.test.id
						severity           = "MINOR"
						name               = "TCP sweep incident"
						description        = "Incident trigger for the unset sweep"
						notify_subscribers = false
					}
				}
			`,
			Check: resource.TestCheckResourceAttr(
				"checkly_tcp_monitor.test",
				"trigger_incident.%",
				"5",
			),
		},
		{
			Config: `
				resource "checkly_status_page_service" "test" {
					name = "tcp-unset-sweep-service"
				}

				resource "checkly_tcp_monitor" "test" {
					name      = "tcp-unset-sweep-trigger-incident"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						hostname = "api.checklyhq.com"
						port     = 443
					}
				}
			`,
			Check: resource.TestCheckNoResourceAttr(
				"checkly_tcp_monitor.test",
				"trigger_incident",
			),
		},
	})
}

func TestAccTCPMonitorUnsetSweepGroupOrder(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "tcp-unset-sweep-group"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}

				resource "checkly_tcp_monitor" "test" {
					name        = "tcp-unset-sweep-group-order"
					activated   = true
					frequency   = 60
					locations   = ["eu-central-1"]
					group_id    = checkly_check_group.test.id
					group_order = 2
					request = {
						hostname = "api.checklyhq.com"
						port     = 443
					}
				}
			`,
			Check: resource.TestCheckResourceAttr(
				"checkly_tcp_monitor.test",
				"group_order",
				"2",
			),
		},
		// Removing group_order while staying in the group must reset it to
		// the default 0.
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "tcp-unset-sweep-group"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}

				resource "checkly_tcp_monitor" "test" {
					name      = "tcp-unset-sweep-group-order"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					group_id  = checkly_check_group.test.id
					request = {
						hostname = "api.checklyhq.com"
						port     = 443
					}
				}
			`,
			Check: resource.TestCheckResourceAttr(
				"checkly_tcp_monitor.test",
				"group_order",
				"0",
			),
		},
	})
}

func TestAccTCPMonitorUnsetSweepRuntimeID(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_tcp_monitor" "test" {
					name       = "tcp-unset-sweep-runtime"
					activated  = true
					frequency  = 60
					locations  = ["eu-central-1"]
					runtime_id = "2025.04"
					request = {
						hostname = "api.checklyhq.com"
						port     = 443
					}
				}
			`,
			Check: resource.TestCheckResourceAttr(
				"checkly_tcp_monitor.test",
				"runtime_id",
				"2025.04",
			),
		},
		{
			Config: `
				resource "checkly_tcp_monitor" "test" {
					name      = "tcp-unset-sweep-runtime"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request = {
						hostname = "api.checklyhq.com"
						port     = 443
					}
				}
			`,
			Check: resource.TestCheckNoResourceAttr(
				"checkly_tcp_monitor.test",
				"runtime_id",
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
					email = {
						address = "tcp-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email = {
						address = "tcp-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email = {
						address = "tcp-acs-3@example.com"
					}
				}

				resource "checkly_tcp_monitor" "test" {
					name      = "tcp-alert-channel-sub-test"
					frequency = 60
					activated = true
					locations = ["eu-central-1"]

					request = {
						hostname = "api.checklyhq.com"
						port     = 80

						assertions = [{
							source     = "RESPONSE_TIME"
							comparison = "LESS_THAN"
							target     = "2000"
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
					"checkly_tcp_monitor.test",
					"alert_channel_subscriptions.#",
					"2",
				),
				testCheckAlertChannelSubscription(
					"checkly_tcp_monitor.test",
					"alert_channel_subscriptions",
					"checkly_alert_channel.email_1",
					"true",
				),
				testCheckAlertChannelSubscription(
					"checkly_tcp_monitor.test",
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
						address = "tcp-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email = {
						address = "tcp-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email = {
						address = "tcp-acs-3@example.com"
					}
				}

				resource "checkly_tcp_monitor" "test" {
					name      = "tcp-alert-channel-sub-test"
					frequency = 60
					activated = true
					locations = ["eu-central-1"]

					request = {
						hostname = "api.checklyhq.com"
						port     = 80

						assertions = [{
							source     = "RESPONSE_TIME"
							comparison = "LESS_THAN"
							target     = "2000"
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
					"checkly_tcp_monitor.test",
					"alert_channel_subscriptions.#",
					"3",
				),
				testCheckAlertChannelSubscription(
					"checkly_tcp_monitor.test",
					"alert_channel_subscriptions",
					"checkly_alert_channel.email_1",
					"false",
				),
				testCheckAlertChannelSubscription(
					"checkly_tcp_monitor.test",
					"alert_channel_subscriptions",
					"checkly_alert_channel.email_2",
					"true",
				),
				testCheckAlertChannelSubscription(
					"checkly_tcp_monitor.test",
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
						address = "tcp-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email = {
						address = "tcp-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email = {
						address = "tcp-acs-3@example.com"
					}
				}

				resource "checkly_tcp_monitor" "test" {
					name      = "tcp-alert-channel-sub-test"
					frequency = 60
					activated = true
					locations = ["eu-central-1"]

					request = {
						hostname = "api.checklyhq.com"
						port     = 80

						assertions = [{
							source     = "RESPONSE_TIME"
							comparison = "LESS_THAN"
							target     = "2000"
						}]
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_tcp_monitor.test",
					"alert_channel_subscriptions.#",
					"0",
				),
			),
		},
	})
}
