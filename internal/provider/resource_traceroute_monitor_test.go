package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/checkly/checkly-go-sdk/v2"
)

// TestTracerouteMonitorRequestFromModelDerivesDefaults asserts that unset
// port and max_unknown_hops values are derived from the current protocol and
// max_hops, mirroring the server's rules, while configured values are sent
// as is. The update endpoint keeps the stored value of an omitted field, so
// sending nothing would keep a value derived for a previous configuration.
func TestTracerouteMonitorRequestFromModelDerivesDefaults(t *testing.T) {
	t.Parallel()

	request := func(protocol string, port, maxHops, maxUnknownHops types.Int64) TracerouteMonitorRequestModel {
		return TracerouteMonitorRequestModel{
			URL:            types.StringValue("api.checklyhq.com"),
			Protocol:       types.StringValue(protocol),
			Port:           port,
			IPFamily:       types.StringValue("IPv4"),
			MaxHops:        maxHops,
			MaxUnknownHops: maxUnknownHops,
			PtrLookup:      types.BoolValue(false),
			Timeout:        types.Int64Unknown(),
			Assertions:     types.SetNull(types.ObjectType{AttrTypes: assertionAttrTypes}),
		}
	}

	tests := []struct {
		name               string
		model              TracerouteMonitorRequestModel
		wantPort           int
		wantMaxUnknownHops int
	}{
		{"TCP", request("TCP", types.Int64Unknown(), types.Int64Value(30), types.Int64Unknown()), 443, 15},
		{"UDP", request("UDP", types.Int64Unknown(), types.Int64Value(10), types.Int64Unknown()), 33434, 10},
		{"SCTP", request("SCTP", types.Int64Unknown(), types.Int64Value(30), types.Int64Unknown()), 33434, 15},
		{"ICMP", request("ICMP", types.Int64Unknown(), types.Int64Value(30), types.Int64Unknown()), 0, 15},
		{"configured", request("UDP", types.Int64Value(8080), types.Int64Value(30), types.Int64Value(5)), 8080, 5},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var diags diag.Diagnostics
			got := tracerouteMonitorRequestFromModel(context.Background(), test.model, &diags)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if got.Port != test.wantPort {
				t.Errorf("port = %d, want %d", got.Port, test.wantPort)
			}
			if got.MaxUnknownHops != test.wantMaxUnknownHops {
				t.Errorf("max_unknown_hops = %d, want %d", got.MaxUnknownHops, test.wantMaxUnknownHops)
			}
			if got.PtrLookup == nil || *got.PtrLookup {
				t.Errorf("ptr_lookup = %v, want an explicit false", got.PtrLookup)
			}
		})
	}
}

func TestAccTracerouteMonitorRequiredFields(t *testing.T) {
	config := `resource "checkly_traceroute_monitor" "test" {}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "name" is required, but no definition was found.`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "request" is required, but no definition was found.`),
		},
	})
}

// TestAccTracerouteMonitorInvalidAssertionSource asserts a mistyped assertion
// source fails at plan time instead of surfacing as an API error at apply time.
func TestAccTracerouteMonitorInvalidAssertionSource(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_traceroute_monitor" "test" {
				  name      = "traceroute-invalid-assertion-source"
				  activated = true
				  frequency = 5
				  locations = ["us-east-1"]
				  request = {
					url = "api.checklyhq.com"
					assertions = [{
					  source     = "PACKETLOSS"
					  comparison = "LESS_THAN"
					  target     = "10"
					}]
				  }
				}
			`,
			ExpectError: regexp.MustCompile(`(?s)request\.assertions\[.*\]\.source\s+value\s+must\s+be\s+one\s+of`),
		},
	})
}

func TestAccTracerouteMonitorBasic(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: tracerouteMonitor_basic,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test",
					"name",
					"Traceroute Monitor 1",
				),
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test",
					"activated",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test",
					"frequency",
					"1",
				),
				resource.TestCheckTypeSetElemAttr(
					"checkly_traceroute_monitor.test",
					"locations.*",
					"us-east-1",
				),
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test",
					"request.url",
					"api.checklyhq.com",
				),
			),
		},
		{
			ResourceName:      "checkly_traceroute_monitor.test",
			ImportState:       true,
			ImportStateVerify: true,
		},
	})
}

func TestAccTracerouteMonitorFull(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: tracerouteMonitor_full,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test",
					"name",
					"tracerouteMonitor_full",
				),
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test",
					"locations.#",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test",
					"request.protocol",
					"TCP",
				),
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test",
					"request.max_hops",
					"30",
				),
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test",
					"request.max_unknown_hops",
					"15",
				),
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test",
					"request.ptr_lookup",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test",
					"request.assertions.#",
					"2",
				),
				resource.TestCheckTypeSetElemNestedAttrs(
					"checkly_traceroute_monitor.test",
					"request.assertions.*",
					map[string]string{
						"source":     "PACKET_LOSS",
						"property":   "",
						"comparison": "LESS_THAN",
						"target":     "10",
					},
				),
			),
		},
		{
			Config:             tracerouteMonitor_full,
			PlanOnly:           true,
			ExpectNonEmptyPlan: false,
		},
	})
}

