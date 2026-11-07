package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

// MaintenanceWindowStatusPageVisibilityModel describes the
// status_page_visibility attribute data model.
type MaintenanceWindowStatusPageVisibilityModel struct {
	Enabled               types.Bool   `tfsdk:"enabled"`
	Severity              types.String `tfsdk:"severity"`
	AffectAllServices     types.Bool   `tfsdk:"affect_all_services"`
	NotifyOnStart         types.Bool   `tfsdk:"notify_on_start"`
	NotifyOnEnd           types.Bool   `tfsdk:"notify_on_end"`
	SuppressAutoIncidents types.Bool   `tfsdk:"suppress_auto_incidents"`
	ReminderMinutesBefore types.Set    `tfsdk:"reminder_minutes_before"`
	AutoStart             types.Bool   `tfsdk:"auto_start"`
	AutoEnd               types.Bool   `tfsdk:"auto_end"`
	ShowAffectedServices  types.Bool   `tfsdk:"show_affected_services"`
	StatusPageIDs         types.Set    `tfsdk:"status_page_ids"`
	ServiceIDs            types.Set    `tfsdk:"service_ids"`
}

var maintenanceWindowStatusPageVisibilityAttrTypes = map[string]attr.Type{
	"enabled":                 types.BoolType,
	"severity":                types.StringType,
	"affect_all_services":     types.BoolType,
	"notify_on_start":         types.BoolType,
	"notify_on_end":           types.BoolType,
	"suppress_auto_incidents": types.BoolType,
	"reminder_minutes_before": types.SetType{ElemType: types.Int64Type},
	"auto_start":              types.BoolType,
	"auto_end":                types.BoolType,
	"show_affected_services":  types.BoolType,
	"status_page_ids":         types.SetType{ElemType: types.StringType},
	"service_ids":             types.SetType{ElemType: types.StringType},
}

// maintenanceWindowStatusPageVisibilitySchema is deliberately an optional
// attribute without a computed value: removing it must reset the window's
// status page settings rather than keep the ones stored by the API. The
// nested defaults are safe because the object is not a set element; defaults
// inside set elements break the framework's lookup of configured element
// values.
func maintenanceWindowStatusPageVisibilitySchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional: true,
		Description: "Whether and how the maintenance window appears on status pages. " +
			"Omitting the attribute hides the window and unlinks it from all status pages and services. " +
			"Only service-based status page links are supported; links to status page components can't be managed here. " +
			"An account without the status page maintenance windows entitlement can only change `enabled` to `false`: " +
			"keep the rest of the attribute as it is, because removing it also changes the other settings, which is rejected.",
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Show the maintenance window on the linked status pages. All other settings in this attribute only take effect when this is `true`. (Default `false`).",
			},
			"severity": schema.StringAttribute{
				Optional:    true,
				Description: "The severity shown on the status page. Possible values are `MINOR`, `MEDIUM`, `MAJOR` and `CRITICAL`.",
				Validators: []validator.String{
					stringvalidator.OneOf("MINOR", "MEDIUM", "MAJOR", "CRITICAL"),
				},
			},
			"affect_all_services": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Mark every service on the linked status pages as affected. Can't be combined with `service_ids`. (Default `false`).",
			},
			"notify_on_start": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Email status page subscribers when the maintenance starts. (Default `false`).",
			},
			"notify_on_end": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Email status page subscribers when the maintenance ends. (Default `false`).",
			},
			"suppress_auto_incidents": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Suppress automatically created incidents for the linked services during the maintenance. (Default `false`).",
			},
			"reminder_minutes_before": schema.SetAttribute{
				Optional:    true,
				ElementType: types.Int64Type,
				Description: "Up to three reminders, in minutes before the maintenance starts (60 to 10080), sent to status page subscribers.",
				Validators: []validator.Set{
					setvalidator.SizeAtMost(3),
					setvalidator.ValueInt64sAre(int64validator.Between(60, 10080)),
				},
			},
			"auto_start": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Start the maintenance automatically at the scheduled time. (Default `true`).",
			},
			"auto_end": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Complete the maintenance automatically at the scheduled end time. (Default `true`).",
			},
			"show_affected_services": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Show which services the maintenance affects. When `false`, downtime during the maintenance counts against the services' uptime. (Default `true`).",
			},
			"status_page_ids": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "The IDs of the status pages (`checkly_status_page`) to show the maintenance window on. Requires `service_ids` or `affect_all_services`.",
			},
			"service_ids": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "The IDs of the affected status page services (`checkly_status_page_service`). Each service must be on one of the linked status pages.",
			},
		},
	}
}

