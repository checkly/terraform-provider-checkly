# Basic traceroute monitor
resource "checkly_traceroute_monitor" "example-traceroute-monitor" {
  name                      = "Example traceroute monitor"
  activated                 = true
  should_fail               = false
  frequency                 = 1
  use_global_alert_settings = true

  locations = [
    "us-west-1"
  ]

  request {
    url = "api.checklyhq.com"
  }
}

# A more complex example tuning the trace and adding assertions
resource "checkly_traceroute_monitor" "example-traceroute-monitor-2" {
  name        = "Example traceroute monitor 2"
  activated   = true
  should_fail = false
  frequency   = 5

  locations = [
    "us-west-1",
    "ap-northeast-1",
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

  request {
    url      = "api.checklyhq.com"
    protocol = "ICMP"
    # ICMP probes have no port; the API also derives the default port for the
    # other protocols (443 for TCP, 33434 for UDP/SCTP), so leave it unset
    # unless a specific port is needed.
    ip_family        = "IPv4"
    max_hops         = 30
    max_unknown_hops = 15
    ptr_lookup       = true
    timeout          = 10

    assertion {
      source     = "PACKET_LOSS"
      property   = ""
      comparison = "LESS_THAN"
      target     = "10"
    }

    assertion {
      source     = "RESPONSE_TIME"
      property   = "avg"
      comparison = "LESS_THAN"
      target     = "2000"
    }
  }
}
