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
- An account ID, for every API key (see [Account ID](#account-id)).

## Recommended procedure

1. Run `terraform plan` with the latest v1.x release and apply any pending changes, so that you start from a clean plan.
1. If you use `checkly_trigger_check` or `checkly_trigger_group`, take them out of your configuration and state, as described in [Removed resource types](#removed-resource-types).
1. Back up your state, for example with `terraform state pull > backup.tfstate`. Once v2 has refreshed the state, v1.x can no longer read it; to go back to v1.x, restore the backup.
1. Update the version constraint, for example `version = "~> 2.0"`, and run `terraform init -upgrade`.
1. Update your configuration as described in [Configuration changes](#configuration-changes). `terraform validate` points at every line that needs to change, except misspelled names inside objects assigned with `= { ... }`, which Terraform ignores (see [Single nested objects are attributes](#single-nested-objects-are-attributes)).
1. Run `terraform plan`. It should report no changes, except for the cases listed under [Expected one-time changes](#expected-one-time-changes). A change to a value inside a converted object, such as a setting going back to its default, usually means that a name in that object is misspelled.

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

Repeated blocks nested inside such an attribute become lists or sets of objects, written as `[{ ... }]` and also assigned with `=`:

```terraform
# v1.x
resource "checkly_url_monitor" "example" {
  # ...
  request {
    url = "https://example.com"
    assertion {
      source     = "STATUS_CODE"
      comparison = "EQUALS"
      target     = "200"
    }
  }
}

# v2
resource "checkly_url_monitor" "example" {
  # ...
  request = {
    url = "https://example.com"
    assertion = [{
      source     = "STATUS_CODE"
      comparison = "EQUALS"
      target     = "200"
    }]
  }
}
```

Leaving such a list out means it is empty, as leaving out the blocks did in v1.x.

A `dynamic` block that generated such nested blocks becomes a `for` expression:

```terraform
# v1.x
request {
  url = "https://example.com"
  dynamic "assertion" {
    for_each = var.assertions
    content {
      source     = assertion.value.source
      comparison = assertion.value.comparison
      target     = assertion.value.target
    }
  }
}

# v2
request = {
  url = "https://example.com"
  assertion = [for a in var.assertions : {
    source     = a.source
    comparison = a.comparison
    target     = a.target
  }]
}
```

`dynamic` blocks keep working for the blocks that remain, such as `alert_channel_subscription` and `environment_variable`.

Terraform does not reject unknown names inside these objects: a misspelled optional attribute, such as `folow_redirects` in `request = { ... }`, is ignored instead of raising an error as it did inside a block. The attribute then keeps its default. Check the names in these objects carefully when you convert them, in particular the rules of `security_baseline` in `checkly_ssl_monitor`, where a misspelled rule silently keeps its default instead of the value you set.

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
| `checkly_maintenance_window`, `checkly_maintenance_windows` | `status_page_visibility` |
| `checkly_status_page_v3` | `theme_colors` (and its `light` and `dark`) |

The table lists every such attribute. Only blocks that can be repeated, such as `alert_channel_subscription` and `environment_variable`, keep their block syntax.

### Removed attributes

These attributes were deprecated in v1.x. Remove them from your configuration:

- `environment_variables` (the map) on `checkly_check` and `checkly_check_group`. Unlike the attributes below, it did take effect: rewrite each entry as an `environment_variable` block with the same `key` and `value`, so that the variables are kept. It was deprecated throughout v1.x and, unlike `double_check`, it is not a separate API setting: the blocks set exactly the same variables, so v2 removes it.
- `ssl_check` on `checkly_check`, which the API ignored.
- `alert_settings.ssl_certificates` on all resource types that had it, which the API ignored.
- `retry_strategy.only_on` on `checkly_dns_monitor`, `checkly_grpc_monitor`, `checkly_icmp_monitor`, `checkly_ssl_monitor`, `checkly_tcp_check`, `checkly_tcp_monitor` and `checkly_traceroute_monitor`, where it had no conditions to set.

`environment_variable` blocks on a `checkly_check` with `type = "API"` have no effect either, because API checks do not use environment variables, and they cause a change in every plan. Remove them as well.

### Deprecated attributes

`slack` on `checkly_alert_channel` is deprecated in favor of `slack_app`. It uses the legacy Slack incoming webhook integration, still works in v2, and will be removed in a future major release.

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

The value of `double_check` is not read back from the API. Terraform keeps the value from your configuration, and a change made outside Terraform shows up in `retry_strategy`. An import stores `false`, so importing a check or group with `double_check = true` in its configuration plans one in-place update, which keeps its retry strategy.

### Account ID

Every API key now needs an account ID, which must be a UUID; v1.x only required one for user-scoped API keys. If you use an account-scoped key without one, set `account_id` in the provider configuration or the `CHECKLY_ACCOUNT_ID` environment variable. Without it, the provider stops with a `Missing Checkly Account ID` error.

### Sensitive values

These attributes are now sensitive, so Terraform redacts them in plan output:

- `value` of `checkly_environment_variable`, and of `environment_variable` blocks on checks, check groups and Playwright check suites.
- `password` of `basic_auth` in a check `request` and in `api_check_defaults`.
- `slack.url`, `webhook.url`, `webhook.webhook_secret`, `webhook.headers`, `webhook.query_parameters`, `opsgenie.api_key` and `pagerduty.service_key` of `checkly_alert_channel`.
- `heartbeat.ping_token` of `checkly_heartbeat_monitor` and `checkly_heartbeat`.

An output that exposes one of them, for example a heartbeat ping URL built from `ping_token`, must set `sensitive = true`.

Header and query parameter values of check requests and of `api_check_defaults` are not redacted, as in v1.x. To keep credentials in them out of plan output, take them from a variable declared with `sensitive = true` or wrap them in `sensitive(...)`, which Terraform redacts wherever the value is used, or reference a global environment variable in the check instead of the secret itself.

### Debug log removed

The `CHECKLY_DEBUG_LOG` environment variable no longer writes API requests and responses to a file. The file included the API key and secret values in plain text. To inspect the provider's API traffic, route it through an HTTPS proxy with the `HTTPS_PROXY` environment variable, which the provider honors. An intercepting proxy only works once its CA certificate is trusted, for example through the system keychain on macOS or `SSL_CERT_FILE` on Linux. Use `TF_LOG` for Terraform's own logs.

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

`checkly_check` now writes through the API endpoint of each check type, which only stores what applies to that type. `request`, `frequency_offset`, `degraded_response_time`, `max_response_time`, `setup_snippet_id`, `teardown_snippet_id`, `local_setup_script` and `local_teardown_script` have no effect on browser and multi-step checks, and neither do `script` on API checks and `ssl_check_domain` on multi-step checks: changing one only changes the state, and setting one gives a warning at plan time. Remove them, or move the check to its per-type resource, which doesn't have them. A browser or multi-step check with an empty `script` is an error, which the plan reports when the script is known there; with `ignore_changes = [script]`, the stored script counts.

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

Maintenance windows can be shown on v3 pages too. `status_page_visibility` on `checkly_maintenance_window` links a v2 page through its services, with `status_page_ids` and either `service_ids` or `affect_all_services`. A v3 page is linked through its components instead: list it in `status_page_ids` and add an entry for it to `component_configs`, with either the affected `components` and the status each one shows, or `affect_all_components = true`:

```terraform
resource "checkly_maintenance_window" "example" {
  name      = "Database upgrade"
  starts_at = "2028-09-24T00:00:00.000Z"
  ends_at   = "2028-09-24T02:00:00.000Z"

  status_page_visibility = {
    enabled         = true
    status_page_ids = [checkly_status_page_v3.example.id]

    component_configs = [{
      status_page_id = checkly_status_page_v3.example.id
      components = [{
        component_id = checkly_status_page_v3_component.database.id
        status       = "UNDER_MAINTENANCE"
      }]
    }]
  }
}
```

## Expected one-time changes

- v1.x stored an unset optional string, number or collection as an empty value (`""`, `0` or `[]`), and v2 stores it as null. If your configuration sets such an attribute to an empty value explicitly, for example `tags = []` or `repeat_interval = 0`, the first plan shows an in-place update to that empty value. Removing the attribute from the configuration has the same effect and avoids the update.
- v1.x could not clear these attributes: leaving one out kept the stored value, and every plan showed the same change. v2 clears them, so a value set in the web app and absent from your configuration is erased by the first apply. To keep such a value, add it to your configuration.
  - `checkly_status_page`: `custom_domain`, `logo`, `redirect_to` and `favicon`.
  - `checkly_dashboard`: `custom_domain`, `logo`, `link`, `description` and `favicon`.
  - `checkly_check`: `tags` and `group_order`, and on API checks only `setup_snippet_id`, `teardown_snippet_id`, `local_setup_script`, `local_teardown_script` and `request.basic_auth`.
  - `checkly_check_group`: `setup_snippet_id`, `teardown_snippet_id`, `local_setup_script` and `local_teardown_script`.
  - `checkly_url_monitor`, `checkly_tcp_monitor`, `checkly_tcp_check`, `checkly_dns_monitor`, `checkly_icmp_monitor`, `checkly_grpc_monitor`, `checkly_ssl_monitor` and `checkly_traceroute_monitor`: `tags` and `group_order`.
  - `checkly_heartbeat_monitor` and `checkly_heartbeat`: `tags`.
  - `checkly_tcp_monitor`, `checkly_tcp_check` and `checkly_traceroute_monitor`: also `request.assertion`.
  - `checkly_ssl_monitor`: also `request.assertion`, `request.server_name` and `request.client_certificate.client_certificate_id`.
  - `checkly_grpc_monitor`: also `request.proto_content` with `service_definition = "REFLECTION"`.
  - Every resource with `alert_settings`: `reminders.amount` and `parallel_run_failure_threshold.enabled`, when the configured object leaves them out.
- `checkly_playwright_check_suite`, and checks moved from `checkly_check` to the per-type resources: an `intent` or `ai_auto_repair_enabled` value set outside Terraform and absent from your configuration is removed by the first apply, which the plan shows. To keep such a value, add it to your configuration, or add the attribute to `ignore_changes` to leave it to the web app.
- `checkly_dns_monitor`, `checkly_icmp_monitor` and `checkly_traceroute_monitor`: `private_locations` set outside Terraform and absent from your configuration are removed by the first apply, which the plan shows. The same goes for `proxy_url` on `checkly_private_location`. To keep such a value, add it to your configuration, or add the attribute to `ignore_changes` to leave it to the web app.
- A check or monitor added to a group in the web app, with no `group_id` in its configuration, leaves the group on the first apply, as it did with v1.x.
- `checkly_maintenance_window` and `checkly_maintenance_windows`: links to v3 status page components set in the web app and absent from your configuration are removed by the first apply, which the plan shows. If the v3 page stays in `status_page_ids` without an entry in `component_configs`, the apply is rejected instead. To keep the links, add them to `component_configs`, or leave them to the web app with `ignore_changes`: `status_page_visibility.component_configs` when your configuration sets `status_page_visibility` and you use Terraform 1.2.7 or later, or `status_page_visibility` as a whole otherwise, which leaves all its settings to the web app. An account without the status page maintenance windows entitlement can't remove the links, so every update of such a window is rejected until they are in the configuration or ignored.
- `checkly_client_certificate`: v1.x stored `path` without a trailing slash. If your configuration has one, for example `path = "/partner/api/"`, the first plan shows an in-place update of `path` that does not change the certificate.
