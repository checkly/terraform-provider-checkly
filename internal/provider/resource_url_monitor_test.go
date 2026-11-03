package provider

import (
	"context"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/checkly/checkly-go-sdk"
)

func TestAccURLMonitorRequiredFields(t *testing.T) {
	config := `resource "checkly_url_monitor" "test" {}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "name" is required, but no definition was found.`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "activated" is required, but no definition was found.`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "frequency" is required, but no definition was found.`),
		},
		{
			// Terraform only reports the missing request block once all
			// required attributes are present.
			Config: `
				resource "checkly_url_monitor" "test" {
					name      = "test-required-fields"
					activated = true
					frequency = 60
				}
			`,
			ExpectError: regexp.MustCompile(`Block request must have a configuration value`),
		},
	})
}

func TestAccURLMonitorBasic(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: urlMonitor_basic,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"name",
					"URL Monitor 1",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"description",
					"URL monitor description",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"activated",
					"true",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					"locations.*",
					"eu-central-1",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					"locations.*",
					"us-east-1",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					"request.*.url",
					"https://api.checklyhq.com",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					"request.*.assertion.*.comparison",
					"EQUALS",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					"request.*.assertion.*.property",
					"",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					"request.*.assertion.*.source",
					"STATUS_CODE",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					"request.*.assertion.*.target",
					"200",
				),
			),
		},
	})
}

func TestAccURLMonitorDescriptionRemoval(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: urlMonitor_basic,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"description",
					"URL monitor description",
				),
			),
		},
		{
			Config: urlMonitor_basic_withoutDescription,
			Check: resource.TestCheckResourceAttr(
				"checkly_url_monitor.test",
				"description",
				"",
			),
		},
	})
}

func TestAccURLMonitorFull(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: urlMonitor_full,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"degraded_response_time",
					"4000",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"max_response_time",
					"5000",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					`"locations.#"`,
					"3",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					"request.*.follow_redirects",
					"true",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					"request.*.skip_ssl",
					"false",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					"request.*.ip_family",
					"IPv6",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					"request.*.assertion.#",
					"2",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					"request.*.assertion.*.comparison",
					"LESS_THAN",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					"request.*.assertion.*.comparison",
					"GREATER_THAN",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					"request.*.assertion.*.target",
					"199",
				),
				testCheckResourceAttrExpr(
					"checkly_url_monitor.test",
					"request.*.assertion.*.target",
					"300",
				),
			),
		},
	})
}

func TestAccURLMonitorWithTriggerIncident(t *testing.T) {
	urlMonitorWithTriggerIncident := `
resource "checkly_status_page_service" "test_url_service" {
	name = "URL Test Service"
}

resource "checkly_url_monitor" "test_trigger_incident" {
	name          = "URL Monitor with Trigger Incident"
	activated     = true
	frequency     = 30
	locations     = ["eu-west-2", "us-east-1"]

	request {
		url = "https://status.example.com/health"
		follow_redirects = false
		assertion {
			source     = "STATUS_CODE"
			comparison = "EQUALS"
			target     = "200"
		}
	}

	trigger_incident {
		service_id         = checkly_status_page_service.test_url_service.id
		severity           = "CRITICAL"
		name               = "URL Monitor Service Down"
		description        = "The URL monitor has detected that the service is unreachable"
		notify_subscribers = true
	}

	use_global_alert_settings = true
}
`
	accTestCase(t, []resource.TestStep{
		{
			Config: urlMonitorWithTriggerIncident,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test_trigger_incident",
					"name",
					"URL Monitor with Trigger Incident",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test_trigger_incident",
					"activated",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test_trigger_incident",
					"frequency",
					"30",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test_trigger_incident",
					"trigger_incident.0.severity",
					"CRITICAL",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test_trigger_incident",
					"trigger_incident.0.name",
					"URL Monitor Service Down",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test_trigger_incident",
					"trigger_incident.0.description",
					"The URL monitor has detected that the service is unreachable",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test_trigger_incident",
					"trigger_incident.0.notify_subscribers",
					"true",
				),
			),
		},
	})
}

