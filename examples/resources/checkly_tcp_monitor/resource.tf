# Basic TCP monitor
resource "checkly_tcp_monitor" "example-tcp-monitor" {
  name                      = "Example TCP monitor"
  activated                 = true
  should_fail               = false
  frequency                 = 1
  use_global_alert_settings = true

  locations = [
    "us-west-1"
  ]

  request {
    hostname = "api.checklyhq.com"
    port     = 80
  }
}

# A more complex example using assertions and setting alerts
resource "checkly_tcp_monitor" "example-tcp-monitor-2" {
  name                   = "Example TCP monitor 2"
  activated              = true
  should_fail            = true
  frequency              = 1
  degraded_response_time = 5000
  max_response_time      = 10000

  locations = [
    "us-west-1",
    "ap-northeast-1",
    "ap-south-1",
  ]

  alert_settings {
    escalation_type = "RUN_BASED"

    run_based_escalation {
      failed_run_threshold = 1
    }

    reminders {
      amount = 1
    }
  }

  retry_strategy {
    type                 = "FIXED"
    base_backoff_seconds = 60
    max_duration_seconds = 600
    max_retries          = 3
    same_region          = false
  }

  request {
    hostname = "api.checklyhq.com"
    port     = 80
    data     = "hello"

    assertion {
      source     = "RESPONSE_DATA"
      property   = ""
      comparison = "CONTAINS"
      target     = "welcome"
    }

    assertion {
      source     = "RESPONSE_TIME"
      property   = ""
      comparison = "LESS_THAN"
      target     = "2000"
    }
  }
}
