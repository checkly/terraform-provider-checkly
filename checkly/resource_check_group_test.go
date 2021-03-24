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
	Name:        "test",
	Activated:   true,
	Muted:       false,
	Tags:        []string{"auto"},
	Locations:   []string{"eu-west-1"},
	Concurrency: 3,
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
		TimeBasedEscalation: checkly.TimeBasedEscalation{
			MinutesFailingThreshold: 5,
		},
		Reminders: checkly.Reminders{
			Amount:   0,
			Interval: 5,
		},
		SSLCertificates: checkly.SSLCertificates{
			Enabled:        true,
			AlertThreshold: 30,
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
			ExpectError: regexp.MustCompile(`The argument "locations" is required`),
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
	})
}

func TestAccCheckGroupBasic(t *testing.T) {
	return
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
					"alert_settings.*.ssl_certificates.#",
					"1",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"alert_settings.*.ssl_certificates.*.enabled",
					"true",
				),
				testCheckResourceAttrExpr(
					"checkly_check_group.test",
					"alert_settings.*.time_based_escalation.*.minutes_failing_threshold",
					"5",
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
	  time_based_escalation {
		minutes_failing_threshold = 5
	  }
	  ssl_certificates {
		enabled         = true
		alert_threshold = 30
	  }
	  reminders {
		amount   = 2
		interval = 5
	  }
	}
	local_setup_script    = "setup-test"
	local_teardown_script = "teardown-test"
  }
`

const testCheck_groupWithChecks = `
  resource "checkly_check" "test" {
	name      = "test"
	type      = "API"
	activated = true
	muted     = true
	frequency = 720
	locations = [ "eu-central-1", "us-east-2" ]
	request {
	  method           = "GET"
	  url              = "https://api.checklyhq.com/public-stats"
	  follow_redirects = true
	}
	group_id    = checkly_check_group.check-group-1.id
	group_order = 1 #The group_order attribute specifies in which order the checks will be executed: 1, 2, 3, etc.
  }
`

const testCheck_checkInGroup = `
  resource "checkly_check" "test" {
	name      = "test"
	type      = "API"
	activated = true
	muted     = true
	frequency = 720
	locations = [ "eu-central-1", "us-east-2" ]
	request {
	  method           = "GET"
	  url              = "https://api.checklyhq.com/public-stats"
	  follow_redirects = true
	}
	group_id    = checkly_check_group.check-group-1.id
	group_order = 2
  }
`
