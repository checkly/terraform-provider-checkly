package provider

import (
	"context"
	"testing"

	"github.com/checkly/checkly-go-sdk/v2"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The API reports no runtime for a check configured with runtime_id = "", so
// each check resource's Read keeps a prior "" instead of refreshing it to
// null, which would plan an update on every run.
func TestCheckToModelRuntimeID(t *testing.T) {
	ctx := context.Background()
	runtime := "2025.04"

	// Each mapping converts a check with the given API runtime and prior
	// runtime_id and returns the resulting runtime_id.
	mappings := map[string]func(apiRuntime *string, prior types.String, diags *diag.Diagnostics) types.String{
		"checkly_api_check": func(apiRuntime *string, prior types.String, diags *diag.Diagnostics) types.String {
			model := APICheckResourceModel{RuntimeID: prior}
			apiCheckToModel(ctx, &checkly.APICheck{ID: "abc", RuntimeID: apiRuntime}, &model, diags)
			return model.RuntimeID
		},
		"checkly_browser_check": func(apiRuntime *string, prior types.String, diags *diag.Diagnostics) types.String {
			model := BrowserCheckResourceModel{RuntimeID: prior}
			browserCheckToModel(ctx, &checkly.BrowserCheck{ID: "abc", Script: "code", RuntimeID: apiRuntime}, &model, diags)
			return model.RuntimeID
		},
		"checkly_multistep_check": func(apiRuntime *string, prior types.String, diags *diag.Diagnostics) types.String {
			model := MultistepCheckResourceModel{RuntimeID: prior}
			multistepCheckToModel(ctx, &checkly.MultiStepCheck{ID: "abc", Script: "code", RuntimeID: apiRuntime}, &model, diags)
			return model.RuntimeID
		},
		"checkly_check": func(apiRuntime *string, prior types.String, diags *diag.Diagnostics) types.String {
			model := emptyCheckModel()
			model.RuntimeID = prior
			checkToModel(ctx, &checkly.APICheck{ID: "abc", RuntimeID: apiRuntime}, &model, diags)
			return model.RuntimeID
		},
	}

	empty := ""
	cases := map[string]struct {
		apiRuntime *string
		prior      types.String
		want       types.String
	}{
		"absent runtime keeps a prior empty string":    {nil, types.StringValue(""), types.StringValue("")},
		"empty runtime keeps a prior empty string":     {&empty, types.StringValue(""), types.StringValue("")},
		"absent runtime without a prior is null":       {nil, types.StringNull(), types.StringNull()},
		"absent runtime with an unknown prior is null": {nil, types.StringUnknown(), types.StringNull()},
		"absent runtime replaces a prior runtime":      {nil, types.StringValue(runtime), types.StringNull()},
		"reported runtime replaces a prior empty one":  {&runtime, types.StringValue(""), types.StringValue(runtime)},
		"reported runtime without a prior":             {&runtime, types.StringNull(), types.StringValue(runtime)},
	}

	for resource, mapping := range mappings {
		for name, tc := range cases {
			t.Run(resource+"/"+name, func(t *testing.T) {
				var diags diag.Diagnostics
				got := mapping(tc.apiRuntime, tc.prior, &diags)
				if diags.HasError() {
					t.Fatal(diags)
				}
				if !got.Equal(tc.want) {
					t.Errorf("runtime_id = %s, want %s", got, tc.want)
				}
			})
		}
	}
}
