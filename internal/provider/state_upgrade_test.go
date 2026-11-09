package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// The fixtures in testdata/sdkv2-state are resource state as serialized by
// the provider's SDK v2 implementation (v1.x), including its empty lists for
// absent blocks and attributes that have since been removed.

func readSDKv2StateFixture(t *testing.T, name string) *tfprotov6.RawState {
	t.Helper()

	b, err := os.ReadFile(filepath.Join("testdata", "sdkv2-state", name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	return &tfprotov6.RawState{JSON: b}
}

func upgradeSDKv2Fixture(t *testing.T, r resource.Resource, fixture string) tfsdk.State {
	t.Helper()
	ctx := context.Background()

	upgrader, ok := r.(resource.ResourceWithUpgradeState).UpgradeState(ctx)[sdkv2SchemaVersion]
	if !ok {
		t.Fatalf("no state upgrader for schema version %d", sdkv2SchemaVersion)
	}

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	resp := resource.UpgradeStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	upgrader.StateUpgrader(ctx, resource.UpgradeStateRequest{RawState: readSDKv2StateFixture(t, fixture)}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrade diagnostics: %v", resp.Diagnostics)
	}
	return resp.State
}

// upgradeSDKv2Raw upgrades raw state written by the SDK v2 implementation of
// r, given as JSON, to r's current schema.
func upgradeSDKv2Raw(t *testing.T, r resource.Resource, raw string) tfsdk.State {
	t.Helper()

	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	value, err := upgradeSDKv2RawState(context.Background(), &tfprotov6.RawState{JSON: []byte(raw)}, schemaResp.Schema)
	if err != nil {
		t.Fatal(err)
	}

	return tfsdk.State{Schema: schemaResp.Schema, Raw: value}
}

// sdkv2FixtureWith returns the JSON of the SDK v2 state fixture name after
// edit has changed its decoded object.
func sdkv2FixtureWith(t *testing.T, name string, edit func(obj map[string]any)) string {
	t.Helper()

	obj := decodeSDKv2TestJSON(t, string(readSDKv2StateFixture(t, name).JSON))
	edit(obj)
	b, err := json.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// environmentVariablesList returns an environment_variables value with a
// single variable that is neither locked nor secret.
func environmentVariablesList(t *testing.T, key, value string) types.List {
	t.Helper()

	return environmentVariableTestList(t, []EnvironmentVariableModel{{
		Key:    types.StringValue(key),
		Value:  types.StringValue(value),
		Locked: types.BoolValue(false),
		Secret: types.BoolValue(false),
	}})
}

// alertChannelSubscriptionsSet returns an alert_channel_subscriptions value
// with a single subscription.
func alertChannelSubscriptionsSet(channelID int64, activated bool) types.Set {
	return types.SetValueMust(
		types.ObjectType{AttrTypes: alertChannelSubscriptionAttrTypes},
		[]attr.Value{types.ObjectValueMust(alertChannelSubscriptionAttrTypes, map[string]attr.Value{
			"channel_id": types.Int64Value(channelID),
			"activated":  types.BoolValue(activated),
		})},
	)
}

func requireStateValue(t *testing.T, state tfsdk.State, p path.Path, want attr.Value) {
	t.Helper()

	var got attr.Value
	if diags := state.GetAttribute(context.Background(), p, &got); diags.HasError() {
		t.Fatalf("%s: %v", p, diags)
	}
	if !got.Equal(want) {
		t.Errorf("%s = %s, want %s", p, got, want)
	}
}

func requireStateNull(t *testing.T, state tfsdk.State, p path.Path) {
	t.Helper()

	var got attr.Value
	if diags := state.GetAttribute(context.Background(), p, &got); diags.HasError() {
		t.Fatalf("%s: %v", p, diags)
	}
	if !got.IsNull() {
		t.Errorf("%s = %s, want null", p, got)
	}
}

func TestUpgradeSDKv2State(t *testing.T) {
	t.Run("url_monitor", func(t *testing.T) {
		state := upgradeSDKv2Fixture(t, NewURLMonitorResource(), "url_monitor")
		requireStateValue(t, state, path.Root("id"), types.StringValue("123"))
		requireStateValue(t, state, path.Root("alert_settings").AtName("escalation_type"), types.StringValue("RUN_BASED"))
		requireStateValue(t, state, path.Root("alert_settings").AtName("reminders").AtName("amount"), types.Int64Value(1))
		requireStateValue(t, state, path.Root("alert_settings").AtName("run_based_escalation").AtName("failed_run_threshold"), types.Int64Value(2))
		// An absent nested block, stored as [] by SDK v2.
		requireStateNull(t, state, path.Root("alert_settings").AtName("time_based_escalation"))
		requireStateValue(t, state, path.Root("retry_strategy").AtName("base_backoff_seconds"), types.Int64Value(30))
		requireStateValue(t, state, path.Root("retry_strategy").AtName("only_on").AtName("network_error"), types.BoolValue(true))
		// The one-element request list is unwrapped into an object; its
		// assertion list becomes the assertions set.
		requireStateValue(t, state, path.Root("request").AtName("url"), types.StringValue("https://example.com"))
		requireStateValue(t, state, path.Root("request").AtName("assertions"), types.SetValueMust(
			types.ObjectType{AttrTypes: assertionAttrTypes},
			[]attr.Value{types.ObjectValueMust(assertionAttrTypes, map[string]attr.Value{
				"source":     types.StringValue("STATUS_CODE"),
				"property":   types.StringValue(""),
				"comparison": types.StringValue("EQUALS"),
				"target":     types.StringValue("200"),
			})},
		))
	})

	t.Run("url_monitor with alert channel subscriptions", func(t *testing.T) {
		raw := sdkv2FixtureWith(t, "url_monitor", func(obj map[string]any) {
			obj["alert_channel_subscription"] = []any{map[string]any{"channel_id": 7, "activated": true}}
		})
		state := upgradeSDKv2Raw(t, NewURLMonitorResource(), raw)
		requireStateValue(t, state, path.Root(alertChannelSubscriptionAttributeName), alertChannelSubscriptionsSet(7, true))
	})

	t.Run("url_monitor without optional blocks", func(t *testing.T) {
		state := upgradeSDKv2Fixture(t, NewURLMonitorResource(), "url_monitor_minimal")
		requireStateNull(t, state, path.Root("alert_settings"))
		// SDK v2 stored no subscriptions as an empty alert_channel_subscription
		// list.
		requireStateNull(t, state, path.Root(alertChannelSubscriptionAttributeName))
		// SDK v2 stores no assertions as an empty list, which upgrades to
		// the empty set the attribute defaults to.
		requireStateValue(t, state, path.Root("request").AtName("assertions"), types.SetValueMust(types.ObjectType{AttrTypes: assertionAttrTypes}, []attr.Value{}))
	})

	t.Run("ssl_monitor", func(t *testing.T) {
		// SDK v2 stored request, client_certificate, security_baseline and
		// each baseline rule as lists of at most one element, empty for the
		// rules that are not configured.
		state := upgradeSDKv2Fixture(t, NewSSLMonitorResource(), "ssl_monitor")
		request := path.Root("request")
		requireStateValue(t, state, request.AtName("hostname"), types.StringValue("example.com"))
		requireStateValue(t, state, request.AtName("client_certificate").AtName("mode"), types.StringValue("account_default"))
		requireStateValue(t, state, request.AtName("security_baseline").AtName("enabled"), types.BoolValue(true))
		requireStateValue(t, state, request.AtName("security_baseline").AtName("min_tls_version").AtName("value"), types.StringValue("TLS1.2"))
		requireStateNull(t, state, request.AtName("security_baseline").AtName("known_bad_ca"))
		requireStateValue(t, state, request.AtName("assertions"), types.SetValueMust(types.ObjectType{AttrTypes: assertionAttrTypes}, []attr.Value{}))
		requireStateValue(t, state, path.Root("retry_strategy").AtName("type"), types.StringValue("NO_RETRIES"))
	})

	t.Run("heartbeat_monitor", func(t *testing.T) {
		// SDK v2 stored heartbeat as a list of one element, and
		// alert_settings.ssl_certificates has been removed.
		state := upgradeSDKv2Fixture(t, NewHeartbeatMonitorResource(), "heartbeat_monitor")
		requireStateValue(t, state, path.Root("heartbeat").AtName("period_unit"), types.StringValue("days"))
		requireStateValue(t, state, path.Root("alert_settings").AtName("time_based_escalation").AtName("minutes_failing_threshold"), types.Int64Value(5))
	})

	t.Run("check_group_v2", func(t *testing.T) {
		state := upgradeSDKv2Fixture(t, NewCheckGroupV2Resource(), "check_group_v2")
		requireStateValue(t, state, path.Root("enforce_alert_settings").AtName("alert_settings").AtName("reminders").AtName("interval"), types.Int64Value(5))
		requireStateValue(t, state, path.Root("enforce_retry_strategy").AtName("retry_strategy").AtName("only_on").AtName("network_error"), types.BoolValue(true))
		requireStateValue(t, state, path.Root("enforce_scheduling_strategy").AtName("run_parallel"), types.BoolValue(true))
		requireStateValue(t, state, path.Root("api_check_defaults").AtName("basic_auth").AtName("username"), types.StringValue("u"))
		// SDK v2 stored default_runtime, setup_script and teardown_script as
		// lists of at most one element, empty when not configured, and stored
		// an unset snippet_id as 0.
		requireStateValue(t, state, path.Root("default_runtime").AtName("runtime_id"), types.StringValue("2024.09"))
		requireStateValue(t, state, path.Root("setup_script").AtName("inline_script"), types.StringValue("console.log('x')"))
		requireStateNull(t, state, path.Root("setup_script").AtName("snippet_id"))
		requireStateNull(t, state, path.Root(environmentVariableAttributeName))
		requireStateNull(t, state, path.Root("teardown_script"))
	})

	t.Run("check_group_v2 with default assertions", func(t *testing.T) {
		// SDK v2 stored the assertions of api_check_defaults under the
		// singular name assertion.
		raw := sdkv2FixtureWith(t, "check_group_v2", func(obj map[string]any) {
			defaults := obj["api_check_defaults"].([]any)[0].(map[string]any)
			defaults["assertion"] = []any{map[string]any{"source": "STATUS_CODE", "property": "", "comparison": "EQUALS", "target": "200"}}
		})
		state := upgradeSDKv2Raw(t, NewCheckGroupV2Resource(), raw)
		requireStateValue(t, state, path.Root(apiCheckDefaultsAttributeName).AtName("assertions"), types.SetValueMust(
			types.ObjectType{AttrTypes: assertionAttrTypes},
			[]attr.Value{types.ObjectValueMust(assertionAttrTypes, map[string]attr.Value{
				"source":     types.StringValue("STATUS_CODE"),
				"property":   types.StringValue(""),
				"comparison": types.StringValue("EQUALS"),
				"target":     types.StringValue("200"),
			})},
		))
	})

	t.Run("check_group_v2 with enforced alert channel subscriptions", func(t *testing.T) {
		// SDK v2 stored the subscriptions of enforce_alert_settings under
		// the singular name, empty when there are none.
		subscriptions := path.Root("enforce_alert_settings").AtName(alertChannelSubscriptionAttributeName)
		requireStateNull(t, upgradeSDKv2Fixture(t, NewCheckGroupV2Resource(), "check_group_v2"), subscriptions)

		raw := sdkv2FixtureWith(t, "check_group_v2", func(obj map[string]any) {
			settings := obj["enforce_alert_settings"].([]any)[0].(map[string]any)
			settings["alert_channel_subscription"] = []any{map[string]any{"channel_id": 7, "activated": true}}
		})
		state := upgradeSDKv2Raw(t, NewCheckGroupV2Resource(), raw)
		requireStateValue(t, state, subscriptions, alertChannelSubscriptionsSet(7, true))
	})

	t.Run("playwright_check_suite", func(t *testing.T) {
		state := upgradeSDKv2Fixture(t, NewPlaywrightCheckSuiteResource(), "playwright_check_suite")
		requireStateValue(t, state, path.Root("runtime").AtName("steps").AtName("install").AtName("command"), types.StringValue("npm ci"))
		requireStateValue(t, state, path.Root("runtime").AtName("playwright").AtName("version"), types.StringValue("1.50.0"))
		// SDK v2 stored the devices under the singular name device.
		requireStateValue(t, state, path.Root("runtime").AtName("playwright").AtName("devices"), types.SetValueMust(
			playwrightCheckSuiteRuntimeDeviceElementType,
			[]attr.Value{types.ObjectValueMust(playwrightCheckSuiteRuntimeDeviceAttrTypes, map[string]attr.Value{"type": types.StringValue("chromium")})},
		))
		requireStateValue(t, state, path.Root("bundle").AtName("id"), types.StringValue("b"))
		requireStateValue(t, state, path.Root("bundle").AtName("metadata"), types.StringValue("{}"))
	})

	t.Run("status_page_v3", func(t *testing.T) {
		// SDK v2 stored theme_colors, light and dark as lists of one
		// element.
		state := upgradeSDKv2Fixture(t, NewStatusPageV3Resource(), "status_page_v3")
		themeColors := path.Root("theme_colors")
		requireStateValue(t, state, themeColors.AtName("light").AtName("link_font_color"), types.StringValue("#FF0000"))
		requireStateValue(t, state, themeColors.AtName("dark").AtName("link_font_color"), types.StringValue("#00FF00"))
		requireStateValue(t, state, themeColors.AtName("dark").AtName("border_color"), types.StringValue("#000000"))
	})

	t.Run("status_page_v3 with default colors", func(t *testing.T) {
		// Without custom colors SDK v2 stored theme_colors as [].
		state := upgradeSDKv2Fixture(t, NewStatusPageV3Resource(), "status_page_v3_default_colors")
		requireStateNull(t, state, path.Root("theme_colors"))
	})

	// State refreshed by SDK v2 holds empty values where the framework
	// stores null.
	t.Run("url_monitor after refresh", func(t *testing.T) {
		state := upgradeSDKv2Fixture(t, NewURLMonitorResource(), "url_monitor_read")
		// description is computed, so its empty value is kept.
		requireStateValue(t, state, path.Root("description"), types.StringValue(""))
		requireStateNull(t, state, path.Root("tags"))
		requireStateNull(t, state, path.Root("private_locations"))
		requireStateNull(t, state, path.Root("retry_strategy").AtName("only_on"))
		requireStateValue(t, state, path.Root("name"), types.StringValue("u"))
	})

	t.Run("client_certificate after refresh", func(t *testing.T) {
		state := upgradeSDKv2Fixture(t, NewClientCertificateResource(), "client_certificate_read")
		requireStateNull(t, state, path.Root("path"))
		requireStateNull(t, state, path.Root("trusted_ca"))
		requireStateValue(t, state, path.Root("host"), types.StringValue("*.example.com"))
	})

	t.Run("client_certificate with an empty passphrase", func(t *testing.T) {
		// An empty passphrase is the same as none, so it upgrades like any
		// other empty optional string.
		raw := `{"id":"cc","host":"*.example.com","certificate":"c","private_key":"k","passphrase":"","path":"","trusted_ca":""}`
		state := upgradeSDKv2Raw(t, NewClientCertificateResource(), raw)
		requireStateNull(t, state, path.Root("passphrase"))
		requireStateNull(t, state, path.Root("path"))
	})

	t.Run("alert_channel", func(t *testing.T) {
		// SDK v2 stored every channel type as a list, empty for the types
		// that are not configured, and "" for unset optional strings.
		raw := `{"id":"1","email":[],"slack":[],"sms":[],"call":[],"opsgenie":[],"pagerduty":[],` +
			`"webhook":[{"name":"w","method":"POST","headers":{},"query_parameters":{},"template":"","url":"https://example.com","webhook_secret":"","webhook_type":""}],` +
			`"send_recovery":true,"send_failure":true,"send_degraded":false,"ssl_expiry":false,"ssl_expiry_threshold":30}`
		state := upgradeSDKv2Raw(t, NewAlertChannelResource(), raw)
		requireStateNull(t, state, path.Root("email"))
		requireStateNull(t, state, path.Root("slack_app"))
		requireStateValue(t, state, path.Root("webhook").AtName("name"), types.StringValue("w"))
		requireStateValue(t, state, path.Root("webhook").AtName("headers"), types.MapValueMust(types.StringType, map[string]attr.Value{}))
		requireStateNull(t, state, path.Root("webhook").AtName("template"))
		requireStateNull(t, state, path.Root("webhook").AtName("webhook_secret"))
	})

	t.Run("dns_monitor", func(t *testing.T) {
		// SDK v2 stored request and its name_server as lists, empty for an
		// unconfigured name server.
		upgrade := func(nameServer string) tfsdk.State {
			raw := `{"id":"1","request":[{"record_type":"A","query":"example.com","protocol":"UDP",` +
				`"name_server":` + nameServer + `,"assertion":[]}]}`
			return upgradeSDKv2Raw(t, NewDNSMonitorResource(), raw)
		}
		state := upgrade(`[{"host":"1.1.1.1","port":53}]`)
		requireStateValue(t, state, path.Root("request").AtName("query"), types.StringValue("example.com"))
		requireStateValue(t, state, path.Root("request").AtName("name_server").AtName("port"), types.Int64Value(53))
		requireStateNull(t, upgrade(`[]`), path.Root("request").AtName("name_server"))
	})

	t.Run("status_page", func(t *testing.T) {
		// SDK v2 stored cards and their service attachments as repeated
		// blocks under the singular names.
		raw := `{"id":"1","name":"p","url":"p","custom_domain":"","logo":"","redirect_to":"","favicon":"","default_theme":"AUTO",` +
			`"card":[{"name":"c","service_attachment":[{"service_id":"s1"},{"service_id":"s2"}]}]}`
		state := upgradeSDKv2Raw(t, NewStatusPageResource(), raw)
		attachmentType := types.ObjectType{AttrTypes: statusPageServiceAttachmentAttrTypes}
		attachment := func(id string) attr.Value {
			return types.ObjectValueMust(statusPageServiceAttachmentAttrTypes, map[string]attr.Value{"service_id": types.StringValue(id)})
		}
		requireStateValue(t, state, path.Root("cards"), types.ListValueMust(
			types.ObjectType{AttrTypes: statusPageCardAttrTypes},
			[]attr.Value{types.ObjectValueMust(statusPageCardAttrTypes, map[string]attr.Value{
				"name":                types.StringValue("c"),
				"service_attachments": types.ListValueMust(attachmentType, []attr.Value{attachment("s1"), attachment("s2")}),
			})},
		))
	})

	t.Run("status_page_v3_automation_rule", func(t *testing.T) {
		// SDK v2 stored components as a repeated block under the singular
		// name, empty when there are none.
		upgrade := func(components string) tfsdk.State {
			raw := `{"id":"r","status_page_id":"p","name":"n","enabled":true,"first_update":"a","last_update":"b",` +
				`"notify_subscribers":true,"cool_down_window_minutes":5,"tags":["t"],"component":` + components + `}`
			return upgradeSDKv2Raw(t, NewStatusPageV3AutomationRuleResource(), raw)
		}
		state := upgrade(`[{"component_id":"c","target_impact":"MAJOR_OUTAGE"}]`)
		requireStateValue(t, state, path.Root("components"), types.SetValueMust(
			types.ObjectType{AttrTypes: statusPageV3AutomationRuleComponentAttrTypes},
			[]attr.Value{types.ObjectValueMust(statusPageV3AutomationRuleComponentAttrTypes, map[string]attr.Value{
				"component_id":  types.StringValue("c"),
				"target_impact": types.StringValue("MAJOR_OUTAGE"),
			})},
		))
		requireStateNull(t, upgrade(`[]`), path.Root("components"))
	})

	t.Run("ssl_monitor after refresh", func(t *testing.T) {
		state := upgradeSDKv2Fixture(t, NewSSLMonitorResource(), "ssl_monitor_read")
		requireStateNull(t, state, path.Root("request").AtName("client_certificate").AtName("client_certificate_id"))
		// An unconfigured security baseline, stored as [] by SDK v2.
		requireStateNull(t, state, path.Root("request").AtName("security_baseline"))
	})

	t.Run("maintenance_windows", func(t *testing.T) {
		// SDK v2 stored status_page_visibility as a list of one element and
		// an unset set as [].
		state := upgradeSDKv2Fixture(t, NewMaintenanceWindowResource(), "maintenance_windows")
		visibility := path.Root("status_page_visibility")
		requireStateValue(t, state, visibility.AtName("enabled"), types.BoolValue(true))
		requireStateValue(t, state, visibility.AtName("severity"), types.StringValue("MAJOR"))
		requireStateValue(t, state, visibility.AtName("auto_end"), types.BoolValue(false))
		requireStateValue(t, state, visibility.AtName("status_page_ids"), types.SetValueMust(types.StringType, []attr.Value{types.StringValue("page")}))
		requireStateNull(t, state, visibility.AtName("service_ids"))
	})

	t.Run("maintenance_windows after refresh", func(t *testing.T) {
		state := upgradeSDKv2Fixture(t, NewMaintenanceWindowResource(), "maintenance_window_read")
		requireStateNull(t, state, path.Root("description"))
		requireStateNull(t, state, path.Root("repeat_unit"))
		requireStateNull(t, state, path.Root("repeat_interval"))
		requireStateNull(t, state, path.Root("silence_alerts_tags"))
		// Not configured, stored as [] by SDK v2.
		requireStateNull(t, state, path.Root("status_page_visibility"))
	})

	t.Run("check", func(t *testing.T) {
		// ssl_check has been removed. The deprecated environment_variables
		// map becomes the environment_variables list.
		state := upgradeSDKv2Fixture(t, NewCheckResource(), "check")
		requireStateValue(t, state, path.Root(environmentVariableAttributeName), environmentVariablesList(t, "A", "b"))
		requireStateValue(t, state, path.Root("type"), types.StringValue("API"))
		requireStateValue(t, state, path.Root("double_check"), types.BoolValue(true))
		requireStateNull(t, state, path.Root(checkIntentAttributeName))
		requireStateNull(t, state, path.Root(aiAutoRepairEnabledAttributeName))
		requireStateValue(t, state, path.Root("retry_strategy").AtName("type"), types.StringValue("FIXED"))
		requireSDKv2CheckRequest(t, state)
		requireStateNull(t, state, path.Root(alertChannelSubscriptionAttributeName))
	})

	t.Run("check_browser", func(t *testing.T) {
		// SDK v2 stores the absent request of a browser check as an empty
		// list, which upgrades to a null object.
		state := upgradeSDKv2Fixture(t, NewCheckResource(), "check_browser")
		requireStateNull(t, state, path.Root("request"))
		// The environment_variable list becomes environment_variables.
		requireStateValue(t, state, path.Root(environmentVariableAttributeName), environmentVariablesList(t, "K", "v"))
	})

	t.Run("check_multistep", func(t *testing.T) {
		// SDK v2 stored no environment variables as an empty
		// environment_variable list and a null environment_variables map.
		state := upgradeSDKv2Fixture(t, NewCheckResource(), "check_multistep")
		requireStateNull(t, state, path.Root(environmentVariableAttributeName))
	})
}

// requireSDKv2CheckRequest asserts the request upgraded from the "check"
// fixture: the empty headers and query_parameters maps SDK v2 stored for the
// unset optional maps are null, and the one-element basic_auth list is
// unwrapped into an object.
func requireSDKv2CheckRequest(t *testing.T, state tfsdk.State) {
	t.Helper()
	request := path.Root("request")
	requireStateNull(t, state, request.AtName("headers"))
	requireStateNull(t, state, request.AtName("query_parameters"))
	requireStateValue(t, state, request.AtName("basic_auth").AtName("username"), types.StringValue("u"))
}

// decodeSDKv2TestJSON decodes JSON the way upgradeSDKv2RawState does.
func decodeSDKv2TestJSON(t *testing.T, s string) map[string]any {
	t.Helper()

	decoder := json.NewDecoder(bytes.NewReader([]byte(s)))
	decoder.UseNumber()
	var obj map[string]any
	if err := decoder.Decode(&obj); err != nil {
		t.Fatalf("decoding %s: %v", s, err)
	}
	return obj
}

// requireNormalizedSDKv2Object normalizes the SDK v2 state raw against attrs
// and compares the result with the JSON object want.
func requireNormalizedSDKv2Object(t *testing.T, attrs map[string]schema.Attribute, raw, want string) {
	t.Helper()

	obj := decodeSDKv2TestJSON(t, raw)
	normalizeSDKv2Object(obj, attrs)
	if wantObj := decodeSDKv2TestJSON(t, want); !reflect.DeepEqual(obj, wantObj) {
		got, _ := json.Marshal(obj)
		t.Errorf("got %s, want %s", got, want)
	}
}

// Collections that SDK v2 stored under a singular name move to their plural
// name, at any depth. The schemas are reduced to the attributes involved.
func TestNormalizeSDKv2PluralRenames(t *testing.T) {
	subscriptions := schema.SetNestedAttribute{
		Optional: true,
		NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
			"channel_id": schema.Int64Attribute{Required: true},
			"activated":  schema.BoolAttribute{Required: true},
		}},
	}
	named := schema.ListNestedAttribute{
		Optional: true,
		NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{Required: true},
		}},
	}

	t.Run("top level", func(t *testing.T) {
		requireNormalizedSDKv2Object(t,
			map[string]schema.Attribute{"alert_channel_subscriptions": subscriptions},
			`{"alert_channel_subscription":[{"channel_id":1,"activated":true}]}`,
			`{"alert_channel_subscriptions":[{"channel_id":1,"activated":true}]}`,
		)
	})

	t.Run("an empty collection becomes null", func(t *testing.T) {
		requireNormalizedSDKv2Object(t,
			map[string]schema.Attribute{"alert_channel_subscriptions": subscriptions},
			`{"alert_channel_subscription":[]}`,
			`{"alert_channel_subscriptions":null}`,
		)
	})

	t.Run("inside a list nested attribute", func(t *testing.T) {
		cards := schema.ListNestedAttribute{
			Required: true,
			NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"name":                schema.StringAttribute{Required: true},
				"service_attachments": named,
			}},
		}
		requireNormalizedSDKv2Object(t,
			map[string]schema.Attribute{"cards": cards},
			`{"card":[{"name":"a","service_attachment":[{"name":"s"}]},{"name":"b","service_attachment":[]}]}`,
			`{"cards":[{"name":"a","service_attachments":[{"name":"s"}]},{"name":"b","service_attachments":null}]}`,
		)
	})

	t.Run("inside a single nested attribute", func(t *testing.T) {
		request := schema.SingleNestedAttribute{
			Required: true,
			Attributes: map[string]schema.Attribute{
				"url":        schema.StringAttribute{Required: true},
				"assertions": named,
			},
		}
		requireNormalizedSDKv2Object(t,
			map[string]schema.Attribute{"request": request},
			`{"request":[{"url":"u","assertion":[{"name":"a"}]}]}`,
			`{"request":{"url":"u","assertions":[{"name":"a"}]}}`,
		)
	})

	t.Run("a value under the plural name is kept", func(t *testing.T) {
		requireNormalizedSDKv2Object(t,
			map[string]schema.Attribute{"devices": named},
			`{"devices":[{"name":"new"}],"device":[{"name":"old"}]}`,
			`{"devices":[{"name":"new"}]}`,
		)
	})

	t.Run("a null value under the plural name is replaced", func(t *testing.T) {
		requireNormalizedSDKv2Object(t,
			map[string]schema.Attribute{"devices": named},
			`{"devices":null,"device":[{"name":"old"}]}`,
			`{"devices":[{"name":"old"}]}`,
		)
	})

	t.Run("not renamed where the schema has no plural name", func(t *testing.T) {
		requireNormalizedSDKv2Object(t,
			map[string]schema.Attribute{"device": named},
			`{"device":[{"name":"d"}]}`,
			`{"device":[{"name":"d"}]}`,
		)
	})
}

