package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGRPCMonitorRequiredFields(t *testing.T) {
	config := `resource "checkly_grpc_monitor" "test" {}`
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
				resource "checkly_grpc_monitor" "test" {
					name      = "test-required-fields"
					activated = true
					frequency = 5
				}
			`,
			ExpectError: regexp.MustCompile(`Block request must have a configuration value`),
		},
	})
}

// TestAccGRPCMonitorInvalidAssertionSource asserts a mistyped assertion source
// fails at plan time instead of surfacing as an API error at apply time.
func TestAccGRPCMonitorInvalidAssertionSource(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_grpc_monitor" "test" {
				  name      = "grpc-invalid-assertion-source"
				  activated = true
				  frequency = 5
				  locations = ["us-east-1"]
				  request {
					host = "grpc.example.com"
					port = 443
					assertion {
					  source     = "GRPC_STATUSCODE"
					  comparison = "EQUALS"
					  target     = "0"
					}
				  }
				}
			`,
			ExpectError: regexp.MustCompile(`(?s)request\[0\]\.assertion\[.*\]\.source\s+value\s+must\s+be\s+one\s+of`),
		},
	})
}

func TestAccGRPCMonitorBasic(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: grpcMonitor_basic,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_grpc_monitor.test",
					"name",
					"gRPC Monitor 1",
				),
				resource.TestCheckResourceAttr(
					"checkly_grpc_monitor.test",
					"activated",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_grpc_monitor.test",
					"frequency",
					"1",
				),
				resource.TestCheckTypeSetElemAttr(
					"checkly_grpc_monitor.test",
					"locations.*",
					"us-east-1",
				),
				resource.TestCheckResourceAttr(
					"checkly_grpc_monitor.test",
					"request.0.host",
					"grpcb.in",
				),
				resource.TestCheckResourceAttr(
					"checkly_grpc_monitor.test",
					"request.0.port",
					"9000",
				),
				resource.TestCheckResourceAttr(
					"checkly_grpc_monitor.test",
					"request.0.grpc_mode",
					"HEALTH",
				),
				resource.TestCheckResourceAttr(
					"checkly_grpc_monitor.test",
					"request.0.service",
					"grpcbin.GRPCBin",
				),
			),
		},
		{
			ResourceName:      "checkly_grpc_monitor.test",
			ImportState:       true,
			ImportStateVerify: true,
		},
	})
}

func TestAccGRPCMonitorFull(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: grpcMonitor_full,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_grpc_monitor.test",
					"name",
					"grpcMonitor_full",
				),
				resource.TestCheckResourceAttr(
					"checkly_grpc_monitor.test",
					"degraded_response_time",
					"3000",
				),
				resource.TestCheckResourceAttr(
					"checkly_grpc_monitor.test",
					"max_response_time",
					"8000",
				),
				resource.TestCheckResourceAttr(
					"checkly_grpc_monitor.test",
					"locations.#",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_grpc_monitor.test",
					"request.0.grpc_mode",
					"BEHAVIOR",
				),
				resource.TestCheckResourceAttr(
					"checkly_grpc_monitor.test",
					"request.0.method",
					"grpcbin.GRPCBin/DummyUnary",
				),
				resource.TestCheckResourceAttr(
					"checkly_grpc_monitor.test",
					"request.0.service_definition",
					"REFLECTION",
				),
				resource.TestCheckResourceAttr(
					"checkly_grpc_monitor.test",
					"request.0.assertion.#",
					"2",
				),
				resource.TestCheckTypeSetElemNestedAttrs(
					"checkly_grpc_monitor.test",
					"request.0.assertion.*",
					map[string]string{
						"source":     "GRPC_STATUS_CODE",
						"comparison": "EQUALS",
						"target":     "0",
					},
				),
				resource.TestCheckTypeSetElemNestedAttrs(
					"checkly_grpc_monitor.test",
					"request.0.metadata.*",
					map[string]string{
						"key":   "x-api-key",
						"value": "supersecret",
					},
				),
			),
		},
		{
			Config:             grpcMonitor_full,
			PlanOnly:           true,
			ExpectNonEmptyPlan: false,
		},
	})
}

// TestAccGRPCMonitorMinimalCleanReplan asserts that a config omitting every
// optional field applies, then re-plans with no diff.
func TestAccGRPCMonitorMinimalCleanReplan(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: grpcMonitor_minimal,
		},
		{
			Config:             grpcMonitor_minimal,
			PlanOnly:           true,
			ExpectNonEmptyPlan: false,
		},
	})
}

const grpcMonitor_basic = `
	resource "checkly_grpc_monitor" "test" {
	  name      = "gRPC Monitor 1"
	  activated = true
	  frequency = 1
	  locations = ["us-east-1"]
	  request {
		host      = "grpcb.in"
		port      = 9000
		grpc_mode = "HEALTH"
		service   = "grpcbin.GRPCBin"
	  }
	}
`

const grpcMonitor_full = `
	resource "checkly_grpc_monitor" "test" {
	  name                   = "grpcMonitor_full"
	  activated              = true
	  frequency              = 5
	  muted                  = true
	  degraded_response_time = 3000
	  max_response_time      = 8000
	  locations = [
		"us-east-1",
		"eu-central-1",
	  ]
	  request {
		host               = "grpcb.in"
		port               = 9000
		grpc_mode          = "BEHAVIOR"
		tls                = true
		timeout            = 30
		service_definition = "REFLECTION"
		method             = "grpcbin.GRPCBin/DummyUnary"
		message            = jsonencode({ f_string = "hello" })

		metadata {
		  key   = "x-api-key"
		  value = "supersecret"
		}

		assertion {
		  source     = "GRPC_STATUS_CODE"
		  property   = ""
		  comparison = "EQUALS"
		  target     = "0"
		}

		assertion {
		  source     = "RESPONSE_TIME"
		  property   = ""
		  comparison = "LESS_THAN"
		  target     = "1000"
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

// grpcMonitor_minimal omits every optional attribute. grpc_mode defaults to
// BEHAVIOR, which requires a method for a valid create; everything else is left
// to schema defaults / server computed values.
const grpcMonitor_minimal = `
	resource "checkly_grpc_monitor" "test" {
	  name      = "grpc-minimal"
	  activated = true
	  frequency = 1
	  locations = ["us-east-1"]
	  request {
		host   = "grpcb.in"
		port   = 9000
		method = "grpcbin.GRPCBin/DummyUnary"
	  }
	}
`
