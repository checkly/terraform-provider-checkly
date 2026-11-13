package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
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

var maintenanceWindowRepeatUnitValues = allowedValues[string]{
	{Value: "DAY"},
	{Value: "WEEK"},
	{Value: "MONTH"},
}

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
				Description: "The maintenance window name. Must be between `1` and `200` characters.",
				Validators: []validator.String{
					utf16LengthBetween(1, 200),
				},
			},
			"starts_at": schema.StringAttribute{
				Required: true,
				Description: "The start of the maintenance window: an RFC 3339 date-time such as `2099-06-01T09:00:00Z` " +
					"or `2099-06-01T09:00:00+02:00`, or a date such as `2099-06-01`, which means midnight UTC.",
				Validators: []validator.String{
					maintenanceWindowDateValidator,
				},
			},
			"ends_at": schema.StringAttribute{
				Required: true,
				Description: "The end of the maintenance window: an RFC 3339 date-time such as `2099-06-01T11:00:00Z` " +
					"or `2099-06-01T11:00:00+02:00`, or a date such as `2099-06-02`, which means midnight UTC.",
				Validators: []validator.String{
					maintenanceWindowDateValidator,
				},
			},
			"repeat_unit": schema.StringAttribute{
				Optional:    true,
				Description: "The repeat cadence for the maintenance window. " + maintenanceWindowRepeatUnitValues.String(),
				Validators: []validator.String{
					stringvalidator.OneOf(maintenanceWindowRepeatUnitValues.Values()...),
				},
			},
			"repeat_interval": schema.Int64Attribute{
				Optional: true,
				Description: "How many `repeat_unit`s apart the occurrences of the maintenance window are, counted from the first " +
					"occurrence. Possible values are between `0` and `365`. `0`, like leaving it out, means the window does not repeat. " +
					"A value of `1` or more requires `repeat_unit`.",
				Validators: []validator.Int64{
					int64validator.Between(0, 365),
				},
			},
			"repeat_ends_at": schema.StringAttribute{
				Optional: true,
				Description: "The last calendar date, in `timezone` (UTC if not set), on which an occurrence may start. Use a date such as " +
					"`2099-12-31`. An RFC 3339 date-time is accepted too and counts as its UTC date: `2099-06-01T23:00:00-05:00` " +
					"means `2099-06-02`. Requires a `repeat_interval` of 1 or more. An empty string, like leaving it out, means " +
					"the window repeats indefinitely.",
				Validators: []validator.String{
					maintenanceWindowRepeatEndsAtValidator,
				},
			},
			"tags": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				Default:     emptySetDefault(types.StringType),
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
				Computed:    true,
				Default:     emptySetDefault(types.StringType),
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
				Description: "A description of the maintenance window. Must be at most `2000` characters. When the window is visible on status pages, the description is shown there too.",
				Validators: []validator.String{
					utf16LengthBetween(0, 2000),
				},
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

// ValidateConfig rejects repeat settings and
// status_page_visibility.component_configs that the API would reject or
// discard on apply.
func (r *MaintenanceWindowResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var visibility types.Object
	var repeatUnit, repeatEndsAt types.String
	var repeatInterval types.Int64
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("status_page_visibility"), &visibility)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("repeat_unit"), &repeatUnit)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("repeat_interval"), &repeatInterval)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("repeat_ends_at"), &repeatEndsAt)...)
	if resp.Diagnostics.HasError() {
		return
	}

	validateMaintenanceWindowRepeat(repeatUnit, repeatInterval, repeatEndsAt, &resp.Diagnostics)
	validateMaintenanceWindowComponentConfigs(ctx, visibility, &resp.Diagnostics)
}

