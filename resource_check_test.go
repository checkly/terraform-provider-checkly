package main

import (
	"net/http"
	"testing"

	"github.com/bitfield/checkly"
	"github.com/google/go-cmp/cmp"
)

func TestEncodeDecodeResource(t *testing.T) {
	want := testCheck("encode-decode-test")
	res := resourceCheck()
	data := res.TestResourceData()
	resourceDataFromCheck(&want, data)
	got, err := checkFromResourceData(data)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func testCheck(name string) checkly.Check {
	return checkly.Check{
		Name:                 name,
		Type:                 checkly.TypeAPI,
		Frequency:            1,
		Activated:            true,
		Muted:                false,
		ShouldFail:           false,
		Locations:            []string{"eu-west-1"},
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
		SSLCheck:            true,
		SSLCheckDomain:      "example.com",
		LocalSetupScript:    "bogus",
		LocalTearDownScript: "bogus",
		AlertSettings: checkly.AlertSettings{
			EscalationType: checkly.RunBased,
			RunBasedEscalation: checkly.RunBasedEscalation{
				FailedRunThreshold: 1,
			},
			TimeBasedEscalation: checkly.TimeBasedEscalation{
				MinutesFailingThreshold: 5,
			},
			Reminders: checkly.Reminders{
				Interval: 5,
			},
			SSLCertificates: checkly.SSLCertificates{
				Enabled:        false,
				AlertThreshold: 3,
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
				checkly.Assertion{
					Source:     checkly.StatusCode,
					Comparison: checkly.Equals,
					Target:     "200",
				},
			},
			Body:     "",
			BodyType: "NONE",
			BasicAuth: checkly.BasicAuth{
				Username: "example",
				Password: "pass",
			},
		},
	}
}
