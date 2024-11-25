package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
)

var (
	_ resource.Resource                = (*SnippetResource)(nil)
	_ resource.ResourceWithConfigure   = (*SnippetResource)(nil)
	_ resource.ResourceWithImportState = (*SnippetResource)(nil)
)

type SnippetResource struct {
	client checkly.Client
}

func NewSnippetResource() resource.Resource {
	return &SnippetResource{}
}

func (r *SnippetResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_snippet"
}

func (r *SnippetResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "", // TODO
			},
			"last_updated": LastUpdatedAttributeSchema,
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the snippet.",
			},
			"script": schema.StringAttribute{
				Required: true,
				Description: "Your Node.js code that interacts with the API " +
					"check lifecycle, or functions as a partial for browser " +
					"checks.",
			},
		},
	}
}

func (r *SnippetResource) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	client, diags := ClientFromProviderData(req.ProviderData)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	r.client = client
}

func (r *SnippetResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *SnippetResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan SnippetResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var desiredModel checkly.Snippet
	resp.Diagnostics.Append(plan.Render(ctx, &desiredModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realizedModel, err := r.client.CreateSnippet(ctx, desiredModel)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Checkly Snippet",
			fmt.Sprintf("Could not create snippet, unexpected error: %s", err),
		)

		return
	}

	resp.Diagnostics.Append(plan.Refresh(ctx, realizedModel, ModelCreated)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *SnippetResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state SnippetResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, diags := SnippetID.FromString(state.ID)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	err := r.client.DeleteSnippet(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Checkly Snippet",
			fmt.Sprintf("Could not delete snippet, unexpected error: %s", err),
		)

		return
	}
}

func (r *SnippetResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state SnippetResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, diags := SnippetID.FromString(state.ID)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	realizedModel, err := r.client.GetSnippet(ctx, id)
	if err != nil {
		if SDKIsHTTPNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading Checkly Snippet",
			fmt.Sprintf("Could not retrieve snippet, unexpected error: %s", err),
		)

		return
	}

	resp.Diagnostics.Append(state.Refresh(ctx, realizedModel, ModelLoaded)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *SnippetResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan SnippetResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, diags := SnippetID.FromString(plan.ID)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	var desiredModel checkly.Snippet
	resp.Diagnostics.Append(plan.Render(ctx, &desiredModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realizedModel, err := r.client.UpdateSnippet(
		ctx,
		id,
		desiredModel,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Checkly Snippet",
			fmt.Sprintf("Could not update snippet, unexpected error: %s", err),
		)

		return
	}

	resp.Diagnostics.Append(plan.Refresh(ctx, realizedModel, ModelUpdated)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

var SnippetID = SDKIdentifier{
	Path:  path.Root("id"),
	Title: "Checkly Snippet ID",
}

var (
	_ ResourceModel[checkly.Snippet] = (*SnippetResourceModel)(nil)
)

type SnippetResourceModel struct {
	ID          types.String `tfsdk:"id"`
	LastUpdated types.String `tfsdk:"last_updated"` // FIXME: Keep this? Old code did not have it.
	Name        types.String `tfsdk:"name"`
	Script      types.String `tfsdk:"script"`
}

func (m *SnippetResourceModel) Refresh(ctx context.Context, from *checkly.Snippet, flags RefreshFlags) diag.Diagnostics {
	if flags.Created() {
		m.ID = SnippetID.IntoString(from.ID)
	}

	if flags.Created() || flags.Updated() {
		m.LastUpdated = LastUpdatedNow()
	}

	m.Name = types.StringValue(from.Name)
	m.Script = types.StringValue(from.Script)

	return nil
}

func (m *SnippetResourceModel) Render(ctx context.Context, into *checkly.Snippet) diag.Diagnostics {
	into.Name = m.Name.ValueString()
	into.Script = m.Script.ValueString()

	return nil
}
