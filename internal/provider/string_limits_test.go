package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// configAt returns a value of typ in which only the attribute at the path
// of attribute names is set to value, and that attribute's path. A list or
// set on the way gets one element. siblings sets more attributes of the
// object that holds the attribute.
func configAt(typ tftypes.Type, names []string, value tftypes.Value, siblings map[string]tftypes.Value) (tftypes.Value, *tftypes.AttributePath) {
	if len(names) == 0 {
		return value, tftypes.NewAttributePath()
	}

	var step tftypes.AttributePathStep
	var result tftypes.Value
	var rest *tftypes.AttributePath
	switch typ := typ.(type) {
	case tftypes.Object:
		var child tftypes.Value
		child, rest = configAt(typ.AttributeTypes[names[0]], names[1:], value, siblings)
		values := map[string]tftypes.Value{names[0]: child}
		if len(names) == 1 {
			for name, sibling := range siblings {
				values[name] = sibling
			}
		}
		step, result = tftypes.AttributeName(names[0]), objectWith(typ, values)
	case tftypes.List:
		var element tftypes.Value
		element, rest = configAt(typ.ElementType, names, value, siblings)
		step, result = tftypes.ElementKeyInt(0), tftypes.NewValue(typ, []tftypes.Value{element})
	case tftypes.Set:
		var element tftypes.Value
		element, rest = configAt(typ.ElementType, names, value, siblings)
		step, result = tftypes.ElementKeyValue(element), tftypes.NewValue(typ, []tftypes.Value{element})
	default:
		panic(fmt.Sprintf("no attribute %q in %s", names[0], typ))
	}
	return result, tftypes.NewAttributePathWithSteps(append([]tftypes.AttributePathStep{step}, rest.Steps()...))
}

// errorsAt returns the errors whose path is p.
func errorsAt(errors []*tfprotov6.Diagnostic, p *tftypes.AttributePath) []*tfprotov6.Diagnostic {
	var at []*tfprotov6.Diagnostic
	for _, d := range errors {
		if d.Attribute.Equal(p) {
			at = append(at, d)
		}
	}
	return at
}

// validateStringAt validates a configuration of the resource r of type
// typeName that sets only the string attribute at the path of attribute
// names, to value, and siblings. It returns the errors about that attribute
// and its path.
func validateStringAt(t *testing.T, r resource.Resource, typeName string, names []string, siblings map[string]tftypes.Value, value string) ([]*tfprotov6.Diagnostic, *tftypes.AttributePath) {
	t.Helper()

	var p *tftypes.AttributePath
	errors := validateResourceConfig(t, r, typeName, names[0], func(objectType tftypes.Object) tftypes.Value {
		var config tftypes.Value
		config, p = configAt(objectType, names, tftypes.NewValue(tftypes.String, value), siblings)
		return config
	})
	return errorsAt(errors, p), p
}

