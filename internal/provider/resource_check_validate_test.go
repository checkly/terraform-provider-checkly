package provider

import (
	"context"
	"math/big"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// objectWith returns a value of the object type in which only the given
// attributes are set.
func objectWith(objectType tftypes.Object, values map[string]tftypes.Value) tftypes.Value {
	attrs := map[string]tftypes.Value{}
	for name, attrType := range objectType.AttributeTypes {
		attrs[name] = tftypes.NewValue(attrType, nil)
	}
	for name, value := range values {
		attrs[name] = value
	}

	return tftypes.NewValue(objectType, attrs)
}

func TestCheckValidateConfigEnvironmentVariables(t *testing.T) {
	ctx := context.Background()

	var schemaResp resource.SchemaResponse
	NewCheckResource().Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	objectType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	listType := objectType.AttributeTypes[environmentVariableAttributeName].(tftypes.List)

	variables := tftypes.NewValue(listType, []tftypes.Value{
		objectWith(listType.ElementType.(tftypes.Object), map[string]tftypes.Value{
			"key":   tftypes.NewValue(tftypes.String, "A"),
			"value": tftypes.NewValue(tftypes.String, "b"),
		}),
	})

	for checkType, wantWarning := range map[string]bool{"API": true, "BROWSER": false} {
		t.Run(checkType, func(t *testing.T) {
			config := tfsdk.Config{Schema: schemaResp.Schema, Raw: objectWith(objectType, map[string]tftypes.Value{
				"type":                           tftypes.NewValue(tftypes.String, checkType),
				environmentVariableAttributeName: variables,
			})}

			var resp resource.ValidateConfigResponse
			r := NewCheckResource().(resource.ResourceWithValidateConfig)
			r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: config}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			if got := resp.Diagnostics.WarningsCount() > 0; got != wantWarning {
				t.Errorf("warning reported = %v, want %v: %v", got, wantWarning, resp.Diagnostics)
			}
		})
	}
}

// validateCheckConfig runs ValidateConfig on a configuration in which only
// the given attributes are set.
func validateCheckConfig(t *testing.T, values map[string]tftypes.Value) resource.ValidateConfigResponse {
	t.Helper()
	ctx := context.Background()

	var schemaResp resource.SchemaResponse
	NewCheckResource().Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	objectType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	config := tfsdk.Config{Schema: schemaResp.Schema, Raw: objectWith(objectType, values)}

	var resp resource.ValidateConfigResponse
	NewCheckResource().(resource.ResourceWithValidateConfig).ValidateConfig(ctx, resource.ValidateConfigRequest{Config: config}, &resp)
	return resp
}

func TestCheckValidateConfigIgnoredAttributes(t *testing.T) {
	ctx := context.Background()

	var schemaResp resource.SchemaResponse
	NewCheckResource().Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	objectType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)

	// Default values, or a valid value for the snippet IDs, which have no
	// default. Each warns like any other value.
	values := map[string]tftypes.Value{
		"script":                 tftypes.NewValue(tftypes.String, ""),
		"request":                objectWith(objectType.AttributeTypes["request"].(tftypes.Object), nil),
		"frequency_offset":       tftypes.NewValue(tftypes.Number, big.NewFloat(0)),
		"degraded_response_time": tftypes.NewValue(tftypes.Number, big.NewFloat(15000)),
		"max_response_time":      tftypes.NewValue(tftypes.Number, big.NewFloat(30000)),
		"setup_snippet_id":       tftypes.NewValue(tftypes.Number, big.NewFloat(1)),
		"teardown_snippet_id":    tftypes.NewValue(tftypes.Number, big.NewFloat(1)),
		"local_setup_script":     tftypes.NewValue(tftypes.String, ""),
		"local_teardown_script":  tftypes.NewValue(tftypes.String, ""),
		"ssl_check_domain":       tftypes.NewValue(tftypes.String, ""),
	}
	apiOnly := []string{"request", "frequency_offset", "degraded_response_time", "max_response_time",
		"setup_snippet_id", "teardown_snippet_id", "local_setup_script", "local_teardown_script"}
	ignored := map[string][]string{
		"API":        {"script", "ssl_check_domain"},
		"BROWSER":    apiOnly,
		"MULTI_STEP": append(slices.Clone(apiOnly), "ssl_check_domain"),
	}

	for checkType, ignoredNames := range ignored {
		for name, value := range values {
			t.Run(checkType+"/"+name, func(t *testing.T) {
				resp := validateCheckConfig(t, map[string]tftypes.Value{
					"type": tftypes.NewValue(tftypes.String, checkType),
					name:   value,
				})
				if resp.Diagnostics.HasError() {
					t.Fatal(resp.Diagnostics)
				}

				want := 0
				if slices.Contains(ignoredNames, name) {
					want = 1
				}
				warnings := resp.Diagnostics.Warnings()
				if len(warnings) != want {
					t.Fatalf("got %d warnings, want %d: %v", len(warnings), want, warnings)
				}
				if want == 1 {
					if got, ok := warnings[0].(interface{ Path() path.Path }); !ok || !got.Path().Equal(path.Root(name)) {
						t.Errorf("warning is not on %q: %v", name, warnings[0])
					}
				}
			})
		}
	}
}

