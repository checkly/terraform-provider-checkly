package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
)

var (
	_ resource.Resource                = (*TriggerCheckResource)(nil)
	_ resource.ResourceWithConfigure   = (*TriggerCheckResource)(nil)
	_ resource.ResourceWithImportState = (*TriggerCheckResource)(nil)
)

// NewTriggerCheckResource returns a new checkly_trigger_check resource.
func NewTriggerCheckResource() resource.Resource {
	return &TriggerCheckResource{}
}

// TriggerCheckResource implements the checkly_trigger_check resource.
type TriggerCheckResource struct {
	client checkly.Client
}

// TriggerCheckResourceModel describes the resource data model.
type TriggerCheckResourceModel struct {
	ID      types.String `tfsdk:"id"`
	CheckID types.String `tfsdk:"check_id"`
	Token   types.String `tfsdk:"token"`
	URL     types.String `tfsdk:"url"`
}

func (r *TriggerCheckResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_trigger_check"
}

func (r *TriggerCheckResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Triggers allow you to run checks from your CI/CD pipeline or from an external system using a webhook URL.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the trigger.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"check_id": schema.StringAttribute{
				Required:    true,
				Description: "The id of the check that you want to attach the trigger to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"token": schema.StringAttribute{
				Computed:    true,
				Description: "The token value created to trigger the check",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"url": schema.StringAttribute{
				Computed:    true,
				Description: "The request URL to trigger the check run.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *TriggerCheckResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *TriggerCheckResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TriggerCheckResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.CreateTriggerCheck(apiCtx, plan.CheckID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Trigger Check",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	triggerCheckToModel(result, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TriggerCheckResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TriggerCheckResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.GetTriggerCheck(apiCtx, state.CheckID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Read Trigger Check",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	triggerCheckToModel(result, &state)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update never modifies the remote trigger: the only configurable attribute
// (check_id) requires replacement and everything else is computed, so a plan
// reaching Update is by definition identical to state.
func (r *TriggerCheckResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TriggerCheckResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TriggerCheckResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TriggerCheckResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	err := r.client.DeleteTriggerCheck(apiCtx, state.CheckID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Delete Trigger Check",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}
}

// ImportState imports a trigger by the ID of the check it is attached to
// (the Checkly API only addresses check triggers by check ID). The id
// attribute is temporarily set to the import identifier and corrected to the
// trigger's own numeric ID by the Read that follows import.
func (r *TriggerCheckResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("check_id"), req.ID)...)
}

// triggerCheckToModel updates the model to match the API representation of
// the trigger. The url is not stored by the API; the SDK derives it from the
// API base URL, the check ID and the token.
func triggerCheckToModel(trigger *checkly.TriggerCheck, model *TriggerCheckResourceModel) {
	model.ID = types.StringValue(encodeNumericID(trigger.ID))
	// All trigger API endpoints address the trigger by check ID, so the
	// model always has the correct check_id already; only take the API's
	// word for it when the response actually includes one.
	if trigger.CheckId != "" {
		model.CheckID = types.StringValue(trigger.CheckId)
	}
	model.Token = types.StringValue(trigger.Token)
	model.URL = types.StringValue(trigger.URL)
}
