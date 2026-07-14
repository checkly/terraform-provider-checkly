# Basic SSL/TLS certificate monitor
resource "checkly_ssl_monitor" "example-ssl-monitor" {
  name                      = "Example SSL monitor"
  activated                 = true
  should_fail               = false
  frequency                 = 60
  use_global_alert_settings = true

  locations = [
    "us-west-1"
  ]

  request {
    hostname                 = "api.checklyhq.com"
    port                     = 443
    alert_days_before_expiry = 30
  }
}

# A more complex example with response-time thresholds, a custom security
# baseline and certificate assertions
resource "checkly_ssl_monitor" "example-ssl-monitor-2" {
  name                   = "Example SSL monitor 2"
  activated              = true
  should_fail            = false
  frequency              = 60
  degraded_response_time = 3000
  max_response_time      = 10000

  locations = [
    "us-west-1",
    "eu-central-1",
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
    hostname                 = "api.checklyhq.com"
    port                     = 443
    server_name              = "api.checklyhq.com"
    ip_family                = "IPv4"
    skip_chain_validation    = false
    handshake_timeout_ms     = 10000
    alert_days_before_expiry = 20

    security_baseline = jsonencode({
      minTlsVersion = "TLS1.2"
      minKeySize    = 2048
    })

    client_certificate {
      mode = "account_default"
    }

    assertion {
      source     = "CERTIFICATE"
      property   = "daysUntilExpiry"
      comparison = "GREATER_THAN"
      target     = "14"
    }

    assertion {
      source     = "CONNECTION"
      property   = "tlsVersion"
      comparison = "EQUALS"
      target     = "TLS1.3"
    }

    assertion {
      source     = "JSON_RESPONSE"
      property   = "$.certificate.keySizeBits"
      comparison = "GREATER_THAN"
      target     = "2048"
    }
  }
}
