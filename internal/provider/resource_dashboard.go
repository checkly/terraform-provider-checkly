package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
)

var (
	_ resource.Resource                 = (*DashboardResource)(nil)
	_ resource.ResourceWithConfigure    = (*DashboardResource)(nil)
	_ resource.ResourceWithUpgradeState = (*DashboardResource)(nil)
	_ resource.ResourceWithImportState  = (*DashboardResource)(nil)
)

// NewDashboardResource returns a new checkly_dashboard resource.
func NewDashboardResource() resource.Resource {
	return &DashboardResource{}
}

// DashboardResource implements the checkly_dashboard resource.
type DashboardResource struct {
	client checkly.Client
}

// DashboardResourceModel describes the resource data model.
type DashboardResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	CustomURL          types.String `tfsdk:"custom_url"`
	CustomDomain       types.String `tfsdk:"custom_domain"`
	Logo               types.String `tfsdk:"logo"`
	Favicon            types.String `tfsdk:"favicon"`
	Link               types.String `tfsdk:"link"`
	Description        types.String `tfsdk:"description"`
	Header             types.String `tfsdk:"header"`
	ShowHeader         types.Bool   `tfsdk:"show_header"`
	Width              types.String `tfsdk:"width"`
	RefreshRate        types.Int64  `tfsdk:"refresh_rate"`
	Paginate           types.Bool   `tfsdk:"paginate"`
	ChecksPerPage      types.Int64  `tfsdk:"checks_per_page"`
	PaginationRate     types.Int64  `tfsdk:"pagination_rate"`
	Tags               types.Set    `tfsdk:"tags"`
	HideTags           types.Bool   `tfsdk:"hide_tags"`
	UseTagsAndOperator types.Bool   `tfsdk:"use_tags_and_operator"`
	EnableIncidents    types.Bool   `tfsdk:"enable_incidents"`
	IsPrivate          types.Bool   `tfsdk:"is_private"`
	Key                types.String `tfsdk:"key"`
	ExpandChecks       types.Bool   `tfsdk:"expand_checks"`
	ShowCheckRunLinks  types.Bool   `tfsdk:"show_check_run_links"`
	CustomCSS          types.String `tfsdk:"custom_css"`
	ShowP95            types.Bool   `tfsdk:"show_p95"`
	ShowP99            types.Bool   `tfsdk:"show_p99"`
}

func (r *DashboardResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dashboard"
}

func (r *DashboardResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     sdkv2SchemaVersion + 1,
		Description: "Dashboards are public status pages that display selected checks and their statuses.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the dashboard.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"custom_url": schema.StringAttribute{
				Required:    true,
				Description: "A subdomain name under 'checklyhq.com'. Needs to be unique across all users.",
			},
			"custom_domain": schema.StringAttribute{
				Optional:    true,
				Description: "A custom user domain, e.g. 'status.example.com'. See the docs on updating your DNS and SSL usage.",
			},
			"logo": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "A URL pointing to an image file to use for the dashboard logo.",
			},
			"favicon": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "A URL pointing to an image file to use as browser favicon.",
			},
			"link": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "A link to for the dashboard logo.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "HTML <meta> description for the dashboard.",
			},
			"header": schema.StringAttribute{
				Required:    true,
				Description: "A piece of text displayed at the top of your dashboard.",
			},
			"show_header": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Show or hide header and description on the dashboard. (Default `true`).",
			},
			"width": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("FULL"),
				Description: "Determines whether to use the full screen or focus in the center. Possible values are `FULL` and `960PX`. (Default `FULL`).",
				Validators: []validator.String{
					stringvalidator.OneOf("FULL", "960PX"),
				},
			},
			"refresh_rate": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(60),
				Description: "How often to refresh the dashboard in seconds. Possible values `60`, '300' and `600`. (Default `60`).",
				Validators: []validator.Int64{
					int64validator.OneOf(60, 300, 600),
				},
			},
			"paginate": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Determines if pagination is on or off. (Default `true`).",
			},
			"checks_per_page": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(15),
				Description: "Determines how many checks to show per page. Possible values are between 1 and 20. (Default `15`).",
				Validators: []validator.Int64{
					int64validator.Between(1, 20),
				},
			},
			"pagination_rate": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(60),
				Description: "How often to trigger pagination in seconds. Possible values `30`, `60` and `300`. (Default `60`).",
				Validators: []validator.Int64{
					int64validator.OneOf(30, 60, 300),
				},
			},
			"tags": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "A list of one or more tags that filter which checks to display on the dashboard.",
			},
			"hide_tags": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Show or hide the tags on the dashboard. (Default `false`).",
			},
			"use_tags_and_operator": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Set when to use AND operator for fetching dashboard tags. (Default `false`).",
			},
			"enable_incidents": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Enable incident support for the dashboard. (Default `false`).",
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
			"expand_checks": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Expand or collapse checks on the dashboard. (Default `false`).",
			},
			"show_check_run_links": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Show or hide check run links on the dashboard. (Default `false`).",
			},
			"custom_css": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Custom CSS to be applied to the dashboard.",
			},
			"show_p95": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Show or hide the P95 stats on the dashboard. (Default `true`).",
			},
			"show_p99": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Show or hide the P99 stats on the dashboard. (Default `true`).",
			},
		},
	}
}

