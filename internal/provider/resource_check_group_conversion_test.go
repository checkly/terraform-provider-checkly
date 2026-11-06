package provider

import (
	"context"
	"testing"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// A group created without alert settings or the other optional settings reads
// as nil pointers. They must convert to the attributes' zero values, and an
// update that follows must send the alert settings as nil and every other
// setting explicitly.
func TestCheckGroupReadThenUpdateWithoutOptionalSettings(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	model := CheckGroupResourceModel{AlertSettings: types.ObjectNull(alertSettingsAttrTypes)}
	checkGroupToModel(ctx, &checkly.GroupV2{ID: 42, Name: "g"}, &model, &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}

	if model.RunParallel != types.BoolValue(false) {
		t.Errorf("run_parallel = %s, want false", model.RunParallel)
	}
	if model.UseGlobalAlertSettings != types.BoolValue(false) {
		t.Errorf("use_global_alert_settings = %s, want false", model.UseGlobalAlertSettings)
	}
	if model.SetupSnippetID != types.Int64Value(0) || model.TeardownSnippetID != types.Int64Value(0) {
		t.Errorf("snippet IDs = %s, %s, want 0", model.SetupSnippetID, model.TeardownSnippetID)
	}
	if model.LocalSetupScript != types.StringValue("") || model.LocalTeardownScript != types.StringValue("") {
		t.Errorf("local scripts = %s, %s, want empty", model.LocalSetupScript, model.LocalTeardownScript)
	}
	if got := model.AlertSettings.Attributes()["escalation_type"]; !got.Equal(types.StringValue("")) {
		t.Errorf("alert_settings.escalation_type = %s, want empty", got)
	}

	group := checkGroupFromModel(ctx, &model, &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}

	if group.AlertSettings != nil {
		t.Errorf("AlertSettings = %+v, want nil", group.AlertSettings)
	}
	if group.RunParallel == nil || *group.RunParallel {
		t.Errorf("RunParallel = %v, want a pointer to false", group.RunParallel)
	}
	if group.UseGlobalAlertSettings == nil || *group.UseGlobalAlertSettings {
		t.Errorf("UseGlobalAlertSettings = %v, want a pointer to false", group.UseGlobalAlertSettings)
	}
	if group.SetupSnippetID != nil || group.TearDownSnippetID != nil {
		t.Errorf("snippet IDs = %v, %v, want nil", group.SetupSnippetID, group.TearDownSnippetID)
	}
	if group.LocalSetupScript != nil || group.LocalTearDownScript != nil {
		t.Errorf("local scripts = %v, %v, want nil", group.LocalSetupScript, group.LocalTearDownScript)
	}
	// A group without a retry strategy keeps overriding its checks with no
	// retries, unlike the FALLBACK that lets each check use its own.
	if group.RetryStrategy == nil || group.RetryStrategy.Type != "NO_RETRIES" {
		t.Errorf("RetryStrategy = %+v, want NO_RETRIES", group.RetryStrategy)
	}
}

func TestCheckGroupFromModelSendsConfiguredSettings(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	group := checkGroupFromModel(ctx, &CheckGroupResourceModel{
		RunParallel:            types.BoolValue(true),
		UseGlobalAlertSettings: types.BoolValue(true),
		SetupSnippetID:         types.Int64Value(1),
		TeardownSnippetID:      types.Int64Value(2),
		LocalSetupScript:       types.StringValue("setup"),
		LocalTeardownScript:    types.StringValue("teardown"),
		AlertSettings:          types.ObjectNull(alertSettingsAttrTypes),
	}, &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}

	if !*group.RunParallel || !*group.UseGlobalAlertSettings {
		t.Errorf("RunParallel = %v, UseGlobalAlertSettings = %v, want true", *group.RunParallel, *group.UseGlobalAlertSettings)
	}
	if *group.SetupSnippetID != 1 || *group.TearDownSnippetID != 2 {
		t.Errorf("snippet IDs = %d, %d, want 1, 2", *group.SetupSnippetID, *group.TearDownSnippetID)
	}
	if *group.LocalSetupScript != "setup" || *group.LocalTearDownScript != "teardown" {
		t.Errorf("local scripts = %q, %q", *group.LocalSetupScript, *group.LocalTearDownScript)
	}
	// Missing alert_settings send the defaults, unlike a stored empty object.
	if group.AlertSettings == nil || group.AlertSettings.EscalationType != checkly.RunBased {
		t.Errorf("AlertSettings = %+v, want the defaults", group.AlertSettings)
	}
}

// The SDK's GroupV2 has no double_check, so it resolves from the model.
func TestCheckGroupToModelDoubleCheck(t *testing.T) {
	cases := map[string]struct {
		prior types.Bool
		want  types.Bool
	}{
		"true is kept":       {prior: types.BoolValue(true), want: types.BoolValue(true)},
		"false is kept":      {prior: types.BoolValue(false), want: types.BoolValue(false)},
		"null becomes false": {prior: types.BoolNull(), want: types.BoolValue(false)},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			var diags diag.Diagnostics
			model := CheckGroupResourceModel{AlertSettings: types.ObjectNull(alertSettingsAttrTypes), DoubleCheck: c.prior}
			checkGroupToModel(context.Background(), &checkly.GroupV2{}, &model, &diags)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if model.DoubleCheck != c.want {
				t.Errorf("double_check = %s, want %s", model.DoubleCheck, c.want)
			}
		})
	}
}

