package main

import (
	"net/http"
	"testing"

	"github.com/checkly/checkly-go-sdk"
	"github.com/google/go-cmp/cmp"
)

var wantCheck = checkly.Check{
	Name:                 "My test check",
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
			{
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

func TestEncodeDecodeResource(t *testing.T) {
	res := resourceCheck()
	data := res.TestResourceData()
	resourceDataFromCheck(&wantCheck, data)
	got, err := checkFromResourceData(data)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(wantCheck, got) {
		t.Error(cmp.Diff(wantCheck, got))
	}
}
