package provider

import (
	"context"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
			Assertion:      types.SetNull(types.ObjectType{AttrTypes: assertionAttrTypes}),
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
			ExpectError: regexp.MustCompile(`The argument "activated" is required, but no definition was found.`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "frequency" is required, but no definition was found.`),
		},
		{
			// Terraform only reports the missing request block once all
			// required attributes are present.
			Config: `
				resource "checkly_traceroute_monitor" "test" {
					name      = "test-required-fields"
					activated = true
					frequency = 5
				}
			`,
			ExpectError: regexp.MustCompile(`Block request must have a configuration value`),
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
				  request {
					url = "api.checklyhq.com"
					assertion {
					  source     = "PACKETLOSS"
					  comparison = "LESS_THAN"
					  target     = "10"
					}
				  }
				}
			`,
			ExpectError: regexp.MustCompile(`(?s)request\[0\]\.assertion\[.*\]\.source\s+value\s+must\s+be\s+one\s+of`),
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
					"request.0.url",
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
					"request.0.protocol",
					"TCP",
				),
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test",
					"request.0.max_hops",
					"30",
				),
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test",
					"request.0.max_unknown_hops",
					"15",
				),
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test",
					"request.0.ptr_lookup",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test",
					"request.0.assertion.#",
					"2",
				),
				resource.TestCheckTypeSetElemNestedAttrs(
					"checkly_traceroute_monitor.test",
					"request.0.assertion.*",
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
					"request.0.port",
					"443",
				),
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test",
					"request.0.max_unknown_hops",
					"15",
				),
			),
		},
		{
			Config: tracerouteMonitor_serverComputedUDP,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test",
					"request.0.port",
					"33434",
				),
				resource.TestCheckResourceAttr(
					"checkly_traceroute_monitor.test",
					"request.0.max_unknown_hops",
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
	  request {
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
	  request {
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
	  request {
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
	  request {
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
	  request {
		url              = "api.checklyhq.com"
		protocol         = "TCP"
		port             = 443
		ip_family        = "IPv4"
		max_hops         = 30
		max_unknown_hops = 15
		ptr_lookup       = true
		timeout          = 10

		assertion {
		  source     = "PACKET_LOSS"
		  property   = ""
		  comparison = "LESS_THAN"
		  target     = "10"
		}

		assertion {
		  source     = "RESPONSE_TIME"
		  property   = "avg"
		  comparison = "LESS_THAN"
		  target     = "2000"
		}
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
	  request {
		url = "api.checklyhq.com"
	  }
	}
`
