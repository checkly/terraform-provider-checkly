package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
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
	client, err := checkly.NewClient(
		checkly.WithBaseURL(server.URL),
		checkly.WithAuth("test-key", "00000000-0000-4000-8000-000000000000"),
		checkly.WithHTTPClient(server.Client()),
	)
	if err != nil {
		t.Fatal(err)
	}
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

func TestSetElementsWithDefaults(t *testing.T) {
	ctx := context.Background()
	attrTypes := map[string]attr.Type{
		"key":   types.StringType,
		"value": types.StringType,
		"note":  types.StringType,
	}
	elemType := types.ObjectType{AttrTypes: attrTypes}
	element := func(key string, value, note types.String) attr.Value {
		return types.ObjectValueMust(attrTypes, map[string]attr.Value{
			"key":   types.StringValue(key),
			"value": value,
			"note":  note,
		})
	}
	set := func(elements ...attr.Value) types.Set {
		return types.SetValueMust(elemType, elements)
	}
	type model struct {
		Key   types.String `tfsdk:"key"`
		Value types.String `tfsdk:"value"`
		Note  types.String `tfsdk:"note"`
	}
	// withDefaults defaults value, transforms key and leaves note alone, so
	// it shows that untouched attributes keep their null or unknown values.
	withDefaults := func(m *model) {
		m.Value = stringOrDefault(m.Value, "")
		m.Key = types.StringValue(strings.ToUpper(m.Key.ValueString()))
	}

	cases := map[string]struct {
		in        types.Set
		priorErr  bool
		want      types.Set
		wantError string
	}{
		"null set": {
			in:   types.SetNull(elemType),
			want: types.SetNull(elemType),
		},
		"unknown set": {
			in:   types.SetUnknown(elemType),
			want: types.SetUnknown(elemType),
		},
		"empty set": {
			in:   set(),
			want: set(),
		},
		"defaults and transforms": {
			in: set(
				element("a", types.StringNull(), types.StringNull()),
				element("b", types.StringUnknown(), types.StringUnknown()),
				element("c", types.StringValue("kept"), types.StringValue("kept")),
			),
			want: set(
				element("A", types.StringValue(""), types.StringNull()),
				element("B", types.StringValue(""), types.StringUnknown()),
				element("C", types.StringValue("kept"), types.StringValue("kept")),
			),
		},
		"null element": {
			in:        set(types.ObjectNull(attrTypes)),
			want:      set(types.ObjectNull(attrTypes)),
			wantError: "Value Conversion Error",
		},
		"unknown element": {
			in:        set(element("a", types.StringNull(), types.StringNull()), types.ObjectUnknown(attrTypes)),
			want:      set(element("a", types.StringNull(), types.StringNull()), types.ObjectUnknown(attrTypes)),
			wantError: "Value Conversion Error",
		},
		"earlier error": {
			in:        set(element("a", types.StringNull(), types.StringNull())),
			priorErr:  true,
			want:      set(element("a", types.StringNull(), types.StringNull())),
			wantError: "Earlier Error",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var diags diag.Diagnostics
			if tc.priorErr {
				diags.AddError("Earlier Error", "")
			}

			got := setElementsWithDefaults(ctx, tc.in, withDefaults, &diags)
			if !got.Equal(tc.want) {
				t.Errorf("got %s, want %s", got, tc.want)
			}

			if tc.wantError == "" {
				if diags.HasError() {
					t.Errorf("unexpected errors: %v", diags)
				}
				return
			}
			if diags.ErrorsCount() != 1 || diags.Errors()[0].Summary() != tc.wantError {
				t.Errorf("errors = %v, want one %q", diags, tc.wantError)
			}
		})
	}
}
