package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/checkly/checkly-go-sdk/v2"
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
		StatusPageVisibility: types.ObjectNull(maintenanceWindowStatusPageVisibilityAttrTypes),
	}
}

// defaultStatusPageVisibility returns a status_page_visibility value with only
// enabled = false configured, with schema defaults applied.
func defaultStatusPageVisibility(t *testing.T) types.Object {
	t.Helper()
	var diags diag.Diagnostics
	obj, d := types.ObjectValueFrom(context.Background(), maintenanceWindowStatusPageVisibilityAttrTypes, MaintenanceWindowStatusPageVisibilityModel{
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
		ComponentConfigs:      types.SetNull(types.ObjectType{AttrTypes: maintenanceWindowComponentConfigAttrTypes}),
	})
	diags.Append(d...)
	if diags.HasError() {
		t.Fatal(diags)
	}
	return obj
}

// The read path keeps a status_page_visibility value that is in the prior
// state, even with default settings, and writes null when neither the prior
// state nor the API has one, so both configurations plan without a diff.
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
		prior types.Object
		want  bool
	}{
		{name: "not configured", prior: nullMaintenanceWindowModel().StatusPageVisibility, want: false},
		{name: "null after import", prior: types.ObjectNull(maintenanceWindowStatusPageVisibilityAttrTypes), want: false},
		{name: "unknown prior", prior: types.ObjectUnknown(maintenanceWindowStatusPageVisibilityAttrTypes), want: false},
		{name: "explicit all-defaults object", prior: defaultStatusPageVisibility(t), want: true},
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
			if got := !model.StatusPageVisibility.IsNull(); got != tc.want {
				t.Errorf("status_page_visibility set = %t, want %t", got, tc.want)
			}
			if tc.want {
				// Empty sets from the API stay null like the prior value's.
				if !reflect.DeepEqual(model.StatusPageVisibility, tc.prior) {
					t.Errorf("status_page_visibility = %v, want %v", model.StatusPageVisibility, tc.prior)
				}
			}
		})
	}

	// Non-default settings from the API are written even without a prior
	// value, so drift made outside Terraform shows up as a diff.
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
	block := objectAs[MaintenanceWindowStatusPageVisibilityModel](ctx, model.StatusPageVisibility, &diags)
	if block == nil {
		t.Fatal("status_page_visibility is null, want a value")
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
			"componentConfigs":      []any{},
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
	block, d := types.ObjectValueFrom(ctx, maintenanceWindowStatusPageVisibilityAttrTypes, MaintenanceWindowStatusPageVisibilityModel{
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
		ComponentConfigs:      types.SetNull(types.ObjectType{AttrTypes: maintenanceWindowComponentConfigAttrTypes}),
	})
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
		// Sent empty, so configs set outside Terraform are removed.
		ComponentConfigs: []checkly.MaintenanceWindowComponentConfig{},
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
		if got := stringValueOrPriorEmpty(tc.api, tc.prior); !got.Equal(tc.want) {
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

			cards = [
				{
					name = "Card"

					service_attachments = [
						{
							service_id = checkly_status_page_service.test.id
						},
					]
				},
			]
		}
	`, rInt, rInt, rInt)
	withEverything := statusPage + fmt.Sprintf(`
		resource "checkly_maintenance_window" "test" {
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

			status_page_visibility = {
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
	withDefaults := statusPage + fmt.Sprintf(`
		resource "checkly_maintenance_window" "test" {
			name      = "Maintenance Window %d"
			starts_at = "2099-01-01T09:00:00.000Z"
			ends_at   = "2099-01-01T10:00:00.000Z"

			status_page_visibility = {
				enabled = false
			}
		}
	`, rInt)
	withEmptyObject := statusPage + fmt.Sprintf(`
		resource "checkly_maintenance_window" "test" {
			name      = "Maintenance Window %d"
			starts_at = "2099-01-01T09:00:00.000Z"
			ends_at   = "2099-01-01T10:00:00.000Z"

			status_page_visibility = {}
		}
	`, rInt)
	withNothing := statusPage + fmt.Sprintf(`
		resource "checkly_maintenance_window" "test" {
			name      = "Maintenance Window %d"
			starts_at = "2099-01-01T09:00:00.000Z"
			ends_at   = "2099-01-01T10:00:00.000Z"
		}
	`, rInt)
	const r = "checkly_maintenance_window.test"
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
				resource.TestCheckResourceAttr(r, "status_page_visibility.enabled", "true"),
				resource.TestCheckResourceAttr(r, "status_page_visibility.severity", "MAJOR"),
				resource.TestCheckResourceAttr(r, "status_page_visibility.auto_end", "false"),
				resource.TestCheckResourceAttr(r, "status_page_visibility.status_page_ids.#", "1"),
				resource.TestCheckResourceAttr(r, "status_page_visibility.service_ids.#", "1"),
			),
		},
		{
			// Imported while every attribute holds a non-default value. An
			// explicit status_page_visibility object with default settings
			// reads back the same as an absent one, so a later step would not
			// round-trip.
			ResourceName:      r,
			ImportState:       true,
			ImportStateVerify: true,
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
				resource.TestCheckNoResourceAttr(r, "status_page_visibility"),
			),
		},
		{
			// An explicit object with default settings is kept in state; the
			// harness's follow-up plan check proves it stays without a diff.
			Config: withDefaults,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(r, "status_page_visibility.enabled", "false"),
				resource.TestCheckResourceAttr(r, "status_page_visibility.auto_start", "true"),
			),
		},
		{
			// An empty object takes every schema default, so it equals the
			// previous step's object and plans no change.
			Config: withEmptyObject,
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectEmptyPlan(),
				},
			},
			Check: resource.TestCheckResourceAttr(r, "status_page_visibility.enabled", "false"),
		},
	})
}

