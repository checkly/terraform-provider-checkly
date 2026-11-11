package provider

import (
	"context"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/checkly/checkly-go-sdk/v2"
)

// TestReferenceAttributeMinimums verifies that the group and snippet
// references, which are null when unset, reject 0 and negative values at plan
// time, and that the group position accepts 0 but rejects negative values.
func TestReferenceAttributeMinimums(t *testing.T) {
	groupMembers := []struct {
		resource resource.Resource
		typeName string
	}{
		{NewAPICheckResource(), "checkly_api_check"},
		{NewBrowserCheckResource(), "checkly_browser_check"},
		{NewMultistepCheckResource(), "checkly_multistep_check"},
		{NewCheckResource(), "checkly_check"},
		{NewURLMonitorResource(), "checkly_url_monitor"},
		{NewTCPMonitorResource(), "checkly_tcp_monitor"},
		{NewTCPCheckResource(), "checkly_tcp_check"},
		{NewDNSMonitorResource(), "checkly_dns_monitor"},
		{NewICMPMonitorResource(), "checkly_icmp_monitor"},
		{NewGRPCMonitorResource(), "checkly_grpc_monitor"},
		{NewSSLMonitorResource(), "checkly_ssl_monitor"},
		{NewTracerouteMonitorResource(), "checkly_traceroute_monitor"},
		{NewPlaywrightCheckSuiteResource(), "checkly_playwright_check_suite"},
	}
	type target struct {
		resource resource.Resource
		typeName string
		attr     string
		min      int64
	}
	var targets []target
	for _, m := range groupMembers {
		targets = append(targets,
			target{m.resource, m.typeName, groupIDAttributeName, 1},
			target{m.resource, m.typeName, groupOrderAttributeName, 0},
		)
	}
	for _, name := range []string{"setup_snippet_id", "teardown_snippet_id"} {
		targets = append(targets,
			target{NewAPICheckResource(), "checkly_api_check", name, 1},
			target{NewCheckResource(), "checkly_check", name, 1},
			target{NewCheckGroupResource(), "checkly_check_group", name, 1},
		)
	}

	for _, c := range targets {
		t.Run(c.typeName+"/"+c.attr, func(t *testing.T) {
			for value, wantErrors := range map[int64]int{c.min - 1: 1, c.min: 0, c.min + 1: 0} {
				errors := validateResourceConfig(t, c.resource, c.typeName, c.attr, func(objectType tftypes.Object) tftypes.Value {
					return objectWith(objectType, map[string]tftypes.Value{
						c.attr: tftypes.NewValue(tftypes.Number, big.NewFloat(float64(value))),
					})
				})
				if len(errors) != wantErrors {
					t.Errorf("%s = %d: got %d errors, want %d: %v", c.attr, value, len(errors), wantErrors, errors)
				}
			}
		})
	}
}

// TestReadsStoreNullForNoReference verifies that the API's 0 for no group, no
// position and no snippet is read as null, and that other values are kept.
func TestReadsStoreNullForNoReference(t *testing.T) {
	ctx := context.Background()

	requireInt64 := func(t *testing.T, name string, got types.Int64, want types.Int64) {
		t.Helper()
		if !got.Equal(want) {
			t.Errorf("%s = %s, want %s", name, got, want)
		}
	}
	cases := []struct {
		value int64
		want  types.Int64
	}{
		{0, types.Int64Null()},
		{7, types.Int64Value(7)},
	}

	t.Run("checkly_api_check", func(t *testing.T) {
		for _, c := range cases {
			var diags diag.Diagnostics
			model := APICheckResourceModel{
				AlertSettings: types.ObjectNull(alertSettingsAttrTypes),
				RetryStrategy: types.ObjectNull(retryStrategyAttrTypes(true)),
				Request:       types.ObjectNull(apiCheckRequestAttrTypes),
			}
			apiCheckToModel(ctx, &checkly.APICheck{
				ID:                "abc",
				GroupID:           c.value,
				GroupOrder:        int(c.value),
				SetupSnippetID:    c.value,
				TearDownSnippetID: c.value,
			}, &model, &diags)
			if diags.HasError() {
				t.Fatal(diags)
			}
			requireInt64(t, "group_id", model.GroupID, c.want)
			requireInt64(t, "group_order", model.GroupOrder, c.want)
			requireInt64(t, "setup_snippet_id", model.SetupSnippetID, c.want)
			requireInt64(t, "teardown_snippet_id", model.TeardownSnippetID, c.want)
		}
	})

	t.Run("checkly_check", func(t *testing.T) {
		for _, c := range cases {
			var diags diag.Diagnostics
			model := emptyCheckModel()
			checkToModel(ctx, &checkly.APICheck{
				ID:                "abc",
				GroupID:           c.value,
				GroupOrder:        int(c.value),
				SetupSnippetID:    c.value,
				TearDownSnippetID: c.value,
			}, &model, &diags)
			if diags.HasError() {
				t.Fatal(diags)
			}
			requireInt64(t, "group_id", model.GroupID, c.want)
			requireInt64(t, "group_order", model.GroupOrder, c.want)
			requireInt64(t, "setup_snippet_id", model.SetupSnippetID, c.want)
			requireInt64(t, "teardown_snippet_id", model.TeardownSnippetID, c.want)
		}
	})

	t.Run("checkly_url_monitor", func(t *testing.T) {
		for _, c := range cases {
			var diags diag.Diagnostics
			model := URLMonitorResourceModel{
				AlertSettings: types.ObjectNull(alertSettingsAttrTypes),
				RetryStrategy: types.ObjectNull(retryStrategyAttrTypes(true)),
			}
			urlMonitorToModel(ctx, &checkly.URLMonitor{ID: "abc", GroupID: c.value, GroupOrder: int(c.value)}, &model, &diags)
			if diags.HasError() {
				t.Fatal(diags)
			}
			requireInt64(t, "group_id", model.GroupID, c.want)
			requireInt64(t, "group_order", model.GroupOrder, c.want)
		}
	})

	t.Run("checkly_check_group", func(t *testing.T) {
		var diags diag.Diagnostics
		model := CheckGroupResourceModel{AlertSettings: types.ObjectNull(alertSettingsAttrTypes)}
		zero, seven := int64(0), int64(7)
		checkGroupToModel(ctx, &checkly.GroupV2{ID: 42, Name: "g", SetupSnippetID: &zero, TearDownSnippetID: &seven}, &model, &diags)
		if diags.HasError() {
			t.Fatal(diags)
		}
		requireInt64(t, "setup_snippet_id", model.SetupSnippetID, types.Int64Null())
		requireInt64(t, "teardown_snippet_id", model.TeardownSnippetID, types.Int64Value(7))
	})
}
