package provider

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// tfValue converts v to a value of typ: a map sets the named attributes of
// an object and leaves the others null, a slice holds the elements of a list
// or set, and nil is null. tftypes.UnknownValue is unknown.
func tfValue(typ tftypes.Type, v any) tftypes.Value {
	if v == nil {
		return tftypes.NewValue(typ, nil)
	}
	if v == tftypes.UnknownValue {
		return tftypes.NewValue(typ, tftypes.UnknownValue)
	}

	switch typ := typ.(type) {
	case tftypes.Object:
		attrs := v.(map[string]any)
		values := map[string]tftypes.Value{}
		for name, attrType := range typ.AttributeTypes {
			values[name] = tfValue(attrType, attrs[name])
		}
		for name := range attrs {
			if _, ok := typ.AttributeTypes[name]; !ok {
				panic(fmt.Sprintf("no attribute %q in %s", name, typ))
			}
		}
		return tftypes.NewValue(typ, values)
	case tftypes.List:
		var elements []tftypes.Value
		for _, e := range v.([]any) {
			elements = append(elements, tfValue(typ.ElementType, e))
		}
		return tftypes.NewValue(typ, elements)
	case tftypes.Set:
		var elements []tftypes.Value
		for _, e := range v.([]any) {
			elements = append(elements, tfValue(typ.ElementType, e))
		}
		return tftypes.NewValue(typ, elements)
	}

	switch v := v.(type) {
	case int:
		return tftypes.NewValue(typ, big.NewFloat(float64(v)))
	default:
		return tftypes.NewValue(typ, v)
	}
}

// monitorConfigCase is a configuration of a monitor resource and the error
// its validation should report about the attribute or block named root: a
// substring of the error detail, or "" for none.
type monitorConfigCase struct {
	resource resource.Resource
	typeName string
	config   map[string]any
	root     string
	wantErr  string
}

func runMonitorConfigCases(t *testing.T, cases map[string]monitorConfigCase) {
	t.Helper()
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			config := map[string]any{"name": "test"}
			for k, v := range c.config {
				config[k] = v
			}
			errors := validateResourceConfig(t, c.resource, c.typeName, c.root, func(objectType tftypes.Object) tftypes.Value {
				return tfValue(objectType, config)
			})

			var details []string
			for _, d := range errors {
				details = append(details, d.Detail)
			}
			switch {
			case c.wantErr == "" && len(errors) > 0:
				t.Errorf("unexpected errors: %q", details)
			case c.wantErr != "" && !strings.Contains(strings.Join(details, "\n"), c.wantErr):
				t.Errorf("want an error containing %q, got: %q", c.wantErr, details)
			}
		})
	}
}

func TestHostSchemeOrPort(t *testing.T) {
	cases := map[string]string{
		"example.com":           "",
		"10.0.0.1":              "",
		"::1":                   "",
		"2001:db8::1":           "",
		"[2001:db8::1]":         "",
		"{{HOST}}":              "",
		"https://example.com":   "a scheme",
		"tcp://example.com:80":  "a scheme",
		"example.com:443":       "a port",
		"10.0.0.1:443":          "a port",
		"[2001:db8::1]:443":     "a port",
		"{{HOST}}:{{PORT}}":     "a port",
		"example.com/some/path": "",
	}
	for host, want := range cases {
		if got := hostSchemeOrPort(host); got != want {
			t.Errorf("hostSchemeOrPort(%q) = %q, want %q", host, got, want)
		}
	}
}

func TestNumericAssertionTarget(t *testing.T) {
	packetLoss := numericAssertionTarget{pattern: nonNegativeNumberRegex, bounded: true, min: 0, max: 100, label: "a number between 0 and 100"}
	cases := map[string]string{
		"":      "requires a target",
		"3.5":   "",
		"100":   "",
		"0":     "",
		" 3":    "whitespace",
		"3\n":   "whitespace",
		"101":   "must be a number between 0 and 100",
		"-1":    "must be a number between 0 and 100",
		"three": "must be a number between 0 and 100",
	}
	for target, want := range cases {
		got := packetLoss.problem("PACKET_LOSS", target)
		if (want == "") != (got == "") || !strings.Contains(got, want) {
			t.Errorf("problem(%q) = %q, want %q", target, got, want)
		}
	}
}

func TestQuotedList(t *testing.T) {
	cases := map[string][]string{
		``:                nil,
		`"A"`:             {"A"},
		`"A" or "B"`:      {"A", "B"},
		`"A", "B" or "C"`: {"A", "B", "C"},
		`"EQUALS", "NOT_EQUALS" or ` + `"LESS_THAN"`: {"EQUALS", "NOT_EQUALS", "LESS_THAN"},
	}
	for want, values := range cases {
		if got := quotedList(values); got != want {
			t.Errorf("quotedList(%q) = %s, want %s", values, got, want)
		}
	}
}