// storedComponentStatuses returns the component configs the API stores for a
// maintenance window, as the statuses of the affected components by status
// page ID, with "*" standing for affect_all_components.
func storedComponentStatuses(windowID string) (map[string]map[string]string, *checkly.MaintenanceWindow, error) {
	id, err := decodeNumericID(windowID)
	if err != nil {
		return nil, nil, err
	}
	mw, err := testAccClient().GetMaintenanceWindow(context.Background(), id)
	if err != nil {
		return nil, nil, err
	}
	stored := map[string]map[string]string{}
	for _, config := range mw.StatusPageVisibility.ComponentConfigs {
		statuses := map[string]string{}
		if config.AffectAllComponents {
			statuses["*"] = ""
		}
		for _, component := range config.Components {
			statuses[component.ComponentID] = string(component.Status)
		}
		stored[config.StatusPageID] = statuses
	}
	return stored, mw, nil
}

func TestAccMaintenanceWindowV3ComponentLinks(t *testing.T) {
	// Requires an account with the status page maintenance windows
	// entitlement.
	rInt := acctest.RandInt()
	page := fmt.Sprintf(`
		resource "checkly_status_page_v3" "test" {
			name = "Maintenance Window Status Page %d"
			url  = "mw-status-page-v3-%d"
		}

		resource "checkly_status_page_v3_component" "a" {
			status_page_id = checkly_status_page_v3.test.id
			name           = "API"
			display_order  = 0
		}

		resource "checkly_status_page_v3_component" "b" {
			status_page_id = checkly_status_page_v3.test.id
			name           = "Database"
			display_order  = 1
		}
	`, rInt, rInt)
	window := func(statusPageVisibility string) string {
		return page + fmt.Sprintf(`
			resource "checkly_maintenance_window" "test" {
				name      = "Maintenance Window %d"
				starts_at = "2099-01-01T09:00:00.000Z"
				ends_at   = "2099-01-01T10:00:00.000Z"
				%s
			}
		`, rInt, statusPageVisibility)
	}
	withComponents := window(`
		status_page_visibility = {
			enabled         = true
			status_page_ids = [checkly_status_page_v3.test.id]

			component_configs = [{
				status_page_id = checkly_status_page_v3.test.id
				components = [
					{
						component_id = checkly_status_page_v3_component.a.id
						status       = "MAJOR_OUTAGE"
					},
					{
						component_id = checkly_status_page_v3_component.b.id
					},
				]
			}]
		}
	`)
	withAllComponents := window(`
		status_page_visibility = {
			enabled         = true
			status_page_ids = [checkly_status_page_v3.test.id]

			component_configs = [{
				status_page_id        = checkly_status_page_v3.test.id
				affect_all_components = true
			}]
		}
	`)
	withoutLinks := window("")
	// A renamed window that leaves status_page_visibility to the web app.
	ignoringLinks := page + fmt.Sprintf(`
		resource "checkly_maintenance_window" "test" {
			name      = "Maintenance Window %d renamed"
			starts_at = "2099-01-01T09:00:00.000Z"
			ends_at   = "2099-01-01T10:00:00.000Z"

			lifecycle {
				ignore_changes = [status_page_visibility]
			}
		}
	`, rInt)
	// The page is linked through component_configs but missing from
	// status_page_ids, which validation rejects at plan time.
	unlinkedComponentConfig := window(`
		status_page_visibility = {
			enabled = true

			component_configs = [{
				status_page_id        = checkly_status_page_v3.test.id
				affect_all_components = true
			}]
		}
	`)

	const r = "checkly_maintenance_window.test"
	var windowID, pageID, componentA, componentB string
	captureIDs := func(s *terraform.State) error {
		resources := s.RootModule().Resources
		windowID = resources[r].Primary.ID
		pageID = resources["checkly_status_page_v3.test"].Primary.ID
		componentA = resources["checkly_status_page_v3_component.a"].Primary.ID
		componentB = resources["checkly_status_page_v3_component.b"].Primary.ID
		return nil
	}
	// stored checks the component configs the API holds for the window;
	// want builds the expected value from the captured IDs.
	stored := func(want func() map[string]map[string]string) resource.TestCheckFunc {
		return func(*terraform.State) error {
			got, _, err := storedComponentStatuses(windowID)
			if err != nil {
				return err
			}
			if !reflect.DeepEqual(got, want()) {
				return fmt.Errorf("stored component configs = %v, want %v", got, want())
			}
			return nil
		}
	}
	unlinked := func(*terraform.State) error {
		got, mw, err := storedComponentStatuses(windowID)
		if err != nil {
			return err
		}
		if len(got) != 0 || len(mw.StatusPageVisibility.StatusPageIDs) != 0 {
			return fmt.Errorf("window still linked: status pages %v, component configs %v", mw.StatusPageVisibility.StatusPageIDs, got)
		}
		return nil
	}

	// linkOutsideTerraform links component A to the window through the API.
	linkOutsideTerraform := func() {
		_, mw, err := storedComponentStatuses(windowID)
		if err != nil {
			t.Fatal(err)
		}
		mw.StatusPageVisibility.Enabled = true
		mw.StatusPageVisibility.StatusPageIDs = []string{pageID}
		mw.StatusPageVisibility.ComponentConfigs = []checkly.MaintenanceWindowComponentConfig{{
			StatusPageID: pageID,
			Components:   []checkly.MaintenanceWindowComponentImpact{{ComponentID: componentA}},
		}}
		if _, err := testAccClient().UpdateMaintenanceWindow(context.Background(), mw.ID, *mw); err != nil {
			t.Fatal(err)
		}
	}

	accTestCase(t, []resource.TestStep{
		{
			// The component IDs are unknown at plan time, so the defaults
			// are only filled in at apply and are checked here rather than
			// in the plan.
			Config: withComponents,
			Check: resource.ComposeTestCheckFunc(
				captureIDs,
				resource.TestCheckResourceAttr(r, "status_page_visibility.component_configs.#", "1"),
				resource.TestCheckTypeSetElemNestedAttrs(r, "status_page_visibility.component_configs.*", map[string]string{
					"affect_all_components": "false",
					"components.#":          "2",
				}),
				stored(func() map[string]map[string]string {
					return map[string]map[string]string{pageID: {
						componentA: "MAJOR_OUTAGE",
						componentB: "UNDER_MAINTENANCE",
					}}
				}),
			),
		},
		{
			// Importing explicit components, one with a defaulted status,
			// reads back what the configuration plans.
			ResourceName:      r,
			ImportState:       true,
			ImportStateVerify: true,
		},
		{
			Config:      unlinkedComponentConfig,
			PlanOnly:    true,
			ExpectError: regexp.MustCompile("Unlinked Status Page in Component Configs"),
		},
		{
			Config: withAllComponents,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckTypeSetElemNestedAttrs(r, "status_page_visibility.component_configs.*", map[string]string{
					"affect_all_components": "true",
				}),
				stored(func() map[string]map[string]string {
					return map[string]map[string]string{pageID: {"*": ""}}
				}),
			),
		},
		{
			ResourceName:      r,
			ImportState:       true,
			ImportStateVerify: true,
		},
		{
			// Leaving the links out of the configuration removes them.
			Config: withoutLinks,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(r, "status_page_visibility"),
				unlinked,
			),
		},
		{
			// With status_page_visibility ignored, an update of another attribute
			// sends the links made outside Terraform back unchanged.
			PreConfig: linkOutsideTerraform,
			Config:    ignoringLinks,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(r, "name", fmt.Sprintf("Maintenance Window %d renamed", rInt)),
				stored(func() map[string]map[string]string {
					return map[string]map[string]string{pageID: {componentA: "UNDER_MAINTENANCE"}}
				}),
			),
		},
		{
			// A component link made outside Terraform shows as a change and
			// is removed by the apply.
			PreConfig: linkOutsideTerraform,
			Config:    withoutLinks,
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(r, plancheck.ResourceActionUpdate),
				},
			},
			Check: unlinked,
		},
	})
}

