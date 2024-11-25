package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
)

var (
	_ resource.Resource                = (*MaintenanceWindowsResource)(nil)
	_ resource.ResourceWithConfigure   = (*MaintenanceWindowsResource)(nil)
	_ resource.ResourceWithImportState = (*MaintenanceWindowsResource)(nil)
)

type MaintenanceWindowsResource struct {
	client checkly.Client
}

func NewMaintenanceWindowsResource() resource.Resource {
	return &MaintenanceWindowsResource{}
}

func (r *MaintenanceWindowsResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_maintenance_windows"
}

func (r *MaintenanceWindowsResource) Schema(
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
				Description: "The maintenance window name.",
			},
			"starts_at": schema.StringAttribute{
				Required:    true,
				Description: "The start date of the maintenance window.",
			},
			"ends_at": schema.StringAttribute{
				Required:    true,
				Description: "The end date of the maintenance window.",
			},
			"repeat_unit": schema.StringAttribute{
				Optional: true,
				Default:  nil,
				Validators: []validator.String{
					stringvalidator.OneOf("DAY", "WEEK", "MONTH"),
				},
				Description: "The repeat cadence for the maintenance window. Possible values `DAY`, `WEEK` and `MONTH`.",
			},
			"repeat_interval": schema.Int32Attribute{
				Optional:    true,
				Description: "The repeat interval of the maintenance window from the first occurrence.",
			},
			"repeat_ends_at": schema.StringAttribute{
				Optional:    true,
				Description: "The date on which the maintenance window should stop repeating.",
			},
			"tags": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "The names of the checks and groups maintenance window should apply to.",
			},
		},
	}
}

func (r *MaintenanceWindowsResource) Configure(
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

func (r *MaintenanceWindowsResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *MaintenanceWindowsResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan MaintenanceWindowsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var desiredModel checkly.MaintenanceWindow
	resp.Diagnostics.Append(plan.Render(ctx, &desiredModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realizedModel, err := r.client.CreateMaintenanceWindow(ctx, desiredModel)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Checkly Maintenance Window",
			fmt.Sprintf("Could not create maintenance window, unexpected error: %s", err),
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

func (r *MaintenanceWindowsResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state MaintenanceWindowsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, diags := MaintenanceWindowID.FromString(state.ID)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	err := r.client.DeleteMaintenanceWindow(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Checkly Maintenance Window",
			fmt.Sprintf("Could not delete maintenance window, unexpected error: %s", err),
		)

		return
	}
}

func (r *MaintenanceWindowsResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state MaintenanceWindowsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, diags := MaintenanceWindowID.FromString(state.ID)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	realizedModel, err := r.client.GetMaintenanceWindow(ctx, id)
	if err != nil {
		if SDKIsHTTPNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading Checkly Maintenance Window",
			fmt.Sprintf("Could not retrieve maintenance window, unexpected error: %s", err),
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

func (r *MaintenanceWindowsResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan MaintenanceWindowsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, diags := MaintenanceWindowID.FromString(plan.ID)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	var desiredModel checkly.MaintenanceWindow
	resp.Diagnostics.Append(plan.Render(ctx, &desiredModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realizedModel, err := r.client.UpdateMaintenanceWindow(
		ctx,
		id,
		desiredModel,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Checkly Maintenance Window",
			fmt.Sprintf("Could not update maintenance window, unexpected error: %s", err),
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

var MaintenanceWindowID = SDKIdentifier{
	Path:  path.Root("id"),
	Title: "Checkly Maintenance Window ID",
}

var (
	_ ResourceModel[checkly.MaintenanceWindow] = (*MaintenanceWindowsResourceModel)(nil)
)

type MaintenanceWindowsResourceModel struct {
	ID             types.String `tfsdk:"id"`
	LastUpdated    types.String `tfsdk:"last_updated"` // FIXME: Keep this? Old code did not have it.
	Name           types.String `tfsdk:"name"`
	StartsAt       types.String `tfsdk:"starts_at"`
	EndsAt         types.String `tfsdk:"ends_at"`
	RepeatUnit     types.String `tfsdk:"repeat_unit"`
	RepeatInterval types.Int32  `tfsdk:"repeat_interval"`
	RepeatEndsAt   types.String `tfsdk:"repeat_ends_at"`
	Tags           types.Set    `tfsdk:"tags"`
}

func (m *MaintenanceWindowsResourceModel) Refresh(ctx context.Context, from *checkly.MaintenanceWindow, flags RefreshFlags) diag.Diagnostics {
	if flags.Created() {
		m.ID = MaintenanceWindowID.IntoString(from.ID)
	}

	if flags.Created() || flags.Updated() {
		m.LastUpdated = LastUpdatedNow()
	}

	m.Name = types.StringValue(from.Name)
	m.StartsAt = types.StringValue(from.StartsAt)
	m.EndsAt = types.StringValue(from.EndsAt)
	m.RepeatUnit = types.StringValue(from.RepeatUnit)
	m.RepeatEndsAt = types.StringValue(from.RepeatEndsAt)
	m.RepeatInterval = types.Int32Value(int32(from.RepeatInterval))
	m.Tags = IntoUntypedStringSet(&from.Tags)

	return nil
}

func (m *MaintenanceWindowsResourceModel) Render(ctx context.Context, into *checkly.MaintenanceWindow) diag.Diagnostics {
	into.Name = m.Name.ValueString()
	into.StartsAt = m.StartsAt.ValueString()
	into.EndsAt = m.EndsAt.ValueString()
	into.RepeatUnit = m.RepeatUnit.ValueString()
	into.RepeatEndsAt = m.RepeatEndsAt.ValueString()
	into.RepeatInterval = int(m.RepeatInterval.ValueInt32())
	into.Tags = FromUntypedStringSet(m.Tags)

	return nil
}