var wantURLMonitor = checkly.URLMonitor{
	Name:                 "My test check",
	Frequency:            1,
	Activated:            true,
	Muted:                false,
	RetryStrategy:        &checkly.RetryStrategy{Type: "NO_RETRIES"},
	ShouldFail:           false,
	Locations:            []string{"eu-west-1"},
	PrivateLocations:     &[]string{},
	DegradedResponseTime: 4000,
	MaxResponseTime:      5000,
	Tags: []string{
		"foo",
		"bar",
	},
	AlertSettings: &checkly.AlertSettings{
		EscalationType: checkly.RunBased,
		RunBasedEscalation: checkly.RunBasedEscalation{
			FailedRunThreshold: 1,
		},
		Reminders: checkly.Reminders{
			Interval: 5,
		},
	},
	UseGlobalAlertSettings: false,
	Request: checkly.URLRequest{
		URL:             "https://example.com",
		FollowRedirects: true,
		SkipSSL:         false,
		IPFamily:        "IPv6",
		Assertions: []checkly.Assertion{
			{
				Source:     checkly.StatusCode,
				Comparison: checkly.Equals,
				Target:     "200",
			},
		},
	},
}

func TestEncodeDecodeURLMonitorResource(t *testing.T) {
	ctx := context.Background()

	var diags diag.Diagnostics

	// Start from a model whose prior-aware values are typed nulls, as they
	// would be when decoded from an empty plan or state.
	model := URLMonitorResourceModel{
		RetryStrategy: types.ObjectNull(retryStrategyAttrTypes(true)),
		AlertSettings: types.ObjectNull(alertSettingsAttrTypes),
	}

	want := wantURLMonitor
	want.AlertChannelSubscriptions = []checkly.AlertChannelSubscription{}

	urlMonitorToModel(ctx, &want, &model, &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}

	got := urlMonitorFromModel(ctx, &model, &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}

	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

const urlMonitor_basic = `
resource "checkly_url_monitor" "test" {
  name                      = "URL Monitor 1"
  description               = "URL monitor description"
  frequency                 = 60
  activated                 = true
  muted                     = true
  max_response_time         = 3000
  locations                 = ["us-east-1", "eu-central-1"]
  use_global_alert_settings = true
  request {
    url = "https://api.checklyhq.com"
    assertion {
      comparison = "EQUALS"
      property   = ""
      source     = "STATUS_CODE"
      target     = "200"
    }
  }
}
`

const urlMonitor_basic_withoutDescription = `
resource "checkly_url_monitor" "test" {
  name                      = "URL Monitor 1"
  frequency                 = 60
  activated                 = true
  muted                     = true
  max_response_time         = 3000
  locations                 = ["us-east-1", "eu-central-1"]
  use_global_alert_settings = true
  request {
    url = "https://api.checklyhq.com"
    assertion {
      comparison = "EQUALS"
      property   = ""
      source     = "STATUS_CODE"
      target     = "200"
    }
  }
}
`

const urlMonitor_full = `
resource "checkly_url_monitor" "test" {
  name                   = "URL Monitor 2"
  frequency              = 120
  activated              = true
  muted                  = true
  degraded_response_time = 4000
  max_response_time      = 5000
  locations = [
    "eu-central-1",
    "us-east-1",
    "ap-northeast-1"
  ]
  request {
    url              = "https://api.checklyhq.com"
    follow_redirects = true
    skip_ssl         = false
    ip_family        = "IPv6"
    assertion {
      comparison = "GREATER_THAN"
      property   = ""
      source     = "STATUS_CODE"
      target     = "199"
    }
    assertion {
      comparison = "LESS_THAN"
      property   = ""
      source     = "STATUS_CODE"
      target     = "300"
    }
  }

  alert_settings = {
    escalation_type = "RUN_BASED"
    reminders = {
      amount   = 0
      interval = 5
    }
    run_based_escalation = {
      failed_run_threshold = 1
    }
    parallel_run_failure_threshold = {
      enabled    = false
      percentage = 10
    }
  }
}
`

func TestAccURLMonitorWithSingleRetry(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_url_monitor" "test" {
					name      = "test-single-retry"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
					retry_strategy = {
						type                 = "SINGLE_RETRY"
						base_backoff_seconds = 30
						same_region          = true
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.type",
					"SINGLE_RETRY",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.base_backoff_seconds",
					"30",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.max_retries",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.max_duration_seconds",
					"600",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.same_region",
					"true",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
	})

}

