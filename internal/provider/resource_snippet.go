package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

var (
	_ resource.Resource                 = (*SnippetResource)(nil)
	_ resource.ResourceWithConfigure    = (*SnippetResource)(nil)
	_ resource.ResourceWithUpgradeState = (*SnippetResource)(nil)
	_ resource.ResourceWithImportState  = (*SnippetResource)(nil)
)

// NewSnippetResource returns a new checkly_snippet resource.
func NewSnippetResource() resource.Resource {
	return &SnippetResource{}
}

// SnippetResource implements the checkly_snippet resource.
type SnippetResource struct {
	client checkly.Client
}

// SnippetResourceModel describes the resource data model.
type SnippetResourceModel struct {
	ID     types.String `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Script types.String `tfsdk:"script"`
}

func (r *SnippetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_snippet"
}

func (r *SnippetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     sdkv2SchemaVersion + 1,
		Description: "Snippets are reusable pieces of code that can be referenced from setup & teardown scripts of API checks, or used as partials in browser checks.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the snippet.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the snippet.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"script": schema.StringAttribute{
				Required:    true,
				Description: "Your Node.js code that interacts with the API check lifecycle, or functions as a partial for browser checks.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

// UpgradeState upgrades state written by the provider's SDK v2
// implementation; see sdkv2StateUpgraders.
func (r *SnippetResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return sdkv2StateUpgraders(ctx, r)
}

func (r *SnippetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *SnippetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SnippetResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.CreateSnippet(apiCtx, checkly.Snippet{
		Name:   plan.Name.ValueString(),
		Script: plan.Script.ValueString(),
	})
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to Create Snippet", err)
		return
	}

	snippetToModel(result, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SnippetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SnippetResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := decodeNumericID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Snippet ID",
			fmt.Sprintf("Snippet ID must be numeric: %s", err),
		)
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	snippet, err := r.client.GetSnippet(apiCtx, id)
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		addAPIError(&resp.Diagnostics, "Unable to Read Snippet", err)
		return
	}

	snippetToModel(snippet, &state)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SnippetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SnippetResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := decodeNumericID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Snippet ID",
			fmt.Sprintf("Snippet ID must be numeric: %s", err),
		)
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.UpdateSnippet(apiCtx, id, checkly.Snippet{
		Name:   plan.Name.ValueString(),
		Script: plan.Script.ValueString(),
	})
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to Update Snippet", err)
		return
	}

	snippetToModel(result, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SnippetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SnippetResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := decodeNumericID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Snippet ID",
			fmt.Sprintf("Snippet ID must be numeric: %s", err),
		)
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	err = r.client.DeleteSnippet(apiCtx, id)
	if err != nil {
		if isNotFoundError(err) {
			return
		}
		addAPIError(&resp.Diagnostics, "Unable to Delete Snippet", err)
		return
	}
}

func (r *SnippetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// snippetToModel updates the model to match the API representation of the
// snippet.
func snippetToModel(snippet *checkly.Snippet, model *SnippetResourceModel) {
	model.ID = types.StringValue(encodeNumericID(snippet.ID))
	model.Name = types.StringValue(snippet.Name)
	model.Script = types.StringValue(snippet.Script)
}