// maintenanceWindowStatusPageVisibilityFromObject returns the zero value for a
// missing attribute, which the SDK sends as the API's defaults.
func maintenanceWindowStatusPageVisibilityFromObject(ctx context.Context, obj types.Object, diags *diag.Diagnostics) checkly.MaintenanceWindowStatusPageVisibility {
	model := objectAs[MaintenanceWindowStatusPageVisibilityModel](ctx, obj, diags)
	if model == nil {
		return checkly.MaintenanceWindowStatusPageVisibility{}
	}

	return checkly.MaintenanceWindowStatusPageVisibility{
		Enabled:               model.Enabled.ValueBool(),
		Severity:              checkly.IncidentSeverity(model.Severity.ValueString()),
		AffectAllServices:     model.AffectAllServices.ValueBool(),
		NotifyOnStart:         model.NotifyOnStart.ValueBool(),
		NotifyOnEnd:           model.NotifyOnEnd.ValueBool(),
		SuppressAutoIncidents: model.SuppressAutoIncidents.ValueBool(),
		ReminderMinutesBefore: intsFromInt64Set(ctx, model.ReminderMinutesBefore, diags),
		AutoStart:             boolPointerOrNil(model.AutoStart),
		AutoEnd:               boolPointerOrNil(model.AutoEnd),
		ShowAffectedServices:  boolPointerOrNil(model.ShowAffectedServices),
		StatusPageIDs:         stringsFromSet(ctx, model.StatusPageIDs, diags),
		ServiceIDs:            stringsFromSet(ctx, model.ServiceIDs, diags),
	}
}

// maintenanceWindowStatusPageVisibilityToObject returns null for default
// settings when the prior state has no value either, so a configuration
// without the attribute doesn't show a diff. A value that is in the prior
// state is always written back, which keeps an explicit all-defaults object
// stable. Empty sets from the API stay null where the prior value had them
// null.
func maintenanceWindowStatusPageVisibilityToObject(ctx context.Context, v checkly.MaintenanceWindowStatusPageVisibility, prior types.Object, diags *diag.Diagnostics) types.Object {
	priorModel := objectAs[MaintenanceWindowStatusPageVisibilityModel](ctx, prior, diags)
	if priorModel == nil && isDefaultMaintenanceWindowStatusPageVisibility(v) {
		return types.ObjectNull(maintenanceWindowStatusPageVisibilityAttrTypes)
	}
	if priorModel == nil {
		priorModel = &MaintenanceWindowStatusPageVisibilityModel{
			Severity:              types.StringNull(),
			ReminderMinutesBefore: types.SetNull(types.Int64Type),
			StatusPageIDs:         types.SetNull(types.StringType),
			ServiceIDs:            types.SetNull(types.StringType),
		}
	}

	obj, d := types.ObjectValueFrom(ctx, maintenanceWindowStatusPageVisibilityAttrTypes, MaintenanceWindowStatusPageVisibilityModel{
		Enabled:               types.BoolValue(v.Enabled),
		Severity:              stringValueOrNull(string(v.Severity)),
		AffectAllServices:     types.BoolValue(v.AffectAllServices),
		NotifyOnStart:         types.BoolValue(v.NotifyOnStart),
		NotifyOnEnd:           types.BoolValue(v.NotifyOnEnd),
		SuppressAutoIncidents: types.BoolValue(v.SuppressAutoIncidents),
		ReminderMinutesBefore: int64SetFromInts(ctx, v.ReminderMinutesBefore, priorModel.ReminderMinutesBefore, diags),
		AutoStart:             types.BoolValue(v.AutoStart == nil || *v.AutoStart),
		AutoEnd:               types.BoolValue(v.AutoEnd == nil || *v.AutoEnd),
		ShowAffectedServices:  types.BoolValue(v.ShowAffectedServices == nil || *v.ShowAffectedServices),
		StatusPageIDs:         stringSetFromAPINullIfUnset(ctx, v.StatusPageIDs, priorModel.StatusPageIDs, diags),
		ServiceIDs:            stringSetFromAPINullIfUnset(ctx, v.ServiceIDs, priorModel.ServiceIDs, diags),
	})
	diags.Append(d...)

	return obj
}

func isDefaultMaintenanceWindowStatusPageVisibility(v checkly.MaintenanceWindowStatusPageVisibility) bool {
	return !v.Enabled &&
		v.Severity == "" &&
		!v.AffectAllServices &&
		!v.NotifyOnStart &&
		!v.NotifyOnEnd &&
		!v.SuppressAutoIncidents &&
		len(v.ReminderMinutesBefore) == 0 &&
		(v.AutoStart == nil || *v.AutoStart) &&
		(v.AutoEnd == nil || *v.AutoEnd) &&
		(v.ShowAffectedServices == nil || *v.ShowAffectedServices) &&
		len(v.StatusPageIDs) == 0 &&
		len(v.ServiceIDs) == 0
}

// boolPointerOrNil returns nil for null and unknown values, which the SDK
// sends as the API's default.
func boolPointerOrNil(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	return value.ValueBoolPointer()
}

// intsFromInt64Set converts a set of integers to a Go slice. Null and unknown
// sets convert to nil.
func intsFromInt64Set(ctx context.Context, set types.Set, diags *diag.Diagnostics) []int {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}

	var values []int64
	diags.Append(set.ElementsAs(ctx, &values, false)...)

	ints := make([]int, len(values))
	for i, v := range values {
		ints[i] = int(v)
	}

	return ints
}

// int64SetFromInts converts integers from the API to a set, resolving the
// empty value against the prior data: when the API has none and none were
// configured, the attribute stays null.
func int64SetFromInts(ctx context.Context, values []int, prior types.Set, diags *diag.Diagnostics) types.Set {
	if len(values) == 0 && (prior.IsNull() || prior.IsUnknown()) {
		return types.SetNull(types.Int64Type)
	}

	int64s := make([]int64, len(values))
	for i, v := range values {
		int64s[i] = int64(v)
	}

	set, d := types.SetValueFrom(ctx, types.Int64Type, int64s)
	diags.Append(d...)

	return set
}
