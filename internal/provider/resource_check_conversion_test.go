package provider

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/checkly/checkly-go-sdk/v2"
	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	updatedAPI     *checkly.APICheck
	updatedBrowser *checkly.BrowserCheck
	updatedMulti   *checkly.MultiStepCheck
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
func checkPlan(t *testing.T, r *CheckResource, attributes map[string]any) tfsdk.Plan {
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

func updateCheck(t *testing.T, client *checkWriteClient, attributes map[string]any) diag.Diagnostics {
	t.Helper()
	ctx := context.Background()

	r := &CheckResource{client: client}
	plan := checkPlan(t, r, attributes)
	resp := resource.UpdateResponse{State: tfsdk.State{Schema: plan.Schema, Raw: plan.Raw}}
	r.Update(ctx, resource.UpdateRequest{Plan: plan}, &resp)
	return resp.Diagnostics
}

func TestCheckUpdateSendsThePlan(t *testing.T) {
	cases := map[string]struct {
		plannedType string
		update      func(c *checkWriteClient) (*string, []checkly.ScriptDependency, *checkly.CheckIntent, *bool)
	}{
		"api": {
			plannedType: "API",
			update: func(c *checkWriteClient) (*string, []checkly.ScriptDependency, *checkly.CheckIntent, *bool) {
				return nil, nil, c.updatedAPI.Intent, c.updatedAPI.AIAutoRepairEnabled
			},
		},
		"browser": {
			plannedType: "BROWSER",
			update: func(c *checkWriteClient) (*string, []checkly.ScriptDependency, *checkly.CheckIntent, *bool) {
				return c.updatedBrowser.ScriptPath, c.updatedBrowser.Dependencies, c.updatedBrowser.Intent, c.updatedBrowser.AIAutoRepairEnabled
			},
		},
		"multi-step": {
			plannedType: "MULTI_STEP",
			update: func(c *checkWriteClient) (*string, []checkly.ScriptDependency, *checkly.CheckIntent, *bool) {
				return c.updatedMulti.ScriptPath, c.updatedMulti.Dependencies, c.updatedMulti.Intent, c.updatedMulti.AIAutoRepairEnabled
			},
		},
	}

	plannedIntent := types.ObjectValueMust(checkIntentAttrTypes, map[string]attr.Value{
		"goal":              types.StringValue("planned"),
		"required_outcomes": types.ListNull(types.StringType),
		"must_preserve":     types.ListNull(types.StringType),
	})

	for name, c := range cases {
		// The intent and the automatic repair override come from the plan,
		// whether it sets them or leaves them out.
		for _, planned := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/planned=%v", name, planned), func(t *testing.T) {
				attributes := map[string]any{"id": "abc", "type": c.plannedType, "script": "code"}
				if planned {
					attributes[checkIntentAttributeName] = plannedIntent
					attributes[aiAutoRepairEnabledAttributeName] = types.BoolValue(false)
				}
				client := &checkWriteClient{}
				diags := updateCheck(t, client, attributes)
				if diags.HasError() {
					t.Fatal(diags)
				}

				gotPath, gotDependencies, gotIntent, gotAIAutoRepair := c.update(client)
				if planned {
					if gotIntent == nil || gotIntent.Goal != "planned" {
						t.Errorf("intent = %+v, want the planned one", gotIntent)
					}
					if gotAIAutoRepair == nil || *gotAIAutoRepair {
						t.Errorf("aiAutoRepairEnabled = %v, want the planned false", gotAIAutoRepair)
					}
				} else if gotIntent != nil || gotAIAutoRepair != nil {
					t.Errorf("intent = %+v, aiAutoRepairEnabled = %v, want both cleared", gotIntent, gotAIAutoRepair)
				}
				// checkly_check has no attributes for the script path and
				// dependencies, so an update clears them.
				if gotPath != nil || len(gotDependencies) != 0 {
					t.Errorf("script path = %v, dependencies = %+v, want both cleared", gotPath, gotDependencies)
				}
			})
		}
	}
}

