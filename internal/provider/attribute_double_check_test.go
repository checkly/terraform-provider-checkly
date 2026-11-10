package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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

func TestDoubleCheckValidator(t *testing.T) {
	ctx := context.Background()

	retryStrategyType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"type": tftypes.String}}
	configType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		doubleCheckAttributeName:   tftypes.Bool,
		retryStrategyAttributeName: retryStrategyType,
	}}
	configSchema := schema.Schema{
		Attributes: map[string]schema.Attribute{
			doubleCheckAttributeName: schema.BoolAttribute{Optional: true},
			retryStrategyAttributeName: schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{Optional: true},
				},
			},
		},
	}

	doubleChecks := map[string]tftypes.Value{
		"true":    tftypes.NewValue(tftypes.Bool, true),
		"false":   tftypes.NewValue(tftypes.Bool, false),
		"unset":   tftypes.NewValue(tftypes.Bool, nil),
		"unknown": tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue),
	}
	retryStrategies := map[string]tftypes.Value{
		"set": tftypes.NewValue(retryStrategyType, map[string]tftypes.Value{
			"type": tftypes.NewValue(tftypes.String, "FIXED"),
		}),
		"unset":   tftypes.NewValue(retryStrategyType, nil),
		"unknown": tftypes.NewValue(retryStrategyType, tftypes.UnknownValue),
	}

	for doubleCheckName, doubleCheck := range doubleChecks {
		for retryStrategyName, retryStrategy := range retryStrategies {
			// Only a known true double_check with a known retry_strategy is
			// rejected; every other combination is valid or is validated
			// again once its values are known.
			wantError := doubleCheckName == "true" && retryStrategyName == "set"

			name := "double_check " + doubleCheckName + " with retry_strategy " + retryStrategyName
			t.Run(name, func(t *testing.T) {
				config := tfsdk.Config{
					Schema: configSchema,
					Raw: tftypes.NewValue(configType, map[string]tftypes.Value{
						doubleCheckAttributeName:   doubleCheck,
						retryStrategyAttributeName: retryStrategy,
					}),
				}

				var configValue types.Bool
				if diags := config.GetAttribute(ctx, path.Root(doubleCheckAttributeName), &configValue); diags.HasError() {
					t.Fatal(diags)
				}

				var resp validator.BoolResponse
				doubleCheckValidator{}.ValidateBool(ctx, validator.BoolRequest{
					Config:      config,
					ConfigValue: configValue,
					Path:        path.Root(doubleCheckAttributeName),
				}, &resp)
				if got := resp.Diagnostics.HasError(); got != wantError {
					t.Fatalf("error reported = %v, want %v: %v", got, wantError, resp.Diagnostics)
				}
				if wantError {
					if got := resp.Diagnostics.Errors()[0].Summary(); got != "Invalid Attribute Combination" {
						t.Errorf("summary = %q, want %q", got, "Invalid Attribute Combination")
					}
				}
			})
		}
	}
}
