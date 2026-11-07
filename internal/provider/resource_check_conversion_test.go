package provider

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/checkly/checkly-go-sdk/v2"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// checkWriteClient serves a stored check and records the per-type updates.
// Calls to any other method panic through the nil embedded Client.
type checkWriteClient struct {
	checkly.Client
	stored         checkly.Check
	err            error
	updatedAPI     *checkly.APICheck
	updatedBrowser *checkly.BrowserCheck
	updatedMulti   *checkly.MultiStepCheck
}

func (c *checkWriteClient) GetCheck(ctx context.Context, id string) (checkly.Check, error) {
	return c.stored, c.err
}

func (c *checkWriteClient) UpdateAPICheck(ctx context.Context, id string, check checkly.APICheck) (*checkly.APICheck, error) {
	c.updatedAPI = &check
	return &check, nil
}

func (c *checkWriteClient) UpdateBrowserCheck(ctx context.Context, id string, check checkly.BrowserCheck) (*checkly.BrowserCheck, error) {
	c.updatedBrowser = &check
	return &check, nil
}

func (c *checkWriteClient) UpdateMultiStepCheck(ctx context.Context, id string, check checkly.MultiStepCheck) (*checkly.MultiStepCheck, error) {
	c.updatedMulti = &check
	return &check, nil
}

