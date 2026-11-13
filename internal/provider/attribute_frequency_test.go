package provider

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// Every check and monitor with a schedule has an optional frequency that
// defaults to the Checkly API default; heartbeat monitors and v1 check groups
// have no frequency at all.
func TestResourcesFrequencyDefault(t *testing.T) {
	ctx := context.Background()

	hasFrequency := map[string]bool{
		"checkly_api_check":              true,
		"checkly_browser_check":          true,
		"checkly_multistep_check":        true,
		"checkly_check":                  true,
		"checkly_url_monitor":            true,
		"checkly_dns_monitor":            true,
		"checkly_icmp_monitor":           true,
		"checkly_tcp_monitor":            true,
		"checkly_tcp_check":              true,
		"checkly_grpc_monitor":           true,
		"checkly_ssl_monitor":            true,
		"checkly_traceroute_monitor":     true,
		"checkly_playwright_check_suite": true,
		"checkly_heartbeat_monitor":      false,
		"checkly_heartbeat":              false,
		"checkly_check_group":            false,
	}

	p := New("test")()
	var providerMetadata provider.MetadataResponse
	p.Metadata(ctx, provider.MetadataRequest{}, &providerMetadata)

	seen := map[string]bool{}
	for _, newResource := range p.Resources(ctx) {
		r := newResource()

		var metadata resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: providerMetadata.TypeName}, &metadata)

		want, ok := hasFrequency[metadata.TypeName]
		if !ok {
			continue
		}
		seen[metadata.TypeName] = true

		var schemaResp resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

		a, has := schemaResp.Schema.Attributes[frequencyAttributeName]
		if !want {
			if has {
				t.Errorf("%s: has %s", metadata.TypeName, frequencyAttributeName)
			}
			continue
		}
		if !has {
			t.Errorf("%s: lacks %s", metadata.TypeName, frequencyAttributeName)
			continue
		}
		frequency, ok := a.(schema.Int64Attribute)
		if !ok {
			t.Errorf("%s: %s is %T", metadata.TypeName, frequencyAttributeName, a)
			continue
		}
		if frequency.Required || !frequency.Optional || !frequency.Computed {
			t.Errorf("%s: required=%v optional=%v computed=%v, want optional and computed", metadata.TypeName,
				frequency.Required, frequency.Optional, frequency.Computed)
		}
		if def := attributeDefault(t, ctx, frequency); def == nil || !def.Equal(types.Int64Value(defaultFrequency)) {
			t.Errorf("%s: default %v, want %d", metadata.TypeName, def, defaultFrequency)
		}
		if !strings.HasSuffix(frequency.Description, "Defaults to `10`.") {
			t.Errorf("%s: description %q does not state the default", metadata.TypeName, frequency.Description)
		}
	}

	for name := range hasFrequency {
		if !seen[name] {
			t.Errorf("%s: not registered", name)
		}
	}
}

// An unset frequency is validated as the default frequency, which does not
// allow frequency_offset.
func TestFrequencyOffsetConfigValidatorUnsetFrequency(t *testing.T) {
	ctx := context.Background()

	var schemaResp resource.SchemaResponse
	NewURLMonitorResource().Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	objectType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)

	cases := map[string]struct {
		offset  tftypes.Value
		wantErr bool
	}{
		"offset unset": {tftypes.NewValue(tftypes.Number, nil), false},
		"offset 0":     {tftypes.NewValue(tftypes.Number, big.NewFloat(0)), false},
		"offset 20":    {tftypes.NewValue(tftypes.Number, big.NewFloat(20)), true},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			config := tfsdk.Config{Schema: schemaResp.Schema, Raw: objectWith(objectType, map[string]tftypes.Value{
				frequencyOffsetAttributeName: c.offset,
			})}

			var resp resource.ValidateConfigResponse
			newFrequencyOffsetConfigValidator().ValidateResource(ctx, resource.ValidateConfigRequest{Config: config}, &resp)
			if got := resp.Diagnostics.HasError(); got != c.wantErr {
				t.Errorf("error reported = %v, want %v: %v", got, c.wantErr, resp.Diagnostics)
			}
			if c.wantErr && !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "can only be set") {
				t.Errorf("unexpected error: %v", resp.Diagnostics)
			}
		})
	}
}

func TestFrequencyOffsetFromAPI(t *testing.T) {
	cases := map[string]struct {
		frequency int
		offset    int
		prior     types.Int64
		want      types.Int64
	}{
		"high frequency uses the API value":       {0, 20, types.Int64Null(), types.Int64Value(20)},
		"high frequency ignores the prior":        {0, 10, types.Int64Value(30), types.Int64Value(10)},
		"high frequency resolves unknown":         {0, 10, types.Int64Unknown(), types.Int64Value(10)},
		"regular frequency keeps null":            {10, 0, types.Int64Null(), types.Int64Null()},
		"regular frequency keeps 0":               {10, 0, types.Int64Value(0), types.Int64Value(0)},
		"regular frequency clears unknown":        {10, 0, types.Int64Unknown(), types.Int64Null()},
		"regular frequency clears another offset": {10, 0, types.Int64Value(20), types.Int64Null()},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if got := frequencyOffsetFromAPI(c.frequency, c.offset, c.prior); !got.Equal(c.want) {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}
