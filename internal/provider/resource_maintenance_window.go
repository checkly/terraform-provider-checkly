package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

var (
	_ resource.Resource                   = (*MaintenanceWindowResource)(nil)
	_ resource.ResourceWithConfigure      = (*MaintenanceWindowResource)(nil)
	_ resource.ResourceWithUpgradeState   = (*MaintenanceWindowResource)(nil)
	_ resource.ResourceWithImportState    = (*MaintenanceWindowResource)(nil)
	_ resource.ResourceWithMoveState      = (*MaintenanceWindowResource)(nil)
	_ resource.ResourceWithModifyPlan     = (*MaintenanceWindowResource)(nil)
	_ resource.ResourceWithValidateConfig = (*MaintenanceWindowResource)(nil)
)

// NewMaintenanceWindowResource returns a new checkly_maintenance_window
// resource.
func NewMaintenanceWindowResource() resource.Resource {
	return &MaintenanceWindowResource{}
}

// MaintenanceWindowResource implements the checkly_maintenance_window
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

	Timezone             types.String `tfsdk:"timezone"`
	PauseAllChecks       types.Bool   `tfsdk:"pause_all_checks"`
	SilenceAlertsTags    types.Set    `tfsdk:"silence_alerts_tags"`
	SilenceAllAlerts     types.Bool   `tfsdk:"silence_all_alerts"`
	Description          types.String `tfsdk:"description"`
	StatusPageVisibility types.Object `tfsdk:"status_page_visibility"`
}

func (r *MaintenanceWindowResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_maintenance_window"
}

func (r *MaintenanceWindowResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     sdkv2SchemaVersion + 1,
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
				Description: "The date on which the maintenance window should stop repeating, interpreted as a calendar date in `timezone`.",
			},
			"tags": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "The tags of the checks and groups that are paused during the maintenance window. Ignored when `pause_all_checks` is `true`.",
			},
			"timezone": schema.StringAttribute{
				Optional: true,
				Description: "The named IANA time zone used to schedule recurring occurrences, e.g. `America/New_York`. " +
					"Occurrences keep the same local time across daylight-saving changes. " +
					"`starts_at` and `ends_at` remain absolute instants; the time zone does not reinterpret them. " +
					"Use the canonical IANA name (e.g. `America/New_York`, not `US/Eastern`): the API stores canonical names, " +
					"so an alias shows as a change on every plan. UTC offsets such as `+05:00` or `Etc/GMT+5` are not accepted. " +
					"Defaults to `UTC`. The time zone cannot be changed while a maintenance is active.",
				Validators: []validator.String{
					maintenanceWindowTimezoneValidator{},
				},
			},
			"pause_all_checks": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Pause every check in the account during the maintenance window, regardless of `tags`. Defaults to `false`.",
			},
			"silence_alerts_tags": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "The tags of the checks and groups whose alerts are silenced during the maintenance window. Ignored when `silence_all_alerts` is `true`.",
			},
			"silence_all_alerts": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Silence alerts for every check in the account during the maintenance window, regardless of `silence_alerts_tags`. Defaults to `false`.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "A description of the maintenance window. When the window is visible on status pages, the description is shown there too.",
			},
			"status_page_visibility": maintenanceWindowStatusPageVisibilitySchema(),
		},
	}
}

// UpgradeState upgrades state written by the provider's SDK v2
// implementation; see sdkv2StateUpgraders.
func (r *MaintenanceWindowResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return sdkv2StateUpgraders(ctx, r)
}

// ModifyPlan applies default values to the optional+computed attributes of
// status_page_visibility.component_configs. Schema defaults cannot be used
// for attributes nested in set elements, so the plan is adjusted here
// instead. The defaults are applied through the parent object, which is left
// alone when it is null: setting the nested path directly would create the
// object in the plan of every window without status_page_visibility.
func (r *MaintenanceWindowResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planNestedDefaults(ctx, req, resp, path.Root("status_page_visibility"), objectWithSetDefaults("component_configs", maintenanceWindowComponentConfigsWithDefaults))
}

// ValidateConfig rejects status_page_visibility.component_configs that the
// API would reject on apply.
func (r *MaintenanceWindowResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var visibility types.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("status_page_visibility"), &visibility)...)
	if resp.Diagnostics.HasError() {
		return
	}

	validateMaintenanceWindowComponentConfigs(ctx, visibility, &resp.Diagnostics)
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

// MoveState enables `moved` blocks from the legacy checkly_maintenance_windows
// resource type. Both resource types share the exact same schema, so the
// source state can be used as the target state without transformation.
func (r *MaintenanceWindowResource) MoveState(ctx context.Context) []resource.StateMover {
	return renamedResourceStateMovers(ctx, r, "checkly_maintenance_windows", "checkly_maintenance_window")
}

