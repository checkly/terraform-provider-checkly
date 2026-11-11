package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

var (
	_ resource.Resource                 = (*StatusPageServiceResource)(nil)
	_ resource.ResourceWithConfigure    = (*StatusPageServiceResource)(nil)
	_ resource.ResourceWithUpgradeState = (*StatusPageServiceResource)(nil)
	_ resource.ResourceWithImportState  = (*StatusPageServiceResource)(nil)
)

// NewStatusPageServiceResource returns a new checkly_status_page_service
// resource.
func NewStatusPageServiceResource() resource.Resource {
	return &StatusPageServiceResource{}
}

// StatusPageServiceResource implements the checkly_status_page_service
// resource.
type StatusPageServiceResource struct {
	client checkly.Client
}

// StatusPageServiceResourceModel describes the resource data model.
type StatusPageServiceResourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func (r *StatusPageServiceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_status_page_service"
}

func (r *StatusPageServiceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: sdkv2SchemaVersion + 1,
		Description: "Status page services represent functional pieces of " +
			"your application or website, such as landing page, API, support portal etc. " +
			"Services belong to legacy v2 status pages: on new v3 pages, use " +
			"`checkly_status_page_v3_component` instead." +
			"\n\n" +
			"The `checkly_status_page_service` resource type is deprecated together with `checkly_status_page` and " +
			"will be removed in a future version of the provider. A maintenance window that affects the service " +
			"links to v3 components through `status_page_visibility.component_configs` on " +
			"`checkly_maintenance_window`.",
		DeprecationMessage: "Use `checkly_status_page_v3_component` resources on a `checkly_status_page_v3` page " +
			"instead; see the v2 upgrade guide.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the service.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the service.",
			},
		},
	}
}

// UpgradeState upgrades state written by the provider's SDK v2
// implementation; see sdkv2StateUpgraders.
func (r *StatusPageServiceResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return sdkv2StateUpgraders(ctx, r)
}

func (r *StatusPageServiceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *StatusPageServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan StatusPageServiceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.CreateStatusPageService(apiCtx, checkly.StatusPageService{
		Name: plan.Name.ValueString(),
	})
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to Create Status Page Service", err)
		return
	}

	statusPageServiceToModel(result, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *StatusPageServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state StatusPageServiceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	service, err := r.client.GetStatusPageService(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		addAPIError(&resp.Diagnostics, "Unable to Read Status Page Service", err)
		return
	}

	statusPageServiceToModel(service, &state)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *StatusPageServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan StatusPageServiceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.UpdateStatusPageService(apiCtx, plan.ID.ValueString(), checkly.StatusPageService{
		ID:   plan.ID.ValueString(),
		Name: plan.Name.ValueString(),
	})
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to Update Status Page Service", err)
		return
	}

	statusPageServiceToModel(result, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *StatusPageServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state StatusPageServiceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	err := r.client.DeleteStatusPageService(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			return
		}
		addAPIError(&resp.Diagnostics, "Unable to Delete Status Page Service", err)
		return
	}
}

func (r *StatusPageServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// statusPageServiceToModel updates the model to match the API representation
// of the status page service.
func statusPageServiceToModel(service *checkly.StatusPageService, model *StatusPageServiceResourceModel) {
	model.ID = types.StringValue(service.ID)
	model.Name = types.StringValue(service.Name)
}