// The deprecated SDK v2 environment_variables map is folded into the
// environment variable list.
func TestNormalizeSDKv2EnvironmentVariables(t *testing.T) {
	attrs := map[string]schema.Attribute{
		"environment_variables": schema.ListNestedAttribute{
			Optional: true,
			NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"key":    schema.StringAttribute{Required: true},
				"value":  schema.StringAttribute{Required: true},
				"locked": schema.BoolAttribute{Optional: true, Computed: true},
				"secret": schema.BoolAttribute{Optional: true, Computed: true},
			}},
		},
	}
	list := `[{"key":"K","value":"v","locked":true,"secret":false}]`

	for name, c := range map[string]struct{ raw, want string }{
		"map": {
			raw: `{"environment_variables":{"B":"2","A":"1"},"environment_variable":[]}`,
			want: `{"environment_variables":[` +
				`{"key":"A","value":"1","locked":false,"secret":false},` +
				`{"key":"B","value":"2","locked":false,"secret":false}]}`,
		},
		"list and an empty map": {
			raw:  `{"environment_variables":{},"environment_variable":` + list + `}`,
			want: `{"environment_variables":` + list + `}`,
		},
		"list and a null map": {
			raw:  `{"environment_variables":null,"environment_variable":` + list + `}`,
			want: `{"environment_variables":` + list + `}`,
		},
		"neither": {
			raw:  `{"environment_variables":{},"environment_variable":[]}`,
			want: `{"environment_variables":null}`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			requireNormalizedSDKv2Object(t, attrs, c.raw, c.want)
		})
	}
}