func TestAccURLMonitorWithSingleRetryOnlyOnNetworkError(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_url_monitor" "test" {
					name      = "test-single-retry"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
					retry_strategy = {
						type = "SINGLE_RETRY"

						only_on = {
							network_error = true
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.only_on.network_error",
					"true",
				),
			),
		},
		{
			Config: `
				resource "checkly_url_monitor" "test" {
					name      = "test-single-retry"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
					retry_strategy = {
						type = "SINGLE_RETRY"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
	})
}

func TestAccURLMonitorWithNoRetries(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_url_monitor" "test" {
					name      = "test-no-retries"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
					retry_strategy = {
						type = "NO_RETRIES"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.type",
					"NO_RETRIES",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.base_backoff_seconds",
					"60",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.max_retries",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.max_duration_seconds",
					"600",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.same_region",
					"true",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
	})
}

func TestAccURLMonitorWithNoRetriesOnlyOnNetworkError(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_url_monitor" "test" {
					name      = "test-no-retries"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
					retry_strategy = {
						type = "NO_RETRIES"

						only_on = {
							network_error = true
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.type",
					"NO_RETRIES",
				),
				// The NO_RETRIES strategy has no retry conditions, but the
				// configured value is preserved in state.
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.only_on.network_error",
					"true",
				),
			),
		},
	})
}

func TestAccURLMonitorWithDefaultNoRetries(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_url_monitor" "test" {
					name      = "test-no-retries"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.type",
					"NO_RETRIES",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.base_backoff_seconds",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.max_retries",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.max_duration_seconds",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.same_region",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
	})
}

func TestAccURLMonitorWithFixedRetry(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_url_monitor" "test" {
					name      = "test-fixed-retry"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
					retry_strategy = {
						type                 = "FIXED"
						base_backoff_seconds = 30
						max_retries          = 3
						max_duration_seconds = 300
						same_region          = false
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.type",
					"FIXED",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.base_backoff_seconds",
					"30",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.max_retries",
					"3",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.max_duration_seconds",
					"300",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.same_region",
					"false",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
	})
}

func TestAccURLMonitorWithFixedRetryOnlyOnNetworkError(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_url_monitor" "test" {
					name      = "test-fixed-retry"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
					retry_strategy = {
						type = "FIXED"

						only_on = {
							network_error = true
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.only_on.network_error",
					"true",
				),
			),
		},
		{
			Config: `
				resource "checkly_url_monitor" "test" {
					name      = "test-fixed-retry"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
					retry_strategy = {
						type = "FIXED"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
	})
}

func TestAccURLMonitorWithExponentialRetry(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_url_monitor" "test" {
					name      = "test-exponential-retry"
					activated = true
					frequency = 720
					locations = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
					retry_strategy = {
						type                 = "EXPONENTIAL"
						base_backoff_seconds = 60
						max_retries          = 5
						max_duration_seconds = 600
						same_region          = true
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.type",
					"EXPONENTIAL",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.base_backoff_seconds",
					"60",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.max_retries",
					"5",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.max_duration_seconds",
					"600",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.same_region",
					"true",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
	})
}

