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
		{NewDNSMonitorResource(), "checkly_dns_monitor", []string{"request", "name_server", "host"}, map[string]tftypes.Value{"port": tftypes.NewValue(tftypes.Number, 53)}, 1, 100, false},
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

// validateGRPCMonitorMetadata validates a checkly_grpc_monitor
// configuration whose request.metadata is the value that metadata returns
// for the set type, and returns the errors about request.metadata itself.
func validateGRPCMonitorMetadata(t *testing.T, metadata func(setType tftypes.Set) tftypes.Value) []*tfprotov6.Diagnostic {
	t.Helper()

	metadataPath := tftypes.NewAttributePath().WithAttributeName("request").WithAttributeName("metadata")
	errors := validateResourceConfig(t, NewGRPCMonitorResource(), "checkly_grpc_monitor", "request", func(objectType tftypes.Object) tftypes.Value {
		requestType := objectType.AttributeTypes["request"].(tftypes.Object)
		return objectWith(objectType, map[string]tftypes.Value{
			"request": objectWith(requestType, map[string]tftypes.Value{
				"metadata": metadata(requestType.AttributeTypes["metadata"].(tftypes.Set)),
			}),
		})
	})
	return errorsAt(errors, metadataPath)
}

// grpcMetadataElements returns a request.metadata set of one element per
// key and value pair. A key or value is passed to tftypes.NewValue, so nil
// is null and tftypes.UnknownValue is unknown.
func grpcMetadataElements(setType tftypes.Set, pairs ...[2]any) tftypes.Value {
	elementType := setType.ElementType.(tftypes.Object)
	elements := make([]tftypes.Value, len(pairs))
	for i, pair := range pairs {
		elements[i] = objectWith(elementType, map[string]tftypes.Value{
			"key":   tftypes.NewValue(tftypes.String, pair[0]),
			"value": tftypes.NewValue(tftypes.String, pair[1]),
		})
	}
	return tftypes.NewValue(setType, elements)
}

func TestGRPCMonitorMetadataSizeLimit(t *testing.T) {
	for n, valid := range map[int]bool{50: true, 51: false} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			errors := validateGRPCMonitorMetadata(t, func(setType tftypes.Set) tftypes.Value {
				pairs := make([][2]any, n)
				for i := range pairs {
					pairs[i] = [2]any{fmt.Sprintf("key-%d", i), "value"}
				}
				return grpcMetadataElements(setType, pairs...)
			})
			if (len(errors) == 0) != valid {
				t.Errorf("valid = %v, errors: %v", valid, errors)
			}
		})
	}
}

func TestGRPCMonitorMetadataDuplicateKeys(t *testing.T) {
	unknown := tftypes.UnknownValue
	cases := map[string]struct {
		pairs [][2]any
		valid bool
	}{
		"different keys":                   {[][2]any{{"a", "1"}, {"b", "2"}}, true},
		"keys that differ in case":         {[][2]any{{"A", "1"}, {"a", "2"}}, false},
		"header-style keys":                {[][2]any{{"X-Foo", "1"}, {"x-foo", "2"}}, false},
		"the same key with another value":  {[][2]any{{"a", "1"}, {"a", "2"}}, false},
		"the same key with null and empty": {[][2]any{{"a", nil}, {"a", ""}}, false},
		// The API lowercases keys with JavaScript's toLowerCase, which maps
		// U+0130 to "i" followed by U+0307 and a final sigma to "ς".
		"U+0130 and i":                  {[][2]any{{"\u0130", "1"}, {"i", "2"}}, true},
		"U+0130 and i with a dot above": {[][2]any{{"\u0130", "1"}, {"i\u0307", "2"}}, false},
		"a final sigma and ς":           {[][2]any{{"\u0391\u03a3", "1"}, {"\u03b1\u03c2", "2"}}, false},
		"a final sigma and σ":           {[][2]any{{"\u0391\u03a3", "1"}, {"\u03b1\u03c3", "2"}}, true},
		"an unknown key":                {[][2]any{{unknown, "1"}, {"a", "2"}}, true},
		"two unknown keys":              {[][2]any{{unknown, "1"}, {unknown, "2"}}, true},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			errors := validateGRPCMonitorMetadata(t, func(setType tftypes.Set) tftypes.Value {
				return grpcMetadataElements(setType, c.pairs...)
			})
			if (len(errors) == 0) != c.valid {
				t.Errorf("valid = %v, errors: %v", c.valid, errors)
			}
		})
	}

	sets := map[string]func(setType tftypes.Set) tftypes.Value{
		"a null set":     func(setType tftypes.Set) tftypes.Value { return tftypes.NewValue(setType, nil) },
		"an unknown set": func(setType tftypes.Set) tftypes.Value { return tftypes.NewValue(setType, unknown) },
		"an unknown element": func(setType tftypes.Set) tftypes.Value {
			element := grpcMetadataElements(setType, [2]any{"a", "1"})
			var elements []tftypes.Value
			if err := element.As(&elements); err != nil {
				panic(err)
			}
			elements = append(elements, tftypes.NewValue(setType.ElementType, unknown))
			return tftypes.NewValue(setType, elements)
		},
	}
	for name, metadata := range sets {
		t.Run(name, func(t *testing.T) {
			if errors := validateGRPCMonitorMetadata(t, metadata); len(errors) != 0 {
				t.Errorf("errors: %v", errors)
			}
		})
	}
}