// Only the environment_variables map is converted to a list. SDK v2 never
// stored a JSON object at any other list position, so such state fails to
// upgrade instead of being dropped.
func TestUpgradeSDKv2ObjectAtListPosition(t *testing.T) {
	s := schema.Schema{Attributes: map[string]schema.Attribute{
		"cards": schema.ListNestedAttribute{
			Optional: true,
			NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"name": schema.StringAttribute{Required: true},
			}},
		},
	}}
	raw := &tfprotov6.RawState{JSON: []byte(`{"cards":{"name":"a"}}`)}
	if _, err := upgradeSDKv2RawState(context.Background(), raw, s); err == nil {
		t.Fatal("expected an error")
	}
}

// A `moved` block can move a resource whose state the SDK v2 implementation
// wrote, in the same plan that upgrades the provider.
func TestMoveSDKv2State(t *testing.T) {
	ctx := context.Background()

	cases := map[string]struct {
		target     resource.Resource
		sourceType string
		fixture    string
		attr       path.Path
		want       attr.Value
		// check optionally asserts more of the moved state.
		check func(t *testing.T, state tfsdk.State)
	}{
		"checkly_check to checkly_api_check": {
			target:     NewAPICheckResource(),
			sourceType: "checkly_check",
			fixture:    "check",
			attr:       path.Root("retry_strategy").AtName("type"),
			want:       types.StringValue("FIXED"),
			check:      requireSDKv2CheckRequest,
		},
		"checkly_check to checkly_browser_check": {
			target:     NewBrowserCheckResource(),
			sourceType: "checkly_check",
			fixture:    "check_browser",
			attr:       path.Root("script"),
			want:       types.StringValue("console.log(1)"),
			check: func(t *testing.T, state tfsdk.State) {
				requireStateValue(t, state, path.Root(environmentVariableAttributeName), environmentVariablesList(t, "K", "v"))
				// checkly_check has no dependencies, so the moved check has
				// none either.
				requireStateNull(t, state, path.Root(scriptDependenciesAttributeName))
			},
		},
		"checkly_check to checkly_multistep_check": {
			target:     NewMultistepCheckResource(),
			sourceType: "checkly_check",
			fixture:    "check_multistep",
			attr:       path.Root("script"),
			want:       types.StringValue("console.log(2)"),
			check: func(t *testing.T, state tfsdk.State) {
				requireStateNull(t, state, path.Root(scriptDependenciesAttributeName))
			},
		},
		"checkly_tcp_check to checkly_tcp_monitor": {
			target:     NewTCPMonitorResource(),
			sourceType: "checkly_tcp_check",
			fixture:    "tcp_monitor",
			attr:       path.Root("retry_strategy").AtName("type"),
			want:       types.StringValue("FIXED"),
			check: func(t *testing.T, state tfsdk.State) {
				requireStateValue(t, state, path.Root("request").AtName("hostname"), types.StringValue("example.com"))
			},
		},
		"checkly_heartbeat to checkly_heartbeat_monitor": {
			target:     NewHeartbeatMonitorResource(),
			sourceType: "checkly_heartbeat",
			fixture:    "heartbeat_monitor",
			attr:       path.Root("heartbeat").AtName("grace_unit"),
			want:       types.StringValue("hours"),
		},
		"checkly_maintenance_windows to checkly_maintenance_window": {
			target:     NewMaintenanceWindowResource(),
			sourceType: "checkly_maintenance_windows",
			fixture:    "maintenance_windows",
			attr:       path.Root("repeat_unit"),
			want:       types.StringValue("WEEK"),
			check: func(t *testing.T, state tfsdk.State) {
				visibility := path.Root("status_page_visibility")
				requireStateValue(t, state, visibility.AtName("enabled"), types.BoolValue(true))
				requireStateNull(t, state, visibility.AtName("service_ids"))
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			var schemaResp resource.SchemaResponse
			c.target.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

			movers := c.target.(resource.ResourceWithMoveState).MoveState(ctx)
			if len(movers) != 1 {
				t.Fatalf("got %d state movers, want 1", len(movers))
			}

			resp := resource.MoveStateResponse{TargetState: tfsdk.State{Schema: schemaResp.Schema}}
			movers[0].StateMover(ctx, resource.MoveStateRequest{
				SourceProviderAddress: "registry.terraform.io/checkly/checkly",
				SourceTypeName:        c.sourceType,
				SourceSchemaVersion:   sdkv2SchemaVersion,
				SourceRawState:        readSDKv2StateFixture(t, c.fixture),
			}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("move diagnostics: %v", resp.Diagnostics)
			}
			requireStateValue(t, resp.TargetState, path.Root("id"), types.StringValue("123"))
			requireStateValue(t, resp.TargetState, c.attr, c.want)
			// v1 had no intent or automatic repair override, so a check moved
			// from its state has neither.
			if c.sourceType == "checkly_check" {
				requireStateNull(t, resp.TargetState, path.Root(checkIntentAttributeName))
				requireStateNull(t, resp.TargetState, path.Root(aiAutoRepairEnabledAttributeName))
			}
			if c.check != nil {
				c.check(t, resp.TargetState)
			}
		})
	}
}

