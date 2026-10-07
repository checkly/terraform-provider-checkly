package checkly

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
	if got := configuredString(cty.NullVal(cty.String), "PROTOBUF"); got != "PROTOBUF" {
		t.Errorf("null string = %q, want default", got)
	}
	if got := configuredString(cty.UnknownVal(cty.String), "PROTOBUF"); got != "" {
		t.Errorf("unknown string = %q, want empty", got)
	}
	if got := configuredString(cty.StringVal("FLATBUFFERS"), "PROTOBUF"); got != "FLATBUFFERS" {
		t.Errorf("set string = %q, want FLATBUFFERS", got)
	}
	for _, test := range []struct {
		value cty.Value
		want  bool
	}{
		{cty.NullVal(cty.String), false},
		{cty.StringVal(""), false},
		{cty.StringVal("c2NoZW1h"), true},
		{cty.UnknownVal(cty.String), true},
	} {
		if got := configuredPresence(test.value); got != test.want {
			t.Errorf("configuredPresence(%#v) = %v, want %v", test.value, got, test.want)
		}
	}
}

func TestBfbsContentValidation(t *testing.T) {
	t.Parallel()
	request := resourceGRPCMonitor().Schema["request"].Elem.(*schema.Resource)
	validate := request.Schema["bfbs_content"].ValidateFunc
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
		_, errs := validate(test.value, "bfbs_content")
		if (len(errs) == 0) != test.valid {
			t.Errorf("bfbs_content of length %d: errors = %v, want valid = %v", len(test.value), errs, test.valid)
		}
	}
}

