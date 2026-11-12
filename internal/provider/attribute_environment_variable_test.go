package provider

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

func environmentVariableTestList(t *testing.T, evs []EnvironmentVariableModel) types.List {
	t.Helper()

	list, diags := types.ListValueFrom(context.Background(), types.ObjectType{AttrTypes: environmentVariableAttrTypes}, evs)
	if diags.HasError() {
		t.Fatalf("building list: %v", diags)
	}

	return list
}

func TestEnvironmentVariablesToListKeepsSecretValues(t *testing.T) {
	ctx := context.Background()

	secret := func(key, value string) EnvironmentVariableModel {
		return EnvironmentVariableModel{
			Key:    types.StringValue(key),
			Value:  types.StringValue(value),
			Locked: types.BoolValue(false),
			Secret: types.BoolValue(true),
		}
	}
	plain := func(key, value string) EnvironmentVariableModel {
		return EnvironmentVariableModel{
			Key:    types.StringValue(key),
			Value:  types.StringValue(value),
			Locked: types.BoolValue(false),
			Secret: types.BoolValue(false),
		}
	}

	tests := []struct {
		name  string
		prior types.List
		// The API returns secrets with an empty value.
		evs  []checkly.EnvironmentVariable
		want []EnvironmentVariableModel
	}{
		{
			name:  "secret values are restored from the prior value",
			prior: environmentVariableTestList(t, []EnvironmentVariableModel{secret("API_KEY", "applied-secret"), plain("REGION", "eu-west-1")}),
			evs: []checkly.EnvironmentVariable{
				{Key: "API_KEY", Value: "", Secret: true},
				{Key: "REGION", Value: "eu-west-1"},
			},
			want: []EnvironmentVariableModel{secret("API_KEY", "applied-secret"), plain("REGION", "eu-west-1")},
		},
		{
			name:  "non-secret API values are kept as returned",
			prior: environmentVariableTestList(t, []EnvironmentVariableModel{plain("REGION", "us-east-1")}),
			evs:   []checkly.EnvironmentVariable{{Key: "REGION", Value: "eu-west-1"}},
			want:  []EnvironmentVariableModel{plain("REGION", "eu-west-1")},
		},
		{
			name:  "secrets without a prior value are left empty",
			prior: environmentVariableTestList(t, []EnvironmentVariableModel{secret("OTHER", "other-secret")}),
			evs:   []checkly.EnvironmentVariable{{Key: "API_KEY", Value: "", Secret: true}},
			want:  []EnvironmentVariableModel{secret("API_KEY", "")},
		},
		{
			name:  "a null prior returns the API values as is",
			prior: types.ListNull(types.ObjectType{AttrTypes: environmentVariableAttrTypes}),
			evs:   []checkly.EnvironmentVariable{{Key: "API_KEY", Value: "", Secret: true}},
			want:  []EnvironmentVariableModel{secret("API_KEY", "")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var diags diag.Diagnostics
			list := environmentVariablesToList(ctx, tt.evs, tt.prior, &diags)
			if diags.HasError() {
				t.Fatalf("environmentVariablesToList: %v", diags)
			}

			var got []EnvironmentVariableModel
			if d := list.ElementsAs(ctx, &got, false); d.HasError() {
				t.Fatalf("reading list: %v", d)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("environmentVariablesToList() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// The API reports no variables as an empty array, which converts to an empty
// list whatever the prior value.
func TestEnvironmentVariablesToListEmpty(t *testing.T) {
	ctx := context.Background()
	elemType := types.ObjectType{AttrTypes: environmentVariableAttrTypes}
	empty := types.ListValueMust(elemType, nil)

	for name, c := range map[string]struct {
		prior types.List
		want  types.List
	}{
		"prior null":    {prior: types.ListNull(elemType), want: empty},
		"prior unknown": {prior: types.ListUnknown(elemType), want: empty},
		"prior empty":   {prior: empty, want: empty},
		"prior non-empty": {
			prior: environmentVariablesList(t, "A", "1"),
			want:  empty,
		},
	} {
		t.Run(name, func(t *testing.T) {
			var diags diag.Diagnostics
			got := environmentVariablesToList(ctx, nil, c.prior, &diags)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if !got.Equal(c.want) {
				t.Errorf("got %s, want %s", got, c.want)
			}
		})
	}
}
