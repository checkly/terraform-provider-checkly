package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
)

var (
	_ resource.Resource                = (*PrivateLocationResource)(nil)
	_ resource.ResourceWithConfigure   = (*PrivateLocationResource)(nil)
	_ resource.ResourceWithImportState = (*PrivateLocationResource)(nil)
)

type PrivateLocationResource struct {
	client checkly.Client
}

func NewPrivateLocationResource() resource.Resource {
	return &PrivateLocationResource{}
}

func (r *PrivateLocationResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_private_location"
}

func (r *PrivateLocationResource) Schema(
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
				Description: "The private location name.",
			},
			"slug_name": schema.StringAttribute{
				Required:    true,
				Description: "Valid slug name.",
			},
			"icon": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("location"),
				Description: "Icon assigned to the private location.",
			},
			"keys": schema.SetAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Sensitive:   true,
				Description: "Private location API keys.",
			},
		},
	}
}

func (r *PrivateLocationResource) Configure(
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

func (r *PrivateLocationResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *PrivateLocationResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan PrivateLocationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var desiredModel checkly.PrivateLocation
	resp.Diagnostics.Append(plan.Render(ctx, &desiredModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realizedModel, err := r.client.CreatePrivateLocation(ctx, desiredModel)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Checkly Private Location",
			fmt.Sprintf("Could not create private location, unexpected error: %s", err),
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

func (r *PrivateLocationResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state PrivateLocationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeletePrivateLocation(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Checkly Private Location",
			fmt.Sprintf("Could not delete private location, unexpected error: %s", err),
		)

		return
	}
}

func (r *PrivateLocationResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state PrivateLocationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realizedModel, err := r.client.GetPrivateLocation(ctx, state.ID.ValueString())
	if err != nil {
		if SDKIsHTTPNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading Checkly Private Location",
			fmt.Sprintf("Could not retrieve private location, unexpected error: %s", err),
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

func (r *PrivateLocationResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan PrivateLocationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var desiredModel checkly.PrivateLocation
	resp.Diagnostics.Append(plan.Render(ctx, &desiredModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realizedModel, err := r.client.UpdatePrivateLocation(
		ctx,
		plan.ID.ValueString(),
		desiredModel,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Checkly Private Location",
			fmt.Sprintf("Could not update private location, unexpected error: %s", err),
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

var (
	_ ResourceModel[checkly.PrivateLocation] = (*PrivateLocationResourceModel)(nil)
)

type PrivateLocationResourceModel struct {
	ID          types.String `tfsdk:"id"`
	LastUpdated types.String `tfsdk:"last_updated"` // FIXME: Keep this? Old code did not have it.
	Name        types.String `tfsdk:"name"`
	SlugName    types.String `tfsdk:"slug_name"`
	Icon        types.String `tfsdk:"icon"`
	Keys        types.Set    `tfsdk:"keys"`
}

func (m *PrivateLocationResourceModel) Refresh(ctx context.Context, from *checkly.PrivateLocation, flags RefreshFlags) diag.Diagnostics {
	if flags.Created() {
		m.ID = types.StringValue(from.ID)
	}

	if flags.Created() || flags.Updated() {
		m.LastUpdated = LastUpdatedNow()
	}

	m.Name = types.StringValue(from.Name)
	m.SlugName = types.StringValue(from.SlugName)
	m.Icon = types.StringValue(from.Icon)

	var keyValues []attr.Value
	for _, key := range from.Keys {
		keyValues = append(keyValues, types.StringValue(key.RawKey))
	}

	keys, diags := types.SetValue(types.StringType, keyValues)
	if diags.HasError() {
		return diags
	}

	m.Keys = keys

	return nil
}

func (m *PrivateLocationResourceModel) Render(ctx context.Context, into *checkly.PrivateLocation) diag.Diagnostics {
	into.Name = m.Name.ValueString()
	into.SlugName = m.SlugName.ValueString()
	into.Icon = m.Icon.ValueString()

	// Keys are intentionally not included.

	return nil
}