func TestValidateGRPCMode(t *testing.T) {
	cases := map[string]struct {
		config  grpcModeConfig
		wantErr string
	}{
		"behavior with method":            {grpcModeConfig{mode: "BEHAVIOR", encoding: "PROTOBUF", serviceDefinition: "REFLECTION", method: valueSet}, ""},
		"behavior without method":         {grpcModeConfig{mode: "BEHAVIOR", encoding: "PROTOBUF", serviceDefinition: "REFLECTION"}, "method is required"},
		"behavior with service":           {grpcModeConfig{mode: "BEHAVIOR", encoding: "PROTOBUF", serviceDefinition: "REFLECTION", method: valueSet, service: valueSet}, "service can only be used"},
		"proto file without content":      {grpcModeConfig{mode: "BEHAVIOR", encoding: "PROTOBUF", serviceDefinition: "PROTO_FILE", method: valueSet}, "proto_content is required"},
		"proto file with content":         {grpcModeConfig{mode: "BEHAVIOR", encoding: "PROTOBUF", serviceDefinition: "PROTO_FILE", method: valueSet, protoContent: valueSet}, ""},
		"flatbuffers proto file":          {grpcModeConfig{mode: "BEHAVIOR", encoding: "FLATBUFFERS", serviceDefinition: "PROTO_FILE", method: valueSet}, ""},
		"unknown encoding proto file":     {grpcModeConfig{mode: "BEHAVIOR", encoding: "", serviceDefinition: "PROTO_FILE", method: valueSet}, ""},
		"health":                          {grpcModeConfig{mode: "HEALTH", serviceDefinition: "REFLECTION", service: valueSet}, ""},
		"health with method":              {grpcModeConfig{mode: "HEALTH", serviceDefinition: "REFLECTION", method: valueSet}, "method cannot be used"},
		"health with message":             {grpcModeConfig{mode: "HEALTH", serviceDefinition: "REFLECTION", message: valueSet}, "message cannot be used"},
		"health with proto content":       {grpcModeConfig{mode: "HEALTH", serviceDefinition: "REFLECTION", protoContent: valueSet}, "proto_content cannot be used"},
		"health with proto file":          {grpcModeConfig{mode: "HEALTH", serviceDefinition: "PROTO_FILE"}, "PROTO_FILE"},
		"unknown mode":                    {grpcModeConfig{mode: "", method: valueSet, service: valueSet}, ""},
		"health with unknown method":      {grpcModeConfig{mode: "HEALTH", serviceDefinition: "REFLECTION", method: valueUnknown, message: valueUnknown, protoContent: valueUnknown}, ""},
		"behavior with unknown service":   {grpcModeConfig{mode: "BEHAVIOR", encoding: "PROTOBUF", serviceDefinition: "REFLECTION", method: valueSet, service: valueUnknown}, ""},
		"behavior with unknown method":    {grpcModeConfig{mode: "BEHAVIOR", encoding: "PROTOBUF", serviceDefinition: "REFLECTION", method: valueUnknown}, ""},
		"proto file with unknown content": {grpcModeConfig{mode: "BEHAVIOR", encoding: "PROTOBUF", serviceDefinition: "PROTO_FILE", method: valueSet, protoContent: valueUnknown}, ""},
		"unknown service definition":      {grpcModeConfig{mode: "HEALTH", serviceDefinition: ""}, ""},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			err := validateGRPCMode(c.config)
			switch {
			case c.wantErr == "" && err != nil:
				t.Errorf("unexpected error: %v", err)
			case c.wantErr != "" && (err == nil || !strings.Contains(err.Error(), c.wantErr)):
				t.Errorf("want an error containing %q, got: %v", c.wantErr, err)
			}
		})
	}
}

