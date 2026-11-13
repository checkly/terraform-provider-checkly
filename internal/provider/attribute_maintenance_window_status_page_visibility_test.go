package provider

import (
	"context"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

const (
	testV3PageA = "5e1c4c6e-3c2b-4a43-9d0a-1c7f1f2b8a01"
	testV3PageB = "5e1c4c6e-3c2b-4a43-9d0a-1c7f1f2b8a02"
	testV3CompA = "7a9d2b10-5f4e-4c1d-8e2a-3b6c9d0e1f01"
	testV3CompB = "7a9d2b10-5f4e-4c1d-8e2a-3b6c9d0e1f02"
)

// testComponents returns a component_configs.components set value.
func testComponents(t *testing.T, impacts ...MaintenanceWindowComponentImpactModel) types.Set {
	t.Helper()
	// A nil slice would convert to a null set.
	if impacts == nil {
		impacts = []MaintenanceWindowComponentImpactModel{}
	}
	set, d := types.SetValueFrom(context.Background(), types.ObjectType{AttrTypes: maintenanceWindowComponentImpactAttrTypes}, impacts)
	if d.HasError() {
		t.Fatal(d)
	}
	return set
}

// testComponentConfigs returns a component_configs set value.
func testComponentConfigs(t *testing.T, configs ...MaintenanceWindowComponentConfigModel) types.Set {
	t.Helper()
	// A nil slice would convert to a null set.
	if configs == nil {
		configs = []MaintenanceWindowComponentConfigModel{}
	}
	set, d := types.SetValueFrom(context.Background(), types.ObjectType{AttrTypes: maintenanceWindowComponentConfigAttrTypes}, configs)
	if d.HasError() {
		t.Fatal(d)
	}
	return set
}

// testStatusPageVisibility returns an enabled status_page_visibility value
// linking statusPageIDs with the given component configs, with schema
// defaults applied.
func testStatusPageVisibility(t *testing.T, statusPageIDs []string, configs types.Set) types.Object {
	t.Helper()
	ctx := context.Background()
	var diags diag.Diagnostics
	pageIDs := setFromStrings(ctx, statusPageIDs, &diags)
	obj, d := types.ObjectValueFrom(ctx, maintenanceWindowStatusPageVisibilityAttrTypes, MaintenanceWindowStatusPageVisibilityModel{
		Enabled:               types.BoolValue(true),
		Severity:              types.StringNull(),
		AffectAllServices:     types.BoolValue(false),
		NotifyOnStart:         types.BoolValue(false),
		NotifyOnEnd:           types.BoolValue(false),
		SuppressAutoIncidents: types.BoolValue(false),
		ReminderMinutesBefore: types.SetValueMust(types.Int64Type, []attr.Value{}),
		AutoStart:             types.BoolValue(true),
		AutoEnd:               types.BoolValue(true),
		ShowAffectedServices:  types.BoolValue(true),
		StatusPageIDs:         pageIDs,
		ServiceIDs:            types.SetValueMust(types.StringType, []attr.Value{}),
		ComponentConfigs:      configs,
	})
	diags.Append(d...)
	if diags.HasError() {
		t.Fatal(diags)
	}
	return obj
}

// plannedComponentConfigs returns the planned component configs of the
// round-trip tests: page A with two components, one with an explicit status
// and one with the default, and page B with all of its components affected.
func plannedComponentConfigs(t *testing.T) types.Set {
	t.Helper()
	return testComponentConfigs(t,
		MaintenanceWindowComponentConfigModel{
			StatusPageID:        types.StringValue(testV3PageA),
			AffectAllComponents: types.BoolValue(false),
			Components: testComponents(t,
				MaintenanceWindowComponentImpactModel{ComponentID: types.StringValue(testV3CompA), Status: types.StringValue("MAJOR_OUTAGE")},
				MaintenanceWindowComponentImpactModel{ComponentID: types.StringValue(testV3CompB), Status: types.StringValue("UNDER_MAINTENANCE")},
			),
		},
		MaintenanceWindowComponentConfigModel{
			StatusPageID:        types.StringValue(testV3PageB),
			AffectAllComponents: types.BoolValue(true),
			Components:          testComponents(t),
		},
	)
}

// The component configs sent to the API match the configuration, and the
// API's response, which reports an entry that affects all components with an
// empty components list, reads back as the same value with or without a
// prior value, so an import of the window plans no change.
func TestMaintenanceWindowComponentConfigsRoundTrip(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	planned := testStatusPageVisibility(t, []string{testV3PageA, testV3PageB}, plannedComponentConfigs(t))

	var diags diag.Diagnostics
	sent := maintenanceWindowStatusPageVisibilityFromObject(ctx, planned, &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}
	configs := sent.ComponentConfigs
	sort.Slice(configs, func(i, j int) bool { return configs[i].StatusPageID < configs[j].StatusPageID })
	for _, config := range configs {
		sort.Slice(config.Components, func(i, j int) bool { return config.Components[i].ComponentID < config.Components[j].ComponentID })
	}
	want := []checkly.MaintenanceWindowComponentConfig{
		{
			StatusPageID: testV3PageA,
			Components: []checkly.MaintenanceWindowComponentImpact{
				{ComponentID: testV3CompA, Status: checkly.StatusPageComponentStatusV3MajorOutage},
				{ComponentID: testV3CompB, Status: checkly.StatusPageComponentStatusV3UnderMaintenance},
			},
		},
		{StatusPageID: testV3PageB, AffectAllComponents: true},
	}
	if !reflect.DeepEqual(configs, want) {
		t.Errorf("ComponentConfigs = %+v, want %+v", configs, want)
	}

	api := sent
	api.ComponentConfigs = []checkly.MaintenanceWindowComponentConfig{
		{StatusPageID: testV3PageB, AffectAllComponents: true, Components: []checkly.MaintenanceWindowComponentImpact{}},
		want[0],
	}
	for name, prior := range map[string]types.Object{
		"prior state": planned,
		"import":      types.ObjectNull(maintenanceWindowStatusPageVisibilityAttrTypes),
	} {
		t.Run(name, func(t *testing.T) {
			var diags diag.Diagnostics
			got := maintenanceWindowStatusPageVisibilityToObject(ctx, api, prior, &diags)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if !got.Equal(planned) {
				t.Errorf("status_page_visibility = %v, want %v", got, planned)
			}
		})
	}
}

// Removing component_configs, or status_page_visibility altogether, sends an
// empty list, which removes the stored component configs; the API keeps them
// when the field is left out.
func TestMaintenanceWindowComponentConfigsFromObjectSendsEmpty(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	for name, obj := range map[string]types.Object{
		"null component_configs":      testStatusPageVisibility(t, nil, types.SetNull(types.ObjectType{AttrTypes: maintenanceWindowComponentConfigAttrTypes})),
		"null status_page_visibility": types.ObjectNull(maintenanceWindowStatusPageVisibilityAttrTypes),
	} {
		t.Run(name, func(t *testing.T) {
			var diags diag.Diagnostics
			got := maintenanceWindowStatusPageVisibilityFromObject(ctx, obj, &diags)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if got.ComponentConfigs == nil || len(got.ComponentConfigs) != 0 {
				t.Errorf("ComponentConfigs = %#v, want an empty non-nil slice", got.ComponentConfigs)
			}
		})
	}
}

// No component configs from the API, and no components in an entry, read as
// empty sets whatever the prior value holds, like the other collections of
// status_page_visibility.
func TestMaintenanceWindowComponentConfigsToSetEmpty(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var diags diag.Diagnostics
	got := maintenanceWindowComponentConfigsToSet(ctx, []checkly.MaintenanceWindowComponentConfig{}, &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if want := testComponentConfigs(t); !got.Equal(want) {
		t.Errorf("component_configs = %v, want %v", got, want)
	}

	want := testComponentConfigs(t, MaintenanceWindowComponentConfigModel{
		StatusPageID:        types.StringValue(testV3PageA),
		AffectAllComponents: types.BoolValue(true),
		Components:          testComponents(t),
	})
	got = maintenanceWindowComponentConfigsToSet(ctx, []checkly.MaintenanceWindowComponentConfig{
		{StatusPageID: testV3PageA, AffectAllComponents: true},
	}, &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if !got.Equal(want) {
		t.Errorf("component_configs = %v, want %v", got, want)
	}
}

// The plan modifier fills in the defaults of the optional+computed attributes
// at both levels of component_configs, keeps configured values, and leaves a
// null status_page_visibility null.
func TestMaintenanceWindowComponentConfigsWithDefaults(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	withDefaults := objectWithSetDefaults("component_configs", maintenanceWindowComponentConfigsWithDefaults)
	impactType := types.ObjectType{AttrTypes: maintenanceWindowComponentImpactAttrTypes}

	var diags diag.Diagnostics
	// An unset components is null in the configuration and unknown in the
	// plan, since the attribute is computed; both become an empty set.
	for name, unsetComponents := range map[string]types.Set{
		"null components":    types.SetNull(impactType),
		"unknown components": types.SetUnknown(impactType),
	} {
		configured := testStatusPageVisibility(t, []string{testV3PageA, testV3PageB}, testComponentConfigs(t,
			MaintenanceWindowComponentConfigModel{
				StatusPageID:        types.StringValue(testV3PageA),
				AffectAllComponents: types.BoolUnknown(),
				Components: testComponents(t,
					MaintenanceWindowComponentImpactModel{ComponentID: types.StringValue(testV3CompA), Status: types.StringValue("MAJOR_OUTAGE")},
					MaintenanceWindowComponentImpactModel{ComponentID: types.StringValue(testV3CompB), Status: types.StringUnknown()},
				),
			},
			MaintenanceWindowComponentConfigModel{
				StatusPageID:        types.StringValue(testV3PageB),
				AffectAllComponents: types.BoolValue(true),
				Components:          unsetComponents,
			},
		))

		got := withDefaults(ctx, configured, &diags)
		if diags.HasError() {
			t.Fatal(diags)
		}
		want := testStatusPageVisibility(t, []string{testV3PageA, testV3PageB}, plannedComponentConfigs(t))
		if !got.Equal(want) {
			t.Errorf("%s: status_page_visibility = %v, want %v", name, got, want)
		}
	}

	null := types.ObjectNull(maintenanceWindowStatusPageVisibilityAttrTypes)
	if got := withDefaults(ctx, null, &diags); !got.IsNull() {
		t.Errorf("status_page_visibility = %v, want null", got)
	}
}

func TestValidateMaintenanceWindowComponentConfigs(t *testing.T) {
	t.Parallel()
	impactType := types.ObjectType{AttrTypes: maintenanceWindowComponentImpactAttrTypes}
	component := func(id string) MaintenanceWindowComponentImpactModel {
		return MaintenanceWindowComponentImpactModel{ComponentID: types.StringValue(id), Status: types.StringNull()}
	}
	entry := func(pageID types.String, affectAll types.Bool, components types.Set) MaintenanceWindowComponentConfigModel {
		return MaintenanceWindowComponentConfigModel{StatusPageID: pageID, AffectAllComponents: affectAll, Components: components}
	}
	pageA, pageB := types.StringValue(testV3PageA), types.StringValue(testV3PageB)
	linked := []string{testV3PageA, testV3PageB}

	cases := []struct {
		name    string
		pages   []string
		configs types.Set
		// wantError is a substring of the expected error summary, or empty.
		wantError string
	}{
		{
			name:    "valid",
			pages:   linked,
			configs: testComponentConfigs(t, entry(pageA, types.BoolNull(), testComponents(t, component(testV3CompA))), entry(pageB, types.BoolValue(true), types.SetNull(impactType))),
		},
		{
			name:    "unknown page ID",
			pages:   linked,
			configs: testComponentConfigs(t, entry(types.StringUnknown(), types.BoolNull(), testComponents(t, component(testV3CompA)))),
		},
		{
			name:    "unknown components",
			pages:   linked,
			configs: testComponentConfigs(t, entry(pageA, types.BoolNull(), types.SetUnknown(impactType))),
		},
		{
			name:      "page not linked",
			pages:     []string{testV3PageB},
			configs:   testComponentConfigs(t, entry(pageA, types.BoolValue(true), types.SetNull(impactType))),
			wantError: "Unlinked Status Page",
		},
		{
			name:      "no pages linked",
			configs:   testComponentConfigs(t, entry(pageA, types.BoolValue(true), types.SetNull(impactType))),
			wantError: "Unlinked Status Page",
		},
		{
			name:      "page listed twice",
			pages:     linked,
			configs:   testComponentConfigs(t, entry(pageA, types.BoolValue(true), types.SetNull(impactType)), entry(pageA, types.BoolNull(), testComponents(t, component(testV3CompA)))),
			wantError: "Duplicate Status Page",
		},
		{
			name:      "affect all with components",
			pages:     linked,
			configs:   testComponentConfigs(t, entry(pageA, types.BoolValue(true), testComponents(t, component(testV3CompA)))),
			wantError: "Conflicting Component Configuration",
		},
		{
			name:      "neither affect all nor components",
			pages:     linked,
			configs:   testComponentConfigs(t, entry(pageA, types.BoolValue(false), types.SetNull(impactType))),
			wantError: "Missing Components",
		},
		{
			name:      "empty components",
			pages:     linked,
			configs:   testComponentConfigs(t, entry(pageA, types.BoolNull(), testComponents(t))),
			wantError: "Missing Components",
		},
		{
			name:    "affect all with empty components",
			pages:   linked,
			configs: testComponentConfigs(t, entry(pageA, types.BoolValue(true), testComponents(t))),
		},
		{
			name:  "component listed twice",
			pages: linked,
			configs: testComponentConfigs(t, entry(pageA, types.BoolNull(), testComponents(t,
				component(testV3CompA),
				MaintenanceWindowComponentImpactModel{ComponentID: types.StringValue(testV3CompA), Status: types.StringValue("MAJOR_OUTAGE")},
			))),
			wantError: "Duplicate Component",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var diags diag.Diagnostics
			validateMaintenanceWindowComponentConfigs(context.Background(), testStatusPageVisibility(t, tc.pages, tc.configs), &diags)
			if tc.wantError == "" {
				if diags.HasError() {
					t.Errorf("unexpected error: %v", diags)
				}
				return
			}
			if !diags.HasError() {
				t.Fatalf("expected a %q error", tc.wantError)
			}
			for _, d := range diags.Errors() {
				if strings.Contains(d.Summary(), tc.wantError) {
					return
				}
			}
			t.Errorf("errors = %v, want one containing %q", diags, tc.wantError)
		})
	}

	// Pages that are not known yet skip the linked-page check.
	var diags diag.Diagnostics
	unknownPages := testStatusPageVisibility(t, nil, testComponentConfigs(t, entry(pageA, types.BoolValue(true), types.SetNull(impactType))))
	attrs := unknownPages.Attributes()
	attrs["status_page_ids"] = types.SetValueMust(types.StringType, []attr.Value{types.StringUnknown()})
	obj, d := types.ObjectValue(maintenanceWindowStatusPageVisibilityAttrTypes, attrs)
	if d.HasError() {
		t.Fatal(d)
	}
	validateMaintenanceWindowComponentConfigs(context.Background(), obj, &diags)
	if diags.HasError() {
		t.Errorf("unexpected error with unknown status_page_ids: %v", diags)
	}
}

func TestInt64SetFromInts(t *testing.T) {
	ctx := context.Background()
	empty := types.SetValueMust(types.Int64Type, []attr.Value{})
	values := types.SetValueMust(types.Int64Type, []attr.Value{types.Int64Value(15), types.Int64Value(60)})

	cases := map[string]struct {
		values []int
		want   types.Set
	}{
		"nil":    {nil, empty},
		"empty":  {[]int{}, empty},
		"values": {[]int{15, 60}, values},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			var diags diag.Diagnostics
			if got := int64SetFromInts(ctx, c.values, &diags); !got.Equal(c.want) {
				t.Errorf("int64SetFromInts = %s, want %s", got, c.want)
			}
			if diags.HasError() {
				t.Fatal(diags)
			}
		})
	}
}
