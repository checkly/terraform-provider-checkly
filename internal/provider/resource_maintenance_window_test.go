package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

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

func TestMaintenanceWindowTimezoneValidator(t *testing.T) {
	t.Parallel()
	validate := func(v string) diag.Diagnostics {
		resp := &validator.StringResponse{}
		maintenanceWindowTimezoneValidator{}.ValidateString(context.Background(), validator.StringRequest{
			Path:        path.Root("timezone"),
			ConfigValue: types.StringValue(v),
		}, resp)
		return resp.Diagnostics
	}
	for _, v := range []string{"America/New_York", "UTC", "Etc/GMT+0", "Etc/GMT-0", ""} {
		if diags := validate(v); diags.HasError() {
			t.Errorf("expected %q to be accepted, got %v", v, diags)
		}
	}
	for _, v := range []string{"+05:00", "-0300", "Etc/GMT+5", "etc/gmt-14"} {
		if diags := validate(v); !diags.HasError() {
			t.Errorf("expected %q to be rejected", v)
		}
	}
}

// nullMaintenanceWindowModel returns a model for a configuration with only
// the required attributes, with schema defaults applied, as in a plan.
func nullMaintenanceWindowModel() MaintenanceWindowResourceModel {
	return MaintenanceWindowResourceModel{
		ID:                   types.StringNull(),
		Name:                 types.StringValue("Maintenance Window"),
		StartsAt:             types.StringValue("2099-01-01T09:00:00.000Z"),
		EndsAt:               types.StringValue("2099-01-01T10:00:00.000Z"),
		RepeatUnit:           types.StringNull(),
		RepeatInterval:       types.Int64Null(),
		RepeatEndsAt:         types.StringNull(),
		Tags:                 types.SetNull(types.StringType),
		Timezone:             types.StringNull(),
		PauseAllChecks:       types.BoolValue(false),
		SilenceAlertsTags:    types.SetNull(types.StringType),
		SilenceAllAlerts:     types.BoolValue(false),
		Description:          types.StringNull(),
		StatusPageVisibility: types.ListValueMust(types.ObjectType{AttrTypes: maintenanceWindowStatusPageVisibilityAttrTypes}, []attr.Value{}),
	}
}

// defaultStatusPageVisibilityBlock returns a block with only enabled = false
// configured, with schema defaults applied.
func defaultStatusPageVisibilityBlock(t *testing.T) types.List {
	t.Helper()
	var diags diag.Diagnostics
	list, d := types.ListValueFrom(context.Background(), types.ObjectType{AttrTypes: maintenanceWindowStatusPageVisibilityAttrTypes}, []MaintenanceWindowStatusPageVisibilityModel{{
		Enabled:               types.BoolValue(false),
		Severity:              types.StringNull(),
		AffectAllServices:     types.BoolValue(false),
		NotifyOnStart:         types.BoolValue(false),
		NotifyOnEnd:           types.BoolValue(false),
		SuppressAutoIncidents: types.BoolValue(false),
		ReminderMinutesBefore: types.SetNull(types.Int64Type),
		AutoStart:             types.BoolValue(true),
		AutoEnd:               types.BoolValue(true),
		ShowAffectedServices:  types.BoolValue(true),
		StatusPageIDs:         types.SetNull(types.StringType),
		ServiceIDs:            types.SetNull(types.StringType),
	}})
	diags.Append(d...)
	if diags.HasError() {
		t.Fatal(diags)
	}
	return list
}

// The read path keeps a block that is in the prior state, even with default
// settings, and writes no block when neither the prior state nor the API has
// one, so both configurations plan without a diff.
func TestMaintenanceWindowToModelStatusPageVisibility(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
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
		name  string
		prior types.List
		want  int
	}{
		{name: "no block", prior: nullMaintenanceWindowModel().StatusPageVisibility, want: 0},
		{name: "null block after import", prior: types.ListNull(types.ObjectType{AttrTypes: maintenanceWindowStatusPageVisibilityAttrTypes}), want: 0},
		{name: "explicit all-defaults block", prior: defaultStatusPageVisibilityBlock(t), want: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var diags diag.Diagnostics
			model := nullMaintenanceWindowModel()
			model.StatusPageVisibility = tc.prior
			maintenanceWindowToModel(ctx, window, &model, &diags)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if got := len(model.StatusPageVisibility.Elements()); got != tc.want {
				t.Errorf("status_page_visibility.# = %d, want %d", got, tc.want)
			}
			if tc.want == 1 {
				// Empty sets from the API stay null like the prior block's.
				if !reflect.DeepEqual(model.StatusPageVisibility, tc.prior) {
					t.Errorf("status_page_visibility = %v, want %v", model.StatusPageVisibility, tc.prior)
				}
			}
		})
	}

	// Non-default settings from the API are written even without a prior
	// block, so drift made outside Terraform shows up as a diff.
	enabled := *window
	enabled.StatusPageVisibility.Enabled = true
	enabled.StatusPageVisibility.ReminderMinutesBefore = []int{60, 120}
	enabled.StatusPageVisibility.StatusPageIDs = []string{"page"}
	var diags diag.Diagnostics
	model := nullMaintenanceWindowModel()
	maintenanceWindowToModel(ctx, &enabled, &model, &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}
	block, ok := maintenanceWindowStatusPageVisibilityModelFromList(ctx, model.StatusPageVisibility, &diags)
	if !ok {
		t.Fatal("status_page_visibility.# = 0, want 1")
	}
	if got := len(block.ReminderMinutesBefore.Elements()); got != 2 {
		t.Errorf("reminder_minutes_before.# = %d, want 2", got)
	}
	if got := len(block.StatusPageIDs.Elements()); got != 1 {
		t.Errorf("status_page_ids.# = %d, want 1", got)
	}
	if !block.ServiceIDs.IsNull() {
		t.Errorf("service_ids = %v, want null", block.ServiceIDs)
	}
}