// Every resource type that the SDK v2 implementation provided must upgrade
// state from sdkv2SchemaVersion.
func TestSDKv2StateUpgradeCoverage(t *testing.T) {
	// Resource types introduced after the SDK v2 implementation.
	newResourceTypes := map[string]bool{
		"checkly_api_check":       true,
		"checkly_browser_check":   true,
		"checkly_multistep_check": true,
	}

	ctx := context.Background()
	p := New("test")()

	var providerMetadata provider.MetadataResponse
	p.Metadata(ctx, provider.MetadataRequest{}, &providerMetadata)

	for _, newResource := range p.Resources(ctx) {
		r := newResource()

		var metadata resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: providerMetadata.TypeName}, &metadata)
		if newResourceTypes[metadata.TypeName] {
			continue
		}

		var schemaResp resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
		if v := schemaResp.Schema.Version; v <= sdkv2SchemaVersion {
			t.Errorf("%s: schema version %d is not newer than the SDK v2 schema version", metadata.TypeName, v)
			continue
		}

		upgradable, ok := r.(resource.ResourceWithUpgradeState)
		if !ok {
			t.Errorf("%s: does not implement UpgradeState", metadata.TypeName)
			continue
		}
		if _, ok := upgradable.UpgradeState(ctx)[sdkv2SchemaVersion]; !ok {
			t.Errorf("%s: has no state upgrader for schema version %d", metadata.TypeName, sdkv2SchemaVersion)
		}
	}
}

