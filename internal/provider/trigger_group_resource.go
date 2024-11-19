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
	_ resource.Resource                = (*TriggerGroupResource)(nil)
	_ resource.ResourceWithConfigure   = (*TriggerGroupResource)(nil)
	_ resource.ResourceWithImportState = (*TriggerGroupResource)(nil)
)

type TriggerGroupResource struct {
	client checkly.Client
}

func NewTriggerGroupResource() resource.Resource {
	return &TriggerGroupResource{}
}

func (r *TriggerGroupResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_trigger_group"
}

func (r *TriggerGroupResource) Schema(
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
			"group_id": schema.Int64Attribute{
				Required:    true,
				Description: "The ID of the group that you want to attach the trigger to.",
			},
			"token": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The token value created to trigger the group.",
			},
			"url": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The request URL to trigger the group run.",
			},
		},
	}
}

func (r *TriggerGroupResource) Configure(
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

func (r *TriggerGroupResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *TriggerGroupResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan TriggerGroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realizedModel, err := r.client.CreateTriggerGroup(
		ctx,
		plan.GroupID.ValueInt64(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Checkly Trigger Group",
			fmt.Sprintf("Could not create trigger group, unexpected error: %s", err),
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

func (r *TriggerGroupResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state TriggerGroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteTriggerGroup(ctx, state.GroupID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Checkly Trigger Group",
			fmt.Sprintf("Could not delete trigger group, unexpected error: %s", err),
		)

		return
	}
}

func (r *TriggerGroupResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state TriggerGroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: Check if we really have to do the weird 404 handling
	realizedModel, err := r.client.GetTriggerGroup(
		ctx,
		state.GroupID.ValueInt64(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Checkly Trigger Group",
			fmt.Sprintf("Could not retrieve trigger group, unexpected error: %s", err),
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

func (r *TriggerGroupResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan TriggerGroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realizedModel, err := r.client.GetTriggerGroup(
		ctx,
		plan.GroupID.ValueInt64(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Checkly Trigger Group",
			fmt.Sprintf("Could not update trigger group, unexpected error: %s", err),
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

var TriggerGroupID = SDKIdentifier{
	Path:  path.Root("id"),
	Title: "Checkly Trigger Group ID",
}

var (
	_ ResourceModel[checkly.TriggerGroup] = (*TriggerGroupResourceModel)(nil)
)

type TriggerGroupResourceModel struct {
	ID          types.String `tfsdk:"id"`
	LastUpdated types.String `tfsdk:"last_updated"` // FIXME: Keep this? Old code did not have it.
	GroupID     types.Int64  `tfsdk:"group_id"`
	Token       types.String `tfsdk:"token"`
	URL         types.String `tfsdk:"url"`
}

func (m *TriggerGroupResourceModel) Refresh(ctx context.Context, from *checkly.TriggerGroup, flags RefreshFlags) diag.Diagnostics {
	if flags.Created() {
		m.ID = TriggerGroupID.IntoString(from.ID)
	}

	if flags.Created() || flags.Updated() {
		m.LastUpdated = LastUpdatedNow()
	}

	m.GroupID = types.Int64Value(from.GroupId)
	m.Token = types.StringValue(from.Token)
	m.URL = types.StringValue(from.URL)

	return nil
}

func (m *TriggerGroupResourceModel) Render(ctx context.Context, into *checkly.TriggerGroup) diag.Diagnostics {
	into.Token = m.Token.ValueString()
	into.URL = m.URL.ValueString()

	return nil
}