// should_fail only applies to API checks. A browser or multi-step plan that
// still holds true, such as one with a stored true that ValidateConfig does
// not see, sends false.
func TestCheckUpdateSendsShouldFailOnlyForAPIChecks(t *testing.T) {
	cases := map[string]struct {
		plannedType string
		want        bool
		sent        func(c *checkWriteClient) bool
	}{
		"api":        {plannedType: "API", want: true, sent: func(c *checkWriteClient) bool { return c.updatedAPI.ShouldFail }},
		"browser":    {plannedType: "BROWSER", want: false, sent: func(c *checkWriteClient) bool { return c.updatedBrowser.ShouldFail }},
		"multi-step": {plannedType: "MULTI_STEP", want: false, sent: func(c *checkWriteClient) bool { return c.updatedMulti.ShouldFail }},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			client := &checkWriteClient{}
			diags := updateCheck(t, client, map[string]any{"id": "abc", "type": c.plannedType, "script": "code", shouldFailAttributeName: true})
			if diags.HasError() {
				t.Fatal(diags)
			}
			if got := c.sent(client); got != c.want {
				t.Errorf("shouldFail = %v, want %v", got, c.want)
			}
		})
	}
}

// A read of a browser or multi-step check, which covers imports, warns when
// the check has should_fail set to true in Checkly, which these resources
// have no attribute for.
func TestCheckToModelWarnsAboutShouldFail(t *testing.T) {
	ctx := context.Background()

	mappings := map[string]func(shouldFail bool, diags *diag.Diagnostics){
		"checkly_browser_check": func(shouldFail bool, diags *diag.Diagnostics) {
			var model BrowserCheckResourceModel
			browserCheckToModel(ctx, &checkly.BrowserCheck{ID: "abc", Script: "code", ShouldFail: shouldFail}, &model, diags)
		},
		"checkly_multistep_check": func(shouldFail bool, diags *diag.Diagnostics) {
			var model MultistepCheckResourceModel
			multistepCheckToModel(ctx, &checkly.MultiStepCheck{ID: "abc", Script: "code", ShouldFail: shouldFail}, &model, diags)
		},
	}

	for name, mapping := range mappings {
		for _, shouldFail := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/should_fail=%v", name, shouldFail), func(t *testing.T) {
				var diags diag.Diagnostics
				mapping(shouldFail, &diags)
				if diags.HasError() {
					t.Fatal(diags)
				}

				warnings := diags.Warnings()
				if shouldFail {
					if len(warnings) != 1 || warnings[0].Summary() != "should_fail Is Not Supported" ||
						!strings.Contains(warnings[0].Detail(), name) {
						t.Errorf("want one should_fail warning naming %s: %v", name, diags)
					}
				} else if len(warnings) != 0 {
					t.Errorf("want no warnings: %v", warnings)
				}
			})
		}
	}
}

