package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/checkly/checkly-go-sdk/v2"
)

func TestAccDashboardCheckRequiredFields(t *testing.T) {
	config := `resource "checkly_dashboard" "test" {}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "custom_url" is required`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "header" is required`),
		},
	})
}

func TestAccDashboardMinimalConfig(t *testing.T) {
	rInt := acctest.RandInt()
	accTestCase(t, []resource.TestStep{
		{
			Config: fmt.Sprintf(`
				resource "checkly_dashboard" "test" {
					custom_url = "test-dashboard-%d-minimal"
					header     = "Minimal Dashboard"
				}
			`, rInt),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"custom_url",
					fmt.Sprintf("test-dashboard-%d-minimal", rInt),
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"logo",
					"",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"favicon",
					"",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"link",
					"",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"description",
					"",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"header",
					"Minimal Dashboard",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"width",
					"FULL",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"refresh_rate",
					"60",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"paginate",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"checks_per_page",
					"15",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"pagination_rate",
					"60",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"hide_tags",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"use_tags_and_operator",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"enable_incidents",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"is_private",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"show_header",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"expand_checks",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"show_check_run_links",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"custom_css",
					"",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"show_p95",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"show_p99",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"sort_by_status",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"show_group_names",
					"true",
				),
			),
		},
	})
}

func TestAccDashboardFullConfig(t *testing.T) {
	rInt := acctest.RandInt()
	accTestCase(t, []resource.TestStep{
		{
			Config: fmt.Sprintf(`
				resource "checkly_dashboard" "test" {
					custom_url            = "test-dashboard-%d-full"
					custom_domain         = "status-%d.example.com"
					logo                  = "https://example.com/logo.png"
					favicon               = "https://example.com/favicon.ico"
					link                  = "https://example.com"
					description           = "Test dashboard description"
					header                = "System Status"
					show_header           = false
					width                 = "960PX"
					refresh_rate          = 300
					paginate              = false
					checks_per_page       = 20
					pagination_rate       = 30
					hide_tags             = true
					use_tags_and_operator = true
					enable_incidents      = true
					expand_checks         = true
					show_check_run_links  = true
					custom_css            = ".header { color: blue; }"
					show_p95              = false
					show_p99              = false
					sort_by_status        = true
					show_group_names      = false
					tags                  = ["production", "api"]
				}
			`, rInt, rInt),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"custom_url",
					fmt.Sprintf("test-dashboard-%d-full", rInt),
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"custom_domain",
					fmt.Sprintf("status-%d.example.com", rInt),
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"logo",
					"https://example.com/logo.png",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"favicon",
					"https://example.com/favicon.ico",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"link",
					"https://example.com",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"description",
					"Test dashboard description",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"header",
					"System Status",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"width",
					"960PX",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"refresh_rate",
					"300",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"paginate",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"checks_per_page",
					"20",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"pagination_rate",
					"30",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"hide_tags",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"use_tags_and_operator",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"enable_incidents",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"tags.#",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"show_header",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"expand_checks",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"show_check_run_links",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"custom_css",
					".header { color: blue; }",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"show_p95",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"show_p99",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"sort_by_status",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"show_group_names",
					"false",
				),
			),
		},
		{
			Config: fmt.Sprintf(`
				resource "checkly_dashboard" "test" {
					custom_url            = "test-dashboard-%d-full-updated"
					custom_domain         = "status-%d.example.com"
					logo                  = "https://example.com/logo2.png"
					favicon               = "https://example.com/favicon2.ico"
					link                  = "https://example2.com"
					description           = "Updated test dashboard description"
					header                = "Updated System Status"
					show_header           = true
					width                 = "FULL"
					refresh_rate          = 600
					paginate              = true
					checks_per_page       = 10
					pagination_rate       = 300
					hide_tags             = false
					use_tags_and_operator = false
					enable_incidents      = false
					expand_checks         = false
					show_check_run_links  = false
					custom_css            = "body { background: #f0f0f0; }"
					show_p95              = true
					show_p99              = true
					sort_by_status        = false
					show_group_names      = true
					tags                  = ["staging"]
				}
			`, rInt, rInt),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"custom_url",
					fmt.Sprintf("test-dashboard-%d-full-updated", rInt),
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"custom_domain",
					// The backend won't let us modify a custom domain for
					// a few minutes, so we're not testing a new value here.
					fmt.Sprintf("status-%d.example.com", rInt),
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"logo",
					"https://example.com/logo2.png",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"favicon",
					"https://example.com/favicon2.ico",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"link",
					"https://example2.com",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"description",
					"Updated test dashboard description",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"header",
					"Updated System Status",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"width",
					"FULL",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"refresh_rate",
					"600",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"paginate",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"checks_per_page",
					"10",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"pagination_rate",
					"300",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"hide_tags",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"use_tags_and_operator",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"enable_incidents",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"tags.#",
					"1",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"show_header",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"expand_checks",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"show_check_run_links",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"custom_css",
					"body { background: #f0f0f0; }",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"show_p95",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"show_p99",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"sort_by_status",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"show_group_names",
					"true",
				),
			),
		},
	})
}

func TestAccDashboardPrivateDashboard(t *testing.T) {
	rInt := acctest.RandInt()
	// The key from creation must survive the in-place update while private.
	sameKey := statecheck.CompareValue(compare.ValuesSame())
	accTestCase(t, []resource.TestStep{
		{
			Config: fmt.Sprintf(`
				resource "checkly_dashboard" "test" {
					custom_url = "test-dashboard-%d-private"
					header     = "Private Dashboard"
					is_private = true
				}
			`, rInt),
			ConfigStateChecks: []statecheck.StateCheck{
				sameKey.AddStateValue("checkly_dashboard.test", tfjsonpath.New("key")),
			},
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"custom_url",
					fmt.Sprintf("test-dashboard-%d-private", rInt),
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"is_private",
					"true",
				),
				resource.TestCheckResourceAttrSet(
					"checkly_dashboard.test",
					"key",
				),
			),
		},
		{
			ResourceName:      "checkly_dashboard.test",
			ImportState:       true,
			ImportStateVerify: true,
			// The API returns a dashboard's key only in the create
			// response, so an imported dashboard has no key.
			ImportStateVerifyIgnore: []string{"key"},
		},
		{
			// An unrelated in-place update keeps the key of a private
			// dashboard, and the plan already knows it.
			Config: fmt.Sprintf(`
				resource "checkly_dashboard" "test" {
					custom_url = "test-dashboard-%d-private"
					header     = "Private Dashboard updated"
					is_private = true
				}
			`, rInt),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction("checkly_dashboard.test", plancheck.ResourceActionUpdate),
					plancheck.ExpectKnownValue("checkly_dashboard.test", tfjsonpath.New("key"), knownvalue.NotNull()),
				},
			},
			ConfigStateChecks: []statecheck.StateCheck{
				sameKey.AddStateValue("checkly_dashboard.test", tfjsonpath.New("key")),
			},
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"header",
					"Private Dashboard updated",
				),
				resource.TestCheckResourceAttrSet(
					"checkly_dashboard.test",
					"key",
				),
			),
		},
		{
			Config: fmt.Sprintf(`
				resource "checkly_dashboard" "test" {
					custom_url = "test-dashboard-%d-private"
					header     = "Private Dashboard"
					is_private = false
				}
			`, rInt),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectKnownValue("checkly_dashboard.test", tfjsonpath.New("key"), knownvalue.Null()),
				},
			},
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"is_private",
					"false",
				),
				// A public dashboard has no access key. In the framework the
				// attribute is null rather than an empty string.
				resource.TestCheckNoResourceAttr(
					"checkly_dashboard.test",
					"key",
				),
			),
		},
		{
			// The update that made the dashboard public deleted its keys on
			// the backend, and update responses never return a key, so the
			// dashboard stays without one after it is made private again.
			Config: fmt.Sprintf(`
				resource "checkly_dashboard" "test" {
					custom_url = "test-dashboard-%d-private"
					header     = "Private Dashboard"
					is_private = true
				}
			`, rInt),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectKnownValue("checkly_dashboard.test", tfjsonpath.New("key"), knownvalue.Null()),
				},
			},
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"is_private",
					"true",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_dashboard.test",
					"key",
				),
			),
		},
	})
}

func TestAccDashboardInvalidWidth(t *testing.T) {
	rInt := acctest.RandInt()
	accTestCase(t, []resource.TestStep{
		{
			Config: fmt.Sprintf(`
				resource "checkly_dashboard" "test" {
					custom_url = "test-dashboard-%d-invalid-width"
					header     = "Invalid Width Test"
					width      = "500PX"
				}
			`, rInt),
			ExpectError: regexp.MustCompile(`value must be one of: \["FULL" "960PX"\]`),
		},
	})
}

func TestAccDashboardInvalidRefreshRate(t *testing.T) {
	rInt := acctest.RandInt()
	accTestCase(t, []resource.TestStep{
		{
			Config: fmt.Sprintf(`
				resource "checkly_dashboard" "test" {
					custom_url   = "test-dashboard-%d-invalid-refresh"
					header       = "Invalid Refresh Test"
					refresh_rate = 120
				}
			`, rInt),
			ExpectError: regexp.MustCompile(`value must be one of: \["60" "300" "600"\]`),
		},
	})
}

func TestAccDashboardInvalidPaginationRate(t *testing.T) {
	rInt := acctest.RandInt()
	accTestCase(t, []resource.TestStep{
		{
			Config: fmt.Sprintf(`
				resource "checkly_dashboard" "test" {
					custom_url      = "test-dashboard-%d-invalid-pagination"
					header          = "Invalid Pagination Test"
					pagination_rate = 90
				}
			`, rInt),
			ExpectError: regexp.MustCompile(`value must be one of: \["30" "60" "300"\]`),
		},
	})
}

func TestAccDashboardInvalidChecksPerPage(t *testing.T) {
	rInt := acctest.RandInt()
	accTestCase(t, []resource.TestStep{
		{
			Config: fmt.Sprintf(`
				resource "checkly_dashboard" "test" {
					custom_url      = "test-dashboard-%d-invalid-checks-per-page"
					header          = "Invalid Checks Per Page Test"
					checks_per_page = 25
				}
			`, rInt),
			ExpectError: regexp.MustCompile(`value must be between 1 and 20`),
		},
	})
}

func TestAccDashboardWithCheckGroup(t *testing.T) {
	rInt := acctest.RandInt()
	accTestCase(t, []resource.TestStep{
		{
			Config: fmt.Sprintf(`
				resource "checkly_check_group" "test" {
					name        = "Test Group for Dashboard"
					activated   = true
					tags        = ["api", "production"]
					locations   = ["us-east-1"]
					concurrency = 3
				}

				resource "checkly_dashboard" "test" {
					custom_url = "test-dashboard-%d-with-tags"
					header     = "Dashboard with Tags"
					tags       = ["api", "production"]

					use_tags_and_operator = true
				}
			`, rInt),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"custom_url",
					fmt.Sprintf("test-dashboard-%d-with-tags", rInt),
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"tags.#",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"use_tags_and_operator",
					"true",
				),
			),
		},
	})
}

// TestAccDashboardUpdateClearsRemovedValues removes values that the API keeps
// when an update leaves them out. The empty plan after the last step shows
// that they are gone.
func TestAccDashboardUpdateClearsRemovedValues(t *testing.T) {
	rInt := acctest.RandInt()
	config := func(values string) string {
		return fmt.Sprintf(`
			resource "checkly_dashboard" "test" {
				custom_url = "test-dashboard-%d-clear"
				header     = "System Status"
				%s
			}
		`, rInt, values)
	}
	const r = "checkly_dashboard.test"

	accTestCase(t, []resource.TestStep{
		{
			Config: config(fmt.Sprintf(`
				custom_domain = "status-%d-clear.example.com"
				logo          = "https://example.com/logo.png"
				favicon       = "https://example.com/favicon.ico"
				link          = "https://example.com"
				description   = "Test dashboard description"
			`, rInt)),
			Check: resource.TestCheckResourceAttr(r, "logo", "https://example.com/logo.png"),
		},
		{
			Config: config(""),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(r, "custom_domain"),
				resource.TestCheckResourceAttr(r, "logo", ""),
				resource.TestCheckResourceAttr(r, "favicon", ""),
				resource.TestCheckResourceAttr(r, "link", ""),
				resource.TestCheckResourceAttr(r, "description", ""),
			),
		},
	})
}

func TestAccDashboardCustomCSS(t *testing.T) {
	rInt := acctest.RandInt()
	accTestCase(t, []resource.TestStep{
		{
			Config: fmt.Sprintf(`
				resource "checkly_dashboard" "test" {
					custom_url = "test-dashboard-%d-custom-css"
					header     = "Dashboard with Custom CSS"
					custom_css = <<-EOT
						body {
							background-color: #f5f5f5;
							font-family: Arial, sans-serif;
						}
						.header {
							color: #333;
							font-size: 24px;
							margin-bottom: 20px;
						}
						.check-item {
							border: 1px solid #ddd;
							padding: 10px;
							margin: 5px 0;
						}
					EOT
				}
			`, rInt),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"custom_url",
					fmt.Sprintf("test-dashboard-%d-custom-css", rInt),
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"custom_css",
					`body {
	background-color: #f5f5f5;
	font-family: Arial, sans-serif;
}
.header {
	color: #333;
	font-size: 24px;
	margin-bottom: 20px;
}
.check-item {
	border: 1px solid #ddd;
	padding: 10px;
	margin: 5px 0;
}
`,
				),
			),
		},
		{
			Config: fmt.Sprintf(`
				resource "checkly_dashboard" "test" {
					custom_url = "test-dashboard-%d-custom-css"
					header     = "Dashboard with Custom CSS"
					custom_css = ""
				}
			`, rInt),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"custom_css",
					"",
				),
			),
		},
	})
}

func TestDashboardAttributeValidation(t *testing.T) {
	r := NewDashboardResource()
	cases := []struct {
		attribute string
		value     string
		valid     bool
	}{
		{"custom_url", "status", true},
		{"custom_url", "My-Status-1", true},
		{"custom_url", "", true},
		{"custom_url", "-status", false},
		{"custom_url", "status-", false},
		{"custom_url", "my_status", false},
		{"custom_url", "status.mycompany", false},
		{"custom_url", "a" + strings.Repeat("b", 62), true},
		{"custom_url", "a" + strings.Repeat("b", 63), false},
		{"logo", "", true},
		{"logo", "https://example.com/logo.png", true},
		{"logo", "logo.png", false},
		{"favicon", "ftp://example.com/favicon.ico", false},
		{"link", "http://example.com", true},
		{"link", "example.com", false},
	}
	for _, c := range cases {
		diags := validateStringAttribute(t, r, c.attribute, types.StringValue(c.value))
		if diags.HasError() == c.valid {
			t.Errorf("%s = %q: valid = %v, diagnostics: %v", c.attribute, c.value, c.valid, diags)
		}
	}
}

// sort_by_status and show_group_names are sent on every request, and the
// API's defaults apply when the response leaves them out.
func TestDashboardSortAndGroupNames(t *testing.T) {
	ctx := context.Background()
	model := DashboardResourceModel{
		SortByStatus:   types.BoolValue(true),
		ShowGroupNames: types.BoolValue(false),
		Tags:           types.SetValueMust(types.StringType, nil),
	}
	var diags diag.Diagnostics
	dashboard := dashboardFromModel(ctx, &model, &diags)
	if !dashboard.SortByStatus || dashboard.ShowGroupNames == nil || *dashboard.ShowGroupNames {
		t.Errorf("SortByStatus = %v, ShowGroupNames = %v", dashboard.SortByStatus, dashboard.ShowGroupNames)
	}

	dashboardToModel(ctx, &checkly.Dashboard{}, &model, &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if !model.SortByStatus.Equal(types.BoolValue(false)) || !model.ShowGroupNames.Equal(types.BoolValue(true)) {
		t.Errorf("sort_by_status = %v, show_group_names = %v", model.SortByStatus, model.ShowGroupNames)
	}
}

func TestDashboardKeyPlanModifier(t *testing.T) {
	ctx := context.Background()

	var schemaResp fwresource.SchemaResponse
	NewDashboardResource().Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	keyAttr := schemaResp.Schema.Attributes["key"].(schema.StringAttribute)
	objectType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)

	// dashboardObject returns a dashboard object whose attributes are all null
	// except is_private.
	dashboardObject := func(isPrivate tftypes.Value) tftypes.Value {
		attrs := make(map[string]tftypes.Value, len(objectType.AttributeTypes))
		for name, typ := range objectType.AttributeTypes {
			attrs[name] = tftypes.NewValue(typ, nil)
		}
		attrs["is_private"] = isPrivate
		return tftypes.NewValue(objectType, attrs)
	}
	private := tftypes.NewValue(tftypes.Bool, true)
	public := tftypes.NewValue(tftypes.Bool, false)
	unknown := tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue)

	cases := []struct {
		name      string
		state     tftypes.Value
		isPrivate tftypes.Value
		stateKey  types.String
		want      types.String
	}{
		{"create", tftypes.NewValue(objectType, nil), private, types.StringNull(), types.StringUnknown()},
		{"planned public", dashboardObject(private), public, types.StringValue("k"), types.StringNull()},
		{"planned private with a key", dashboardObject(private), private, types.StringValue("k"), types.StringValue("k")},
		{"planned private without a key", dashboardObject(public), private, types.StringNull(), types.StringNull()},
		{"is_private unknown", dashboardObject(private), unknown, types.StringValue("k"), types.StringUnknown()},
	}

	for _, c := range cases {
		req := planmodifier.StringRequest{
			Path:       path.Root("key"),
			StateValue: c.stateKey,
			PlanValue:  types.StringUnknown(),
			State:      tfsdk.State{Schema: schemaResp.Schema, Raw: c.state},
			Plan:       tfsdk.Plan{Schema: schemaResp.Schema, Raw: dashboardObject(c.isPrivate)},
		}
		resp := planmodifier.StringResponse{PlanValue: req.PlanValue}
		for _, m := range keyAttr.PlanModifiers {
			m.PlanModifyString(ctx, req, &resp)
		}
		if resp.Diagnostics.HasError() {
			t.Errorf("%s: unexpected diagnostics: %v", c.name, resp.Diagnostics)
			continue
		}
		if !resp.PlanValue.Equal(c.want) {
			t.Errorf("%s: planned key = %s, want %s", c.name, resp.PlanValue, c.want)
		}
	}
}
