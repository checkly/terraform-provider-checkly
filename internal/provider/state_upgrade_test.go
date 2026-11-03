package provider

import (
	"context"
	"os"
	"path/filepath"
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
	})

	t.Run("url_monitor without optional blocks", func(t *testing.T) {
		state := upgradeSDKv2Fixture(t, NewURLMonitorResource(), "url_monitor_minimal")
		requireStateNull(t, state, path.Root("alert_settings"))
	})

	t.Run("ssl_monitor", func(t *testing.T) {
		// client_certificate is a single nested attribute inside the request list.
		state := upgradeSDKv2Fixture(t, NewSSLMonitorResource(), "ssl_monitor")
		requireStateValue(t, state, path.Root("request").AtListIndex(0).AtName("client_certificate").AtName("mode"), types.StringValue("account_default"))
		requireStateValue(t, state, path.Root("retry_strategy").AtName("type"), types.StringValue("NO_RETRIES"))
	})

	t.Run("heartbeat_monitor", func(t *testing.T) {
		// heartbeat is a single nested block, alert_settings.ssl_certificates
		// has been removed.
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
		requireStateValue(t, state, path.Root("default_runtime").AtListIndex(0).AtName("runtime_id"), types.StringValue("2024.09"))
	})

	t.Run("playwright_check_suite", func(t *testing.T) {
		state := upgradeSDKv2Fixture(t, NewPlaywrightCheckSuiteResource(), "playwright_check_suite")
		requireStateValue(t, state, path.Root("runtime").AtName("steps").AtName("install").AtName("command"), types.StringValue("npm ci"))
		requireStateValue(t, state, path.Root("runtime").AtName("playwright").AtName("version"), types.StringValue("1.50.0"))
		requireStateValue(t, state, path.Root("bundle").AtListIndex(0).AtName("id"), types.StringValue("b"))
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

	t.Run("ssl_monitor after refresh", func(t *testing.T) {
		state := upgradeSDKv2Fixture(t, NewSSLMonitorResource(), "ssl_monitor_read")
		requireStateNull(t, state, path.Root("request").AtListIndex(0).AtName("client_certificate").AtName("client_certificate_id"))
	})

	t.Run("maintenance_windows after refresh", func(t *testing.T) {
		state := upgradeSDKv2Fixture(t, NewMaintenanceWindowResource(), "maintenance_window_read")
		requireStateNull(t, state, path.Root("description"))
		requireStateNull(t, state, path.Root("repeat_unit"))
		requireStateNull(t, state, path.Root("repeat_interval"))
		requireStateNull(t, state, path.Root("silence_alerts_tags"))
	})

	t.Run("check", func(t *testing.T) {
		// ssl_check and environment_variables have been removed.
		state := upgradeSDKv2Fixture(t, NewCheckResource(), "check")
		requireStateValue(t, state, path.Root("type"), types.StringValue("API"))
		requireStateValue(t, state, path.Root("double_check"), types.BoolValue(true))
		requireStateValue(t, state, path.Root("retry_strategy").AtName("type"), types.StringValue("FIXED"))
	})
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
	}{
		"checkly_check to checkly_api_check": {
			target:     NewAPICheckResource(),
			sourceType: "checkly_check",
			fixture:    "check",
			attr:       path.Root("retry_strategy").AtName("type"),
			want:       types.StringValue("FIXED"),
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