func TestAttributeStringLimits(t *testing.T) {
	cases := []struct {
		resource          resource.Resource
		typeName          string
		names             []string
		siblings          map[string]tftypes.Value
		min, max          int
		controlCharacters bool
	}{
		{NewAPICheckResource(), "checkly_api_check", []string{"request", "url"}, nil, 1, 2048, false},
		{NewCheckResource(), "checkly_check", []string{"request", "url"}, nil, 1, 2048, false},
		{NewURLMonitorResource(), "checkly_url_monitor", []string{"request", "url"}, nil, 1, 2048, false},
		{NewTCPMonitorResource(), "checkly_tcp_monitor", []string{"request", "hostname"}, nil, 1, 2048, false},
		{NewTCPCheckResource(), "checkly_tcp_check", []string{"request", "hostname"}, nil, 1, 2048, false},
		{NewICMPMonitorResource(), "checkly_icmp_monitor", []string{"request", "hostname"}, nil, 1, 2048, false},
		{NewGRPCMonitorResource(), "checkly_grpc_monitor", []string{"request", "host"}, nil, 1, 2048, true},
		{NewSSLMonitorResource(), "checkly_ssl_monitor", []string{"request", "hostname"}, nil, 1, 2048, true},
		{NewSSLMonitorResource(), "checkly_ssl_monitor", []string{"request", "server_name"}, nil, 0, 2048, true},
		{NewTracerouteMonitorResource(), "checkly_traceroute_monitor", []string{"request", "url"}, nil, 1, 2048, true},
		{NewDNSMonitorResource(), "checkly_dns_monitor", []string{"request", "query"}, nil, 1, 100, false},
		// The name server host requires its port.
		{NewDNSMonitorResource(), "checkly_dns_monitor", []string{"request", "name_server", "host"}, map[string]tftypes.Value{"port": tftypes.NewValue(tftypes.Number, 53)}, 0, 100, false},
		{NewGRPCMonitorResource(), "checkly_grpc_monitor", []string{"request", "metadata", "key"}, nil, 1, 256, false},
		{NewGRPCMonitorResource(), "checkly_grpc_monitor", []string{"request", "metadata", "value"}, map[string]tftypes.Value{"key": tftypes.NewValue(tftypes.String, "key")}, 0, 8192, false},
		{NewGRPCMonitorResource(), "checkly_grpc_monitor", []string{"request", "proto_content"}, nil, 0, 102400, false},
		{NewGRPCMonitorResource(), "checkly_grpc_monitor", []string{"request", "message"}, nil, 0, 65536, false},
		{NewBrowserCheckResource(), "checkly_browser_check", []string{scriptDependenciesAttributeName, "path"}, map[string]tftypes.Value{"content": tftypes.NewValue(tftypes.String, "content")}, 1, 1000, false},
		{NewMultistepCheckResource(), "checkly_multistep_check", []string{scriptDependenciesAttributeName, "path"}, map[string]tftypes.Value{"content": tftypes.NewValue(tftypes.String, "content")}, 1, 1000, false},
		{NewStatusPageV3AutomationRuleResource(), "checkly_status_page_v3_automation_rule", []string{"name"}, nil, 1, 255, false},
	}

	for _, c := range cases {
		values := []struct {
			name  string
			value string
			valid bool
		}{
			{"empty", "", c.min == 0},
			{"one character", "a", true},
			{"the longest", strings.Repeat("a", c.max), true},
			{"too long", strings.Repeat("a", c.max+1), false},
			{"emoji too long", strings.Repeat(emoji, c.max/2+1), false},
			{"U+0001", "a\x01", !c.controlCharacters},
			{"DEL", "a\x7f", !c.controlCharacters},
			{"the C1 control U+0085", "a\u0085", true},
		}
		for _, v := range values {
			t.Run(c.typeName+"."+strings.Join(c.names, ".")+" "+v.name, func(t *testing.T) {
				errors, p := validateStringAt(t, c.resource, c.typeName, c.names, c.siblings, v.value)
				if (len(errors) == 0) != v.valid {
					t.Errorf("%s: valid = %v, errors: %v", p, v.valid, errors)
				}
			})
		}
	}
}

func TestGRPCMonitorMetadataSizeLimit(t *testing.T) {
	for n, valid := range map[int]bool{50: true, 51: false} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			metadataPath := tftypes.NewAttributePath().WithAttributeName("request").WithAttributeName("metadata")
			errors := validateResourceConfig(t, NewGRPCMonitorResource(), "checkly_grpc_monitor", "request", func(objectType tftypes.Object) tftypes.Value {
				requestType := objectType.AttributeTypes["request"].(tftypes.Object)
				setType := requestType.AttributeTypes["metadata"].(tftypes.Set)
				elements := make([]tftypes.Value, n)
				for i := range elements {
					elements[i] = objectWith(setType.ElementType.(tftypes.Object), map[string]tftypes.Value{
						"key":   tftypes.NewValue(tftypes.String, fmt.Sprintf("key-%d", i)),
						"value": tftypes.NewValue(tftypes.String, "value"),
					})
				}
				return objectWith(objectType, map[string]tftypes.Value{
					"request": objectWith(requestType, map[string]tftypes.Value{
						"metadata": tftypes.NewValue(setType, elements),
					}),
				})
			})
			if errors := errorsAt(errors, metadataPath); (len(errors) == 0) != valid {
				t.Errorf("valid = %v, errors: %v", valid, errors)
			}
		})
	}
}
