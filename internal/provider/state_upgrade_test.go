package provider

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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
		// assertion list becomes a set of objects.
		requireStateValue(t, state, path.Root("request").AtName("url"), types.StringValue("https://example.com"))
		requireStateValue(t, state, path.Root("request").AtName("assertion"), types.SetValueMust(
			types.ObjectType{AttrTypes: assertionAttrTypes},
			[]attr.Value{types.ObjectValueMust(assertionAttrTypes, map[string]attr.Value{
				"source":     types.StringValue("STATUS_CODE"),
				"property":   types.StringValue(""),
				"comparison": types.StringValue("EQUALS"),
				"target":     types.StringValue("200"),
			})},
		))
	})

	t.Run("url_monitor without optional blocks", func(t *testing.T) {
		state := upgradeSDKv2Fixture(t, NewURLMonitorResource(), "url_monitor_minimal")
		requireStateNull(t, state, path.Root("alert_settings"))
		// SDK v2 stores no assertions as an empty list, which upgrades to
		// the empty set the attribute defaults to.
		requireStateValue(t, state, path.Root("request").AtName("assertion"), types.SetValueMust(types.ObjectType{AttrTypes: assertionAttrTypes}, []attr.Value{}))
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
		requireStateValue(t, state, request.AtName("assertion"), types.SetValueMust(types.ObjectType{AttrTypes: assertionAttrTypes}, []attr.Value{}))
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
		requireStateNull(t, state, path.Root("teardown_script"))
	})

	t.Run("playwright_check_suite", func(t *testing.T) {
		state := upgradeSDKv2Fixture(t, NewPlaywrightCheckSuiteResource(), "playwright_check_suite")
		requireStateValue(t, state, path.Root("runtime").AtName("steps").AtName("install").AtName("command"), types.StringValue("npm ci"))
		requireStateValue(t, state, path.Root("runtime").AtName("playwright").AtName("version"), types.StringValue("1.50.0"))
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
		var schemaResp resource.SchemaResponse
		NewClientCertificateResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
		value, err := upgradeSDKv2RawState(context.Background(), &tfprotov6.RawState{JSON: []byte(raw)}, schemaResp.Schema)
		if err != nil {
			t.Fatal(err)
		}
		state := tfsdk.State{Schema: schemaResp.Schema, Raw: value}
		requireStateNull(t, state, path.Root("passphrase"))
		requireStateNull(t, state, path.Root("path"))
	})

	t.Run("alert_channel", func(t *testing.T) {
		// SDK v2 stored every channel type as a list, empty for the types
		// that are not configured, and "" for unset optional strings.
		raw := `{"id":"1","email":[],"slack":[],"sms":[],"call":[],"opsgenie":[],"pagerduty":[],` +
			`"webhook":[{"name":"w","method":"POST","headers":{},"query_parameters":{},"template":"","url":"https://example.com","webhook_secret":"","webhook_type":""}],` +
			`"send_recovery":true,"send_failure":true,"send_degraded":false,"ssl_expiry":false,"ssl_expiry_threshold":30}`
		var schemaResp resource.SchemaResponse
		NewAlertChannelResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
		value, err := upgradeSDKv2RawState(context.Background(), &tfprotov6.RawState{JSON: []byte(raw)}, schemaResp.Schema)
		if err != nil {
			t.Fatal(err)
		}
		state := tfsdk.State{Schema: schemaResp.Schema, Raw: value}
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
			var schemaResp resource.SchemaResponse
			NewDNSMonitorResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			value, err := upgradeSDKv2RawState(context.Background(), &tfprotov6.RawState{JSON: []byte(raw)}, schemaResp.Schema)
			if err != nil {
				t.Fatal(err)
			}
			return tfsdk.State{Schema: schemaResp.Schema, Raw: value}
		}
		state := upgrade(`[{"host":"1.1.1.1","port":53}]`)
		requireStateValue(t, state, path.Root("request").AtName("query"), types.StringValue("example.com"))
		requireStateValue(t, state, path.Root("request").AtName("name_server").AtName("port"), types.Int64Value(53))
		requireStateNull(t, upgrade(`[]`), path.Root("request").AtName("name_server"))
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
		// ssl_check and environment_variables have been removed.
		state := upgradeSDKv2Fixture(t, NewCheckResource(), "check")
		requireStateValue(t, state, path.Root("type"), types.StringValue("API"))
		requireStateValue(t, state, path.Root("double_check"), types.BoolValue(true))
		requireStateValue(t, state, path.Root("retry_strategy").AtName("type"), types.StringValue("FIXED"))
		requireSDKv2CheckRequest(t, state)
	})

	t.Run("check_browser", func(t *testing.T) {
		// SDK v2 stores the absent request of a browser check as an empty
		// list, which upgrades to a null object.
		state := upgradeSDKv2Fixture(t, NewCheckResource(), "check_browser")
		requireStateNull(t, state, path.Root("request"))
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
		},
		"checkly_check to checkly_multistep_check": {
			target:     NewMultistepCheckResource(),
			sourceType: "checkly_check",
			fixture:    "check_multistep",
			attr:       path.Root("script"),
			want:       types.StringValue("console.log(2)"),
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
