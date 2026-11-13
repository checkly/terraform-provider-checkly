package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

var maintenanceWindowComponentStatusValues = allowedValues[string]{
	{Value: string(checkly.StatusPageComponentStatusV3Operational)},
	{Value: string(checkly.StatusPageComponentStatusV3UnderMaintenance)},
	{Value: string(checkly.StatusPageComponentStatusV3DegradedPerformance)},
	{Value: string(checkly.StatusPageComponentStatusV3PartialOutage)},
	{Value: string(checkly.StatusPageComponentStatusV3MajorOutage)},
}

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
	ComponentConfigs      types.Set    `tfsdk:"component_configs"`
}

// MaintenanceWindowComponentConfigModel describes the data model of a
// status_page_visibility.component_configs element.
type MaintenanceWindowComponentConfigModel struct {
	StatusPageID        types.String `tfsdk:"status_page_id"`
	AffectAllComponents types.Bool   `tfsdk:"affect_all_components"`
	Components          types.Set    `tfsdk:"components"`
}

// MaintenanceWindowComponentImpactModel describes the data model of a
// status_page_visibility.component_configs.components element.
type MaintenanceWindowComponentImpactModel struct {
	ComponentID types.String `tfsdk:"component_id"`
	Status      types.String `tfsdk:"status"`
}

var maintenanceWindowComponentImpactAttrTypes = map[string]attr.Type{
	"component_id": types.StringType,
	"status":       types.StringType,
}

var maintenanceWindowComponentConfigAttrTypes = map[string]attr.Type{
	"status_page_id":        types.StringType,
	"affect_all_components": types.BoolType,
	"components":            types.SetType{ElemType: types.ObjectType{AttrTypes: maintenanceWindowComponentImpactAttrTypes}},
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
	"component_configs":       types.SetType{ElemType: types.ObjectType{AttrTypes: maintenanceWindowComponentConfigAttrTypes}},
}

