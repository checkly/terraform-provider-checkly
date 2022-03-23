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
    url = "https://slack.com/webhook"
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

# An Pagerduty alert channel
resource "checkly_alert_channel" "pagerduty_ac" {
  pagerduty {
    account      = "checkly"
    service_key  = "key1"
    service_name = "pdalert"
  }
}

# An Webhook alert channel
resource "checkly_alert_channel" "webhook_ac" {
  webhook {
    name = "foo"
    method = "get"
    template = "footemplate"
    url = "https://example.com/foo"
    webhook_secret = "foosecret"
  }
}

# Connecting the alert channel to a check
resource "checkly_check" "example-check" {
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