// Ignoring only status_page_visibility.component_configs keeps a window's
// component links while the rest of status_page_visibility follows the
// configuration.
func TestAccMaintenanceWindowIgnoreComponentConfigs(t *testing.T) {
	// Requires an account with the status page maintenance windows
	// entitlement.
	rInt := acctest.RandInt()
	window := func(statusPageVisibility string) string {
		return fmt.Sprintf(`
			resource "checkly_status_page_v3" "test" {
				name = "Maintenance Window Status Page %d"
				url  = "mw-ignore-status-page-v3-%d"
			}

			resource "checkly_status_page_v3_component" "a" {
				status_page_id = checkly_status_page_v3.test.id
				name           = "API"
				display_order  = 0
			}

			resource "checkly_maintenance_window" "test" {
				name      = "Maintenance Window %d"
				starts_at = "2099-01-01T09:00:00.000Z"
				ends_at   = "2099-01-01T10:00:00.000Z"
				%s
			}
		`, rInt, rInt, rInt, statusPageVisibility)
	}

	const r = "checkly_maintenance_window.test"
	var windowID, pageID, componentA string
	accTestCase(t, []resource.TestStep{
		{
			Config: window(`
				status_page_visibility = {
					enabled         = true
					status_page_ids = [checkly_status_page_v3.test.id]

					component_configs = [{
						status_page_id = checkly_status_page_v3.test.id
						components = [{
							component_id = checkly_status_page_v3_component.a.id
							status       = "MAJOR_OUTAGE"
						}]
					}]
				}
			`),
			Check: func(s *terraform.State) error {
				resources := s.RootModule().Resources
				windowID = resources[r].Primary.ID
				pageID = resources["checkly_status_page_v3.test"].Primary.ID
				componentA = resources["checkly_status_page_v3_component.a"].Primary.ID
				return nil
			},
		},
		{
			Config: window(`
				status_page_visibility = {
					enabled         = true
					severity        = "MINOR"
					status_page_ids = [checkly_status_page_v3.test.id]
				}

				lifecycle {
					ignore_changes = [status_page_visibility.component_configs]
				}
			`),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(r, "status_page_visibility.severity", "MINOR"),
				func(*terraform.State) error {
					got, _, err := storedComponentStatuses(windowID)
					if err != nil {
						return err
					}
					want := map[string]map[string]string{pageID: {componentA: "MAJOR_OUTAGE"}}
					if !reflect.DeepEqual(got, want) {
						return fmt.Errorf("stored component configs = %v, want %v", got, want)
					}
					return nil
				},
			),
		},
	})
}

