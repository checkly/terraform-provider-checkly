package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// scriptedCheckResources returns the per-type check resources whose script
// must not be empty.
func scriptedCheckResources() map[string]fwresource.ResourceWithModifyPlan {
	return map[string]fwresource.ResourceWithModifyPlan{
		"checkly_browser_check":   &BrowserCheckResource{},
		"checkly_multistep_check": &MultistepCheckResource{},
	}
}

func TestScriptedCheckModifyPlanEmptyScript(t *testing.T) {
	ctx := context.Background()

	str := func(v any) tftypes.Value { return tftypes.NewValue(tftypes.String, v) }
	cases := map[string]struct {
		plannedScript tftypes.Value
		configScript  tftypes.Value
		destroy       bool
		wantError     bool
	}{
		"empty":  {plannedScript: str(""), configScript: str(""), wantError: true},
		"script": {plannedScript: str("x"), configScript: str("x")},
		// With ignore_changes = [script], the plan holds the stored script.
		"ignored":        {plannedScript: str("x"), configScript: str("")},
		"known at apply": {plannedScript: str(tftypes.UnknownValue), configScript: str(tftypes.UnknownValue)},
		"destroy":        {destroy: true},
	}

	for resourceName, r := range scriptedCheckResources() {
		var schemaResp fwresource.SchemaResponse
		r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
		s := schemaResp.Schema
		objectType := s.Type().TerraformType(ctx).(tftypes.Object)

		for name, c := range cases {
			t.Run(resourceName+"/"+name, func(t *testing.T) {
				plan := tfsdk.Plan{Schema: s, Raw: tftypes.NewValue(objectType, nil)}
				config := tfsdk.Config{Schema: s, Raw: tftypes.NewValue(objectType, nil)}
				if !c.destroy {
					plan.Raw = objectWith(objectType, map[string]tftypes.Value{"script": c.plannedScript})
					config.Raw = objectWith(objectType, map[string]tftypes.Value{"script": c.configScript})
				}

				resp := fwresource.ModifyPlanResponse{Plan: plan}
				r.ModifyPlan(ctx, fwresource.ModifyPlanRequest{Plan: plan, Config: config}, &resp)
				if got := resp.Diagnostics.HasError(); got != c.wantError {
					t.Fatalf("error reported = %v, want %v: %v", got, c.wantError, resp.Diagnostics)
				}
				if c.wantError && resp.Diagnostics.Errors()[0].Summary() != "Missing Check Script" {
					t.Errorf("diagnostics = %v", resp.Diagnostics)
				}
			})
		}
	}
}

// An empty planned script fails before any request, which the nil client of
// the resource would turn into a panic.
func TestScriptedCheckEmptyScriptSendsNoRequest(t *testing.T) {
	ctx := context.Background()

	for resourceName, r := range scriptedCheckResources() {
		t.Run(resourceName, func(t *testing.T) {
			var schemaResp fwresource.SchemaResponse
			r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
			s := schemaResp.Schema
			plan := tfsdk.Plan{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
			for name, value := range map[string]any{"id": "abc", "script": ""} {
				if diags := plan.SetAttribute(ctx, path.Root(name), value); diags.HasError() {
					t.Fatal(diags)
				}
			}

			create := fwresource.CreateResponse{State: tfsdk.State{Schema: s, Raw: plan.Raw}}
			r.Create(ctx, fwresource.CreateRequest{Plan: plan}, &create)
			if !create.Diagnostics.HasError() || create.Diagnostics.Errors()[0].Summary() != "Missing Check Script" {
				t.Errorf("create diagnostics = %v", create.Diagnostics)
			}

			update := fwresource.UpdateResponse{State: tfsdk.State{Schema: s, Raw: plan.Raw}}
			r.Update(ctx, fwresource.UpdateRequest{Plan: plan}, &update)
			if !update.Diagnostics.HasError() || update.Diagnostics.Errors()[0].Summary() != "Missing Check Script" {
				t.Errorf("update diagnostics = %v", update.Diagnostics)
			}
		})
	}
}

// TestAccScriptedCheckEmptyScript covers an empty script on create and on
// update, set directly (reported in the plan) and known only at apply. The API
// would accept the update and clear the stored script.
func TestAccScriptedCheckEmptyScript(t *testing.T) {
	// Replacing the terraform_data resource makes its output unknown in the
	// plan of an update as well.
	knownOnlyAtApply := func(revision int) string {
		return fmt.Sprintf(`
			resource "terraform_data" "empty" {
				input            = ""
				triggers_replace = %d
			}
		`, revision)
	}

	cases := map[string]struct {
		extra  func(revision int) string
		script string
	}{
		"empty": {extra: func(int) string { return "" }, script: `""`},
		"known only at apply": {
			extra:  knownOnlyAtApply,
			script: `terraform_data.empty.output`,
		},
	}

	for _, resourceType := range []string{"checkly_browser_check", "checkly_multistep_check"} {
		for name, c := range cases {
			t.Run(resourceType+"/"+name, func(t *testing.T) {
				config := func(revision int, script string) string {
					return c.extra(revision) + fmt.Sprintf(`
						resource %q "test" {
							name      = "tf-acc-empty-script"
							activated = false
							frequency = 10
							locations = ["eu-central-1"]
							script    = %s
						}
					`, resourceType, script)
				}
				missingScript := regexp.MustCompile(`Missing\s+Check\s+Script`)

				accTestCase(t, []resource.TestStep{
					{
						Config:      config(1, c.script),
						ExpectError: missingScript,
					},
					{
						Config: config(1, `"console.log('hi')"`),
					},
					{
						Config:      config(2, c.script),
						ExpectError: missingScript,
					},
				})
			})
		}
	}
}