func TestAccURLMonitorWithExponentialRetryOnlyOnNetworkError(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_url_monitor" "test" {
					name      = "test-exponential-retry"
					activated = true
					frequency = 720
					locations = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
					retry_strategy = {
						type = "EXPONENTIAL"

						only_on = {
							network_error = true
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.only_on.network_error",
					"true",
				),
			),
		},
		{
			Config: `
				resource "checkly_url_monitor" "test" {
					name      = "test-exponential-retry"
					activated = true
					frequency = 720
					locations = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
					retry_strategy = {
						type = "EXPONENTIAL"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
	})
}

func TestAccURLMonitorWithLinearRetry(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_url_monitor" "test" {
					name      = "test-linear-retry"
					activated = true
					frequency = 720
					locations = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
					retry_strategy = {
						type                 = "LINEAR"
						base_backoff_seconds = 45
						max_retries          = 4
						max_duration_seconds = 450
						same_region          = false
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.type",
					"LINEAR",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.base_backoff_seconds",
					"45",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.max_retries",
					"4",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.max_duration_seconds",
					"450",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.same_region",
					"false",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
	})
}

func TestAccURLMonitorWithLinearRetryOnlyOnNetworkError(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_url_monitor" "test" {
					name      = "test-linear-retry"
					activated = true
					frequency = 720
					locations = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
					retry_strategy = {
						type = "LINEAR"

						only_on = {
							network_error = true
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.only_on.network_error",
					"true",
				),
			),
		},
		{
			Config: `
				resource "checkly_url_monitor" "test" {
					name      = "test-linear-retry"
					activated = true
					frequency = 720
					locations = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
					retry_strategy = {
						type = "LINEAR"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.only_on.network_error",
				),
			),
		},
	})
}

