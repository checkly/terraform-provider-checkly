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
	"github.com/checkly/terraform-provider-checkly/internal/sdkutil"
)

var (
	_ resource.Resource                = (*TriggerCheckResource)(nil)
	_ resource.ResourceWithConfigure   = (*TriggerCheckResource)(nil)
	_ resource.ResourceWithImportState = (*TriggerCheckResource)(nil)
)

type TriggerCheckResource struct {
	client checkly.Client
}

func NewTriggerCheckResource() resource.Resource {
	return &TriggerCheckResource{}
}

func (r *TriggerCheckResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_trigger_check"
}

func (r *TriggerCheckResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":           IDResourceAttributeSchema,
			"last_updated": LastUpdatedAttributeSchema,
			"check_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the check that you want to attach the trigger to.",
			},
			"token": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The token value created to trigger the check.",
			},
			"url": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The request URL to trigger the check run.",
			},
		},
	}
}

func (r *TriggerCheckResource) Configure(
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

func (r *TriggerCheckResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *TriggerCheckResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan TriggerCheckResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realizedModel, err := r.client.CreateTriggerCheck(
		ctx,
		plan.CheckID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Checkly Trigger Check",
			fmt.Sprintf("Could not create trigger check, unexpected error: %s", err),
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

func (r *TriggerCheckResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state TriggerCheckResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteTriggerCheck(ctx, state.CheckID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Checkly Trigger Check",
			fmt.Sprintf("Could not delete trigger check, unexpected error: %s", err),
		)

		return
	}
}

func (r *TriggerCheckResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state TriggerCheckResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realizedModel, err := r.client.GetTriggerCheck(
		ctx,
		state.CheckID.ValueString(),
	)
	if err != nil {
		if sdkutil.IsHTTPNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading Checkly Trigger Check",
			fmt.Sprintf("Could not retrieve trigger check, unexpected error: %s", err),
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

func (r *TriggerCheckResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan TriggerCheckResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realizedModel, err := r.client.GetTriggerCheck(
		ctx,
		plan.CheckID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Checkly Trigger Check",
			fmt.Sprintf("Could not update trigger check, unexpected error: %s", err),
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

var TriggerCheckID = sdkutil.Identifier{
	Path:  path.Root("id"),
	Title: "Checkly Trigger Check ID",
}

var (
	_ ResourceModel[checkly.TriggerCheck] = (*TriggerCheckResourceModel)(nil)
)

type TriggerCheckResourceModel struct {
	ID          types.String `tfsdk:"id"`
	LastUpdated types.String `tfsdk:"last_updated"` // FIXME: Keep this? Old code did not have it.
	CheckID     types.String `tfsdk:"check_id"`
	Token       types.String `tfsdk:"token"`
	URL         types.String `tfsdk:"url"`
}

func (m *TriggerCheckResourceModel) Refresh(ctx context.Context, from *checkly.TriggerCheck, flags RefreshFlags) diag.Diagnostics {
	// TODO: Always update ID? CheckID, which is used for lookup, is user-modifiable,
	// and we could receive back a complete different ID.
	if flags.Created() {
		m.ID = TriggerCheckID.IntoString(from.ID)
	}

	if flags.Created() || flags.Updated() {
		m.LastUpdated = LastUpdatedNow()
	}

	m.CheckID = types.StringValue(from.CheckId)
	m.Token = types.StringValue(from.Token)
	m.URL = types.StringValue(from.URL)

	return nil
}

func (m *TriggerCheckResourceModel) Render(ctx context.Context, into *checkly.TriggerCheck) diag.Diagnostics {
	into.Token = m.Token.ValueString()
	into.URL = m.URL.ValueString()

	return nil
}
