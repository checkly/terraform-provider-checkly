package checkly

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	checkly "github.com/checkly/checkly-go-sdk"
)

func TestTimezonesEquivalent(t *testing.T) {
	t.Parallel()
	cases := []struct {
		a, b string
		want bool
	}{
		{"", "", true},
		{"", "UTC", true},
		{"UTC", "Etc/UTC", true},
		{"GMT", "Etc/Zulu", true},
		{"Etc/GMT+0", "UTC", true},
		{"GMT-0", "UTC", true},
		{"America/New_York", "america/new_york", true},
		{"Europe/Berlin", "Europe/Berlin", true},
		{"", "Europe/Berlin", false},
		{"UTC", "Europe/London", false},
		{"Europe/Berlin", "Europe/Paris", false},
		{"US/Eastern", "America/New_York", false},
	}
	for _, tc := range cases {
		if got := timezonesEquivalent(tc.a, tc.b); got != tc.want {
			t.Errorf("timezonesEquivalent(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestValidateMaintenanceWindowTimezone(t *testing.T) {
	t.Parallel()
	for _, v := range []string{"America/New_York", "UTC", "Etc/GMT+0", "Etc/GMT-0", ""} {
		if _, errs := validateMaintenanceWindowTimezone(v, "timezone"); len(errs) != 0 {
			t.Errorf("expected %q to be accepted, got %v", v, errs)
		}
	}
	for _, v := range []string{"+05:00", "-0300", "Etc/GMT+5", "etc/gmt-14"} {
		if _, errs := validateMaintenanceWindowTimezone(v, "timezone"); len(errs) == 0 {
			t.Errorf("expected %q to be rejected", v)
		}
	}
}

// The read path keeps a block that is in the prior state, even with default
// settings, and writes no block when neither the prior state nor the API has
// one, so both configurations plan without a diff.
func TestResourceDataFromMaintenanceWindowsStatusPageVisibility(t *testing.T) {
	t.Parallel()
	trueValue := true
	window := &checkly.MaintenanceWindow{
		Name: "Maintenance Window",
		StatusPageVisibility: checkly.MaintenanceWindowStatusPageVisibility{
			ReminderMinutesBefore: []int{},
			AutoStart:             &trueValue,
			AutoEnd:               &trueValue,
			ShowAffectedServices:  &trueValue,
			StatusPageIDs:         []string{},
			ServiceIDs:            []string{},
		},
	}

	cases := []struct {
		name   string
		config map[string]any
		want   string
	}{
		{name: "no block", config: map[string]any{}, want: "0"},
		{
			name: "explicit all-defaults block",
			config: map[string]any{
				"status_page_visibility": []any{map[string]any{"enabled": false}},
			},
			want: "1",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := schema.TestResourceDataRaw(t, resourceMaintenanceWindow().Schema, tc.config)
			if err := resourceDataFromMaintenanceWindows(window, d); err != nil {
				t.Fatal(err)
			}
			if got := d.Get("status_page_visibility.#").(int); fmt.Sprint(got) != tc.want {
				t.Errorf("status_page_visibility.# = %d, want %s", got, tc.want)
			}
		})
	}

	// Non-default settings from the API are written even without a prior
	// block, so drift made outside Terraform shows up as a diff.
	enabled := *window
	enabled.StatusPageVisibility.Enabled = true
	enabled.StatusPageVisibility.ReminderMinutesBefore = []int{60, 120}
	enabled.StatusPageVisibility.StatusPageIDs = []string{"page"}
	d := schema.TestResourceDataRaw(t, resourceMaintenanceWindow().Schema, map[string]any{})
	if err := resourceDataFromMaintenanceWindows(&enabled, d); err != nil {
		t.Fatal(err)
	}
	if got := d.Get("status_page_visibility.0.reminder_minutes_before.#").(int); got != 2 {
		t.Errorf("reminder_minutes_before.# = %d, want 2", got)
	}
	if got := d.Get("status_page_visibility.0.status_page_ids.#").(int); got != 1 {
		t.Errorf("status_page_ids.# = %d, want 1", got)
	}
}

// A configuration without the optional attributes sends every one of them
// with the API's default, so removing an attribute resets it.
func TestMaintenanceWindowsFromResourceDataResetsOmittedAttributes(t *testing.T) {
	t.Parallel()
	d := schema.TestResourceDataRaw(t, resourceMaintenanceWindow().Schema, map[string]any{
		"name":      "Maintenance Window",
		"starts_at": "2099-01-01T09:00:00.000Z",
		"ends_at":   "2099-01-01T10:00:00.000Z",
	})
	window, err := maintenanceWindowsFromResourceData(d)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(window)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"repeatInterval":    nil,
		"repeatUnit":        nil,
		"repeatEndsAt":      nil,
		"tags":              []any{},
		"timezone":          nil,
		"pauseAllChecks":    false,
		"silenceAlertsTags": []any{},
		"silenceAllAlerts":  false,
		"description":       nil,
		"statusPageVisibility": map[string]any{
			"enabled":               false,
			"severity":              nil,
			"affectAllServices":     false,
			"notifyOnStart":         false,
			"notifyOnEnd":           false,
			"suppressAutoIncidents": false,
			"reminderMinutesBefore": []any{},
			"autoStart":             true,
			"autoEnd":               true,
			"showAffectedServices":  true,
			"statusPageIds":         []any{},
			"serviceIds":            []any{},
		},
	}
	// Compared whole, so a key the SDK leaves out (which would keep the
	// stored value) or a field it gains fails until its default is decided.
	for _, configured := range []string{"id", "name", "startsAt", "endsAt", "created_at", "updated_at"} {
		delete(got, configured)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("payload = %#v, want %#v", got, want)
	}
}

func TestMaintenanceWindowsFromResourceDataStatusPageVisibility(t *testing.T) {
	t.Parallel()
	d := schema.TestResourceDataRaw(t, resourceMaintenanceWindow().Schema, map[string]any{
		"name":      "Maintenance Window",
		"starts_at": "2099-01-01T09:00:00.000Z",
		"ends_at":   "2099-01-01T10:00:00.000Z",
		"status_page_visibility": []any{map[string]any{
			"enabled":                 true,
			"severity":                "MAJOR",
			"notify_on_end":           true,
			"suppress_auto_incidents": true,
			"reminder_minutes_before": []any{60},
			"auto_end":                false,
			"show_affected_services":  false,
			"status_page_ids":         []any{"page"},
			"service_ids":             []any{"service"},
		}},
	})
	window, err := maintenanceWindowsFromResourceData(d)
	if err != nil {
		t.Fatal(err)
	}
	trueValue, falseValue := true, false
	want := checkly.MaintenanceWindowStatusPageVisibility{
		Enabled:               true,
		Severity:              "MAJOR",
		NotifyOnEnd:           true,
		SuppressAutoIncidents: true,
		ReminderMinutesBefore: []int{60},
		AutoStart:             &trueValue,
		AutoEnd:               &falseValue,
		ShowAffectedServices:  &falseValue,
		StatusPageIDs:         []string{"page"},
		ServiceIDs:            []string{"service"},
	}
	if !reflect.DeepEqual(window.StatusPageVisibility, want) {
		t.Errorf("StatusPageVisibility = %+v, want %+v", window.StatusPageVisibility, want)
	}
}

// The API reports a window without a timezone as null or as "UTC". Every
// UTC-family name, "Etc/UTC" included, is stored as "", so none of them shows a
// diff against a configuration without a timezone.
func TestResourceDataFromMaintenanceWindowsTimezone(t *testing.T) {
	t.Parallel()
	for apiValue, want := range map[string]string{"": "", "UTC": "", "Etc/UTC": "", "Europe/Berlin": "Europe/Berlin"} {
		d := schema.TestResourceDataRaw(t, resourceMaintenanceWindow().Schema, map[string]any{})
		if err := resourceDataFromMaintenanceWindows(&checkly.MaintenanceWindow{Timezone: apiValue}, d); err != nil {
			t.Fatal(err)
		}
		if got := d.Get("timezone").(string); got != want {
			t.Errorf("timezone from %q = %q, want %q", apiValue, got, want)
		}
	}
}

func TestAccMaintenanceWindowControlsAllFields(t *testing.T) {
	// Requires an account with the status page maintenance windows
	// entitlement. The window lies far in the future so no maintenance is
	// active: the API rejects timezone changes while a maintenance is running.
	rInt := acctest.RandInt()
	statusPage := fmt.Sprintf(`
		resource "checkly_status_page_service" "test" {
			name = "Maintenance Window Service %d"
		}

		resource "checkly_status_page" "test" {
			name = "Maintenance Window Status Page %d"
			url  = "mw-status-page-%d"

			card {
				name = "Card"

				service_attachment {
					service_id = checkly_status_page_service.test.id
				}
			}
		}
	`, rInt, rInt, rInt)
	withEverything := statusPage + fmt.Sprintf(`
		resource "checkly_maintenance_windows" "test" {
			name                = "Maintenance Window %d"
			starts_at           = "2099-01-01T09:00:00.000Z"
			ends_at             = "2099-01-01T10:00:00.000Z"
			repeat_unit         = "WEEK"
			repeat_interval     = 1
			repeat_ends_at      = "2099-06-01T00:00:00.000Z"
			tags                = ["tf-acc"]
			timezone            = "Europe/Berlin"
			pause_all_checks    = true
			silence_alerts_tags = ["tf-acc-silenced"]
			silence_all_alerts  = true
			description         = "Planned upgrade"

			status_page_visibility {
				enabled                 = true
				severity                = "MAJOR"
				notify_on_start         = true
				reminder_minutes_before = [60]
				auto_end                = false
				status_page_ids         = [checkly_status_page.test.id]
				service_ids             = [checkly_status_page_service.test.id]
			}
		}
	`, rInt)
	withDefaultBlock := statusPage + fmt.Sprintf(`
		resource "checkly_maintenance_windows" "test" {
			name      = "Maintenance Window %d"
			starts_at = "2099-01-01T09:00:00.000Z"
			ends_at   = "2099-01-01T10:00:00.000Z"

			status_page_visibility {
				enabled = false
			}
		}
	`, rInt)
	withNothing := statusPage + fmt.Sprintf(`
		resource "checkly_maintenance_windows" "test" {
			name      = "Maintenance Window %d"
			starts_at = "2099-01-01T09:00:00.000Z"
			ends_at   = "2099-01-01T10:00:00.000Z"
		}
	`, rInt)
	const r = "checkly_maintenance_windows.test"
	accTestCase(t, []resource.TestStep{
		{
			Config: withEverything,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(r, "repeat_unit", "WEEK"),
				resource.TestCheckResourceAttr(r, "repeat_interval", "1"),
				resource.TestCheckResourceAttr(r, "tags.#", "1"),
				resource.TestCheckResourceAttr(r, "timezone", "Europe/Berlin"),
				resource.TestCheckResourceAttr(r, "pause_all_checks", "true"),
				resource.TestCheckTypeSetElemAttr(r, "silence_alerts_tags.*", "tf-acc-silenced"),
				resource.TestCheckResourceAttr(r, "silence_all_alerts", "true"),
				resource.TestCheckResourceAttr(r, "description", "Planned upgrade"),
				resource.TestCheckResourceAttr(r, "status_page_visibility.#", "1"),
				resource.TestCheckResourceAttr(r, "status_page_visibility.0.enabled", "true"),
				resource.TestCheckResourceAttr(r, "status_page_visibility.0.severity", "MAJOR"),
				resource.TestCheckResourceAttr(r, "status_page_visibility.0.auto_end", "false"),
				resource.TestCheckResourceAttr(r, "status_page_visibility.0.status_page_ids.#", "1"),
				resource.TestCheckResourceAttr(r, "status_page_visibility.0.service_ids.#", "1"),
			),
		},
		{
			// Removing every optional attribute must reset it rather than keep
			// the stored value.
			Config: withNothing,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(r, "repeat_unit", ""),
				resource.TestCheckResourceAttr(r, "repeat_interval", "0"),
				resource.TestCheckResourceAttr(r, "repeat_ends_at", ""),
				resource.TestCheckResourceAttr(r, "tags.#", "0"),
				resource.TestCheckResourceAttr(r, "timezone", ""),
				resource.TestCheckResourceAttr(r, "pause_all_checks", "false"),
				resource.TestCheckResourceAttr(r, "silence_alerts_tags.#", "0"),
				resource.TestCheckResourceAttr(r, "silence_all_alerts", "false"),
				resource.TestCheckResourceAttr(r, "description", ""),
				resource.TestCheckResourceAttr(r, "status_page_visibility.#", "0"),
			),
		},
		{
			// An explicit block with default settings is kept in state; the
			// harness's follow-up plan check proves it stays without a diff.
			Config: withDefaultBlock,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(r, "status_page_visibility.#", "1"),
				resource.TestCheckResourceAttr(r, "status_page_visibility.0.enabled", "false"),
			),
		},
	})
}
