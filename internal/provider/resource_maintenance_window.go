package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
)

var (
	_ resource.Resource                = (*MaintenanceWindowResource)(nil)
	_ resource.ResourceWithConfigure   = (*MaintenanceWindowResource)(nil)
	_ resource.ResourceWithImportState = (*MaintenanceWindowResource)(nil)
)

// NewMaintenanceWindowResource returns a new checkly_maintenance_windows
// resource.
func NewMaintenanceWindowResource() resource.Resource {
	return &MaintenanceWindowResource{}
}

// MaintenanceWindowResource implements the checkly_maintenance_windows
// resource.
type MaintenanceWindowResource struct {
	client checkly.Client
}

// MaintenanceWindowResourceModel describes the resource data model.
type MaintenanceWindowResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	StartsAt       types.String `tfsdk:"starts_at"`
	EndsAt         types.String `tfsdk:"ends_at"`
	RepeatUnit     types.String `tfsdk:"repeat_unit"`
	RepeatInterval types.Int64  `tfsdk:"repeat_interval"`
	RepeatEndsAt   types.String `tfsdk:"repeat_ends_at"`
	Tags           types.Set    `tfsdk:"tags"`
}

func (r *MaintenanceWindowResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_maintenance_windows"
}

func (r *MaintenanceWindowResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Maintenance windows pause alerting during scheduled maintenance for the checks and groups they apply to.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the maintenance window.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
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
				Optional:    true,
				Description: "The repeat cadence for the maintenance window. Possible values `DAY`, `WEEK` and `MONTH`.",
				Validators: []validator.String{
					stringvalidator.OneOf("DAY", "WEEK", "MONTH"),
				},
			},
			"repeat_interval": schema.Int64Attribute{
				Optional:    true,
				Description: "The repeat interval of the maintenance window from the first occurrence.",
			},
			"repeat_ends_at": schema.StringAttribute{
				Optional:    true,
				Description: "The date on which the maintenance window should stop repeating.",
			},
			"tags": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "The names of the checks and groups maintenance window should apply to.",
			},
		},
	}
}

func (r *MaintenanceWindowResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *MaintenanceWindowResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MaintenanceWindowResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mw := maintenanceWindowFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.CreateMaintenanceWindow(apiCtx, mw)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Maintenance Window",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	plan.ID = types.StringValue(encodeNumericID(result.ID))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MaintenanceWindowResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state MaintenanceWindowResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := decodeNumericID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Maintenance Window ID",
			fmt.Sprintf("Maintenance window ID must be numeric: %s", err),
		)
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	mw, err := r.client.GetMaintenanceWindow(apiCtx, id)
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Read Maintenance Window",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	maintenanceWindowToModel(ctx, mw, &state, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *MaintenanceWindowResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan MaintenanceWindowResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := decodeNumericID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Maintenance Window ID",
			fmt.Sprintf("Maintenance window ID must be numeric: %s", err),
		)
		return
	}

	mw := maintenanceWindowFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	mw.ID = id

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	_, err = r.client.UpdateMaintenanceWindow(apiCtx, id, mw)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Maintenance Window",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MaintenanceWindowResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state MaintenanceWindowResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := decodeNumericID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Maintenance Window ID",
			fmt.Sprintf("Maintenance window ID must be numeric: %s", err),
		)
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	err = r.client.DeleteMaintenanceWindow(apiCtx, id)
	if err != nil {
		if isNotFoundError(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Delete Maintenance Window",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}
}

func (r *MaintenanceWindowResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func maintenanceWindowFromModel(ctx context.Context, model *MaintenanceWindowResourceModel, diags *diag.Diagnostics) checkly.MaintenanceWindow {
	return checkly.MaintenanceWindow{
		Name:           model.Name.ValueString(),
		StartsAt:       model.StartsAt.ValueString(),
		EndsAt:         model.EndsAt.ValueString(),
		RepeatUnit:     model.RepeatUnit.ValueString(),
		RepeatInterval: int(model.RepeatInterval.ValueInt64()),
		RepeatEndsAt:   model.RepeatEndsAt.ValueString(),
		Tags:           stringsFromSet(ctx, model.Tags, diags),
	}
}

// maintenanceWindowToModel updates the model to match the API representation
// of the maintenance window. The model must hold the prior state so that
// zero values the API reports for unset optional attributes can be resolved
// back to null.
func maintenanceWindowToModel(ctx context.Context, mw *checkly.MaintenanceWindow, model *MaintenanceWindowResourceModel, diags *diag.Diagnostics) {
	model.ID = types.StringValue(encodeNumericID(mw.ID))
	model.Name = types.StringValue(mw.Name)
	model.StartsAt = types.StringValue(mw.StartsAt)
	model.EndsAt = types.StringValue(mw.EndsAt)
	// The repeat_unit validator rejects an empty string, so an empty value
	// from the API always means the attribute is unset.
	model.RepeatUnit = stringValueOrNull(mw.RepeatUnit)
	model.RepeatInterval = maintenanceWindowRepeatInterval(mw.RepeatInterval, model.RepeatInterval)
	model.RepeatEndsAt = stringValueOrNull(mw.RepeatEndsAt)
	model.Tags = maintenanceWindowTagsToSet(ctx, mw.Tags, model.Tags, diags)
}

// maintenanceWindowRepeatInterval converts the repeat interval from the API,
// resolving the zero value against the prior data: the API reports 0 when no
// interval is set, which maps back to null unless 0 was explicitly
// configured.
func maintenanceWindowRepeatInterval(value int, prior types.Int64) types.Int64 {
	if value == 0 && (prior.IsNull() || prior.IsUnknown()) {
		return types.Int64Null()
	}

	return types.Int64Value(int64(value))
}

// maintenanceWindowTagsToSet converts tags from the API to a set, resolving
// the empty value against the prior data: when no tags exist remotely and
// none were configured, the attribute stays null.
func maintenanceWindowTagsToSet(ctx context.Context, tags []string, prior types.Set, diags *diag.Diagnostics) types.Set {
	if len(tags) == 0 && (prior.IsNull() || prior.IsUnknown()) {
		return types.SetNull(types.StringType)
	}

	return setFromStrings(ctx, tags, diags)
}
