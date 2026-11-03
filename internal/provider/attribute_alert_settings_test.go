package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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
