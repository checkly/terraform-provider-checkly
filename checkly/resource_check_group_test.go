package checkly

import (
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/checkly/checkly-go-sdk"
)

func TestEncodeDecodeGroupResource(t *testing.T) {
	res := resourceCheckGroup()
	data := res.TestResourceData()
	resourceDataFromCheckGroup(&wantGroup, data)
	gotGroup, err := checkGroupFromResourceData(data)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(wantGroup, gotGroup) {
		t.Error(cmp.Diff(wantGroup, gotGroup))
	}
}

var wantGroup = checkly.Group{
	Name:             "test",
	Activated:        true,
	Muted:            false,
	Tags:             []string{"auto"},
	Locations:        []string{"eu-west-1"},
	PrivateLocations: &[]string{},
	Concurrency:      3,
	APICheckDefaults: checkly.APICheckDefaults{
		BaseURL: "example.com/api/test",
		Headers: []checkly.KeyValue{
			{
				Key:   "X-Test",
				Value: "foo",
			},
		},
		QueryParameters: []checkly.KeyValue{
			{
				Key:   "query",
				Value: "foo",
			},
		},
		Assertions: []checkly.Assertion{
			{
				Source:     checkly.StatusCode,
				Comparison: checkly.Equals,
				Target:     "200",
			},
		},
		BasicAuth: checkly.BasicAuth{
			Username: "user",
			Password: "pass",
		},
	},
	EnvironmentVariables: []checkly.EnvironmentVariable{
		{
			Key:   "ENVTEST",
			Value: "Hello world",
		},
	},
	DoubleCheck:            true,
	UseGlobalAlertSettings: false,
	AlertSettings: checkly.AlertSettings{
		EscalationType: checkly.RunBased,
		RunBasedEscalation: checkly.RunBasedEscalation{
			FailedRunThreshold: 1,
		},
		Reminders: checkly.Reminders{
			Amount:   0,
			Interval: 5,
		},
	},
	LocalSetupScript:          "setup-test",
	LocalTearDownScript:       "teardown-test",
	AlertChannelSubscriptions: []checkly.AlertChannelSubscription{},
}

func TestAccCheckGroupEmptyConfig(t *testing.T) {
	config := `resource "checkly_check_group" "test" {}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "name" is required`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "concurrency" is required`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "activated" is required`),
		},
	})
}

func TestAccCheckGroupInvalid(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config:      testCheckGroup_invalid,
			ExpectError: regexp.MustCompile(`Inappropriate value for attribute "locations"`),
		},
		{
			Config:      testCheckGroup_invalid,
			ExpectError: regexp.MustCompile(`Inappropriate value for attribute "muted"`),
		},
		{
			Config:      testCheckGroup_invalid,
			ExpectError: regexp.MustCompile(`Inappropriate value for attribute "activated"`),
		},
		{
			Config:      testCheckGroup_invalid,
			ExpectError: regexp.MustCompile(`The argument "concurrency" is required`),
		},
		{
			Config:      testCheckGroup_invalid,
			ExpectError: regexp.MustCompile(`Missing required argument`),
		},
	})
}

func TestAccCheckGroupBasic(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: testCheckGroup_basic,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"name",
					"test",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"activated",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"muted",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"concurrency",
					"3",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"locations.#",
					"2",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"locations.*",
					"us-east-1",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"locations.*",
					"eu-central-1",
				),
			),
		},
	})
}

func TestAccCheckGroupWithApiDefaults(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: testCheckGroup_withApiDefaults,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"name",
					"test",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"api_check_defaults.*.url",
					"http://api.example.com/",
				),
			),
		},
	})
}

func TestAccCheckGroupFull(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: testCheckGroup_full,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"name",
					"test",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"activated",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"muted",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"concurrency",
					"3",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"double_check",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"use_global_alert_settings",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"locations.#",
					"2",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"locations.*",
					"us-east-1",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"locations.*",
					"eu-central-1",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"environment_variables.FOO",
					"BAR",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"alert_settings.*.escalation_type",
					"RUN_BASED",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"alert_settings.*.reminders.#",
					"1",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"alert_settings.*.reminders.*.amount",
					"2",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"alert_settings.*.reminders.*.interval",
					"5",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"alert_settings.*.run_based_escalation.#",
					"1",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"alert_settings.*.run_based_escalation.*.failed_run_threshold",
					"1",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"api_check_defaults.#",
					"1",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"api_check_defaults.*.assertion.#",
					"1",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"api_check_defaults.*.assertion.*.comparison",
					"EQUALS",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"api_check_defaults.*.assertion.*.property",
					"",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"api_check_defaults.*.assertion.*.source",
					"STATUS_CODE",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"api_check_defaults.*.assertion.*.target",
					"200",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"api_check_defaults.*.basic_auth.#",
					"1",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"api_check_defaults.*.basic_auth.*.password",
					"pass",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"api_check_defaults.*.basic_auth.*.username",
					"user",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"api_check_defaults.*.headers.X-Test",
					"foo",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"api_check_defaults.*.query_parameters.query",
					"foo",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"api_check_defaults.*.url",
					"http://example.com/",
				),
			),
		},
	})
}

