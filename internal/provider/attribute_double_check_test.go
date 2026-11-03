package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestDoubleCheckRetryStrategyPlanModifier(t *testing.T) {
	ctx := context.Background()

	configWithDoubleCheck := func(value tftypes.Value) tfsdk.Config {
		return tfsdk.Config{
			Schema: schema.Schema{
				Attributes: map[string]schema.Attribute{
					doubleCheckAttributeName: schema.BoolAttribute{Optional: true},
				},
			},
			Raw: tftypes.NewValue(
				tftypes.Object{AttributeTypes: map[string]tftypes.Type{doubleCheckAttributeName: tftypes.Bool}},
				map[string]tftypes.Value{doubleCheckAttributeName: value},
			),
		}
	}

	unconfigured := types.ObjectNull(retryStrategyAttrTypes(true))
	configured := doubleCheckRetryStrategy(true)

	cases := map[string]struct {
		doubleCheck tftypes.Value
		config      types.Object
		want        types.Object
	}{
		"unset keeps the default": {
			doubleCheck: tftypes.NewValue(tftypes.Bool, nil),
			config:      unconfigured,
			want:        noRetriesObject(true),
		},
		"false keeps the default": {
			doubleCheck: tftypes.NewValue(tftypes.Bool, false),
			config:      unconfigured,
			want:        noRetriesObject(true),
		},
		"true plans the implied strategy": {
			doubleCheck: tftypes.NewValue(tftypes.Bool, true),
			config:      unconfigured,
			want:        doubleCheckRetryStrategy(true),
		},
		"unknown plans an unknown strategy": {
			doubleCheck: tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue),
			config:      unconfigured,
			want:        types.ObjectUnknown(retryStrategyAttrTypes(true)),
		},
		"a configured strategy is kept": {
			doubleCheck: tftypes.NewValue(tftypes.Bool, true),
			config:      configured,
			want:        configured,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			plan := noRetriesObject(true)
			if !c.config.IsNull() {
				plan = c.config
			}
			resp := planmodifier.ObjectResponse{PlanValue: plan}
			doubleCheckRetryStrategyPlanModifier{}.PlanModifyObject(ctx, planmodifier.ObjectRequest{
				Config:      configWithDoubleCheck(c.doubleCheck),
				ConfigValue: c.config,
				PlanValue:   plan,
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