func TestAccURLMonitorRetryStrategyRemoval(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_url_monitor" "test" {
					name      = "test-single-retry"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
					retry_strategy = {
						type = "LINEAR"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.type",
					"LINEAR",
				),
			),
		},
		{
			Config: `
				resource "checkly_url_monitor" "test" {
					name      = "test-single-retry"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"retry_strategy.type",
					"NO_RETRIES",
				),
			),
		},
	})
}

func TestAccURLMonitorGroupAssignment(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group" "test1" {
					name        = "test-group-assignment"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}

				resource "checkly_check_group" "test2" {
					name        = "test-group-assignment"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}

				resource "checkly_url_monitor" "test1" {
					name      = "test-group-assignment"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					group_id  = checkly_check_group.test1.id
					request {
						url = "https://checkly.com"
					}
				}

				resource "checkly_url_monitor" "test2" {
					name      = "test-group-assignment"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrPair(
					"checkly_url_monitor.test1",
					"group_id",
					"checkly_check_group.test1",
					"id",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test2",
					"group_id",
					"0",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group" "test1" {
					name        = "test-group-assignment"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}

				resource "checkly_check_group" "test2" {
					name        = "test-group-assignment"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}

				resource "checkly_url_monitor" "test1" {
					name      = "test-group-assignment"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					group_id  = checkly_check_group.test2.id
					request {
						url = "https://checkly.com"
					}
				}

				resource "checkly_url_monitor" "test2" {
					name      = "test-group-assignment"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					group_id  = checkly_check_group.test2.id
					request {
						url = "https://checkly.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrPair(
					"checkly_url_monitor.test1",
					"group_id",
					"checkly_check_group.test2",
					"id",
				),
				resource.TestCheckResourceAttrPair(
					"checkly_url_monitor.test2",
					"group_id",
					"checkly_check_group.test2",
					"id",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group" "test1" {
					name        = "test-group-assignment"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}

				resource "checkly_check_group" "test2" {
					name        = "test-group-assignment"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}

				resource "checkly_url_monitor" "test1" {
					name      = "test-group-assignment"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
				}

				resource "checkly_url_monitor" "test2" {
					name      = "test-group-assignment"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test1",
					"group_id",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test2",
					"group_id",
					"0",
				),
			),
		},
	})
}

func TestAccURLMonitorFrequencyValidation(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_url_monitor" "test-missing-frequency_offset" {
					name      = "test-missing-frequency_offset"
					activated = true
					frequency = 0
					locations = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`"frequency_offset" is required when "frequency" is 0`),
		},
		{
			Config: `
				resource "checkly_url_monitor" "test-frequency_offset-ok-10" {
					name             = "test-frequency_offset-ok-10"
					activated        = true
					frequency        = 0
					frequency_offset = 10
					locations        = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
				}

				resource "checkly_url_monitor" "test-frequency_offset-ok-20" {
					name             = "test-frequency_offset-ok-20"
					activated        = true
					frequency        = 0
					frequency_offset = 20
					locations        = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
				}

				resource "checkly_url_monitor" "test-frequency_offset-ok-30" {
					name             = "test-frequency_offset-ok-30"
					activated        = true
					frequency        = 0
					frequency_offset = 30
					locations        = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test-frequency_offset-ok-10",
					"frequency",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test-frequency_offset-ok-10",
					"frequency_offset",
					"10",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test-frequency_offset-ok-20",
					"frequency",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test-frequency_offset-ok-20",
					"frequency_offset",
					"20",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test-frequency_offset-ok-30",
					"frequency",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test-frequency_offset-ok-30",
					"frequency_offset",
					"30",
				),
			),
		},
		{
			Config: `
				resource "checkly_url_monitor" "test-extra-frequency_offset" {
					name             = "test-extra-frequency_offset"
					activated        = true
					frequency        = 60
					frequency_offset = 30
					locations        = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`"frequency_offset" can only be set when "frequency" is 0`),
		},
		{
			Config: `
				resource "checkly_url_monitor" "test-bad-frequency" {
					name      = "test-bad-frequency"
					activated = true
					frequency = 9999
					locations = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`Attribute frequency value must be one of`),
		},
		{
			Config: `
				resource "checkly_url_monitor" "test-bad-frequency_offset" {
					name             = "test-bad-frequency_offset"
					activated        = true
					frequency        = 60
					frequency_offset = 99
					locations        = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
				}
			`,
			ExpectError: regexp.MustCompile(`Attribute frequency_offset value must be one of`),
		},
		{
			Config: `
				resource "checkly_url_monitor" "test-frequency-ok-60" {
					name             = "test-frequency-ok-60"
					activated        = true
					frequency        = 60
					frequency_offset = 0
					locations        = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test-frequency-ok-60",
					"frequency",
					"60",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test-frequency-ok-60",
					"frequency_offset",
					"0",
				),
			),
		},
	})
}

func TestAccURLMonitorFrequencyUpdates(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_url_monitor" "test1" {
					name      = "test-normal-frequency"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
				}

				resource "checkly_url_monitor" "test2" {
					name             = "test-high-frequency"
					activated        = true
					frequency        = 0
					frequency_offset = 30
					locations        = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test1",
					"frequency",
					"60",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_url_monitor.test1",
					"frequency_offset",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test2",
					"frequency",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test2",
					"frequency_offset",
					"30",
				),
			),
		},
		{
			Config: `
				resource "checkly_url_monitor" "test1" {
					name             = "test-normal-frequency-updated-to-high"
					activated        = true
					frequency        = 0
					frequency_offset = 10
					locations        = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
				}

				resource "checkly_url_monitor" "test2" {
					name      = "test-high-frequency-updated-to-normal"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						url = "https://checkly.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test1",
					"frequency",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test1",
					"frequency_offset",
					"10",
				),
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test2",
					"frequency",
					"60",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_url_monitor.test2",
					"frequency_offset",
				),
			),
		},
	})
}

