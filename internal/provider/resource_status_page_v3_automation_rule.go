package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
)

var (
	_ resource.Resource                 = (*StatusPageV3AutomationRuleResource)(nil)
	_ resource.ResourceWithConfigure    = (*StatusPageV3AutomationRuleResource)(nil)
	_ resource.ResourceWithUpgradeState = (*StatusPageV3AutomationRuleResource)(nil)
	_ resource.ResourceWithImportState  = (*StatusPageV3AutomationRuleResource)(nil)
)

var statusPageV3TargetImpactValues = allowedValues[string]{
	{Value: "UNDER_MAINTENANCE"},
	{Value: "DEGRADED_PERFORMANCE"},
	{Value: "PARTIAL_OUTAGE"},
	{Value: "MAJOR_OUTAGE"},
}

// NewStatusPageV3AutomationRuleResource returns a new
// checkly_status_page_v3_automation_rule resource.
func NewStatusPageV3AutomationRuleResource() resource.Resource {
	return &StatusPageV3AutomationRuleResource{}
}

// StatusPageV3AutomationRuleResource implements the
// checkly_status_page_v3_automation_rule resource.
type StatusPageV3AutomationRuleResource struct {
	client checkly.Client
}

// StatusPageV3AutomationRuleResourceModel describes the resource data model.
type StatusPageV3AutomationRuleResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	StatusPageID          types.String `tfsdk:"status_page_id"`
	Name                  types.String `tfsdk:"name"`
	Enabled               types.Bool   `tfsdk:"enabled"`
	FirstUpdate           types.String `tfsdk:"first_update"`
	LastUpdate            types.String `tfsdk:"last_update"`
	NotifySubscribers     types.Bool   `tfsdk:"notify_subscribers"`
	CoolDownWindowMinutes types.Int64  `tfsdk:"cool_down_window_minutes"`
	Tags                  types.Set    `tfsdk:"tags"`
	Components            types.Set    `tfsdk:"component"`
}

// statusPageV3AutomationRuleComponentModel describes the component block
// data model.
type statusPageV3AutomationRuleComponentModel struct {
	ComponentID  types.String `tfsdk:"component_id"`
	TargetImpact types.String `tfsdk:"target_impact"`
}

var statusPageV3AutomationRuleComponentAttrTypes = map[string]attr.Type{
	"component_id":  types.StringType,
	"target_impact": types.StringType,
}

func (r *StatusPageV3AutomationRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_status_page_v3_automation_rule"
}

func (r *StatusPageV3AutomationRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: sdkv2SchemaVersion + 1,
		Description: "An automation rule of a v3 status page. When a check " +
			"whose tags overlap with the rule's tags fails, Checkly opens " +
			"one incident on the page impacting the listed components, and " +
			"resolves it when the check recovers. Import uses the composite " +
			"ID `<status_page_id>/<rule_id>`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the automation rule.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status_page_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the v3 status page the rule belongs to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the rule.",
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "A disabled rule never opens incidents. (Default `true`).",
			},
			"first_update": schema.StringAttribute{
				Required:    true,
				Description: "Body of the status update that opens the incident.",
			},
			"last_update": schema.StringAttribute{
				Required:    true,
				Description: "Body of the status update that resolves the incident.",
			},
			"notify_subscribers": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether subscribers are notified of the automated updates. (Default `true`).",
			},
			"cool_down_window_minutes": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(5),
				Description: "Minimum minutes after an automated incident before this rule may open the next one. `0` disables the cool down. (Default `5`).",
				Validators: []validator.Int64{
					int64validator.Between(0, 1440),
				},
			},
			"tags": schema.SetAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "A failing check matches this rule when it, or its group, carries ANY of these tags. At least one tag is required.",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"component": schema.SetNestedBlock{
				Description: "The components an automated incident impacts, with the impact each gets.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"component_id": schema.StringAttribute{
							Required:    true,
							Description: "The ID of the impacted component. Must be on the same status page.",
						},
						"target_impact": schema.StringAttribute{
							Required:    true,
							Description: "The impact set on the component while the incident is open. " + statusPageV3TargetImpactValues.String(),
							Validators: []validator.String{
								stringvalidator.OneOf(statusPageV3TargetImpactValues.Values()...),
							},
						},
					},
				},
			},
		},
	}
}

// UpgradeState upgrades state written by the provider's SDK v2
// implementation; see sdkv2StateUpgraders.
func (r *StatusPageV3AutomationRuleResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return sdkv2StateUpgraders(ctx, r)
}