// A configuration without the optional attributes sends every one of them
// with the API's default, so removing an attribute resets it.
func TestMaintenanceWindowFromModelResetsOmittedAttributes(t *testing.T) {
	t.Parallel()
	var diags diag.Diagnostics
	model := nullMaintenanceWindowModel()
	window := maintenanceWindowFromModel(context.Background(), &model, &diags)
	if diags.HasError() {
		t.Fatal(diags)
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

func TestMaintenanceWindowFromModelStatusPageVisibility(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	var diags diag.Diagnostics
	block, d := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: maintenanceWindowStatusPageVisibilityAttrTypes}, []MaintenanceWindowStatusPageVisibilityModel{{
		Enabled:               types.BoolValue(true),
		Severity:              types.StringValue("MAJOR"),
		AffectAllServices:     types.BoolValue(false),
		NotifyOnStart:         types.BoolValue(false),
		NotifyOnEnd:           types.BoolValue(true),
		SuppressAutoIncidents: types.BoolValue(true),
		ReminderMinutesBefore: types.SetValueMust(types.Int64Type, []attr.Value{types.Int64Value(60)}),
		AutoStart:             types.BoolValue(true),
		AutoEnd:               types.BoolValue(false),
		ShowAffectedServices:  types.BoolValue(false),
		StatusPageIDs:         types.SetValueMust(types.StringType, []attr.Value{types.StringValue("page")}),
		ServiceIDs:            types.SetValueMust(types.StringType, []attr.Value{types.StringValue("service")}),
	}})
	diags.Append(d...)
	model := nullMaintenanceWindowModel()
	model.StatusPageVisibility = block
	window := maintenanceWindowFromModel(ctx, &model, &diags)
	if diags.HasError() {
		t.Fatal(diags)
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

// The API reports a window without a timezone as null or as "UTC". Without a
// prior value, every UTC-family name, "Etc/UTC" included, is stored as null,
// so none of them shows a diff against a configuration without a timezone. A
// prior value equivalent to the API's is kept, so a configured alias or a
// different casing doesn't show a diff either.
func TestMaintenanceWindowTimezone(t *testing.T) {
	t.Parallel()
	cases := []struct {
		api   string
		prior types.String
		want  types.String
	}{
		{"", types.StringNull(), types.StringNull()},
		{"UTC", types.StringNull(), types.StringNull()},
		{"Etc/UTC", types.StringNull(), types.StringNull()},
		{"Europe/Berlin", types.StringNull(), types.StringValue("Europe/Berlin")},
		{"UTC", types.StringValue("Etc/UTC"), types.StringValue("Etc/UTC")},
		{"", types.StringValue("UTC"), types.StringValue("UTC")},
		{"", types.StringValue(""), types.StringValue("")},
		{"America/New_York", types.StringValue("america/new_york"), types.StringValue("america/new_york")},
		{"Europe/Paris", types.StringValue("Europe/Berlin"), types.StringValue("Europe/Paris")},
		{"UTC", types.StringValue("Europe/Berlin"), types.StringNull()},
	}
	for _, tc := range cases {
		if got := maintenanceWindowTimezone(tc.api, tc.prior); !got.Equal(tc.want) {
			t.Errorf("timezone from %q with prior %v = %v, want %v", tc.api, tc.prior, got, tc.want)
		}
	}
}

// The SDK sends an empty description as null, so a configured empty string
// must survive a read that reports no description.
func TestMaintenanceWindowDescription(t *testing.T) {
	t.Parallel()
	cases := []struct {
		api   string
		prior types.String
		want  types.String
	}{
		{"", types.StringNull(), types.StringNull()},
		{"", types.StringValue(""), types.StringValue("")},
		{"", types.StringValue("Planned upgrade"), types.StringNull()},
		{"Planned upgrade", types.StringNull(), types.StringValue("Planned upgrade")},
	}
	for _, tc := range cases {
		if got := maintenanceWindowDescription(tc.api, tc.prior); !got.Equal(tc.want) {
			t.Errorf("description from %q with prior %v = %v, want %v", tc.api, tc.prior, got, tc.want)
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
			// the stored value. Unset optional attributes are null in the
			// framework's state, so they are checked as absent.
			Config: withNothing,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(r, "repeat_unit"),
				resource.TestCheckNoResourceAttr(r, "repeat_interval"),
				resource.TestCheckNoResourceAttr(r, "repeat_ends_at"),
				resource.TestCheckNoResourceAttr(r, "tags"),
				resource.TestCheckNoResourceAttr(r, "timezone"),
				resource.TestCheckResourceAttr(r, "pause_all_checks", "false"),
				resource.TestCheckNoResourceAttr(r, "silence_alerts_tags"),
				resource.TestCheckResourceAttr(r, "silence_all_alerts", "false"),
				resource.TestCheckNoResourceAttr(r, "description"),
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