// maintenanceWindowStatusPageVisibilitySchema is deliberately an optional
// attribute without a computed value: removing it must reset the window's
// status page settings rather than keep the ones stored by the API. The
// nested defaults are safe because the object is not a set element; defaults
// inside set elements break the framework's lookup of configured element
// values. The optional+computed attributes of component_configs are therefore
// defaulted in ModifyPlan instead; see
// maintenanceWindowComponentConfigsWithDefaults.
func maintenanceWindowStatusPageVisibilitySchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional: true,
		Description: "Whether and how the maintenance window appears on status pages. " +
			"Omitting the attribute hides the window and unlinks it from all status pages and services. " +
			"An account without the status page maintenance windows entitlement can only change `enabled` to `false`: " +
			"keep the rest of the attribute as it is, because removing it also changes the other settings, which is rejected. " +
			"For the same reason, such an account must copy the window's existing component links into `component_configs`, " +
			"since removing them is a change it can't make.",
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Show the maintenance window on the linked status pages. Defaults to `false`. All other settings in this attribute only take effect when this is `true`.",
			},
			"severity": schema.StringAttribute{
				Optional:    true,
				Description: "The severity shown on the status page. " + incidentSeverityValues.String(),
				Validators: []validator.String{
					stringvalidator.OneOf(incidentSeverityValues.Values()...),
				},
			},
			"affect_all_services": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Mark every service on the linked v2 status pages (`checkly_status_page`) as affected. Defaults to `false`. Can't be combined with `service_ids`. v3 status pages use `component_configs` instead.",
			},
			"notify_on_start": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Email status page subscribers when the maintenance starts. Defaults to `false`.",
			},
			"notify_on_end": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Email status page subscribers when the maintenance ends. Defaults to `false`.",
			},
			"suppress_auto_incidents": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Suppress automatically created incidents for the linked services during the maintenance. Defaults to `false`.",
			},
			"reminder_minutes_before": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				Default:     emptySetDefault(types.Int64Type),
				ElementType: types.Int64Type,
				Description: "Up to three reminders, in minutes before the maintenance starts, sent to status page subscribers. Possible values are between `60` and `10080`.",
				Validators: []validator.Set{
					setvalidator.SizeAtMost(3),
					setvalidator.ValueInt64sAre(int64validator.Between(60, 10080)),
				},
			},
			"auto_start": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Start the maintenance automatically at the scheduled time. Defaults to `true`.",
			},
			"auto_end": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Complete the maintenance automatically at the scheduled end time. Defaults to `true`.",
			},
			"show_affected_services": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Show which services the maintenance affects. When `false`, downtime during the maintenance counts against the services' uptime. Defaults to `true`.",
			},
			"status_page_ids": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				Default:     emptySetDefault(types.StringType),
				ElementType: types.StringType,
				Description: "The IDs of the status pages to show the maintenance window on. " +
					"A v2 status page (`checkly_status_page`) requires `service_ids` or `affect_all_services`, " +
					"and a v3 status page (`checkly_status_page_v3`) requires an entry in `component_configs`.",
			},
			"service_ids": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				Default:     emptySetDefault(types.StringType),
				ElementType: types.StringType,
				Description: "The IDs of the affected status page services (`checkly_status_page_service`). Each service must be on one of the linked v2 status pages.",
			},
			"component_configs": schema.SetNestedAttribute{
				Optional: true,
				Computed: true,
				Default:  emptySetDefault(types.ObjectType{AttrTypes: maintenanceWindowComponentConfigAttrTypes}),
				Description: "The affected components of each linked v3 status page (`checkly_status_page_v3`), at most one entry per page. " +
					"Omitting the attribute removes the window's component links.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"status_page_id": schema.StringAttribute{
							Required:    true,
							Description: "The ID of a v3 status page (`checkly_status_page_v3`). The page must also be listed in `status_page_ids`.",
						},
						"affect_all_components": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Mark every component of the status page as affected, including components added later. Defaults to `false`. Can't be combined with `components`.",
						},
						// The empty default is applied in ModifyPlan (see
						// maintenanceWindowComponentConfigsWithDefaults), as
						// defaults inside set elements break element identity.
						"components": schema.SetNestedAttribute{
							Optional:    true,
							Computed:    true,
							Description: "The affected components of the status page, each with the status it shows while the maintenance is in progress. Required unless `affect_all_components` is `true`; an empty set means the same as leaving it out.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"component_id": schema.StringAttribute{
										Required:    true,
										Description: "The ID of a component (`checkly_status_page_v3_component`) on the status page.",
									},
									"status": schema.StringAttribute{
										Optional:    true,
										Computed:    true,
										Description: "The status the component shows while the maintenance is in progress. " + maintenanceWindowComponentStatusValues.String() + " Defaults to `UNDER_MAINTENANCE`.",
										Validators: []validator.String{
											stringvalidator.OneOf(maintenanceWindowComponentStatusValues.Values()...),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// maintenanceWindowStatusPageVisibilityFromObject returns the zero value for a
// missing attribute, which the SDK sends as the API's defaults. Component
// configs are always sent, as an empty list when none are configured: the API
// keeps the stored ones when the field is left out, so removing them from the
// configuration must send the empty list to remove them.
func maintenanceWindowStatusPageVisibilityFromObject(ctx context.Context, obj types.Object, diags *diag.Diagnostics) checkly.MaintenanceWindowStatusPageVisibility {
	model := objectAs[MaintenanceWindowStatusPageVisibilityModel](ctx, obj, diags)
	if model == nil {
		return checkly.MaintenanceWindowStatusPageVisibility{
			ComponentConfigs: []checkly.MaintenanceWindowComponentConfig{},
		}
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
		ComponentConfigs:      maintenanceWindowComponentConfigsFromSet(ctx, model.ComponentConfigs, diags),
	}
}

// maintenanceWindowStatusPageVisibilityToObject returns null for default
// settings when the prior state has no value either, so a configuration
// without the attribute doesn't show a diff. A value that is in the prior
// state is always written back, which keeps an explicit all-defaults object
// stable.
func maintenanceWindowStatusPageVisibilityToObject(ctx context.Context, v checkly.MaintenanceWindowStatusPageVisibility, prior types.Object, diags *diag.Diagnostics) types.Object {
	if (prior.IsNull() || prior.IsUnknown()) && isDefaultMaintenanceWindowStatusPageVisibility(v) {
		return types.ObjectNull(maintenanceWindowStatusPageVisibilityAttrTypes)
	}

	obj, d := types.ObjectValueFrom(ctx, maintenanceWindowStatusPageVisibilityAttrTypes, MaintenanceWindowStatusPageVisibilityModel{
		Enabled:               types.BoolValue(v.Enabled),
		Severity:              stringValueOrNull(string(v.Severity)),
		AffectAllServices:     types.BoolValue(v.AffectAllServices),
		NotifyOnStart:         types.BoolValue(v.NotifyOnStart),
		NotifyOnEnd:           types.BoolValue(v.NotifyOnEnd),
		SuppressAutoIncidents: types.BoolValue(v.SuppressAutoIncidents),
		ReminderMinutesBefore: int64SetFromInts(ctx, v.ReminderMinutesBefore, diags),
		AutoStart:             types.BoolValue(v.AutoStart == nil || *v.AutoStart),
		AutoEnd:               types.BoolValue(v.AutoEnd == nil || *v.AutoEnd),
		ShowAffectedServices:  types.BoolValue(v.ShowAffectedServices == nil || *v.ShowAffectedServices),
		StatusPageIDs:         setFromStrings(ctx, v.StatusPageIDs, diags),
		ServiceIDs:            setFromStrings(ctx, v.ServiceIDs, diags),
		ComponentConfigs:      maintenanceWindowComponentConfigsToSet(ctx, v.ComponentConfigs, diags),
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
		len(v.ServiceIDs) == 0 &&
		len(v.ComponentConfigs) == 0
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

// int64SetFromInts converts integers from the API to a set. No integers
// convert to an empty set.
func int64SetFromInts(ctx context.Context, values []int, diags *diag.Diagnostics) types.Set {
	int64s := make([]int64, len(values))
	for i, v := range values {
		int64s[i] = int64(v)
	}

	set, d := types.SetValueFrom(ctx, types.Int64Type, int64s)
	diags.Append(d...)

	return set
}

// maintenanceWindowComponentConfigsFromSet converts the component_configs
// set to its API representation. Null and unknown sets convert to an empty
// (non-nil) slice, which removes the stored component configs. An entry
// without components sends none, which the SDK leaves out of the request.
func maintenanceWindowComponentConfigsFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) []checkly.MaintenanceWindowComponentConfig {
	configs := []checkly.MaintenanceWindowComponentConfig{}
	if set.IsNull() || set.IsUnknown() {
		return configs
	}

	var models []MaintenanceWindowComponentConfigModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return configs
	}

	for _, model := range models {
		var components []checkly.MaintenanceWindowComponentImpact
		if !model.Components.IsNull() && !model.Components.IsUnknown() {
			var impacts []MaintenanceWindowComponentImpactModel
			diags.Append(model.Components.ElementsAs(ctx, &impacts, false)...)
			for _, impact := range impacts {
				components = append(components, checkly.MaintenanceWindowComponentImpact{
					ComponentID: impact.ComponentID.ValueString(),
					Status:      checkly.StatusPageComponentStatusV3(impact.Status.ValueString()),
				})
			}
		}

		configs = append(configs, checkly.MaintenanceWindowComponentConfig{
			StatusPageID:        model.StatusPageID.ValueString(),
			AffectAllComponents: model.AffectAllComponents.ValueBool(),
			Components:          components,
		})
	}

	return configs
}

// maintenanceWindowComponentConfigsToSet converts the component configs from
// the API to a set. No configs convert to an empty set, and so do the
// components of an entry without any, such as one that affects all
// components, which the API reports with an empty components list.
func maintenanceWindowComponentConfigsToSet(ctx context.Context, configs []checkly.MaintenanceWindowComponentConfig, diags *diag.Diagnostics) types.Set {
	configType := types.ObjectType{AttrTypes: maintenanceWindowComponentConfigAttrTypes}
	impactType := types.ObjectType{AttrTypes: maintenanceWindowComponentImpactAttrTypes}
	models := make([]MaintenanceWindowComponentConfigModel, len(configs))
	for i, config := range configs {
		impacts := make([]MaintenanceWindowComponentImpactModel, len(config.Components))
		for j, component := range config.Components {
			impacts[j] = MaintenanceWindowComponentImpactModel{
				ComponentID: types.StringValue(component.ComponentID),
				Status:      types.StringValue(string(component.Status)),
			}
		}
		components, d := types.SetValueFrom(ctx, impactType, impacts)
		diags.Append(d...)

		models[i] = MaintenanceWindowComponentConfigModel{
			StatusPageID:        types.StringValue(config.StatusPageID),
			AffectAllComponents: types.BoolValue(config.AffectAllComponents),
			Components:          components,
		}
	}

	set, d := types.SetValueFrom(ctx, configType, models)
	diags.Append(d...)

	return set
}

// maintenanceWindowComponentConfigsWithDefaults applies the default values of
// the optional+computed component_configs attributes during plan
// modification, where schema defaults cannot be used because the attributes
// are nested in set elements: affect_all_components defaults to false,
// components to an empty set and a component's status to UNDER_MAINTENANCE,
// as in the API.
func maintenanceWindowComponentConfigsWithDefaults(ctx context.Context, set types.Set, diags *diag.Diagnostics) types.Set {
	return setElementsWithDefaults(ctx, set, func(m *MaintenanceWindowComponentConfigModel) {
		if m.AffectAllComponents.IsNull() || m.AffectAllComponents.IsUnknown() {
			m.AffectAllComponents = types.BoolValue(false)
		}
		if m.Components.IsNull() || m.Components.IsUnknown() {
			m.Components = types.SetValueMust(types.ObjectType{AttrTypes: maintenanceWindowComponentImpactAttrTypes}, []attr.Value{})
		}
		m.Components = maintenanceWindowComponentsWithDefaults(ctx, m.Components, diags)
	}, diags)
}

// maintenanceWindowComponentsWithDefaults applies the default status to the
// elements of a component_configs.components set.
func maintenanceWindowComponentsWithDefaults(ctx context.Context, set types.Set, diags *diag.Diagnostics) types.Set {
	return setElementsWithDefaults(ctx, set, func(m *MaintenanceWindowComponentImpactModel) {
		m.Status = stringOrDefault(m.Status, string(checkly.StatusPageComponentStatusV3UnderMaintenance))
	}, diags)
}

// validateMaintenanceWindowComponentConfigs rejects component_configs that
// the API would reject on apply. obj is the configured status_page_visibility
// value; checks that depend on values not yet known are skipped.
func validateMaintenanceWindowComponentConfigs(ctx context.Context, obj types.Object, diags *diag.Diagnostics) {
	visibility := objectAs[MaintenanceWindowStatusPageVisibilityModel](ctx, obj, diags)
	if visibility == nil || visibility.ComponentConfigs.IsNull() || visibility.ComponentConfigs.IsUnknown() {
		return
	}

	// The linked pages are only compared once all of them are known.
	var linked map[string]bool
	if !valueContainsUnknown(ctx, visibility.StatusPageIDs) {
		linked = map[string]bool{}
		for _, id := range stringsFromSet(ctx, visibility.StatusPageIDs, diags) {
			linked[id] = true
		}
	}

	configsPath := path.Root("status_page_visibility").AtName("component_configs")
	seenPages := map[string]bool{}
	for _, element := range visibility.ComponentConfigs.Elements() {
		entryObj, ok := element.(types.Object)
		if !ok {
			continue
		}
		entry := objectAs[MaintenanceWindowComponentConfigModel](ctx, entryObj, diags)
		if entry == nil {
			continue
		}
		entryPath := configsPath.AtSetValue(element)

		if !entry.StatusPageID.IsNull() && !entry.StatusPageID.IsUnknown() {
			pageID := entry.StatusPageID.ValueString()
			if linked != nil && !linked[pageID] {
				diags.AddAttributeError(
					entryPath.AtName("status_page_id"),
					"Unlinked Status Page in Component Configs",
					fmt.Sprintf("status_page_visibility.component_configs lists status page %q, which must also be listed in status_page_visibility.status_page_ids.", pageID),
				)
			}
			if seenPages[pageID] {
				diags.AddAttributeError(
					entryPath.AtName("status_page_id"),
					"Duplicate Status Page in Component Configs",
					fmt.Sprintf("status_page_visibility.component_configs lists status page %q more than once; combine its components into one entry.", pageID),
				)
			}
			seenPages[pageID] = true
		}

		if !entry.AffectAllComponents.IsUnknown() && !entry.Components.IsUnknown() {
			affectAll := entry.AffectAllComponents.ValueBool()
			hasComponents := len(entry.Components.Elements()) > 0
			switch {
			case affectAll && hasComponents:
				diags.AddAttributeError(
					entryPath.AtName("components"),
					"Conflicting Component Configuration",
					"status_page_visibility.component_configs.components can't be set when status_page_visibility.component_configs.affect_all_components is true.",
				)
			case !affectAll && !hasComponents:
				diags.AddAttributeError(
					entryPath,
					"Missing Components",
					"Each status_page_visibility.component_configs entry needs affect_all_components = true or at least one element in components.",
				)
			}
		}

		seenComponents := map[string]bool{}
		for _, componentElement := range entry.Components.Elements() {
			componentObj, ok := componentElement.(types.Object)
			if !ok {
				continue
			}
			component := objectAs[MaintenanceWindowComponentImpactModel](ctx, componentObj, diags)
			if component == nil || component.ComponentID.IsNull() || component.ComponentID.IsUnknown() {
				continue
			}
			componentID := component.ComponentID.ValueString()
			if seenComponents[componentID] {
				diags.AddAttributeError(
					entryPath.AtName("components").AtSetValue(componentElement),
					"Duplicate Component in Component Configs",
					fmt.Sprintf("status_page_visibility.component_configs.components lists component %q more than once.", componentID),
				)
			}
			seenComponents[componentID] = true
		}
	}
}