func TestMonitorResponseTimeValidation(t *testing.T) {
	runMonitorConfigCases(t, map[string]monitorConfigCase{
		"grpc degraded 0":             {NewGRPCMonitorResource(), "checkly_grpc_monitor", map[string]any{"degraded_response_time": 0}, "degraded_response_time", "between 1 and 180000"},
		"grpc max 0":                  {NewGRPCMonitorResource(), "checkly_grpc_monitor", map[string]any{"max_response_time": 0}, "max_response_time", "between 1 and 180000"},
		"ssl degraded 0":              {NewSSLMonitorResource(), "checkly_ssl_monitor", map[string]any{"degraded_response_time": 0}, "degraded_response_time", "between 1 and 30000"},
		"ssl max 0":                   {NewSSLMonitorResource(), "checkly_ssl_monitor", map[string]any{"max_response_time": 0}, "max_response_time", "between 1 and 30000"},
		"traceroute degraded 0":       {NewTracerouteMonitorResource(), "checkly_traceroute_monitor", map[string]any{"degraded_response_time": 0}, "degraded_response_time", "between 1 and 30000"},
		"traceroute max 0":            {NewTracerouteMonitorResource(), "checkly_traceroute_monitor", map[string]any{"max_response_time": 0}, "max_response_time", ""},
		"traceroute max 0 degraded":   {NewTracerouteMonitorResource(), "checkly_traceroute_monitor", map[string]any{"degraded_response_time": 30000, "max_response_time": 0}, "max_response_time", ""},
		"traceroute max -1":           {NewTracerouteMonitorResource(), "checkly_traceroute_monitor", map[string]any{"max_response_time": -1}, "max_response_time", "between 0 and 30000"},
		"tcp 45000":                   {NewTCPMonitorResource(), "checkly_tcp_monitor", map[string]any{"degraded_response_time": 45000, "max_response_time": 45000}, "max_response_time", ""},
		"tcp 45001":                   {NewTCPMonitorResource(), "checkly_tcp_monitor", map[string]any{"max_response_time": 45001}, "max_response_time", "between 0 and 45000"},
		"tcp check 45000":             {NewTCPCheckResource(), "checkly_tcp_check", map[string]any{"degraded_response_time": 45000}, "degraded_response_time", ""},
		"url 45000":                   {NewURLMonitorResource(), "checkly_url_monitor", map[string]any{"degraded_response_time": 45000, "max_response_time": 45000}, "max_response_time", ""},
		"url 45001":                   {NewURLMonitorResource(), "checkly_url_monitor", map[string]any{"degraded_response_time": 45001}, "degraded_response_time", "between 0 and 45000"},
		"api check 45000":             {NewAPICheckResource(), "checkly_api_check", map[string]any{"degraded_response_time": 45000, "max_response_time": 45000}, "max_response_time", ""},
		"api check 45001":             {NewAPICheckResource(), "checkly_api_check", map[string]any{"max_response_time": 45001}, "max_response_time", "between 0 and 45000"},
		"check 45000":                 {NewCheckResource(), "checkly_check", map[string]any{"degraded_response_time": 45000, "max_response_time": 45000}, "max_response_time", ""},
		"check 45001":                 {NewCheckResource(), "checkly_check", map[string]any{"degraded_response_time": 45001}, "degraded_response_time", "between 0 and 45000"},
		"ssl max below degraded":      {NewSSLMonitorResource(), "checkly_ssl_monitor", map[string]any{"degraded_response_time": 5000, "max_response_time": 4000}, "max_response_time", "max_response_time (4000) must be greater than or equal to degraded_response_time (5000)"},
		"ssl max below default":       {NewSSLMonitorResource(), "checkly_ssl_monitor", map[string]any{"max_response_time": 5000}, "max_response_time", "degraded_response_time (10000, the default)"},
		"ssl degraded above default":  {NewSSLMonitorResource(), "checkly_ssl_monitor", map[string]any{"degraded_response_time": 25000}, "max_response_time", "max_response_time (20000, the default)"},
		"ssl equal":                   {NewSSLMonitorResource(), "checkly_ssl_monitor", map[string]any{"degraded_response_time": 5000, "max_response_time": 5000}, "max_response_time", ""},
		"ssl unknown":                 {NewSSLMonitorResource(), "checkly_ssl_monitor", map[string]any{"degraded_response_time": tftypes.UnknownValue, "max_response_time": 1}, "max_response_time", ""},
		"traceroute max below":        {NewTracerouteMonitorResource(), "checkly_traceroute_monitor", map[string]any{"degraded_response_time": 5000, "max_response_time": 4000}, "max_response_time", "must be greater than or equal to degraded_response_time"},
		"traceroute defaults":         {NewTracerouteMonitorResource(), "checkly_traceroute_monitor", map[string]any{}, "max_response_time", ""},
		"icmp max below degraded":     {NewICMPMonitorResource(), "checkly_icmp_monitor", map[string]any{"degraded_packet_loss_threshold": 50, "max_packet_loss_threshold": 40}, "max_packet_loss_threshold", "must be greater than or equal to degraded_packet_loss_threshold"},
		"icmp degraded above default": {NewICMPMonitorResource(), "checkly_icmp_monitor", map[string]any{"degraded_packet_loss_threshold": 30}, "max_packet_loss_threshold", "max_packet_loss_threshold (20, the default)"},
		"icmp ordered":                {NewICMPMonitorResource(), "checkly_icmp_monitor", map[string]any{"degraded_packet_loss_threshold": 30, "max_packet_loss_threshold": 30}, "max_packet_loss_threshold", ""},
	})
}

