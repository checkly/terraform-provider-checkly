# Email Alert Channel
resource "checkly_alert_channel" "email_ac" {
  email {
    address = "info@example.com"
  }
  send_recovery = true
  send_failure  = false
  send_degraded = true
  ssl_expiry    = false
}

# Slack Alert Channel
resource "checkly_alert_channel" "slack_ac" {
  slack {
    channel = "checkly_alerts"
    url     = "https://hooks.slack.com/services/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"
  }
  send_recovery        = true
  send_failure         = true
  send_degraded        = false
  ssl_expiry           = true
  ssl_expiry_threshold = 11
}

# SMS Alert Channel
resource "checkly_alert_channel" "sms_ac" {
  sms {
    name   = "smsalerts"
    number = "4917512345678"
  }
}

# Phone Call Alert Channel
resource "checkly_alert_channel" "call_ac" {
  call {
    name   = "phone call alert"
    number = "+4917512345678"
  }
}

# Opsgenie Alert Channel
resource "checkly_alert_channel" "opsgenie_ac" {
  opsgenie {
    name     = "opsalert"
    api_key  = "key1"
    region   = "EU"
    priority = "P1"
  }
}

# Pagerduty Alert Channel
resource "checkly_alert_channel" "pagerduty_ac" {
  pagerduty {
    account      = "checkly"
    service_key  = "key1"
    service_name = "pdalert"
  }
}

# Webhook Alert Channel
resource "checkly_alert_channel" "webhook_ac" {
  webhook {
    name   = "webhhookalerts"
    method = "get"
    headers = {
      X-HEADER-1 = "foo"
    }
    query_parameters = {
      query1 = "bar"
    }
    template       = "tmpl"
    url            = "https://example.com/webhook"
    webhook_secret = "foo-secret"
  }
}

resource "checkly_alert_channel" "firehydrant_ac" {
  webhook {
    name         = "firehydrant"
    method       = "post"
    template     = <<EOT
{
  "event": "{{ALERT_TITLE}}",
  "link": "{{RESULT_LINK}}",
  "check_id": "{{CHECK_ID}}",
  "check_type": "{{CHECK_TYPE}}",
  "alert_type": "{{ALERT_TYPE}}",
  "started_at": "{{STARTED_AT}}",
  "check_result_id": "{{CHECK_RESULT_ID}}"
},
    EOT
    url          = "https://app.firehydrant.io/integrations/alerting/webhooks/2/checkly"
    webhook_type = "WEBHOOK_FIREHYDRANT"
  }
}