func TestCheckValidateConfigIgnoredAttributesSkipsUnknownValues(t *testing.T) {
	cases := map[string]map[string]tftypes.Value{
		"unknown value": {
			"type":              tftypes.NewValue(tftypes.String, "BROWSER"),
			"max_response_time": tftypes.NewValue(tftypes.Number, tftypes.UnknownValue),
		},
		"unknown type": {
			"type":              tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
			"max_response_time": tftypes.NewValue(tftypes.Number, big.NewFloat(1000)),
		},
	}

	for name, values := range cases {
		t.Run(name, func(t *testing.T) {
			if resp := validateCheckConfig(t, values); len(resp.Diagnostics) > 0 {
				t.Error(resp.Diagnostics)
			}
		})
	}
}

func TestCheckValidateConfigSSLCheckDomainOnAPICheck(t *testing.T) {
	resp := validateCheckConfig(t, map[string]tftypes.Value{
		"type":             tftypes.NewValue(tftypes.String, "API"),
		"ssl_check_domain": tftypes.NewValue(tftypes.String, "example.com"),
	})
	if resp.Diagnostics.ErrorsCount() != 1 || resp.Diagnostics.WarningsCount() != 0 {
		t.Errorf("want only the existing error: %v", resp.Diagnostics)
	}
}

func TestCheckModifyPlanEmptyScript(t *testing.T) {
	ctx := context.Background()

	str := func(v any) tftypes.Value { return tftypes.NewValue(tftypes.String, v) }
	cases := map[string]struct {
		checkType     string
		plannedScript tftypes.Value
		configScript  tftypes.Value
		destroy       bool
		wantError     bool
	}{
		"browser":    {checkType: "BROWSER", plannedScript: str(""), configScript: str(nil), wantError: true},
		"multi-step": {checkType: "MULTI_STEP", plannedScript: str(""), configScript: str(""), wantError: true},
		"script":     {checkType: "BROWSER", plannedScript: str("x"), configScript: str("x")},
		// With ignore_changes = [script], the plan holds the stored script.
		"ignored":        {checkType: "BROWSER", plannedScript: str("x"), configScript: str("")},
		"known at apply": {checkType: "BROWSER", plannedScript: str(tftypes.UnknownValue), configScript: str(tftypes.UnknownValue)},
		"api":            {checkType: "API", plannedScript: str(""), configScript: str(nil)},
		"destroy":        {destroy: true},
	}

	r := &CheckResource{}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema
	objectType := s.Type().TerraformType(ctx).(tftypes.Object)

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			plan := tfsdk.Plan{Schema: s, Raw: tftypes.NewValue(objectType, nil)}
			config := tfsdk.Config{Schema: s, Raw: tftypes.NewValue(objectType, nil)}
			if !c.destroy {
				plan.Raw = objectWith(objectType, map[string]tftypes.Value{"type": str(c.checkType), "script": c.plannedScript})
				config.Raw = objectWith(objectType, map[string]tftypes.Value{"type": str(c.checkType), "script": c.configScript})
			}

			resp := resource.ModifyPlanResponse{Plan: plan}
			r.ModifyPlan(ctx, resource.ModifyPlanRequest{Plan: plan, Config: config}, &resp)
			if got := resp.Diagnostics.HasError(); got != c.wantError {
				t.Fatalf("error reported = %v, want %v: %v", got, c.wantError, resp.Diagnostics)
			}
			if c.wantError && resp.Diagnostics.Errors()[0].Summary() != "Missing Check Script" {
				t.Errorf("diagnostics = %v", resp.Diagnostics)
			}
		})
	}
}
