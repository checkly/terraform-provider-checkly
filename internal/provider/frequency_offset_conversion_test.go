package provider

import (
	"context"
	"testing"

	"github.com/checkly/checkly-go-sdk/v2"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Every resource with frequency_offset reads it back the same way. With high
// frequency scheduling (frequency 0) the API value is stored. With a regular
// frequency the API reports an offset it assigned itself, which is ignored: a
// prior null or 0 is kept, and any other prior is cleared, so a frequency
// changed outside Terraform also shows the configured offset as a change.
func TestToModelFrequencyOffset(t *testing.T) {
	ctx := context.Background()

	// Each mapping converts an API response with the given frequency and
	// offset onto a model holding the prior frequency_offset, and returns the
	// resulting frequency_offset.
	mappings := map[string]func(frequency, offset int, prior types.Int64, diags *diag.Diagnostics) types.Int64{
		"checkly_url_monitor": func(frequency, offset int, prior types.Int64, diags *diag.Diagnostics) types.Int64 {
			model := URLMonitorResourceModel{FrequencyOffset: prior}
			urlMonitorToModel(ctx, &checkly.URLMonitor{ID: "abc", Frequency: frequency, FrequencyOffset: offset}, &model, diags)
			return model.FrequencyOffset
		},
		"checkly_dns_monitor": func(frequency, offset int, prior types.Int64, diags *diag.Diagnostics) types.Int64 {
			model := DNSMonitorResourceModel{FrequencyOffset: prior}
			dnsMonitorToModel(ctx, &checkly.DNSMonitor{ID: "abc", Frequency: frequency, FrequencyOffset: offset}, &model, diags)
			return model.FrequencyOffset
		},
		"checkly_icmp_monitor": func(frequency, offset int, prior types.Int64, diags *diag.Diagnostics) types.Int64 {
			model := ICMPMonitorResourceModel{FrequencyOffset: prior}
			icmpMonitorToModel(ctx, &checkly.ICMPMonitor{ID: "abc", Frequency: frequency, FrequencyOffset: offset}, &model, diags)
			return model.FrequencyOffset
		},
		"checkly_grpc_monitor": func(frequency, offset int, prior types.Int64, diags *diag.Diagnostics) types.Int64 {
			model := GRPCMonitorResourceModel{FrequencyOffset: prior}
			grpcMonitorToModel(ctx, &checkly.GRPCMonitor{ID: "abc", Frequency: frequency, FrequencyOffset: offset}, &model, diags)
			return model.FrequencyOffset
		},
		"checkly_ssl_monitor": func(frequency, offset int, prior types.Int64, diags *diag.Diagnostics) types.Int64 {
			model := SSLMonitorResourceModel{FrequencyOffset: prior}
			sslMonitorToModel(ctx, &checkly.SSLMonitor{ID: "abc", Frequency: frequency, FrequencyOffset: offset}, &model, diags)
			return model.FrequencyOffset
		},
		"checkly_tcp_monitor": func(frequency, offset int, prior types.Int64, diags *diag.Diagnostics) types.Int64 {
			model := TCPMonitorResourceModel{FrequencyOffset: prior}
			tcpMonitorToModel(ctx, &checkly.TCPMonitor{ID: "abc", Frequency: frequency, FrequencyOffset: offset}, &model, diags)
			return model.FrequencyOffset
		},
		"checkly_traceroute_monitor": func(frequency, offset int, prior types.Int64, diags *diag.Diagnostics) types.Int64 {
			model := TracerouteMonitorResourceModel{FrequencyOffset: prior}
			tracerouteMonitorToModel(ctx, &checkly.TracerouteMonitor{ID: "abc", Frequency: frequency, FrequencyOffset: offset}, &model, diags)
			return model.FrequencyOffset
		},
		"checkly_api_check": func(frequency, offset int, prior types.Int64, diags *diag.Diagnostics) types.Int64 {
			model := APICheckResourceModel{FrequencyOffset: prior}
			apiCheckToModel(ctx, &checkly.APICheck{ID: "abc", Frequency: frequency, FrequencyOffset: offset}, &model, diags)
			return model.FrequencyOffset
		},
		"checkly_check": func(frequency, offset int, prior types.Int64, diags *diag.Diagnostics) types.Int64 {
			model := emptyCheckModel()
			model.FrequencyOffset = prior
			checkToModel(ctx, &checkly.APICheck{ID: "abc", Frequency: frequency, FrequencyOffset: offset}, &model, diags)
			return model.FrequencyOffset
		},
	}

	cases := map[string]struct {
		frequency int
		offset    int
		prior     types.Int64
		want      types.Int64
	}{
		"high frequency stores the API value":           {0, 20, types.Int64Null(), types.Int64Value(20)},
		"high frequency replaces the prior":             {0, 10, types.Int64Value(30), types.Int64Value(10)},
		"regular frequency keeps a null prior":          {5, 37, types.Int64Null(), types.Int64Null()},
		"regular frequency keeps a prior 0":             {5, 37, types.Int64Value(0), types.Int64Value(0)},
		"regular frequency clears a prior offset":       {5, 37, types.Int64Value(10), types.Int64Null()},
		"regular frequency resolves an unknown to null": {5, 37, types.Int64Unknown(), types.Int64Null()},
	}

	for resource, mapping := range mappings {
		for name, tc := range cases {
			t.Run(resource+"/"+name, func(t *testing.T) {
				var diags diag.Diagnostics
				got := mapping(tc.frequency, tc.offset, tc.prior, &diags)
				if diags.HasError() {
					t.Fatal(diags)
				}
				if !got.Equal(tc.want) {
					t.Errorf("frequency_offset = %s, want %s", got, tc.want)
				}
			})
		}
	}
}
