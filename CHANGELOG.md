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
- **Breaking:** the provider serves plugin protocol 6 only and requires Terraform 1.0 or later.
- **Breaking:** every API key needs an account ID, and it must be a UUID. Account-scoped keys used without `account_id` or `CHECKLY_ACCOUNT_ID` stop with a `Missing Checkly Account ID` error.
- **Breaking:** the first apply erases values that are set in the web app and absent from your configuration, for the attributes v1.x could not clear. See [Expected one-time changes](docs/guides/upgrading-to-v2.md#expected-one-time-changes) in the upgrade guide.
- **Breaking:** credentials are sensitive, including nested ones, so Terraform redacts them in plan output. An output that exposes one, such as a heartbeat ping URL built from `ping_token`, must set `sensitive = true`.
- An invalid `API_CALL_TIMEOUT` is reported as a configuration error instead of crashing the provider.
- The provider uses `checkly-go-sdk` v2, and `checkly_check` writes through the API endpoint of each check type. Attributes that don't apply to the type (`degraded_response_time`, `max_response_time`, `setup_snippet_id`, `teardown_snippet_id`, `local_setup_script` and `local_teardown_script` on browser and multi-step checks, `script` on API checks and `ssl_check_domain` on multi-step checks) no longer reach the API, so changing one only changes the state.
- `double_check` is not read back from the API. An import stores `false`, so importing a check or group with `double_check = true` in its configuration plans one in-place update.
- An empty `api_key`, `api_url` or `account_id` in the provider configuration, such as a variable that defaults to `""`, counts as not set, so the matching environment variable applies. An empty `api_key` used to stop the provider, and an empty `api_url` used to select the default URL over `CHECKLY_API_URL`.
- A browser or multi-step `checkly_check` with an empty `script` fails with a `Missing Check Script` error before any request.
- API error messages name the request, the status and the API's own message, for example `GET /v1/checks/…: 404 Not Found`. Scripts that matched the old `unexpected response status` text need updating.

### Added
- `checkly_api_check`, `checkly_browser_check` and `checkly_multistep_check`, which replace `checkly_check`.
- `checkly_maintenance_window`, which replaces `checkly_maintenance_windows`.
- `intent` and `ai_auto_repair_enabled` on `checkly_api_check`, `checkly_browser_check`, `checkly_multistep_check` and `checkly_playwright_check_suite`. A value set outside Terraform and absent from the configuration is removed by the next apply, and `ignore_changes` leaves it to the web app.
- `private_locations` on `checkly_dns_monitor`, `checkly_icmp_monitor` and `checkly_traceroute_monitor`, and `proxy_url` on `checkly_private_location`. A value set outside Terraform and absent from the configuration is removed by the next apply, and `ignore_changes` leaves it to the web app.
- Import instructions on every importable resource's documentation page.

### Deprecated
- `checkly_check`, in favor of the per-type check resources.
- `checkly_heartbeat`, `checkly_tcp_check` and `checkly_maintenance_windows`, in favor of `checkly_heartbeat_monitor`, `checkly_tcp_monitor` and `checkly_maintenance_window`. They keep working, and `moved` blocks (Terraform 1.8 or later) switch them without recreating them.
- `checkly_check_group`, in favor of `checkly_check_group_v2`.
- `checkly_status_page` and `checkly_status_page_service`, in favor of the v3 status page resources.
- `slack` on `checkly_alert_channel`, in favor of `slack_app`.

### Removed
- **Breaking:** `checkly_trigger_check` and `checkly_trigger_group`. Use the Checkly CLI (`checkly trigger`) or `POST /v1/check-sessions/trigger`.
- **Breaking:** `environment_variables` (the map) on `checkly_check` and `checkly_check_group`; use `environment_variable` blocks.
- **Breaking:** attributes the API ignored: `ssl_check` on `checkly_check`, `alert_settings.ssl_certificates` on every resource that had it, and `retry_strategy.only_on` on the monitors that had no conditions to set.
- **Breaking:** the `CHECKLY_DEBUG_LOG` request dump. Use an HTTPS proxy through `HTTPS_PROXY` to inspect requests.

### Fixed
- Only real 404 responses remove a resource from state.
- A dashboard's `key` no longer shows as changing on every update.
- Removing `tags` or `group_order` from a `checkly_check`, or a snippet or a local script from an API check, clears the value in Checkly, instead of showing the same change in every plan.
- Removing `tags` from a monitor, or `group_order` from a monitor that has it, clears the value in Checkly, instead of showing the same change in every plan.
- Removing `custom_domain`, `logo`, `redirect_to` or `favicon` from a `checkly_status_page`, or `custom_domain`, `logo`, `link`, `description` or `favicon` from a `checkly_dashboard`, clears the value in Checkly, instead of showing the same change in every plan. This also erases a value set in the web app that is not in the configuration; see the upgrade guide.
- Removing `setup_snippet_id`, `teardown_snippet_id`, `local_setup_script` or `local_teardown_script` from a `checkly_check_group` clears the value in Checkly.
- `checkly_check_group` can be created without `locations` or `private_locations`, which the API used to reject. Its checks then keep their own locations.

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