// TestGRPCRequestFromListDropsInapplicableFields asserts the payload omits
// state-held fields the configured mode or encoding does not use, such as the
// server-reported `REFLECTION` service definition on a FLATBUFFERS monitor.
func TestGRPCRequestFromListDropsInapplicableFields(t *testing.T) {
	t.Parallel()
	request := func(mode, encoding string) []any {
		return []any{tfMap{
			"host": "grpc.example.com", "port": 443, "ip_family": "IPv4", "skip_ssl": false, "timeout": 10,
			"assertion": schema.NewSet(schema.HashString, nil),
			"metadata":  schema.NewSet(schema.HashString, nil),
			"grpc_mode": mode, "tls": true, "encoding": encoding, "service": "",
			"service_definition": "REFLECTION", "proto_content": "",
			"bfbs_content": "c2NoZW1h", "method": "a.B/C", "message": "",
		}}
	}

	fb := grpcRequestFromList(request("BEHAVIOR", "FLATBUFFERS")).GRPCConfig
	if fb.Encoding != "FLATBUFFERS" || fb.BfbsContent != "c2NoZW1h" || fb.ServiceDefinition != "" {
		t.Errorf("FLATBUFFERS config = %+v", fb)
	}
	pb := grpcRequestFromList(request("BEHAVIOR", "PROTOBUF")).GRPCConfig
	if pb.Encoding != "PROTOBUF" || pb.ServiceDefinition != "REFLECTION" {
		t.Errorf("PROTOBUF config = %+v", pb)
	}
	health := grpcRequestFromList(request("HEALTH", "PROTOBUF")).GRPCConfig
	if health.Encoding != "" {
		t.Errorf("HEALTH config = %+v", health)
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

// TestAccGRPCMonitorFlatBuffers walks one monitor through every encoding and
// service definition transition. The update endpoint keeps stored values for omitted fields, so
// each step that removes an attribute from the config asserts the server
// actually cleared it. The test framework also fails any step whose apply is
// followed by a non-empty plan.
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
		`, `cannot be used when grpc_mode is HEALTH`},
		{"invalid base64", `
			encoding     = "FLATBUFFERS"
			bfbs_content = "not base64!"
			method       = "example.Greeter/Greet"
		`, `expected "request.0.bfbs_content" to be a base64 string`},
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

// TestAccGRPCMonitorUnsetAllAttributes sets the optional attributes, then
// removes them all and asserts each one is cleared or back at its default. The
// update endpoint keeps stored values for omitted fields, so an attribute
// whose empty value is left out of the payload would otherwise stay set. The
// test framework also fails any step whose plan is not empty after a refresh,
// which catches values the API kept even when the read right after the update
// shows them cleared.
//
// Not covered here: tags, group_order and alert_settings, which cannot be
// removed on any check type yet; locations, which every monitor needs; the
// FlatBuffers-related request attributes, covered by
// TestAccGRPCMonitorFlatBuffers; and frequency_offset, covered by
// TestAccGRPCMonitorUnsetFrequencyOffset.
func TestAccGRPCMonitorUnsetAllAttributes(t *testing.T) {
	const name = "checkly_grpc_monitor.test"
	// Private location slugs are unique per account and limited to 30
	// characters, so a short random suffix avoids collisions between runs.
	slug := fmt.Sprintf("tf-grpc-unset-%d", acctest.RandInt()%100000)
	config := func(monitor, request string) string {
		return grpcMonitorUnsetConfig(slug, monitor, request)
	}
	accTestCase(t, []resource.TestStep{
		{
			Config: config(`
				description            = "description"
				muted                  = true
				should_fail            = true
				run_parallel           = true
				degraded_response_time = 3000
				max_response_time      = 8000
				group_id               = checkly_check_group.test.id
				use_global_alert_settings = true
				private_locations      = [checkly_private_location.test.slug_name]
				runtime_id             = "2023.09"

				alert_channel_subscription {
					channel_id = checkly_alert_channel.test.id
					activated  = true
				}

				retry_strategy {
					type = "LINEAR"
				}

				trigger_incident {
					service_id         = checkly_status_page_service.test.id
					severity           = "MINOR"
					name               = "incident"
					description        = "incident"
					notify_subscribers = false
				}
			`, grpcUnsetMethod+`
				ip_family = "IPv6"
				tls       = false
				skip_ssl  = true
				timeout   = 30
				message   = jsonencode({ name = "Checkly" })

				metadata {
					key   = "x-key"
					value = "value"
				}

				assertion {
					source     = "GRPC_STATUS_CODE"
					comparison = "EQUALS"
					target     = "0"
				}
			`),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(name, "description", "description"),
				resource.TestCheckResourceAttr(name, "alert_channel_subscription.#", "1"),
				resource.TestCheckResourceAttr(name, "trigger_incident.#", "1"),
				resource.TestCheckResourceAttr(name, "private_locations.#", "1"),
				resource.TestCheckResourceAttr(name, "runtime_id", "2023.09"),
				resource.TestCheckResourceAttr(name, "request.0.timeout", "30"),
				resource.TestCheckResourceAttr(name, "request.0.metadata.#", "1"),
				resource.TestCheckResourceAttr(name, "request.0.assertion.#", "1"),
			),
		},
		{
			Config: config("", grpcUnsetMethod),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(name, "description", ""),
				resource.TestCheckResourceAttr(name, "degraded_response_time", "10000"),
				resource.TestCheckResourceAttr(name, "max_response_time", "20000"),
				resource.TestCheckResourceAttr(name, "muted", "false"),
				resource.TestCheckResourceAttr(name, "should_fail", "false"),
				resource.TestCheckResourceAttr(name, "run_parallel", "false"),
				resource.TestCheckResourceAttr(name, "group_id", "0"),
				resource.TestCheckResourceAttr(name, "use_global_alert_settings", "false"),
				resource.TestCheckResourceAttr(name, "alert_channel_subscription.#", "0"),
				resource.TestCheckResourceAttr(name, "retry_strategy.0.type", "NO_RETRIES"),
				resource.TestCheckResourceAttr(name, "trigger_incident.#", "0"),
				resource.TestCheckResourceAttr(name, "private_locations.#", "0"),
				resource.TestCheckResourceAttr(name, "runtime_id", ""),
				resource.TestCheckResourceAttr(name, "request.0.ip_family", "IPv4"),
				resource.TestCheckResourceAttr(name, "request.0.tls", "true"),
				resource.TestCheckResourceAttr(name, "request.0.skip_ssl", "false"),
				resource.TestCheckResourceAttr(name, "request.0.timeout", "60"),
				resource.TestCheckResourceAttr(name, "request.0.message", ""),
				resource.TestCheckResourceAttr(name, "request.0.metadata.#", "0"),
				resource.TestCheckResourceAttr(name, "request.0.assertion.#", "0"),
			),
		},
		{
			Config: config("", grpcUnsetHealthService),
			Check:  resource.TestCheckResourceAttr(name, "request.0.service", "grpc.health.v1.Health"),
		},
		{
			Config: config("", `grpc_mode = "HEALTH"`),
			Check:  resource.TestCheckResourceAttr(name, "request.0.service", ""),
		},
		{
			Config: config("", grpcUnsetHealthService),
			Check:  resource.TestCheckResourceAttr(name, "request.0.service", "grpc.health.v1.Health"),
		},
		{
			// Unset grpc_mode, switching back to BEHAVIOR, which has no service.
			Config: config("", grpcUnsetMethod),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(name, "request.0.grpc_mode", "BEHAVIOR"),
				resource.TestCheckResourceAttr(name, "request.0.service", ""),
			),
		},
	})
}

// TestAccGRPCMonitorUnsetFrequencyOffset asserts removing frequency_offset
// together with a switch to a regular frequency converges. frequency_offset is
// required when frequency is 0 and forbidden otherwise, so it cannot be
// removed on its own. The read discards the offset of a monitor with a regular
// frequency, so this does not show whether the API still stores the old one.
func TestAccGRPCMonitorUnsetFrequencyOffset(t *testing.T) {
	const name = "checkly_grpc_monitor.test"
	config := func(frequency string) string {
		return fmt.Sprintf(`
			resource "checkly_grpc_monitor" "test" {
			  name      = "grpc-unset-frequency-offset"
			  activated = false
			  locations = ["us-east-1"]
			  %s

			  request {
				host   = "grpc.example.com"
				port   = 443
				method = "example.Greeter/Greet"
			  }
			}
		`, frequency)
	}
	accTestCase(t, []resource.TestStep{
		{
			Config: config("frequency = 0\n\t\t\t  frequency_offset = 10"),
			Check:  resource.TestCheckResourceAttr(name, "frequency_offset", "10"),
		},
		{
			Config: config("frequency = 5"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(name, "frequency", "5"),
				resource.TestCheckResourceAttr(name, "frequency_offset", "0"),
			),
		},
	})
}

const (
	grpcUnsetMethod        = `method = "example.Greeter/Greet"`
	grpcUnsetHealthService = `
		grpc_mode = "HEALTH"
		service   = "grpc.health.v1.Health"
	`
)

// grpcMonitorUnsetConfig renders the monitor of
// TestAccGRPCMonitorUnsetAllAttributes with the given monitor and request
// attributes, plus the resources the fully populated monitor references. Those
// stay in every step so removing a reference does not also destroy its target.
func grpcMonitorUnsetConfig(slug, monitor, request string) string {
	return fmt.Sprintf(`
		resource "checkly_check_group" "test" {
		  name        = "grpc-unset"
		  activated   = true
		  concurrency = 1
		  locations   = ["us-east-1"]
		}

		resource "checkly_alert_channel" "test" {
		  email {
			address = "grpc-unset@example.com"
		  }
		}

		resource "checkly_status_page_service" "test" {
		  name = "grpc-unset"
		}

		resource "checkly_private_location" "test" {
		  name      = "grpc-unset"
		  slug_name = "%s"
		  icon      = "bell-fill"
		}

		resource "checkly_grpc_monitor" "test" {
		  name      = "grpc-unset"
		  activated = false
		  frequency = 5
		  locations = ["us-east-1"]
		  %s

		  request {
			host = "grpc.example.com"
			port = 443
			%s
		  }
		}
	`, slug, monitor, request)
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
