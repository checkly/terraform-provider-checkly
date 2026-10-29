package checkly

import (
	"testing"

	"github.com/google/go-cmp/cmp"

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
