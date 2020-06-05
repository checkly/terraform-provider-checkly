package main

import (
	"testing"

	"github.com/checkly/checkly-go-sdk"
	"github.com/google/go-cmp/cmp"
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
	LocalSetupScript:    "setup-test",
	LocalTearDownScript: "teardown-test",
}
