package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccEnvVarCheckRequiredFields(t *testing.T) {
	config := `resource "checkly_environment_variable" "test" {}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "key" is required`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "value" is required`),
		},
	})
}

func TestAccEnvVarValueTooLong(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `resource "checkly_environment_variable" "test" {
				key    = "TF_ACC_VALUE_TOO_LONG"
				value  = format("%031745d", 0)
				secret = true
			}`,
			PlanOnly:    true,
			ExpectError: regexp.MustCompile(`at\s+most\s+31744\s+bytes`),
		},
	})
}

func TestAccEnvVarSuccess(t *testing.T) {
	// Environment variable keys are unique per account, so a random suffix
	// avoids collisions between runs.
	key := fmt.Sprintf("TF_ACC_API_URL_%d", acctest.RandInt())
	config := fmt.Sprintf(`resource "checkly_environment_variable" "test" {
		key     = %q
		value   = "https://api.checklyhq.com"
	}`, key)
	accTestCase(t, []resource.TestStep{
		{
			Config: config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_environment_variable.test",
					"key",
					key,
				),
				resource.TestCheckResourceAttr(
					"checkly_environment_variable.test",
					"value",
					"https://api.checklyhq.com",
				),
			),
		},
		{
			ResourceName:      "checkly_environment_variable.test",
			ImportState:       true,
			ImportStateVerify: true,
		},
	})
}

func TestAccSecretEnvVarSuccess(t *testing.T) {
	key := fmt.Sprintf("TF_ACC_SECRET_%d", acctest.RandInt())
	accTestCase(t, []resource.TestStep{
		{
			Config: fmt.Sprintf(`resource "checkly_environment_variable" "test" {
				key     = %q
				value   = "https://api.checklyhq.com"
				secret  = true
			}`, key),
		},
		{
			ResourceName:      "checkly_environment_variable.test",
			ImportState:       true,
			ImportStateVerify: true,
			// The API never returns the value of a secret environment variable.
			ImportStateVerifyIgnore: []string{"value"},
		},
	})
}

// A resource deleted outside Terraform must drop out of state on refresh, so
// that the next plan recreates it, rather than failing the refresh with the
// API's 404.
func TestAccEnvVarDeletedOutsideTerraform(t *testing.T) {
	const name = "checkly_environment_variable.test"
	key := fmt.Sprintf("TF_ACC_DELETED_%d", acctest.RandInt())
	config := fmt.Sprintf(`resource "checkly_environment_variable" "test" {
		key   = %q
		value = "deleted outside Terraform"
	}`, key)
	accTestCase(t, []resource.TestStep{
		{
			Config: config,
			Check:  resource.TestCheckResourceAttr(name, "key", key),
		},
		{
			PreConfig: func() {
				if err := testAccClient().DeleteEnvironmentVariable(context.Background(), key); err != nil {
					t.Fatalf("deleting environment variable %s outside Terraform: %v", key, err)
				}
			},
			RefreshState:       true,
			ExpectNonEmptyPlan: true,
			Check: func(s *terraform.State) error {
				if _, ok := s.RootModule().Resources[name]; ok {
					return fmt.Errorf("%s is still in state after a refresh", name)
				}
				return nil
			},
		},
	})
}

func TestEnvironmentVariableSecretRequiresReplace(t *testing.T) {
	ctx := context.Background()
	var schemaResp fwresource.SchemaResponse
	NewEnvironmentVariableResource().Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	secretAttr := schemaResp.Schema.Attributes["secret"].(schema.BoolAttribute)

	cases := []struct {
		state, plan types.Bool
		want        bool
	}{
		{types.BoolValue(true), types.BoolValue(false), true},
		{types.BoolValue(true), types.BoolUnknown(), true},
		{types.BoolValue(false), types.BoolValue(true), false},
		{types.BoolValue(true), types.BoolValue(true), false},
	}

	for _, c := range cases {
		req := planmodifier.BoolRequest{
			Path:        path.Root("secret"),
			StateValue:  c.state,
			PlanValue:   c.plan,
			ConfigValue: c.plan,
			State:       tfsdk.State{Raw: tftypes.NewValue(tftypes.Object{}, map[string]tftypes.Value{})},
			Plan:        tfsdk.Plan{Raw: tftypes.NewValue(tftypes.Object{}, map[string]tftypes.Value{})},
		}
		resp := planmodifier.BoolResponse{PlanValue: c.plan}
		for _, m := range secretAttr.PlanModifiers {
			m.PlanModifyBool(ctx, req, &resp)
		}
		if resp.RequiresReplace != c.want {
			t.Errorf("secret %s -> %s: RequiresReplace = %t, want %t", c.state, c.plan, resp.RequiresReplace, c.want)
		}
	}
}

func TestEnvironmentVariableAttributeValidators(t *testing.T) {
	var schemaResp fwresource.SchemaResponse
	NewEnvironmentVariableResource().Schema(context.Background(), fwresource.SchemaRequest{}, &schemaResp)

	runAttributeValidationCases(t, schemaResp.Schema, []attributeValidationCase{
		{[]string{"key"}, types.StringValue("API_KEY"), true},
		{[]string{"key"}, types.StringValue(""), false},
	})
}

// An empty value fails the plan, which the API rejects at apply.
func TestAccEnvVarEmptyValue(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `resource "checkly_environment_variable" "test" {
				key   = "TF_ACC_EMPTY"
				value = ""
			}`,
			PlanOnly:    true,
			ExpectError: regexp.MustCompile(`string length must be at least 1`),
		},
	})
}

// Turning a secret environment variable back into a regular one replaces
// it, since the API refuses that update.
func TestAccSecretEnvVarUnsetSecret(t *testing.T) {
	const name = "checkly_environment_variable.test"
	key := fmt.Sprintf("TF_ACC_UNSECRET_%d", acctest.RandInt())
	config := func(secret bool) string {
		return fmt.Sprintf(`resource "checkly_environment_variable" "test" {
			key    = %q
			value  = "a secret value"
			secret = %t
		}`, key, secret)
	}
	accTestCase(t, []resource.TestStep{
		{
			Config: config(true),
		},
		{
			Config: config(false),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(name, plancheck.ResourceActionDestroyBeforeCreate),
				},
			},
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(name, "secret", "false"),
				resource.TestCheckResourceAttr(name, "value", "a secret value"),
			),
		},
		{
			Config: config(false),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
			},
		},
	})
}

func TestEnvironmentVariableModifyPlanSecretLength(t *testing.T) {
	ctx := context.Background()
	r := NewEnvironmentVariableResource().(*EnvironmentVariableResource)
	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)

	model := func(value string, secret, locked bool) *EnvironmentVariableResourceModel {
		return &EnvironmentVariableResourceModel{
			ID:     types.StringValue("K"),
			Key:    types.StringValue("K"),
			Value:  types.StringValue(value),
			Locked: types.BoolValue(locked),
			Secret: types.BoolValue(secret),
		}
	}

	cases := map[string]struct {
		state, plan           *EnvironmentVariableResourceModel
		wantErrors, wantWarns int
	}{
		"create with a long secret":         {plan: model("q7z", true, false)},
		"create with a short secret":        {plan: model("q7", true, false), wantErrors: 1},
		"create with a short regular value": {plan: model("q7", false, false)},
		"no change":                         {state: model("q7", true, false), plan: model("q7", true, false), wantWarns: 1},
		// Every update sends the value, and the API checks it.
		"another change":   {state: model("q7", true, false), plan: model("q7", true, true), wantErrors: 1},
		"a changed value":  {state: model("q7z", true, false), plan: model("q7", true, false), wantErrors: 1},
		"after an import":  {state: &EnvironmentVariableResourceModel{ID: types.StringValue("K"), Key: types.StringValue("K"), Value: types.StringNull(), Locked: types.BoolValue(false), Secret: types.BoolValue(true)}, plan: model("q7", true, false), wantErrors: 1},
		"destroy":          {state: model("q7", true, false)},
		"an unknown value": {plan: &EnvironmentVariableResourceModel{ID: types.StringUnknown(), Key: types.StringValue("K"), Value: types.StringUnknown(), Locked: types.BoolValue(false), Secret: types.BoolValue(true)}},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			state := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}
			if c.state != nil {
				if diags := state.Set(ctx, c.state); diags.HasError() {
					t.Fatal(diags)
				}
			}
			plan := tfsdk.Plan{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}
			if c.plan != nil {
				if diags := plan.Set(ctx, c.plan); diags.HasError() {
					t.Fatal(diags)
				}
			}

			resp := fwresource.ModifyPlanResponse{Plan: plan}
			r.ModifyPlan(ctx, fwresource.ModifyPlanRequest{State: state, Plan: plan}, &resp)

			if got := resp.Diagnostics.ErrorsCount(); got != c.wantErrors {
				t.Errorf("errors = %d, want %d: %v", got, c.wantErrors, resp.Diagnostics)
			}
			if got := resp.Diagnostics.WarningsCount(); got != c.wantWarns {
				t.Errorf("warnings = %d, want %d: %v", got, c.wantWarns, resp.Diagnostics)
			}
		})
	}
}

// A short secret value fails the plan that would create it.
func TestAccSecretEnvVarTooShort(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `resource "checkly_environment_variable" "test" {
				key    = "TF_ACC_SHORT_SECRET"
				value  = "q7"
				secret = true
			}`,
			PlanOnly:    true,
			ExpectError: regexp.MustCompile(`Secret Environment Variable Value Too Short`),
		},
	})
}

// A new short secret value in a check group's environment_variables fails
// the plan.
func TestAccCheckGroupSecretEnvVarTooShort(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `resource "checkly_check_group_v2" "test" {
				name = "short secret"
				environment_variables = [{
					key    = "SHORT"
					value  = "q7"
					secret = true
				}]
			}`,
			PlanOnly:    true,
			ExpectError: regexp.MustCompile(`Secret Environment Variable Value Too Short`),
		},
	})
}
