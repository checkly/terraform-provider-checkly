# An Email alert channel
resource "checkly_alert_channel" "email_ac" {
  email {
    address = "john@example.com"
  }
  send_recovery = true
  send_failure = false
  send_degraded = true
  ssl_expiry = true
  ssl_expiry_threshold = 22
}

# A SMS alert channel
resource "checkly_alert_channel" "sms_ac" {
  sms {
    name = "john"
    number = "+5491100001111"
  }
  send_recovery = true
  send_failure = true
}

# A Slack alert channel
resource "checkly_alert_channel" "slack_ac" {
  slack {
    channel = "#checkly-notifications"
    url = "https://hooks.slack.com/services/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"
  }
}

# An Opsgenie alert channel
resource "checkly_alert_channel" "opsgenie_ac" {
  opsgenie {
    name = "opsalerts"
    api_key = "fookey"
    region = "fooregion"
    priority = "foopriority"
  }
}

# A Pagerduty alert channel
resource "checkly_alert_channel" "pagerduty_ac" {
  pagerduty {
    account      = "checkly"
    service_key  = "key1"
    service_name = "pdalert"
  }
}

# A Webhook alert channel
resource "checkly_alert_channel" "webhook_ac" {
  webhook {
    name = "foo"
    method = "get"
    template = "footemplate"
    url = "https://example.com/foo"
    webhook_secret = "foosecret"
  }
}

# A Firehydran alert channel integration
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

# Connecting the alert channel to a check
resource "checkly_check" "example_check" {
  name = "Example check"

  alert_channel_subscription {
    channel_id = checkly_alert_channel.email_ac.id
    activated  = true
  }

  alert_channel_subscription {
    channel_id = checkly_alert_channel.sms_ac.id
    activated  = true
  }
}