func TestMonitorRequestValidation(t *testing.T) {
	tcpRequest := func(extra map[string]any) map[string]any {
		request := map[string]any{"hostname": "example.com", "port": 80}
		for k, v := range extra {
			request[k] = v
		}
		return map[string]any{"request": request}
	}
	traceroute := func(extra map[string]any) map[string]any {
		request := map[string]any{"url": "example.com"}
		for k, v := range extra {
			request[k] = v
		}
		return map[string]any{"request": request}
	}
	dnsNameServer := func(host, port any) map[string]any {
		return map[string]any{"request": map[string]any{"record_type": "A", "query": "example.com", "name_server": map[string]any{"host": host, "port": port}}}
	}
	sslCertificate := func(mode, id any) map[string]any {
		return map[string]any{"request": map[string]any{"hostname": "example.com", "client_certificate": map[string]any{"mode": mode, "client_certificate_id": id}}}
	}
	grpc := func(extra map[string]any) map[string]any {
		request := map[string]any{"host": "example.com", "port": 443}
		for k, v := range extra {
			request[k] = v
		}
		return map[string]any{"request": request}
	}
	heartbeat := func(extra map[string]any) map[string]any {
		hb := map[string]any{"period": 1, "period_unit": "hours", "grace": 1, "grace_unit": "hours"}
		for k, v := range extra {
			hb[k] = v
		}
		return map[string]any{"heartbeat": hb}
	}
	runMonitorConfigCases(t, map[string]monitorConfigCase{
		"tcp port 65535":                        {NewTCPMonitorResource(), "checkly_tcp_monitor", tcpRequest(map[string]any{"port": 65535}), "request", ""},
		"tcp port 70000":                        {NewTCPMonitorResource(), "checkly_tcp_monitor", tcpRequest(map[string]any{"port": 70000}), "request", "between 1 and 65535"},
		"tcp port 0":                            {NewTCPMonitorResource(), "checkly_tcp_monitor", tcpRequest(map[string]any{"port": 0}), "request", "between 1 and 65535"},
		"tcp hostname with scheme":              {NewTCPMonitorResource(), "checkly_tcp_monitor", tcpRequest(map[string]any{"hostname": "tcp://example.com"}), "request", "but it has a scheme"},
		"tcp hostname with port":                {NewTCPMonitorResource(), "checkly_tcp_monitor", tcpRequest(map[string]any{"hostname": "example.com:80"}), "request", "but it has a port"},
		"tcp hostname ipv6":                     {NewTCPMonitorResource(), "checkly_tcp_monitor", tcpRequest(map[string]any{"hostname": "2001:db8::1"}), "request", ""},
		"ssl hostname with scheme":              {NewSSLMonitorResource(), "checkly_ssl_monitor", map[string]any{"request": map[string]any{"hostname": "https://example.com"}}, "request", "but it has a scheme"},
		"grpc host with port":                   {NewGRPCMonitorResource(), "checkly_grpc_monitor", grpc(map[string]any{"host": "example.com:443", "method": "a.B/C"}), "request", "but it has a port"},
		"icmp hostname with scheme":             {NewICMPMonitorResource(), "checkly_icmp_monitor", map[string]any{"request": map[string]any{"hostname": "https://example.com"}}, "request", "but it has a scheme"},
		"icmp hostname with port":               {NewICMPMonitorResource(), "checkly_icmp_monitor", map[string]any{"request": map[string]any{"hostname": "example.com:443"}}, "request", "but it has a port"},
		"icmp hostname":                         {NewICMPMonitorResource(), "checkly_icmp_monitor", map[string]any{"request": map[string]any{"hostname": "example.com"}}, "request", ""},
		"traceroute url with scheme":            {NewTracerouteMonitorResource(), "checkly_traceroute_monitor", traceroute(map[string]any{"url": "https://example.com"}), "request", "but it has a scheme"},
		"traceroute icmp":                       {NewTracerouteMonitorResource(), "checkly_traceroute_monitor", traceroute(map[string]any{"protocol": "ICMP"}), "request", ""},
		"traceroute icmp with port":             {NewTracerouteMonitorResource(), "checkly_traceroute_monitor", traceroute(map[string]any{"protocol": "ICMP", "port": 443}), "request", `cannot be set when request.protocol is "ICMP"`},
		"traceroute tcp with port":              {NewTracerouteMonitorResource(), "checkly_traceroute_monitor", traceroute(map[string]any{"protocol": "TCP", "port": 443}), "request", ""},
		"traceroute unknown hops":               {NewTracerouteMonitorResource(), "checkly_traceroute_monitor", traceroute(map[string]any{"max_hops": 10}), "request", ""},
		"traceroute unknown hops above":         {NewTracerouteMonitorResource(), "checkly_traceroute_monitor", traceroute(map[string]any{"max_hops": 10, "max_unknown_hops": 15}), "request", "request.max_hops (10) must be greater than or equal to request.max_unknown_hops (15)"},
		"traceroute unknown hops above default": {NewTracerouteMonitorResource(), "checkly_traceroute_monitor", traceroute(map[string]any{"max_unknown_hops": 30}), "request", ""},
		"traceroute unknown request":            {NewTracerouteMonitorResource(), "checkly_traceroute_monitor", map[string]any{"request": tftypes.UnknownValue}, "request", ""},
		"dns name server":                       {NewDNSMonitorResource(), "checkly_dns_monitor", dnsNameServer("1.1.1.1", 53), "request", ""},
		"dns name server empty host":            {NewDNSMonitorResource(), "checkly_dns_monitor", dnsNameServer("", 53), "request", "1 to 100 UTF-16 code units"},
		"dns name server port 70000":            {NewDNSMonitorResource(), "checkly_dns_monitor", dnsNameServer("1.1.1.1", 70000), "request", "between 1 and 65535"},
		"dns https record":                      {NewDNSMonitorResource(), "checkly_dns_monitor", map[string]any{"request": map[string]any{"record_type": "HTTPS", "query": "example.com"}}, "request", ""},
		"ssl explicit certificate":              {NewSSLMonitorResource(), "checkly_ssl_monitor", sslCertificate("explicit", "22868839-8450-4010-9241-1ea83a2e425f"), "request", ""},
		"ssl explicit no certificate":           {NewSSLMonitorResource(), "checkly_ssl_monitor", sslCertificate("explicit", nil), "request", "client_certificate_id is required"},
		"ssl explicit empty certificate":        {NewSSLMonitorResource(), "checkly_ssl_monitor", sslCertificate("explicit", ""), "request", "client_certificate_id is required"},
		"ssl explicit unknown certificate":      {NewSSLMonitorResource(), "checkly_ssl_monitor", sslCertificate("explicit", tftypes.UnknownValue), "request", ""},
		"ssl auto":                              {NewSSLMonitorResource(), "checkly_ssl_monitor", sslCertificate("auto", nil), "request", ""},
		"grpc behavior without method":          {NewGRPCMonitorResource(), "checkly_grpc_monitor", grpc(nil), "request", "method is required"},
		"grpc health with method":               {NewGRPCMonitorResource(), "checkly_grpc_monitor", grpc(map[string]any{"grpc_mode": "HEALTH", "method": "a.B/C"}), "request", "method cannot be used"},
		"grpc health with unknown method":       {NewGRPCMonitorResource(), "checkly_grpc_monitor", grpc(map[string]any{"grpc_mode": "HEALTH", "method": tftypes.UnknownValue, "message": tftypes.UnknownValue}), "request", ""},
		"grpc behavior with unknown method":     {NewGRPCMonitorResource(), "checkly_grpc_monitor", grpc(map[string]any{"method": tftypes.UnknownValue, "service": tftypes.UnknownValue}), "request", ""},
		"grpc health":                           {NewGRPCMonitorResource(), "checkly_grpc_monitor", grpc(map[string]any{"grpc_mode": "HEALTH", "service": "svc"}), "request", ""},
		"heartbeat":                             {NewHeartbeatMonitorResource(), "checkly_heartbeat_monitor", heartbeat(nil), "heartbeat", ""},
		"heartbeat ping token":                  {NewHeartbeatMonitorResource(), "checkly_heartbeat_monitor", heartbeat(map[string]any{"ping_token": "22868839-8450-4010-9241-1ea83a2e425f"}), "heartbeat", ""},
		"heartbeat invalid ping token":          {NewHeartbeatMonitorResource(), "checkly_heartbeat_monitor", heartbeat(map[string]any{"ping_token": "my-token"}), "heartbeat", "must be a lowercase UUID"},
		"heartbeat uppercase ping token":        {NewHeartbeatMonitorResource(), "checkly_heartbeat_monitor", heartbeat(map[string]any{"ping_token": "22868839-8450-4010-9241-1EA83A2E425F"}), "heartbeat", "must be a lowercase UUID"},
		"heartbeat empty ping token":            {NewHeartbeatMonitorResource(), "checkly_heartbeat_monitor", heartbeat(map[string]any{"ping_token": ""}), "heartbeat", "must be a lowercase UUID"},
		"heartbeat alias ping token":            {NewHeartbeatResource(), "checkly_heartbeat", heartbeat(map[string]any{"ping_token": "my-token"}), "heartbeat", "must be a lowercase UUID"},
		"heartbeat negative grace":              {NewHeartbeatMonitorResource(), "checkly_heartbeat_monitor", heartbeat(map[string]any{"grace": -1}), "heartbeat", "at least 0"},
		"heartbeat zero grace":                  {NewHeartbeatMonitorResource(), "checkly_heartbeat_monitor", heartbeat(map[string]any{"grace": 0}), "heartbeat", ""},
	})
}

