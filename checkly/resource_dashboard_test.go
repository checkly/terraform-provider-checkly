package checkly

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
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
			),
		},
	})
}

func TestAccDashboardPrivateDashboard(t *testing.T) {
	rInt := acctest.RandInt()
	accTestCase(t, []resource.TestStep{
		{
			Config: fmt.Sprintf(`
				resource "checkly_dashboard" "test" {
					custom_url = "test-dashboard-%d-private"
					header     = "Private Dashboard"
					is_private = true
				}
			`, rInt),
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
			Config: fmt.Sprintf(`
				resource "checkly_dashboard" "test" {
					custom_url = "test-dashboard-%d-private"
					header     = "Private Dashboard"
					is_private = false
				}
			`, rInt),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"is_private",
					"false",
				),
				resource.TestCheckResourceAttr(
					"checkly_dashboard.test",
					"key",
					"",
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
			ExpectError: regexp.MustCompile(`"width" must be one of \[FULL 960PX\], got: 500PX`),
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
			ExpectError: regexp.MustCompile(`"refresh_rate" must be one of \[60 300 600\], got: 120`),
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
			ExpectError: regexp.MustCompile(`"pagination_rate" must be one of \[30 60 300\], got: 90`),
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
			ExpectError: regexp.MustCompile(`"checks_per_page" must be between 1 and 20, got: 25`),
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