func (r *StatusPageV3AutomationRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

// readInto fetches the rule and updates the model to match it. notFound is
// true when the rule, or its page, no longer exists.
func (r *StatusPageV3AutomationRuleResource) readInto(ctx context.Context, model *StatusPageV3AutomationRuleResourceModel, diags *diag.Diagnostics) (notFound bool, err error) {
	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	rule, err := r.client.GetStatusPageAutomationRuleV3(apiCtx, model.StatusPageID.ValueString(), model.ID.ValueString())
	if err != nil {
		return isNotFoundError(err), err
	}

	statusPageV3AutomationRuleToModel(ctx, rule, model, diags)

	return false, nil
}

func (r *StatusPageV3AutomationRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan StatusPageV3AutomationRuleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := statusPageV3AutomationRuleFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Invalid Status Page V3 Automation Rule", err.Error())
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.CreateStatusPageAutomationRuleV3(apiCtx, plan.StatusPageID.ValueString(), rule)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Status Page V3 Automation Rule",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	plan.ID = types.StringValue(result.ID)

	if _, err := r.readInto(ctx, &plan, &resp.Diagnostics); err != nil {
		// Keep the created rule in state so that it is not orphaned.
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.AddError(
			"Unable to Read Status Page V3 Automation Rule",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *StatusPageV3AutomationRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state StatusPageV3AutomationRuleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	notFound, err := r.readInto(ctx, &state, &resp.Diagnostics)
	if err != nil {
		if notFound {
			// Deleted remotely, either the rule itself or the whole page:
			// mark the resource as gone.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Read Status Page V3 Automation Rule",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *StatusPageV3AutomationRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan StatusPageV3AutomationRuleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := statusPageV3AutomationRuleFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Invalid Status Page V3 Automation Rule", err.Error())
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	_, err = r.client.UpdateStatusPageAutomationRuleV3(apiCtx, plan.StatusPageID.ValueString(), rule.ID, rule)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Status Page V3 Automation Rule",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	if _, err := r.readInto(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Status Page V3 Automation Rule",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *StatusPageV3AutomationRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state StatusPageV3AutomationRuleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	err := r.client.DeleteStatusPageAutomationRuleV3(apiCtx, state.StatusPageID.ValueString(), state.ID.ValueString())
	if err != nil {
		// Already gone (deleted along with its page): nothing left to do.
		if isNotFoundError(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Delete Status Page V3 Automation Rule",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}
}

func (r *StatusPageV3AutomationRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	statusPageV3CompositeImportState(ctx, "rule", req, resp)
}

// statusPageV3AutomationRuleFromModel converts the resource model to its API
// representation. The page ID travels through the URL path, not the
// payload. A non-nil error reports an invalid configuration.
func statusPageV3AutomationRuleFromModel(ctx context.Context, model *StatusPageV3AutomationRuleResourceModel, diags *diag.Diagnostics) (checkly.StatusPageAutomationRuleV3, error) {
	components := []checkly.StatusPageAutomationRuleComponentV3{}

	if !model.Components.IsNull() && !model.Components.IsUnknown() {
		var componentModels []statusPageV3AutomationRuleComponentModel
		diags.Append(model.Components.ElementsAs(ctx, &componentModels, false)...)
		if diags.HasError() {
			return checkly.StatusPageAutomationRuleV3{}, nil
		}

		seen := map[string]bool{}
		for _, component := range componentModels {
			componentID := component.ComponentID.ValueString()
			// The API stores one impact per (rule, component); a duplicate
			// would surface as an opaque server error.
			if seen[componentID] {
				return checkly.StatusPageAutomationRuleV3{}, fmt.Errorf("component %q is listed more than once", componentID)
			}
			seen[componentID] = true
			components = append(components, checkly.StatusPageAutomationRuleComponentV3{
				ComponentID:  componentID,
				TargetImpact: checkly.StatusPageTargetImpactV3(component.TargetImpact.ValueString()),
			})
		}
	}

	return checkly.StatusPageAutomationRuleV3{
		ID:                    model.ID.ValueString(),
		Name:                  model.Name.ValueString(),
		Enabled:               model.Enabled.ValueBool(),
		FirstUpdate:           model.FirstUpdate.ValueString(),
		LastUpdate:            model.LastUpdate.ValueString(),
		NotifySubscribers:     model.NotifySubscribers.ValueBool(),
		CoolDownWindowMinutes: int(model.CoolDownWindowMinutes.ValueInt64()),
		Tags:                  stringsFromSet(ctx, model.Tags, diags),
		Components:            components,
	}, nil
}

// statusPageV3AutomationRuleToModel updates the model to match the API
// representation of the rule.
func statusPageV3AutomationRuleToModel(ctx context.Context, rule *checkly.StatusPageAutomationRuleV3, model *StatusPageV3AutomationRuleResourceModel, diags *diag.Diagnostics) {
	componentModels := make([]statusPageV3AutomationRuleComponentModel, 0, len(rule.Components))
	for _, component := range rule.Components {
		componentModels = append(componentModels, statusPageV3AutomationRuleComponentModel{
			ComponentID:  types.StringValue(component.ComponentID),
			TargetImpact: types.StringValue(string(component.TargetImpact)),
		})
	}

	components, d := types.SetValueFrom(
		ctx,
		types.ObjectType{AttrTypes: statusPageV3AutomationRuleComponentAttrTypes},
		componentModels,
	)
	diags.Append(d...)

	model.ID = types.StringValue(rule.ID)
	model.StatusPageID = types.StringValue(rule.StatusPageID)
	model.Name = types.StringValue(rule.Name)
	model.Enabled = types.BoolValue(rule.Enabled)
	model.FirstUpdate = types.StringValue(rule.FirstUpdate)
	model.LastUpdate = types.StringValue(rule.LastUpdate)
	model.NotifySubscribers = types.BoolValue(rule.NotifySubscribers)
	model.CoolDownWindowMinutes = types.Int64Value(int64(rule.CoolDownWindowMinutes))
	model.Tags = setFromStrings(ctx, rule.Tags, diags)
	model.Components = components
}
