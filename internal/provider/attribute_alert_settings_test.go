package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/checkly/checkly-go-sdk/v2"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestInactiveEscalationPlanModifier(t *testing.T) {
	ctx := context.Background()

	var schemaResp resource.SchemaResponse
	NewURLMonitorResource().Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema

	planWithEscalationType := func(escalationType string) tfsdk.Plan {
		plan := tfsdk.Plan{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
		if diags := plan.SetAttribute(ctx, path.Root("alert_settings").AtName("escalation_type"), escalationType); diags.HasError() {
			t.Fatal(diags)
		}
		return plan
	}

	unknown := types.ObjectUnknown(timeBasedEscalationAttrTypes)
	null := types.ObjectNull(timeBasedEscalationAttrTypes)

	cases := map[string]struct {
		escalationType string
		want           types.Object
	}{
		"the inactive escalation keeps its stored null": {"RUN_BASED", null},
		"the active escalation stays unknown":           {"TIME_BASED", unknown},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			resp := planmodifier.ObjectResponse{PlanValue: unknown}
			inactiveEscalationPlanModifier{escalationType: "TIME_BASED"}.PlanModifyObject(ctx, planmodifier.ObjectRequest{
				Path:        path.Root("alert_settings").AtName("time_based_escalation"),
				Plan:        planWithEscalationType(c.escalationType),
				StateValue:  null,
				ConfigValue: null,
				PlanValue:   unknown,
			}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			if !resp.PlanValue.Equal(c.want) {
				t.Errorf("planned %s, want %s", resp.PlanValue, c.want)
			}
		})
	}
}

// An escalation object that is not configured is planned unknown, so the
// conversion has to apply its documented default.
func TestAlertSettingsFromObjectEscalationDefaults(t *testing.T) {
	ctx := context.Background()

	alertSettings := func(escalationType string) types.Object {
		return types.ObjectValueMust(alertSettingsAttrTypes, map[string]attr.Value{
			"escalation_type":                types.StringValue(escalationType),
			"run_based_escalation":           types.ObjectUnknown(runBasedEscalationAttrTypes),
			"time_based_escalation":          types.ObjectUnknown(timeBasedEscalationAttrTypes),
			"reminders":                      types.ObjectUnknown(remindersAttrTypes),
			"parallel_run_failure_threshold": types.ObjectUnknown(parallelRunFailureThresholdAttrTypes),
		})
	}

	var diags diag.Diagnostics
	runBased := alertSettingsFromObject(ctx, alertSettings("RUN_BASED"), &diags)
	timeBased := alertSettingsFromObject(ctx, alertSettings("TIME_BASED"), &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}

	if got := runBased.RunBasedEscalation.FailedRunThreshold; got != 1 {
		t.Errorf("failed run threshold = %d, want 1", got)
	}
	if got := timeBased.TimeBasedEscalation.MinutesFailingThreshold; got != 5 {
		t.Errorf("minutes failing threshold = %d, want 5", got)
	}
}

func TestAlertSettingsFromAPI(t *testing.T) {
	ctx := context.Background()

	apiSettings := &checkly.AlertSettings{
		EscalationType:     checkly.EscalationTypeRunBased,
		RunBasedEscalation: checkly.RunBasedEscalation{FailedRunThreshold: 2},
	}

	var diags diag.Diagnostics
	known := alertSettingsToObject(ctx, *apiSettings, types.ObjectNull(alertSettingsAttrTypes), &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}

	cases := map[string]struct {
		as    *checkly.AlertSettings
		prior types.Object
		want  types.Object
	}{
		"reported, prior null":      {apiSettings, types.ObjectNull(alertSettingsAttrTypes), known},
		"reported, prior unknown":   {apiSettings, types.ObjectUnknown(alertSettingsAttrTypes), known},
		"not reported, prior known": {nil, known, known},
		"not reported, prior null":  {nil, types.ObjectNull(alertSettingsAttrTypes), types.ObjectNull(alertSettingsAttrTypes)},
		"not reported, prior unknown": {
			nil, types.ObjectUnknown(alertSettingsAttrTypes), types.ObjectNull(alertSettingsAttrTypes),
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			var diags diag.Diagnostics
			got := alertSettingsFromAPI(ctx, c.as, c.prior, &diags)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if !got.Equal(c.want) {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

// TestAlertSettingsGroupWording checks that the alert settings and alert
// channel subscriptions of both check group resources describe the checks and
// monitors in the group, at every level, instead of a single check.
func TestAlertSettingsGroupWording(t *testing.T) {
	ctx := context.Background()
	groupSchema := func(r resource.Resource) map[string]schema.Attribute {
		var resp resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &resp)
		return resp.Schema.Attributes
	}
	v1 := groupSchema(NewCheckGroupResource())
	v2 := groupSchema(NewCheckGroupV2Resource())

	attributes := map[string]schema.Attribute{
		"checkly_check_group.alert_settings":              v1[alertSettingsAttributeName],
		"checkly_check_group.alert_channel_subscriptions": v1[alertChannelSubscriptionAttributeName],
		"checkly_check_group_v2.enforce_alert_settings":   v2["enforce_alert_settings"],
	}

	var check func(name string, a schema.Attribute)
	check = func(name string, a schema.Attribute) {
		d := a.GetDescription()
		if strings.Contains(d, "the check ") || strings.Contains(d, "the check.") ||
			strings.Contains(d, "check runs") || strings.Contains(d, "your check") {
			t.Errorf("%s: description talks about a single check: %q", name, d)
		}
		var nested map[string]schema.Attribute
		switch a := a.(type) {
		case schema.SingleNestedAttribute:
			nested = a.Attributes
		case schema.SetNestedAttribute:
			nested = a.NestedObject.Attributes
		}
		for n, child := range nested {
			check(name+"."+n, child)
		}
	}
	for name, a := range attributes {
		if a == nil {
			t.Fatalf("%s: attribute not found", name)
		}
		check(name, a)
	}

	want := "Determines the alert escalation policy for the checks and monitors in the group."
	if got := v1[alertSettingsAttributeName].GetDescription(); got != want {
		t.Errorf("checkly_check_group.alert_settings description = %q, want %q", got, want)
	}
}
