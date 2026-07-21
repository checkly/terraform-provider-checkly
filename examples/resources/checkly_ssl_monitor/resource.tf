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

    # The server fills in any baseline rule that is not listed, so enumerate
    # every rule: a partial baseline would re-plan with a diff on every run.
    security_baseline = jsonencode({
      enabled                 = true
      minTLSVersion           = { value = "TLS1.2", severity = "fail" }
      minKeySizeBits          = { value = 2048, severity = "fail" }
      weakSignatureAlgorithm  = { severity = "fail" }
      weakCipherSuite         = { severity = "fail" }
      knownBadCA              = { severity = "fail" }
      recommendedTLSVersion   = { value = "TLS1.3", severity = "ignore" }
      recommendedKeySizeBits  = { value = 3072, severity = "ignore" }
      ocspMustStapleRespected = { severity = "ignore" }
      sctPresent              = { severity = "ignore" }
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
      property   = "hostnameVerified"
      comparison = "EQUALS"
      target     = "true"
    }
  }
}
