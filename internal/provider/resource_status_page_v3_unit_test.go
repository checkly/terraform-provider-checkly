package provider

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/checkly/checkly-go-sdk/v2"
)

func TestEncodeDecodeStatusPageV3Resource(t *testing.T) {
	ctx := context.Background()
	want := checkly.StatusPageV3{
		Name:               "Foo v3 status page",
		URL:                "foo-v3-status-page",
		CustomDomain:       "status.example.org",
		Description:        "All Foo systems",
		Logo:               "https://example.org/logo.png",
		LogoDark:           "https://example.org/logo-dark.png",
		RedirectTo:         "https://example.org",
		Favicon:            "https://example.org/favicon.png",
		DefaultTheme:       checkly.StatusPageThemeDark,
		PrivacyPolicyLink:  "https://example.org/privacy",
		TermsOfServiceLink: "https://example.org/terms",
		SupportLink:        "mailto:support@example.org",
		FooterText:         "Foo Inc.",
		GoogleAnalyticsTag: "G-XXXXXXXXXX",
		AllowIndexing:      true,
	}
	var diags diag.Diagnostics
	var model StatusPageV3ResourceModel
	statusPageV3ToModel(ctx, &want, &model, &diags)
	got := statusPageV3FromModel(ctx, &model, &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func testStatusPageV3ThemeColors() *checkly.StatusPageV3ThemeColors {
	return &checkly.StatusPageV3ThemeColors{
		Light: checkly.StatusPageV3ThemeColorGroup{
			BodyBackgroundColor:          "#F9FAFB",
			HeaderBackgroundColor:        "#FFFFFF",
			HeaderFontColor:              "#151A1E",
			TitleFontColor:               "#212930",
			BodyFontColor:                "#475766",
			BodyFontColorMuted:           "#60758A",
			NavigationFontColor:          "#151A1E",
			LinkFontColor:                "#FF0000",
			CardBackgroundColor:          "#FFFFFF",
			BorderColor:                  "#E0E5EB",
			PrimaryButtonBackgroundColor: "#151A1E",
			PrimaryButtonFontColor:       "#FFFFFF",
		},
		Dark: checkly.StatusPageV3ThemeColorGroup{
			BodyBackgroundColor:          "#14171C",
			HeaderBackgroundColor:        "#171B21",
			HeaderFontColor:              "#FFFFFF",
			TitleFontColor:               "#ECEEF2",
			BodyFontColor:                "#C6CDD7",
			BodyFontColorMuted:           "#A3B3C2",
			NavigationFontColor:          "#FFFFFF",
			LinkFontColor:                "#248AFF",
			CardBackgroundColor:          "#171B21",
			BorderColor:                  "#242B34",
			PrimaryButtonBackgroundColor: "#242B34",
			PrimaryButtonFontColor:       "#FFFFFF",
		},
	}
}

func TestEncodeDecodeStatusPageV3ThemeColors(t *testing.T) {
	ctx := context.Background()
	want := checkly.StatusPageV3{
		Name:        "Foo v3 status page",
		URL:         "foo-v3-status-page",
		ThemeColors: testStatusPageV3ThemeColors(),
	}
	var diags diag.Diagnostics
	// Configured: the palette is tracked.
	model := StatusPageV3ResourceModel{
		ThemeColors: statusPageV3ThemeColorsToObject(want.ThemeColors, types.ObjectNull(statusPageV3ThemeColorsAttrTypes), &diags),
	}
	statusPageV3ToModel(ctx, &want, &model, &diags)
	got := statusPageV3FromModel(ctx, &model, &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
	if got.ThemeColors.Light.LinkFontColor != "#FF0000" || got.ThemeColors.Dark.LinkFontColor != "#248AFF" {
		t.Errorf("expected the link colors to round-trip, got %+v", got.ThemeColors)
	}
}

func TestStatusPageV3ThemeColorsKeepConfiguredCase(t *testing.T) {
	ctx := context.Background()
	configured := testStatusPageV3ThemeColors()
	configured.Light.LinkFontColor = "#ff0000"
	remote := checkly.StatusPageV3{
		Name:        "Foo v3 status page",
		URL:         "foo-v3-status-page",
		ThemeColors: testStatusPageV3ThemeColors(),
	}
	var diags diag.Diagnostics
	model := StatusPageV3ResourceModel{
		ThemeColors: statusPageV3ThemeColorsToObject(configured, types.ObjectNull(statusPageV3ThemeColorsAttrTypes), &diags),
	}
	statusPageV3ToModel(ctx, &remote, &model, &diags)
	got := statusPageV3FromModel(ctx, &model, &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if got.ThemeColors.Light.LinkFontColor != "#ff0000" {
		t.Errorf("expected the configured spelling to be kept, got %s", got.ThemeColors.Light.LinkFontColor)
	}
}

func TestStatusPageV3ThemeColorsNotTrackedUnlessConfigured(t *testing.T) {
	ctx := context.Background()
	// The API always reports a palette (the defaults when none is set);
	// without a theme_colors attribute it must not land in state.
	remote := checkly.StatusPageV3{
		Name: "Foo v3 status page",
		URL:  "foo-v3-status-page",
		ThemeColors: &checkly.StatusPageV3ThemeColors{
			Light: checkly.StatusPageV3ThemeColorGroup{LinkFontColor: "#005AC2"},
			Dark:  checkly.StatusPageV3ThemeColorGroup{LinkFontColor: "#248AFF"},
		},
	}
	for name, prior := range map[string]types.Object{
		"null":    types.ObjectNull(statusPageV3ThemeColorsAttrTypes),
		"unknown": types.ObjectUnknown(statusPageV3ThemeColorsAttrTypes),
	} {
		var diags diag.Diagnostics
		model := StatusPageV3ResourceModel{ThemeColors: prior}
		statusPageV3ToModel(ctx, &remote, &model, &diags)
		if diags.HasError() {
			t.Fatal(diags)
		}
		if got := statusPageV3FromModel(ctx, &model, &diags); got.ThemeColors != nil {
			t.Errorf("%s: expected no theme colors in state, got %+v", name, got.ThemeColors)
		}
		if !model.ThemeColors.IsNull() {
			t.Errorf("%s: expected a null theme_colors attribute, got %s", name, model.ThemeColors)
		}
	}
}

func TestStatusPageV3HexColorValidation(t *testing.T) {
	for _, valid := range []string{"#FFF", "#ff0000", "#005AC2"} {
		if !validateStatusPageV3HexColor(valid) {
			t.Errorf("expected %q to be valid", valid)
		}
	}
	for _, invalid := range []string{"red", "FF0000", "#12345", "#GGGGGG", ""} {
		if validateStatusPageV3HexColor(invalid) {
			t.Errorf("expected %q to be rejected", invalid)
		}
	}
}

func TestEncodeDecodeStatusPageV3ComponentResource(t *testing.T) {
	showHistoricalData := false
	want := checkly.StatusPageComponentV3{
		StatusPageID: "e35f7e14-91b2-4d24-b7b6-e0f9e2f8e51c",
		Type:         checkly.StatusPageComponentV3TypeService,
		Name:         "Foo API",
		Description:  "The Foo public API",
		DisplayOrder: 3,
		Hidden:       true,
		Configuration: &checkly.StatusPageComponentV3Configuration{
			ShowHistoricalData: &showHistoricalData,
		},
		ParentID: "0a2f26fb-47cc-42b7-91c6-40de3ec91a52",
	}
	var model StatusPageV3ComponentResourceModel
	statusPageV3ComponentToModel(&want, &model)
	got, err := statusPageV3ComponentFromModel(&model)
	if err != nil {
		t.Fatal(err)
	}
	// StatusPageID travels through the "status_page_id" attribute, not the
	// component payload.
	got.StatusPageID = model.StatusPageID.ValueString()
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestEncodeDecodeStatusPageV3GroupComponentResource(t *testing.T) {
	showHistoricalData := true
	expandedByDefault := true
	want := checkly.StatusPageComponentV3{
		StatusPageID: "e35f7e14-91b2-4d24-b7b6-e0f9e2f8e51c",
		Type:         checkly.StatusPageComponentV3TypeGroup,
		Name:         "Foo group",
		DisplayOrder: 0,
		Configuration: &checkly.StatusPageComponentV3Configuration{
			ShowHistoricalData: &showHistoricalData,
			ExpandedByDefault:  &expandedByDefault,
		},
	}
	var model StatusPageV3ComponentResourceModel
	statusPageV3ComponentToModel(&want, &model)
	got, err := statusPageV3ComponentFromModel(&model)
	if err != nil {
		t.Fatal(err)
	}
	got.StatusPageID = model.StatusPageID.ValueString()
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestStatusPageV3ServiceComponentRejectsExpandedByDefault(t *testing.T) {
	model := StatusPageV3ComponentResourceModel{
		Type:               types.StringValue("SERVICE"),
		Name:               types.StringValue("Foo API"),
		ShowHistoricalData: types.BoolValue(true),
		ExpandedByDefault:  types.BoolValue(true),
	}
	if _, err := statusPageV3ComponentFromModel(&model); err == nil {
		t.Error("expected an error for expanded_by_default on a SERVICE component")
	}
}

func TestEncodeDecodeStatusPageV3AutomationRuleResource(t *testing.T) {
	ctx := context.Background()
	want := checkly.StatusPageAutomationRuleV3{
		StatusPageID:          "e35f7e14-91b2-4d24-b7b6-e0f9e2f8e51c",
		Name:                  "Foo API outage",
		Enabled:               true,
		FirstUpdate:           "We are investigating an issue.",
		LastUpdate:            "The issue has been resolved.",
		NotifySubscribers:     false,
		CoolDownWindowMinutes: 30,
		Tags:                  []string{"bar-api", "foo-api"},
		Components: []checkly.StatusPageAutomationRuleComponentV3{
			{
				ComponentID:  "0e4f5a72-6a5c-42a1-9a8a-5f7d38c8a9d1",
				TargetImpact: checkly.StatusPageTargetImpactV3MajorOutage,
			},
			{
				ComponentID:  "b7a2f0cd-4f8e-4ec0-8f5a-1c9a7e2d3b40",
				TargetImpact: checkly.StatusPageTargetImpactV3DegradedPerformance,
			},
		},
	}
	var diags diag.Diagnostics
	var model StatusPageV3AutomationRuleResourceModel
	statusPageV3AutomationRuleToModel(ctx, &want, &model, &diags)
	got, err := statusPageV3AutomationRuleFromModel(ctx, &model, &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if err != nil {
		t.Fatal(err)
	}
	got.StatusPageID = model.StatusPageID.ValueString()
	// tags and components are sets: their order is not preserved.
	sort.Strings(got.Tags)
	sort.Slice(got.Components, func(i, j int) bool {
		return got.Components[i].ComponentID < got.Components[j].ComponentID
	})
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestStatusPageV3AutomationRuleDuplicateComponents(t *testing.T) {
	ctx := context.Background()
	rule := checkly.StatusPageAutomationRuleV3{
		Name:        "Foo API outage",
		FirstUpdate: "Investigating.",
		LastUpdate:  "Resolved.",
		Tags:        []string{"foo-api"},
		Components: []checkly.StatusPageAutomationRuleComponentV3{
			{
				ComponentID:  "0e4f5a72-6a5c-42a1-9a8a-5f7d38c8a9d1",
				TargetImpact: checkly.StatusPageTargetImpactV3MajorOutage,
			},
			{
				ComponentID:  "0e4f5a72-6a5c-42a1-9a8a-5f7d38c8a9d1",
				TargetImpact: checkly.StatusPageTargetImpactV3PartialOutage,
			},
		},
	}
	var diags diag.Diagnostics
	var model StatusPageV3AutomationRuleResourceModel
	statusPageV3AutomationRuleToModel(ctx, &rule, &model, &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if _, err := statusPageV3AutomationRuleFromModel(ctx, &model, &diags); err == nil {
		t.Error("expected an error for a duplicate component_id")
	}
}

// The API reports no components as an empty array, which converts to an
// empty set whatever the prior value.
func TestStatusPageV3AutomationRuleComponentsEmpty(t *testing.T) {
	ctx := context.Background()
	componentType := types.ObjectType{AttrTypes: statusPageV3AutomationRuleComponentAttrTypes}
	empty := types.SetValueMust(componentType, nil)

	for name, c := range map[string]struct {
		prior types.Set
		want  types.Set
	}{
		"prior null":    {prior: types.SetNull(componentType), want: empty},
		"prior unknown": {prior: types.SetUnknown(componentType), want: empty},
		"prior empty":   {prior: empty, want: empty},
	} {
		t.Run(name, func(t *testing.T) {
			var diags diag.Diagnostics
			model := StatusPageV3AutomationRuleResourceModel{Components: c.prior}
			statusPageV3AutomationRuleToModel(ctx, &checkly.StatusPageAutomationRuleV3{Tags: []string{"t"}}, &model, &diags)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if !model.Components.Equal(c.want) {
				t.Errorf("components = %s, want %s", model.Components, c.want)
			}
		})
	}
}

func TestStatusPageV3CustomDomainValidation(t *testing.T) {
	for _, valid := range []string{"", "status.example.com", "a.b.example.co.uk", "status-1.example.com"} {
		if !statusPageV3CustomDomainRegex.MatchString(valid) {
			t.Errorf("expected %q to be accepted", valid)
		}
	}
	for _, invalid := range []string{"Status.example.com", "https://status.example.com", "status.example.com/", "example", "-status.example.com"} {
		if statusPageV3CustomDomainRegex.MatchString(invalid) {
			t.Errorf("expected %q to be rejected", invalid)
		}
	}
}

// The v3 status page attributes reject at plan time what the API rejects.
func TestStatusPageV3AttributeValidation(t *testing.T) {
	r := NewStatusPageV3Resource()
	cases := []struct {
		attribute string
		value     string
		valid     bool
	}{
		{"description", strings.Repeat("a", 500), true},
		{"description", strings.Repeat("a", 501), false},
		{"footer_text", strings.Repeat("a", 1000), true},
		{"footer_text", strings.Repeat("a", 1001), false},
		{"logo", "", true},
		{"logo", "https://example.org/logo.png", true},
		{"logo", "logo.png", false},
		{"logo_dark", "ftp://example.org/logo.png", false},
		{"redirect_to", "http://example.org", true},
		{"redirect_to", "example.org", false},
		{"favicon", "data:image/png;base64,AAAA", false},
		{"privacy_policy_link", "https://example.org/privacy", true},
		{"privacy_policy_link", "https://example.org/" + strings.Repeat("a", 236), false},
		{"privacy_policy_link", "mailto:legal@example.org", false},
		{"terms_of_service_link", "", true},
		{"terms_of_service_link", "/terms", false},
		{"support_link", "mailto:support@example.org", true},
		{"support_link", "https://example.org/help", true},
		{"support_link", "support@example.org", false},
		{"google_analytics_tag", "", true},
		{"google_analytics_tag", "G-XXXXXXXXXX", true},
		{"google_analytics_tag", "GTM-abc123", true},
		{"google_analytics_tag", "UA-12345-1", true},
		{"google_analytics_tag", "G_XXXX", false},
		{"google_analytics_tag", "G-XXXX ", false},
		{"google_analytics_tag", "G.XXXX", false},
		{"google_analytics_tag", strings.Repeat("a", 255), true},
		{"google_analytics_tag", "<script>", false},
		{"google_analytics_tag", strings.Repeat("a", 256), false},
	}
	for _, c := range cases {
		diags := validateStringAttribute(t, r, c.attribute, types.StringValue(c.value))
		if diags.HasError() == c.valid {
			t.Errorf("%s = %q: valid = %v, diagnostics: %v", c.attribute, c.value, c.valid, diags)
		}
	}
}

// A component_id listed twice with different impacts fails at plan time; a
// component_id that is still unknown is left for the apply-time check.
func TestStatusPageV3AutomationRuleValidateConfigDuplicateComponents(t *testing.T) {
	ctx := context.Background()
	r := NewStatusPageV3AutomationRuleResource()
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	objectType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	setType := objectType.AttributeTypes["components"].(tftypes.Set)
	elementType := setType.ElementType.(tftypes.Object)
	component := func(id any, impact string) tftypes.Value {
		return tftypes.NewValue(elementType, map[string]tftypes.Value{
			"component_id":  tftypes.NewValue(tftypes.String, id),
			"target_impact": tftypes.NewValue(tftypes.String, impact),
		})
	}
	const id = "0e4f5a72-6a5c-42a1-9a8a-5f7d38c8a9d1"

	for name, c := range map[string]struct {
		components []tftypes.Value
		wantError  bool
	}{
		"distinct":  {[]tftypes.Value{component(id, "MAJOR_OUTAGE"), component("1e4f5a72-6a5c-42a1-9a8a-5f7d38c8a9d1", "MAJOR_OUTAGE")}, false},
		"duplicate": {[]tftypes.Value{component(id, "MAJOR_OUTAGE"), component(id, "PARTIAL_OUTAGE")}, true},
		"unknown":   {[]tftypes.Value{component(id, "MAJOR_OUTAGE"), component(tftypes.UnknownValue, "PARTIAL_OUTAGE")}, false},
		// A whole element from a variable that is not known yet.
		"unknown element": {[]tftypes.Value{component(id, "MAJOR_OUTAGE"), tftypes.NewValue(elementType, tftypes.UnknownValue)}, false},
		"duplicate next to an unknown element": {[]tftypes.Value{
			component(id, "MAJOR_OUTAGE"), component(id, "PARTIAL_OUTAGE"), tftypes.NewValue(elementType, tftypes.UnknownValue),
		}, true},
	} {
		t.Run(name, func(t *testing.T) {
			config := tfsdk.Config{Schema: schemaResp.Schema, Raw: objectWith(objectType, map[string]tftypes.Value{
				"components": tftypes.NewValue(setType, c.components),
			})}
			var resp resource.ValidateConfigResponse
			r.(resource.ResourceWithValidateConfig).ValidateConfig(ctx, resource.ValidateConfigRequest{Config: config}, &resp)
			if resp.Diagnostics.HasError() != c.wantError {
				t.Errorf("error = %v, want %v: %v", resp.Diagnostics.HasError(), c.wantError, resp.Diagnostics)
			}
		})
	}
}

func TestStatusPageV3AutomationRuleRejectsEmptyStrings(t *testing.T) {
	r := NewStatusPageV3AutomationRuleResource()
	for _, attribute := range []string{"first_update", "last_update"} {
		if diags := validateStringAttribute(t, r, attribute, types.StringValue("")); !diags.HasError() {
			t.Errorf("an empty %s was accepted", attribute)
		}
	}

	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	tags := schemaResp.Schema.Attributes["tags"].(schema.SetAttribute)
	for name, c := range map[string]struct {
		elements []string
		valid    bool
	}{
		"one tag":   {[]string{"foo-api"}, true},
		"empty tag": {[]string{"foo-api", ""}, false},
	} {
		value, d := types.SetValueFrom(ctx, types.StringType, c.elements)
		if d.HasError() {
			t.Fatal(d)
		}
		req := validator.SetRequest{Path: path.Root("tags"), ConfigValue: value}
		var resp validator.SetResponse
		for _, v := range tags.Validators {
			v.ValidateSet(ctx, req, &resp)
		}
		if resp.Diagnostics.HasError() == c.valid {
			t.Errorf("%s: valid = %v, diagnostics: %v", name, c.valid, resp.Diagnostics)
		}
	}
}
