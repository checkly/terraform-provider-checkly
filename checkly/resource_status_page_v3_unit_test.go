package checkly

import (
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"

	checkly "github.com/checkly/checkly-go-sdk"
)

func TestEncodeDecodeStatusPageV3Resource(t *testing.T) {
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
	data := resourceStatusPageV3().TestResourceData()
	if err := resourceDataFromStatusPageV3(&want, data); err != nil {
		t.Fatal(err)
	}
	got := statusPageV3FromResourceData(data)
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestEncodeDecodeStatusPageV3ThemeColors(t *testing.T) {
	want := checkly.StatusPageV3{
		Name: "Foo v3 status page",
		URL:  "foo-v3-status-page",
		ThemeColors: &checkly.StatusPageV3ThemeColors{
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
		},
	}
	data := resourceStatusPageV3().TestResourceData()
	// Configured: the palette is tracked.
	if err := data.Set("theme_colors", statusPageV3ThemeColorsToList(want.ThemeColors)); err != nil {
		t.Fatal(err)
	}
	if err := resourceDataFromStatusPageV3(&want, data); err != nil {
		t.Fatal(err)
	}
	got := statusPageV3FromResourceData(data)
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
	if got.ThemeColors.Light.LinkFontColor != "#FF0000" || got.ThemeColors.Dark.LinkFontColor != "#248AFF" {
		t.Errorf("expected the link colors to round-trip, got %+v", got.ThemeColors)
	}
}

func TestStatusPageV3ThemeColorsNotTrackedUnlessConfigured(t *testing.T) {
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
	data := resourceStatusPageV3().TestResourceData()
	if err := resourceDataFromStatusPageV3(&remote, data); err != nil {
		t.Fatal(err)
	}
	if got := statusPageV3FromResourceData(data); got.ThemeColors != nil {
		t.Errorf("expected no theme colors in state, got %+v", got.ThemeColors)
	}
}

func TestStatusPageV3HexColorValidation(t *testing.T) {
	for _, valid := range []string{"#FFF", "#ff0000", "#005AC2"} {
		if _, errs := validateStatusPageV3HexColor(valid, "link_font_color"); len(errs) > 0 {
			t.Errorf("expected %q to be valid, got %v", valid, errs)
		}
	}
	for _, invalid := range []string{"red", "FF0000", "#12345", "#GGGGGG", ""} {
		if _, errs := validateStatusPageV3HexColor(invalid, "link_font_color"); len(errs) == 0 {
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
	data := resourceStatusPageV3Component().TestResourceData()
	if err := resourceDataFromStatusPageV3Component(&want, data); err != nil {
		t.Fatal(err)
	}
	got, err := statusPageV3ComponentFromResourceData(data)
	if err != nil {
		t.Fatal(err)
	}
	// StatusPageID travels through the "status_page_id" attribute, not the
	// component payload.
	got.StatusPageID = data.Get("status_page_id").(string)
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
	data := resourceStatusPageV3Component().TestResourceData()
	if err := resourceDataFromStatusPageV3Component(&want, data); err != nil {
		t.Fatal(err)
	}
	got, err := statusPageV3ComponentFromResourceData(data)
	if err != nil {
		t.Fatal(err)
	}
	got.StatusPageID = data.Get("status_page_id").(string)
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestStatusPageV3ServiceComponentRejectsExpandedByDefault(t *testing.T) {
	data := resourceStatusPageV3Component().TestResourceData()
	if err := data.Set("type", "SERVICE"); err != nil {
		t.Fatal(err)
	}
	if err := data.Set("name", "Foo API"); err != nil {
		t.Fatal(err)
	}
	if err := data.Set("expanded_by_default", true); err != nil {
		t.Fatal(err)
	}
	if _, err := statusPageV3ComponentFromResourceData(data); err == nil {
		t.Error("expected an error for expanded_by_default on a SERVICE component")
	}
}

func TestEncodeDecodeStatusPageV3AutomationRuleResource(t *testing.T) {
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
	data := resourceStatusPageV3AutomationRule().TestResourceData()
	if err := resourceDataFromStatusPageV3AutomationRule(&want, data); err != nil {
		t.Fatal(err)
	}
	got, err := statusPageV3AutomationRuleFromResourceData(data)
	if err != nil {
		t.Fatal(err)
	}
	got.StatusPageID = data.Get("status_page_id").(string)
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
	data := resourceStatusPageV3AutomationRule().TestResourceData()
	if err := resourceDataFromStatusPageV3AutomationRule(&rule, data); err != nil {
		t.Fatal(err)
	}
	if _, err := statusPageV3AutomationRuleFromResourceData(data); err == nil {
		t.Error("expected an error for a duplicate component_id")
	}
}
