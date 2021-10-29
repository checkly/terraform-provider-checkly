# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).


## [1.2.1] - 2021-10-19
### Changed
- Fix optional and required values in webhooks alert channels [#82](https://github.com/checkly/terraform-provider-checkly/pull/82)

## [1.2.0] - 2021-07-14
### Added
- Support for versioned runtimes  [#31](https://github.com/checkly/checkly-go-sdk/issues/31).

## [1.2.0-rc1] - 2021-06-02
### Added
- Support for PagerDuty alert channels integration [#53](https://github.com/checkly/terraform-provider-checkly/issues/53).


## [1.1.0] - 2021-05-28
### Added
- Support for API high frequency checks [#68](https://github.com/checkly/terraform-provider-checkly/issues/68).
- Add `setupSnippetId` and `teardownSnippetID` to `check_group` resource [#69](https://github.com/checkly/terraform-provider-checkly/issues/69).

## [1.0.0] - 2021-04-09
### Added
- Apple Silicon support is now added. The Terraform provider now also has `darwin_arm64` binaries

### Changed
- [🚨 BREAKING CHANGE] The default behavior of assigning all alert channels to checks and check groups is now removed. You can add alerts to your checks and check groups using the `alert_channel_subscription`
- Support for go1.16
