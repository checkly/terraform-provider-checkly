package checkly

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/checkly/checkly-go-sdk"
)

func TestAccCheckRequiredFields(t *testing.T) {
	config := `resource "checkly_check" "test" {}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "type" is required, but no definition was found.`),
		},
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
	})
}

func TestAccBrowserCheckInvalidInputs(t *testing.T) {
	config := `resource "checkly_check" "test" {
		name                      = 1
		type                      = "BROWSER"
		activated                 = "invalid"
		should_fail               = "invalid"
		double_check              = "invalid"
		use_global_alert_settings = "invalid"
		locations = "invalid"
		script = 4
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`Inappropriate value for attribute "activated"`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "frequency" is required`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`Inappropriate value for attribute "should_fail"`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`Inappropriate value for attribute "use_global_alert_settings"`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`Inappropriate value for attribute "double_check"`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`Inappropriate value for attribute "locations"`),
		},
	})
}

func TestAccBrowserCheckMissingScript(t *testing.T) {
	config := `resource "checkly_check" "test" {
		type = "BROWSER"
		activated = true
		frequency = 10
		name = "browser check"
		locations = [ "us-west-1" ]
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`API error 1: unexpected response status 400`),
		},
	})
}

func TestAccBrowserCheckBasic(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: browserCheck_basic,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"name",
					"Browser Check",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"type",
					"BROWSER",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"activated",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"script",
					"console.log('test')",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"locations.#",
					"2",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"locations.*",
					"us-east-1",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"locations.*",
					"eu-central-1",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"tags.#",
					"2",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"tags.*",
					"browser",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"tags.*",
					"e2e",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_check.test",
					"request",
				),
			),
		},
	})
}

func TestAccApiCheckBasic(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: apiCheck_basic,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"name",
					"API Check 1",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"activated",
					"true",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"locations.*",
					"eu-central-1",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"locations.*",
					"us-east-1",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.method",
					"GET",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.url",
					"https://api.checklyhq.com/public-stats",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.assertion.*.comparison",
					"EQUALS",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.assertion.*.property",
					"",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.assertion.*.source",
					"STATUS_CODE",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.assertion.*.target",
					"200",
				),
			),
		},
	})
}

func TestAccApiCheckFull(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: apiCheck_full,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"degraded_response_time",
					"15000",
				),
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"max_response_time",
					"30000",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_check.test",
					"environment_variables",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					`"locations.#"`,
					"3",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					`"request.*.headers.X-CUSTOM-1"`,
					"1",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.headers.X-CUSTOM-2",
					"FOO",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.query_parameters.param1",
					"123",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.query_parameters.param2",
					"bar",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.basic_auth.*.username",
					"user",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.basic_auth.*.password",
					"pass",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.assertion.#",
					"3",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.assertion.*.comparison",
					"EQUALS",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.assertion.*.comparison",
					"GREATER_THAN",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.assertion.*.target",
					"200",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.assertion.*.target",
					"no-cache",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.assertion.*.target",
					"100",
				),
			),
		},
	})
}

func TestAccApiCheckMore(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: apiCheck_post,
			Check: resource.ComposeTestCheckFunc(
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.method",
					"POST",
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.body",
					`{\"message\":\"hello checkly\",\"messageId\":1}`,
				),
				testCheckResourceAttrExpr(
					"checkly_check.test",
					"request.*.body_type",
					"JSON",
				),
			),
		},
		{
			Config: apiCheck_withEmptyBasicAuth,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_check.test",
					"name",
					"api check with empty basic_auth",
				),
			),
		},
	})
}

var wantCheck = checkly.Check{
	Name:                 "My test check",
	Type:                 checkly.TypeAPI,
	Frequency:            1,
	Activated:            true,
	Muted:                false,
	ShouldFail:           false,
	Locations:            []string{"eu-west-1"},
	PrivateLocations:     &[]string{},
	Script:               "foo",
	DegradedResponseTime: 15000,
	MaxResponseTime:      30000,
	EnvironmentVariables: []checkly.EnvironmentVariable{
		{
			Key:   "ENVTEST",
			Value: "Hello world",
		},
	},
	DoubleCheck: false,
	Tags: []string{
		"foo",
		"bar",
	},
	SSLCheck:            false,
	LocalSetupScript:    "bogus",
	LocalTearDownScript: "bogus",
	AlertSettings: checkly.AlertSettings{
		EscalationType: checkly.RunBased,
		RunBasedEscalation: checkly.RunBasedEscalation{
			FailedRunThreshold: 1,
		},
		Reminders: checkly.Reminders{
			Interval: 5,
		},
	},
	UseGlobalAlertSettings: false,
	Request: checkly.Request{
		Method: http.MethodGet,
		URL:    "http://example.com",
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
		Body:     "",
		BodyType: "NONE",
		BasicAuth: &checkly.BasicAuth{
			Username: "example",
			Password: "pass",
		},
	},
}

