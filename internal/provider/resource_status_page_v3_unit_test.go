package provider

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
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
		ThemeColors: statusPageV3ThemeColorsToList(want.ThemeColors, types.ListNull(statusPageV3ThemeColorsObjectType), &diags),
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
		ThemeColors: statusPageV3ThemeColorsToList(configured, types.ListNull(statusPageV3ThemeColorsObjectType), &diags),
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
	// without a theme_colors block it must not land in state.
	remote := checkly.StatusPageV3{
		Name: "Foo v3 status page",
		URL:  "foo-v3-status-page",
		ThemeColors: &checkly.StatusPageV3ThemeColors{
			Light: checkly.StatusPageV3ThemeColorGroup{LinkFontColor: "#005AC2"},
			Dark:  checkly.StatusPageV3ThemeColorGroup{LinkFontColor: "#248AFF"},
		},
	}
	for name, prior := range map[string]types.List{
		"null":  types.ListNull(statusPageV3ThemeColorsObjectType),
		"empty": types.ListValueMust(statusPageV3ThemeColorsObjectType, []attr.Value{}),
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
		if model.ThemeColors.IsNull() || len(model.ThemeColors.Elements()) != 0 {
			t.Errorf("%s: expected an empty theme_colors block list, got %s", name, model.ThemeColors)
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
	// tags and component are sets: their order is not preserved.
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

func TestIsStatusPageV3NotFound(t *testing.T) {
	if !isStatusPageV3NotFound(errors.New(`unexpected response status 404: "{\"statusCode\":404}"`)) {
		t.Error("expected a 404 response to be reported as not found")
	}
	// The 400 body quotes component names, which may contain "404".
	if isStatusPageV3NotFound(errors.New(`unexpected response status 400: "\"API 404\" is not a GROUP component."`)) {
		t.Error("expected a 400 response to not be reported as not found")
	}
	if isStatusPageV3NotFound(errors.New(`unexpected response status 400: "\"unexpected response status 404: x\" is not a GROUP component"`)) {
		t.Error("expected a 400 response quoting the 404 prefix to not be reported as not found")
	}
}