const testCheckGroup_invalid = `
	resource "checkly_check_group" "test" {
		name = "test"
		activated = "invalid"
		muted = "invalid"
		locations = "invalid"
	}
`

const testCheckGroup_basic = `
	resource "checkly_check_group" "test" {
		name        = "test"
		activated   = true
		muted       = false
		concurrency = 3
		locations   = [
			"us-east-1",
			"eu-central-1",
		]
	}
`

const testCheckGroup_withApiDefaults = `
	resource "checkly_check_group" "test" {
		name        = "test"
		activated   = true
		muted       = false
		concurrency = 3
		locations   = [
			"eu-west-1",
			"eu-west-2"
		]
		api_check_defaults {
			url = "http://api.example.com/"
		}
	}
`

const testCheckGroup_full = `
  resource "checkly_check_group" "test" {
	name                      = "test"
	activated                 = true
	muted                     = false
	concurrency               = 3
	double_check              = true
	use_global_alert_settings = false
	locations = [ "us-east-1", "eu-central-1" ]
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
	environment_variables = {
	  FOO = "BAR"
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
		enabled = false
		percentage = 10
		}
	}
	local_setup_script    = "setup-test"
	local_teardown_script = "teardown-test"
  }
`

func TestAccCheckGroupWithSingleRetry(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "test-single-retry"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
					retry_strategy {
						type                 = "SINGLE_RETRY"
						base_backoff_seconds = 30
						same_region          = true
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.0.type",
					"SINGLE_RETRY",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.0.base_backoff_seconds",
					"30",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.0.max_retries",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.0.max_duration_seconds",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.0.same_region",
					"true",
				),
			),
		},
	})
}

func TestAccCheckGroupWithNoRetries(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "test-no-retries"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
					retry_strategy {
						type = "NO_RETRIES"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.0.type",
					"NO_RETRIES",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.0.base_backoff_seconds",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.0.max_retries",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.0.max_duration_seconds",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.0.same_region",
					"false",
				),
			),
		},
	})
}

func TestAccCheckGroupWithDefaultNoRetries(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "test-no-retries"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.0.type",
					"NO_RETRIES",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.0.base_backoff_seconds",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.0.max_retries",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.0.max_duration_seconds",
					"0",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.0.same_region",
					"false",
				),
			),
		},
	})
}

func TestAccCheckGroupWithFixedRetry(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "test-fixed-retry"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
					retry_strategy {
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
					"checkly_check_group.test",
					"retry_strategy.0.type",
					"FIXED",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.0.base_backoff_seconds",
					"30",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.0.max_retries",
					"3",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.0.max_duration_seconds",
					"300",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.0.same_region",
					"false",
				),
			),
		},
	})
}

func TestAccCheckGroupWithExponentialRetry(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "test-exponential-retry"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
					retry_strategy {
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
					"checkly_check_group.test",
					"retry_strategy.0.type",
					"EXPONENTIAL",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.0.base_backoff_seconds",
					"60",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.0.max_retries",
					"5",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.0.max_duration_seconds",
					"600",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.0.same_region",
					"true",
				),
			),
		},
	})
}

func TestAccCheckGroupWithLinearRetry(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "test-linear-retry"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
					retry_strategy {
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
					"checkly_check_group.test",
					"retry_strategy.0.type",
					"LINEAR",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.0.base_backoff_seconds",
					"45",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.0.max_retries",
					"4",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.0.max_duration_seconds",
					"450",
				),
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.0.same_region",
					"false",
				),
			),
		},
	})
}

func TestAccCheckGroupRetryStrategyRemoval(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "test-linear-retry"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
					retry_strategy {
						type = "LINEAR"
					}
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.0.type",
					"LINEAR",
				),
			),
		},
		{
			Config: `
				resource "checkly_check_group" "test" {
					name        = "test-linear-retry"
					activated   = true
					concurrency = 1
					locations   = ["eu-central-1"]
				}
			`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check_group.test",
					"retry_strategy.0.type",
					"NO_RETRIES",
				),
			),
		},
	})
}
