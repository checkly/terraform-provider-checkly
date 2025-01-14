package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/checkly/terraform-provider-checkly/internal/provider/interop"
	"github.com/checkly/terraform-provider-checkly/internal/provider/resources/attributes"
	"github.com/checkly/terraform-provider-checkly/internal/sdkutil"
)

var (
	_ resource.Resource                = (*DashboardResource)(nil)
	_ resource.ResourceWithConfigure   = (*DashboardResource)(nil)
	_ resource.ResourceWithImportState = (*DashboardResource)(nil)
)

type DashboardResource struct {
	client checkly.Client
}

func NewDashboardResource() resource.Resource {
	return &DashboardResource{}
}

func (r *DashboardResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_dashboard"
}

func (r *DashboardResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": attributes.IDAttributeSchema,
			"custom_url": schema.StringAttribute{
				Required:    true,
				Description: "A subdomain name under 'checklyhq.com'. Needs to be unique across all users.",
			},
			"custom_domain": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     nil, // TODO
				Description: "A custom user domain, e.g. 'status.example.com'. See the docs on updating your DNS and SSL usage.",
			},
			"logo": schema.StringAttribute{
				Optional:    true,
				Description: "A URL pointing to an image file to use for the dashboard logo.",
			},
			"favicon": schema.StringAttribute{
				Optional:    true,
				Description: "A URL pointing to an image file to use as browser favicon.",
			},
			"link": schema.StringAttribute{
				Optional:    true,
				Description: "A link to for the dashboard logo.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "HTML <meta> description for the dashboard.",
			},
			"header": schema.StringAttribute{
				Optional:    true,
				Description: "A piece of text displayed at the top of your dashboard.",
			},
			"width": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("FULL"),
				Validators: []validator.String{
					stringvalidator.OneOf("FULL", "960PX"),
				},
				Description: "Determines whether to use the full screen or focus in the center. Possible values `FULL` and `960PX`.",
			},
			"refresh_rate": schema.Int32Attribute{
				Optional: true,
				Computed: true,
				Default:  int32default.StaticInt32(60),
				Validators: []validator.Int32{
					int32validator.OneOf(60, 300, 600),
				},
				Description: "How often to refresh the dashboard in seconds. Possible values `60`, '300' and `600`.",
			},
			"paginate": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Determines if pagination is on or off.",
			},
			"checks_per_page": schema.Int32Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int32default.StaticInt32(15),
				Description: "Determines how many checks to show per page.",
			},
			"pagination_rate": schema.Int32Attribute{
				Optional: true,
				Computed: true,
				Default:  int32default.StaticInt32(60),
				Validators: []validator.Int32{
					int32validator.OneOf(30, 60, 300),
				},
				Description: "How often to trigger pagination in seconds. Possible values `30`, `60` and `300`.",
			},
			"tags": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "A list of one or more tags that filter which checks to display on the dashboard.",
			},
			"hide_tags": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Show or hide the tags on the dashboard.",
			},
			"use_tags_and_operator": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Set when to use AND operator for fetching dashboard tags.",
			},
			"is_private": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Set your dashboard as private and generate key.",
			},
			"key": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The access key when the dashboard is private.",
			},
		},
	}
}

func (r *DashboardResource) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	client, diags := interop.ClientFromProviderData(req.ProviderData)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	r.client = client
}

func (r *DashboardResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *DashboardResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan DashboardResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var desiredModel checkly.Dashboard
	resp.Diagnostics.Append(plan.Render(ctx, &desiredModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realizedModel, err := r.client.CreateDashboard(ctx, desiredModel)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Checkly Dashboard",
			fmt.Sprintf("Could not create dashboard, unexpected error: %s", err),
		)

		return
	}

	resp.Diagnostics.Append(plan.Refresh(ctx, realizedModel, interop.Created)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *DashboardResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state DashboardResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDashboard(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Checkly Dashboard",
			fmt.Sprintf("Could not delete dashboard, unexpected error: %s", err),
		)

		return
	}
}