func TestAccURLMonitorAlertChannelSubscriptions(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		// Step 1: Add two alert channel subscriptions.
		{
			Config: `
				resource "checkly_alert_channel" "email_1" {
					email {
						address = "url-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email {
						address = "url-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email {
						address = "url-acs-3@example.com"
					}
				}

				resource "checkly_url_monitor" "test" {
					name      = "url-alert-channel-sub-test"
					frequency = 60
					activated = true
					locations = ["eu-central-1"]

					request {
						url = "https://api.checklyhq.com"

						assertion {
							source     = "STATUS_CODE"
							comparison = "EQUALS"
							target     = "200"
						}
					}

					alert_channel_subscription {
						channel_id = checkly_alert_channel.email_1.id
						activated  = true
					}

					alert_channel_subscription {
						channel_id = checkly_alert_channel.email_2.id
						activated  = false
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"alert_channel_subscription.#",
					"2",
				),
				testCheckAlertChannelSubscription(
					"checkly_url_monitor.test",
					"alert_channel_subscription",
					"checkly_alert_channel.email_1",
					"true",
				),
				testCheckAlertChannelSubscription(
					"checkly_url_monitor.test",
					"alert_channel_subscription",
					"checkly_alert_channel.email_2",
					"false",
				),
			),
		},
		// Step 2: Change activations and add a third subscription.
		{
			Config: `
				resource "checkly_alert_channel" "email_1" {
					email {
						address = "url-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email {
						address = "url-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email {
						address = "url-acs-3@example.com"
					}
				}

				resource "checkly_url_monitor" "test" {
					name      = "url-alert-channel-sub-test"
					frequency = 60
					activated = true
					locations = ["eu-central-1"]

					request {
						url = "https://api.checklyhq.com"

						assertion {
							source     = "STATUS_CODE"
							comparison = "EQUALS"
							target     = "200"
						}
					}

					alert_channel_subscription {
						channel_id = checkly_alert_channel.email_1.id
						activated  = false
					}

					alert_channel_subscription {
						channel_id = checkly_alert_channel.email_2.id
						activated  = true
					}

					alert_channel_subscription {
						channel_id = checkly_alert_channel.email_3.id
						activated  = true
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"alert_channel_subscription.#",
					"3",
				),
				testCheckAlertChannelSubscription(
					"checkly_url_monitor.test",
					"alert_channel_subscription",
					"checkly_alert_channel.email_1",
					"false",
				),
				testCheckAlertChannelSubscription(
					"checkly_url_monitor.test",
					"alert_channel_subscription",
					"checkly_alert_channel.email_2",
					"true",
				),
				testCheckAlertChannelSubscription(
					"checkly_url_monitor.test",
					"alert_channel_subscription",
					"checkly_alert_channel.email_3",
					"true",
				),
			),
		},
		// Step 3: Remove all alert channel subscriptions.
		{
			Config: `
				resource "checkly_alert_channel" "email_1" {
					email {
						address = "url-acs-1@example.com"
					}
				}

				resource "checkly_alert_channel" "email_2" {
					email {
						address = "url-acs-2@example.com"
					}
				}

				resource "checkly_alert_channel" "email_3" {
					email {
						address = "url-acs-3@example.com"
					}
				}

				resource "checkly_url_monitor" "test" {
					name      = "url-alert-channel-sub-test"
					frequency = 60
					activated = true
					locations = ["eu-central-1"]

					request {
						url = "https://api.checklyhq.com"

						assertion {
							source     = "STATUS_CODE"
							comparison = "EQUALS"
							target     = "200"
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_url_monitor.test",
					"alert_channel_subscription.#",
					"0",
				),
			),
		},
	})
}

// TestAccURLMonitorUnsetSweepBasics sets every plain optional attribute to a
// non-default value and then removes them all, verifying the server actually
// clears (or resets to the default) each value. The Checkly PUT endpoint is
// PATCH-style, so a key omitted from the update payload keeps its server-side
// value; the implicit post-apply refresh would surface that as a non-empty
// plan and fail the test.
func TestAccURLMonitorUnsetSweepBasics(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_url_monitor" "test" {
					name                      = "url-unset-sweep-basics"
					activated                 = true
					frequency                 = 60
					locations                 = ["eu-central-1"]
					muted                     = true
					should_fail               = true
					run_parallel              = true
					use_global_alert_settings = false
					degraded_response_time    = 2500
					max_response_time         = 9000
					tags                      = ["sweep-a", "sweep-b"]
					request {
						url = "https://api.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_url_monitor.test", "muted", "true"),
				resource.TestCheckResourceAttr("checkly_url_monitor.test", "should_fail", "true"),
				resource.TestCheckResourceAttr("checkly_url_monitor.test", "run_parallel", "true"),
				resource.TestCheckResourceAttr("checkly_url_monitor.test", "use_global_alert_settings", "false"),
				resource.TestCheckResourceAttr("checkly_url_monitor.test", "degraded_response_time", "2500"),
				resource.TestCheckResourceAttr("checkly_url_monitor.test", "max_response_time", "9000"),
				resource.TestCheckResourceAttr("checkly_url_monitor.test", "tags.#", "2"),
			),
		},
		{
			Config: `
				resource "checkly_url_monitor" "test" {
					name      = "url-unset-sweep-basics"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						url = "https://api.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_url_monitor.test", "muted", "false"),
				resource.TestCheckResourceAttr("checkly_url_monitor.test", "should_fail", "false"),
				resource.TestCheckResourceAttr("checkly_url_monitor.test", "run_parallel", "false"),
				resource.TestCheckResourceAttr("checkly_url_monitor.test", "use_global_alert_settings", "true"),
				resource.TestCheckResourceAttr("checkly_url_monitor.test", "degraded_response_time", "3000"),
				resource.TestCheckResourceAttr("checkly_url_monitor.test", "max_response_time", "5000"),
				resource.TestCheckNoResourceAttr("checkly_url_monitor.test", "tags.#"),
			),
		},
	})
}

