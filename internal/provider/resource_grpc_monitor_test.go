package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestValidateGRPCEncoding(t *testing.T) {
	t.Parallel()
	flatBuffers := grpcEncodingConfig{mode: "BEHAVIOR", encoding: "FLATBUFFERS", bfbsContent: true}
	tests := []struct {
		name      string
		config    grpcEncodingConfig
		wantError string
	}{
		{name: "default Protobuf", config: grpcEncodingConfig{mode: "BEHAVIOR", encoding: "PROTOBUF"}},
		{name: "Protobuf with proto file", config: grpcEncodingConfig{mode: "BEHAVIOR", encoding: "PROTOBUF", serviceDefinition: true, protoContent: true}},
		{name: "valid FlatBuffers", config: flatBuffers},
		{name: "health", config: grpcEncodingConfig{mode: "HEALTH", encoding: "PROTOBUF"}},
		{name: "unknown mode", config: grpcEncodingConfig{encoding: "FLATBUFFERS"}},
		{name: "unknown encoding", config: grpcEncodingConfig{mode: "BEHAVIOR", bfbsContent: true}},
		{name: "missing schema", config: grpcEncodingConfig{mode: "BEHAVIOR", encoding: "FLATBUFFERS"}, wantError: "bfbs_content is required"},
		{name: "schema on Protobuf", config: grpcEncodingConfig{mode: "BEHAVIOR", encoding: "PROTOBUF", bfbsContent: true}, wantError: "can only be used when encoding is FLATBUFFERS"},
		{name: "service definition on FlatBuffers", config: grpcEncodingConfig{mode: "BEHAVIOR", encoding: "FLATBUFFERS", bfbsContent: true, serviceDefinition: true}, wantError: "service_definition cannot be used"},
		{name: "proto content on FlatBuffers", config: grpcEncodingConfig{mode: "BEHAVIOR", encoding: "FLATBUFFERS", bfbsContent: true, protoContent: true}, wantError: "proto_content cannot be used"},
		{name: "FlatBuffers in health mode", config: grpcEncodingConfig{mode: "HEALTH", encoding: "FLATBUFFERS"}, wantError: "cannot be used when grpc_mode is HEALTH"},
		{name: "schema in health mode", config: grpcEncodingConfig{mode: "HEALTH", encoding: "PROTOBUF", bfbsContent: true}, wantError: "cannot be used when grpc_mode is HEALTH"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateGRPCEncoding(test.config)
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

func TestConfiguredGRPCValues(t *testing.T) {
	t.Parallel()
	if got := configuredString(types.StringNull(), "PROTOBUF"); got != "PROTOBUF" {
		t.Errorf("null string = %q, want default", got)
	}
	if got := configuredString(types.StringUnknown(), "PROTOBUF"); got != "" {
		t.Errorf("unknown string = %q, want empty", got)
	}
	if got := configuredString(types.StringValue("FLATBUFFERS"), "PROTOBUF"); got != "FLATBUFFERS" {
		t.Errorf("set string = %q, want FLATBUFFERS", got)
	}
	for _, test := range []struct {
		value types.String
		want  bool
	}{
		{types.StringNull(), false},
		{types.StringValue(""), false},
		{types.StringValue("c2NoZW1h"), true},
		{types.StringUnknown(), true},
	} {
		if got := configuredPresence(test.value); got != test.want {
			t.Errorf("configuredPresence(%s) = %v, want %v", test.value, got, test.want)
		}
	}
}

func TestBfbsContentValidation(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		value string
		valid bool
	}{
		{"", true},
		{"c2NoZW1h", true},
		{"not base64!", false},
		{strings.Repeat("A", maxGRPCSchemaContentLength), true},
		{strings.Repeat("A", maxGRPCSchemaContentLength+4), false},
	} {
		req := validator.StringRequest{
			Path:        path.Root("request").AtListIndex(0).AtName("bfbs_content"),
			ConfigValue: types.StringValue(test.value),
		}
		var resp validator.StringResponse
		grpcMonitorBfbsContentValidator{}.ValidateString(context.Background(), req, &resp)
		if resp.Diagnostics.HasError() == test.valid {
			t.Errorf("bfbs_content of length %d: diagnostics = %v, want valid = %v", len(test.value), resp.Diagnostics, test.valid)
		}
	}
}

// TestGRPCMonitorRequestFromModelDropsInapplicableFields asserts the payload
// omits planned fields the configured mode or encoding does not use, such as
// the server-reported `REFLECTION` service definition on a FLATBUFFERS
// monitor.
func TestGRPCMonitorRequestFromModelDropsInapplicableFields(t *testing.T) {
	t.Parallel()
	request := func(mode, encoding string) GRPCMonitorRequestModel {
		return GRPCMonitorRequestModel{
			Host:              types.StringValue("grpc.example.com"),
			Port:              types.Int64Value(443),
			IPFamily:          types.StringValue("IPv4"),
			GRPCMode:          types.StringValue(mode),
			TLS:               types.BoolValue(true),
			SkipSSL:           types.BoolValue(false),
			Timeout:           types.Int64Value(10),
			Encoding:          types.StringValue(encoding),
			Service:           types.StringValue(""),
			ServiceDefinition: types.StringValue("REFLECTION"),
			ProtoContent:      types.StringValue(""),
			BfbsContent:       types.StringValue("c2NoZW1h"),
			Method:            types.StringValue("a.B/C"),
			Message:           types.StringValue(""),
			Metadata:          types.SetNull(types.ObjectType{AttrTypes: grpcMonitorMetadataAttrTypes}),
			Assertion:         types.SetNull(types.ObjectType{AttrTypes: grpcMonitorAssertionAttrTypes}),
		}
	}

	ctx := context.Background()
	var diags diag.Diagnostics

	fb := grpcMonitorRequestFromModel(ctx, request("BEHAVIOR", "FLATBUFFERS"), &diags).GRPCConfig
	if fb.Encoding != "FLATBUFFERS" || fb.BfbsContent != "c2NoZW1h" || fb.ServiceDefinition != "" {
		t.Errorf("FLATBUFFERS config = %+v", fb)
	}
	pb := grpcMonitorRequestFromModel(ctx, request("BEHAVIOR", "PROTOBUF"), &diags).GRPCConfig
	if pb.Encoding != "PROTOBUF" || pb.ServiceDefinition != "REFLECTION" {
		t.Errorf("PROTOBUF config = %+v", pb)
	}
	health := grpcMonitorRequestFromModel(ctx, request("HEALTH", "PROTOBUF"), &diags).GRPCConfig
	if health.Encoding != "" {
		t.Errorf("HEALTH config = %+v", health)
	}
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
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

// TestAccGRPCMonitorFlatBuffers walks one monitor through every encoding and
// service definition transition. The update endpoint keeps stored values for
// omitted fields, so each step that removes an attribute from the config
// asserts the server actually cleared it. The test framework also fails any
// step whose apply is followed by a non-empty plan.
func TestAccGRPCMonitorFlatBuffers(t *testing.T) {
	const name = "checkly_grpc_monitor.test"
	schema1 := base64.StdEncoding.EncodeToString([]byte("schema-1"))
	schema2 := base64.StdEncoding.EncodeToString([]byte("schema-2"))
	accTestCase(t, []resource.TestStep{
		{
			Config: grpcMonitorFlatBuffersConfig(`
				encoding     = "FLATBUFFERS"
				bfbs_content = base64encode("schema-1")
				method       = "example.Greeter/Greet"
			`),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(name, "request.0.encoding", "FLATBUFFERS"),
				resource.TestCheckResourceAttr(name, "request.0.bfbs_content", schema1),
			),
		},
		{
			Config: grpcMonitorFlatBuffersConfig(`
				encoding     = "FLATBUFFERS"
				bfbs_content = base64encode("schema-2")
				method       = "example.Greeter/Greet"
			`),
			Check: resource.TestCheckResourceAttr(name, "request.0.bfbs_content", schema2),
		},
		{
			// Unset encoding and bfbs_content.
			Config: grpcMonitorFlatBuffersConfig(`
				method = "example.Greeter/Greet"
			`),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(name, "request.0.encoding", "PROTOBUF"),
				resource.TestCheckResourceAttr(name, "request.0.bfbs_content", ""),
				resource.TestCheckResourceAttr(name, "request.0.service_definition", "REFLECTION"),
			),
		},
		{
			Config: grpcMonitorFlatBuffersConfig(`
				service_definition = "PROTO_FILE"
				proto_content      = "syntax = \"proto3\";"
				method             = "example.Greeter/Greet"
			`),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(name, "request.0.encoding", "PROTOBUF"),
				resource.TestCheckResourceAttr(name, "request.0.service_definition", "PROTO_FILE"),
			),
		},
		{
			// Unset service_definition and proto_content.
			Config: grpcMonitorFlatBuffersConfig(`
				method = "example.Greeter/Greet"
			`),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(name, "request.0.service_definition", "REFLECTION"),
				resource.TestCheckResourceAttr(name, "request.0.proto_content", ""),
			),
		},
		{
			Config: grpcMonitorFlatBuffersConfig(`
				service_definition = "PROTO_FILE"
				proto_content      = "syntax = \"proto3\";"
				method             = "example.Greeter/Greet"
			`),
			Check: resource.TestCheckResourceAttr(name, "request.0.service_definition", "PROTO_FILE"),
		},
		{
			// Switch from a proto file straight to FlatBuffers, unsetting
			// service_definition and proto_content.
			Config: grpcMonitorFlatBuffersConfig(`
				encoding     = "FLATBUFFERS"
				bfbs_content = base64encode("schema-1")
				method       = "example.Greeter/Greet"
			`),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(name, "request.0.encoding", "FLATBUFFERS"),
				resource.TestCheckResourceAttr(name, "request.0.bfbs_content", schema1),
				resource.TestCheckResourceAttr(name, "request.0.proto_content", ""),
			),
		},
		{
			// Switch to HEALTH mode, unsetting every BEHAVIOR-only attribute.
			Config: grpcMonitorFlatBuffersConfig(`
				grpc_mode = "HEALTH"
			`),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(name, "request.0.grpc_mode", "HEALTH"),
				resource.TestCheckResourceAttr(name, "request.0.encoding", "PROTOBUF"),
				resource.TestCheckResourceAttr(name, "request.0.bfbs_content", ""),
				resource.TestCheckResourceAttr(name, "request.0.method", ""),
			),
		},
		{
			Config: grpcMonitorFlatBuffersConfig(`
				encoding     = "FLATBUFFERS"
				bfbs_content = base64encode("schema-2")
				method       = "example.Greeter/Greet"
			`),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(name, "request.0.grpc_mode", "BEHAVIOR"),
				resource.TestCheckResourceAttr(name, "request.0.encoding", "FLATBUFFERS"),
				resource.TestCheckResourceAttr(name, "request.0.bfbs_content", schema2),
			),
		},
		{
			ResourceName:      name,
			ImportState:       true,
			ImportStateVerify: true,
		},
	})
}