// TestAccTracerouteMonitorServerComputedDefaults asserts that the
// server-derived request fields follow their determining inputs instead of
// going stale: the API derives port from protocol (443 TCP, 33434 UDP/SCTP,
// none for ICMP) and max_unknown_hops from max_hops (min(15, max_hops)). When
// protocol or max_hops changes while the config leaves the derived field
// unset, the provider must re-request the server default rather than re-send
// the value computed for the previous configuration.
func TestAccTracerouteMonitorServerComputedDefaults(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: tracerouteMonitor_serverComputedTCP,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test",
					"request.port",
					"443",
				),
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test",
					"request.max_unknown_hops",
					"15",
				),
			),
		},
		{
			Config: tracerouteMonitor_serverComputedUDP,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test",
					"request.port",
					"33434",
				),
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test",
					"request.max_unknown_hops",
					"10",
				),
			),
		},
		{
			Config:             tracerouteMonitor_serverComputedUDP,
			PlanOnly:           true,
			ExpectNonEmptyPlan: false,
		},
		{
			Config: tracerouteMonitor_serverComputedICMP,
		},
		{
			Config:             tracerouteMonitor_serverComputedICMP,
			PlanOnly:           true,
			ExpectNonEmptyPlan: false,
		},
	})
}

// TestAccTracerouteMonitorMinimalCleanReplan asserts that a config omitting
// every optional field applies, then re-plans with no diff.
func TestAccTracerouteMonitorMinimalCleanReplan(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: tracerouteMonitor_minimal,
		},
		{
			Config:             tracerouteMonitor_minimal,
			PlanOnly:           true,
			ExpectNonEmptyPlan: false,
		},
	})
}

const tracerouteMonitor_serverComputedTCP = `
	resource "checkly_traceroute_monitor" "test" {
	  name      = "tracerouteMonitor_serverComputed"
	  activated = true
	  frequency = 5
	  locations = ["us-east-1"]
	  request = {
		url      = "api.checklyhq.com"
		max_hops = 30
	  }
	}
`

const tracerouteMonitor_serverComputedUDP = `
	resource "checkly_traceroute_monitor" "test" {
	  name      = "tracerouteMonitor_serverComputed"
	  activated = true
	  frequency = 5
	  locations = ["us-east-1"]
	  request = {
		url      = "api.checklyhq.com"
		protocol = "UDP"
		max_hops = 10
	  }
	}
`

const tracerouteMonitor_serverComputedICMP = `
	resource "checkly_traceroute_monitor" "test" {
	  name      = "tracerouteMonitor_serverComputed"
	  activated = true
	  frequency = 5
	  locations = ["us-east-1"]
	  request = {
		url      = "api.checklyhq.com"
		protocol = "ICMP"
		max_hops = 10
	  }
	}
`

const tracerouteMonitor_basic = `
	resource "checkly_traceroute_monitor" "test" {
	  name      = "Traceroute Monitor 1"
	  activated = true
	  frequency = 1
	  locations = ["us-east-1"]
	  request = {
		url = "api.checklyhq.com"
	  }
	}
`

const tracerouteMonitor_full = `
	resource "checkly_traceroute_monitor" "test" {
	  name      = "tracerouteMonitor_full"
	  activated = true
	  frequency = 5
	  muted     = true
	  locations = [
		"us-east-1",
		"ap-northeast-1",
	  ]
	  request = {
		url              = "api.checklyhq.com"
		protocol         = "TCP"
		port             = 443
		ip_family        = "IPv4"
		max_hops         = 30
		max_unknown_hops = 15
		ptr_lookup       = true
		timeout          = 10

		assertions = [{
		  source     = "PACKET_LOSS"
		  property   = ""
		  comparison = "LESS_THAN"
		  target     = "10"
		}, {
		  source     = "RESPONSE_TIME"
		  property   = "avg"
		  comparison = "LESS_THAN"
		  target     = "2000"
		}]
	  }

	  alert_settings = {
		escalation_type = "RUN_BASED"
		reminders = {
		  amount   = 1
		  interval = 5
		}
		run_based_escalation = {
		  failed_run_threshold = 1
		}
	  }
	}
`

// tracerouteMonitor_minimal omits every optional attribute; only the required
// name/activated/frequency/locations and request.url are set.
const tracerouteMonitor_minimal = `
	resource "checkly_traceroute_monitor" "test" {
	  name      = "traceroute-minimal"
	  activated = true
	  frequency = 1
	  locations = ["us-east-1"]
	  request = {
		url = "api.checklyhq.com"
	  }
	}
`

