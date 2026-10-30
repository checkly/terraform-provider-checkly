package checkly

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	checkly "github.com/checkly/checkly-go-sdk"
)

func TestMergeSecretEnvironmentVariableValues(t *testing.T) {
	tests := []struct {
		name     string
		existing []checkly.EnvironmentVariable
		evs      []checkly.EnvironmentVariable
		want     []checkly.EnvironmentVariable
	}{
		{
			name: "secret values are restored from the last known values",
			existing: []checkly.EnvironmentVariable{
				{Key: "API_KEY", Value: "applied-secret", Secret: true},
				{Key: "REGION", Value: "eu-west-1"},
			},
			evs: []checkly.EnvironmentVariable{
				{Key: "API_KEY", Value: "", Secret: true},
				{Key: "REGION", Value: "eu-west-1"},
			},
			want: []checkly.EnvironmentVariable{
				{Key: "API_KEY", Value: "applied-secret", Secret: true},
				{Key: "REGION", Value: "eu-west-1"},
			},
		},
		{
			name:     "non-secret API values are kept as returned",
			existing: []checkly.EnvironmentVariable{{Key: "API_KEY", Value: "applied-secret", Secret: true}},
			evs:      []checkly.EnvironmentVariable{{Key: "REGION", Value: "eu-west-1"}},
			want:     []checkly.EnvironmentVariable{{Key: "REGION", Value: "eu-west-1"}},
		},
		{
			name:     "secrets without a known value are left untouched",
			existing: []checkly.EnvironmentVariable{{Key: "OTHER", Value: "other-secret", Secret: true}},
			evs:      []checkly.EnvironmentVariable{{Key: "API_KEY", Value: "", Secret: true}},
			want:     []checkly.EnvironmentVariable{{Key: "API_KEY", Value: "", Secret: true}},
		},
		{
			name:     "no existing secrets returns the API values as is",
			existing: []checkly.EnvironmentVariable{{Key: "REGION", Value: "eu-west-1"}},
			evs:      []checkly.EnvironmentVariable{{Key: "REGION", Value: "eu-west-1"}},
			want:     []checkly.EnvironmentVariable{{Key: "REGION", Value: "eu-west-1"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeSecretEnvironmentVariableValues(tt.existing, tt.evs)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mergeSecretEnvironmentVariableValues() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMergeSecretEnvironmentVariableValuesDoesNotMutateInput(t *testing.T) {
	evs := []checkly.EnvironmentVariable{{Key: "API_KEY", Value: "", Secret: true}}
	mergeSecretEnvironmentVariableValues(
		[]checkly.EnvironmentVariable{{Key: "API_KEY", Value: "applied-secret", Secret: true}},
		evs,
	)
	if evs[0].Value != "" {
		t.Errorf("input evs was mutated: value = %q", evs[0].Value)
	}
}

// Every resource with an inline environment_variable attribute must keep a
// secret's value in state when the API returns it without one.
func TestReadKeepsInlineSecretEnvironmentVariableValues(t *testing.T) {
	// The API returns secrets with a null value.
	apiVars := func() []checkly.EnvironmentVariable {
		return []checkly.EnvironmentVariable{
			{Key: "API_KEY", Value: "", Secret: true},
			{Key: "REGION", Value: "eu-west-1"},
		}
	}

	tests := []struct {
		name     string
		resource *schema.Resource
		// state holds attributes the read path needs besides the
		// environment variables.
		state map[string]any
		read  func(d *schema.ResourceData) error
	}{
		{
			name:     "checkly_check",
			resource: resourceCheck(),
			read: func(d *schema.ResourceData) error {
				return resourceDataFromCheck(&checkly.Check{EnvironmentVariables: apiVars()}, d)
			},
		},
		{
			name:     "checkly_check_group",
			resource: resourceCheckGroup(),
			read: func(d *schema.ResourceData) error {
				return resourceDataFromCheckGroup(&checkly.Group{EnvironmentVariables: apiVars()}, d)
			},
		},
		{
			name:     "checkly_check_group_v2",
			resource: resourceCheckGroupV2(),
			read: func(d *schema.ResourceData) error {
				r, err := CheckGroupV2ResourceFromAPIModel(&checkly.GroupV2{EnvironmentVariables: apiVars()}, d)
				if err != nil {
					return err
				}
				return r.StoreResourceData(d)
			},
		},
		{
			name:     "checkly_playwright_check_suite",
			resource: resourcePlaywrightCheckSuite(),
			state: map[string]any{
				"runtime": []any{map[string]any{"auto_detect": true}},
			},
			read: func(d *schema.ResourceData) error {
				r, err := PlaywrightCheckSuiteResourceFromAPIModel(&checkly.PlaywrightCheck{
					EnvironmentVariables: apiVars(),
					AlertSettings:        &checkly.AlertSettings{},
				}, d)
				if err != nil {
					return err
				}
				return r.StoreResourceData(d)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := map[string]any{
				environmentVariableAttributeName: []any{
					map[string]any{"key": "API_KEY", "value": "applied-secret", "secret": true},
					map[string]any{"key": "REGION", "value": "eu-west-1"},
				},
			}
			for k, v := range tt.state {
				state[k] = v
			}
			d := schema.TestResourceDataRaw(t, tt.resource.Schema, state)

			if err := tt.read(d); err != nil {
				t.Fatalf("read failed: %v", err)
			}

			got, err := environmentVariablesFromResourceData(d)
			if err != nil {
				t.Fatal(err)
			}
			want := []checkly.EnvironmentVariable{
				{Key: "API_KEY", Value: "applied-secret", Secret: true},
				{Key: "REGION", Value: "eu-west-1"},
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("environment_variable mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