// TestAccURLMonitorUnsetSweepResponseTimeZero verifies that the response time
// limits accept an explicit 0 (a schema-valid value that omitempty-style
// serialization would drop) and reset to their documented defaults when
// removed from the configuration.
func TestAccURLMonitorUnsetSweepResponseTimeZero(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_url_monitor" "test" {
					name                   = "url-unset-sweep-response-time-zero"
					activated              = true
					frequency              = 60
					locations              = ["eu-central-1"]
					degraded_response_time = 0
					max_response_time      = 0
					request {
						url = "https://api.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_url_monitor.test", "degraded_response_time", "0"),
				resource.TestCheckResourceAttr("checkly_url_monitor.test", "max_response_time", "0"),
			),
		},
		{
			Config: `
				resource "checkly_url_monitor" "test" {
					name      = "url-unset-sweep-response-time-zero"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						url = "https://api.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_url_monitor.test", "degraded_response_time", "3000"),
				resource.TestCheckResourceAttr("checkly_url_monitor.test", "max_response_time", "5000"),
			),
		},
	})
}

// TestAccURLMonitorUnsetSweepLocations verifies that removing the locations
// attribute entirely clears the locations on the server.
func TestAccURLMonitorUnsetSweepLocations(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_url_monitor" "test" {
					name      = "url-unset-sweep-locations"
					activated = true
					frequency = 60
					locations = ["eu-central-1", "us-east-1"]
					request {
						url = "https://api.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_url_monitor.test", "locations.#", "2"),
			),
		},
		{
			Config: `
				resource "checkly_url_monitor" "test" {
					name      = "url-unset-sweep-locations"
					activated = true
					frequency = 60
					request {
						url = "https://api.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr("checkly_url_monitor.test", "locations.#"),
			),
		},
	})
}

// TestAccURLMonitorUnsetSweepPrivateLocations verifies that removing the
// private_locations attribute clears the private locations on the server.
func TestAccURLMonitorUnsetSweepPrivateLocations(t *testing.T) {
	privateLocation := `
		resource "checkly_private_location" "url_sweep" {
			name      = "URL Monitor Unset Sweep"
			slug_name = "tf-url-monitor-unset-sweep"
		}
	`
	accTestCase(t, []resource.TestStep{
		{
			Config: privateLocation + `
				resource "checkly_url_monitor" "test" {
					name              = "url-unset-sweep-private-locations"
					activated         = true
					frequency         = 60
					locations         = ["eu-central-1"]
					private_locations = [checkly_private_location.url_sweep.slug_name]
					request {
						url = "https://api.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_url_monitor.test", "private_locations.#", "1"),
			),
		},
		{
			Config: privateLocation + `
				resource "checkly_url_monitor" "test" {
					name      = "url-unset-sweep-private-locations"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						url = "https://api.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr("checkly_url_monitor.test", "private_locations.#"),
			),
		},
	})
}