func (r *DashboardResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state DashboardResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realizedModel, err := r.client.GetDashboard(ctx, state.ID.ValueString())
	if err != nil {
		if sdkutil.IsHTTPNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading Checkly Dashboard",
			fmt.Sprintf("Could not retrieve dashboard, unexpected error: %s", err),
		)

		return
	}

	resp.Diagnostics.Append(state.Refresh(ctx, realizedModel, interop.Loaded)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *DashboardResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan DashboardResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var desiredModel checkly.Dashboard
	resp.Diagnostics.Append(plan.Render(ctx, &desiredModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realizedModel, err := r.client.UpdateDashboard(
		ctx,
		plan.ID.ValueString(),
		desiredModel,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Checkly Dashboard",
			fmt.Sprintf("Could not update dashboard, unexpected error: %s", err),
		)

		return
	}

	resp.Diagnostics.Append(plan.Refresh(ctx, realizedModel, interop.Updated)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

var (
	_ interop.Model[checkly.Dashboard] = (*DashboardResourceModel)(nil)
)

type DashboardResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	CustomURL          types.String `tfsdk:"custom_url"`
	CustomDomain       types.String `tfsdk:"custom_domain"`
	Logo               types.String `tfsdk:"logo"`
	Favicon            types.String `tfsdk:"favicon"`
	Link               types.String `tfsdk:"link"`
	Description        types.String `tfsdk:"description"`
	Header             types.String `tfsdk:"header"`
	Width              types.String `tfsdk:"width"`
	RefreshRate        types.Int32  `tfsdk:"refresh_rate"`
	Paginate           types.Bool   `tfsdk:"paginate"`
	ChecksPerPage      types.Int32  `tfsdk:"checks_per_page"`
	PaginationRate     types.Int32  `tfsdk:"pagination_rate"`
	Tags               types.Set    `tfsdk:"tags"`
	HideTags           types.Bool   `tfsdk:"hide_tags"`
	UseTagsAndOperator types.Bool   `tfsdk:"use_tags_and_operator"`
	IsPrivate          types.Bool   `tfsdk:"is_private"`
	Key                types.String `tfsdk:"key"`
}

func (m *DashboardResourceModel) Refresh(ctx context.Context, from *checkly.Dashboard, flags interop.RefreshFlags) diag.Diagnostics {
	if flags.Created() {
		m.ID = types.StringValue(from.DashboardID)
	}

	m.CustomURL = types.StringValue(from.CustomUrl)
	m.CustomDomain = types.StringValue(from.CustomDomain)
	m.Logo = types.StringValue(from.Logo)
	m.Favicon = types.StringValue(from.Favicon)
	m.Link = types.StringValue(from.Link)
	m.Description = types.StringValue(from.Description)
	m.Header = types.StringValue(from.Header)
	m.Width = types.StringValue(from.Width)
	m.RefreshRate = types.Int32Value(int32(from.RefreshRate))
	m.Paginate = types.BoolValue(from.Paginate)
	m.ChecksPerPage = types.Int32Value(int32(from.ChecksPerPage))
	m.PaginationRate = types.Int32Value(int32(from.PaginationRate))
	m.HideTags = types.BoolValue(from.HideTags)
	m.UseTagsAndOperator = types.BoolValue(from.UseTagsAndOperator)
	m.IsPrivate = types.BoolValue(from.IsPrivate)
	m.Tags = interop.IntoUntypedStringSet(&from.Tags)

	if from.IsPrivate {
		if len(from.Keys) > 0 {
			m.Key = types.StringValue(from.Keys[0].RawKey)
		}
	} else {
		m.Key = types.StringNull()
	}

	return nil
}

func (m *DashboardResourceModel) Render(ctx context.Context, into *checkly.Dashboard) diag.Diagnostics {
	into.CustomUrl = m.CustomURL.ValueString()
	into.CustomDomain = m.CustomDomain.ValueString()
	into.IsPrivate = m.IsPrivate.ValueBool()
	into.Logo = m.Logo.ValueString()
	into.Link = m.Link.ValueString()
	into.Description = m.Description.ValueString()
	into.Favicon = m.Favicon.ValueString()
	into.Header = m.Header.ValueString()
	into.Width = m.Width.ValueString()
	into.RefreshRate = int(m.RefreshRate.ValueInt32())
	into.ChecksPerPage = int(m.ChecksPerPage.ValueInt32())
	into.PaginationRate = int(m.PaginationRate.ValueInt32())
	into.Paginate = m.Paginate.ValueBool()
	into.Tags = interop.FromUntypedStringSet(m.Tags)
	into.HideTags = m.HideTags.ValueBool()
	into.UseTagsAndOperator = m.UseTagsAndOperator.ValueBool()

	return nil
}
