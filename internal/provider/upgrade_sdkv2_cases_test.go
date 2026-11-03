package provider

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

// upgradeRunID makes names, slugs and URLs unique per test run, since many of
// them must be unique within the account.
var upgradeRunID = strconv.Itoa(acctest.RandIntRange(100000, 999999))

// upgradeFixturesDir is absolute because the SDK v2 release runs as a separate
// process whose working directory is Terraform's, not this package's.
var upgradeFixturesDir = func() string {
	dir, err := filepath.Abs("../../fixtures")
	if err != nil {
		panic(err)
	}
	return dir
}()

// newSDKv2UpgradeCase fills in the {{id}} and {{fixtures}} placeholders of
// both configurations.
func newSDKv2UpgradeCase(sdkv2Config, config string) sdkv2UpgradeCase {
	r := strings.NewReplacer("{{id}}", upgradeRunID, "{{fixtures}}", upgradeFixturesDir)
	return sdkv2UpgradeCase{
		sdkv2Config: r.Replace(sdkv2Config),
		config:      r.Replace(config),
	}
}

// upgradeEmailChannel is a helper alert channel for cases that subscribe to
// one. Its syntax is the same in both providers.
const upgradeEmailChannel = `
resource "checkly_alert_channel" "email" {
  email {
    address = "tf-upgrade-{{id}}@example.com"
  }
}
`

// upgradeStatusPageService is a helper service for trigger_incident blocks and
// status pages. Its syntax is the same in both providers.
const upgradeStatusPageService = `
resource "checkly_status_page_service" "test" {
  name = "tf-upgrade-{{id}}-service"
}
`

// upgradeCertificate is the client certificate fixture also used by
// TestAccClientCertificate.
const upgradeCertificate = `
resource "checkly_client_certificate" "test" {
  host = "*.acme.com"
  path = "/tf-upgrade-{{id}}/api"

  certificate = <<-EOT
    -----BEGIN CERTIFICATE-----
    MIICDzCCAbagAwIBAgIUMTZlfGA7WcD8e4/zt2MqxvEgQPYwCgYIKoZIzj0EAwIw
    VDELMAkGA1UEBhMCVVMxCzAJBgNVBAgMAkNBMREwDwYDVQQHDAhUb29udG93bjES
    MBAGA1UECgwJQWNtZSBJbmMuMREwDwYDVQQDDAhhY21lLmNvbTAeFw0yNTAzMDMw
    NTQ2NTJaFw00OTEwMjMwNTQ2NTJaMHgxCzAJBgNVBAYTAlVTMQswCQYDVQQIDAJD
    QTERMA8GA1UEBwwIVG9vbnRvd24xEjAQBgNVBAoMCUFjbWUgSW5jLjEXMBUGA1UE
    AwwOV2lsZSBFLiBDb3lvdGUxHDAaBgkqhkiG9w0BCQEWDXdpbGVAYWNtZS5jb20w
    WTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAATAjjDGsKFS1qgdNqziDZoD5hamTfdH
    0P+Ukk1RIue57QYVXhQSyNzcEz15kQnwYezEqfN+FtjtTwdk/CgnAELlo0IwQDAd
    BgNVHQ4EFgQU9C9CpZqM2WMrOs3vAYsc5GbjyzswHwYDVR0jBBgwFoAUnlOyzF/N
    K7YmKQegLdbdyIOCT/UwCgYIKoZIzj0EAwIDRwAwRAIgGgSnBymlH4MkZCVk5DYH
    PdnDo2Xf5uFi1Eyn2LTYP1MCIEtiGtsf0qYv6NzIPd5uTTZoB/8hPrAgM1QzWG4O
    3C/I
    -----END CERTIFICATE-----
  EOT

  private_key = <<-EOT
    -----BEGIN ENCRYPTED PRIVATE KEY-----
    MIH0MF8GCSqGSIb3DQEFDTBSMDEGCSqGSIb3DQEFDDAkBBA5yR3aqy8mZD2wQzp1
    FH2JAgIIADAMBggqhkiG9w0CCQUAMB0GCWCGSAFlAwQBKgQQA49YCnXvfJ2CsQsV
    9C5JJwSBkNkWunSlqyeVW6OFa/+OjlLArgTGvW5ul08qu/145O9PO4Nr2CXeK5N2
    uvHwkWGfD8IVke+sgZPUjLoHsJ4h4AnyxlNHpIxgOfm0CoXT7PTaFb//d5NC6XyB
    K7ZpBzIThGlbuS/b9wp4MPmSaJn5Fci+84VG7KYK5RxU0fcU0rGSBynrZw803wnO
    FjP7qaq5bw==
    -----END ENCRYPTED PRIVATE KEY-----
  EOT

  passphrase = "secret password"

  trusted_ca = <<-EOT
    -----BEGIN CERTIFICATE-----
    MIIB/jCCAaOgAwIBAgIUZzxdNpoDYXaNiIBsh0/s++I+ZOEwCgYIKoZIzj0EAwIw
    VDELMAkGA1UEBhMCVVMxCzAJBgNVBAgMAkNBMREwDwYDVQQHDAhUb29udG93bjES
    MBAGA1UECgwJQWNtZSBJbmMuMREwDwYDVQQDDAhhY21lLmNvbTAeFw0yNTAzMDMw
    NTQzMDZaFw0yNTA0MDIwNTQzMDZaMFQxCzAJBgNVBAYTAlVTMQswCQYDVQQIDAJD
    QTERMA8GA1UEBwwIVG9vbnRvd24xEjAQBgNVBAoMCUFjbWUgSW5jLjERMA8GA1UE
    AwwIYWNtZS5jb20wWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAARDH3KGK6Vsk1A4
    yGf9ItQIS3yuAOi0n0ihmPzIOOOEN0c758ETABeUdgH55bakdx6q5KYSxf4TuXsJ
    2nCihqVVo1MwUTAdBgNVHQ4EFgQUnlOyzF/NK7YmKQegLdbdyIOCT/UwHwYDVR0j
    BBgwFoAUnlOyzF/NK7YmKQegLdbdyIOCT/UwDwYDVR0TAQH/BAUwAwEB/zAKBggq
    hkjOPQQDAgNJADBGAiEA/cJ9jV8MQz4ypQsFvUatrnbxyHO0f+pJhf09pAk6Kj8C
    IQCkSbope5r0KlVdqBeFF8wCfE3plwpelve3jqVIz6MedQ==
    -----END CERTIFICATE-----
  EOT
}
`

// upgradeCodeBundle is the code bundle shared by the Playwright check suite
// cases. Its syntax is the same in both providers.
const upgradeCodeBundle = `
resource "checkly_playwright_code_bundle" "test" {
  prebuilt_archive {
    file = "{{fixtures}}/playwright-project-pnpm.tar.gz"
  }
}
`

