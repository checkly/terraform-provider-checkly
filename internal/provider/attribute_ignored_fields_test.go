package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func retryStrategyValue(t *testing.T, values map[string]attr.Value) types.Object {
	t.Helper()

	obj, diags := types.ObjectValue(retryStrategyAttrTypes(false), values)
	if diags.HasError() {
		t.Fatal(diags)
	}
	return obj
}

func TestRetryStrategyIgnoredFields(t *testing.T) {
	ctx := context.Background()

	// retry_strategy of a monitor upgraded from v1.x state, where SDK v2
	// stored zero values for the fields NO_RETRIES does not use.
	var upgraded types.Object
	state := upgradeSDKv2Fixture(t, NewSSLMonitorResource(), "ssl_monitor_read")
	if diags := state.GetAttribute(ctx, path.Root("retry_strategy"), &upgraded); diags.HasError() {
		t.Fatal(diags)
	}

	singleRetry := func(maxRetries, maxDuration attr.Value) types.Object {
		return retryStrategyValue(t, map[string]attr.Value{
			"type":                 types.StringValue("SINGLE_RETRY"),
			"base_backoff_seconds": types.Int64Value(60),
			"max_retries":          maxRetries,
			"max_duration_seconds": maxDuration,
			"same_region":          types.BoolValue(true),
		})
	}

	cases := map[string]struct {
		state, config, plan, want types.Object
	}{
		"unconfigured NO_RETRIES keeps the stored zero values": {
			state:  upgraded,
			config: types.ObjectNull(retryStrategyAttrTypes(false)),
			plan:   noRetriesObject(false),
			want:   upgraded,
		},
		"unused SINGLE_RETRY fields keep their stored values": {
			state:  singleRetry(types.Int64Value(0), types.Int64Value(0)),
			config: singleRetry(types.Int64Null(), types.Int64Null()),
			plan:   singleRetry(types.Int64Value(2), types.Int64Value(600)),
			want:   singleRetry(types.Int64Value(0), types.Int64Value(0)),
		},
		"configured fields are planned as configured": {
			state:  singleRetry(types.Int64Value(0), types.Int64Value(0)),
			config: singleRetry(types.Int64Value(3), types.Int64Null()),
			plan:   singleRetry(types.Int64Value(3), types.Int64Value(600)),
			want:   singleRetry(types.Int64Value(3), types.Int64Value(0)),
		},
		"a changed type is planned as is": {
			state:  upgraded,
			config: singleRetry(types.Int64Null(), types.Int64Null()),
			plan:   singleRetry(types.Int64Value(2), types.Int64Value(600)),
			want:   singleRetry(types.Int64Value(2), types.Int64Value(600)),
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			resp := planmodifier.ObjectResponse{PlanValue: c.plan}
			retryStrategyIgnoredFields.PlanModifyObject(ctx, planmodifier.ObjectRequest{
				StateValue:  c.state,
				ConfigValue: c.config,
				PlanValue:   c.plan,
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

func TestCheckGroupV2DisabledEnforcementIgnoredFields(t *testing.T) {
	ctx := context.Background()

	enforcement := func(enabled bool, runParallel attr.Value) types.Object {
		return types.ObjectValueMust(checkGroupV2EnforceSchedulingStrategyAttrTypes, map[string]attr.Value{
			"enabled":      types.BoolValue(enabled),
			"run_parallel": runParallel,
		})
	}

	disabledDefault := checkGroupV2DisabledEnforcementObject(checkGroupV2EnforceSchedulingStrategyAttrTypes)

	cases := map[string]struct {
		state, config, plan, want types.Object
	}{
		"a disabled enforcement keeps stored fields": {
			state:  enforcement(false, types.BoolValue(false)),
			config: types.ObjectNull(checkGroupV2EnforceSchedulingStrategyAttrTypes),
			plan:   disabledDefault,
			want:   enforcement(false, types.BoolValue(false)),
		},
		"enabling an enforcement plans its fields": {
			state:  enforcement(false, types.BoolValue(false)),
			config: enforcement(true, types.BoolValue(true)),
			plan:   enforcement(true, types.BoolValue(true)),
			want:   enforcement(true, types.BoolValue(true)),
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			resp := planmodifier.ObjectResponse{PlanValue: c.plan}
			checkGroupV2DisabledEnforcementIgnoredFields.PlanModifyObject(ctx, planmodifier.ObjectRequest{
				StateValue:  c.state,
				ConfigValue: c.config,
				PlanValue:   c.plan,
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