// checkPlan returns a plan of checkly_check with the given attributes set.
func checkPlan(t *testing.T, r *CheckResource, attributes map[string]string) tfsdk.Plan {
	t.Helper()
	ctx := context.Background()

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema

	plan := tfsdk.Plan{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
	for name, value := range attributes {
		if diags := plan.SetAttribute(ctx, path.Root(name), value); diags.HasError() {
			t.Fatal(diags)
		}
	}
	return plan
}

func updateCheck(t *testing.T, client *checkWriteClient, attributes map[string]string) diag.Diagnostics {
	t.Helper()
	ctx := context.Background()

	r := &CheckResource{client: client}
	plan := checkPlan(t, r, attributes)
	resp := resource.UpdateResponse{State: tfsdk.State{Schema: plan.Schema, Raw: plan.Raw}}
	r.Update(ctx, resource.UpdateRequest{Plan: plan}, &resp)
	return resp.Diagnostics
}

func TestCheckUpdateKeepsUnmanagedFields(t *testing.T) {
	scriptPath := "tests/main.spec.js"
	dependencies := []checkly.ScriptDependency{{Path: "tests/helper.js", Content: "helper"}}
	intent := &checkly.CheckIntent{Goal: "stay up"}
	aiAutoRepair := true

	cases := map[string]struct {
		plannedType string
		stored      checkly.Check
		// Whether the update carries the script path and dependencies, which
		// an API check doesn't have.
		wantScript bool
		update     func(c *checkWriteClient) (*string, []checkly.ScriptDependency, *checkly.CheckIntent, *bool)
	}{
		"api": {
			plannedType: "API",
			stored:      &checkly.APICheck{Intent: intent, AIAutoRepairEnabled: &aiAutoRepair},
			update: func(c *checkWriteClient) (*string, []checkly.ScriptDependency, *checkly.CheckIntent, *bool) {
				return nil, nil, c.updatedAPI.Intent, c.updatedAPI.AIAutoRepairEnabled
			},
		},
		"browser": {
			plannedType: "BROWSER",
			stored:      &checkly.BrowserCheck{ScriptPath: &scriptPath, Dependencies: dependencies, Intent: intent, AIAutoRepairEnabled: &aiAutoRepair},
			wantScript:  true,
			update: func(c *checkWriteClient) (*string, []checkly.ScriptDependency, *checkly.CheckIntent, *bool) {
				return c.updatedBrowser.ScriptPath, c.updatedBrowser.Dependencies, c.updatedBrowser.Intent, c.updatedBrowser.AIAutoRepairEnabled
			},
		},
		"multi-step": {
			plannedType: "MULTI_STEP",
			stored:      &checkly.MultiStepCheck{ScriptPath: &scriptPath, Dependencies: dependencies, Intent: intent, AIAutoRepairEnabled: &aiAutoRepair},
			wantScript:  true,
			update: func(c *checkWriteClient) (*string, []checkly.ScriptDependency, *checkly.CheckIntent, *bool) {
				return c.updatedMulti.ScriptPath, c.updatedMulti.Dependencies, c.updatedMulti.Intent, c.updatedMulti.AIAutoRepairEnabled
			},
		},
		"an API check changing type dispatches on the planned type": {
			plannedType: "BROWSER",
			stored:      &checkly.APICheck{Intent: intent, AIAutoRepairEnabled: &aiAutoRepair},
			update: func(c *checkWriteClient) (*string, []checkly.ScriptDependency, *checkly.CheckIntent, *bool) {
				return c.updatedBrowser.ScriptPath, c.updatedBrowser.Dependencies, c.updatedBrowser.Intent, c.updatedBrowser.AIAutoRepairEnabled
			},
		},
		"a browser check changing type dispatches on the planned type": {
			plannedType: "MULTI_STEP",
			stored:      &checkly.BrowserCheck{ScriptPath: &scriptPath, Dependencies: dependencies, Intent: intent, AIAutoRepairEnabled: &aiAutoRepair},
			wantScript:  true,
			update: func(c *checkWriteClient) (*string, []checkly.ScriptDependency, *checkly.CheckIntent, *bool) {
				return c.updatedMulti.ScriptPath, c.updatedMulti.Dependencies, c.updatedMulti.Intent, c.updatedMulti.AIAutoRepairEnabled
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			client := &checkWriteClient{stored: c.stored}
			diags := updateCheck(t, client, map[string]string{"id": "abc", "type": c.plannedType, "script": "code"})
			if diags.HasError() {
				t.Fatal(diags)
			}

			gotPath, gotDependencies, gotIntent, gotAIAutoRepair := c.update(client)
			if gotIntent != intent {
				t.Errorf("intent = %+v, want the stored one", gotIntent)
			}
			if gotAIAutoRepair != &aiAutoRepair {
				t.Errorf("aiAutoRepairEnabled = %v, want the stored one", gotAIAutoRepair)
			}
			if c.wantScript && (gotPath != &scriptPath || len(gotDependencies) != 1) {
				t.Errorf("script path = %v, dependencies = %+v, want the stored ones", gotPath, gotDependencies)
			}
		})
	}
}

func TestCheckUpdateSendsNoUpdateWhenTheReadFails(t *testing.T) {
	client := &checkWriteClient{err: errors.New("502 Bad Gateway")}
	diags := updateCheck(t, client, map[string]string{"id": "abc", "type": "API"})

	if !diags.HasError() {
		t.Fatal("expected an error diagnostic")
	}
	if got := diags.Errors()[0]; got.Summary() != "Unable to Update Check" || !strings.Contains(got.Detail(), "502 Bad Gateway") {
		t.Errorf("diagnostic = %q: %q", got.Summary(), got.Detail())
	}
	if client.updatedAPI != nil {
		t.Errorf("sent an update: %+v", client.updatedAPI)
	}
}

func TestCheckUpdateRejectsOtherStoredTypes(t *testing.T) {
	client := &checkWriteClient{stored: &checkly.TCPMonitor{}}
	diags := updateCheck(t, client, map[string]string{"id": "abc", "type": "API"})

	if !diags.HasError() {
		t.Fatal("expected an error diagnostic")
	}
	detail := diags.Errors()[0].Detail()
	if !strings.Contains(detail, `"TCP"`) || !strings.Contains(detail, "checkly_tcp_monitor") {
		t.Errorf("detail = %q, want the stored type and its resource type", detail)
	}
	if client.updatedAPI != nil {
		t.Errorf("sent an update: %+v", client.updatedAPI)
	}
}

// An empty planned script fails before any request, which the nil client of
// the resource would turn into a panic. The omitted, null and apply-time
// values all reach the provider as this plan, as the acceptance test shows.
func TestCheckEmptyScriptSendsNoRequest(t *testing.T) {
	ctx := context.Background()

	for _, checkType := range []string{"BROWSER", "MULTI_STEP"} {
		t.Run(checkType, func(t *testing.T) {
			r := &CheckResource{}
			plan := checkPlan(t, r, map[string]string{"id": "abc", "type": checkType, "script": ""})

			create := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema, Raw: plan.Raw}}
			r.Create(ctx, resource.CreateRequest{Plan: plan}, &create)
			if !create.Diagnostics.HasError() || create.Diagnostics.Errors()[0].Summary() != "Missing Check Script" {
				t.Errorf("create diagnostics = %v", create.Diagnostics)
			}

			update := resource.UpdateResponse{State: tfsdk.State{Schema: plan.Schema, Raw: plan.Raw}}
			r.Update(ctx, resource.UpdateRequest{Plan: plan}, &update)
			if !update.Diagnostics.HasError() || update.Diagnostics.Errors()[0].Summary() != "Missing Check Script" {
				t.Errorf("update diagnostics = %v", update.Diagnostics)
			}
		})
	}
}

func TestValidateCheckScriptAllowsAPIChecks(t *testing.T) {
	var diags diag.Diagnostics
	validateCheckScript(&CheckResourceModel{Type: types.StringValue("API"), Script: types.StringValue("")}, &diags)
	if diags.HasError() {
		t.Error(diags)
	}
}

// emptyCheckModel returns a model as a read of an imported check starts from.
func emptyCheckModel() CheckResourceModel {
	return CheckResourceModel{
		AlertSettings: types.ObjectNull(alertSettingsAttrTypes),
		RetryStrategy: types.ObjectNull(retryStrategyAttrTypes(true)),
		Request:       types.ObjectNull(apiCheckRequestAttrTypes),
	}
}