func TestAccTracerouteMonitorPrivateLocations(t *testing.T) {
	testAccMonitorPrivateLocations(t, monitorPrivateLocationsCase{
		resource: "checkly_traceroute_monitor.test",
		config: func(name, attributes string) string {
			return fmt.Sprintf(`
				resource "checkly_traceroute_monitor" "test" {
					name                      = %q
					frequency                 = 60
					activated                 = false
					locations                 = ["eu-central-1"]
					use_global_alert_settings = true

					request = {
						url = "example.com"
					}
					%s
				}
			`, name, attributes)
		},
		set: func(ctx context.Context, client checkly.Client, id string, privateLocations []string) error {
			monitor, err := client.GetTracerouteMonitor(ctx, id)
			if err != nil {
				return err
			}
			monitor.PrivateLocations = &privateLocations
			_, err = client.UpdateTracerouteMonitor(ctx, id, *monitor)
			return err
		},
		read: func(ctx context.Context, client checkly.Client, id string) (*[]string, error) {
			monitor, err := client.GetTracerouteMonitor(ctx, id)
			if err != nil {
				return nil, err
			}
			return monitor.PrivateLocations, nil
		},
	})
}

// TestAccTracerouteMonitorTimeoutDefault asserts that removing
// request.timeout from the configuration resets it to its default.
func TestAccTracerouteMonitorTimeoutDefault(t *testing.T) {
	config := func(timeout string) string {
		return fmt.Sprintf(`
			resource "checkly_traceroute_monitor" "test" {
			  name      = "tracerouteMonitorTimeoutDefault"
			  activated = false
			  frequency = 60
			  locations = ["us-east-1"]
			  request = {
				url = "api.checklyhq.com"
				%s
			  }
			}
		`, timeout)
	}
	accTestCase(t, []resource.TestStep{
		{
			Config: config(`timeout = 20`),
			Check:  resource.TestCheckResourceAttr("checkly_traceroute_monitor.test", "request.timeout", "20"),
		},
		{
			Config: config(""),
			Check:  resource.TestCheckResourceAttr("checkly_traceroute_monitor.test", "request.timeout", "10"),
		},
		{
			Config:   config(""),
			PlanOnly: true,
		},
	})
}

// TestTracerouteMonitorMaxResponseTimeZero asserts that a max_response_time
// of 0, which means the monitor never fails on response time, round-trips
// through the SDK's separate flag for no failing limit, and that other values
// are sent as the limit.
func TestTracerouteMonitorMaxResponseTimeZero(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		monitor checkly.TracerouteMonitor
		want    int64
	}{
		{"no limit", checkly.TracerouteMonitor{ID: "abc", DegradedResponseTime: 25000, NoMaxResponseTime: true}, 0},
		{"limit", checkly.TracerouteMonitor{ID: "abc", DegradedResponseTime: 10000, MaxResponseTime: 20000}, 20000},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			var diags diag.Diagnostics

			model := TracerouteMonitorResourceModel{
				RetryStrategy: types.ObjectNull(retryStrategyAttrTypes(false)),
				AlertSettings: types.ObjectNull(alertSettingsAttrTypes),
			}
			tracerouteMonitorToModel(ctx, &test.monitor, &model, &diags)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if got := model.MaxResponseTime.ValueInt64(); got != test.want {
				t.Errorf("max_response_time = %d, want %d", got, test.want)
			}

			got := tracerouteMonitorFromModel(ctx, &model, &diags)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if got.NoMaxResponseTime != test.monitor.NoMaxResponseTime || got.MaxResponseTime != test.monitor.MaxResponseTime {
				t.Errorf("NoMaxResponseTime, MaxResponseTime = %v, %d, want %v, %d", got.NoMaxResponseTime, got.MaxResponseTime, test.monitor.NoMaxResponseTime, test.monitor.MaxResponseTime)
			}
		})
	}
}

// TestAccTracerouteMonitorNoMaxResponseTime asserts that updating a monitor
// to max_response_time = 0, which means it never fails on response time,
// stores no limit together with a degraded_response_time above the default
// limit, and that the next plan is empty.
func TestAccTracerouteMonitorNoMaxResponseTime(t *testing.T) {
	config := func(responseTimes string) string {
		return fmt.Sprintf(`
			resource "checkly_traceroute_monitor" "test" {
			  name      = "tracerouteMonitorNoMaxResponseTime"
			  activated = false
			  frequency = 60
			  locations = ["us-east-1"]
			  %s
			  request = {
				url = "api.checklyhq.com"
			  }
			}
		`, responseTimes)
	}
	noMax := `
			  degraded_response_time = 25000
			  max_response_time      = 0`
	accTestCase(t, []resource.TestStep{
		{
			Config: config(""),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_traceroute_monitor.test", "degraded_response_time", "500"),
				resource.TestCheckResourceAttr("checkly_traceroute_monitor.test", "max_response_time", "1000"),
			),
		},
		{
			Config: config(noMax),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_traceroute_monitor.test", "max_response_time", "0"),
				resource.TestCheckResourceAttr("checkly_traceroute_monitor.test", "degraded_response_time", "25000"),
			),
		},
		{
			Config:   config(noMax),
			PlanOnly: true,
		},
	})
}