// validateMaintenanceWindowRepeat requires repeat_unit and a repeat_interval
// of 1 or more together, and such a repeat_interval for repeat_ends_at. The
// API rejects a repeat_interval without a unit. Without a repeat_interval,
// it clears repeat_unit and repeat_ends_at on update, which would show as a
// change on every plan. Unknown values are left to the API.
func validateMaintenanceWindowRepeat(repeatUnit types.String, repeatInterval types.Int64, repeatEndsAt types.String, diags *diag.Diagnostics) {
	if repeatInterval.IsUnknown() {
		return
	}
	repeats := repeatInterval.ValueInt64() >= 1

	if repeats && repeatUnit.IsNull() {
		diags.AddAttributeError(
			path.Root("repeat_unit"),
			"Missing Repeat Unit",
			"repeat_unit is required when repeat_interval is 1 or more.",
		)
	}
	if !repeats && !repeatUnit.IsNull() && !repeatUnit.IsUnknown() {
		diags.AddAttributeError(
			path.Root("repeat_unit"),
			"Missing Repeat Interval",
			"repeat_unit requires a repeat_interval of 1 or more. Remove repeat_unit for a window that does not repeat.",
		)
	}
	if !repeats && !repeatEndsAt.IsUnknown() && repeatEndsAt.ValueString() != "" {
		diags.AddAttributeError(
			path.Root("repeat_ends_at"),
			"Missing Repeat Interval",
			"repeat_ends_at requires a repeat_interval of 1 or more. Remove repeat_ends_at for a window that does not repeat.",
		)
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
		addAPIError(&resp.Diagnostics, "Unable to Create Maintenance Window", err)
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
		addAPIError(&resp.Diagnostics, "Unable to Read Maintenance Window", err)
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
		addAPIError(&resp.Diagnostics, "Unable to Update Maintenance Window", err)
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
		addAPIError(&resp.Diagnostics, "Unable to Delete Maintenance Window", err)
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
	model.StartsAt = maintenanceWindowInstant(mw.StartsAt, model.StartsAt)
	model.EndsAt = maintenanceWindowInstant(mw.EndsAt, model.EndsAt)
	// The repeat_unit validator rejects an empty string, so an empty value
	// from the API always means the attribute is unset.
	model.RepeatUnit = stringValueOrNull(string(mw.RepeatUnit))
	model.RepeatInterval = maintenanceWindowRepeatInterval(mw.RepeatInterval, model.RepeatInterval)
	model.RepeatEndsAt = maintenanceWindowRepeatEndsAt(mw.RepeatEndsAt, model.RepeatEndsAt)
	model.Tags = setFromStrings(ctx, mw.Tags, diags)
	model.Timezone = maintenanceWindowTimezone(mw.Timezone, model.Timezone)
	model.PauseAllChecks = types.BoolValue(mw.PauseAllChecks)
	model.SilenceAlertsTags = setFromStrings(ctx, mw.SilenceAlertsTags, diags)
	model.SilenceAllAlerts = types.BoolValue(mw.SilenceAllAlerts)
	model.Description = stringValueOrPriorEmpty(mw.Description, model.Description)
	model.StatusPageVisibility = maintenanceWindowStatusPageVisibilityToObject(ctx, mw.StatusPageVisibility, model.StatusPageVisibility, diags)
}

// maintenanceWindowDateLayout is the date-only form of starts_at, ends_at and
// repeat_ends_at.
const maintenanceWindowDateLayout = "2006-01-02"

// maintenanceWindowDateRegex matches the forms parseMaintenanceWindowDate
// accepts. time.Parse alone is more lenient than JavaScript's Date, which the
// API parses with: it also takes a comma before the fraction of a second.
var maintenanceWindowDateRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}(?:T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2}))?$`)

// parseMaintenanceWindowDate parses an RFC 3339 date-time, or a date, which
// means midnight UTC, as the API parses it.
func parseMaintenanceWindowDate(value string) (time.Time, bool) {
	if !maintenanceWindowDateRegex.MatchString(value) {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, true
	}
	if t, err := time.Parse(maintenanceWindowDateLayout, value); err == nil {
		return t, true
	}
	return time.Time{}, false
}

// maintenanceWindowInstant converts starts_at or ends_at from the API, which
// reports them in UTC with milliseconds, such as "2099-06-01T09:00:00.000Z".
// The prior value is kept when it is the same instant at the millisecond
// precision the API stores, so that a configuration written in another form,
// such as a date or with a UTC offset, does not show a change on every plan.
func maintenanceWindowInstant(value string, prior types.String) types.String {
	if !prior.IsNull() && !prior.IsUnknown() {
		p, okPrior := parseMaintenanceWindowDate(prior.ValueString())
		v, okValue := parseMaintenanceWindowDate(value)
		if okPrior && okValue && p.Truncate(time.Millisecond).Equal(v.Truncate(time.Millisecond)) {
			return prior
		}
	}

	return types.StringValue(value)
}

// maintenanceWindowRepeatEndsAt converts repeat_ends_at from the API, which
// stores the UTC date of the configured value and reports it as midnight
// UTC, such as "2099-12-31T00:00:00.000Z". The prior value is kept when it
// has the same UTC date. No value maps to null, or to the empty string when
// that was the prior value.
func maintenanceWindowRepeatEndsAt(value string, prior types.String) types.String {
	if value != "" && !prior.IsNull() && !prior.IsUnknown() {
		p, okPrior := parseMaintenanceWindowDate(prior.ValueString())
		v, okValue := parseMaintenanceWindowDate(value)
		if okPrior && okValue && p.UTC().Format(maintenanceWindowDateLayout) == v.UTC().Format(maintenanceWindowDateLayout) {
			return prior
		}
	}

	return stringValueOrPriorEmpty(value, prior)
}

// maintenanceWindowDateValidator accepts the forms of starts_at and ends_at
// that parseMaintenanceWindowDate understands. The API accepts any string
// JavaScript's Date can parse, but only these forms are unambiguous: a
// date-time without a UTC offset, for example, is read in the server's time
// zone.
var maintenanceWindowDateValidator = predicateStringValidator{
	check: func(value string) bool {
		_, ok := parseMaintenanceWindowDate(value)
		return ok
	},
	message:     maintenanceWindowDateMessage,
	description: maintenanceWindowDateDescription,
}

// maintenanceWindowRepeatEndsAtValidator is maintenanceWindowDateValidator
// for repeat_ends_at, which also accepts the empty string as unset.
var maintenanceWindowRepeatEndsAtValidator = predicateStringValidator{
	check: func(value string) bool {
		_, ok := parseMaintenanceWindowDate(value)
		return ok || value == ""
	},
	message:     maintenanceWindowDateMessage,
	description: maintenanceWindowDateDescription + " or empty",
}

const (
	maintenanceWindowDateMessage     = `%q must be an RFC 3339 date-time with a UTC offset, such as "2099-06-01T09:00:00Z" or "2099-06-01T09:00:00+02:00", or a date, such as "2099-06-01", got: %s`
	maintenanceWindowDateDescription = `value must be an RFC 3339 date-time, such as "2099-06-01T09:00:00Z", or a date, such as "2099-06-01"`
)

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