// A mover must leave state it does not handle to other movers, and refuse
// moves that would lose data.
func TestMoveSDKv2StateRejections(t *testing.T) {
	ctx := context.Background()

	move := func(t *testing.T, target resource.Resource, req resource.MoveStateRequest) resource.MoveStateResponse {
		t.Helper()

		var schemaResp resource.SchemaResponse
		target.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
		resp := resource.MoveStateResponse{TargetState: tfsdk.State{Schema: schemaResp.Schema}}
		target.(resource.ResourceWithMoveState).MoveState(ctx)[0].StateMover(ctx, req, &resp)
		return resp
	}

	t.Run("a check of another type", func(t *testing.T) {
		resp := move(t, NewAPICheckResource(), resource.MoveStateRequest{
			SourceProviderAddress: "registry.terraform.io/checkly/checkly",
			SourceTypeName:        "checkly_check",
			SourceSchemaVersion:   sdkv2SchemaVersion,
			SourceRawState:        readSDKv2StateFixture(t, "check_browser"),
		})
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected an error")
		}
		if detail := resp.Diagnostics.Errors()[0].Detail(); !strings.Contains(detail, "checkly_browser_check") {
			t.Errorf("error does not name checkly_browser_check: %s", detail)
		}
	})

	for name, req := range map[string]resource.MoveStateRequest{
		"another provider": {
			SourceProviderAddress: "registry.terraform.io/other/example",
			SourceTypeName:        "checkly_check",
			SourceSchemaVersion:   sdkv2SchemaVersion,
			SourceRawState:        readSDKv2StateFixture(t, "check"),
		},
		"another resource type": {
			SourceProviderAddress: "registry.terraform.io/checkly/checkly",
			SourceTypeName:        "checkly_url_monitor",
			SourceSchemaVersion:   sdkv2SchemaVersion,
			SourceRawState:        readSDKv2StateFixture(t, "url_monitor"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			for _, target := range []resource.Resource{NewAPICheckResource(), NewHeartbeatMonitorResource()} {
				resp := move(t, target, req)
				if resp.Diagnostics.HasError() {
					t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
				}
				if !resp.TargetState.Raw.IsNull() {
					t.Errorf("target state was set: %v", resp.TargetState.Raw)
				}
			}
		})
	}

	t.Run("state that cannot be decoded", func(t *testing.T) {
		resp := move(t, NewHeartbeatMonitorResource(), resource.MoveStateRequest{
			SourceProviderAddress: "registry.terraform.io/checkly/checkly",
			SourceTypeName:        "checkly_heartbeat",
			SourceSchemaVersion:   sdkv2SchemaVersion + 1,
		})
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected an error")
		}
	})
}
