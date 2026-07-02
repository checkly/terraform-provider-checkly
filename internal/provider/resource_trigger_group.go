package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
)

var (
	_ resource.Resource                = (*TriggerGroupResource)(nil)
	_ resource.ResourceWithConfigure   = (*TriggerGroupResource)(nil)
	_ resource.ResourceWithImportState = (*TriggerGroupResource)(nil)
)

// NewTriggerGroupResource returns a new checkly_trigger_group resource.
func NewTriggerGroupResource() resource.Resource {
	return &TriggerGroupResource{}
}

// TriggerGroupResource implements the checkly_trigger_group resource.
type TriggerGroupResource struct {
	client checkly.Client
}

// TriggerGroupResourceModel describes the resource data model.
type TriggerGroupResourceModel struct {
	ID      types.String `tfsdk:"id"`
	GroupID types.Int64  `tfsdk:"group_id"`
	Token   types.String `tfsdk:"token"`
	URL     types.String `tfsdk:"url"`
}

func (r *TriggerGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_trigger_group"
}

func (r *TriggerGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Triggers allow you to run checks in a group from your CI/CD pipeline or from an external system using a webhook URL.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the trigger.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"group_id": schema.Int64Attribute{
				Required:    true,
				Description: "The id of the group that you want to attach the trigger to.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"token": schema.StringAttribute{
				Computed:    true,
				Description: "The token value created to trigger the group",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"url": schema.StringAttribute{
				Computed:    true,
				Description: "The request URL to trigger the group run.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *TriggerGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *TriggerGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TriggerGroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.CreateTriggerGroup(apiCtx, plan.GroupID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Trigger Group",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	triggerGroupToModel(result, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TriggerGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TriggerGroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.GetTriggerGroup(apiCtx, state.GroupID.ValueInt64())
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Read Trigger Group",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	triggerGroupToModel(result, &state)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update never modifies the remote trigger: the only configurable attribute
// (group_id) requires replacement and everything else is computed, so a plan
// reaching Update is by definition identical to state.
func (r *TriggerGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TriggerGroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TriggerGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TriggerGroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	err := r.client.DeleteTriggerGroup(apiCtx, state.GroupID.ValueInt64())
	if err != nil {
		if isNotFoundError(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Delete Trigger Group",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}
}

// ImportState imports a trigger by the ID of the group it is attached to
// (the Checkly API only addresses group triggers by group ID). The id
// attribute is temporarily set to the import identifier and corrected to the
// trigger's own numeric ID by the Read that follows import.
func (r *TriggerGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	groupID, err := decodeNumericID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Trigger Group Import ID",
			fmt.Sprintf("The import ID must be the numeric ID of the group the trigger is attached to: %s", err),
		)
		return
	}

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_id"), groupID)...)
}

// triggerGroupToModel updates the model to match the API representation of
// the trigger. The url is not stored by the API; the SDK derives it from the
// API base URL, the group ID and the token.
func triggerGroupToModel(trigger *checkly.TriggerGroup, model *TriggerGroupResourceModel) {
	model.ID = types.StringValue(encodeNumericID(trigger.ID))
	// All trigger API endpoints address the trigger by group ID, so the
	// model always has the correct group_id already; only take the API's
	// word for it when the response actually includes one.
	if trigger.GroupId != 0 {
		model.GroupID = types.Int64Value(trigger.GroupId)
	}
	model.Token = types.StringValue(trigger.Token)
	model.URL = types.StringValue(trigger.URL)
}
