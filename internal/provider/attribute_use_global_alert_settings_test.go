package provider

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// useGlobalAlertSettingsTestObjectType is a minimal resource object type with
// both attributes the use_global_alert_settings plan modifier and warning
// read. Only whether alert_settings is null, unknown or set matters to them,
// so a single nested attribute stands in for the full alert_settings object.
var useGlobalAlertSettingsTestObjectType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	useGlobalAlertSettingsAttributeName: tftypes.Bool,
	alertSettingsAttributeName: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"escalation_type": tftypes.String,
	}},
}}

var useGlobalAlertSettingsTestSchema = schema.Schema{
	Attributes: map[string]schema.Attribute{
		useGlobalAlertSettingsAttributeName: schema.BoolAttribute{Optional: true, Computed: true},
		alertSettingsAttributeName: schema.SingleNestedAttribute{
			Optional: true,
			Attributes: map[string]schema.Attribute{
				"escalation_type": schema.StringAttribute{Optional: true},
			},
		},
	},
}

var (
	alertSettingsNull    = tftypes.NewValue(useGlobalAlertSettingsTestObjectType.AttributeTypes[alertSettingsAttributeName], nil)
	alertSettingsUnknown = tftypes.NewValue(useGlobalAlertSettingsTestObjectType.AttributeTypes[alertSettingsAttributeName], tftypes.UnknownValue)
	alertSettingsSet     = tftypes.NewValue(useGlobalAlertSettingsTestObjectType.AttributeTypes[alertSettingsAttributeName], map[string]tftypes.Value{
		"escalation_type": tftypes.NewValue(tftypes.String, "RUN_BASED"),
	})
)

func useGlobalAlertSettingsTestConfig(useGlobal, alertSettings tftypes.Value) tfsdk.Config {
	return tfsdk.Config{
		Schema: useGlobalAlertSettingsTestSchema,
		Raw: tftypes.NewValue(useGlobalAlertSettingsTestObjectType, map[string]tftypes.Value{
			useGlobalAlertSettingsAttributeName: useGlobal,
			alertSettingsAttributeName:          alertSettings,
		}),
	}
}

