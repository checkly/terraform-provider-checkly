# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Releases from v1.6.7 through v1.32.0 are described in their
[GitHub release notes](https://github.com/checkly/terraform-provider-checkly/releases)
instead of this file.

## [v2.0.0] - Unreleased

The provider moves from the Terraform Plugin SDK v2 to the Terraform Plugin
Framework. State written by v1.x is upgraded automatically; configurations
need changes. Follow the [upgrade guide](docs/guides/upgrading-to-v2.md).

### Changed
- **Breaking:** single nested objects that were blocks are attributes (`alert_settings = { ... }` instead of `alert_settings { ... }`).
- **Breaking:** no resource has blocks any more. Every block that could be repeated is a list or set attribute with a plural name (`alert_channel_subscriptions = [{ ... }]` instead of `alert_channel_subscription { ... }` blocks), so it can be assigned from an expression instead of `dynamic` blocks. This renames the blocks `alert_channel_subscription` (also inside `enforce_alert_settings` of `checkly_check_group_v2`), `environment_variable`, `component`, `card` and `card.service_attachment`, and the blocks nested in single nested attributes `request.assertion`, `api_check_defaults.assertion` and `runtime.playwright.device`. State is upgraded automatically; see [Collections are plural attributes](docs/guides/upgrading-to-v2.md#collections-are-plural-attributes) in the upgrade guide.
- **Breaking:** the provider serves plugin protocol 6 only and requires Terraform 1.8 or later.
- **Breaking:** every API key needs an account ID, and it must be a UUID. Account-scoped keys used without `account_id` or `CHECKLY_ACCOUNT_ID` stop with a `Missing Checkly Account ID` error.
- **Breaking:** the first apply erases values that are set in the web app and absent from your configuration, for the attributes v1.x could not clear. See [Expected one-time changes](docs/guides/upgrading-to-v2.md#expected-one-time-changes) in the upgrade guide.
- **Breaking:** credentials are sensitive, including nested ones, so Terraform redacts them in plan output. An output that exposes one, such as a heartbeat ping URL built from `ping_token`, must set `sensitive = true`. Header and query parameter values, of checks, gRPC monitor metadata and webhook alert channels, are not sensitive, as in v1.x; redact them with a sensitive variable or `sensitive(...)`.
- An invalid `API_CALL_TIMEOUT` is reported as a configuration error instead of crashing the provider.
- The provider uses `checkly-go-sdk` v2, and `checkly_check` writes through the API endpoint of each check type. Attributes that don't apply to the type (`request`, `frequency_offset`, `degraded_response_time`, `max_response_time`, `setup_snippet_id`, `teardown_snippet_id`, `local_setup_script` and `local_teardown_script` on browser and multi-step checks, `script` on API checks and `ssl_check_domain` on multi-step checks) no longer reach the API, so changing one only changes the state. Setting one gives a warning at plan time.
- `double_check` is not read back from the API. An import stores `false`, so importing a check or group with `double_check = true` in its configuration plans one in-place update.
- An empty `api_key`, `api_url` or `account_id` in the provider configuration, such as a variable that defaults to `""`, counts as not set, so the matching environment variable applies. An empty `api_key` used to stop the provider, and an empty `api_url` used to select the default URL over `CHECKLY_API_URL`.
- **Breaking:** every update of a browser or multi-step `checkly_check` clears the script path and dependencies that the Checkly CLI sets for multi-file scripts, which v1.x left alone. To keep them, move the check to `checkly_browser_check` or `checkly_multistep_check` and set `script_path` and `dependencies` there.
- A browser or multi-step `checkly_check` with an empty `script` fails with a `Missing Check Script` error: in the plan when the script is known there, otherwise before any request. With `ignore_changes = [script]`, the stored script counts.
- API error messages name the request, the status and the API's own message, for example `GET /v1/checks/…: 404 Not Found`. Scripts that matched the old `unexpected response status` text need updating.
- A request that gets no response, for example because its connection was reset or another request broke the shared connection, is re-sent up to twice when re-sending cannot apply a change twice: reads, updates and deletes, but not creates. Failures a retry would only repeat, such as a timeout, a DNS or certificate failure or a refused connection, are not re-sent. The error says how many times a request was re-sent.
- An API key with a control character, such as a trailing newline, fails with an `Invalid Checkly API Key` error when the provider is configured, instead of failing every request.
- `checkly_playwright_check_suite`: leaving out `runtime.playwright.devices` selects the default devices (chromium, firefox and webkit) also when `runtime.auto_detect` is `false`, which used to fail the plan.
- `checkly_ssl_monitor`: a `request.security_baseline` rule left out of the configuration, or the whole attribute, now means its default instead of leaving the stored value unmanaged.
- **Breaking:** values the API rejects or changes because of their length or format fail at plan time. Some configurations that applied with v1.x now fail the plan: an uppercase `url` on `checkly_status_page`, which the API lowercased, and an oversized element of `environment_variables` on a check or check group, which the API dropped without an error. Other values the limits catch already failed at apply. Each attribute's documentation states its length limit. The limits cover:
  - `environment_variables` (at most 200 elements, with values of at most 64000 UTF-8 bytes, or 31744 for secret or locked ones) and the `value` of `checkly_environment_variable`.
  - `name` and `description` of `checkly_maintenance_window` and `checkly_maintenance_windows`.
  - `url` of `checkly_status_page`, `slug_name` of `checkly_private_location`, and `host` and `passphrase` of `checkly_client_certificate`.
  - `name` of `checkly_status_page_v3_component`, which must also have no leading or trailing whitespace, and `name` of `checkly_status_page_v3_automation_rule`.
  - `request.url` of `checkly_api_check`, `checkly_check` and `checkly_url_monitor`, and `request.hostname` of `checkly_tcp_monitor`, `checkly_tcp_check` and `checkly_icmp_monitor`.
  - `request.host` of `checkly_grpc_monitor`, `request.hostname` and `request.server_name` of `checkly_ssl_monitor`, and `request.url` of `checkly_traceroute_monitor`. These must also have no control characters.
  - `request.query` and `request.name_server.host` of `checkly_dns_monitor`.
  - `request.metadata` (at most 50 elements, with limits on each `key` and `value`, and keys that are unique ignoring case), `request.proto_content` and `request.message` of `checkly_grpc_monitor`. The metadata keys and values, `request.proto_content` and `request.message` must also have no NUL characters.
  - The `path` of `dependencies` elements of `checkly_browser_check` and `checkly_multistep_check`.
  - `starts_at`, `ends_at` and `repeat_ends_at` of `checkly_maintenance_window` and `checkly_maintenance_windows`, which must be RFC 3339 date-times with a UTC offset or `YYYY-MM-DD` dates. v1.x accepted any string; a value the API read in another way, such as a date-time without a UTC offset, applied and then showed a change on every plan. `repeat_interval` must be 0 to 365, `repeat_unit` and a `repeat_interval` of 1 or more must be set together, and `repeat_ends_at` needs such a `repeat_interval`.
  - `custom_domain` of `checkly_status_page`, which must be a lowercase hostname without a scheme, as on `checkly_status_page_v3`. The API lowercased it and removed a scheme, which v1.x showed as a change on every plan.
  - `description`, `footer_text`, `privacy_policy_link`, `terms_of_service_link`, `support_link` and `google_analytics_tag` of `checkly_status_page_v3`. `google_analytics_tag` must have only letters, digits and dashes.
  - URLs, which must be http or https URLs or empty: `logo`, `logo_dark`, `redirect_to`, `favicon`, `privacy_policy_link` and `terms_of_service_link` of `checkly_status_page_v3`, `logo`, `redirect_to` and `favicon` of `checkly_status_page`, and `logo`, `favicon` and `link` of `checkly_dashboard`. `support_link` of `checkly_status_page_v3` may also be a mailto URL.
  - `custom_url` of `checkly_dashboard`: letters, digits and dashes, with no leading or trailing dash.
  - `first_update`, `last_update` and the `tags` of `checkly_status_page_v3_automation_rule`, which must not be empty, and a `component_id` listed more than once in its `components`.
- **Breaking:** `use_global_alert_settings` is derived from `alert_settings` when it is not set, as in the Checkly CLI: `true` when `alert_settings` is not set either, and `false` when it is. v1.x defaulted it to `false` on most resources and to `true` on `checkly_url_monitor`, `checkly_dns_monitor`, `checkly_icmp_monitor` and `checkly_playwright_check_suite`. The first plan updates it in place on resources that relied on the old default: from `false` to `true` on `checkly_check_group` and the other checks and monitors when neither attribute is set, so their alerts follow the account level alert settings, and from `true` to `false` on those four resources when `alert_settings` is set, so it takes effect. `enforce_alert_settings.use_global_alert_settings` of `checkly_check_group_v2` is derived from `enforce_alert_settings.alert_settings` in the same way while the enforcement is enabled; v1.x defaulted it to `false`, so a group with `enabled = true` that sets neither plans it from `false` to `true`. Setting `alert_settings` together with `use_global_alert_settings = true` gives a warning at plan time, since `alert_settings` then has no effect. See [Expected one-time changes](docs/guides/upgrading-to-v2.md#expected-one-time-changes) in the upgrade guide.
- **Breaking:** an optional string or number attribute without a default value that is not configured, such as `runtime_id`, `frequency_offset` or a maintenance window's `repeat_interval`, is stored as null instead of `""` or `0`. A configuration that sets such an attribute to `""` or `0` explicitly plans one in-place update after the upgrade or an import. See [Expected one-time changes](docs/guides/upgrading-to-v2.md#expected-one-time-changes) in the upgrade guide.
- **Breaking:** `group_id` and `group_order` on every check and monitor resource, and `setup_snippet_id` and `teardown_snippet_id` on `checkly_check`, `checkly_api_check` and `checkly_check_group`, are null when not set instead of `0`. `group_id` and the snippet IDs must be at least `1`: a configuration that sets one of them to `0`, for example `group_id = var.grouped ? checkly_check_group.example.id : 0`, fails the plan; remove the attribute, or use `null` instead of `0`. `group_order` may be `0`, the first position in a group, as in the web app. State written by v1.x is upgraded without a change, except that a configured `group_order = 0` plans one in-place update. See [No group or snippet is null](docs/guides/upgrading-to-v2.md#no-group-or-snippet-is-null) in the upgrade guide.
- The descriptions of the attributes that checks, monitors and check groups share are consistent. `locations` no longer claims a default of `us-east-1`, which the API never applied.
- `activated` and `frequency` are optional on every check, monitor and `checkly_check_group` resource that has them, with the Checkly API defaults `true` and `10`. They were required everywhere except `checkly_api_check`, `checkly_browser_check` and `checkly_multistep_check`.
- **Breaking:** some check settings that v1.x accepted but that couldn't work fail the plan: a `runtime.playwright.devices` `type` of `checkly_playwright_check_suite` other than `chromium`, `firefox`, `webkit`, `chrome` and `msedge`, which the check runners don't provide; an empty `runtime.steps.install.command` or `runtime.steps.test.command`, which the API doesn't store, so it showed as a change in every plan; and a negative `retry_strategy.base_backoff_seconds` on any resource, which can't delay a retry.
- Configurations that the API rejected at apply fail the plan instead: a `checkly_check` with `type = "API"` and no `request`, a `checkly_check` with `type = "MULTI_STEP"` and `frequency = 0` (only `type = "BROWSER"` was caught), and a `request.method` of `checkly_api_check` or `checkly_check` other than `GET`, `POST`, `PUT`, `HEAD`, `DELETE` and `PATCH`, such as a lowercase `get`.
- `checkly_api_check` and `checkly_check`: empty basic auth credentials from the API are read as no `request.basic_auth`, so importing a check created in the web app without basic authentication plans no change for a configuration without `basic_auth`. A configuration that sets an empty `username` and `password` keeps them.
- **Breaking:** more values that the API rejects, changes or never applies fail at plan time:
  - An empty `name` or `icon` of `checkly_private_location`, `key` or `value` of `checkly_environment_variable`, `name` or `script` of `checkly_snippet`, and `certificate` or `private_key` of `checkly_client_certificate`. An empty `private_key` created a certificate without a key.
  - On `checkly_alert_channel`: an empty `email.address`, `webhook.name`, `webhook.url`, `opsgenie.name`, `opsgenie.api_key` or `pagerduty.service_key`; an `opsgenie.region` other than `EU` or `US`; an `opsgenie.priority` other than `P1` to `P5`; an empty `webhook.method`, which the API stored as `POST`, so every plan showed the same change; and a `slack_app.slack_channels` element that doesn't start with `#` or `@`, or that ends in whitespace, which the API removed.
  - A `path` of `checkly_client_certificate` that the API rewrites or rejects. The API percent-encodes non-ASCII characters and the characters `` "<>^`{} ``, which replaced the certificate on every apply, and rejects whitespace, control characters, `//`, `.` and `..` segments, `?`, `#` and `\`, which used to fail at apply. Write the characters it encodes percent-encoded, for example `%7B` for `{`.
- **Breaking:** a secret environment variable value shorter than 3 characters fails the plan when the plan would send it, which the API rejects because runners mask every occurrence of a secret value in the check output: when `checkly_environment_variable` is created or updated in any way, and when an element of `environment_variables` of a check, check group or Playwright check suite is new, newly secret or has a changed value, including an empty secret value. A short secret value that the plan leaves as it is only gives a warning, since the API keeps the stored value. Characters are Unicode code points, so an emoji counts as one.
- **Breaking:** changing `slug_name` of an existing `checkly_private_location` fails the plan. The API never changes a slug, so the update had no effect and every plan showed it again. This includes `terraform apply -replace` and `replace_triggered_by`. To use a different slug, add a new private location resource with it, move your checks and agents to it, then remove the old one. Replacing the location would fail while checks or groups use the location, and would disconnect its agents, whose API keys belong to the old location.
- `checkly_environment_variable`: changing `secret` from `true` to `false` replaces the variable. The API refuses to turn a secret variable back into a regular one, so the update used to fail.
- `checkly_static_ips` warns about a `locations` element that has no static IPs, such as a misspelled or private location, and lists the locations that have them. It used to return no addresses for it silently.
- Creating or updating a `slack_app` alert channel on an account without a connected Slack workspace, or with `slack_channels` handles that the workspace cannot resolve, fails with an error on `slack_app` that says how to fix it, instead of the generic API error.
- An invalid `api_url` or `CHECKLY_API_URL` fails with an `Invalid Checkly API URL` error on `api_url` that says what is wrong with it, instead of `Unable to Create the Checkly API Client`.
- **Breaking:** monitor settings that the API rejects, ignores or changes, or that couldn't work, fail at plan time. Some of them applied with v1.x:
  - `request.port` of `checkly_tcp_monitor` and `checkly_tcp_check` must be between 1 and 65535. A larger port used to be sent modulo 65536, so the monitor connected to another port.
  - `degraded_response_time` and `max_response_time` of `checkly_grpc_monitor` and `checkly_ssl_monitor`, and `degraded_response_time` of `checkly_traceroute_monitor`, must be at least `1`. A `0` was not sent, so the API kept its default or the previous value.
  - In `checkly_grpc_monitor`, `request.method`, `request.message`, `request.proto_content` and `request.service_definition = "PROTO_FILE"` cannot be used in `HEALTH` mode, and `request.service` only in `HEALTH` mode; the API dropped them. `BEHAVIOR` mode requires `request.method`, and `request.service_definition = "PROTO_FILE"` requires `request.proto_content`.
  - `request.port` of `checkly_traceroute_monitor` cannot be set with `request.protocol = "ICMP"`, which has no port; the API dropped it.
  - The host of `checkly_tcp_monitor`, `checkly_tcp_check`, `checkly_ssl_monitor`, `checkly_grpc_monitor`, `checkly_traceroute_monitor` and `checkly_icmp_monitor` must not include a scheme, such as `https://`, or a port, such as `:443`. The monitor tried to resolve them as part of the host name.
  - The target of a `LATENCY` assertion of `checkly_icmp_monitor`, of a `RESPONSE_TIME` assertion of `checkly_tcp_monitor` and `checkly_tcp_check`, and of a `RESPONSE_TIME` assertion of `checkly_dns_monitor` with `EQUALS`, `NOT_EQUALS`, `GREATER_THAN` or `LESS_THAN`, must be a non-negative number of milliseconds without leading or trailing whitespace. Any other target failed the assertion, or made a `NOT_EQUALS` assertion always pass.
  - Assertions of `checkly_tcp_monitor`, `checkly_tcp_check`, `checkly_icmp_monitor`, `checkly_grpc_monitor` and `checkly_traceroute_monitor` must use a source, comparison and property that the API accepts for the source. The numeric targets of traceroute and gRPC assertions must not have leading or trailing whitespace, which the API removed.
  - `max_response_time` must be at least `degraded_response_time` on `checkly_ssl_monitor`, and on `checkly_traceroute_monitor` unless it is `0`, and `max_packet_loss_threshold` at least `degraded_packet_loss_threshold` on `checkly_icmp_monitor`, counting an unset attribute as its default. `request.max_unknown_hops` of `checkly_traceroute_monitor` must not exceed `request.max_hops`.
  - `request.client_certificate.mode = "explicit"` of `checkly_ssl_monitor` requires `request.client_certificate.client_certificate_id`.
  - `request.name_server.host` of `checkly_dns_monitor` must not be empty, and `request.name_server.port` must be between 1 and 65535.
  - `heartbeat.ping_token` of `checkly_heartbeat_monitor` and `checkly_heartbeat` must be a lowercase UUID with hyphens, such as `22868839-8450-4010-9241-1ea83a2e425f`, and `heartbeat.grace` must be at least `0`.
- **Breaking:** response time defaults that didn't match the API's now do:
  - `degraded_response_time` and `max_response_time` of `checkly_ssl_monitor` default to `3000` and `10000`, instead of `10000` and `20000`. With the default `request.handshake_timeout_ms` of `10000`, the old failing limit could never be reached.
  - `degraded_response_time` of `checkly_tcp_monitor` and `checkly_tcp_check` defaults to `3000`, instead of `4000`.

  A monitor that leaves them out plans an update to the new values. On `checkly_ssl_monitor`, a configuration that sets only `degraded_response_time`, to a value above `10000`, fails the plan; set `max_response_time` as well. See [Expected one-time changes](docs/guides/upgrading-to-v2.md#expected-one-time-changes) in the upgrade guide.
- **Breaking:** `max_response_time = 0` on `checkly_traceroute_monitor` means the monitor never fails on response time. v1.x did not send the `0`, so the monitor kept its default of `20000` or the previous value. See [Expected one-time changes](docs/guides/upgrading-to-v2.md#expected-one-time-changes) in the upgrade guide.
- `degraded_response_time` and `max_response_time` of `checkly_tcp_monitor`, `checkly_tcp_check`, `checkly_url_monitor`, `checkly_api_check` and `checkly_check` accept values up to `45000` if extended limits have been approved for your account; `checkly_check` used to accept up to `30000`. On other accounts, a value above `5000` on TCP monitors, or above `30000` on URL monitors and API checks, fails the plan.

### Added
- `checkly_api_check`, `checkly_browser_check` and `checkly_multistep_check`, which replace `checkly_check`.
- `checkly_maintenance_window`, which replaces `checkly_maintenance_windows`.
- `component_configs` on `status_page_visibility` of `checkly_maintenance_window` and `checkly_maintenance_windows`, which shows a maintenance window on v3 status pages with the affected components. Leaving it out removes the window's component links, including ones set in the web app; see the upgrade guide.
- `intent` and `ai_auto_repair_enabled` on `checkly_check`, `checkly_api_check`, `checkly_browser_check`, `checkly_multistep_check` and `checkly_playwright_check_suite`. A `moved` block from `checkly_check` to a per-type resource keeps them. A value set outside Terraform and absent from the configuration is removed by the next apply, and `ignore_changes` leaves it to the web app.
- `private_locations` on `checkly_dns_monitor`, `checkly_icmp_monitor` and `checkly_traceroute_monitor`, and `proxy_url` on `checkly_private_location`. A value set outside Terraform and absent from the configuration is removed by the next apply, and `ignore_changes` leaves it to the web app.
- Import instructions on every importable resource's documentation page.
- `ip_family = "all"` on the `checkly_static_ips` data source selects the addresses of both IP families. It is the default, and the data source now reports the selected family in `ip_family`.
- `sort_by_status` and `show_group_names` on `checkly_dashboard`, with the API defaults `false` and `true`. Every update with v1.x reset them to those defaults; now the plan shows that change, so the first plan after the upgrade updates a dashboard whose web app settings differ. See [Expected one-time changes](docs/guides/upgrading-to-v2.md#expected-one-time-changes) in the upgrade guide.
- `auto_subscribe` on `checkly_alert_channel`, which makes the web app preselect the channel for checks created there and the Vercel integration subscribe the checks it creates. Checks created through the API, the Checkly CLI or Terraform are not subscribed; use `alert_channel_subscriptions` for them. Every update used to turn the setting off, also when it was turned on in the web app; now it follows the configuration, with a default of `false`. See [Expected one-time changes](docs/guides/upgrading-to-v2.md#expected-one-time-changes) in the upgrade guide.
- `record_type = "HTTPS"` and every assertion comparison the API accepts on `checkly_dns_monitor`, and the remaining `JSON_RESPONSE` comparisons, such as `CONTAINS` and `NOT_NULL`, on `checkly_icmp_monitor`. The assertion `target` of both can be left out for comparisons that take no value, such as `IS_EMPTY` and `NOT_NULL`.

### Deprecated
- `checkly_check`, in favor of the per-type check resources.
- `checkly_heartbeat`, `checkly_tcp_check` and `checkly_maintenance_windows`, in favor of `checkly_heartbeat_monitor`, `checkly_tcp_monitor` and `checkly_maintenance_window`. They keep working, and `moved` blocks switch them without recreating them.
- `checkly_check_group`, in favor of `checkly_check_group_v2`. A `moved` block switches a group without recreating it; see [Check groups](docs/guides/upgrading-to-v2.md#check-groups) in the upgrade guide.
- `checkly_status_page` and `checkly_status_page_service`, in favor of the v3 status page resources.
- `slack` on `checkly_alert_channel`, in favor of `slack_app`.

### Removed
- **Breaking:** `checkly_trigger_check` and `checkly_trigger_group`. Use the Checkly CLI (`checkly trigger`) or `POST /v1/check-sessions/trigger`.
- **Breaking:** `environment_variables` as a map on `checkly_check` and `checkly_check_group`; write each entry as an element of the `environment_variables` list.
- **Breaking:** attributes the API ignored: `ssl_check` on `checkly_check`, `alert_settings.ssl_certificates` on every resource that had it, and `retry_strategy.only_on` on the monitors that had no conditions to set.
- **Breaking:** `should_fail` on `checkly_grpc_monitor`, `checkly_ssl_monitor` and `checkly_traceroute_monitor`, which ignored it, and on browser and multi-step `checkly_check`, where setting it fails the plan. Matching the Checkly CLI, only API checks, URL monitors and TCP monitors have it, so `checkly_browser_check` and `checkly_multistep_check` don't either. Browser and multi-step checks that had it set to `true` are set to `false` by their next update; until then the per-type resources warn about it.
- **Breaking:** the `CHECKLY_DEBUG_LOG` request dump. Use an HTTPS proxy through `HTTPS_PROXY` to inspect requests.

### Fixed
- Only real 404 responses remove a resource from state.
- A dashboard's `key` no longer shows as changing on every update.
- Removing `tags` or `group_order` from a `checkly_check`, or a snippet or a local script from an API check, clears the value in Checkly, instead of showing the same change in every plan.
- Removing `tags` from a monitor, or `group_order` from a monitor that has it, clears the value in Checkly, instead of showing the same change in every plan.
- Removing `custom_domain`, `logo`, `redirect_to` or `favicon` from a `checkly_status_page`, or `custom_domain`, `logo`, `link`, `description` or `favicon` from a `checkly_dashboard`, clears the value in Checkly, instead of showing the same change in every plan. This also erases a value set in the web app that is not in the configuration; see the upgrade guide.
- Removing `setup_snippet_id`, `teardown_snippet_id`, `local_setup_script` or `local_teardown_script` from a `checkly_check_group` clears the value in Checkly.
- `starts_at`, `ends_at` and `repeat_ends_at` of a maintenance window written in another form than the API's, such as `2099-06-01` or `2099-06-01T09:00:00+02:00`, no longer show a change on every plan. A value for the same instant, or for `repeat_ends_at` the same UTC date, keeps its configured form.
- `custom_domain` of `checkly_status_page` is no longer rejected for a domain that only ends in the letters of a documentation domain, such as `status.myexample.com`. Domains under `example.com`, `example.net` and `example.org` still are.
- `checkly_check_group` can be created without `locations` or `private_locations`, which the API used to reject. Its checks then keep their own locations.
- The first apply after importing a `checkly_client_certificate` no longer replaces the certificate. The API never returns `private_key` or `passphrase`, so that apply records the configured values in Terraform state only and sends nothing to Checkly. To upload a different key, replace the resource, for example with `terraform apply -replace`.
- Removing `request.timeout` from a `checkly_traceroute_monitor` resets it to `10`, instead of keeping the previous value.
- Updating a `checkly_url_monitor` keeps request headers set on the monitor in the web app or through the API, instead of clearing them.
- Removing `request.data` from a `checkly_tcp_monitor` or `checkly_tcp_check` clears the data in Checkly, instead of keeping it. This also erases data set in the web app that is not in the configuration; see the upgrade guide.

## [v1.6.6](https://github.com/checkly/terraform-provider-checkly/releases/tag/v1.6.6) - 2023-06-06
### Added
- Support new private dashboards [#ba48b60](https://github.com/checkly/terraform-provider-checkly/commit/ba48b60)

## [v1.6.3](https://github.com/checkly/terraform-provider-checkly/releases/tag/v1.6.3) - 2022-10-26

### Added
- Support locked environment variables in `Check` and `Group`[#186](https://github.com/checkly/terraform-provider-checkly/issues/186)
- Add new `Dashboard` fields [#199](https://github.com/checkly/terraform-provider-checkly/issues/199)

### Changed
- Remove guides from provider documentation [#196](https://github.com/checkly/terraform-provider-checkly/issues/196)
- Add alternatives auth methods in documentation [#191](https://github.com/checkly/terraform-provider-checkly/issues/191)
- Follow resource naming convention in examples [#190](https://github.com/checkly/terraform-provider-checkly/issues/190)


### Fixed
- Avoid using all escalation configuration when no required [#202](https://github.com/checkly/terraform-provider-checkly/issues/194)
- Stop saving deprecated `ssl_certificates` property [#193](https://github.com/checkly/terraform-provider-checkly/issues/193)



## [v1.6.2](https://github.com/checkly/terraform-provider-checkly/releases/tag/v1.6.2) - 2022-08-02
### Changed
- Support new check intervals [#ba8eed7](https://github.com/checkly/terraform-provider-checkly/commit/ba8eed7)
### Fixed
- Fix style issues [#dc55746](https://github.com/checkly/terraform-provider-checkly/commit/dc55746)

## [v1.6.1](https://github.com/checkly/terraform-provider-checkly/releases/tag/v1.6.1) - 2022-07-06
### Changed
- Make `locations` property from `Group` optional [#2901337](https://github.com/checkly/terraform-provider-checkly/commit/2901337)
### Fixed
- Fix `PrivateLocation` docs which were missing [#d99105f](https://github.com/checkly/terraform-provider-checkly/commit/d99105f)

## [v1.6.0](https://github.com/checkly/terraform-provider-checkly/releases/tag/v1.6.0) - 2022-06-21
### Added
- Support Private Locations resources [#164](https://github.com/checkly/terraform-provider-checkly/issues/164)
- Allow Checks/Groups use private locations [#159](https://github.com/checkly/terraform-provider-checkly/issues/159)

## [v1.5.0](https://github.com/checkly/terraform-provider-checkly/releases/tag/v1.5.0) - 2022-06-21

### Added
- Support for global Environment Variables [#41](https://github.com/checkly/terraform-provider-checkly/issues/41)
- Implemented tfdocs for automatic docs generation [#125](https://github.com/checkly/terraform-provider-checkly/issues/125)

### Changed
- Remove unneeded `required` from `PublicDashboard`[#121](https://github.com/checkly/terraform-provider-checkly/issues/121)
- Remove unneeded `required` from `MaintenanceWindow`[#140](https://github.com/checkly/terraform-provider-checkly/issues/140)
- Improve CI and local tooling [#130](https://github.com/checkly/terraform-provider-checkly/issues/130)
- Update TF SDK and refactor local build [#124](https://github.com/checkly/terraform-provider-checkly/pull/124)
- Flag `sslCertificates` as deprecated [#137](https://github.com/checkly/terraform-provider-checkly/pull/137)
- Migrate to new Terraform schema Importer [#138](https://github.com/checkly/terraform-provider-checkly/pull/138)
- Allow set empty `locations` in checks within groups [#98](https://github.com/checkly/terraform-provider-checkly/issues/98)
- Bump github.com/google/go-cmp to v0.5.8 [#149](https://github.com/checkly/terraform-provider-checkly/pull/149)
- Bump github/codeql-action to v2 [#148](https://github.com/checkly/terraform-provider-checkly/pull/148)
- Bump hashicorp/setup-terraform to v2 [#146](https://github.com/checkly/terraform-provider-checkly/pull/146)
- Bump goreleaser/goreleaser-action to v3 [#154](https://github.com/checkly/terraform-provider-checkly/pull/154)
- Bump actions/setup-go to v3 [#154](https://github.com/checkly/terraform-provider-checkly/pull/154)
- Bump github.com/gruntwork-io/terratest to v0.40.17 [#169](https://github.com/checkly/terraform-provider-checkly/pull/169)
- Bump actions/checkout to v3 [#132](https://github.com/checkly/terraform-provider-checkly/pull/132)
- Bump github.com/checkly/checkly-go-sdk to v1.5.7 [#134](https://github.com/checkly/terraform-provider-checkly/pull/134)
- Bump github.com/hashicorp/terraform-plugin-sdk/v2 to v2.12.0 [#135](https://github.com/checkly/terraform-provider-checkly/pull/135)

## [v1.4.3](https://github.com/checkly/terraform-provider-checkly/releases/tag/v1.4.3) - 2022-03-09
### Changed
- Use generic provider descriptions in docs [#e7bb925](https://github.com/checkly/terraform-provider-checkly/commit/e7bb925)

## [v1.4.2](https://github.com/checkly/terraform-provider-checkly/releases/tag/v1.4.2) - 2022-03-02
### Added
- Add `CHECKLY_API_SOURCE` env variable [#120](https://github.com/checkly/terraform-provider-checkly/issues/120)

## [v1.4.1](https://github.com/checkly/terraform-provider-checkly/releases/tag/v1.4.1) - 2022-02-08

### Changed
- Migrate project test cases to User API Keys [#b35d8a7](https://github.com/checkly/terraform-provider-checkly/commit/b35d8a7)

## [v1.4.0](https://github.com/checkly/terraform-provider-checkly/releases/tag/v1.4.0) - 2022-01-28
### Added
- Support for Command Line Triggers [#87](https://github.com/checkly/terraform-provider-checkly/issues/87)
- Add Public API `source` HTTP header [#110](https://github.com/checkly/terraform-provider-checkly/issues/110)
- Allow skip ssl validation to api checks [#112](https://github.com/checkly/terraform-provider-checkly/issues/112)

### Changed
- Made check/group `locations` property optional [#103](https://github.com/checkly/terraform-provider-checkly/issues/103)
- Rename default branch to `main` [#99](https://github.com/checkly/terraform-provider-checkly/issues/99)
- Improve User API Keys docs [#95](https://github.com/checkly/terraform-provider-checkly/issues/95)

## [v1.3.0](https://github.com/checkly/terraform-provider-checkly/releases/tag/v1.3.0) - 2021-11-10
### Added
- Support for Public Dashboards [#81](https://github.com/checkly/terraform-provider-checkly/issues/81)
- Support for Maintenance Windows [#83](https://github.com/checkly/terraform-provider-checkly/issues/83)
- Support for User API Keys [#88](https://github.com/checkly/terraform-provider-checkly/issues/88)

## [v1.2.1](https://github.com/checkly/terraform-provider-checkly/releases/tag/v1.2.1) - 2021-10-19
### Changed
- Fix optional and required values in webhooks alert channels [#82](https://github.com/checkly/terraform-provider-checkly/pull/82)

## [v1.2.0](https://github.com/checkly/terraform-provider-checkly/releases/tag/v1.2.0) - 2021-07-14
### Added
- Support for versioned runtimes  [#31](https://github.com/checkly/checkly-go-sdk/issues/31).
- Support for PagerDuty alert channels integration [#53](https://github.com/checkly/terraform-provider-checkly/issues/53).


## [v1.1.0](https://github.com/checkly/terraform-provider-checkly/releases/tag/v1.1.0) - 2021-05-28
### Added
- Support for API high frequency checks [#68](https://github.com/checkly/terraform-provider-checkly/issues/68).
- Add `setupSnippetId` and `teardownSnippetID` to `check_group` resource [#69](https://github.com/checkly/terraform-provider-checkly/issues/69).

## [v1.0.0](https://github.com/checkly/terraform-provider-checkly/releases/tag/v1.4.3) - 2021-04-09
### Added
- Apple Silicon support is now added. The Terraform provider now also has `darwin_arm64` binaries

### Changed
- [🚨 BREAKING CHANGE] The default behavior of assigning all alert channels to checks and check groups is now removed. You can add alerts to your checks and check groups using the `alert_channel_subscription`
- Support for go1.16