// UpgradeState upgrades state written by the provider's SDK v2
// implementation; see sdkv2StateUpgraders.
func (r *DashboardResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return sdkv2StateUpgraders(ctx, r)
}

func (r *DashboardResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *DashboardResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DashboardResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dashboard := dashboardFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.CreateDashboard(apiCtx, dashboard)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Dashboard",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	plan.ID = types.StringValue(result.DashboardID)

	// The raw access key of a private dashboard is only included in create
	// and update responses, never in reads, so it must be captured here.
	dashboardToModel(ctx, result, &plan, &resp.Diagnostics)
	plan.Key = dashboardKeyValue(result, plan.Key)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DashboardResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DashboardResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	dashboard, err := r.client.GetDashboard(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Read Dashboard",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	dashboardToModel(ctx, dashboard, &state, &resp.Diagnostics)
	state.Key = dashboardKeyValue(dashboard, state.Key)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DashboardResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state DashboardResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dashboard := dashboardFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.UpdateDashboard(apiCtx, plan.ID.ValueString(), dashboard)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Dashboard",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	plan.ID = types.StringValue(result.DashboardID)

	// The raw access key of a private dashboard is only included in create
	// and update responses, never in reads, so it must be captured here. If
	// the response has no keys, the key already in state remains valid.
	dashboardToModel(ctx, result, &plan, &resp.Diagnostics)
	plan.Key = dashboardKeyValue(result, state.Key)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DashboardResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DashboardResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	err := r.client.DeleteDashboard(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Delete Dashboard",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}
}

func (r *DashboardResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func dashboardFromModel(ctx context.Context, model *DashboardResourceModel, diags *diag.Diagnostics) checkly.Dashboard {
	showHeader := model.ShowHeader.ValueBool()
	showP95 := model.ShowP95.ValueBool()
	showP99 := model.ShowP99.ValueBool()

	return checkly.Dashboard{
		CustomUrl:          model.CustomURL.ValueString(),
		CustomDomain:       model.CustomDomain.ValueString(),
		Logo:               model.Logo.ValueString(),
		Favicon:            model.Favicon.ValueString(),
		Link:               model.Link.ValueString(),
		Description:        model.Description.ValueString(),
		Header:             model.Header.ValueString(),
		ShowHeader:         &showHeader,
		Width:              model.Width.ValueString(),
		RefreshRate:        int(model.RefreshRate.ValueInt64()),
		Paginate:           model.Paginate.ValueBool(),
		ChecksPerPage:      int(model.ChecksPerPage.ValueInt64()),
		PaginationRate:     int(model.PaginationRate.ValueInt64()),
		Tags:               stringsFromSet(ctx, model.Tags, diags),
		HideTags:           model.HideTags.ValueBool(),
		UseTagsAndOperator: model.UseTagsAndOperator.ValueBool(),
		EnableIncidents:    model.EnableIncidents.ValueBool(),
		IsPrivate:          model.IsPrivate.ValueBool(),
		ExpandChecks:       model.ExpandChecks.ValueBool(),
		ShowCheckRunLinks:  model.ShowCheckRunLinks.ValueBool(),
		CustomCSS:          model.CustomCSS.ValueString(),
		ShowP95:            &showP95,
		ShowP99:            &showP99,
	}
}

// dashboardToModel updates the model to match the API representation of the
// dashboard. It does not touch the ID or the access key; the access key needs
// prior state awareness and is handled by dashboardKeyValue. The model must
// hold the prior data (the plan during create/update, the state during read)
// so that values the API omits can be resolved against it.
func dashboardToModel(ctx context.Context, dashboard *checkly.Dashboard, model *DashboardResourceModel, diags *diag.Diagnostics) {
	model.CustomURL = types.StringValue(dashboard.CustomUrl)
	// The API returns an empty custom domain when none is configured; map it
	// to null to match an unset optional attribute.
	model.CustomDomain = stringValueOrNull(dashboard.CustomDomain)
	model.Logo = types.StringValue(dashboard.Logo)
	model.Favicon = types.StringValue(dashboard.Favicon)
	model.Link = types.StringValue(dashboard.Link)
	model.Description = types.StringValue(dashboard.Description)
	model.Header = types.StringValue(dashboard.Header)
	model.ShowHeader = dashboardBoolValue(dashboard.ShowHeader, true)
	model.Width = types.StringValue(dashboard.Width)
	model.RefreshRate = types.Int64Value(int64(dashboard.RefreshRate))
	model.Paginate = types.BoolValue(dashboard.Paginate)
	model.ChecksPerPage = types.Int64Value(int64(dashboard.ChecksPerPage))
	model.PaginationRate = types.Int64Value(int64(dashboard.PaginationRate))
	model.Tags = dashboardTagsToSet(ctx, dashboard.Tags, model.Tags, diags)
	model.HideTags = types.BoolValue(dashboard.HideTags)
	model.UseTagsAndOperator = types.BoolValue(dashboard.UseTagsAndOperator)
	model.EnableIncidents = types.BoolValue(dashboard.EnableIncidents)
	model.IsPrivate = types.BoolValue(dashboard.IsPrivate)
	model.ExpandChecks = types.BoolValue(dashboard.ExpandChecks)
	model.ShowCheckRunLinks = types.BoolValue(dashboard.ShowCheckRunLinks)
	model.CustomCSS = types.StringValue(dashboard.CustomCSS)
	model.ShowP95 = dashboardBoolValue(dashboard.ShowP95, true)
	model.ShowP99 = dashboardBoolValue(dashboard.ShowP99, true)
}

// dashboardKeyValue determines the value of the key attribute. A public
// dashboard has no key. For a private dashboard, the raw key is only present
// in create and update API responses; when it is absent, the prior value
// remains valid.
func dashboardKeyValue(dashboard *checkly.Dashboard, prior types.String) types.String {
	if !dashboard.IsPrivate {
		return types.StringNull()
	}

	if len(dashboard.Keys) > 0 {
		return types.StringValue(dashboard.Keys[0].RawKey)
	}

	if prior.IsUnknown() {
		return types.StringNull()
	}

	return prior
}

// dashboardBoolValue converts an optional bool from the API, applying the
// API-side default when the value is absent.
func dashboardBoolValue(value *bool, absent bool) types.Bool {
	if value == nil {
		return types.BoolValue(absent)
	}

	return types.BoolValue(*value)
}

// dashboardTagsToSet converts tags from the API to a set, resolving the empty
// value against the prior data: when no tags exist remotely and none were
// configured, the attribute stays null.
func dashboardTagsToSet(ctx context.Context, tags []string, prior types.Set, diags *diag.Diagnostics) types.Set {
	if len(tags) == 0 && (prior.IsNull() || prior.IsUnknown()) {
		return types.SetNull(types.StringType)
	}

	return setFromStrings(ctx, tags, diags)
}