func TestPlanUnconfiguredAlertSettingsAsUnknown(t *testing.T) {
	ctx := context.Background()

	var diags diag.Diagnostics
	storedEmpty := alertSettingsToObject(ctx, checkly.AlertSettings{}, types.ObjectNull(alertSettingsAttrTypes), &diags)
	stored := alertSettingsToObject(ctx, checkly.AlertSettings{
		EscalationType:     checkly.RunBased,
		RunBasedEscalation: checkly.RunBasedEscalation{FailedRunThreshold: 2},
	}, types.ObjectNull(alertSettingsAttrTypes), &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}

	reminders := types.ObjectValueMust(remindersAttrTypes, map[string]attr.Value{
		"amount":   types.Int64Value(1),
		"interval": types.Int64Value(5),
	})
	configured := types.ObjectValueMust(alertSettingsAttrTypes, map[string]attr.Value{
		"escalation_type":                types.StringValue("TIME_BASED"),
		"run_based_escalation":           types.ObjectNull(runBasedEscalationAttrTypes),
		"time_based_escalation":          types.ObjectNull(timeBasedEscalationAttrTypes),
		"reminders":                      reminders,
		"parallel_run_failure_threshold": types.ObjectNull(parallelRunFailureThresholdAttrTypes),
	})

	// The plan as the attribute plan modifiers leave it: the configured values
	// over the stored sub-objects.
	planFrom := func(stored types.Object) types.Object {
		attrs := stored.Attributes()
		attrs["escalation_type"] = configured.Attributes()["escalation_type"]
		attrs["reminders"] = reminders
		return types.ObjectValueMust(alertSettingsAttrTypes, attrs)
	}

	s := schema.Schema{Attributes: map[string]schema.Attribute{
		alertSettingsAttributeName: makeAlertSettingsAttributeSchema(AlertSettingsAttributeSchemaOptions{}),
	}}
	raw := func(t *testing.T, value types.Object) tftypes.Value {
		t.Helper()
		state := tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
		if diags := state.SetAttribute(ctx, path.Root(alertSettingsAttributeName), value); diags.HasError() {
			t.Fatal(diags)
		}
		return state.Raw
	}
	null := tftypes.NewValue(s.Type().TerraformType(ctx), nil)

	cases := map[string]struct {
		state, config, plan tftypes.Value
		wantUnchanged       bool
	}{
		"a stored empty escalation_type replans unconfigured sub-objects": {
			state:  raw(t, storedEmpty),
			config: raw(t, configured),
			plan:   raw(t, planFrom(storedEmpty)),
		},
		"stored settings are left alone": {
			state:         raw(t, stored),
			config:        raw(t, configured),
			plan:          raw(t, planFrom(stored)),
			wantUnchanged: true,
		},
		"without alert_settings in the configuration the stored object stays": {
			state:         raw(t, storedEmpty),
			config:        raw(t, types.ObjectNull(alertSettingsAttrTypes)),
			plan:          raw(t, storedEmpty),
			wantUnchanged: true,
		},
		"a create has no stored settings": {
			state:         null,
			config:        raw(t, configured),
			plan:          raw(t, configured),
			wantUnchanged: true,
		},
		"a destroy has no plan": {
			state:         raw(t, storedEmpty),
			config:        null,
			plan:          null,
			wantUnchanged: true,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			plan := tfsdk.Plan{Schema: s, Raw: c.plan}
			resp := resource.ModifyPlanResponse{Plan: plan}
			planUnconfiguredAlertSettingsAsUnknown(ctx, resource.ModifyPlanRequest{
				Config: tfsdk.Config{Schema: s, Raw: c.config},
				State:  tfsdk.State{Schema: s, Raw: c.state},
				Plan:   plan,
			}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}

			if c.wantUnchanged {
				if !resp.Plan.Raw.Equal(c.plan) {
					t.Errorf("plan = %s, want it unchanged", resp.Plan.Raw)
				}
				return
			}

			var got types.Object
			if diags := resp.Plan.GetAttribute(ctx, path.Root(alertSettingsAttributeName), &got); diags.HasError() {
				t.Fatal(diags)
			}
			attrs := got.Attributes()
			for _, name := range []string{"run_based_escalation", "time_based_escalation", "parallel_run_failure_threshold"} {
				if !attrs[name].IsUnknown() {
					t.Errorf("%s = %s, want unknown", name, attrs[name])
				}
			}
			if !attrs["reminders"].Equal(reminders) {
				t.Errorf("reminders = %s, want the configured value", attrs["reminders"])
			}
			if !attrs["escalation_type"].Equal(types.StringValue("TIME_BASED")) {
				t.Errorf("escalation_type = %s, want TIME_BASED", attrs["escalation_type"])
			}

			// The escalation the plan leaves unknown gets its documented
			// default, not the stored zero value.
			var d diag.Diagnostics
			settings := alertSettingsFromObject(ctx, got, &d)
			if d.HasError() {
				t.Fatal(d)
			}
			if settings.TimeBasedEscalation.MinutesFailingThreshold != 5 {
				t.Errorf("MinutesFailingThreshold = %d, want 5", settings.TimeBasedEscalation.MinutesFailingThreshold)
			}
		})
	}
}