func TestCheckToModelResolvesAttributesOfOtherTypes(t *testing.T) {
	ctx := context.Background()

	t.Run("an import falls back to the schema defaults", func(t *testing.T) {
		var diags diag.Diagnostics
		model := emptyCheckModel()
		checkToModel(ctx, &checkly.BrowserCheck{ID: "abc", Script: "code"}, &model, &diags)
		if diags.HasError() {
			t.Fatal(diags)
		}

		if model.Type != types.StringValue("BROWSER") {
			t.Errorf("type = %s, want BROWSER", model.Type)
		}
		if model.DegradedResponseTime != types.Int64Value(15000) || model.MaxResponseTime != types.Int64Value(30000) {
			t.Errorf("response times = %s, %s, want the defaults", model.DegradedResponseTime, model.MaxResponseTime)
		}
		if model.SetupSnippetID != types.Int64Value(0) || model.TeardownSnippetID != types.Int64Value(0) {
			t.Errorf("snippet IDs = %s, %s, want 0", model.SetupSnippetID, model.TeardownSnippetID)
		}
		if model.LocalSetupScript != types.StringValue("") || model.LocalTeardownScript != types.StringValue("") {
			t.Errorf("local scripts = %s, %s, want empty", model.LocalSetupScript, model.LocalTeardownScript)
		}
		if model.DoubleCheck != types.BoolValue(false) {
			t.Errorf("double_check = %s, want false", model.DoubleCheck)
		}

		apiModel := emptyCheckModel()
		checkToModel(ctx, &checkly.APICheck{ID: "abc"}, &apiModel, &diags)
		if diags.HasError() {
			t.Fatal(diags)
		}
		if apiModel.Script != types.StringValue("") {
			t.Errorf("script = %s, want empty", apiModel.Script)
		}
	})

	t.Run("a stored value is kept", func(t *testing.T) {
		var diags diag.Diagnostics
		model := emptyCheckModel()
		model.DegradedResponseTime = types.Int64Value(1000)
		model.LocalSetupScript = types.StringValue("setup")
		model.DoubleCheck = types.BoolValue(true)
		checkToModel(ctx, &checkly.MultiStepCheck{ID: "abc", Script: "code"}, &model, &diags)
		if diags.HasError() {
			t.Fatal(diags)
		}

		if model.DegradedResponseTime != types.Int64Value(1000) {
			t.Errorf("degraded_response_time = %s, want 1000", model.DegradedResponseTime)
		}
		if model.LocalSetupScript != types.StringValue("setup") {
			t.Errorf("local_setup_script = %s, want setup", model.LocalSetupScript)
		}
		if model.DoubleCheck != types.BoolValue(true) {
			t.Errorf("double_check = %s, want true", model.DoubleCheck)
		}

		apiModel := emptyCheckModel()
		apiModel.Script = types.StringValue("stored")
		checkToModel(ctx, &checkly.APICheck{ID: "abc"}, &apiModel, &diags)
		if diags.HasError() {
			t.Fatal(diags)
		}
		if apiModel.Script != types.StringValue("stored") {
			t.Errorf("script = %s, want stored", apiModel.Script)
		}

		request := apiCheckRequestToObject(ctx, checkly.Request{URL: "https://example.com"}, types.ObjectNull(apiCheckRequestAttrTypes), &diags)
		if diags.HasError() {
			t.Fatal(diags)
		}
		for name, stored := range map[string]checkly.Check{
			"browser":    &checkly.BrowserCheck{ID: "abc", Script: "code"},
			"multi-step": &checkly.MultiStepCheck{ID: "abc", Script: "code"},
		} {
			model := emptyCheckModel()
			model.SSLCheckDomain = types.StringValue("example.com")
			model.Request = request
			checkToModel(ctx, stored, &model, &diags)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if model.SSLCheckDomain != types.StringValue("example.com") {
				t.Errorf("%s: ssl_check_domain = %s, want example.com", name, model.SSLCheckDomain)
			}
			if !model.Request.Equal(request) {
				t.Errorf("%s: request = %s, want the stored one", name, model.Request)
			}
		}
	})
}

// A configured environment variable stays out of the state of an API check,
// which is what makes ValidateConfig's warning about a change in every plan
// true.
func TestCheckToModelDropsEnvironmentVariablesOfAPIChecks(t *testing.T) {
	ctx := context.Background()

	var diags diag.Diagnostics
	model := emptyCheckModel()
	model.EnvironmentVariable = environmentVariablesToList(ctx, []checkly.EnvironmentVariable{{Key: "A", Value: "1"}}, types.ListNull(types.ObjectType{AttrTypes: environmentVariableAttrTypes}), &diags)
	checkToModel(ctx, &checkly.APICheck{ID: "abc"}, &model, &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if len(model.EnvironmentVariable.Elements()) != 0 {
		t.Errorf("environment_variable = %s, want none", model.EnvironmentVariable)
	}
}

func TestCheckToModelRejectsOtherTypes(t *testing.T) {
	var diags diag.Diagnostics
	model := emptyCheckModel()
	checkToModel(context.Background(), &checkly.HeartbeatMonitor{}, &model, &diags)

	if !diags.HasError() {
		t.Fatal("expected an error diagnostic")
	}
	detail := diags.Errors()[0].Detail()
	if !strings.Contains(detail, `"HEARTBEAT"`) || !strings.Contains(detail, "checkly_heartbeat_monitor") {
		t.Errorf("detail = %q, want the type and its resource type", detail)
	}
}