func maintenanceWindowFromModel(ctx context.Context, model *MaintenanceWindowResourceModel, diags *diag.Diagnostics) checkly.MaintenanceWindow {
	return checkly.MaintenanceWindow{
		Name:           model.Name.ValueString(),
		StartsAt:       model.StartsAt.ValueString(),
		EndsAt:         model.EndsAt.ValueString(),
		RepeatUnit:     checkly.MaintenanceWindowRepeatUnit(model.RepeatUnit.ValueString()),
		RepeatInterval: int(model.RepeatInterval.ValueInt64()),
		RepeatEndsAt:   model.RepeatEndsAt.ValueString(),
		Tags:           stringsFromSet(ctx, model.Tags, diags),
		// The SDK sends every field, and unset values as the API's defaults,
		// so removing an attribute from the configuration resets it.
		Timezone:             model.Timezone.ValueString(),
		PauseAllChecks:       model.PauseAllChecks.ValueBool(),
		SilenceAlertsTags:    stringsFromSet(ctx, model.SilenceAlertsTags, diags),
		SilenceAllAlerts:     model.SilenceAllAlerts.ValueBool(),
		Description:          model.Description.ValueString(),
		StatusPageVisibility: maintenanceWindowStatusPageVisibilityFromObject(ctx, model.StatusPageVisibility, diags),
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
	model.RepeatUnit = stringValueOrNull(string(mw.RepeatUnit))
	model.RepeatInterval = maintenanceWindowRepeatInterval(mw.RepeatInterval, model.RepeatInterval)
	model.RepeatEndsAt = stringValueOrPriorEmpty(mw.RepeatEndsAt, model.RepeatEndsAt)
	model.Tags = stringSetFromAPINullIfUnset(ctx, mw.Tags, model.Tags, diags)
	model.Timezone = maintenanceWindowTimezone(mw.Timezone, model.Timezone)
	model.PauseAllChecks = types.BoolValue(mw.PauseAllChecks)
	model.SilenceAlertsTags = stringSetFromAPINullIfUnset(ctx, mw.SilenceAlertsTags, model.SilenceAlertsTags, diags)
	model.SilenceAllAlerts = types.BoolValue(mw.SilenceAllAlerts)
	model.Description = stringValueOrPriorEmpty(mw.Description, model.Description)
	model.StatusPageVisibility = maintenanceWindowStatusPageVisibilityToObject(ctx, mw.StatusPageVisibility, model.StatusPageVisibility, diags)
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

// maintenanceWindowTimezone converts the time zone from the API. The API
// stores canonical names, reporting a configured value with different casing,
// and reports a UTC window as null or as "UTC" depending on its history. A
// value equivalent to the prior one keeps the prior value so that neither
// shows a diff; otherwise UTC maps to null, the value of an unconfigured
// time zone.
func maintenanceWindowTimezone(value string, prior types.String) types.String {
	if !prior.IsNull() && !prior.IsUnknown() && timezonesEquivalent(value, prior.ValueString()) {
		return prior
	}

	if timezonesEquivalent(value, "") {
		return types.StringNull()
	}

	return types.StringValue(value)
}

// fixedOffsetTimezoneRegex matches UTC offsets, which the API rejects because
// they are not named IANA zones: "+05:00" style identifiers and the
// fixed-offset Etc/GMT±N zones. Etc/GMT+0 and Etc/GMT-0 are aliases of UTC and
// are accepted.
var fixedOffsetTimezoneRegex = regexp.MustCompile(`(?i)^[+-]|^Etc/GMT[+-]0*[1-9]\d?$`)

var _ validator.String = maintenanceWindowTimezoneValidator{}

// maintenanceWindowTimezoneValidator rejects UTC offsets at plan time.
type maintenanceWindowTimezoneValidator struct{}

func (v maintenanceWindowTimezoneValidator) Description(ctx context.Context) string {
	return "value must be a named IANA time zone, not a UTC offset"
}

func (v maintenanceWindowTimezoneValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v maintenanceWindowTimezoneValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	if fixedOffsetTimezoneRegex.MatchString(value) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Time Zone",
			fmt.Sprintf("%s must be a named IANA time zone such as \"America/New_York\", got %q; UTC offsets are not supported", req.Path, value),
		)
	}
}

// utcTimezoneAliases are the names the API resolves to UTC. The API may report
// a UTC window as "UTC" or as unset, so all of them count as the same value.
var utcTimezoneAliases = map[string]bool{
	"":              true,
	"utc":           true,
	"etc/utc":       true,
	"etc/uct":       true,
	"uct":           true,
	"gmt":           true,
	"etc/gmt":       true,
	"gmt0":          true,
	"etc/gmt0":      true,
	"zulu":          true,
	"etc/zulu":      true,
	"universal":     true,
	"etc/universal": true,
	"greenwich":     true,
	"etc/greenwich": true,
	"etc/gmt+0":     true,
	"etc/gmt-0":     true,
	"gmt+0":         true,
	"gmt-0":         true,
}

// timezonesEquivalent reports whether two timezone values schedule the same
// way. The API canonicalizes the casing of stored names, so names are compared
// case-insensitively.
func timezonesEquivalent(a, b string) bool {
	a, b = strings.ToLower(a), strings.ToLower(b)
	if utcTimezoneAliases[a] && utcTimezoneAliases[b] {
		return true
	}
	return a == b
}
