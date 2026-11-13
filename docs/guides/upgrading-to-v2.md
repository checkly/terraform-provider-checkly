---
page_title: "Upgrading to v2 of the Checkly provider"
description: |-
  How to upgrade from v1.x of the Checkly provider, and what changed in v2.
---

# Upgrading to v2 of the Checkly provider

Version 2 of the provider is built on the [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework) instead of the Terraform Plugin SDK v2. Most resources keep their names and attributes, but some configuration syntax changes, and a few deprecated attributes are gone.

Existing state is upgraded automatically: you do not need to import or recreate any resources. The exceptions are the [removed resource types](#removed-resource-types), which you need to take out of your state before upgrading. After updating your configuration as described below, `terraform plan` should report no changes.

## Requirements

- Terraform 1.8 or later. It is the first version that can move resources between resource types with `moved` blocks, which switch the [deprecated resource types](#deprecated-resource-types) to their replacements.
- An account ID, for every API key (see [Account ID](#account-id)).

## Recommended procedure

1. Upgrade Terraform to 1.8 or later, if you haven't already. Do this before upgrading the provider: the v1.x releases you're on today already work with Terraform 1.8, so Terraform and the provider don't have to change at the same time.

   To make an older Terraform fail at `terraform init` with a clear message, rather than at the first cross-type `moved` block, require 1.8 in your configuration:

   ```terraform
   terraform {
     required_version = ">= 1.8"
   }
   ```
1. Run `terraform plan` with the latest v1.x release and apply any pending changes, so that you start from a clean plan.
1. If you use `checkly_trigger_check` or `checkly_trigger_group`, take them out of your configuration and state, as described in [Removed resource types](#removed-resource-types).
1. Back up your state, for example with `terraform state pull > backup.tfstate`. Once v2 has refreshed the state, v1.x can no longer read it; to go back to v1.x, restore the backup.
1. Update the version constraint, for example `version = "~> 2.0"`, and run `terraform init -upgrade`.
1. Update your configuration as described in [Configuration changes](#configuration-changes). `terraform validate` points at every line that needs to change, except misspelled names inside objects written as `{ ... }`, which Terraform ignores (see [Single nested objects are attributes](#single-nested-objects-are-attributes)).
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

Repeated blocks nested inside such an attribute become lists or sets of objects with plural names, written as `[{ ... }]` and also assigned with `=` (see [Collections are plural attributes](#collections-are-plural-attributes)):

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
    assertions = [{
      source     = "STATUS_CODE"
      comparison = "EQUALS"
      target     = "200"
    }]
  }
}
```

Terraform does not reject unknown names inside these objects: a misspelled optional attribute, such as `folow_redirects` in `request = { ... }`, is ignored instead of raising an error as it did inside a block. The attribute then keeps its default. Check the names in these objects carefully when you convert them, in particular the rules of `security_baseline` in `checkly_ssl_monitor`, where a misspelled rule silently keeps its default instead of the value you set.

References to values inside these objects no longer go through a list or set of one element. For example, `one(checkly_alert_channel.example.email).address`, `one(checkly_check.example.request).url` and `checkly_url_monitor.example.retry_strategy[0].type` become `checkly_alert_channel.example.email.address`, `checkly_check.example.request.url` and `checkly_url_monitor.example.retry_strategy.type`.

This affects:

| Resource type | Attributes |
|---|---|
| `checkly_alert_channel` | `call`, `email`, `opsgenie`, `pagerduty`, `slack`, `slack_app`, `sms`, `webhook` |
| `checkly_check` | `alert_settings` (and its `parallel_run_failure_threshold`, `reminders`, `run_based_escalation`, `time_based_escalation`), `request` (and its `assertions` and `basic_auth`), `retry_strategy` (and its `only_on`), `trigger_incident` |
| `checkly_check_group` | `alert_settings` (and nested objects as above), `api_check_defaults` (and its `basic_auth` and `assertions`), `retry_strategy` (and its `only_on`) |
| `checkly_check_group_v2` | `api_check_defaults` (and its `basic_auth` and `assertions`), `default_runtime`, `enforce_alert_settings` (and its `alert_settings` and `alert_channel_subscriptions`), `enforce_locations`, `enforce_retry_strategy` (and its `retry_strategy`), `enforce_scheduling_strategy`, `setup_script`, `teardown_script` |
| `checkly_heartbeat`, `checkly_heartbeat_monitor` | `alert_settings` (and nested objects as above), `heartbeat`, `trigger_incident` |
| `checkly_url_monitor` | `alert_settings` (and nested objects as above), `request` (and its `assertions`), `retry_strategy` (and its `only_on`), `trigger_incident` |
| `checkly_dns_monitor` | `alert_settings` (and nested objects as above), `request` (and its `assertions` and `name_server`), `retry_strategy`, `trigger_incident` |
| `checkly_grpc_monitor` | `alert_settings` (and nested objects as above), `request` (and its `assertions` and `metadata`), `retry_strategy`, `trigger_incident` |
| `checkly_icmp_monitor`, `checkly_tcp_check`, `checkly_tcp_monitor`, `checkly_traceroute_monitor` | `alert_settings` (and nested objects as above), `request` (and its `assertions`), `retry_strategy`, `trigger_incident` |
| `checkly_ssl_monitor` | `alert_settings` (and nested objects as above), `request` (and its `assertions`, `client_certificate` and `security_baseline` with each of its rules, such as `min_tls_version`), `retry_strategy`, `trigger_incident` |
| `checkly_playwright_check_suite` | `alert_settings` (and nested objects as above), `bundle`, `runtime` (and its `engine`, `playwright`, `steps`), `trigger_incident` |
| `checkly_playwright_code_bundle` | `prebuilt_archive` |
| `checkly_maintenance_window`, `checkly_maintenance_windows` | `status_page_visibility` |
| `checkly_status_page_v3` | `theme_colors` (and its `light` and `dark`) |

The table lists every such attribute.

### Collections are plural attributes

No resource has blocks any more. Every block that could be repeated, including the ones nested inside single nested objects such as `request`, is now a list or set of objects with a plural name, assigned with `=`:

| v1.x block | v2 attribute | Resource types |
|---|---|---|
| `alert_channel_subscription` | `alert_channel_subscriptions` | every check and monitor resource type, `checkly_check_group`, `checkly_playwright_check_suite`, and `enforce_alert_settings.alert_channel_subscription` of `checkly_check_group_v2` |
| `environment_variable` | `environment_variables` | `checkly_check`, `checkly_check_group`, `checkly_check_group_v2`, `checkly_playwright_check_suite` |
| `request.assertion` | `request.assertions` | every check and monitor resource type with a `request` |
| `api_check_defaults.assertion` | `api_check_defaults.assertions` | `checkly_check_group`, `checkly_check_group_v2` |
| `runtime.playwright.device` | `runtime.playwright.devices` | `checkly_playwright_check_suite` |
| `component` | `components` | `checkly_status_page_v3_automation_rule` |
| `card` | `cards` | `checkly_status_page` |
| `card.service_attachment` | `cards[*].service_attachments` | `checkly_status_page` |

The resource types that are new in v2 follow the same rule, for example `dependencies` on `checkly_browser_check` and `checkly_multistep_check`.

The deprecated `environment_variables` map of `checkly_check` and `checkly_check_group` becomes the same `environment_variables` list: write each entry as an element `{ key = "...", value = "..." }`. Its state converts automatically; see [Removed attributes](#removed-attributes) and [Expected one-time changes](#expected-one-time-changes).

Each block becomes one element of the list:

```terraform
# v1.x
resource "checkly_url_monitor" "example" {
  # ...
  alert_channel_subscription {
    channel_id = checkly_alert_channel.email.id
    activated  = true
  }
  alert_channel_subscription {
    channel_id = checkly_alert_channel.sms.id
    activated  = false
  }
}

# v2
resource "checkly_url_monitor" "example" {
  # ...
  alert_channel_subscriptions = [
    {
      channel_id = checkly_alert_channel.email.id
      activated  = true
    },
    {
      channel_id = checkly_alert_channel.sms.id
      activated  = false
    },
  ]
}
```

A collection can now be assigned from any expression, such as `alert_channel_subscriptions = local.channels`. A `dynamic` block becomes a `for` expression:

```terraform
# v1.x
dynamic "environment_variable" {
  for_each = var.environment
  content {
    key   = environment_variable.key
    value = environment_variable.value
  }
}

# v2
environment_variables = [for key, value in var.environment : {
  key   = key
  value = value
}]
```

Leaving out `alert_channel_subscriptions`, `environment_variables` or `components` still removes them in Checkly, as leaving out all the blocks did in v1.x. Leaving out `request.assertions` or `api_check_defaults.assertions` removes all assertions too. Leaving out `runtime.playwright.devices` selects the default devices, chromium, firefox and webkit, whether or not `runtime.auto_detect` is enabled.

As with [single nested objects](#single-nested-objects-are-attributes), a misspelled name inside an element is ignored instead of raising an error. Entries in `ignore_changes` and references to these attributes, such as `checkly_url_monitor.example.alert_channel_subscription`, must use the new names too.

### No group or snippet is null

`group_id` and `group_order` on checks and monitors, and `setup_snippet_id` and `teardown_snippet_id` on `checkly_check`, `checkly_api_check` and `checkly_check_group`, are null when the check is not in a group, has no position in it or has no snippet. v1.x stored `0`. Setting `group_id` or a snippet ID to `0` is now an error at plan time; remove the attribute instead, or write `null` where an expression chooses between a value and none:

```terraform
# v1.x
group_id = var.grouped ? checkly_check_group.example.id : 0

# v2
group_id = var.grouped ? checkly_check_group.example.id : null
```

An expression that compares one of these attributes with `0` must compare it with `null` instead. `group_order` may be `0`: positions start at `0`, as in the web app, and a group runs a check without a position as if it were at position `0`. For the same reason, when `group_order` is not set, a position of `0` given to the check outside Terraform is not shown as a change. If your configuration sets `group_order = 0`, the first plan after the upgrade shows an in-place update to `0`, as described in [Expected one-time changes](#expected-one-time-changes).

### Removed attributes

These attributes were deprecated in v1.x. Remove them from your configuration:

- `environment_variables` as a map on `checkly_check` and `checkly_check_group`. Unlike the attributes below, it did take effect: rewrite each entry as an element of the `environment_variables` list with the same `key` and `value`, so that the variables are kept. The state is converted automatically, but an unconverted map fails with Terraform's `Inappropriate value for attribute "environment_variables": list of object required` error. It was deprecated throughout v1.x and, unlike `double_check`, it is not a separate API setting: the list sets exactly the same variables.
- `ssl_check` on `checkly_check`, which the API ignored.
- `alert_settings.ssl_certificates` on all resource types that had it, which the API ignored.
- `retry_strategy.only_on` on `checkly_dns_monitor`, `checkly_grpc_monitor`, `checkly_icmp_monitor`, `checkly_ssl_monitor`, `checkly_tcp_check`, `checkly_tcp_monitor` and `checkly_traceroute_monitor`, where it had no conditions to set.

`environment_variables` on a `checkly_check` with `type = "API"` have no effect either, because API checks do not use environment variables, and they cause a change in every plan. Remove them as well.

### should_fail only on API checks, URL monitors and TCP monitors

As in the Checkly CLI, only API checks, URL monitors and TCP monitors have `should_fail`: `checkly_check` with `type = "API"`, `checkly_api_check`, `checkly_url_monitor`, `checkly_tcp_monitor` and `checkly_tcp_check`. `checkly_browser_check` and `checkly_multistep_check` don't have it.

- `checkly_grpc_monitor`, `checkly_ssl_monitor` and `checkly_traceroute_monitor` no longer have `should_fail`, which they ignored. Delete the line.
- On a `checkly_check` with `type = "BROWSER"` or `type = "MULTI_STEP"`, setting `should_fail` fails the plan, even to `false`. Delete the line.

A browser or multi-step check with `should_fail = true` passed when its script failed and failed when it passed. Once an update sets `should_fail` to `false`, it reports the outcome of its script as it is.

**Before you move a browser or multi-step `checkly_check` with `should_fail = true` to `checkly_browser_check` or `checkly_multistep_check`, remove `should_fail` from the `checkly_check` and apply, which updates it to `false`. Only then add the `moved` block.** A check moved, imported or refreshed with `should_fail` still `true` in Checkly gives a `should_fail Is Not Supported` warning. See [Expected one-time changes](#expected-one-time-changes).

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

- `value` of `checkly_environment_variable`, and of `environment_variables` on checks, check groups and Playwright check suites.
- `password` of `basic_auth` in a check `request` and in `api_check_defaults`.
- `slack.url`, `webhook.url`, `webhook.webhook_secret`, `opsgenie.api_key` and `pagerduty.service_key` of `checkly_alert_channel`.
- `heartbeat.ping_token` of `checkly_heartbeat_monitor` and `checkly_heartbeat`.

An output that exposes one of them, for example a heartbeat ping URL built from `ping_token`, must set `sensitive = true`.

Header and query parameter values are not redacted, as in v1.x: those of check requests, of `api_check_defaults`, `request.metadata` of `checkly_grpc_monitor`, and `webhook.headers` and `webhook.query_parameters` of `checkly_alert_channel`. To keep credentials in them out of plan output, take them from a variable declared with `sensitive = true` or wrap them in `sensitive(...)`, which Terraform redacts wherever the value is used. In a check, you can also reference a global environment variable instead of the secret itself.

### Values the API rejects or changes

The provider checks lengths and formats at plan time, so some configurations that applied with v1.x now fail the plan:

- An uppercase `url` on `checkly_status_page`. The API lowercased it; use the lowercase URL.
- An element of `environment_variables` on a check or check group with a value over 64000 bytes in UTF-8, or 31744 bytes if it is secret or locked, or more than 200 elements in `environment_variables`. The API dropped those variables without an error.
- An uppercase `custom_domain` on `checkly_status_page`, or one with a scheme such as `https://`. The API lowercased it and removed the scheme; use the lowercase hostname.
- A `starts_at`, `ends_at` or `repeat_ends_at` on `checkly_maintenance_window` or `checkly_maintenance_windows` that is neither an RFC 3339 date-time with a UTC offset, such as `2099-06-01T09:00:00Z`, nor a date, such as `2099-06-01`. A date-time without a UTC offset, for example, was read in the server's time zone. A `repeat_unit` without a `repeat_interval` of 1 or more, or a `repeat_ends_at` without one, also fails the plan: the API dropped them on update. Remove them from a window that does not repeat.
- An empty `webhook.method` on `checkly_alert_channel`, or a `slack_app.slack_channels` element that ends in whitespace. The API stored `POST` for the empty method and removed the whitespace, so every plan showed the same change. Remove the empty method to get the default, `POST`, and remove the whitespace. An empty `email.address`, which the API stored although no alert can reach it, fails the plan too.
- A `path` on `checkly_client_certificate` with a character that the API percent-encodes, such as a non-ASCII letter or `{`, which replaced the certificate on every apply. Write the character percent-encoded, as the API stores it, for example `/caf%C3%A9` for `/café`.
- A secret environment variable value shorter than 3 characters, in `checkly_environment_variable` or in `environment_variables` of a check, check group or Playwright check suite, when the plan would send it: a new variable, a changed value, or any update of `checkly_environment_variable`. The API rejects such values at apply, since runners mask every occurrence of a secret value in the check output. A short value stored before the API added the minimum only gives a warning while the plan leaves it as it is. Use a longer value or make the variable not secret.
- A changed `slug_name` on an existing `checkly_private_location`. The API never changed the slug, so every plan showed the change again. Set `slug_name` back to the slug the location has in Checkly, or create a new private location with the new slug.
- A `request.port` above 65535 on `checkly_tcp_monitor` or `checkly_tcp_check`. It was sent modulo 65536, so the monitor connected to another port; set the port you meant.
- A `degraded_response_time` or `max_response_time` of `0` on `checkly_grpc_monitor` or `checkly_ssl_monitor`, or a `degraded_response_time` of `0` on `checkly_traceroute_monitor`. The `0` was not sent, so the monitor kept its default or an earlier value; set the value you want, or remove the attribute to use the default.
- Attributes of `checkly_grpc_monitor` that don't apply to its `grpc_mode`, such as `request.method` in `HEALTH` mode or `request.service` in `BEHAVIOR` mode, and `request.port` on a `checkly_traceroute_monitor` with `request.protocol = "ICMP"`. The API dropped them; remove them.
- A numeric assertion target with leading or trailing whitespace on `checkly_grpc_monitor` or `checkly_traceroute_monitor`, such as `" 100"`. The API removed the whitespace; remove it.
- A negative `heartbeat.grace` on `checkly_heartbeat_monitor` or `checkly_heartbeat`.
- A `heartbeat.ping_token` on `checkly_heartbeat_monitor` or `checkly_heartbeat` that is not a lowercase UUID with hyphens, such as `22868839-8450-4010-9241-1ea83a2e425f`. The API stored an uppercase or unhyphenated token in that form, so every plan showed a change, and rejected other spellings; set the token as it appears in Checkly.

### Settings that couldn't work

Some settings that v1.x accepted, and that the API accepts too, couldn't work. They now fail the plan:

- A `runtime.playwright.devices` `type` of `checkly_playwright_check_suite` other than `chromium`, `firefox`, `webkit`, `chrome` and `msedge`. The check runners don't provide other browsers.
- An empty `runtime.steps.install.command` or `runtime.steps.test.command` of `checkly_playwright_check_suite`. The API doesn't store an empty command, so it showed as a change in every plan; leave the attribute out to use the default command.
- A negative `retry_strategy.base_backoff_seconds`. A retry can't wait a negative time; use `0` to retry without waiting.
- A scheme or a port in the host of `checkly_tcp_monitor`, `checkly_tcp_check`, `checkly_ssl_monitor`, `checkly_grpc_monitor`, `checkly_traceroute_monitor` or `checkly_icmp_monitor`, such as `https://example.com` or `example.com:443`. The monitor tried to resolve the scheme or port as part of the host name. Remove them, and set the port with the port attribute on the monitors that have one.
- A `LATENCY` assertion of `checkly_icmp_monitor`, a `RESPONSE_TIME` assertion of `checkly_tcp_monitor` or `checkly_tcp_check`, or a `RESPONSE_TIME` assertion of `checkly_dns_monitor` with `EQUALS`, `NOT_EQUALS`, `GREATER_THAN` or `LESS_THAN`, whose `target` is not a non-negative number of milliseconds, such as a missing target or `" 100"`. The assertion failed, or always passed with `NOT_EQUALS`. Set the number of milliseconds without surrounding whitespace.

### Debug log removed

The `CHECKLY_DEBUG_LOG` environment variable no longer writes API requests and responses to a file. The file included the API key and secret values in plain text. To inspect the provider's API traffic, route it through an HTTPS proxy with the `HTTPS_PROXY` environment variable, which the provider honors. An intercepting proxy only works once its CA certificate is trusted, for example through the system keychain on macOS or `SSL_CERT_FILE` on Linux. Use `TF_LOG` for Terraform's own logs.

## Removed resource types

`checkly_trigger_check` and `checkly_trigger_group` are gone. They managed command line trigger URLs, which Checkly has deprecated. To run checks on demand, use the [Checkly CLI](https://www.checklyhq.com/docs/cli/) (`npx checkly trigger`) or the check sessions trigger endpoint of the Checkly API (`POST /v1/check-sessions/trigger`).

v2 fails to plan while these resource types are in state. Remove them from your configuration, then either:

- Delete the triggers by running `terraform apply` while still on v1.x.
- Keep the trigger URLs working for now by removing the resources from state only. While still on v1.x, add a `removed` block for each resource and run `terraform apply`:

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

`checkly_check` now writes through the API endpoint of each check type, which only stores what applies to that type. `request`, `frequency_offset`, `degraded_response_time`, `max_response_time`, `setup_snippet_id`, `teardown_snippet_id`, `local_setup_script` and `local_teardown_script` have no effect on browser and multi-step checks, and neither do `script` on API checks and `ssl_check_domain` on multi-step checks: changing one only changes the state, and setting one gives a warning at plan time. Remove them, or move the check to its per-type resource, which doesn't have them. A browser or multi-step check with an empty `script` is an error, on `checkly_check` as on the per-type resources, which the plan reports when the script is known there; with `ignore_changes = [script]`, the stored script counts.

Every update of a browser or multi-step `checkly_check` clears the script path and dependencies that the Checkly CLI sets for multi-file scripts, because `checkly_check` has no attributes for them. v1.x left them alone. To keep them, move the check to `checkly_browser_check` or `checkly_multistep_check` and set `script_path` and `dependencies` there, since those resources also clear values that are absent from the configuration. Or keep managing the check with the Checkly CLI.

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

When moving an API check from `checkly_check` to `checkly_api_check`, note that the default `degraded_response_time` and `max_response_time` are 5000 and 20000 instead of 15000 and 30000. Set them explicitly to keep the previous thresholds.

Check groups move to `checkly_check_group_v2` with a `moved` block too, but their configuration changes more than the resource type; see [Check groups](#check-groups).

### Check groups

`checkly_check_group` is deprecated in favor of `checkly_check_group_v2`. A v1 group always applies its alert settings, retry strategy and scheduling strategy to its checks, and its locations when it has any; a v2 group only applies the settings you enforce with its `enforce_*` attributes.

Both resource types manage the same check groups, so you can switch a group over without recreating it or its checks: change the resource type to `checkly_check_group_v2`, translate its attributes as shown below, and add a `moved` block. This works in the same plan as the provider upgrade, and the group keeps the values that the API never returns, such as secret environment variables.

```terraform
# v1.x
resource "checkly_check_group" "example" {
  name                      = "Example group"
  activated                 = true
  concurrency               = 2
  run_parallel              = true
  locations                 = ["eu-central-1"]
  use_global_alert_settings = false
  local_setup_script        = "console.log('setup')"
  runtime_id                = "2024.09"

  alert_settings {
    escalation_type = "RUN_BASED"
    run_based_escalation {
      failed_run_threshold = 2
    }
  }

  retry_strategy {
    type = "FIXED"
  }

  alert_channel_subscription {
    channel_id = checkly_alert_channel.example.id
    activated  = true
  }
}

# v2
resource "checkly_check_group_v2" "example" {
  name        = "Example group"
  activated   = true
  concurrency = 2

  setup_script = {
    inline_script = "console.log('setup')"
  }

  default_runtime = {
    runtime_id = "2024.09"
  }

  enforce_locations = {
    enabled   = true
    locations = ["eu-central-1"]
  }

  enforce_scheduling_strategy = {
    enabled      = true
    run_parallel = true
  }

  enforce_alert_settings = {
    enabled = true
    alert_settings = {
      escalation_type = "RUN_BASED"
      run_based_escalation = {
        failed_run_threshold = 2
      }
    }
    alert_channel_subscriptions = [
      {
        channel_id = checkly_alert_channel.example.id
        activated  = true
      },
    ]
  }

  enforce_retry_strategy = {
    enabled = true
    retry_strategy = {
      type = "FIXED"
    }
  }
}

moved {
  from = checkly_check_group.example
  to   = checkly_check_group_v2.example
}
```

The attributes translate as follows. The left column uses the v2 names of `checkly_check_group`; v1.x configurations use the v1.x forms, for example the singular `alert_channel_subscription` and `environment_variable` blocks and the deprecated `environment_variables` map (see [Collections are plural attributes](#collections-are-plural-attributes)).

| `checkly_check_group` | `checkly_check_group_v2` |
|---|---|
| `name`, `activated`, `muted`, `concurrency`, `tags`, `api_check_defaults`, `environment_variables` | The same attributes. |
| `locations`, `private_locations` | `enforce_locations = { enabled = true, locations = [...], private_locations = [...] }`. Leave `enforce_locations` out if the group has neither. |
| `run_parallel` | `enforce_scheduling_strategy = { enabled = true, run_parallel = ... }`, with `run_parallel = false` if the v1 group does not set it. |
| `use_global_alert_settings`, `alert_settings`, `alert_channel_subscriptions` | `enforce_alert_settings = { enabled = true, alert_settings = { ... }, alert_channel_subscriptions = [...] }`, with `use_global_alert_settings = ...` if the v1 group sets it. |
| `retry_strategy` | `enforce_retry_strategy = { enabled = true, retry_strategy = { ... } }`, with `retry_strategy = { type = "NO_RETRIES" }` if the v1 group does not set one. A group whose stored retry strategy is `FALLBACK`, set outside Terraform, lets each check use its own and moves without the enforcement: leave `enforce_retry_strategy` out. |
| `double_check = true` | The retry strategy shown under [Deprecated attributes](#deprecated-attributes), as `enforce_retry_strategy.retry_strategy`. `double_check` itself has no equivalent and needs no conversion before the move. |
| `setup_snippet_id` or `local_setup_script` | `setup_script = { snippet_id = ... }` or `setup_script = { inline_script = ... }`. |
| `teardown_snippet_id` or `local_teardown_script` | `teardown_script = { snippet_id = ... }` or `teardown_script = { inline_script = ... }`. |
| `runtime_id` | `default_runtime = { runtime_id = ... }`. |

Some values need attention:

- `use_global_alert_settings`: leave `enforce_alert_settings.use_global_alert_settings` out unless the v1 group sets it. While the enforcement is enabled, it is then derived as for `checkly_check_group`: `true` if `enforce_alert_settings` does not set `alert_settings`, and `false` if it does. The move keeps the value stored for the group, so the plan shows an update of `use_global_alert_settings` where the stored value differs from the derived one, for example on a group that v1.x stored with `false` and that has no `alert_settings` (see [Expected one-time changes](#expected-one-time-changes)). To keep the stored value instead, set it explicitly.
- `alert_settings`: a v1 group created without `alert_settings` still has alert settings, because the provider sent the API's defaults. They move with the group, so `enforce_alert_settings` can leave `alert_settings` out and the plan keeps the stored settings. They don't apply, though: without `alert_settings` in the configuration, `use_global_alert_settings` is derived as `true`, so the group's checks follow the account level alert settings. To apply the stored settings, set `use_global_alert_settings = false`.
- A group with both a snippet and a local script for the same phase: see [Expected one-time changes](#expected-one-time-changes).
- Point references such as `group_id` at `checkly_check_group_v2.example.id`. The ID does not change, so the checks plan no update.

Run `terraform plan`. The move should plan no changes, except for the update of `use_global_alert_settings` described above, which a group that v1.x stored with `false` and that sets neither `use_global_alert_settings` nor `alert_settings` plans as `false -> true` (see [Expected one-time changes](#expected-one-time-changes)). If it plans any other update, adjust the configuration until it does not, unless you mean to change the group. Once the group has moved, removing an `enforce_*` attribute lets its checks use their own setting, which the plan shows as an update of the group.

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

- v1.x stored an unset optional string or number attribute as `""` or `0`. v2 stores null instead for one without a documented default, such as `runtime_id`, `frequency_offset` or a maintenance window's `repeat_interval`; an attribute with a documented default, such as a check's `description` or `activated`, keeps storing that default. `group_id`, `group_order`, `setup_snippet_id` and `teardown_snippet_id` are stored as null when unset, and setting `group_id` or a snippet ID to `0` fails the plan; see [No group or snippet is null](#no-group-or-snippet-is-null). If your configuration sets an attribute that v2 stores as null to `""` or `0` explicitly, for example `repeat_interval = 0`, the first plan shows an in-place update to that value. The same goes for the first plan after an import, which also stores null for such a value. Removing the attribute from the configuration has the same effect and avoids the update. Unset collections are stored as empty lists, sets or maps, as in v1.x, so `tags = []` and leaving `tags` out both plan no change.
- v1.x could not clear these attributes: leaving one out kept the stored value, and every plan showed the same change. v2 clears them, so a value set in the web app and absent from your configuration is erased by the first apply. To keep such a value, add it to your configuration.
  - `checkly_status_page`: `custom_domain`, `logo`, `redirect_to` and `favicon`.
  - `checkly_dashboard`: `custom_domain`, `logo`, `link`, `description` and `favicon`.
  - `checkly_check`: `tags` and `group_order`, and on API checks only `setup_snippet_id`, `teardown_snippet_id`, `local_setup_script`, `local_teardown_script` and `request.basic_auth`.
  - `checkly_check_group`: `setup_snippet_id`, `teardown_snippet_id`, `local_setup_script` and `local_teardown_script`.
  - `checkly_url_monitor`, `checkly_tcp_monitor`, `checkly_tcp_check`, `checkly_dns_monitor`, `checkly_icmp_monitor`, `checkly_grpc_monitor`, `checkly_ssl_monitor` and `checkly_traceroute_monitor`: `tags` and `group_order`.
  - `checkly_heartbeat_monitor` and `checkly_heartbeat`: `tags`.
  - `checkly_tcp_monitor` and `checkly_tcp_check`: also `request.assertions` and `request.data`.
  - `checkly_traceroute_monitor`: also `request.assertions`, and `request.timeout`, which goes back to its default of `10`.
  - `checkly_ssl_monitor`: also `request.assertions`, `request.server_name` and `request.client_certificate.client_certificate_id`.
  - `checkly_grpc_monitor`: also `request.proto_content` with `service_definition = "REFLECTION"`.
  - Every resource with `alert_settings`: `reminders.amount` and `parallel_run_failure_threshold.enabled`, when the configured object leaves them out.
- `checkly_check`, `checkly_playwright_check_suite`, and checks moved from `checkly_check` to the per-type resources: an `intent` or `ai_auto_repair_enabled` value set outside Terraform and absent from your configuration is removed by the first apply, which the plan shows. To keep such a value, add it to your configuration, or add the attribute to `ignore_changes` to leave it to the web app.
- `checkly_check` with `type = "BROWSER"` or `type = "MULTI_STEP"` whose `should_fail` is `true` in Checkly: once `should_fail` is removed from the configuration, the first plan shows an in-place update of `should_fail` to `false`. Apply that update before you move the check to `checkly_browser_check` or `checkly_multistep_check`: those resources have no `should_fail`, so a check moved with `true` keeps it in Checkly until an update for another change sends `false`. A move, import or refresh of such a check warns with `should_fail Is Not Supported`. See [should_fail only on API checks, URL monitors and TCP monitors](#should_fail-only-on-api-checks-url-monitors-and-tcp-monitors).
- `checkly_ssl_monitor`, `checkly_tcp_monitor` and `checkly_tcp_check` that leave out `degraded_response_time` or `max_response_time`: their v1.x defaults didn't match the API's, so the first apply lowers them to the new defaults. They are `3000` and `10000` on `checkly_ssl_monitor` (from `10000` and `20000`), and `3000` for `degraded_response_time` on `checkly_tcp_monitor` and `checkly_tcp_check` (from `4000`). To keep the old limits, set them in the configuration. A `checkly_ssl_monitor` that sets only `degraded_response_time`, to a value above `10000`, fails the plan, because `max_response_time` must be at least as high; set `max_response_time` as well.
- `checkly_traceroute_monitor` with `max_response_time = 0`: v1.x did not send the `0`, so the monitor kept its default of `20000` or an earlier value. v2 sends it, and `0` means the monitor never fails on response time, so the first apply removes the monitor's failing response time limit. To keep a limit, set `max_response_time` to it.
- `checkly_dns_monitor`, `checkly_icmp_monitor` and `checkly_traceroute_monitor`: `private_locations` set outside Terraform and absent from your configuration are removed by the first apply, which the plan shows. The same goes for `proxy_url` on `checkly_private_location`. To keep such a value, add it to your configuration, or add the attribute to `ignore_changes` to leave it to the web app.
- `checkly_ssl_monitor`: v1.x left the `request.security_baseline` rules that your configuration doesn't list unmanaged. In v2, a rule left out of the configuration, or the whole `security_baseline`, means its default, so the first apply resets a rule that was changed in the web app and isn't in your configuration. To keep such a value, add the rule to your configuration.
- A check or monitor added to a group in the web app, with no `group_id` in its configuration, leaves the group on the first apply, as it did with v1.x.
- `checkly_maintenance_window` and `checkly_maintenance_windows` whose `starts_at`, `ends_at` or `repeat_ends_at` is written differently from the API's form, such as `2099-06-01` or `2099-06-01T09:00:00+02:00` instead of `2099-06-01T00:00:00.000Z`: after the upgrade from v1.x, or after an import, the state holds the API's form, so the first plan shows an in-place update to the configured form. It changes nothing in Checkly, and from then on the configured form is kept. With v1.x, such a value showed a change on every plan.
- `checkly_dashboard`: every update with v1.x reset "sort by status" and "show group names" to the API defaults. v2 manages them as `sort_by_status` (default `false`) and `show_group_names` (default `true`), so the first plan shows an in-place update of a dashboard whose web app settings differ from the defaults. To keep the web app settings, add them to your configuration.
- `checkly_alert_channel`: `auto_subscribe` is new and defaults to `false`. A channel that has automatic subscription of new checks turned on in the web app plans an in-place update of `auto_subscribe` from `true` to `false`. v1.x turned it off with any update of the channel. To keep it on, set `auto_subscribe = true`.
- `checkly_maintenance_window` and `checkly_maintenance_windows`: links to v3 status page components set in the web app and absent from your configuration are removed by the first apply, which the plan shows. If the v3 page stays in `status_page_ids` without an entry in `component_configs`, the apply is rejected instead. To keep the links, add them to `component_configs`, or leave them to the web app with `ignore_changes`: `status_page_visibility.component_configs` when your configuration sets `status_page_visibility`, or `status_page_visibility` as a whole otherwise, which leaves all its settings to the web app. An account without the status page maintenance windows entitlement can't remove the links, so every update of such a window is rejected until they are in the configuration or ignored.
- `checkly_check` and `checkly_check_group` with an `environment_variables` map of several entries: v1.x sent the map's variables in no particular order. If that order differs from the order of the list in your configuration, the first plan shows the variables reordered, and applying it stores them in the configured order.
- `checkly_client_certificate`: v1.x stored `path` without a trailing slash. If your configuration has one, for example `path = "/partner/api/"`, the first plan shows an in-place update of `path` that does not change the certificate. A certificate imported with v1.x whose configuration was never applied shows an in-place update of `private_key`, and of `passphrase` if you set one, that also does not change the certificate in Checkly.
- `use_global_alert_settings` no longer has a fixed default. When it is not set, it is `true` if `alert_settings` is not set either, and `false` if it is, as in the Checkly CLI. v1.x defaulted it to `false` on most resources and to `true` on a few, so the first plan can show an in-place update of `use_global_alert_settings` in one of two directions:
  - `checkly_check`, `checkly_api_check`, `checkly_browser_check`, `checkly_multistep_check`, `checkly_check_group`, `checkly_tcp_monitor`, `checkly_tcp_check`, `checkly_grpc_monitor`, `checkly_ssl_monitor`, `checkly_traceroute_monitor`, `checkly_heartbeat_monitor` and `checkly_heartbeat` that set neither attribute plan `false -> true`. From then on their alerts follow the account level alert settings instead of their own alert settings, which were the API defaults. To keep the old behavior, set `use_global_alert_settings = false`.
  - `checkly_url_monitor`, `checkly_dns_monitor`, `checkly_icmp_monitor` and `checkly_playwright_check_suite` that set `alert_settings` but not `use_global_alert_settings` plan `true -> false`. From then on their configured `alert_settings` take effect; with v1.x the account level alert settings applied instead. To keep the old behavior, set `use_global_alert_settings = true` and remove `alert_settings`.
  - `checkly_check_group_v2` derives `enforce_alert_settings.use_global_alert_settings` from `enforce_alert_settings.alert_settings` in the same way while `enforce_alert_settings.enabled` is `true`. A group with `enabled = true` that sets neither of the two plans `false -> true`. From then on its checks follow the account level alert settings instead of the fallback alert settings that the group enforced. To keep the old behavior, set `use_global_alert_settings = false` in `enforce_alert_settings`. A disabled enforcement keeps its stored value, which the API ignores, and plans no change.

  Setting `alert_settings` together with `use_global_alert_settings = true` gives a warning at plan time: the account level alert settings then apply, and `alert_settings` has no effect.

  A `checkly_check_group` moved to `checkly_check_group_v2` keeps its stored value and plans the same update as `checkly_check_group_v2` where that value differs from the derived one; see [Check groups](#check-groups).
- `checkly_check_group` moved to `checkly_check_group_v2` with both a snippet and a local script for the same phase, such as `setup_snippet_id` and `local_setup_script`: `checkly_check_group_v2` takes one of them, so the first plan updates the group and clears the one your configuration leaves out. Keep the one you want the group's checks to run. If you are unsure which one runs today, look at the recent results of the group's API checks before you apply.