// The error about an invalid ping token must not repeat the token, which is
// sensitive.
func TestHeartbeatPingTokenErrorHidesValue(t *testing.T) {
	errors := validateResourceConfig(t, NewHeartbeatMonitorResource(), "checkly_heartbeat_monitor", "heartbeat", func(objectType tftypes.Object) tftypes.Value {
		return tfValue(objectType, map[string]any{"name": "test", "heartbeat": map[string]any{"period": 1, "period_unit": "hours", "grace": 1, "grace_unit": "hours", "ping_token": "secret-token"}})
	})
	if len(errors) == 0 {
		t.Fatal("want an error")
	}
	for _, d := range errors {
		if strings.Contains(d.Summary+d.Detail, "secret-token") {
			t.Errorf("error repeats the token: %s: %s", d.Summary, d.Detail)
		}
	}
}

func TestMonitorAssertionValidation(t *testing.T) {
	assertion := func(source, property, comparison, target any) map[string]any {
		return map[string]any{"source": source, "property": property, "comparison": comparison, "target": target}
	}
	tcp := func(a map[string]any) map[string]any {
		return map[string]any{"request": map[string]any{"hostname": "example.com", "port": 80, "assertions": []any{a}}}
	}
	icmp := func(a map[string]any) map[string]any {
		return map[string]any{"request": map[string]any{"hostname": "example.com", "assertions": []any{a}}}
	}
	dns := func(a map[string]any) map[string]any {
		return map[string]any{"request": map[string]any{"record_type": "A", "query": "example.com", "assertions": []any{a}}}
	}
	traceroute := func(a map[string]any) map[string]any {
		return map[string]any{"request": map[string]any{"url": "example.com", "assertions": []any{a}}}
	}
	grpc := func(a map[string]any) map[string]any {
		return map[string]any{"request": map[string]any{"host": "example.com", "port": 443, "method": "a.B/C", "assertions": []any{a}}}
	}
	runMonitorConfigCases(t, map[string]monitorConfigCase{
		"tcp response data contains":           {NewTCPMonitorResource(), "checkly_tcp_monitor", tcp(assertion("RESPONSE_DATA", nil, "CONTAINS", "ok")), "request", ""},
		"tcp response data has key":            {NewTCPMonitorResource(), "checkly_tcp_monitor", tcp(assertion("RESPONSE_DATA", nil, "HAS_KEY", "ok")), "request", "The comparison of a RESPONSE_DATA assertion must be one of"},
		"tcp response time contains":           {NewTCPMonitorResource(), "checkly_tcp_monitor", tcp(assertion("RESPONSE_TIME", nil, "CONTAINS", "1")), "request", "The comparison of a RESPONSE_TIME assertion must be one of"},
		"tcp invalid source":                   {NewTCPMonitorResource(), "checkly_tcp_monitor", tcp(assertion("TEXT_BODY", nil, "EQUALS", "1")), "request", "value must be one of"},
		"icmp latency":                         {NewICMPMonitorResource(), "checkly_icmp_monitor", icmp(assertion("LATENCY", "avg", "LESS_THAN", "100")), "request", ""},
		"icmp latency without property":        {NewICMPMonitorResource(), "checkly_icmp_monitor", icmp(assertion("LATENCY", nil, "LESS_THAN", "100")), "request", "A LATENCY assertion requires a property"},
		"icmp latency invalid property":        {NewICMPMonitorResource(), "checkly_icmp_monitor", icmp(assertion("LATENCY", "median", "LESS_THAN", "100")), "request", "A LATENCY assertion requires a property"},
		"icmp latency is empty":                {NewICMPMonitorResource(), "checkly_icmp_monitor", icmp(assertion("LATENCY", "avg", "IS_EMPTY", nil)), "request", "The comparison of a LATENCY assertion must be one of"},
		"icmp latency no target":               {NewICMPMonitorResource(), "checkly_icmp_monitor", icmp(assertion("LATENCY", "avg", "LESS_THAN", nil)), "request", "A LATENCY assertion requires a target, which must be a non-negative number of milliseconds"},
		"icmp latency text target":             {NewICMPMonitorResource(), "checkly_icmp_monitor", icmp(assertion("LATENCY", "avg", "LESS_THAN", "fast")), "request", "must be a non-negative number of milliseconds"},
		"icmp latency padded target":           {NewICMPMonitorResource(), "checkly_icmp_monitor", icmp(assertion("LATENCY", "avg", "LESS_THAN", "100 ")), "request", "whitespace"},
		"icmp latency decimal target":          {NewICMPMonitorResource(), "checkly_icmp_monitor", icmp(assertion("LATENCY", "avg", "GREATER_THAN", "0.5")), "request", ""},
		"icmp json no target":                  {NewICMPMonitorResource(), "checkly_icmp_monitor", icmp(assertion("JSON_RESPONSE", "$.x", "EQUALS", nil)), "request", ""},
		"tcp response time":                    {NewTCPMonitorResource(), "checkly_tcp_monitor", tcp(assertion("RESPONSE_TIME", nil, "LESS_THAN", "100")), "request", ""},
		"tcp response time no target":          {NewTCPMonitorResource(), "checkly_tcp_monitor", tcp(assertion("RESPONSE_TIME", nil, "LESS_THAN", nil)), "request", "A RESPONSE_TIME assertion requires a target"},
		"tcp check response time text":         {NewTCPCheckResource(), "checkly_tcp_check", tcp(assertion("RESPONSE_TIME", nil, "GREATER_THAN", "slow")), "request", "must be a non-negative number of milliseconds"},
		"tcp response data no target":          {NewTCPMonitorResource(), "checkly_tcp_monitor", tcp(assertion("RESPONSE_DATA", nil, "IS_EMPTY", nil)), "request", ""},
		"dns response time":                    {NewDNSMonitorResource(), "checkly_dns_monitor", dns(assertion("RESPONSE_TIME", nil, "LESS_THAN", "100")), "request", ""},
		"dns response time no target":          {NewDNSMonitorResource(), "checkly_dns_monitor", dns(assertion("RESPONSE_TIME", nil, "LESS_THAN", nil)), "request", "A RESPONSE_TIME assertion requires a target"},
		"dns response time text target":        {NewDNSMonitorResource(), "checkly_dns_monitor", dns(assertion("RESPONSE_TIME", nil, "NOT_EQUALS", "slow")), "request", "must be a non-negative number of milliseconds"},
		"dns response time not empty":          {NewDNSMonitorResource(), "checkly_dns_monitor", dns(assertion("RESPONSE_TIME", nil, "NOT_EMPTY", nil)), "request", ""},
		"dns response time unknown comparison": {NewDNSMonitorResource(), "checkly_dns_monitor", dns(assertion("RESPONSE_TIME", nil, tftypes.UnknownValue, nil)), "request", ""},
		"dns response code no target":          {NewDNSMonitorResource(), "checkly_dns_monitor", dns(assertion("RESPONSE_CODE", nil, "EQUALS", nil)), "request", ""},
		"icmp json not null":                   {NewICMPMonitorResource(), "checkly_icmp_monitor", icmp(assertion("JSON_RESPONSE", "$.x", "NOT_NULL", nil)), "request", ""},
		"dns json not null":                    {NewDNSMonitorResource(), "checkly_dns_monitor", dns(assertion("JSON_ANSWER", "$.x", "NOT_NULL", nil)), "request", ""},
		"dns text contains":                    {NewDNSMonitorResource(), "checkly_dns_monitor", dns(assertion("TEXT_ANSWER", nil, "CONTAINS", "x")), "request", ""},
		"traceroute response time":             {NewTracerouteMonitorResource(), "checkly_traceroute_monitor", traceroute(assertion("RESPONSE_TIME", "avg", "LESS_THAN", "100.5")), "request", ""},
		"traceroute no property":               {NewTracerouteMonitorResource(), "checkly_traceroute_monitor", traceroute(assertion("RESPONSE_TIME", nil, "LESS_THAN", "100")), "request", "A RESPONSE_TIME assertion requires a property"},
		"traceroute padded target":             {NewTracerouteMonitorResource(), "checkly_traceroute_monitor", traceroute(assertion("HOP_COUNT", nil, "LESS_THAN", " 10")), "request", "whitespace"},
		"traceroute decimal hops":              {NewTracerouteMonitorResource(), "checkly_traceroute_monitor", traceroute(assertion("HOP_COUNT", nil, "LESS_THAN", "1.5")), "request", "must be a non-negative integer"},
		"traceroute packet loss 101":           {NewTracerouteMonitorResource(), "checkly_traceroute_monitor", traceroute(assertion("PACKET_LOSS", nil, "LESS_THAN", "101")), "request", "between 0 and 100"},
		"traceroute no target":                 {NewTracerouteMonitorResource(), "checkly_traceroute_monitor", traceroute(assertion("PACKET_LOSS", nil, "LESS_THAN", nil)), "request", "requires a target"},
		"traceroute hop count not equal":       {NewTracerouteMonitorResource(), "checkly_traceroute_monitor", traceroute(assertion("HOP_COUNT", nil, "NOT_EQUALS", "3")), "request", "The comparison of a HOP_COUNT assertion must be one of"},
		"traceroute unknown target":            {NewTracerouteMonitorResource(), "checkly_traceroute_monitor", traceroute(assertion("HOP_COUNT", nil, "LESS_THAN", tftypes.UnknownValue)), "request", ""},
		"grpc status code":                     {NewGRPCMonitorResource(), "checkly_grpc_monitor", grpc(assertion("GRPC_STATUS_CODE", nil, "EQUALS", "0")), "request", ""},
		"grpc status code 17":                  {NewGRPCMonitorResource(), "checkly_grpc_monitor", grpc(assertion("GRPC_STATUS_CODE", nil, "EQUALS", "17")), "request", "an integer between 0 and 16"},
		"grpc padded response time":            {NewGRPCMonitorResource(), "checkly_grpc_monitor", grpc(assertion("RESPONSE_TIME", nil, "LESS_THAN", "100 ")), "request", "whitespace"},
		"grpc health status greater":           {NewGRPCMonitorResource(), "checkly_grpc_monitor", grpc(assertion("GRPC_HEALTHCHECK_STATUS", nil, "GREATER_THAN", "1")), "request", "The comparison of a GRPC_HEALTHCHECK_STATUS assertion must be one of"},
		"grpc response has key":                {NewGRPCMonitorResource(), "checkly_grpc_monitor", grpc(assertion("GRPC_RESPONSE", "$.x", "HAS_KEY", "x")), "request", ""},
		"grpc response free target":            {NewGRPCMonitorResource(), "checkly_grpc_monitor", grpc(assertion("TEXT_BODY", nil, "CONTAINS", " padded ")), "request", ""},
	})
}

