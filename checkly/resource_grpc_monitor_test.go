package checkly

import (
	"encoding/base64"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func validGRPCConfigMap() tfMap {
	return tfMap{
		"grpc_mode":          "BEHAVIOR",
		"encoding":           "PROTOBUF",
		"bfbs_content":       "",
		"service_definition": "REFLECTION",
		"proto_content":      "",
	}
}

func TestValidateGRPCFlatBuffersConfig(t *testing.T) {
	t.Parallel()
	validBFBS := base64.StdEncoding.EncodeToString([]byte("schema"))
	tests := []struct {
		name      string
		mutate    func(tfMap)
		wantError string
	}{
		{name: "valid FlatBuffers", mutate: func(c tfMap) {
			c["encoding"], c["bfbs_content"], c["service_definition"] = "FLATBUFFERS", validBFBS, ""
		}},
		{name: "missing schema", mutate: func(c tfMap) {
			c["encoding"], c["service_definition"] = "FLATBUFFERS", ""
		}, wantError: "bfbs_content is required"},
		{name: "schema on Protobuf", mutate: func(c tfMap) {
			c["bfbs_content"] = validBFBS
		}, wantError: "can only be used"},
		{name: "invalid base64", mutate: func(c tfMap) {
			c["encoding"], c["bfbs_content"], c["service_definition"] = "FLATBUFFERS", "not base64", ""
		}, wantError: "valid base64"},
		{name: "schema too large", mutate: func(c tfMap) {
			c["encoding"] = "FLATBUFFERS"
			c["bfbs_content"] = strings.Repeat("A", maxGRPCSchemaContentLength+1)
			c["service_definition"] = ""
		}, wantError: "must not exceed"},
		{name: "service definition on FlatBuffers", mutate: func(c tfMap) {
			c["encoding"], c["bfbs_content"] = "FLATBUFFERS", validBFBS
		}, wantError: "service_definition cannot be used"},
		{name: "proto content on FlatBuffers", mutate: func(c tfMap) {
			c["encoding"], c["bfbs_content"], c["service_definition"] = "FLATBUFFERS", validBFBS, ""
			c["proto_content"] = "syntax = \"proto3\";"
		}, wantError: "proto_content cannot be used"},
		{name: "encoding in health mode", mutate: func(c tfMap) {
			c["grpc_mode"] = "HEALTH"
			c["encoding"] = "FLATBUFFERS"
		}, wantError: "cannot be used when grpc_mode is HEALTH"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := validGRPCConfigMap()
			test.mutate(config)
			err := validateGRPCConfig(config)
			if test.wantError == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("error = %v, want it to contain %q", err, test.wantError)
			}
		})
	}
}

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
			Config:      config,
			ExpectError: regexp.MustCompile(`At least 1 "request" blocks are required.`),
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
			ExpectError: regexp.MustCompile(`"request\.0\.assertion\.\d+\.source" must be one of`),
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
				testCheckResourceAttrExpr(
					"checkly_grpc_monitor.test",
					"locations.*",
					"us-east-1",
				),
				testCheckResourceAttrExpr(
					"checkly_grpc_monitor.test",
					"request.*.host",
					"grpcb.in",
				),
				testCheckResourceAttrExpr(
					"checkly_grpc_monitor.test",
					"request.*.port",
					"9000",
				),
				testCheckResourceAttrExpr(
					"checkly_grpc_monitor.test",
					"request.*.grpc_mode",
					"HEALTH",
				),
				testCheckResourceAttrExpr(
					"checkly_grpc_monitor.test",
					"request.*.service",
					"grpcbin.GRPCBin",
				),
			),
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
				testCheckResourceAttrExpr(
					"checkly_grpc_monitor.test",
					`"locations.#"`,
					"2",
				),
				testCheckResourceAttrExpr(
					"checkly_grpc_monitor.test",
					"request.*.grpc_mode",
					"BEHAVIOR",
				),
				testCheckResourceAttrExpr(
					"checkly_grpc_monitor.test",
					"request.*.method",
					"grpcbin.GRPCBin/DummyUnary",
				),
				testCheckResourceAttrExpr(
					"checkly_grpc_monitor.test",
					"request.*.service_definition",
					"REFLECTION",
				),
				testCheckResourceAttrExpr(
					"checkly_grpc_monitor.test",
					"request.*.assertion.#",
					"2",
				),
				testCheckResourceAttrExpr(
					"checkly_grpc_monitor.test",
					"request.*.assertion.*.source",
					"GRPC_STATUS_CODE",
				),
				testCheckResourceAttrExpr(
					"checkly_grpc_monitor.test",
					"request.*.metadata.*.key",
					"x-api-key",
				),
			),
		},
	})
}

func TestAccGRPCMonitorFlatBuffers(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: grpcMonitorFlatBuffers,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_grpc_monitor.test", "request.0.encoding", "FLATBUFFERS"),
				resource.TestCheckResourceAttr("checkly_grpc_monitor.test", "request.0.bfbs_content", "c2NoZW1h"),
				resource.TestCheckNoResourceAttr("checkly_grpc_monitor.test", "request.0.service_definition"),
				resource.TestCheckNoResourceAttr("checkly_grpc_monitor.test", "request.0.proto_content"),
			),
		},
	})
}

// TestAccGRPCMonitorMinimalCleanReplan asserts anti-pattern B is avoided: a
// config omitting every optional field applies, then re-plans with no diff.
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

	  alert_settings {
		escalation_type = "RUN_BASED"
		reminders {
		  amount   = 1
		  interval = 5
		}
		run_based_escalation {
		  failed_run_threshold = 1
		}
	  }
	}
`

const grpcMonitorFlatBuffers = `
	resource "checkly_grpc_monitor" "test" {
	  name      = "grpc-flatbuffers"
	  activated = false
	  frequency = 5
	  locations = ["us-east-1"]

	  request {
		host         = "grpc.example.com"
		port         = 443
		grpc_mode    = "BEHAVIOR"
		encoding     = "FLATBUFFERS"
		bfbs_content = base64encode("schema")
		method       = "example.Greeter/Greet"
		message      = jsonencode({ name = "Checkly" })
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
