# checkly_alert_channel
The `checkly_alert_channel` resource allows users to manage checkly Alert Channels.  

Checkly's Alert Channels feature allows you to define global alerting channels for the checks in your account:

## Example usage
*An Email alert channel*
```terraform
resource "checkly_alert_channel" "ac1" {
    email {
        address = "john@example.com"
  }
}
```  

*A SMS alert channel*
```terraform
resource "checkly_alert_channel" "ac1" {
    sms {
        name = "john"
        number = "0123456789"
  }
}
```  

*A Slack alert channel*
```terraform
resource "checkly_alert_channel" "ac1" {
    slack {
        channel = "#checkly-notifications"
        url = "https://slack.com/webhook"
  }
}
```  

*An Opsgenie alert channel*
```terraform
resource "checkly_alert_channel" "ac1" {
    opsgenie {
        name = "opsalerts"
        api_key = "fookey"
        region = "fooregion"
        priority = "foopriority"

  }
}
```  


*An Webhook alert channel*
```terraform
resource "checkly_alert_channel" "ac1" {
    opsgenie {
        name = "foo"
        method = "get"
        template = "footemplate"
        url = "http://example.com/foo"
        webhook_secret = "foosecret"

  }
}
```  

*Connecting the alert channel to a check
```terraform
resource "checkly_check" "example-check" {
  name                      = "Example check"
  ....

  alert_channel_subscription {
    channel_id = checkly_alert_channel.email_ac.id
    activated  = true
  }


   alert_channel_subscription {
    channel_id = checkly_alert_channel.sms_ac.id
    activated  = true
  }

}
```

*Connecting the alert channel to a check group
```terraform
resource "checkly_check_group" "test-group1" {
  name                      = "Check group"
  ....

  alert_channel_subscription {
    channel_id = checkly_alert_channel.email_ac.id
    activated  = true
  }


   alert_channel_subscription {
    channel_id = checkly_alert_channel.sms_ac.id
    activated  = true
  }

}
```

## Argument Reference
    a `checkly_alert_channel` should contain configuration for a single alerting channel, 
    which can be one of the following: `email`, `sms`, `slack`, `opsgenie`, `webhook`.
### Argument Reference for Email Alert Channel
* `email` (Optional): 
    * `address` (Required) the email address of this email alert channel.
### SMS Alert Channel
* `sms` (Optional):
    * `name` (Required) Name of this channel.
    * `number` (Required) Mobile number to receive alerts.
### Argument Reference for Slack Alert Channel
* `slack` (Optional)
    * `channel` (Required) Slack's channel name.
    * `url` (Required) Slack-Webhook's url.
### Argument Reference for Opsgenie Alert Channel
* `opsgenie` (Optional)
    * `name` (Required) Opsgenie's channel name.
    * `api_key` (Required).
    * `region` (Required).
    * `priority` (Required).
### Argument Reference for Opsgenie Alert Channel
* `webhook` (Optional)
    * `name` (Required) Webhook's channel name.
    * `method` (Required).
    * `headers` (Optional).
    * `query_parameters` (Optional).
    * `template` (Optional).
    * `url` (Required).
    * `webhook_secret` (Optional).