// An empty planned script fails before any request, which the nil client of
// the resource would turn into a panic. ModifyPlan reports an omitted or null
// script at plan time already; a script known only at apply time reaches
// Create and Update as this plan, as the acceptance test shows.
func TestCheckEmptyScriptSendsNoRequest(t *testing.T) {
	ctx := context.Background()

	for _, checkType := range []string{"BROWSER", "MULTI_STEP"} {
		t.Run(checkType, func(t *testing.T) {
			r := &CheckResource{}
			plan := checkPlan(t, r, map[string]any{"id": "abc", "type": checkType, "script": ""})

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
		if !model.SetupSnippetID.IsNull() || !model.TeardownSnippetID.IsNull() {
			t.Errorf("snippet IDs = %s, %s, want null", model.SetupSnippetID, model.TeardownSnippetID)
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

		request := apiCheckRequestToObject(ctx, checkly.Request{URL: "https://example.com"}, &diags)
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
			model.FrequencyOffset = types.Int64Value(10)
			checkToModel(ctx, stored, &model, &diags)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if model.SSLCheckDomain != types.StringValue("example.com") {
				t.Errorf("%s: ssl_check_domain = %s, want example.com", name, model.SSLCheckDomain)
			}
			if model.FrequencyOffset != types.Int64Value(10) {
				t.Errorf("%s: frequency_offset = %s, want 10", name, model.FrequencyOffset)
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
	model.EnvironmentVariables = environmentVariablesToList(ctx, []checkly.EnvironmentVariable{{Key: "A", Value: "1"}}, types.ListNull(types.ObjectType{AttrTypes: environmentVariableAttrTypes}), &diags)
	checkToModel(ctx, &checkly.APICheck{ID: "abc"}, &model, &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}
	// The prior value is a known list, so the result is an empty list, not
	// null.
	if model.EnvironmentVariables.IsNull() || len(model.EnvironmentVariables.Elements()) != 0 {
		t.Errorf("environment_variables = %s, want an empty list", model.EnvironmentVariables)
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

// storedAlertSettings returns a known alert_settings value that differs from
// the one an empty API representation converts to.
func storedAlertSettings(t *testing.T) types.Object {
	t.Helper()

	var diags diag.Diagnostics
	stored := alertSettingsToObject(context.Background(), checkly.AlertSettings{
		EscalationType:     checkly.EscalationTypeRunBased,
		RunBasedEscalation: checkly.RunBasedEscalation{FailedRunThreshold: 3},
	}, types.ObjectNull(alertSettingsAttrTypes), &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}
	return stored
}

// An API representation without alert settings leaves the planned or stored
// value alone instead of replacing it with empty settings.
func TestCheckAlertSettingsMissingFromTheResponse(t *testing.T) {
	ctx := context.Background()

	t.Run("resolve: an unknown plan resolves to null", func(t *testing.T) {
		var diags diag.Diagnostics
		model := emptyCheckModel()
		model.AlertSettings = types.ObjectUnknown(alertSettingsAttrTypes)
		checkResolveComputedValues(ctx, &checkly.APICheck{ID: "abc"}, &model, &diags)
		if diags.HasError() {
			t.Fatal(diags)
		}
		if !model.AlertSettings.IsNull() {
			t.Errorf("alert_settings = %s, want null", model.AlertSettings)
		}
	})

	t.Run("resolve: a known plan is kept", func(t *testing.T) {
		var diags diag.Diagnostics
		model := emptyCheckModel()
		model.AlertSettings = storedAlertSettings(t)
		checkResolveComputedValues(ctx, &checkly.APICheck{ID: "abc"}, &model, &diags)
		if diags.HasError() {
			t.Fatal(diags)
		}
		if !model.AlertSettings.Equal(storedAlertSettings(t)) {
			t.Errorf("alert_settings = %s, want the planned value", model.AlertSettings)
		}
	})

	t.Run("resolve: settings in the response are converted", func(t *testing.T) {
		var diags diag.Diagnostics
		settings := checkly.AlertSettings{
			EscalationType:      checkly.EscalationTypeTimeBased,
			TimeBasedEscalation: checkly.TimeBasedEscalation{MinutesFailingThreshold: 10},
		}
		model := emptyCheckModel()
		model.AlertSettings = types.ObjectUnknown(alertSettingsAttrTypes)
		checkResolveComputedValues(ctx, &checkly.APICheck{ID: "abc", AlertSettings: &settings}, &model, &diags)
		if diags.HasError() {
			t.Fatal(diags)
		}
		want := alertSettingsToObject(ctx, settings, types.ObjectUnknown(alertSettingsAttrTypes), &diags)
		if diags.HasError() {
			t.Fatal(diags)
		}
		if !model.AlertSettings.Equal(want) {
			t.Errorf("alert_settings = %s, want %s", model.AlertSettings, want)
		}
	})

	t.Run("read: a stored value is kept", func(t *testing.T) {
		var diags diag.Diagnostics
		model := emptyCheckModel()
		model.AlertSettings = storedAlertSettings(t)
		checkToModel(ctx, &checkly.APICheck{ID: "abc"}, &model, &diags)
		if diags.HasError() {
			t.Fatal(diags)
		}
		if !model.AlertSettings.Equal(storedAlertSettings(t)) {
			t.Errorf("alert_settings = %s, want the stored value", model.AlertSettings)
		}
	})

	t.Run("read: an unknown prior value resolves to null", func(t *testing.T) {
		var diags diag.Diagnostics
		model := emptyCheckModel()
		model.AlertSettings = types.ObjectUnknown(alertSettingsAttrTypes)
		checkToModel(ctx, &checkly.APICheck{ID: "abc"}, &model, &diags)
		if diags.HasError() {
			t.Fatal(diags)
		}
		if !model.AlertSettings.IsNull() {
			t.Errorf("alert_settings = %s, want null", model.AlertSettings)
		}
	})
}