// sdkv2UpgradeCases covers every resource type of the SDK v2 release, with
// extra cases where an absent nested object (stored by SDK v2 as an empty
// list) or a different nesting is common in practice.
var sdkv2UpgradeCases = map[string]sdkv2UpgradeCase{
	"check_api": newSDKv2UpgradeCase(upgradeEmailChannel+`
resource "checkly_check" "test" {
  name                      = "tf-upgrade-{{id}}-check-api"
  description               = "API check upgraded from SDK v2"
  type                      = "API"
  frequency                 = 60
  activated                 = true
  muted                     = true
  should_fail               = false
  run_parallel              = true
  degraded_response_time    = 15000
  max_response_time         = 30000
  locations                 = ["eu-central-1", "us-east-1"]
  tags                      = ["tf-upgrade", "api"]
  use_global_alert_settings = false
  local_setup_script        = "console.log('setup')"
  local_teardown_script     = "console.log('teardown')"

  request {
    method           = "GET"
    url              = "https://api.checklyhq.com/public-stats"
    follow_redirects = true
    skip_ssl         = false
    ip_family        = "IPv4"
    headers = {
      X-Custom = "foo"
    }
    query_parameters = {
      param = "bar"
    }
    basic_auth {
      username = "user"
      password = "pass"
    }
    assertion {
      source     = "STATUS_CODE"
      property   = ""
      comparison = "EQUALS"
      target     = "200"
    }
    assertion {
      source     = "JSON_BODY"
      property   = "$.apiCheckResults"
      comparison = "GREATER_THAN"
      target     = "100"
    }
  }

  alert_settings {
    escalation_type = "RUN_BASED"
    run_based_escalation {
      failed_run_threshold = 2
    }
    reminders {
      amount   = 1
      interval = 5
    }
    parallel_run_failure_threshold {
      enabled    = true
      percentage = 50
    }
  }

  retry_strategy {
    type                 = "FIXED"
    base_backoff_seconds = 60
    max_retries          = 2
    max_duration_seconds = 600
    same_region          = true
    only_on {
      network_error = true
    }
  }

  alert_channel_subscription {
    channel_id = checkly_alert_channel.email.id
    activated  = true
  }
}
`, upgradeEmailChannel+`
resource "checkly_check" "test" {
  name                      = "tf-upgrade-{{id}}-check-api"
  description               = "API check upgraded from SDK v2"
  type                      = "API"
  frequency                 = 60
  activated                 = true
  muted                     = true
  should_fail               = false
  run_parallel              = true
  degraded_response_time    = 15000
  max_response_time         = 30000
  locations                 = ["eu-central-1", "us-east-1"]
  tags                      = ["tf-upgrade", "api"]
  use_global_alert_settings = false
  local_setup_script        = "console.log('setup')"
  local_teardown_script     = "console.log('teardown')"

  request {
    method           = "GET"
    url              = "https://api.checklyhq.com/public-stats"
    follow_redirects = true
    skip_ssl         = false
    ip_family        = "IPv4"
    headers = {
      X-Custom = "foo"
    }
    query_parameters = {
      param = "bar"
    }
    basic_auth = {
      username = "user"
      password = "pass"
    }
    assertion {
      source     = "STATUS_CODE"
      property   = ""
      comparison = "EQUALS"
      target     = "200"
    }
    assertion {
      source     = "JSON_BODY"
      property   = "$.apiCheckResults"
      comparison = "GREATER_THAN"
      target     = "100"
    }
  }

  alert_settings = {
    escalation_type = "RUN_BASED"
    run_based_escalation = {
      failed_run_threshold = 2
    }
    reminders = {
      amount   = 1
      interval = 5
    }
    parallel_run_failure_threshold = {
      enabled    = true
      percentage = 50
    }
  }

  retry_strategy = {
    type                 = "FIXED"
    base_backoff_seconds = 60
    max_retries          = 2
    max_duration_seconds = 600
    same_region          = true
    only_on = {
      network_error = true
    }
  }

  alert_channel_subscription {
    channel_id = checkly_alert_channel.email.id
    activated  = true
  }
}
`),

	// No alert_settings, retry_strategy or basic_auth: SDK v2 stores the
	// absent blocks as empty lists, or as the API's values where computed.
	"check_api_minimal": newSDKv2UpgradeCase(`
resource "checkly_check" "test" {
  name                      = "tf-upgrade-{{id}}-check-api-minimal"
  type                      = "API"
  frequency                 = 60
  activated                 = true
  locations                 = ["eu-central-1"]
  use_global_alert_settings = true

  request {
    url = "https://api.checklyhq.com/public-stats"
  }
}
`, `
resource "checkly_check" "test" {
  name                      = "tf-upgrade-{{id}}-check-api-minimal"
  type                      = "API"
  frequency                 = 60
  activated                 = true
  locations                 = ["eu-central-1"]
  use_global_alert_settings = true

  request {
    url = "https://api.checklyhq.com/public-stats"
  }
}
`),

	// double_check is deprecated but still works, so it must upgrade
	// together with the retry strategy it implies.
	"check_double_check": newSDKv2UpgradeCase(`
resource "checkly_check" "test" {
  name                      = "tf-upgrade-{{id}}-check-double-check"
  type                      = "API"
  frequency                 = 60
  activated                 = false
  locations                 = ["eu-central-1"]
  use_global_alert_settings = true
  double_check              = true

  request {
    url = "https://api.checklyhq.com/public-stats"
  }
}
`, `
resource "checkly_check" "test" {
  name                      = "tf-upgrade-{{id}}-check-double-check"
  type                      = "API"
  frequency                 = 60
  activated                 = false
  locations                 = ["eu-central-1"]
  use_global_alert_settings = true
  double_check              = true

  request {
    url = "https://api.checklyhq.com/public-stats"
  }
}
`),

	"check_group_double_check": newSDKv2UpgradeCase(`
resource "checkly_check_group" "test" {
  name         = "tf-upgrade-{{id}}-group-double-check"
  activated    = true
  concurrency  = 1
  locations    = ["eu-central-1"]
  double_check = true
}
`, `
resource "checkly_check_group" "test" {
  name         = "tf-upgrade-{{id}}-group-double-check"
  activated    = true
  concurrency  = 1
  locations    = ["eu-central-1"]
  double_check = true
}
`),

	"check_browser": newSDKv2UpgradeCase(`
resource "checkly_check" "test" {
  name                      = "tf-upgrade-{{id}}-check-browser"
  description               = "Browser check upgraded from SDK v2"
  type                      = "BROWSER"
  frequency                 = 720
  activated                 = true
  should_fail               = false
  locations                 = ["eu-central-1", "us-east-1"]
  tags                      = ["tf-upgrade", "browser"]
  use_global_alert_settings = false
  script                    = "console.log('test')"

  environment_variable {
    key   = "BASE_URL"
    value = "https://www.checklyhq.com"
  }

  alert_settings {
    escalation_type = "TIME_BASED"
    time_based_escalation {
      minutes_failing_threshold = 5
    }
    reminders {
      amount   = 2
      interval = 10
    }
  }

  retry_strategy {
    type                 = "LINEAR"
    base_backoff_seconds = 30
    max_retries          = 3
    max_duration_seconds = 300
    same_region          = false
  }
}
`, `
resource "checkly_check" "test" {
  name                      = "tf-upgrade-{{id}}-check-browser"
  description               = "Browser check upgraded from SDK v2"
  type                      = "BROWSER"
  frequency                 = 720
  activated                 = true
  should_fail               = false
  locations                 = ["eu-central-1", "us-east-1"]
  tags                      = ["tf-upgrade", "browser"]
  use_global_alert_settings = false
  script                    = "console.log('test')"

  environment_variable {
    key   = "BASE_URL"
    value = "https://www.checklyhq.com"
  }

  alert_settings = {
    escalation_type = "TIME_BASED"
    time_based_escalation = {
      minutes_failing_threshold = 5
    }
    reminders = {
      amount   = 2
      interval = 10
    }
  }

  retry_strategy = {
    type                 = "LINEAR"
    base_backoff_seconds = 30
    max_retries          = 3
    max_duration_seconds = 300
    same_region          = false
  }
}
`),

	"heartbeat": newSDKv2UpgradeCase(`
resource "checkly_heartbeat" "test" {
  name                      = "tf-upgrade-{{id}}-heartbeat"
  description               = "Heartbeat upgraded from SDK v2"
  activated                 = true
  muted                     = false
  tags                      = ["tf-upgrade"]
  use_global_alert_settings = false

  heartbeat {
    period      = 5
    period_unit = "days"
    grace       = 1
    grace_unit  = "hours"
  }

  alert_settings {
    escalation_type = "RUN_BASED"
    run_based_escalation {
      failed_run_threshold = 1
    }
    reminders {
      amount   = 1
      interval = 5
    }
  }
}
`, `
resource "checkly_heartbeat" "test" {
  name                      = "tf-upgrade-{{id}}-heartbeat"
  description               = "Heartbeat upgraded from SDK v2"
  activated                 = true
  muted                     = false
  tags                      = ["tf-upgrade"]
  use_global_alert_settings = false

  heartbeat {
    period      = 5
    period_unit = "days"
    grace       = 1
    grace_unit  = "hours"
  }

  alert_settings = {
    escalation_type = "RUN_BASED"
    run_based_escalation = {
      failed_run_threshold = 1
    }
    reminders = {
      amount   = 1
      interval = 5
    }
  }
}
`),

	"heartbeat_monitor": newSDKv2UpgradeCase(upgradeStatusPageService+upgradeEmailChannel+`
resource "checkly_heartbeat_monitor" "test" {
  name                      = "tf-upgrade-{{id}}-heartbeat-monitor"
  description               = "Heartbeat monitor upgraded from SDK v2"
  activated                 = true
  muted                     = true
  tags                      = ["tf-upgrade"]
  use_global_alert_settings = false

  heartbeat {
    period      = 10
    period_unit = "minutes"
    grace       = 5
    grace_unit  = "minutes"
  }

  alert_settings {
    escalation_type = "RUN_BASED"
    run_based_escalation {
      failed_run_threshold = 1
    }
    reminders {
      amount   = 0
      interval = 5
    }
  }

  trigger_incident {
    service_id         = checkly_status_page_service.test.id
    severity           = "MEDIUM"
    name               = "tf-upgrade-{{id}} heartbeat incident"
    description        = "No ping within the grace period"
    notify_subscribers = true
  }

  alert_channel_subscription {
    channel_id = checkly_alert_channel.email.id
    activated  = true
  }
}
`, upgradeStatusPageService+upgradeEmailChannel+`
resource "checkly_heartbeat_monitor" "test" {
  name                      = "tf-upgrade-{{id}}-heartbeat-monitor"
  description               = "Heartbeat monitor upgraded from SDK v2"
  activated                 = true
  muted                     = true
  tags                      = ["tf-upgrade"]
  use_global_alert_settings = false

  heartbeat {
    period      = 10
    period_unit = "minutes"
    grace       = 5
    grace_unit  = "minutes"
  }

  alert_settings = {
    escalation_type = "RUN_BASED"
    run_based_escalation = {
      failed_run_threshold = 1
    }
    reminders = {
      amount   = 0
      interval = 5
    }
  }

  trigger_incident {
    service_id         = checkly_status_page_service.test.id
    severity           = "MEDIUM"
    name               = "tf-upgrade-{{id}} heartbeat incident"
    description        = "No ping within the grace period"
    notify_subscribers = true
  }

  alert_channel_subscription {
    channel_id = checkly_alert_channel.email.id
    activated  = true
  }
}
`),

	"tcp_check": newSDKv2UpgradeCase(`
resource "checkly_tcp_check" "test" {
  name                   = "tf-upgrade-{{id}}-tcp-check"
  description            = "TCP check upgraded from SDK v2"
  frequency              = 120
  activated              = true
  muted                  = true
  degraded_response_time = 4000
  max_response_time      = 5000
  locations              = ["eu-central-1", "us-east-1"]

  request {
    hostname = "api.checklyhq.com"
    port     = 80
    assertion {
      source     = "RESPONSE_TIME"
      property   = ""
      comparison = "LESS_THAN"
      target     = "2000"
    }
  }

  alert_settings {
    escalation_type = "RUN_BASED"
    run_based_escalation {
      failed_run_threshold = 1
    }
    reminders {
      amount   = 0
      interval = 5
    }
    parallel_run_failure_threshold {
      enabled    = false
      percentage = 10
    }
  }

  retry_strategy {
    type                 = "EXPONENTIAL"
    base_backoff_seconds = 10
    max_retries          = 3
    max_duration_seconds = 600
    same_region          = true
  }
}
`, `
resource "checkly_tcp_check" "test" {
  name                   = "tf-upgrade-{{id}}-tcp-check"
  description            = "TCP check upgraded from SDK v2"
  frequency              = 120
  activated              = true
  muted                  = true
  degraded_response_time = 4000
  max_response_time      = 5000
  locations              = ["eu-central-1", "us-east-1"]

  request {
    hostname = "api.checklyhq.com"
    port     = 80
    assertion {
      source     = "RESPONSE_TIME"
      property   = ""
      comparison = "LESS_THAN"
      target     = "2000"
    }
  }

  alert_settings = {
    escalation_type = "RUN_BASED"
    run_based_escalation = {
      failed_run_threshold = 1
    }
    reminders = {
      amount   = 0
      interval = 5
    }
    parallel_run_failure_threshold = {
      enabled    = false
      percentage = 10
    }
  }

  retry_strategy = {
    type                 = "EXPONENTIAL"
    base_backoff_seconds = 10
    max_retries          = 3
    max_duration_seconds = 600
    same_region          = true
  }
}
`),

	"tcp_monitor": newSDKv2UpgradeCase(`
resource "checkly_tcp_monitor" "test" {
  name                   = "tf-upgrade-{{id}}-tcp-monitor"
  description            = "TCP monitor upgraded from SDK v2"
  frequency              = 120
  activated              = true
  muted                  = true
  degraded_response_time = 4000
  max_response_time      = 5000
  locations              = ["eu-central-1", "us-east-1"]
  tags                   = ["tf-upgrade"]

  request {
    hostname  = "api.checklyhq.com"
    port      = 80
    ip_family = "IPv4"
    assertion {
      source     = "RESPONSE_TIME"
      property   = ""
      comparison = "LESS_THAN"
      target     = "2000"
    }
    assertion {
      source     = "RESPONSE_DATA"
      property   = ""
      comparison = "CONTAINS"
      target     = "Foo"
    }
  }

  alert_settings {
    escalation_type = "RUN_BASED"
    run_based_escalation {
      failed_run_threshold = 1
    }
    reminders {
      amount   = 0
      interval = 5
    }
    parallel_run_failure_threshold {
      enabled    = false
      percentage = 10
    }
  }

  retry_strategy {
    type                 = "FIXED"
    base_backoff_seconds = 60
    max_retries          = 1
    max_duration_seconds = 600
    same_region          = true
  }
}
`, `
resource "checkly_tcp_monitor" "test" {
  name                   = "tf-upgrade-{{id}}-tcp-monitor"
  description            = "TCP monitor upgraded from SDK v2"
  frequency              = 120
  activated              = true
  muted                  = true
  degraded_response_time = 4000
  max_response_time      = 5000
  locations              = ["eu-central-1", "us-east-1"]
  tags                   = ["tf-upgrade"]

  request {
    hostname  = "api.checklyhq.com"
    port      = 80
    ip_family = "IPv4"
    assertion {
      source     = "RESPONSE_TIME"
      property   = ""
      comparison = "LESS_THAN"
      target     = "2000"
    }
    assertion {
      source     = "RESPONSE_DATA"
      property   = ""
      comparison = "CONTAINS"
      target     = "Foo"
    }
  }

  alert_settings = {
    escalation_type = "RUN_BASED"
    run_based_escalation = {
      failed_run_threshold = 1
    }
    reminders = {
      amount   = 0
      interval = 5
    }
    parallel_run_failure_threshold = {
      enabled    = false
      percentage = 10
    }
  }

  retry_strategy = {
    type                 = "FIXED"
    base_backoff_seconds = 60
    max_retries          = 1
    max_duration_seconds = 600
    same_region          = true
  }
}
`),

	"check_group": newSDKv2UpgradeCase(upgradeEmailChannel+`
resource "checkly_check_group" "test" {
  name                      = "tf-upgrade-{{id}}-check-group"
  activated                 = true
  muted                     = false
  concurrency               = 3
  run_parallel              = true
  locations                 = ["eu-central-1", "us-east-1"]
  tags                      = ["tf-upgrade"]
  use_global_alert_settings = false
  local_setup_script        = "setup-test"
  local_teardown_script     = "teardown-test"

  api_check_defaults {
    url = "http://example.com/"
    headers = {
      X-Test = "foo"
    }
    query_parameters = {
      query = "foo"
    }
    assertion {
      source     = "STATUS_CODE"
      property   = ""
      comparison = "EQUALS"
      target     = "200"
    }
    basic_auth {
      username = "user"
      password = "pass"
    }
  }

  environment_variable {
    key    = "FOO"
    value  = "BAR"
    locked = false
  }

  alert_settings {
    escalation_type = "RUN_BASED"
    run_based_escalation {
      failed_run_threshold = 1
    }
    reminders {
      amount   = 2
      interval = 5
    }
    parallel_run_failure_threshold {
      enabled    = false
      percentage = 10
    }
  }

  retry_strategy {
    type                 = "FIXED"
    base_backoff_seconds = 60
    max_retries          = 2
    max_duration_seconds = 600
    same_region          = true
    only_on {
      network_error = true
    }
  }

  alert_channel_subscription {
    channel_id = checkly_alert_channel.email.id
    activated  = true
  }
}
`, upgradeEmailChannel+`
resource "checkly_check_group" "test" {
  name                      = "tf-upgrade-{{id}}-check-group"
  activated                 = true
  muted                     = false
  concurrency               = 3
  run_parallel              = true
  locations                 = ["eu-central-1", "us-east-1"]
  tags                      = ["tf-upgrade"]
  use_global_alert_settings = false
  local_setup_script        = "setup-test"
  local_teardown_script     = "teardown-test"

  api_check_defaults = {
    url = "http://example.com/"
    headers = {
      X-Test = "foo"
    }
    query_parameters = {
      query = "foo"
    }
    assertion = [
      {
        source     = "STATUS_CODE"
        property   = ""
        comparison = "EQUALS"
        target     = "200"
      },
    ]
    basic_auth = {
      username = "user"
      password = "pass"
    }
  }

  environment_variable {
    key    = "FOO"
    value  = "BAR"
    locked = false
  }

  alert_settings = {
    escalation_type = "RUN_BASED"
    run_based_escalation = {
      failed_run_threshold = 1
    }
    reminders = {
      amount   = 2
      interval = 5
    }
    parallel_run_failure_threshold = {
      enabled    = false
      percentage = 10
    }
  }

  retry_strategy = {
    type                 = "FIXED"
    base_backoff_seconds = 60
    max_retries          = 2
    max_duration_seconds = 600
    same_region          = true
    only_on = {
      network_error = true
    }
  }

  alert_channel_subscription {
    channel_id = checkly_alert_channel.email.id
    activated  = true
  }
}
`),

	// A folder: every enforce_* object is absent, and SDK v2 stored the
	// API's disabled enforcement for each.
	"check_group_v2_minimal": newSDKv2UpgradeCase(`
resource "checkly_check_group_v2" "test" {
  name = "tf-upgrade-{{id}}-group-v2-minimal"
}
`, `
resource "checkly_check_group_v2" "test" {
  name = "tf-upgrade-{{id}}-group-v2-minimal"
}
`),

	"check_group_v2": newSDKv2UpgradeCase(upgradeEmailChannel+`
resource "checkly_private_location" "test" {
  name      = "tf-upgrade-{{id}}-group-v2"
  slug_name = "tf-upgrade-g2-{{id}}"
  icon      = "bell-fill"
}

resource "checkly_check_group_v2" "test" {
  name        = "tf-upgrade-{{id}}-group-v2"
  activated   = true
  muted       = true
  concurrency = 5
  tags        = ["tf-upgrade"]

  api_check_defaults {
    url = "https://api.example.com/"
    headers = {
      X-Api-Key = "test-key"
    }
    query_parameters = {
      version = "v2"
    }
    assertion {
      source     = "STATUS_CODE"
      property   = ""
      comparison = "EQUALS"
      target     = "200"
    }
    basic_auth {
      username = "admin"
      password = "pass"
    }
  }

  default_runtime {
    runtime_id = "2023.09"
  }

  setup_script {
    inline_script = "console.log('setup')"
  }

  teardown_script {
    inline_script = "console.log('teardown')"
  }

  environment_variable {
    key    = "FOO"
    value  = "bar"
    locked = true
  }

  enforce_locations {
    enabled           = true
    locations         = ["us-east-1", "eu-west-1"]
    private_locations = [checkly_private_location.test.slug_name]
  }

  enforce_retry_strategy {
    enabled = true
    retry_strategy {
      type                 = "FIXED"
      base_backoff_seconds = 30
      max_retries          = 3
      max_duration_seconds = 300
      same_region          = false
      only_on {
        network_error = true
      }
    }
  }

  enforce_scheduling_strategy {
    enabled      = true
    run_parallel = true
  }

  enforce_alert_settings {
    enabled                   = true
    use_global_alert_settings = false
    alert_settings {
      escalation_type = "RUN_BASED"
      run_based_escalation {
        failed_run_threshold = 2
      }
      reminders {
        amount   = 3
        interval = 10
      }
      parallel_run_failure_threshold {
        enabled    = true
        percentage = 50
      }
    }
    alert_channel_subscription {
      channel_id = checkly_alert_channel.email.id
      activated  = true
    }
  }
}
`, upgradeEmailChannel+`
resource "checkly_private_location" "test" {
  name      = "tf-upgrade-{{id}}-group-v2"
  slug_name = "tf-upgrade-g2-{{id}}"
  icon      = "bell-fill"
}

resource "checkly_check_group_v2" "test" {
  name        = "tf-upgrade-{{id}}-group-v2"
  activated   = true
  muted       = true
  concurrency = 5
  tags        = ["tf-upgrade"]

  api_check_defaults = {
    url = "https://api.example.com/"
    headers = {
      X-Api-Key = "test-key"
    }
    query_parameters = {
      version = "v2"
    }
    assertion = [
      {
        source     = "STATUS_CODE"
        property   = ""
        comparison = "EQUALS"
        target     = "200"
      },
    ]
    basic_auth = {
      username = "admin"
      password = "pass"
    }
  }

  default_runtime {
    runtime_id = "2023.09"
  }

  setup_script {
    inline_script = "console.log('setup')"
  }

  teardown_script {
    inline_script = "console.log('teardown')"
  }

  environment_variable {
    key    = "FOO"
    value  = "bar"
    locked = true
  }

  enforce_locations = {
    enabled           = true
    locations         = ["us-east-1", "eu-west-1"]
    private_locations = [checkly_private_location.test.slug_name]
  }

  enforce_retry_strategy = {
    enabled = true
    retry_strategy = {
      type                 = "FIXED"
      base_backoff_seconds = 30
      max_retries          = 3
      max_duration_seconds = 300
      same_region          = false
      only_on = {
        network_error = true
      }
    }
  }

  enforce_scheduling_strategy = {
    enabled      = true
    run_parallel = true
  }

  enforce_alert_settings = {
    enabled                   = true
    use_global_alert_settings = false
    alert_settings = {
      escalation_type = "RUN_BASED"
      run_based_escalation = {
        failed_run_threshold = 2
      }
      reminders = {
        amount   = 3
        interval = 10
      }
      parallel_run_failure_threshold = {
        enabled    = true
        percentage = 50
      }
    }
    alert_channel_subscription = [
      {
        channel_id = checkly_alert_channel.email.id
        activated  = true
      },
    ]
  }
}
`),

	"snippet": newSDKv2UpgradeCase(`
resource "checkly_snippet" "test" {
  name   = "tf-upgrade-{{id}}-snippet"
  script = "console.log('bar')"
}
`, `
resource "checkly_snippet" "test" {
  name   = "tf-upgrade-{{id}}-snippet"
  script = "console.log('bar')"
}
`),

	"dashboard": newSDKv2UpgradeCase(`
resource "checkly_dashboard" "test" {
  custom_url            = "tf-upgrade-{{id}}"
  custom_domain         = "tf-upgrade-{{id}}.example.com"
  logo                  = "https://example.com/logo.png"
  favicon               = "https://example.com/favicon.ico"
  link                  = "https://example.com"
  description           = "Dashboard upgraded from SDK v2"
  header                = "System Status"
  show_header           = false
  width                 = "960PX"
  refresh_rate          = 300
  paginate              = false
  checks_per_page       = 20
  pagination_rate       = 30
  hide_tags             = true
  use_tags_and_operator = true
  enable_incidents      = true
  expand_checks         = true
  show_check_run_links  = true
  custom_css            = ".header { color: blue; }"
  show_p95              = false
  show_p99              = false
  tags                  = ["tf-upgrade"]
}
`, `
resource "checkly_dashboard" "test" {
  custom_url            = "tf-upgrade-{{id}}"
  custom_domain         = "tf-upgrade-{{id}}.example.com"
  logo                  = "https://example.com/logo.png"
  favicon               = "https://example.com/favicon.ico"
  link                  = "https://example.com"
  description           = "Dashboard upgraded from SDK v2"
  header                = "System Status"
  show_header           = false
  width                 = "960PX"
  refresh_rate          = 300
  paginate              = false
  checks_per_page       = 20
  pagination_rate       = 30
  hide_tags             = true
  use_tags_and_operator = true
  enable_incidents      = true
  expand_checks         = true
  show_check_run_links  = true
  custom_css            = ".header { color: blue; }"
  show_p95              = false
  show_p99              = false
  tags                  = ["tf-upgrade"]
}
`),

	"maintenance_windows": newSDKv2UpgradeCase(`
resource "checkly_maintenance_windows" "test" {
  name      = "tf-upgrade-{{id}}-maintenance"
  starts_at = "2099-01-01T09:00:00.000Z"
  ends_at   = "2099-01-01T10:00:00.000Z"
  tags      = ["tf-upgrade"]
}
`, `
resource "checkly_maintenance_windows" "test" {
  name      = "tf-upgrade-{{id}}-maintenance"
  starts_at = "2099-01-01T09:00:00.000Z"
  ends_at   = "2099-01-01T10:00:00.000Z"
  tags      = ["tf-upgrade"]
}
`),

	// Like TestAccMaintenanceWindowControlsAllFields, this needs the status
	// page maintenance windows entitlement.
	"maintenance_windows_status_page": newSDKv2UpgradeCase(upgradeStatusPageService+`
resource "checkly_status_page" "test" {
  name = "tf-upgrade-{{id}}-maintenance"
  url  = "tf-upgrade-mw-{{id}}"

  card {
    name = "Card"
    service_attachment {
      service_id = checkly_status_page_service.test.id
    }
  }
}

resource "checkly_maintenance_windows" "test" {
  name                = "tf-upgrade-{{id}}-maintenance"
  starts_at           = "2099-01-01T09:00:00.000Z"
  ends_at             = "2099-01-01T10:00:00.000Z"
  repeat_unit         = "WEEK"
  repeat_interval     = 1
  repeat_ends_at      = "2099-06-01T00:00:00.000Z"
  tags                = ["tf-upgrade"]
  timezone            = "Europe/Berlin"
  pause_all_checks    = true
  silence_alerts_tags = ["tf-upgrade-silenced"]
  silence_all_alerts  = true
  description         = "Planned upgrade"

  status_page_visibility {
    enabled                 = true
    severity                = "MAJOR"
    notify_on_start         = true
    reminder_minutes_before = [60]
    auto_end                = false
    status_page_ids         = [checkly_status_page.test.id]
    service_ids             = [checkly_status_page_service.test.id]
  }
}
`, upgradeStatusPageService+`
resource "checkly_status_page" "test" {
  name = "tf-upgrade-{{id}}-maintenance"
  url  = "tf-upgrade-mw-{{id}}"

  card {
    name = "Card"
    service_attachment {
      service_id = checkly_status_page_service.test.id
    }
  }
}

resource "checkly_maintenance_windows" "test" {
  name                = "tf-upgrade-{{id}}-maintenance"
  starts_at           = "2099-01-01T09:00:00.000Z"
  ends_at             = "2099-01-01T10:00:00.000Z"
  repeat_unit         = "WEEK"
  repeat_interval     = 1
  repeat_ends_at      = "2099-06-01T00:00:00.000Z"
  tags                = ["tf-upgrade"]
  timezone            = "Europe/Berlin"
  pause_all_checks    = true
  silence_alerts_tags = ["tf-upgrade-silenced"]
  silence_all_alerts  = true
  description         = "Planned upgrade"

  status_page_visibility {
    enabled                 = true
    severity                = "MAJOR"
    notify_on_start         = true
    reminder_minutes_before = [60]
    auto_end                = false
    status_page_ids         = [checkly_status_page.test.id]
    service_ids             = [checkly_status_page_service.test.id]
  }
}
`),

	"alert_channel_email": newSDKv2UpgradeCase(`
resource "checkly_alert_channel" "test" {
  send_recovery        = true
  send_failure         = true
  send_degraded        = true
  ssl_expiry           = true
  ssl_expiry_threshold = 14

  email {
    address = "tf-upgrade-{{id}}-channel@example.com"
  }
}
`, `
resource "checkly_alert_channel" "test" {
  send_recovery        = true
  send_failure         = true
  send_degraded        = true
  ssl_expiry           = true
  ssl_expiry_threshold = 14

  email {
    address = "tf-upgrade-{{id}}-channel@example.com"
  }
}
`),

	"environment_variable": newSDKv2UpgradeCase(`
resource "checkly_environment_variable" "plain" {
  key    = "TF_UPGRADE_{{id}}_PLAIN"
  value  = "https://api.checklyhq.com"
  locked = true
}

resource "checkly_environment_variable" "secret" {
  key    = "TF_UPGRADE_{{id}}_SECRET"
  value  = "s3cr3t"
  secret = true
}
`, `
resource "checkly_environment_variable" "plain" {
  key    = "TF_UPGRADE_{{id}}_PLAIN"
  value  = "https://api.checklyhq.com"
  locked = true
}

resource "checkly_environment_variable" "secret" {
  key    = "TF_UPGRADE_{{id}}_SECRET"
  value  = "s3cr3t"
  secret = true
}
`),

	"private_location": newSDKv2UpgradeCase(`
resource "checkly_private_location" "test" {
  name      = "tf-upgrade-{{id}}-private-location"
  slug_name = "tf-upgrade-pl-{{id}}"
  icon      = "bell-fill"
}
`, `
resource "checkly_private_location" "test" {
  name      = "tf-upgrade-{{id}}-private-location"
  slug_name = "tf-upgrade-pl-{{id}}"
  icon      = "bell-fill"
}
`),

	// The path has no trailing slash: SDK v2 stored it trimmed, which would
	// differ from a configured trailing slash.
	"client_certificate": newSDKv2UpgradeCase(upgradeCertificate, upgradeCertificate),

	// v1.32.0 stored the path without its trailing slash. The first plan
	// updates the stored spelling in place instead of replacing the
	// certificate.
	"client_certificate_trailing_slash": func() sdkv2UpgradeCase {
		config := strings.Replace(upgradeCertificate, `path = "/tf-upgrade-{{id}}/api"`, `path = "/tf-upgrade-{{id}}/api/"`, 1)
		if config == upgradeCertificate {
			panic("upgradeCertificate no longer sets the expected path")
		}
		c := newSDKv2UpgradeCase(config, config)
		c.expectUpdate = "checkly_client_certificate.test"
		return c
	}(),

	"status_page": newSDKv2UpgradeCase(upgradeStatusPageService+`
resource "checkly_status_page" "test" {
  name          = "tf-upgrade-{{id}}-status-page"
  url           = "tf-upgrade-sp-{{id}}"
  logo          = "https://example.com/logo.png"
  favicon       = "https://example.com/favicon.ico"
  redirect_to   = "https://example.com"
  default_theme = "DARK"

  card {
    name = "Card"
    service_attachment {
      service_id = checkly_status_page_service.test.id
    }
  }
}
`, upgradeStatusPageService+`
resource "checkly_status_page" "test" {
  name          = "tf-upgrade-{{id}}-status-page"
  url           = "tf-upgrade-sp-{{id}}"
  logo          = "https://example.com/logo.png"
  favicon       = "https://example.com/favicon.ico"
  redirect_to   = "https://example.com"
  default_theme = "DARK"

  card {
    name = "Card"
    service_attachment {
      service_id = checkly_status_page_service.test.id
    }
  }
}
`),

	"status_page_service": newSDKv2UpgradeCase(upgradeStatusPageService, upgradeStatusPageService),

	"status_page_v3": newSDKv2UpgradeCase(`
resource "checkly_status_page_v3" "test" {
  name                  = "tf-upgrade-{{id}}-status-page-v3"
  url                   = "tf-upgrade-sp3-{{id}}"
  description           = "All systems"
  logo                  = "https://example.org/logo.png"
  logo_dark             = "https://example.org/logo-dark.png"
  redirect_to           = "https://example.org"
  favicon               = "https://example.org/favicon.png"
  default_theme         = "DARK"
  privacy_policy_link   = "https://example.org/privacy"
  terms_of_service_link = "https://example.org/terms"
  support_link          = "mailto:support@example.org"
  footer_text           = "Example Inc."
  google_analytics_tag  = "G-XXXXXXXXXX"
  allow_indexing        = false
}
`, `
resource "checkly_status_page_v3" "test" {
  name                  = "tf-upgrade-{{id}}-status-page-v3"
  url                   = "tf-upgrade-sp3-{{id}}"
  description           = "All systems"
  logo                  = "https://example.org/logo.png"
  logo_dark             = "https://example.org/logo-dark.png"
  redirect_to           = "https://example.org"
  favicon               = "https://example.org/favicon.png"
  default_theme         = "DARK"
  privacy_policy_link   = "https://example.org/privacy"
  terms_of_service_link = "https://example.org/terms"
  support_link          = "mailto:support@example.org"
  footer_text           = "Example Inc."
  google_analytics_tag  = "G-XXXXXXXXXX"
  allow_indexing        = false
}
`),

	"status_page_v3_component": newSDKv2UpgradeCase(`
resource "checkly_status_page_v3" "test" {
  name = "tf-upgrade-{{id}}-components"
  url  = "tf-upgrade-sp3c-{{id}}"
}

resource "checkly_status_page_v3_component" "group" {
  status_page_id      = checkly_status_page_v3.test.id
  type                = "GROUP"
  name                = "Group"
  display_order       = 0
  expanded_by_default = true
}

resource "checkly_status_page_v3_component" "api" {
  status_page_id       = checkly_status_page_v3.test.id
  name                 = "API"
  description          = "The public API"
  display_order        = 1
  parent_id            = checkly_status_page_v3_component.group.id
  show_historical_data = true
}
`, `
resource "checkly_status_page_v3" "test" {
  name = "tf-upgrade-{{id}}-components"
  url  = "tf-upgrade-sp3c-{{id}}"
}

resource "checkly_status_page_v3_component" "group" {
  status_page_id      = checkly_status_page_v3.test.id
  type                = "GROUP"
  name                = "Group"
  display_order       = 0
  expanded_by_default = true
}

resource "checkly_status_page_v3_component" "api" {
  status_page_id       = checkly_status_page_v3.test.id
  name                 = "API"
  description          = "The public API"
  display_order        = 1
  parent_id            = checkly_status_page_v3_component.group.id
  show_historical_data = true
}
`),

	"status_page_v3_automation_rule": newSDKv2UpgradeCase(`
resource "checkly_status_page_v3" "test" {
  name = "tf-upgrade-{{id}}-rules"
  url  = "tf-upgrade-sp3r-{{id}}"
}

resource "checkly_status_page_v3_component" "api" {
  status_page_id = checkly_status_page_v3.test.id
  name           = "API"
  display_order  = 0
}

resource "checkly_status_page_v3_automation_rule" "test" {
  status_page_id           = checkly_status_page_v3.test.id
  name                     = "tf-upgrade-{{id}} API outage"
  enabled                  = false
  first_update             = "We are investigating an issue."
  last_update              = "The issue has been resolved."
  notify_subscribers       = false
  cool_down_window_minutes = 30
  tags                     = ["tf-upgrade-api"]

  component {
    component_id  = checkly_status_page_v3_component.api.id
    target_impact = "MAJOR_OUTAGE"
  }
}
`, `
resource "checkly_status_page_v3" "test" {
  name = "tf-upgrade-{{id}}-rules"
  url  = "tf-upgrade-sp3r-{{id}}"
}

resource "checkly_status_page_v3_component" "api" {
  status_page_id = checkly_status_page_v3.test.id
  name           = "API"
  display_order  = 0
}

resource "checkly_status_page_v3_automation_rule" "test" {
  status_page_id           = checkly_status_page_v3.test.id
  name                     = "tf-upgrade-{{id}} API outage"
  enabled                  = false
  first_update             = "We are investigating an issue."
  last_update              = "The issue has been resolved."
  notify_subscribers       = false
  cool_down_window_minutes = 30
  tags                     = ["tf-upgrade-api"]

  component {
    component_id  = checkly_status_page_v3_component.api.id
    target_impact = "MAJOR_OUTAGE"
  }
}
`),

	"url_monitor": newSDKv2UpgradeCase(upgradeStatusPageService+upgradeEmailChannel+`
resource "checkly_url_monitor" "test" {
  name                      = "tf-upgrade-{{id}}-url-monitor"
  description               = "URL monitor upgraded from SDK v2"
  frequency                 = 120
  activated                 = true
  muted                     = true
  degraded_response_time    = 4000
  max_response_time         = 5000
  locations                 = ["eu-central-1", "us-east-1"]
  tags                      = ["tf-upgrade"]
  use_global_alert_settings = false

  request {
    url              = "https://api.checklyhq.com"
    follow_redirects = true
    skip_ssl         = false
    ip_family        = "IPv6"
    assertion {
      source     = "STATUS_CODE"
      property   = ""
      comparison = "GREATER_THAN"
      target     = "199"
    }
    assertion {
      source     = "STATUS_CODE"
      property   = ""
      comparison = "LESS_THAN"
      target     = "300"
    }
  }

  alert_settings {
    escalation_type = "RUN_BASED"
    run_based_escalation {
      failed_run_threshold = 1
    }
    reminders {
      amount   = 0
      interval = 5
    }
    parallel_run_failure_threshold {
      enabled    = false
      percentage = 10
    }
  }

  retry_strategy {
    type                 = "FIXED"
    base_backoff_seconds = 60
    max_retries          = 2
    max_duration_seconds = 600
    same_region          = true
    only_on {
      network_error = true
    }
  }

  trigger_incident {
    service_id         = checkly_status_page_service.test.id
    severity           = "MAJOR"
    name               = "tf-upgrade-{{id}} URL incident"
    description        = "The URL monitor is failing"
    notify_subscribers = false
  }

  alert_channel_subscription {
    channel_id = checkly_alert_channel.email.id
    activated  = true
  }
}
`, upgradeStatusPageService+upgradeEmailChannel+`
resource "checkly_url_monitor" "test" {
  name                      = "tf-upgrade-{{id}}-url-monitor"
  description               = "URL monitor upgraded from SDK v2"
  frequency                 = 120
  activated                 = true
  muted                     = true
  degraded_response_time    = 4000
  max_response_time         = 5000
  locations                 = ["eu-central-1", "us-east-1"]
  tags                      = ["tf-upgrade"]
  use_global_alert_settings = false

  request {
    url              = "https://api.checklyhq.com"
    follow_redirects = true
    skip_ssl         = false
    ip_family        = "IPv6"
    assertion {
      source     = "STATUS_CODE"
      property   = ""
      comparison = "GREATER_THAN"
      target     = "199"
    }
    assertion {
      source     = "STATUS_CODE"
      property   = ""
      comparison = "LESS_THAN"
      target     = "300"
    }
  }

  alert_settings = {
    escalation_type = "RUN_BASED"
    run_based_escalation = {
      failed_run_threshold = 1
    }
    reminders = {
      amount   = 0
      interval = 5
    }
    parallel_run_failure_threshold = {
      enabled    = false
      percentage = 10
    }
  }

  retry_strategy = {
    type                 = "FIXED"
    base_backoff_seconds = 60
    max_retries          = 2
    max_duration_seconds = 600
    same_region          = true
    only_on = {
      network_error = true
    }
  }

  trigger_incident {
    service_id         = checkly_status_page_service.test.id
    severity           = "MAJOR"
    name               = "tf-upgrade-{{id}} URL incident"
    description        = "The URL monitor is failing"
    notify_subscribers = false
  }

  alert_channel_subscription {
    channel_id = checkly_alert_channel.email.id
    activated  = true
  }
}
`),

	// No alert_settings or retry_strategy: SDK v2 stored the API's values.
	"url_monitor_minimal": newSDKv2UpgradeCase(`
resource "checkly_url_monitor" "test" {
  name      = "tf-upgrade-{{id}}-url-monitor-minimal"
  frequency = 60
  activated = true
  locations = ["eu-central-1"]

  request {
    url = "https://api.checklyhq.com"
  }
}
`, `
resource "checkly_url_monitor" "test" {
  name      = "tf-upgrade-{{id}}-url-monitor-minimal"
  frequency = 60
  activated = true
  locations = ["eu-central-1"]

  request {
    url = "https://api.checklyhq.com"
  }
}
`),

	"dns_monitor": newSDKv2UpgradeCase(`
resource "checkly_dns_monitor" "test" {
  name                   = "tf-upgrade-{{id}}-dns-monitor"
  description            = "DNS monitor upgraded from SDK v2"
  frequency              = 120
  activated              = true
  muted                  = true
  degraded_response_time = 400
  max_response_time      = 500
  locations              = ["eu-central-1", "us-east-1"]
  tags                   = ["tf-upgrade"]

  request {
    record_type = "AAAA"
    query       = "api.checklyhq.com"
    protocol    = "TCP"

    name_server {
      host = "1.1.1.1"
      port = 53
    }

    assertion {
      source     = "RESPONSE_CODE"
      property   = ""
      comparison = "EQUALS"
      target     = "NOERROR"
    }
  }

  alert_settings {
    escalation_type = "RUN_BASED"
    run_based_escalation {
      failed_run_threshold = 1
    }
    reminders {
      amount   = 1
      interval = 5
    }
    parallel_run_failure_threshold {
      enabled    = false
      percentage = 10
    }
  }

  retry_strategy {
    type                 = "LINEAR"
    base_backoff_seconds = 30
    max_retries          = 2
    max_duration_seconds = 600
    same_region          = false
  }
}
`, `
resource "checkly_dns_monitor" "test" {
  name                   = "tf-upgrade-{{id}}-dns-monitor"
  description            = "DNS monitor upgraded from SDK v2"
  frequency              = 120
  activated              = true
  muted                  = true
  degraded_response_time = 400
  max_response_time      = 500
  locations              = ["eu-central-1", "us-east-1"]
  tags                   = ["tf-upgrade"]

  request {
    record_type = "AAAA"
    query       = "api.checklyhq.com"
    protocol    = "TCP"

    name_server {
      host = "1.1.1.1"
      port = 53
    }

    assertion {
      source     = "RESPONSE_CODE"
      property   = ""
      comparison = "EQUALS"
      target     = "NOERROR"
    }
  }

  alert_settings = {
    escalation_type = "RUN_BASED"
    run_based_escalation = {
      failed_run_threshold = 1
    }
    reminders = {
      amount   = 1
      interval = 5
    }
    parallel_run_failure_threshold = {
      enabled    = false
      percentage = 10
    }
  }

  retry_strategy = {
    type                 = "LINEAR"
    base_backoff_seconds = 30
    max_retries          = 2
    max_duration_seconds = 600
    same_region          = false
  }
}
`),

	"icmp_monitor": newSDKv2UpgradeCase(`
resource "checkly_icmp_monitor" "test" {
  name                           = "tf-upgrade-{{id}}-icmp-monitor"
  description                    = "ICMP monitor upgraded from SDK v2"
  frequency                      = 120
  activated                      = true
  muted                          = true
  degraded_packet_loss_threshold = 10
  max_packet_loss_threshold      = 60
  locations                      = ["eu-central-1", "us-east-1"]
  tags                           = ["tf-upgrade"]

  request {
    hostname   = "api.checklyhq.com"
    ip_family  = "IPv4"
    ping_count = 10

    assertion {
      source     = "LATENCY"
      property   = "avg"
      comparison = "LESS_THAN"
      target     = "200"
    }
  }

  alert_settings {
    escalation_type = "RUN_BASED"
    run_based_escalation {
      failed_run_threshold = 1
    }
    reminders {
      amount   = 1
      interval = 5
    }
  }

  retry_strategy {
    type                 = "SINGLE_RETRY"
    base_backoff_seconds = 30
    same_region          = true
  }
}
`, `
resource "checkly_icmp_monitor" "test" {
  name                           = "tf-upgrade-{{id}}-icmp-monitor"
  description                    = "ICMP monitor upgraded from SDK v2"
  frequency                      = 120
  activated                      = true
  muted                          = true
  degraded_packet_loss_threshold = 10
  max_packet_loss_threshold      = 60
  locations                      = ["eu-central-1", "us-east-1"]
  tags                           = ["tf-upgrade"]

  request {
    hostname   = "api.checklyhq.com"
    ip_family  = "IPv4"
    ping_count = 10

    assertion {
      source     = "LATENCY"
      property   = "avg"
      comparison = "LESS_THAN"
      target     = "200"
    }
  }

  alert_settings = {
    escalation_type = "RUN_BASED"
    run_based_escalation = {
      failed_run_threshold = 1
    }
    reminders = {
      amount   = 1
      interval = 5
    }
  }

  retry_strategy = {
    type                 = "SINGLE_RETRY"
    base_backoff_seconds = 30
    same_region          = true
  }
}
`),

	"grpc_monitor": newSDKv2UpgradeCase(`
resource "checkly_grpc_monitor" "test" {
  name                   = "tf-upgrade-{{id}}-grpc-monitor"
  description            = "gRPC monitor upgraded from SDK v2"
  activated              = true
  frequency              = 5
  muted                  = true
  degraded_response_time = 3000
  max_response_time      = 8000
  locations              = ["eu-central-1", "us-east-1"]
  tags                   = ["tf-upgrade"]

  request {
    host               = "grpcb.in"
    port               = 9000
    grpc_mode          = "BEHAVIOR"
    tls                = true
    timeout            = 30
    service_definition = "REFLECTION"
    method             = "grpcbin.GRPCBin/DummyUnary"
    message            = jsonencode({ f_string = "hello" })

    metadata {
      key   = "x-api-key"
      value = "supersecret"
    }

    assertion {
      source     = "GRPC_STATUS_CODE"
      property   = ""
      comparison = "EQUALS"
      target     = "0"
    }
  }

  alert_settings {
    escalation_type = "RUN_BASED"
    run_based_escalation {
      failed_run_threshold = 1
    }
    reminders {
      amount   = 1
      interval = 5
    }
  }

  retry_strategy {
    type                 = "FIXED"
    base_backoff_seconds = 60
    max_retries          = 1
    max_duration_seconds = 600
    same_region          = true
  }
}
`, `
resource "checkly_grpc_monitor" "test" {
  name                   = "tf-upgrade-{{id}}-grpc-monitor"
  description            = "gRPC monitor upgraded from SDK v2"
  activated              = true
  frequency              = 5
  muted                  = true
  degraded_response_time = 3000
  max_response_time      = 8000
  locations              = ["eu-central-1", "us-east-1"]
  tags                   = ["tf-upgrade"]

  request {
    host               = "grpcb.in"
    port               = 9000
    grpc_mode          = "BEHAVIOR"
    tls                = true
    timeout            = 30
    service_definition = "REFLECTION"
    method             = "grpcbin.GRPCBin/DummyUnary"
    message            = jsonencode({ f_string = "hello" })

    metadata {
      key   = "x-api-key"
      value = "supersecret"
    }

    assertion {
      source     = "GRPC_STATUS_CODE"
      property   = ""
      comparison = "EQUALS"
      target     = "0"
    }
  }

  alert_settings = {
    escalation_type = "RUN_BASED"
    run_based_escalation = {
      failed_run_threshold = 1
    }
    reminders = {
      amount   = 1
      interval = 5
    }
  }

  retry_strategy = {
    type                 = "FIXED"
    base_backoff_seconds = 60
    max_retries          = 1
    max_duration_seconds = 600
    same_region          = true
  }
}
`),

	"traceroute_monitor": newSDKv2UpgradeCase(`
resource "checkly_traceroute_monitor" "test" {
  name                   = "tf-upgrade-{{id}}-traceroute-monitor"
  description            = "Traceroute monitor upgraded from SDK v2"
  activated              = true
  frequency              = 5
  muted                  = true
  degraded_response_time = 10000
  max_response_time      = 20000
  locations              = ["us-east-1", "ap-northeast-1"]
  tags                   = ["tf-upgrade"]

  request {
    url              = "api.checklyhq.com"
    protocol         = "TCP"
    port             = 443
    ip_family        = "IPv4"
    max_hops         = 30
    max_unknown_hops = 15
    ptr_lookup       = true
    timeout          = 10

    assertion {
      source     = "PACKET_LOSS"
      property   = ""
      comparison = "LESS_THAN"
      target     = "10"
    }
  }

  alert_settings {
    escalation_type = "RUN_BASED"
    run_based_escalation {
      failed_run_threshold = 1
    }
    reminders {
      amount   = 1
      interval = 5
    }
  }

  retry_strategy {
    type                 = "FIXED"
    base_backoff_seconds = 60
    max_retries          = 1
    max_duration_seconds = 600
    same_region          = true
  }
}
`, `
resource "checkly_traceroute_monitor" "test" {
  name                   = "tf-upgrade-{{id}}-traceroute-monitor"
  description            = "Traceroute monitor upgraded from SDK v2"
  activated              = true
  frequency              = 5
  muted                  = true
  degraded_response_time = 10000
  max_response_time      = 20000
  locations              = ["us-east-1", "ap-northeast-1"]
  tags                   = ["tf-upgrade"]

  request {
    url              = "api.checklyhq.com"
    protocol         = "TCP"
    port             = 443
    ip_family        = "IPv4"
    max_hops         = 30
    max_unknown_hops = 15
    ptr_lookup       = true
    timeout          = 10

    assertion {
      source     = "PACKET_LOSS"
      property   = ""
      comparison = "LESS_THAN"
      target     = "10"
    }
  }

  alert_settings = {
    escalation_type = "RUN_BASED"
    run_based_escalation = {
      failed_run_threshold = 1
    }
    reminders = {
      amount   = 1
      interval = 5
    }
  }

  retry_strategy = {
    type                 = "FIXED"
    base_backoff_seconds = 60
    max_retries          = 1
    max_duration_seconds = 600
    same_region          = true
  }
}
`),

	"ssl_monitor": newSDKv2UpgradeCase(`
resource "checkly_ssl_monitor" "test" {
  name                   = "tf-upgrade-{{id}}-ssl-monitor"
  description            = "SSL monitor upgraded from SDK v2"
  activated              = true
  frequency              = 60
  muted                  = true
  degraded_response_time = 3000
  max_response_time      = 10000
  locations              = ["us-east-1", "eu-central-1"]
  tags                   = ["tf-upgrade"]

  request {
    hostname                 = "api.checklyhq.com"
    port                     = 443
    server_name              = "api.checklyhq.com"
    ip_family                = "IPv4"
    skip_chain_validation    = false
    handshake_timeout_ms     = 10000
    alert_days_before_expiry = 20

    security_baseline {
      enabled = true
      min_tls_version {
        value    = "TLS1.2"
        severity = "fail"
      }
      min_key_size_bits {
        value    = 2048
        severity = "fail"
      }
      weak_signature_algorithm {
        severity = "fail"
      }
      known_bad_ca {
        severity = "fail"
      }
    }

    client_certificate {
      mode = "account_default"
    }

    assertion {
      source     = "CERTIFICATE"
      property   = "daysUntilExpiry"
      comparison = "GREATER_THAN"
      target     = "14"
    }
  }

  alert_settings {
    escalation_type = "RUN_BASED"
    run_based_escalation {
      failed_run_threshold = 1
    }
    reminders {
      amount   = 1
      interval = 5
    }
  }

  retry_strategy {
    type                 = "FIXED"
    base_backoff_seconds = 60
    max_retries          = 1
    max_duration_seconds = 600
    same_region          = true
  }
}
`, `
resource "checkly_ssl_monitor" "test" {
  name                   = "tf-upgrade-{{id}}-ssl-monitor"
  description            = "SSL monitor upgraded from SDK v2"
  activated              = true
  frequency              = 60
  muted                  = true
  degraded_response_time = 3000
  max_response_time      = 10000
  locations              = ["us-east-1", "eu-central-1"]
  tags                   = ["tf-upgrade"]

  request {
    hostname                 = "api.checklyhq.com"
    port                     = 443
    server_name              = "api.checklyhq.com"
    ip_family                = "IPv4"
    skip_chain_validation    = false
    handshake_timeout_ms     = 10000
    alert_days_before_expiry = 20

    security_baseline {
      enabled = true
      min_tls_version {
        value    = "TLS1.2"
        severity = "fail"
      }
      min_key_size_bits {
        value    = 2048
        severity = "fail"
      }
      weak_signature_algorithm {
        severity = "fail"
      }
      known_bad_ca {
        severity = "fail"
      }
    }

    client_certificate = {
      mode = "account_default"
    }

    assertion {
      source     = "CERTIFICATE"
      property   = "daysUntilExpiry"
      comparison = "GREATER_THAN"
      target     = "14"
    }
  }

  alert_settings = {
    escalation_type = "RUN_BASED"
    run_based_escalation = {
      failed_run_threshold = 1
    }
    reminders = {
      amount   = 1
      interval = 5
    }
  }

  retry_strategy = {
    type                 = "FIXED"
    base_backoff_seconds = 60
    max_retries          = 1
    max_duration_seconds = 600
    same_region          = true
  }
}
`),

	// No client_certificate or security_baseline: the API fills in its
	// defaults, which SDK v2 stored.
	"ssl_monitor_minimal": newSDKv2UpgradeCase(`
resource "checkly_ssl_monitor" "test" {
  name      = "tf-upgrade-{{id}}-ssl-monitor-minimal"
  activated = true
  frequency = 60
  locations = ["us-east-1"]

  request {
    hostname = "api.checklyhq.com"
  }
}
`, `
resource "checkly_ssl_monitor" "test" {
  name      = "tf-upgrade-{{id}}-ssl-monitor-minimal"
  activated = true
  frequency = 60
  locations = ["us-east-1"]

  request {
    hostname = "api.checklyhq.com"
  }
}
`),

	"playwright_check_suite": newSDKv2UpgradeCase(upgradeCodeBundle+`
resource "checkly_playwright_check_suite" "test" {
  name                      = "tf-upgrade-{{id}}-playwright"
  description               = "Playwright check suite upgraded from SDK v2"
  activated                 = true
  frequency                 = 720
  muted                     = true
  locations                 = ["us-east-1"]
  tags                      = ["tf-upgrade"]
  use_global_alert_settings = false

  bundle {
    id       = checkly_playwright_code_bundle.test.id
    metadata = checkly_playwright_code_bundle.test.metadata
  }

  runtime {
    steps {
      install {
        command = "pnpm i"
      }
      test {
        command = "pnpm playwright test"
      }
    }
    playwright {
      version = "1.56.1"
      device {
        type = "chromium"
      }
    }
  }

  environment_variable {
    key    = "FOO"
    value  = "bar"
    locked = false
  }

  alert_settings {
    escalation_type = "RUN_BASED"
    run_based_escalation {
      failed_run_threshold = 1
    }
    reminders {
      amount   = 1
      interval = 5
    }
    parallel_run_failure_threshold {
      enabled    = false
      percentage = 10
    }
  }
}
`, upgradeCodeBundle+`
resource "checkly_playwright_check_suite" "test" {
  name                      = "tf-upgrade-{{id}}-playwright"
  description               = "Playwright check suite upgraded from SDK v2"
  activated                 = true
  frequency                 = 720
  muted                     = true
  locations                 = ["us-east-1"]
  tags                      = ["tf-upgrade"]
  use_global_alert_settings = false

  bundle {
    id       = checkly_playwright_code_bundle.test.id
    metadata = checkly_playwright_code_bundle.test.metadata
  }

  runtime = {
    steps = {
      install = {
        command = "pnpm i"
      }
      test = {
        command = "pnpm playwright test"
      }
    }
    playwright = {
      version = "1.56.1"
      device = [
        {
          type = "chromium"
        },
      ]
    }
  }

  environment_variable {
    key    = "FOO"
    value  = "bar"
    locked = false
  }

  alert_settings = {
    escalation_type = "RUN_BASED"
    run_based_escalation = {
      failed_run_threshold = 1
    }
    reminders = {
      amount   = 1
      interval = 5
    }
    parallel_run_failure_threshold = {
      enabled    = false
      percentage = 10
    }
  }
}
`),

	// The runtime is detected from the code bundle except for the engine.
	"playwright_check_suite_engine": newSDKv2UpgradeCase(upgradeCodeBundle+`
resource "checkly_playwright_check_suite" "test" {
  name                      = "tf-upgrade-{{id}}-playwright-engine"
  activated                 = true
  frequency                 = 720
  use_global_alert_settings = true
  locations                 = ["us-east-1"]

  bundle {
    id       = checkly_playwright_code_bundle.test.id
    metadata = checkly_playwright_code_bundle.test.metadata
  }

  runtime {
    engine {
      name    = "node"
      version = "24"
    }
  }
}
`, upgradeCodeBundle+`
resource "checkly_playwright_check_suite" "test" {
  name                      = "tf-upgrade-{{id}}-playwright-engine"
  activated                 = true
  frequency                 = 720
  use_global_alert_settings = true
  locations                 = ["us-east-1"]

  bundle {
    id       = checkly_playwright_code_bundle.test.id
    metadata = checkly_playwright_code_bundle.test.metadata
  }

  runtime = {
    engine = {
      name    = "node"
      version = "24"
    }
  }
}
`),

	// A moved block in the same plan as the provider upgrade: the source
	// state is still at the SDK v2 schema version.
	"moved_check_to_api_check": newSDKv2UpgradeCase(`
resource "checkly_check" "test" {
  name                   = "tf-upgrade-{{id}}-moved-api"
  type                   = "API"
  frequency              = 60
  activated              = true
  degraded_response_time = 15000
  max_response_time      = 30000
  locations              = ["eu-central-1"]

  request {
    url = "https://api.checklyhq.com/public-stats"
    assertion {
      source     = "STATUS_CODE"
      comparison = "EQUALS"
      target     = "200"
    }
  }

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
`, `
resource "checkly_api_check" "test" {
  name                   = "tf-upgrade-{{id}}-moved-api"
  frequency              = 60
  activated              = true
  degraded_response_time = 15000
  max_response_time      = 30000
  locations              = ["eu-central-1"]

  request {
    url = "https://api.checklyhq.com/public-stats"
    assertion {
      source     = "STATUS_CODE"
      comparison = "EQUALS"
      target     = "200"
    }
  }

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

moved {
  from = checkly_check.test
  to   = checkly_api_check.test
}
`),

	"moved_heartbeat_to_heartbeat_monitor": newSDKv2UpgradeCase(`
resource "checkly_heartbeat" "test" {
  name      = "tf-upgrade-{{id}}-moved-heartbeat"
  activated = true

  heartbeat {
    period      = 5
    period_unit = "days"
    grace       = 1
    grace_unit  = "hours"
  }
}
`, `
resource "checkly_heartbeat_monitor" "test" {
  name      = "tf-upgrade-{{id}}-moved-heartbeat"
  activated = true

  heartbeat {
    period      = 5
    period_unit = "days"
    grace       = 1
    grace_unit  = "hours"
  }
}

moved {
  from = checkly_heartbeat.test
  to   = checkly_heartbeat_monitor.test
}
`),

	"moved_maintenance_windows_to_maintenance_window": newSDKv2UpgradeCase(`
resource "checkly_maintenance_windows" "test" {
  name      = "tf-upgrade-{{id}}-moved-maintenance"
  starts_at = "2099-01-01T09:00:00.000Z"
  ends_at   = "2099-01-01T10:00:00.000Z"
  tags      = ["tf-upgrade"]
}
`, `
resource "checkly_maintenance_window" "test" {
  name      = "tf-upgrade-{{id}}-moved-maintenance"
  starts_at = "2099-01-01T09:00:00.000Z"
  ends_at   = "2099-01-01T10:00:00.000Z"
  tags      = ["tf-upgrade"]
}

moved {
  from = checkly_maintenance_windows.test
  to   = checkly_maintenance_window.test
}
`),
}
