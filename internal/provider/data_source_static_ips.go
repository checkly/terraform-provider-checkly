package provider

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

var (
	_ datasource.DataSource              = (*StaticIPsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*StaticIPsDataSource)(nil)
)

// NewStaticIPsDataSource returns a new checkly_static_ips data source.
func NewStaticIPsDataSource() datasource.DataSource {
	return &StaticIPsDataSource{}
}

// StaticIPsDataSource implements the checkly_static_ips data source.
type StaticIPsDataSource struct {
	client checkly.Client
}

// StaticIPsDataSourceModel describes the data source data model.
type StaticIPsDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Addresses types.Set    `tfsdk:"addresses"`
	Locations types.Set    `tfsdk:"locations"`
	IPFamily  types.String `tfsdk:"ip_family"`
}

// staticIPFamilyAll selects the addresses of both IP families. It is the
// value of ip_family when the configuration leaves it out.
const staticIPFamilyAll = "all"

var staticIPFamilies = allowedValues[string]{
	{Value: staticIPFamilyAll, Description: "both families"},
	{Value: "IPv4"},
	{Value: "IPv6"},
}

func (d *StaticIPsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_static_ips"
}

func (d *StaticIPsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Static IP addresses for Checkly's runner infrastructure. Useful when you need to allowlist Checkly traffic.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the static IPs data source.",
			},
			"addresses": schema.SetAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Static IP addresses for Checkly's runner infrastructure, in CIDR notation (for example `3.64.0.1/32`).",
			},
			"locations": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Specify the locations you want to get static IPs for, such as `eu-central-1`. A location that has no static IPs, such as a misspelled or private one, adds no addresses and gives a warning. When not set, the addresses of every location are returned.",
			},
			"ip_family": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Specify the IP family you want to get static IPs for. " + staticIPFamilies.String() + " (Default `" + staticIPFamilyAll + "`).",
				Validators: []validator.String{
					stringvalidator.OneOf(staticIPFamilies.Values()...),
				},
			},
		},
	}
}

func (d *StaticIPsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (d *StaticIPsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config StaticIPsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var locations []string
	if !config.Locations.IsNull() {
		resp.Diagnostics.Append(config.Locations.ElementsAs(ctx, &locations, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// A data source schema has no defaults, so Read fills in the default.
	if config.IPFamily.IsNull() {
		config.IPFamily = types.StringValue(staticIPFamilyAll)
	}
	ipFamily := config.IPFamily.ValueString()

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	staticIPs, err := d.client.GetStaticIPs(apiCtx)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to Read Static IPs", err)
		return
	}

	// A location without static IPs, such as a misspelled or private one,
	// adds no addresses. That only warns: the same list of locations often
	// goes to checks, where private locations and regions without static
	// IPs are valid.
	regions := map[string]bool{}
	for _, ip := range staticIPs {
		regions[ip.Region] = true
	}
	var unknown []string
	for _, location := range locations {
		if !regions[location] {
			unknown = append(unknown, fmt.Sprintf("%q", location))
		}
	}
	if len(unknown) > 0 {
		slices.Sort(unknown)
		resp.Diagnostics.AddAttributeWarning(
			path.Root("locations"),
			"Location Without Static IPs",
			fmt.Sprintf("Checkly has no static IPs for %s, so they add no addresses. Private locations and misspelled locations have none. The locations with static IPs are: %s.",
				strings.Join(unknown, ", "), strings.Join(slices.Sorted(maps.Keys(regions)), ", ")),
		)
	}

	var addresses []string
	for _, ip := range staticIPs {
		if len(locations) > 0 && !slices.Contains(locations, ip.Region) {
			continue
		}

		if ipFamily == "IPv4" && !ip.Address.Addr().Is4() {
			continue
		}

		if ipFamily == "IPv6" && !ip.Address.Addr().Is6() {
			continue
		}

		addresses = append(addresses, ip.Address.String())
	}

	config.ID = types.StringValue("checkly_static_ips_data_source_id")
	config.Addresses = setFromStrings(ctx, addresses, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
