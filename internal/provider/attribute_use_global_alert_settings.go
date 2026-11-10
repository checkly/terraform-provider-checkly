package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const useGlobalAlertSettingsAttributeName = "use_global_alert_settings"

// makeUseGlobalAlertSettingsAttributeSchema returns the schema of the
// use_global_alert_settings attribute of a resource that also has
// alert_settings. subject names the resource in the description, e.g.
// "check", "monitor" or "check group".
//
// The attribute has no static default: when it is not configured, its value
// follows from whether alert_settings is configured, the same way the Checkly
// CLI derives it.
func makeUseGlobalAlertSettingsAttributeSchema(subject string) schema.BoolAttribute {
	return schema.BoolAttribute{
		Optional: true,
		Computed: true,
		Description: fmt.Sprintf("When true, the account level alert settings are used instead of the "+
			"`alert_settings` of this %s. Defaults to `true` when `alert_settings` is not set, and to `false` "+
			"when it is.", subject),
		PlanModifiers: []planmodifier.Bool{
			useGlobalAlertSettingsPlanModifier{},
		},
		Validators: []validator.Bool{
			useGlobalAlertSettingsValidator{},
		},
	}
}

var _ planmodifier.Bool = useGlobalAlertSettingsPlanModifier{}

// useGlobalAlertSettingsPlanModifier plans an unconfigured
// use_global_alert_settings as true when alert_settings is not configured
// either, and as false when it is, so that configured alert settings take
// effect. It applies to every plan, not only to creation, so removing
// alert_settings later plans true again.
type useGlobalAlertSettingsPlanModifier struct{}

func (m useGlobalAlertSettingsPlanModifier) Description(ctx context.Context) string {
	return "Plans true when alert_settings is not configured, and false when it is."
}

func (m useGlobalAlertSettingsPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m useGlobalAlertSettingsPlanModifier) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if req.Plan.Raw.IsNull() || !req.ConfigValue.IsNull() {
		return
	}

	var alertSettings types.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(alertSettingsAttributeName), &alertSettings)...)
	if resp.Diagnostics.HasError() {
		return
	}

	switch {
	case alertSettings.IsUnknown():
		// Whether alert_settings is set is known only at apply time.
		resp.PlanValue = types.BoolUnknown()
	case alertSettings.IsNull():
		resp.PlanValue = types.BoolValue(true)
	default:
		resp.PlanValue = types.BoolValue(false)
	}
}

var _ validator.Bool = useGlobalAlertSettingsValidator{}

// useGlobalAlertSettingsValidator warns when use_global_alert_settings is
// configured as true together with alert_settings. The account level alert
// settings then take precedence, and alert_settings has no effect.
type useGlobalAlertSettingsValidator struct{}

func (v useGlobalAlertSettingsValidator) Description(ctx context.Context) string {
	return "alert_settings has no effect when use_global_alert_settings is true"
}

func (v useGlobalAlertSettingsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v useGlobalAlertSettingsValidator) ValidateBool(ctx context.Context, req validator.BoolRequest, resp *validator.BoolResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || !req.ConfigValue.ValueBool() {
		return
	}

	var alertSettings types.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(alertSettingsAttributeName), &alertSettings)...)
	if resp.Diagnostics.HasError() || alertSettings.IsNull() || alertSettings.IsUnknown() {
		return
	}

	resp.Diagnostics.AddAttributeWarning(
		req.Path,
		"Alert Settings Have No Effect",
		"use_global_alert_settings is true, so the account level alert settings are used and the "+
			"configured alert_settings have no effect. To keep the account level alert settings, remove "+
			"alert_settings. To apply alert_settings, remove use_global_alert_settings or set it to false.",
	)
}