func TestAccGRPCMonitorFlatBuffersInvalidConfig(t *testing.T) {
	tests := []struct {
		name    string
		request string
		error   string
	}{
		{"missing schema", `
			encoding = "FLATBUFFERS"
			method   = "example.Greeter/Greet"
		`, `bfbs_content is required when encoding is FLATBUFFERS`},
		{"schema without FlatBuffers", `
			bfbs_content = base64encode("schema")
			method       = "example.Greeter/Greet"
		`, `bfbs_content can only be used when encoding is FLATBUFFERS`},
		{"service definition with FlatBuffers", `
			encoding           = "FLATBUFFERS"
			bfbs_content       = base64encode("schema")
			service_definition = "REFLECTION"
			method             = "example.Greeter/Greet"
		`, `service_definition cannot be used when encoding is FLATBUFFERS`},
		{"FlatBuffers in health mode", `
			grpc_mode = "HEALTH"
			encoding  = "FLATBUFFERS"
		`, `cannot be used when grpc_mode is\s+HEALTH`},
		{"invalid base64", `
			encoding     = "FLATBUFFERS"
			bfbs_content = "not base64!"
			method       = "example.Greeter/Greet"
		`, `(?s)request\[0\]\.bfbs_content must be\s+a\s+base64\s+string`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			accTestCase(t, []resource.TestStep{{
				Config:      grpcMonitorFlatBuffersConfig(test.request),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(test.error),
			}})
		})
	}
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

// grpcMonitorFlatBuffersConfig renders a deactivated monitor whose request
// block holds the common connection settings plus the given attributes.
func grpcMonitorFlatBuffersConfig(request string) string {
	return fmt.Sprintf(`
		resource "checkly_grpc_monitor" "test" {
		  name      = "grpc-flatbuffers"
		  activated = false
		  frequency = 5
		  locations = ["us-east-1"]

		  request {
			host = "grpc.example.com"
			port = 443
			%s
		  }
		}
	`, request)
}

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
