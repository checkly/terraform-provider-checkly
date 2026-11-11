package provider

import (
	"context"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
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
				Description: "Specify the locations you want to get static IPs for.",
			},
			"ip_family": schema.StringAttribute{
				Optional:    true,
				Description: "Specify the IP families you want to get static IPs for. Only `IPv6` or `IPv4` are valid options.",
				Validators: []validator.String{
					stringvalidator.OneOf("IPv4", "IPv6"),
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

	ipFamily := config.IPFamily.ValueString()

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	staticIPs, err := d.client.GetStaticIPs(apiCtx)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to Read Static IPs", err)
		return
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
