package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

var (
	_ resource.Resource                     = (*StatusPageV3ComponentResource)(nil)
	_ resource.ResourceWithConfigure        = (*StatusPageV3ComponentResource)(nil)
	_ resource.ResourceWithUpgradeState     = (*StatusPageV3ComponentResource)(nil)
	_ resource.ResourceWithImportState      = (*StatusPageV3ComponentResource)(nil)
	_ resource.ResourceWithConfigValidators = (*StatusPageV3ComponentResource)(nil)
)

var statusPageV3ComponentTypeValues = allowedValues[string]{
	{Value: "SERVICE", Description: "a monitored thing with its own status"},
	{Value: "GROUP", Description: "a container for other components"},
}

// statusPageV3CompositeImportState imports nested v3 status page resources,
// whose API routes need the page ID in addition to their own: the import ID
// is "<status_page_id>/<id>".
func statusPageV3CompositeImportState(ctx context.Context, resourceName string, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("invalid import ID %q, expected \"<status_page_id>/<%s_id>\"", req.ID, resourceName),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("status_page_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

// NewStatusPageV3ComponentResource returns a new
// checkly_status_page_v3_component resource.
func NewStatusPageV3ComponentResource() resource.Resource {
	return &StatusPageV3ComponentResource{}
}

// StatusPageV3ComponentResource implements the
// checkly_status_page_v3_component resource.
type StatusPageV3ComponentResource struct {
	client checkly.Client
}

// StatusPageV3ComponentResourceModel describes the resource data model.
type StatusPageV3ComponentResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	StatusPageID       types.String `tfsdk:"status_page_id"`
	Type               types.String `tfsdk:"type"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	DisplayOrder       types.Int64  `tfsdk:"display_order"`
	Hidden             types.Bool   `tfsdk:"hidden"`
	ShowHistoricalData types.Bool   `tfsdk:"show_historical_data"`
	ExpandedByDefault  types.Bool   `tfsdk:"expanded_by_default"`
	ParentID           types.String `tfsdk:"parent_id"`
}

func (r *StatusPageV3ComponentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_status_page_v3_component"
}

func (r *StatusPageV3ComponentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: sdkv2SchemaVersion + 1,
		Description: "A component of a v3 status page: either a SERVICE (a " +
			"monitored thing with its own status) or a GROUP (a container " +
			"for other components). Import uses the composite ID " +
			"`<status_page_id>/<component_id>`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the component.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status_page_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the v3 status page the component belongs to. A component cannot be moved to another page.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("SERVICE"),
				Description: "The type of the component. " + statusPageV3ComponentTypeValues.String() + " (Default `SERVICE`).",
				Validators: []validator.String{
					stringvalidator.OneOf(statusPageV3ComponentTypeValues.Values()...),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name shown on the status page (1 to 255 characters, without leading or trailing whitespace).",
				Validators: []validator.String{
					utf16LengthBetween(1, 255),
					trimmedStringValidator{},
				},
			},
			"description": statusPageV3OptionalString("An optional description shown next to the name."),
			"display_order": schema.Int64Attribute{
				Required:    true,
				Description: "The position among siblings; lower comes first.",
			},
			"hidden": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Hide the component from the public page while keeping it available for incidents and automation. (Default `false`).",
			},
			"show_historical_data": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Show the historical status (the uptime bar) of the component on the status page. (Default `true`).",
			},
			"expanded_by_default": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Render the group expanded when the status page loads. Only available on a `GROUP` component. (Default `false`).",
			},
			"parent_id": statusPageV3OptionalString("The ID of the GROUP component to nest this component under. Must be on the same status page."),
		},
	}
}

func (r *StatusPageV3ComponentResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		statusPageV3ComponentConfigurationValidator{},
	}
}

// statusPageV3ComponentConfigurationValidator rejects expanded_by_default on
// a SERVICE component at plan time rather than mid-apply.
type statusPageV3ComponentConfigurationValidator struct{}

func (v statusPageV3ComponentConfigurationValidator) Description(ctx context.Context) string {
	return "expanded_by_default is only available on a GROUP component"
}

func (v statusPageV3ComponentConfigurationValidator) MarkdownDescription(ctx context.Context) string {
	return "`expanded_by_default` is only available on a `GROUP` component"
}

func (v statusPageV3ComponentConfigurationValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var componentType types.String
	var expandedByDefault types.Bool

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("type"), &componentType)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("expanded_by_default"), &expandedByDefault)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// An unknown value (e.g. computed from another resource) is checked
	// again when the payload is built at apply time.
	if componentType.IsUnknown() || expandedByDefault.IsUnknown() {
		return
	}

	typeValue := "SERVICE"
	if !componentType.IsNull() {
		typeValue = componentType.ValueString()
	}

	if err := validateStatusPageV3ComponentConfiguration(checkly.StatusPageComponentV3Type(typeValue), expandedByDefault.ValueBool()); err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("expanded_by_default"),
			"Invalid Attribute Combination",
			err.Error(),
		)
	}
}

// The API's configuration shape is discriminated by type: a SERVICE only
// supports showHistoricalData and rejects expandedByDefault. The attribute's
// false default is indistinguishable from an explicit false, so only a true
// value on a SERVICE can (and must) be rejected.
func validateStatusPageV3ComponentConfiguration(componentType checkly.StatusPageComponentV3Type, expandedByDefault bool) error {
	if componentType != checkly.StatusPageComponentV3TypeGroup && expandedByDefault {
		return fmt.Errorf(`"expanded_by_default" is only available on a GROUP component`)
	}
	return nil
}

// UpgradeState upgrades state written by the provider's SDK v2
// implementation; see sdkv2StateUpgraders.
func (r *StatusPageV3ComponentResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return sdkv2StateUpgraders(ctx, r)
}

func (r *StatusPageV3ComponentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

// readInto fetches the component and updates the model to match it.
// notFound is true when the component, or its page, no longer exists.
func (r *StatusPageV3ComponentResource) readInto(ctx context.Context, model *StatusPageV3ComponentResourceModel) (notFound bool, err error) {
	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	component, err := r.client.GetStatusPageComponentV3(apiCtx, model.StatusPageID.ValueString(), model.ID.ValueString())
	if err != nil {
		return isNotFoundError(err), err
	}

	statusPageV3ComponentToModel(component, model)

	return false, nil
}

func (r *StatusPageV3ComponentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan StatusPageV3ComponentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	component, err := statusPageV3ComponentFromModel(&plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Status Page V3 Component", err.Error())
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.CreateStatusPageComponentV3(apiCtx, plan.StatusPageID.ValueString(), component)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Status Page V3 Component",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	plan.ID = types.StringValue(result.ID)

	if _, err := r.readInto(ctx, &plan); err != nil {
		// Keep the created component in state so that it is not orphaned.
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.AddError(
			"Unable to Read Status Page V3 Component",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *StatusPageV3ComponentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state StatusPageV3ComponentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	notFound, err := r.readInto(ctx, &state)
	if err != nil {
		if notFound {
			// Deleted remotely, either the component itself or the whole
			// page: mark the resource as gone.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Read Status Page V3 Component",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *StatusPageV3ComponentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan StatusPageV3ComponentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	component, err := statusPageV3ComponentFromModel(&plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Status Page V3 Component", err.Error())
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	_, err = r.client.UpdateStatusPageComponentV3(apiCtx, plan.StatusPageID.ValueString(), component.ID, component)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Status Page V3 Component",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	if _, err := r.readInto(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Status Page V3 Component",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *StatusPageV3ComponentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state StatusPageV3ComponentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	err := r.client.DeleteStatusPageComponentV3(apiCtx, state.StatusPageID.ValueString(), state.ID.ValueString())
	if err != nil {
		// Already gone, e.g. deleted along with its page.
		if isNotFoundError(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Delete Status Page V3 Component",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}
}

func (r *StatusPageV3ComponentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	statusPageV3CompositeImportState(ctx, "component", req, resp)
}

// statusPageV3ComponentFromModel converts the resource model to its API
// representation. The page ID travels through the URL path, not the
// payload.
func statusPageV3ComponentFromModel(model *StatusPageV3ComponentResourceModel) (checkly.StatusPageComponentV3, error) {
	componentType := checkly.StatusPageComponentV3Type(model.Type.ValueString())
	showHistoricalData := model.ShowHistoricalData.ValueBool()
	expandedByDefault := model.ExpandedByDefault.ValueBool()

	if err := validateStatusPageV3ComponentConfiguration(componentType, expandedByDefault); err != nil {
		return checkly.StatusPageComponentV3{}, err
	}

	configuration := &checkly.StatusPageComponentV3Configuration{
		ShowHistoricalData: &showHistoricalData,
	}
	if componentType == checkly.StatusPageComponentV3TypeGroup {
		configuration.ExpandedByDefault = &expandedByDefault
	}

	return checkly.StatusPageComponentV3{
		ID:            model.ID.ValueString(),
		Type:          componentType,
		Name:          model.Name.ValueString(),
		Description:   model.Description.ValueString(),
		DisplayOrder:  int(model.DisplayOrder.ValueInt64()),
		Hidden:        model.Hidden.ValueBool(),
		Configuration: configuration,
		ParentID:      model.ParentID.ValueString(),
	}, nil
}

// statusPageV3ComponentToModel updates the model to match the API
// representation of the component.
func statusPageV3ComponentToModel(c *checkly.StatusPageComponentV3, model *StatusPageV3ComponentResourceModel) {
	// Reads return the full configuration of the component's type; fall back
	// to the API defaults defensively. ExpandedByDefault is absent on a
	// SERVICE, where false matches the attribute's default.
	showHistoricalData := true
	expandedByDefault := false
	if c.Configuration != nil {
		if c.Configuration.ShowHistoricalData != nil {
			showHistoricalData = *c.Configuration.ShowHistoricalData
		}
		if c.Configuration.ExpandedByDefault != nil {
			expandedByDefault = *c.Configuration.ExpandedByDefault
		}
	}

	model.ID = types.StringValue(c.ID)
	model.StatusPageID = types.StringValue(c.StatusPageID)
	model.Type = types.StringValue(string(c.Type))
	model.Name = types.StringValue(c.Name)
	model.Description = types.StringValue(c.Description)
	model.DisplayOrder = types.Int64Value(int64(c.DisplayOrder))
	model.Hidden = types.BoolValue(c.Hidden)
	model.ShowHistoricalData = types.BoolValue(showHistoricalData)
	model.ExpandedByDefault = types.BoolValue(expandedByDefault)
	model.ParentID = types.StringValue(c.ParentID)
}
