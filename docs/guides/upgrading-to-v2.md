---
page_title: "Upgrading to v2 of the Checkly provider"
description: |-
  How to upgrade from v1.x of the Checkly provider, and what changed in v2.
---

# Upgrading to v2 of the Checkly provider

Version 2 of the provider is built on the [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework) instead of the Terraform Plugin SDK v2. Most resources keep their names and attributes, but some configuration syntax changes, and a few deprecated attributes are gone.

Existing state is upgraded automatically: you do not need to import or recreate any resources. The exceptions are the [removed resource types](#removed-resource-types), which you need to take out of your state before upgrading. After updating your configuration as described below, `terraform plan` should report no changes.

## Requirements

- Terraform 1.0 or later. The provider now uses version 6 of the Terraform plugin protocol.
- Terraform 1.8 or later to move resources between resource types with `moved` blocks (see [Deprecated resource types](#deprecated-resource-types)).

## Recommended procedure

1. Run `terraform plan` with the latest v1.x release and apply any pending changes, so that you start from a clean plan.
1. If you use `checkly_trigger_check` or `checkly_trigger_group`, take them out of your configuration and state, as described in [Removed resource types](#removed-resource-types).
1. Back up your state, for example with `terraform state pull > backup.tfstate`. Once v2 has refreshed the state, v1.x can no longer read it; to go back to v1.x, restore the backup.
1. Update the version constraint, for example `version = "~> 2.0"`, and run `terraform init -upgrade`.
1. Update your configuration as described in [Configuration changes](#configuration-changes). `terraform validate` points at every line that needs to change.
1. Run `terraform plan`. It should report no changes, except for the cases listed under [Expected one-time changes](#expected-one-time-changes).

## Configuration changes

### Single nested objects are attributes

Some objects that a resource can hold at most one of, such as `alert_settings` and `retry_strategy`, were blocks in v1.x. They are now attributes, which are assigned with `=`:

```terraform
# v1.x
resource "checkly_url_monitor" "example" {
  # ...
  alert_settings {
    escalation_type = "RUN_BASED"
    run_based_escalation {
      failed_run_threshold = 1
    }
  }
  retry_strategy {
    type = "FIXED"
  }
}

# v2
resource "checkly_url_monitor" "example" {
  # ...
  alert_settings = {
    escalation_type = "RUN_BASED"
    run_based_escalation = {
      failed_run_threshold = 1
    }
  }
  retry_strategy = {
    type = "FIXED"
  }
}
```

Collections nested inside such an attribute become lists of objects, for example `assertion = [{ ... }]` inside `api_check_defaults`.

References to values inside these objects no longer go through a list or set of one element. For example, `one(checkly_alert_channel.example.email).address`, `one(checkly_check.example.request).url` and `checkly_url_monitor.example.retry_strategy[0].type` become `checkly_alert_channel.example.email.address`, `checkly_check.example.request.url` and `checkly_url_monitor.example.retry_strategy.type`.

This affects:

| Resource type | Attributes |
|---|---|
| `checkly_alert_channel` | `call`, `email`, `opsgenie`, `pagerduty`, `slack`, `slack_app`, `sms`, `webhook` |
| `checkly_check` | `alert_settings` (and its `parallel_run_failure_threshold`, `reminders`, `run_based_escalation`, `time_based_escalation`), `request` (and its `assertion` and `basic_auth`), `retry_strategy` (and its `only_on`), `trigger_incident` |
| `checkly_check_group` | `alert_settings` (and nested objects as above), `api_check_defaults` (and its `basic_auth` and `assertion`), `retry_strategy` (and its `only_on`) |
| `checkly_check_group_v2` | `api_check_defaults` (and its `basic_auth` and `assertion`), `default_runtime`, `enforce_alert_settings` (and its `alert_settings` and `alert_channel_subscription`), `enforce_locations`, `enforce_retry_strategy` (and its `retry_strategy`), `enforce_scheduling_strategy`, `setup_script`, `teardown_script` |
| `checkly_heartbeat`, `checkly_heartbeat_monitor` | `alert_settings` (and nested objects as above), `heartbeat`, `trigger_incident` |
| `checkly_url_monitor` | `alert_settings` (and nested objects as above), `request` (and its `assertion`), `retry_strategy` (and its `only_on`), `trigger_incident` |
| `checkly_dns_monitor` | `alert_settings` (and nested objects as above), `request` (and its `assertion` and `name_server`), `retry_strategy`, `trigger_incident` |
| `checkly_grpc_monitor` | `alert_settings` (and nested objects as above), `request` (and its `assertion` and `metadata`), `retry_strategy`, `trigger_incident` |
| `checkly_icmp_monitor`, `checkly_tcp_check`, `checkly_tcp_monitor`, `checkly_traceroute_monitor` | `alert_settings` (and nested objects as above), `request` (and its `assertion`), `retry_strategy`, `trigger_incident` |
| `checkly_ssl_monitor` | `alert_settings` (and nested objects as above), `request` (and its `assertion`, `client_certificate` and `security_baseline` with each of its rules, such as `min_tls_version`), `retry_strategy`, `trigger_incident` |
| `checkly_playwright_check_suite` | `alert_settings` (and nested objects as above), `bundle`, `runtime` (and its `engine`, `playwright`, `steps`), `trigger_incident` |
| `checkly_playwright_code_bundle` | `prebuilt_archive` |
| `checkly_status_page_v3` | `theme_colors` (and its `light` and `dark`) |

The table lists every such attribute. All other blocks, such as the repeated `alert_channel_subscription` block, keep their block syntax.

### Removed attributes

These attributes were deprecated in v1.x. Remove them from your configuration:

- `environment_variables` (the map) on `checkly_check` and `checkly_check_group`. Unlike the attributes below, it did take effect: rewrite each entry as an `environment_variable` block with the same `key` and `value`, so that the variables are kept. It was deprecated throughout v1.x and, unlike `double_check`, it is not a separate API setting: the blocks set exactly the same variables, so v2 removes it.
- `ssl_check` on `checkly_check`, which the API ignored.
- `alert_settings.ssl_certificates` on all resource types that had it, which the API ignored.
- `retry_strategy.only_on` on `checkly_dns_monitor`, `checkly_grpc_monitor`, `checkly_icmp_monitor`, `checkly_ssl_monitor`, `checkly_tcp_check`, `checkly_tcp_monitor` and `checkly_traceroute_monitor`, where it had no conditions to set.

`environment_variable` blocks on a `checkly_check` with `type = "API"` have no effect either, because API checks do not use environment variables, and they cause a change in every plan. Remove them as well.

### Deprecated attributes

`double_check` on `checkly_check` and `checkly_check_group` is still deprecated and still works as in v1.x: without a `retry_strategy`, it retries a failed run once in another region. To replace it, remove it and set the equivalent retry strategy:

```terraform
retry_strategy = {
  type                 = "FIXED"
  max_retries          = 1
  base_backoff_seconds = 0
  max_duration_seconds = 600
  same_region          = false
}
```

### Sensitive values

These attributes are now sensitive, so Terraform redacts them in plan output:

- `value` of `checkly_environment_variable`, and of `environment_variable` blocks on checks, check groups and Playwright check suites.
- `password` of `basic_auth` in a check `request` and in `api_check_defaults`.
- `slack.url`, `webhook.url`, `webhook.webhook_secret`, `webhook.headers`, `webhook.query_parameters`, `opsgenie.api_key` and `pagerduty.service_key` of `checkly_alert_channel`.
- `heartbeat.ping_token` of `checkly_heartbeat_monitor` and `checkly_heartbeat`.

An output that exposes one of them, for example a heartbeat ping URL built from `ping_token`, must set `sensitive = true`.

Header and query parameter values of check requests and of `api_check_defaults` are not redacted: keep credentials in them out of shared plan output, for example by using a global environment variable in a check.

## Removed resource types

`checkly_trigger_check` and `checkly_trigger_group` are gone. They managed command line trigger URLs, which Checkly has deprecated. To run checks on demand, use the [Checkly CLI](https://www.checklyhq.com/docs/cli/) (`npx checkly trigger`) or the check sessions trigger endpoint of the Checkly API (`POST /v1/check-sessions/trigger`).

v2 fails to plan while these resource types are in state. Remove them from your configuration, then either:

- Delete the triggers by running `terraform apply` while still on v1.x.
- Keep the trigger URLs working for now by removing the resources from state only. While still on v1.x, with Terraform 1.7 or later, add a `removed` block for each resource and run `terraform apply`:

  ```terraform
  removed {
    from = checkly_trigger_check.example

    lifecycle {
      destroy = false
    }
  }
  ```

  Alternatively, run `terraform state rm checkly_trigger_check.example`, which also works after upgrading to v2. A `removed` block does not.

## Deprecated resource types

`checkly_check` is deprecated in favor of `checkly_api_check`, `checkly_browser_check` and `checkly_multistep_check`, which only expose the attributes of their check type. `checkly_heartbeat` and `checkly_tcp_check` are deprecated in favor of `checkly_heartbeat_monitor` and `checkly_tcp_monitor`. `checkly_maintenance_windows` is deprecated in favor of `checkly_maintenance_window`, because each resource manages a single maintenance window. The deprecated types keep working in v2. `checkly_check_group`, `checkly_status_page` and `checkly_status_page_service` are deprecated too; see [Check groups](#check-groups) and [Status pages](#status-pages).

To switch a check, heartbeat, TCP or maintenance window resource to its new type without recreating it, change the resource type and add a `moved` block. This works in the same plan as the provider upgrade:

```terraform
resource "checkly_heartbeat_monitor" "example" {
  # ...
}

moved {
  from = checkly_heartbeat.example
  to   = checkly_heartbeat_monitor.example
}
```

The per-type check resources have no `double_check`. Before moving a check that sets it, replace it with the equivalent `retry_strategy` shown under [Deprecated attributes](#deprecated-attributes), or the check stops retrying.

When moving an API check from `checkly_check` to `checkly_api_check`, note that the default `degraded_response_time` and `max_response_time` are 5000 and 20000 instead of 15000 and 30000. Set them explicitly to keep the previous thresholds. Also remove `frequency_offset = 0`: `checkly_api_check` expresses a disabled offset by leaving `frequency_offset` unset and rejects 0.

### Check groups

`checkly_check_group` is deprecated in favor of `checkly_check_group_v2`. A v1 group always applies its locations, alert settings, retry strategy and scheduling strategy to its checks; a v2 group only applies the settings you enforce with its `enforce_*` attributes.

Both resource types manage the same check groups, so you can switch a group over without recreating it or its checks. This needs Terraform 1.7 or later:

1. Add a `checkly_check_group_v2` resource with an `import` block that takes the ID of the existing group. To keep the group's behavior, enforce the settings that the v1 group applies:

   ```terraform
   resource "checkly_check_group_v2" "example" {
     name = "Example group"

     enforce_locations = {
       enabled   = true
       locations = ["eu-central-1"]
     }

     enforce_alert_settings = {
       enabled                   = true
       use_global_alert_settings = false
       # alert_settings = { ... } as on the v1 group
     }

     enforce_retry_strategy = {
       enabled = true
       retry_strategy = {
         type = "NO_RETRIES" # as on the v1 group
       }
     }

     enforce_scheduling_strategy = {
       enabled      = true
       run_parallel = false
     }
   }

   import {
     to = checkly_check_group_v2.example
     id = checkly_check_group.example.id
   }
   ```

1. Run `terraform plan`. The import should not plan an update; if it does, adjust the configuration until it does not, unless you mean to change the group.
1. Apply. Then remove the `import` block, replace the `checkly_check_group` resource with a `removed` block, so that Terraform forgets it without deleting the group, and point references such as `group_id` at the new resource:

   ```terraform
   removed {
     from = checkly_check_group.example

     lifecycle {
       destroy = false
     }
   }
   ```

### Status pages

`checkly_status_page` and `checkly_status_page_service` are deprecated in favor of `checkly_status_page_v3`, together with `checkly_status_page_v3_component` and `checkly_status_page_v3_automation_rule`. v3 status pages are separate objects, so an existing page cannot be converted: create a v3 page alongside it, then remove the old one.

Maintenance windows can only be shown on v2 status pages for now: `status_page_visibility` on `checkly_maintenance_window` takes the IDs of `checkly_status_page` and `checkly_status_page_service` resources. Keep the v2 page if you rely on that.

## Expected one-time changes

- v1.x stored an unset optional string, number or collection as an empty value (`""`, `0` or `[]`), and v2 stores it as null. If your configuration sets such an attribute to an empty value explicitly, for example `tags = []` or `repeat_interval = 0`, the first plan shows an in-place update to that empty value. Removing the attribute from the configuration has the same effect and avoids the update.
- `checkly_client_certificate`: v1.x stored `path` without a trailing slash. If your configuration has one, for example `path = "/partner/api/"`, the first plan shows an in-place update of `path` that does not change the certificate.
