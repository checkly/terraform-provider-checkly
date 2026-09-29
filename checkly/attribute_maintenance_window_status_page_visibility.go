package checkly

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	checkly "github.com/checkly/checkly-go-sdk"
)

// maintenanceWindowStatusPageVisibilitySchema is deliberately not Computed:
// removing the block must reset the window's status page settings rather than
// keep the ones stored by the API.
func maintenanceWindowStatusPageVisibilitySchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 1,
		Description: "Whether and how the maintenance window appears on status pages. " +
			"Omitting the block hides the window and unlinks it from all status pages and services. " +
			"Only service-based status page links are supported; links to status page components can't be managed here. " +
			"An account without the status page maintenance windows entitlement can only change `enabled` to `false`: " +
			"keep the rest of the block as it is, because removing the block also changes the other settings, which is rejected.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"enabled": {
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
					Description: "Show the maintenance window on the linked status pages. All other settings in this block only take effect when this is `true`. (Default `false`).",
				},
				"severity": {
					Type:         schema.TypeString,
					Optional:     true,
					ValidateFunc: validateOneOf([]string{"MINOR", "MEDIUM", "MAJOR", "CRITICAL"}),
					Description:  "The severity shown on the status page. Possible values are `MINOR`, `MEDIUM`, `MAJOR` and `CRITICAL`.",
				},
				"affect_all_services": {
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
					Description: "Mark every service on the linked status pages as affected. Can't be combined with `service_ids`. (Default `false`).",
				},
				"notify_on_start": {
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
					Description: "Email status page subscribers when the maintenance starts. (Default `false`).",
				},
				"notify_on_end": {
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
					Description: "Email status page subscribers when the maintenance ends. (Default `false`).",
				},
				"suppress_auto_incidents": {
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
					Description: "Suppress automatically created incidents for the linked services during the maintenance. (Default `false`).",
				},
				"reminder_minutes_before": {
					Type:     schema.TypeSet,
					Optional: true,
					MaxItems: 3,
					Elem: &schema.Schema{
						Type:         schema.TypeInt,
						ValidateFunc: validateBetween(60, 10080),
					},
					Description: "Up to three reminders, in minutes before the maintenance starts (60 to 10080), sent to status page subscribers.",
				},
				"auto_start": {
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     true,
					Description: "Start the maintenance automatically at the scheduled time. (Default `true`).",
				},
				"auto_end": {
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     true,
					Description: "Complete the maintenance automatically at the scheduled end time. (Default `true`).",
				},
				"show_affected_services": {
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     true,
					Description: "Show which services the maintenance affects. When `false`, downtime during the maintenance counts against the services' uptime. (Default `true`).",
				},
				"status_page_ids": {
					Type:     schema.TypeSet,
					Optional: true,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Description: "The IDs of the status pages (`checkly_status_page`) to show the maintenance window on. Requires `service_ids` or `affect_all_services`.",
				},
				"service_ids": {
					Type:     schema.TypeSet,
					Optional: true,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Description: "The IDs of the affected status page services (`checkly_status_page_service`). Each service must be on one of the linked status pages.",
				},
			},
		},
	}
}

// maintenanceWindowStatusPageVisibilityFromList returns the zero value for a
// missing block, which the SDK sends as the API's defaults.
func maintenanceWindowStatusPageVisibilityFromList(l []any) checkly.MaintenanceWindowStatusPageVisibility {
	if len(l) == 0 || l[0] == nil {
		return checkly.MaintenanceWindowStatusPageVisibility{}
	}
	m := l[0].(tfMap)
	autoStart := m["auto_start"].(bool)
	autoEnd := m["auto_end"].(bool)
	showAffectedServices := m["show_affected_services"].(bool)
	return checkly.MaintenanceWindowStatusPageVisibility{
		Enabled:               m["enabled"].(bool),
		Severity:              m["severity"].(string),
		AffectAllServices:     m["affect_all_services"].(bool),
		NotifyOnStart:         m["notify_on_start"].(bool),
		NotifyOnEnd:           m["notify_on_end"].(bool),
		SuppressAutoIncidents: m["suppress_auto_incidents"].(bool),
		ReminderMinutesBefore: intsFromSet(m["reminder_minutes_before"].(*schema.Set)),
		AutoStart:             &autoStart,
		AutoEnd:               &autoEnd,
		ShowAffectedServices:  &showAffectedServices,
		StatusPageIDs:         stringsFromSet(m["status_page_ids"].(*schema.Set)),
		ServiceIDs:            stringsFromSet(m["service_ids"].(*schema.Set)),
	}
}

// maintenanceWindowStatusPageVisibilityToList returns no block for default
// settings when the prior state has no block either, so a configuration
// without the block doesn't show a diff. A block that is in the prior state
// is always written back, which keeps an explicit all-defaults block stable.
func maintenanceWindowStatusPageVisibilityToList(v checkly.MaintenanceWindowStatusPageVisibility, priorEmpty bool) []any {
	if priorEmpty && isDefaultMaintenanceWindowStatusPageVisibility(v) {
		return []any{}
	}
	return []any{
		tfMap{
			"enabled":                 v.Enabled,
			"severity":                v.Severity,
			"affect_all_services":     v.AffectAllServices,
			"notify_on_start":         v.NotifyOnStart,
			"notify_on_end":           v.NotifyOnEnd,
			"suppress_auto_incidents": v.SuppressAutoIncidents,
			"reminder_minutes_before": v.ReminderMinutesBefore,
			"auto_start":              v.AutoStart == nil || *v.AutoStart,
			"auto_end":                v.AutoEnd == nil || *v.AutoEnd,
			"show_affected_services":  v.ShowAffectedServices == nil || *v.ShowAffectedServices,
			"status_page_ids":         v.StatusPageIDs,
			"service_ids":             v.ServiceIDs,
		},
	}
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

func intsFromSet(s *schema.Set) []int {
	r := make([]int, s.Len())
	for i, item := range s.List() {
		r[i] = item.(int)
	}
	return r
}