// TestAccMaintenanceWindowsMovedToMaintenanceWindow exercises the MoveState
// implementation of checkly_maintenance_window via a `moved` block from
// checkly_maintenance_windows.
func TestAccMaintenanceWindowsMovedToMaintenanceWindow(t *testing.T) {
	name := fmt.Sprintf("Maintenance Window %d", acctest.RandInt())
	accTestCase(t, []resource.TestStep{
		{
			Config: fmt.Sprintf(`
				resource "checkly_maintenance_windows" "test" {
					name      = %q
					starts_at = "2099-01-01T09:00:00.000Z"
					ends_at   = "2099-01-01T10:00:00.000Z"
					tags      = ["tf-acc"]
				}
			`, name),
		},
		{
			Config: fmt.Sprintf(`
				resource "checkly_maintenance_window" "test" {
					name      = %q
					starts_at = "2099-01-01T09:00:00.000Z"
					ends_at   = "2099-01-01T10:00:00.000Z"
					tags      = ["tf-acc"]
				}

				moved {
					from = checkly_maintenance_windows.test
					to   = checkly_maintenance_window.test
				}
			`, name),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_maintenance_window.test", "name", name),
				resource.TestCheckResourceAttr("checkly_maintenance_window.test", "tags.#", "1"),
			),
		},
	})
}