func TestEncodeDecodeResource(t *testing.T) {
	res := resourceCheck()
	data := res.TestResourceData()
	wantCheck.AlertChannelSubscriptions = []checkly.AlertChannelSubscription{}
	resourceDataFromCheck(&wantCheck, data)
	got, err := checkFromResourceData(data)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(wantCheck, got) {
		t.Error(cmp.Diff(wantCheck, got))
	}
}

const browserCheck_basic = `
	resource "checkly_check" "test" {
		name                      = "Browser Check"
		type                      = "BROWSER"
		activated                 = true
		should_fail               = false
		frequency                 = 720
		double_check              = true
		use_global_alert_settings = true
		locations                 = [ "us-east-1", "eu-central-1" ]
		tags                      = [ "browser", "e2e" ]
		script                    = "console.log('test')"
	}
`
const apiCheck_basic = `
	resource "checkly_check" "test" {
	  name                      = "API Check 1"
	  type                      = "API"
	  frequency                 = 60
	  activated                 = true
	  muted                     = true
	  double_check              = true
	  max_response_time         = 18000
	  locations                 = [ "us-east-1", "eu-central-1" ]
	  use_global_alert_settings = true
	  request {
		method     = "GET"
		url        = "https://api.checklyhq.com/public-stats"
		assertion {
		  comparison = "EQUALS"
		  property   = ""
		  source     = "STATUS_CODE"
		  target     = "200"
		}
	  }
	}
`

const apiCheck_full = `
  resource "checkly_check" "test" {
	name                   = "apiCheck_full"
	type                   = "API"
	frequency              = 120
	activated              = true
	muted                  = true
	double_check           = true
	degraded_response_time = 15000
	max_response_time      = 30000
	environment_variables  = null
	locations = [
	  "eu-central-1",
	  "us-east-1",
	  "ap-northeast-1"
	]
	request {
	  method           = "GET"
	  url              = "https://api.checklyhq.com/public-stats"
	  follow_redirects = true
	  headers = {
		X-CUSTOM-1 = 1
		X-CUSTOM-2 = "foo"
	  }
	  query_parameters = {
		param1 = 123
		param2 = "bar"
	  }
	  basic_auth {
		username = "user"
		password = "pass"
	  }
	  assertion {
		comparison = "EQUALS"
		property   = ""
		source     = "STATUS_CODE"
		target     = "200"
	  }
	  assertion {
		comparison = "EQUALS"
		property   = "cache-control"
		source     = "HEADERS"
		target     = "no-cache"
	  }
	  assertion {
		comparison = "GREATER_THAN"
		property   = "$.apiCheckResults"
		source     = "JSON_BODY"
		target     = "100"
	  }
	}

	alert_settings {
	  escalation_type = "RUN_BASED"
	  reminders {
		amount   = 0
		interval = 5
	  }
	  run_based_escalation {
		failed_run_threshold = 1
	  }
	}
  }
`

const apiCheck_post = `
  resource "checkly_check" "test" {
	name         = "apiCheck_post"
	type         = "API"
	activated    = true
	double_check = true
	frequency    = 720
	locations = [ "eu-central-1", "us-east-2" ]
	max_response_time     = 18000
	muted                 = true
	environment_variables = null
	request {
	  method           = "POST"
	  url              = "https://jsonplaceholder.typicode.com/posts"
	  headers = {
		Content-type = "application/json; charset=UTF-8"
	  }
	  body      = "{\"message\":\"hello checkly\",\"messageId\":1}"
	  body_type = "JSON"
	}
	use_global_alert_settings = true
  }
`

const apiCheck_withEmptyBasicAuth = `
  resource "checkly_check" "test" {
	name                   = "api check with empty basic_auth"
	type                   = "API"
	activated              = true
	should_fail            = false
	frequency              = 1
	degraded_response_time = 3000
	max_response_time      = 6000
	tags = [
	  "testing",
	  "bug"
	]
	locations = [ "eu-central-1" ]
	request {
	  follow_redirects = false
	  url              = "https://api.checklyhq.com/public-stats"
	  basic_auth {
		username = ""
		password = ""
	  }
	  assertion {
		source     = "STATUS_CODE"
		property   = ""
		comparison = "EQUALS"
		target     = "200"
	  }
	}
  }
`