// TestAccURLMonitorUnsetSweepRequest sets the optional request attributes to
// non-default values plus an assertion, then removes them, verifying the
// attributes reset to their defaults and the assertions are cleared on the
// server.
func TestAccURLMonitorUnsetSweepRequest(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_url_monitor" "test" {
					name      = "url-unset-sweep-request"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						url              = "https://api.checklyhq.com"
						follow_redirects = false
						skip_ssl         = true
						ip_family        = "IPv6"
						assertion {
							source     = "STATUS_CODE"
							comparison = "EQUALS"
							target     = "200"
						}
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				testCheckResourceAttrExpr("checkly_url_monitor.test", `follow_redirects":"false"`, "false"),
				testCheckResourceAttrExpr("checkly_url_monitor.test", `skip_ssl":"true"`, "true"),
				testCheckResourceAttrExpr("checkly_url_monitor.test", `ip_family":"IPv6"`, "IPv6"),
				testCheckResourceAttrExpr("checkly_url_monitor.test", `assertion.#":"1"`, "1"),
			),
		},
		{
			Config: `
				resource "checkly_url_monitor" "test" {
					name      = "url-unset-sweep-request"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						url = "https://api.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				testCheckResourceAttrExpr("checkly_url_monitor.test", `follow_redirects":"true"`, "true"),
				testCheckResourceAttrExpr("checkly_url_monitor.test", `skip_ssl":"false"`, "false"),
				testCheckResourceAttrExpr("checkly_url_monitor.test", `ip_family":"IPv4"`, "IPv4"),
				testCheckResourceAttrExpr("checkly_url_monitor.test", `assertion.#":"0"`, "0"),
			),
		},
	})
}

// TestAccURLMonitorUnsetSweepTriggerIncident verifies that removing the
// trigger_incident block clears the incident trigger on the server.
func TestAccURLMonitorUnsetSweepTriggerIncident(t *testing.T) {
	service := `
		resource "checkly_status_page_service" "url_sweep" {
			name = "URL Monitor Unset Sweep Service"
		}
	`
	accTestCase(t, []resource.TestStep{
		{
			Config: service + `
				resource "checkly_url_monitor" "test" {
					name      = "url-unset-sweep-trigger-incident"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						url = "https://api.checklyhq.com"
					}
					trigger_incident {
						service_id         = checkly_status_page_service.url_sweep.id
						severity           = "MEDIUM"
						name               = "URL sweep incident"
						description        = "Incident created by the unset sweep test"
						notify_subscribers = false
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_url_monitor.test", "trigger_incident.#", "1"),
			),
		},
		{
			Config: service + `
				resource "checkly_url_monitor" "test" {
					name      = "url-unset-sweep-trigger-incident"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						url = "https://api.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_url_monitor.test", "trigger_incident.#", "0"),
			),
		},
	})
}

// TestAccURLMonitorUnsetSweepGroup verifies that removing group_id and
// group_order detaches the monitor from the group and resets the order.
func TestAccURLMonitorUnsetSweepGroup(t *testing.T) {
	group := `
		resource "checkly_check_group" "url_sweep" {
			name        = "url-unset-sweep-group"
			activated   = true
			concurrency = 1
			locations   = ["eu-central-1"]
		}
	`
	accTestCase(t, []resource.TestStep{
		{
			Config: group + `
				resource "checkly_url_monitor" "test" {
					name        = "url-unset-sweep-group"
					activated   = true
					frequency   = 60
					locations   = ["eu-central-1"]
					group_id    = checkly_check_group.url_sweep.id
					group_order = 2
					request {
						url = "https://api.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrPair(
					"checkly_url_monitor.test",
					"group_id",
					"checkly_check_group.url_sweep",
					"id",
				),
				resource.TestCheckResourceAttr("checkly_url_monitor.test", "group_order", "2"),
			),
		},
		{
			Config: group + `
				resource "checkly_url_monitor" "test" {
					name      = "url-unset-sweep-group"
					activated = true
					frequency = 60
					locations = ["eu-central-1"]
					request {
						url = "https://api.checklyhq.com"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_url_monitor.test", "group_id", "0"),
				resource.TestCheckResourceAttr("checkly_url_monitor.test", "group_order", "0"),
			),
		},
	})
}