func TestGRPCMonitorMetadataDuplicateKeyError(t *testing.T) {
	errors := validateGRPCMonitorMetadata(t, func(setType tftypes.Set) tftypes.Value {
		return grpcMetadataElements(setType, [2]any{"A", "v-secret-1"}, [2]any{"a", "v-secret-2"}, [2]any{"b", "v-secret-3"})
	})
	if len(errors) != 1 {
		t.Fatalf("want one error, got: %v", errors)
	}
	detail := errors[0].Detail
	if !strings.Contains(detail, `"A"`) || !strings.Contains(detail, `"a"`) {
		t.Errorf("the error doesn't name both keys: %s", detail)
	}
	if strings.Contains(detail, "v-secret") {
		t.Errorf("the error repeats a metadata value: %s", detail)
	}
}

func TestGRPCMonitorNULCharacters(t *testing.T) {
	key := map[string]tftypes.Value{"key": tftypes.NewValue(tftypes.String, "key")}
	attributes := []struct {
		names    []string
		siblings map[string]tftypes.Value
	}{
		{[]string{"request", "metadata", "key"}, nil},
		{[]string{"request", "metadata", "value"}, key},
		{[]string{"request", "proto_content"}, nil},
		{[]string{"request", "message"}, nil},
	}
	values := []struct {
		name  string
		value string
		valid bool
	}{
		{"a NUL character", "a\x00b", false},
		{"only a NUL character", "\x00", false},
		// Only NUL is rejected; proto files and messages contain tabs and
		// newlines.
		{"other control characters", "a\x01\t\n", true},
		{"the symbol for NUL", "\u2400", true},
	}
	for _, a := range attributes {
		for _, v := range values {
			t.Run(strings.Join(a.names, ".")+" "+v.name, func(t *testing.T) {
				errors, p := validateStringAt(t, NewGRPCMonitorResource(), "checkly_grpc_monitor", a.names, a.siblings, v.value)
				if (len(errors) == 0) != v.valid {
					t.Errorf("%s: valid = %v, errors: %v", p, v.valid, errors)
				}
			})
		}
	}
}

func TestGRPCMonitorNULCharacterErrorOmitsValue(t *testing.T) {
	cases := []struct {
		names    []string
		siblings map[string]tftypes.Value
		value    string
	}{
		{[]string{"request", "metadata", "value"}, map[string]tftypes.Value{"key": tftypes.NewValue(tftypes.String, "key")}, "secret\x00"},
		{[]string{"request", "proto_content"}, nil, "syntax-secret\x00"},
	}
	for _, c := range cases {
		t.Run(strings.Join(c.names, "."), func(t *testing.T) {
			errors, _ := validateStringAt(t, NewGRPCMonitorResource(), "checkly_grpc_monitor", c.names, c.siblings, c.value)
			if len(errors) != 1 {
				t.Fatalf("want one error, got: %v", errors)
			}
			if detail := errors[0].Detail; !strings.Contains(detail, "NUL") || strings.Contains(detail, "secret") {
				t.Errorf("the error must mention NUL and not repeat the value: %s", detail)
			}
		})
	}
}
