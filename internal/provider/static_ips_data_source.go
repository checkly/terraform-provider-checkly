package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
)

var (
	_ datasource.DataSource              = (*StaticIPsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*StaticIPsDataSource)(nil)
)

type StaticIPsDataSource struct {
	client checkly.Client
}

func NewStaticIPsDataSource() datasource.DataSource {
	return &StaticIPsDataSource{}
}

type StaticIPsDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Addresses types.Set    `tfsdk:"addresses"`
	Locations types.Set    `tfsdk:"locations"`
	IPFamily  types.String `tfsdk:"ip_family"`
}

func (d *StaticIPsDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_static_ips"
}

func (d *StaticIPsDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description:         "", // TODO
		MarkdownDescription: "", // TODO
		Attributes: map[string]schema.Attribute{
			"id": IDDataSourceAttributeSchema,
			"addresses": schema.SetAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Static IP addresses for Checkly's runner infrastructure.",
			},
			"locations": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Specify the locations you want to get static IPs for.",
			},
			"ip_family": schema.StringAttribute{
				Optional: true,
				Description: "Specify the IP families you want to get static " +
					"IPs for. Only `IPv6` or `IPv4` are valid options.",
				Validators: []validator.String{
					stringvalidator.OneOf("IPv6", "IPv4"),
				},
			},
		},
	}
}

func (d *StaticIPsDataSource) Configure(
	ctx context.Context,
	req datasource.ConfigureRequest,
	resp *datasource.ConfigureResponse,
) {
	client, diags := ClientFromProviderData(req.ProviderData)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	d.client = client
}

func (d *StaticIPsDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data StaticIPsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	staticIPs, err := d.client.GetStaticIPs(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Checkly Static IPs",
			fmt.Sprintf("Could not retrieve static IPs, unexpected error: %s", err),
		)

		return
	}

	var haveLocationFilters bool
	includeLocation := make(map[string]bool)
	for _, el := range data.Locations.Elements() {
		haveLocationFilters = true
		includeLocation[el.(types.String).ValueString()] = true
	}

	ipFamilyFilter := data.IPFamily.ValueString()

	only4 := ipFamilyFilter == "IPv4"
	only6 := ipFamilyFilter == "IPv6"

	var addressValues []attr.Value
	for _, ip := range staticIPs {
		switch {
		case only4 && !ip.Address.Addr().Is4():
			continue
		case only6 && !ip.Address.Addr().Is6():
			continue
		case haveLocationFilters && !includeLocation[ip.Region]:
			continue
		}

		addressValues = append(addressValues, types.StringValue(ip.Address.String()))
	}

	addresses, diags := types.SetValue(types.StringType, addressValues)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	data.ID = types.StringValue("checkly_static_ips_data_source_id")
	data.Addresses = addresses

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
