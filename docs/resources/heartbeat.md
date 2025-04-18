---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "checkly_heartbeat Resource - terraform-provider-checkly"
subcategory: ""
description: |-
  Heartbeats allows you to monitor your cron jobs and set up alerting, so you get a notification when things break or slow down.
---

# checkly_heartbeat (Resource)

Heartbeats allows you to monitor your cron jobs and set up alerting, so you get a notification when things break or slow down.

## Example Usage

```terraform
resource "checkly_heartbeat" "example-heartbeat" {
  name      = "Example heartbeat"
  activated = true
  heartbeat {
    period      = 7
    period_unit = "days"
    grace       = 1
    grace_unit  = "days"
  }
  use_global_alert_settings = true
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `activated` (Boolean) Determines if the check is running or not. Possible values `true`, and `false`.
- `heartbeat` (Block Set, Min: 1, Max: 1) (see [below for nested schema](#nestedblock--heartbeat))
- `name` (String) The name of the check.

### Optional

- `alert_channel_subscription` (Block List) (see [below for nested schema](#nestedblock--alert_channel_subscription))
- `alert_settings` (Block List, Max: 1) (see [below for nested schema](#nestedblock--alert_settings))
- `muted` (Boolean) Determines if any notifications will be sent out when a check fails/degrades/recovers.
- `tags` (Set of String) A list of tags for organizing and filtering checks.
- `use_global_alert_settings` (Boolean) When true, the account level alert settings will be used, not the alert setting defined on this check.

### Read-Only

- `id` (String) The ID of this resource.

<a id="nestedblock--heartbeat"></a>
### Nested Schema for `heartbeat`

Required:

- `grace` (Number) How long Checkly should wait before triggering any alerts when a ping does not arrive within the set period.
- `grace_unit` (String) Possible values `seconds`, `minutes`, `hours` and `days`.
- `period` (Number) How often you expect a ping to the ping URL.
- `period_unit` (String) Possible values `seconds`, `minutes`, `hours` and `days`.

Optional:

- `ping_token` (String) Custom token to generate your ping URL. Checkly will expect a ping to `https://ping.checklyhq.com/[PING_TOKEN]`.


<a id="nestedblock--alert_channel_subscription"></a>
### Nested Schema for `alert_channel_subscription`

Required:

- `activated` (Boolean)
- `channel_id` (Number)


<a id="nestedblock--alert_settings"></a>
### Nested Schema for `alert_settings`

Optional:

- `escalation_type` (String) Determines what type of escalation to use. Possible values are `RUN_BASED` or `TIME_BASED`.
- `parallel_run_failure_threshold` (Block List) (see [below for nested schema](#nestedblock--alert_settings--parallel_run_failure_threshold))
- `reminders` (Block List) (see [below for nested schema](#nestedblock--alert_settings--reminders))
- `run_based_escalation` (Block List) (see [below for nested schema](#nestedblock--alert_settings--run_based_escalation))
- `ssl_certificates` (Block Set, Deprecated) (see [below for nested schema](#nestedblock--alert_settings--ssl_certificates))
- `time_based_escalation` (Block List) (see [below for nested schema](#nestedblock--alert_settings--time_based_escalation))

<a id="nestedblock--alert_settings--parallel_run_failure_threshold"></a>
### Nested Schema for `alert_settings.parallel_run_failure_threshold`

Optional:

- `enabled` (Boolean) Applicable only for checks scheduled in parallel in multiple locations.
- `percentage` (Number) Possible values are `10`, `20`, `30`, `40`, `50`, `60`, `70`, `80`, `100`, and `100`. (Default `10`).


<a id="nestedblock--alert_settings--reminders"></a>
### Nested Schema for `alert_settings.reminders`

Optional:

- `amount` (Number) How many reminders to send out after the initial alert notification. Possible values are `0`, `1`, `2`, `3`, `4`, `5`, and `100000`
- `interval` (Number) Possible values are `5`, `10`, `15`, and `30`. (Default `5`).


<a id="nestedblock--alert_settings--run_based_escalation"></a>
### Nested Schema for `alert_settings.run_based_escalation`

Optional:

- `failed_run_threshold` (Number) After how many failed consecutive check runs an alert notification should be sent. Possible values are between 1 and 5. (Default `1`).


<a id="nestedblock--alert_settings--ssl_certificates"></a>
### Nested Schema for `alert_settings.ssl_certificates`

Optional:

- `alert_threshold` (Number) How long before SSL certificate expiry to send alerts. Possible values `3`, `7`, `14`, `30`. (Default `3`).
- `enabled` (Boolean) Determines if alert notifications should be sent for expiring SSL certificates. Possible values `true`, and `false`. (Default `false`).


<a id="nestedblock--alert_settings--time_based_escalation"></a>
### Nested Schema for `alert_settings.time_based_escalation`

Optional:

- `minutes_failing_threshold` (Number) After how many minutes after a check starts failing an alert should be sent. Possible values are `5`, `10`, `15`, and `30`. (Default `5`).