func TestUseGlobalAlertSettingsPlanModifier(t *testing.T) {
	ctx := context.Background()

	unset := tftypes.NewValue(tftypes.Bool, nil)

	cases := map[string]struct {
		useGlobal     tftypes.Value
		alertSettings tftypes.Value
		destroy       bool
		want          types.Bool
	}{
		"neither set plans true": {
			useGlobal:     unset,
			alertSettings: alertSettingsNull,
			want:          types.BoolValue(true),
		},
		"alert_settings set plans false": {
			useGlobal:     unset,
			alertSettings: alertSettingsSet,
			want:          types.BoolValue(false),
		},
		"unknown alert_settings plans unknown": {
			useGlobal:     unset,
			alertSettings: alertSettingsUnknown,
			want:          types.BoolUnknown(),
		},
		"configured true is kept": {
			useGlobal:     tftypes.NewValue(tftypes.Bool, true),
			alertSettings: alertSettingsSet,
			want:          types.BoolValue(true),
		},
		"configured false is kept": {
			useGlobal:     tftypes.NewValue(tftypes.Bool, false),
			alertSettings: alertSettingsNull,
			want:          types.BoolValue(false),
		},
		"destroy is left alone": {
			useGlobal:     unset,
			alertSettings: alertSettingsNull,
			destroy:       true,
			want:          types.BoolNull(),
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			config := useGlobalAlertSettingsTestConfig(c.useGlobal, c.alertSettings)

			var configValue types.Bool
			if diags := config.GetAttribute(ctx, path.Root(useGlobalAlertSettingsAttributeName), &configValue); diags.HasError() {
				t.Fatal(diags)
			}

			plan := tfsdk.Plan{Schema: config.Schema, Raw: config.Raw}
			planValue := configValue
			if c.destroy {
				plan.Raw = tftypes.NewValue(useGlobalAlertSettingsTestObjectType, nil)
				planValue = types.BoolNull()
			} else if configValue.IsNull() {
				// Terraform proposes unknown for an unconfigured computed
				// attribute of a new resource.
				planValue = types.BoolUnknown()
			}

			resp := planmodifier.BoolResponse{PlanValue: planValue}
			useGlobalAlertSettingsPlanModifier{}.PlanModifyBool(ctx, planmodifier.BoolRequest{
				Config:      config,
				ConfigValue: configValue,
				Plan:        plan,
				PlanValue:   planValue,
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

func TestUseGlobalAlertSettingsValidator(t *testing.T) {
	ctx := context.Background()

	cases := map[string]struct {
		useGlobal     tftypes.Value
		alertSettings tftypes.Value
		wantWarning   bool
	}{
		"true with alert_settings": {
			useGlobal:     tftypes.NewValue(tftypes.Bool, true),
			alertSettings: alertSettingsSet,
			wantWarning:   true,
		},
		"true without alert_settings": {
			useGlobal:     tftypes.NewValue(tftypes.Bool, true),
			alertSettings: alertSettingsNull,
		},
		"true with unknown alert_settings": {
			useGlobal:     tftypes.NewValue(tftypes.Bool, true),
			alertSettings: alertSettingsUnknown,
		},
		"false with alert_settings": {
			useGlobal:     tftypes.NewValue(tftypes.Bool, false),
			alertSettings: alertSettingsSet,
		},
		"unset with alert_settings": {
			useGlobal:     tftypes.NewValue(tftypes.Bool, nil),
			alertSettings: alertSettingsSet,
		},
		"unknown with alert_settings": {
			useGlobal:     tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue),
			alertSettings: alertSettingsSet,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			config := useGlobalAlertSettingsTestConfig(c.useGlobal, c.alertSettings)

			var configValue types.Bool
			if diags := config.GetAttribute(ctx, path.Root(useGlobalAlertSettingsAttributeName), &configValue); diags.HasError() {
				t.Fatal(diags)
			}

			var resp validator.BoolResponse
			useGlobalAlertSettingsValidator{}.ValidateBool(ctx, validator.BoolRequest{
				Config:      config,
				ConfigValue: configValue,
				Path:        path.Root(useGlobalAlertSettingsAttributeName),
			}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			if got := resp.Diagnostics.WarningsCount() > 0; got != c.wantWarning {
				t.Errorf("warning reported = %v, want %v: %v", got, c.wantWarning, resp.Diagnostics)
			}
		})
	}
}

// Every resource with a top-level use_global_alert_settings derives it from
// alert_settings the same way, so each must use the shared schema, which warns
// through the shared attribute validator.
func TestUseGlobalAlertSettingsIsShared(t *testing.T) {
	ctx := context.Background()
	p := New("test")()

	var providerMetadata provider.MetadataResponse
	p.Metadata(ctx, provider.MetadataRequest{}, &providerMetadata)

	var got []string
	for _, newResource := range p.Resources(ctx) {
		r := newResource()

		var metadata resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: providerMetadata.TypeName}, &metadata)

		var schemaResp resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

		attribute, ok := schemaResp.Schema.Attributes[useGlobalAlertSettingsAttributeName].(schema.BoolAttribute)
		if !ok {
			continue
		}
		got = append(got, metadata.TypeName)

		if _, ok := schemaResp.Schema.Attributes[alertSettingsAttributeName]; !ok {
			t.Errorf("%s: has %s but no %s", metadata.TypeName, useGlobalAlertSettingsAttributeName, alertSettingsAttributeName)
		}
		if attribute.Default != nil {
			t.Errorf("%s: %s has a static default", metadata.TypeName, useGlobalAlertSettingsAttributeName)
		}
		if !slices.ContainsFunc(attribute.PlanModifiers, func(m planmodifier.Bool) bool {
			_, ok := m.(useGlobalAlertSettingsPlanModifier)
			return ok
		}) {
			t.Errorf("%s: %s lacks the shared plan modifier", metadata.TypeName, useGlobalAlertSettingsAttributeName)
		}
		if !slices.ContainsFunc([]string{"check", "monitor", "check group"}, func(subject string) bool {
			return attribute.Description == makeUseGlobalAlertSettingsAttributeSchema(subject).Description
		}) {
			t.Errorf("%s: %s has a custom description: %q", metadata.TypeName, useGlobalAlertSettingsAttributeName, attribute.Description)
		}

		if !slices.ContainsFunc(attribute.Validators, func(v validator.Bool) bool {
			_, ok := v.(useGlobalAlertSettingsValidator)
			return ok
		}) {
			t.Errorf("%s: %s lacks the shared validator", metadata.TypeName, useGlobalAlertSettingsAttributeName)
		}
	}

	want := []string{
		"checkly_api_check",
		"checkly_browser_check",
		"checkly_check",
		"checkly_check_group",
		"checkly_dns_monitor",
		"checkly_grpc_monitor",
		"checkly_heartbeat",
		"checkly_heartbeat_monitor",
		"checkly_icmp_monitor",
		"checkly_multistep_check",
		"checkly_playwright_check_suite",
		"checkly_ssl_monitor",
		"checkly_tcp_check",
		"checkly_tcp_monitor",
		"checkly_traceroute_monitor",
		"checkly_url_monitor",
	}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("resources with %s = %v, want %v", useGlobalAlertSettingsAttributeName, got, want)
	}
}

// useGlobalAlertSettingsAccSteps returns steps that walk a resource through
// every way use_global_alert_settings is derived: from no alert_settings, to
// configured alert_settings, back to none, and finally to an explicit true
// alongside alert_settings, which wins over the derived value. Removing
// alert_settings keeps the stored alert settings, so the last step configures
// the stored ones again and, with use_global_alert_settings still true, plans
// no change at all. The implicit empty-plan check after each step verifies
// that the API stores the planned value.
func useGlobalAlertSettingsAccSteps(resourceType, body string) []tfresource.TestStep {
	address := resourceType + ".test"
	config := func(extra string) string {
		return fmt.Sprintf("resource %q \"test\" {\n%s\n%s\n}\n", resourceType, body, extra)
	}
	const alertSettings = `
		alert_settings = {
			escalation_type = "RUN_BASED"
			run_based_escalation = {
				failed_run_threshold = 2
			}
		}
	`
	step := func(extra string, action plancheck.ResourceActionType, want bool) tfresource.TestStep {
		value := tfjsonpath.New(useGlobalAlertSettingsAttributeName)
		return tfresource.TestStep{
			Config: config(extra),
			ConfigPlanChecks: tfresource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(address, action),
					plancheck.ExpectKnownValue(address, value, knownvalue.Bool(want)),
				},
			},
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue(address, value, knownvalue.Bool(want)),
			},
		}
	}

	return []tfresource.TestStep{
		step("", plancheck.ResourceActionCreate, true),
		step(alertSettings, plancheck.ResourceActionUpdate, false),
		step("", plancheck.ResourceActionUpdate, true),
		step(alertSettings+"use_global_alert_settings = true", plancheck.ResourceActionNoop, true),
	}
}

