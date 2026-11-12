package provider

import (
	"context"
	"fmt"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/checkly/checkly-go-sdk/v2"
)

func checkResourceAttrGreaterThanOrEqualTo(n int) resource.CheckResourceAttrWithFunc {
	return func(value string) error {
		i, err := strconv.Atoi(value)
		if err != nil {
			return err
		}

		if i < n {
			return fmt.Errorf("%v must be greater than or equal to %v", i, n)
		}

		return nil
	}
}

func TestAccStaticIPsAll(t *testing.T) {
	config := `data "checkly_static_ips" "test" {}`

	accTestCase(t, []resource.TestStep{
		{
			Config: config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrWith(
					"data.checkly_static_ips.test",
					"addresses.#",
					checkResourceAttrGreaterThanOrEqualTo(1),
				),
			),
		},
	})
}

func TestAccStaticIPsTwoRegionsOnly(t *testing.T) {
	config := `data "checkly_static_ips" "test" {
    locations = ["us-east-1","ap-southeast-1"]
  }`

	accTestCase(t, []resource.TestStep{
		{
			Config: config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrWith(
					"data.checkly_static_ips.test",
					"addresses.#",
					checkResourceAttrGreaterThanOrEqualTo(1),
				),
			),
		},
	})
}

func TestAccStaticIPsIPv6Only(t *testing.T) {
	config := `data "checkly_static_ips" "test" {
    ip_family = "IPv6"
  }`

	accTestCase(t, []resource.TestStep{
		{
			Config: config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrWith(
					"data.checkly_static_ips.test",
					"addresses.#",
					checkResourceAttrGreaterThanOrEqualTo(1),
				),
			),
		},
	})
}

func TestAccStaticIPsIPv4Only(t *testing.T) {
	config := `data "checkly_static_ips" "test" {
    ip_family = "IPv4"
  }`

	accTestCase(t, []resource.TestStep{
		{
			Config: config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrWith(
					"data.checkly_static_ips.test",
					"addresses.#",
					checkResourceAttrGreaterThanOrEqualTo(1),
				),
			),
		},
	})
}

func TestAccStaticIPsIPv6AndOneRegionOnly(t *testing.T) {
	config := `data "checkly_static_ips" "test" {
    ip_family = "IPv6"
    locations = ["us-east-1"]
  }`

	accTestCase(t, []resource.TestStep{
		{
			Config: config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrWith(
					"data.checkly_static_ips.test",
					"addresses.#",
					checkResourceAttrGreaterThanOrEqualTo(1),
				),
			),
		},
	})
}

func TestAccStaticIPsIPv4AndOneRegionOnly(t *testing.T) {
	config := `data "checkly_static_ips" "test" {
    ip_family = "IPv4"
    locations = ["us-east-1"]
  }`

	accTestCase(t, []resource.TestStep{
		{
			Config: config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrWith(
					"data.checkly_static_ips.test",
					"addresses.#",
					checkResourceAttrGreaterThanOrEqualTo(1),
				),
			),
		},
	})
}

func TestAccStaticIPsInvalidIPFamily(t *testing.T) {
	config := `data "checkly_static_ips" "test" {
    ip_family = "invalid"
  }`

	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`value must be one of`),
		},
	})
}

// staticIPsClient serves a fixed list of static IPs.
type staticIPsClient struct {
	checkly.Client
	ips []checkly.StaticIP
}

func (c *staticIPsClient) GetStaticIPs(ctx context.Context) ([]checkly.StaticIP, error) {
	return c.ips, nil
}

func TestStaticIPsUnknownLocation(t *testing.T) {
	ctx := context.Background()
	d := &StaticIPsDataSource{client: &staticIPsClient{ips: []checkly.StaticIP{
		{Region: "eu-central-1", Address: netip.MustParsePrefix("3.64.0.1/32")},
		{Region: "us-east-1", Address: netip.MustParsePrefix("3.80.0.1/32")},
	}}}

	var schemaResp datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)
	objectType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)

	read := func(ipFamily any, locations ...string) datasource.ReadResponse {
		values := make([]tftypes.Value, 0, len(locations))
		for _, l := range locations {
			values = append(values, tftypes.NewValue(tftypes.String, l))
		}
		config := tfsdk.Config{Schema: schemaResp.Schema, Raw: objectWith(objectType, map[string]tftypes.Value{
			"locations": tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, values),
			"ip_family": tftypes.NewValue(tftypes.String, ipFamily),
		})}
		resp := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		d.Read(ctx, datasource.ReadRequest{Config: config}, &resp)
		return resp
	}

	if resp := read(nil, "eu-central-1"); len(resp.Diagnostics) > 0 {
		t.Errorf("known location: %v", resp.Diagnostics)
	}
	// A known location without addresses of the IP family gives no warning.
	if resp := read("IPv6", "eu-central-1"); len(resp.Diagnostics) > 0 {
		t.Errorf("known location without IPv6 addresses: %v", resp.Diagnostics)
	}

	// A location without static IPs, such as a private location's slug,
	// only warns, and the other locations still give their addresses.
	resp := read(nil, "eu-central-1", "eu-centrall-1")
	if resp.Diagnostics.HasError() || resp.Diagnostics.WarningsCount() != 1 {
		t.Fatalf("want one warning and no error, got: %v", resp.Diagnostics)
	}
	var addresses types.Set
	if diags := resp.State.GetAttribute(ctx, path.Root("addresses"), &addresses); diags.HasError() {
		t.Fatal(diags)
	}
	if want := types.SetValueMust(types.StringType, []attr.Value{types.StringValue("3.64.0.1/32")}); !addresses.Equal(want) {
		t.Errorf("addresses = %s, want %s", addresses, want)
	}
	got := resp.Diagnostics.Warnings()[0]
	if got.Summary() != "Location Without Static IPs" ||
		!strings.Contains(got.Detail(), `"eu-centrall-1"`) ||
		!strings.Contains(got.Detail(), "eu-central-1, us-east-1") {
		t.Errorf("unexpected diagnostic %q: %q", got.Summary(), got.Detail())
	}
	if withPath, ok := got.(diag.DiagnosticWithPath); !ok || !withPath.Path().Equal(path.Root("locations")) {
		t.Errorf("diagnostic is not attached to locations: %v", got)
	}
}

// A location without static IPs only warns, so the read still returns the
// addresses of the other locations.
func TestAccStaticIPsUnknownLocation(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `data "checkly_static_ips" "test" {
				locations = ["us-east-1", "eu-centrall-1"]
			}`,
			Check: resource.TestCheckResourceAttrWith(
				"data.checkly_static_ips.test",
				"addresses.#",
				checkResourceAttrGreaterThanOrEqualTo(1),
			),
		},
	})
}
