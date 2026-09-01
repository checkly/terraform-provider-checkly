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

func TestEncodeDecodeStatusPageV3ComponentResource(t *testing.T) {
	want := checkly.StatusPageComponentV3{
		StatusPageID: "e35f7e14-91b2-4d24-b7b6-e0f9e2f8e51c",
		Type:         checkly.StatusPageComponentV3TypeService,
		Name:         "Foo API",
		Description:  "The Foo public API",
		DisplayOrder: 3,
		Hidden:       true,
		ParentID:     "0a2f26fb-47cc-42b7-91c6-40de3ec91a52",
	}
	data := resourceStatusPageV3Component().TestResourceData()
	if err := resourceDataFromStatusPageV3Component(&want, data); err != nil {
		t.Fatal(err)
	}
	got := statusPageV3ComponentFromResourceData(data)
	// StatusPageID travels through the "status_page_id" attribute, not the
	// component payload.
	got.StatusPageID = data.Get("status_page_id").(string)
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
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
