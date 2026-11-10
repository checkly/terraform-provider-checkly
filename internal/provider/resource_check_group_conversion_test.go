package provider

import (
	"context"
	"testing"

	"github.com/checkly/checkly-go-sdk/v2"
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
	if !model.SetupSnippetID.IsNull() || !model.TeardownSnippetID.IsNull() {
		t.Errorf("snippet IDs = %s, %s, want null", model.SetupSnippetID, model.TeardownSnippetID)
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
	if group.AlertSettings == nil || group.AlertSettings.EscalationType != checkly.EscalationTypeRunBased {
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
		EscalationType:     checkly.EscalationTypeRunBased,
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
		alertSettingsAttributeName: makeAlertSettingsAttributeSchema(AlertSettingsAttributeSchemaOptions{Subject: subjectCheck}),
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

// checkGroupModelForMove returns the model of a checkly_check_group that
// configures nothing optional, as checkly_check_group stores it: zero snippet
// IDs and scripts, no alert settings, and no retries.
func checkGroupModelForMove() CheckGroupResourceModel {
	stringSet := types.SetNull(types.StringType)
	return CheckGroupResourceModel{
		ID:                        types.StringValue("42"),
		Name:                      types.StringValue("g"),
		Concurrency:               types.Int64Value(1),
		Activated:                 types.BoolValue(true),
		Muted:                     types.BoolValue(false),
		RunParallel:               types.BoolValue(false),
		Locations:                 stringSet,
		PrivateLocations:          stringSet,
		Tags:                      stringSet,
		EnvironmentVariables:      types.ListNull(types.ObjectType{AttrTypes: environmentVariableAttrTypes}),
		SetupSnippetID:            types.Int64Null(),
		TeardownSnippetID:         types.Int64Null(),
		LocalSetupScript:          types.StringValue(""),
		LocalTeardownScript:       types.StringValue(""),
		RuntimeID:                 types.StringNull(),
		AlertChannelSubscriptions: types.SetNull(types.ObjectType{AttrTypes: alertChannelSubscriptionAttrTypes}),
		AlertSettings:             types.ObjectNull(alertSettingsAttrTypes),
		UseGlobalAlertSettings:    types.BoolValue(true),
		DoubleCheck:               types.BoolValue(false),
		APICheckDefaults:          apiCheckDefaultsDefaultObject(),
		RetryStrategy:             noRetriesObject(true),
	}
}

func TestCheckGroupV2ModelFromCheckGroupModel(t *testing.T) {
	ctx := context.Background()

	stringSet := func(values ...string) types.Set {
		elements := make([]attr.Value, 0, len(values))
		for _, value := range values {
			elements = append(elements, types.StringValue(value))
		}
		return types.SetValueMust(types.StringType, elements)
	}
	enabled := func(attrTypes map[string]attr.Type, values map[string]attr.Value) types.Object {
		values["enabled"] = types.BoolValue(true)
		return types.ObjectValueMust(attrTypes, values)
	}
	var diags diag.Diagnostics
	alertSettings := alertSettingsToObject(ctx, checkly.AlertSettings{
		EscalationType:     checkly.EscalationTypeRunBased,
		RunBasedEscalation: checkly.RunBasedEscalation{FailedRunThreshold: 2},
	}, types.ObjectNull(alertSettingsAttrTypes), &diags)
	// checkGroupToModel stores a group without alert settings as zero values
	// with an empty escalation type.
	zeroAlertSettings := alertSettingsToObject(ctx, checkly.AlertSettings{}, types.ObjectNull(alertSettingsAttrTypes), &diags)
	fallback := retryStrategyToObject(ctx, &checkly.RetryStrategy{Type: checkly.RetryStrategyTypeFallback}, types.ObjectNull(retryStrategyAttrTypes(true)), &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}
	secretVariables := environmentVariableTestList(t, []EnvironmentVariableModel{{
		Key:    types.StringValue("TOKEN"),
		Value:  types.StringValue("s3cret"),
		Locked: types.BoolValue(false),
		Secret: types.BoolValue(true),
	}})

	cases := map[string]struct {
		edit  func(source *CheckGroupResourceModel)
		check func(t *testing.T, target CheckGroupV2ResourceModel)
	}{
		"a group without optional settings": {
			check: func(t *testing.T, target CheckGroupV2ResourceModel) {
				requireValue(t, "id", target.ID, types.StringValue("42"))
				requireValue(t, "tags", target.Tags, types.SetNull(types.StringType))
				requireValue(t, "enforce_locations", target.EnforceLocations, checkGroupV2DisabledEnforcementObject(checkGroupV2EnforceLocationsAttrTypes))
				requireValue(t, "enforce_alert_settings", target.EnforceAlertSettings, enabled(checkGroupV2EnforceAlertSettingsAttrTypes, map[string]attr.Value{
					alertSettingsAttributeName:            types.ObjectNull(alertSettingsAttrTypes),
					"use_global_alert_settings":           types.BoolValue(true),
					alertChannelSubscriptionAttributeName: types.SetNull(checkGroupV2AlertChannelSubscriptionSetType.ElemType),
				}))
				requireValue(t, "enforce_retry_strategy", target.EnforceRetryStrategy, enabled(checkGroupV2EnforceRetryStrategyAttrTypes, map[string]attr.Value{
					retryStrategyAttributeName: noRetriesObject(true),
				}))
				requireValue(t, "enforce_scheduling_strategy", target.EnforceSchedulingStrategy, enabled(checkGroupV2EnforceSchedulingStrategyAttrTypes, map[string]attr.Value{
					"run_parallel": types.BoolValue(false),
				}))
				requireValue(t, "setup_script", target.SetupScript, types.ObjectNull(checkGroupV2ScriptAttrTypes))
				requireValue(t, "teardown_script", target.TeardownScript, types.ObjectNull(checkGroupV2ScriptAttrTypes))
				requireValue(t, "default_runtime", target.DefaultRuntime, types.ObjectNull(checkGroupV2DefaultRuntimeAttrTypes))
				requireValue(t, "api_check_defaults", target.APICheckDefaults, apiCheckDefaultsDefaultObject())
			},
		},
		"empty collections become null": {
			edit: func(source *CheckGroupResourceModel) {
				source.Tags = stringSet()
				source.Locations = stringSet()
				source.PrivateLocations = stringSet()
				source.AlertChannelSubscriptions = types.SetValueMust(types.ObjectType{AttrTypes: alertChannelSubscriptionAttrTypes}, []attr.Value{})
			},
			check: func(t *testing.T, target CheckGroupV2ResourceModel) {
				requireValue(t, "tags", target.Tags, types.SetNull(types.StringType))
				requireValue(t, "enforce_locations", target.EnforceLocations, checkGroupV2DisabledEnforcementObject(checkGroupV2EnforceLocationsAttrTypes))
				requireValue(t, "enforce_alert_settings.alert_channel_subscriptions", target.EnforceAlertSettings.Attributes()[alertChannelSubscriptionAttributeName], types.SetNull(checkGroupV2AlertChannelSubscriptionSetType.ElemType))
			},
		},
		"tags and locations": {
			edit: func(source *CheckGroupResourceModel) {
				source.Tags = stringSet("a")
				source.Locations = stringSet()
				source.PrivateLocations = stringSet("dc-1")
			},
			check: func(t *testing.T, target CheckGroupV2ResourceModel) {
				requireValue(t, "tags", target.Tags, stringSet("a"))
				requireValue(t, "enforce_locations", target.EnforceLocations, enabled(checkGroupV2EnforceLocationsAttrTypes, map[string]attr.Value{
					"locations":         types.SetNull(types.StringType),
					"private_locations": stringSet("dc-1"),
				}))
			},
		},
		"alert settings": {
			edit: func(source *CheckGroupResourceModel) {
				source.AlertSettings = alertSettings
				source.UseGlobalAlertSettings = types.BoolValue(false)
				source.AlertChannelSubscriptions = alertChannelSubscriptionsSet(7, true)
			},
			check: func(t *testing.T, target CheckGroupV2ResourceModel) {
				requireValue(t, "enforce_alert_settings", target.EnforceAlertSettings, enabled(checkGroupV2EnforceAlertSettingsAttrTypes, map[string]attr.Value{
					alertSettingsAttributeName:            alertSettings,
					"use_global_alert_settings":           types.BoolValue(false),
					alertChannelSubscriptionAttributeName: alertChannelSubscriptionsSet(7, true),
				}))
			},
		},
		"alert settings with an empty escalation type": {
			edit: func(source *CheckGroupResourceModel) {
				source.AlertSettings = zeroAlertSettings
				source.UseGlobalAlertSettings = types.BoolNull()
			},
			check: func(t *testing.T, target CheckGroupV2ResourceModel) {
				attrs := target.EnforceAlertSettings.Attributes()
				requireValue(t, "enforce_alert_settings.enabled", attrs["enabled"], types.BoolValue(true))
				requireValue(t, "enforce_alert_settings.alert_settings", attrs[alertSettingsAttributeName], types.ObjectNull(alertSettingsAttrTypes))
				requireValue(t, "enforce_alert_settings.use_global_alert_settings", attrs["use_global_alert_settings"], types.BoolValue(false))
			},
		},
		"no retry strategy": {
			edit: func(source *CheckGroupResourceModel) {
				source.RetryStrategy = types.ObjectNull(retryStrategyAttrTypes(true))
			},
			check: func(t *testing.T, target CheckGroupV2ResourceModel) {
				requireValue(t, "enforce_retry_strategy", target.EnforceRetryStrategy, enabled(checkGroupV2EnforceRetryStrategyAttrTypes, map[string]attr.Value{
					retryStrategyAttributeName: noRetriesObject(true),
				}))
			},
		},
		"a FALLBACK retry strategy": {
			edit: func(source *CheckGroupResourceModel) {
				source.RetryStrategy = fallback
			},
			check: func(t *testing.T, target CheckGroupV2ResourceModel) {
				requireValue(t, "enforce_retry_strategy", target.EnforceRetryStrategy, checkGroupV2DisabledEnforcementObject(checkGroupV2EnforceRetryStrategyAttrTypes))
			},
		},
		"double_check": {
			edit: func(source *CheckGroupResourceModel) {
				source.DoubleCheck = types.BoolValue(true)
				source.RetryStrategy = doubleCheckRetryStrategy(true)
			},
			check: func(t *testing.T, target CheckGroupV2ResourceModel) {
				requireValue(t, "enforce_retry_strategy", target.EnforceRetryStrategy, enabled(checkGroupV2EnforceRetryStrategyAttrTypes, map[string]attr.Value{
					retryStrategyAttributeName: doubleCheckRetryStrategy(true),
				}))
			},
		},
		"run_parallel": {
			edit: func(source *CheckGroupResourceModel) {
				source.RunParallel = types.BoolValue(true)
			},
			check: func(t *testing.T, target CheckGroupV2ResourceModel) {
				requireValue(t, "enforce_scheduling_strategy.run_parallel", target.EnforceSchedulingStrategy.Attributes()["run_parallel"], types.BoolValue(true))
			},
		},
		"scripts and runtime": {
			edit: func(source *CheckGroupResourceModel) {
				source.SetupSnippetID = types.Int64Value(3)
				source.LocalSetupScript = types.StringValue("setup")
				source.LocalTeardownScript = types.StringValue("teardown")
				source.RuntimeID = types.StringValue("2025.04")
			},
			check: func(t *testing.T, target CheckGroupV2ResourceModel) {
				// Both values of a phase are kept, as the API stores them.
				requireValue(t, "setup_script", target.SetupScript, types.ObjectValueMust(checkGroupV2ScriptAttrTypes, map[string]attr.Value{
					"snippet_id":    types.Int64Value(3),
					"inline_script": types.StringValue("setup"),
				}))
				requireValue(t, "teardown_script", target.TeardownScript, types.ObjectValueMust(checkGroupV2ScriptAttrTypes, map[string]attr.Value{
					"snippet_id":    types.Int64Null(),
					"inline_script": types.StringValue("teardown"),
				}))
				requireValue(t, "default_runtime", target.DefaultRuntime, types.ObjectValueMust(checkGroupV2DefaultRuntimeAttrTypes, map[string]attr.Value{
					"runtime_id": types.StringValue("2025.04"),
				}))
			},
		},
		"snippet IDs of 0 stored by earlier versions": {
			edit: func(source *CheckGroupResourceModel) {
				source.SetupSnippetID = types.Int64Value(0)
				source.TeardownSnippetID = types.Int64Value(0)
				source.LocalSetupScript = types.StringValue("setup")
			},
			check: func(t *testing.T, target CheckGroupV2ResourceModel) {
				requireValue(t, "setup_script", target.SetupScript, types.ObjectValueMust(checkGroupV2ScriptAttrTypes, map[string]attr.Value{
					"snippet_id":    types.Int64Null(),
					"inline_script": types.StringValue("setup"),
				}))
				requireValue(t, "teardown_script", target.TeardownScript, types.ObjectNull(checkGroupV2ScriptAttrTypes))
			},
		},
		"an empty runtime ID": {
			edit: func(source *CheckGroupResourceModel) {
				source.RuntimeID = types.StringValue("")
			},
			check: func(t *testing.T, target CheckGroupV2ResourceModel) {
				requireValue(t, "default_runtime", target.DefaultRuntime, types.ObjectNull(checkGroupV2DefaultRuntimeAttrTypes))
			},
		},
		"no environment variables": {
			edit: func(source *CheckGroupResourceModel) {
				source.EnvironmentVariables = types.ListValueMust(types.ObjectType{AttrTypes: environmentVariableAttrTypes}, []attr.Value{})
			},
			check: func(t *testing.T, target CheckGroupV2ResourceModel) {
				requireValue(t, "environment_variables", target.EnvironmentVariables, types.ListNull(types.ObjectType{AttrTypes: environmentVariableAttrTypes}))
			},
		},
		"secret environment variables": {
			edit: func(source *CheckGroupResourceModel) {
				source.EnvironmentVariables = secretVariables
			},
			check: func(t *testing.T, target CheckGroupV2ResourceModel) {
				requireValue(t, "environment_variables", target.EnvironmentVariables, secretVariables)
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			source := checkGroupModelForMove()
			if c.edit != nil {
				c.edit(&source)
			}

			var diags diag.Diagnostics
			target := checkGroupV2ModelFromCheckGroupModel(ctx, &source, &diags)
			if diags.HasError() {
				t.Fatal(diags)
			}
			c.check(t, target)
		})
	}
}

// requireValue fails the test unless got equals want.
func requireValue(t *testing.T, name string, got, want attr.Value) {
	t.Helper()

	if !got.Equal(want) {
		t.Errorf("%s = %s, want %s", name, got, want)
	}
}

// A `moved` block from a checkly_check_group written by this provider moves
// its state, which the framework has decoded with the checkly_check_group
// schema, to the checkly_check_group_v2 schema.
func TestCheckGroupV2MoveStateFromCurrentState(t *testing.T) {
	ctx := context.Background()

	var sourceSchema resource.SchemaResponse
	NewCheckGroupResource().Schema(ctx, resource.SchemaRequest{}, &sourceSchema)

	model := checkGroupModelForMove()
	model.Locations = types.SetValueMust(types.StringType, []attr.Value{types.StringValue("eu-west-1")})
	sourceState := tfsdk.State{Schema: sourceSchema.Schema}
	if diags := sourceState.Set(ctx, &model); diags.HasError() {
		t.Fatal(diags)
	}

	resp := moveState(t, NewCheckGroupV2Resource(), resource.MoveStateRequest{
		SourceProviderAddress: "registry.terraform.io/checkly/checkly",
		SourceTypeName:        "checkly_check_group",
		SourceSchemaVersion:   sourceSchema.Schema.Version,
		SourceState:           &sourceState,
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("move diagnostics: %v", resp.Diagnostics)
	}

	var target CheckGroupV2ResourceModel
	if diags := resp.TargetState.Get(ctx, &target); diags.HasError() {
		t.Fatal(diags)
	}
	requireValue(t, "id", target.ID, types.StringValue("42"))
	requireValue(t, "enforce_locations.locations", target.EnforceLocations.Attributes()["locations"], model.Locations)
}