func TestAccUseGlobalAlertSettingsAPICheck(t *testing.T) {
	accTestCase(t, useGlobalAlertSettingsAccSteps("checkly_api_check", fmt.Sprintf(`
		name      = "use-global-alert-settings-%d"
		locations = ["eu-central-1"]
		request = {
			url = "https://api.checklyhq.com/public-stats"
		}
	`, acctest.RandInt())))
}

func TestAccUseGlobalAlertSettingsURLMonitor(t *testing.T) {
	accTestCase(t, useGlobalAlertSettingsAccSteps("checkly_url_monitor", fmt.Sprintf(`
		name      = "use-global-alert-settings-%d"
		activated = true
		frequency = 60
		locations = ["eu-central-1"]
		request = {
			url = "https://api.checklyhq.com"
		}
	`, acctest.RandInt())))
}

// A check group request omits alert_settings when it is not configured, so
// the group covers the derivation without alert settings in the request.
func TestAccUseGlobalAlertSettingsCheckGroup(t *testing.T) {
	accTestCase(t, useGlobalAlertSettingsAccSteps("checkly_check_group", fmt.Sprintf(`
		name        = "use-global-alert-settings-%d"
		activated   = true
		concurrency = 1
		locations   = ["eu-central-1"]
	`, acctest.RandInt())))
}