// TestUUIDRegex asserts that a ping token must be a UUID in the lowercase,
// hyphenated form the API stores it in.
func TestUUIDRegex(t *testing.T) {
	t.Parallel()

	tests := map[string]bool{
		"22868839-8450-4010-9241-1ea83a2e425f":   true,
		"00000000-0000-0000-0000-000000000000":   true,
		"22868839-8450-4010-9241-1EA83A2E425F":   false,
		"228688398450401092411ea83a2e425f":       false,
		"{22868839-8450-4010-9241-1ea83a2e425f}": false,
		"22868839:8450:4010:9241:1ea83a2e425f":   false,
		"22868839-8450-4010-9241-1ea83a2e425":    false,
		"":                                       false,
	}

	for value, want := range tests {
		if got := uuidRegex.MatchString(value); got != want {
			t.Errorf("uuidRegex.MatchString(%q) = %v, want %v", value, got, want)
		}
	}
}

// TestHostWithoutSchemeOrPortHint asserts that the error points to the port
// attribute only on monitors that have one.
func TestHostWithoutSchemeOrPortHint(t *testing.T) {
	t.Parallel()

	for _, v := range []hostWithoutSchemeOrPortValidator{{}, {noPortAttribute: true}} {
		req := validator.StringRequest{Path: path.Root("request").AtName("hostname"), ConfigValue: types.StringValue("example.com:443")}
		var resp validator.StringResponse
		v.ValidateString(context.Background(), req, &resp)
		if resp.Diagnostics.ErrorsCount() != 1 {
			t.Fatalf("noPortAttribute %v: want one error, got: %v", v.noPortAttribute, resp.Diagnostics)
		}
		hint := strings.Contains(resp.Diagnostics[0].Detail(), "port attribute")
		if hint == v.noPortAttribute {
			t.Errorf("noPortAttribute %v: detail %q", v.noPortAttribute, resp.Diagnostics[0].Detail())
		}
	}
}
