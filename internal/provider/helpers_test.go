package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	checkly "github.com/checkly/checkly-go-sdk"
)

func TestIsNotFoundError(t *testing.T) {
	// One real 404 through the SDK client checks that the client's error
	// for a missing resource is what isNotFoundError recognises.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"statusCode":404,"error":"Not Found","message":"Not Found"}`))
	}))
	defer server.Close()
	client := checkly.NewClient(server.URL, "test-key", server.Client(), nil)
	_, notFound := client.GetEnvironmentVariable(context.Background(), "MISSING")
	if notFound == nil {
		t.Fatal("GetEnvironmentVariable on a 404 succeeded, want an error")
	}

	cases := map[string]struct {
		err  error
		want bool
	}{
		"nil":          {nil, false},
		"404 response": {notFound, true},
		"wrapped 404": {
			fmt.Errorf("reading environment variable: %w", &checkly.APIError{StatusCode: http.StatusNotFound}), true,
		},
		// A validation message can quote user input that contains "404".
		"400 response whose message mentions 404": {
			&checkly.APIError{StatusCode: http.StatusBadRequest, Message: `"API 404" is not a valid key, 404 Not Found`}, false,
		},
		"500 response for an ID containing 404": {
			&checkly.APIError{Method: http.MethodGet, Path: "/v1/variables/KEY_1404", StatusCode: http.StatusInternalServerError}, false,
		},
		"transport error for a path containing 404": {
			errors.New("GET /v1/variables/PAGE_404_URL: request to api.checklyhq.com failed: context deadline exceeded"), false,
		},
		// Only typed errors count: text in the format of older SDK versions
		// is not a 404.
		"404 in the old text format": {
			errors.New(`unexpected response status 404: "{\"statusCode\":404}"`), false,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if got := isNotFoundError(c.err); got != c.want {
				t.Errorf("isNotFoundError(%v) = %v, want %v", c.err, got, c.want)
			}
		})
	}
}

func TestParseAPIRequestTimeout(t *testing.T) {
	cases := map[string]struct {
		value   string
		want    time.Duration
		wantErr bool
	}{
		"unset":         {"", 15 * time.Second, false},
		"whole seconds": {"60", 60 * time.Second, false},
		"with a unit":   {"30s", 0, true},
		"fractional":    {"1.5", 0, true},
		"zero":          {"0", 0, true},
		"negative":      {"-5", 0, true},
		"not a number":  {"abc", 0, true},
		"overflowing":   {"9223372037", 0, true},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv("API_CALL_TIMEOUT", c.value)

			got, err := parseAPIRequestTimeout()
			if (err != nil) != c.wantErr {
				t.Fatalf("parseAPIRequestTimeout() error = %v, wantErr %v", err, c.wantErr)
			}
			if got != c.want {
				t.Errorf("parseAPIRequestTimeout() = %v, want %v", got, c.want)
			}
		})
	}
}
