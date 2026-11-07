package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/checkly/checkly-go-sdk/v2"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// failingReadClient fails the reads that an update starts with. An update
// that went ahead would panic through the nil embedded Client.
type failingReadClient struct {
	checkly.Client
}

var errFailingRead = errors.New("502 Bad Gateway")

func (failingReadClient) GetAPICheck(context.Context, string) (*checkly.APICheck, error) {
	return nil, errFailingRead
}

func (failingReadClient) GetBrowserCheck(context.Context, string) (*checkly.BrowserCheck, error) {
	return nil, errFailingRead
}

func (failingReadClient) GetMultiStepCheck(context.Context, string) (*checkly.MultiStepCheck, error) {
	return nil, errFailingRead
}

func (failingReadClient) GetPlaywrightCheck(context.Context, string) (*checkly.PlaywrightCheck, error) {
	return nil, errFailingRead
}

func (failingReadClient) GetDNSMonitor(context.Context, string) (*checkly.DNSMonitor, error) {
	return nil, errFailingRead
}

func (failingReadClient) GetICMPMonitor(context.Context, string) (*checkly.ICMPMonitor, error) {
	return nil, errFailingRead
}

func (failingReadClient) GetTracerouteMonitor(context.Context, string) (*checkly.TracerouteMonitor, error) {
	return nil, errFailingRead
}

func (failingReadClient) GetPrivateLocation(context.Context, string) (*checkly.PrivateLocation, error) {
	return nil, errFailingRead
}

// An update that can't read what it has to keep stops without writing.
func TestUpdateSendsNoWriteWhenTheReadFails(t *testing.T) {
	ctx := context.Background()

	cases := map[string]resource.Resource{
		"checkly_api_check":              &APICheckResource{client: failingReadClient{}},
		"checkly_browser_check":          &BrowserCheckResource{client: failingReadClient{}},
		"checkly_multistep_check":        &MultistepCheckResource{client: failingReadClient{}},
		"checkly_playwright_check_suite": &PlaywrightCheckSuiteResource{client: failingReadClient{}},
		"checkly_dns_monitor":            &DNSMonitorResource{client: failingReadClient{}},
		"checkly_icmp_monitor":           &ICMPMonitorResource{client: failingReadClient{}},
		"checkly_traceroute_monitor":     &TracerouteMonitorResource{client: failingReadClient{}},
		"checkly_private_location":       &PrivateLocationResource{client: failingReadClient{}},
	}

	for name, r := range cases {
		t.Run(name, func(t *testing.T) {
			var schemaResp resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
			s := schemaResp.Schema

			plan := tfsdk.Plan{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
			if diags := plan.SetAttribute(ctx, path.Root("id"), "abc"); diags.HasError() {
				t.Fatal(diags)
			}

			resp := resource.UpdateResponse{State: tfsdk.State{Schema: s, Raw: plan.Raw}}
			r.Update(ctx, resource.UpdateRequest{Plan: plan}, &resp)

			if !resp.Diagnostics.HasError() {
				t.Fatal("expected an error diagnostic")
			}
			got := resp.Diagnostics.Errors()[0]
			if !strings.HasPrefix(got.Summary(), "Unable to Update") || !strings.Contains(got.Detail(), errFailingRead.Error()) {
				t.Errorf("diagnostic = %q: %q", got.Summary(), got.Detail())
			}
		})
	}
}

// keepsUnmanagedCase sets a value that the resource has no attribute for
// outside Terraform, and checks it after a Terraform update.
type keepsUnmanagedCase struct {
	resource string
	config   func(name string) string
	set      func(ctx context.Context, client checkly.Client, id string) error
	check    func(ctx context.Context, client checkly.Client, id string) error
}

func testAccUpdateKeepsUnmanaged(t *testing.T, c keepsUnmanagedCase) {
	t.Helper()

	name := fmt.Sprintf("tf-acc-keep-%d", acctest.RandInt())

	var id string
	accTestCase(t, []tfresource.TestStep{
		{
			Config: c.config(name),
			Check: func(s *terraform.State) error {
				id = s.RootModule().Resources[c.resource].Primary.ID
				return nil
			},
		},
		{
			PreConfig: func() {
				if err := c.set(context.Background(), testAccClient(), id); err != nil {
					t.Fatal(err)
				}
			},
			Config: c.config(name + "-renamed"),
			Check: func(*terraform.State) error {
				return c.check(context.Background(), testAccClient(), id)
			},
		},
	})
}

const keptIntentGoal = "tf-acc: the page keeps loading"

var keptIntent = &checkly.CheckIntent{Goal: keptIntentGoal}

func checkKeptIntent(intent *checkly.CheckIntent, aiAutoRepairEnabled *bool) error {
	if intent == nil || intent.Goal != keptIntentGoal {
		return fmt.Errorf("intent = %+v, want the goal %q", intent, keptIntentGoal)
	}
	if aiAutoRepairEnabled == nil || *aiAutoRepairEnabled {
		return fmt.Errorf("aiAutoRepairEnabled = %v, want false", aiAutoRepairEnabled)
	}
	return nil
}

func TestAccAPICheckUpdateKeepsUnmanagedValues(t *testing.T) {
	testAccUpdateKeepsUnmanaged(t, keepsUnmanagedCase{
		resource: "checkly_api_check.test",
		config: func(name string) string {
			return fmt.Sprintf(`
				resource "checkly_api_check" "test" {
					name      = %q
					activated = false
					locations = ["eu-central-1"]
					request = {
						url = "https://api.checklyhq.com/public-stats"
					}
				}
			`, name)
		},
		set: func(ctx context.Context, client checkly.Client, id string) error {
			check, err := client.GetAPICheck(ctx, id)
			if err != nil {
				return err
			}
			aiAutoRepair := false
			check.Intent, check.AIAutoRepairEnabled = keptIntent, &aiAutoRepair
			_, err = client.UpdateAPICheck(ctx, id, *check)
			return err
		},
		check: func(ctx context.Context, client checkly.Client, id string) error {
			check, err := client.GetAPICheck(ctx, id)
			if err != nil {
				return err
			}
			return checkKeptIntent(check.Intent, check.AIAutoRepairEnabled)
		},
	})
}

func TestAccBrowserCheckUpdateKeepsUnmanagedValues(t *testing.T) {
	testAccUpdateKeepsUnmanaged(t, keepsUnmanagedCase{
		resource: "checkly_browser_check.test",
		config: func(name string) string {
			return fmt.Sprintf(`
				resource "checkly_browser_check" "test" {
					name      = %q
					activated = false
					locations = ["eu-central-1"]
					script    = "console.log('test')"
				}
			`, name)
		},
		set: func(ctx context.Context, client checkly.Client, id string) error {
			check, err := client.GetBrowserCheck(ctx, id)
			if err != nil {
				return err
			}
			aiAutoRepair := false
			check.Intent, check.AIAutoRepairEnabled = keptIntent, &aiAutoRepair
			_, err = client.UpdateBrowserCheck(ctx, id, *check)
			return err
		},
		check: func(ctx context.Context, client checkly.Client, id string) error {
			check, err := client.GetBrowserCheck(ctx, id)
			if err != nil {
				return err
			}
			return checkKeptIntent(check.Intent, check.AIAutoRepairEnabled)
		},
	})
}

func TestAccMultistepCheckUpdateKeepsUnmanagedValues(t *testing.T) {
	testAccUpdateKeepsUnmanaged(t, keepsUnmanagedCase{
		resource: "checkly_multistep_check.test",
		config: func(name string) string {
			return fmt.Sprintf(`
				resource "checkly_multistep_check" "test" {
					name      = %q
					activated = false
					locations = ["eu-central-1"]
					script    = "console.log('test')"
				}
			`, name)
		},
		set: func(ctx context.Context, client checkly.Client, id string) error {
			check, err := client.GetMultiStepCheck(ctx, id)
			if err != nil {
				return err
			}
			aiAutoRepair := false
			check.Intent, check.AIAutoRepairEnabled = keptIntent, &aiAutoRepair
			_, err = client.UpdateMultiStepCheck(ctx, id, *check)
			return err
		},
		check: func(ctx context.Context, client checkly.Client, id string) error {
			check, err := client.GetMultiStepCheck(ctx, id)
			if err != nil {
				return err
			}
			return checkKeptIntent(check.Intent, check.AIAutoRepairEnabled)
		},
	})
}

func TestAccPlaywrightCheckSuiteUpdateKeepsUnmanagedValues(t *testing.T) {
	testAccUpdateKeepsUnmanaged(t, keepsUnmanagedCase{
		resource: "checkly_playwright_check_suite.test",
		config: func(name string) string {
			return playwrightCheckSuiteBase + fmt.Sprintf(`
				resource "checkly_playwright_check_suite" "test" {
					name                      = %q
					activated                 = false
					frequency                 = 720
					use_global_alert_settings = true
					locations                 = ["us-east-1"]

					bundle = {
						id       = checkly_playwright_code_bundle.test.id
						metadata = checkly_playwright_code_bundle.test.metadata
					}

					runtime = {
						steps = {
							install = {
								command = "pnpm i"
							}

							test = {
								command = "pnpm playwright test"
							}
						}

						playwright = {
							version = "1.56.1"
							device = [
								{
									type = "chromium"
								},
							]
						}
					}
				}
			`, name)
		},
		set: func(ctx context.Context, client checkly.Client, id string) error {
			check, err := client.GetPlaywrightCheck(ctx, id)
			if err != nil {
				return err
			}
			aiAutoRepair := false
			check.Intent, check.AIAutoRepairEnabled = keptIntent, &aiAutoRepair
			_, err = client.UpdatePlaywrightCheck(ctx, id, *check)
			return err
		},
		check: func(ctx context.Context, client checkly.Client, id string) error {
			check, err := client.GetPlaywrightCheck(ctx, id)
			if err != nil {
				return err
			}
			return checkKeptIntent(check.Intent, check.AIAutoRepairEnabled)
		},
	})
}

// privateLocationConfig declares the private location that a monitor is
// assigned to outside Terraform. Monitors depend on it, so that they are
// destroyed first.
func privateLocationConfig(slug string) string {
	return fmt.Sprintf(`
		resource "checkly_private_location" "keep" {
			name      = %q
			slug_name = %q
		}
	`, slug, slug)
}

func keptPrivateLocationSlug() string {
	return fmt.Sprintf("tf-keep-%d", acctest.RandInt()%10000000)
}

func checkKeptPrivateLocation(privateLocations *[]string, slug string) error {
	if privateLocations == nil || len(*privateLocations) != 1 || (*privateLocations)[0] != slug {
		return fmt.Errorf("privateLocations = %v, want [%s]", privateLocations, slug)
	}
	return nil
}

func TestAccDNSMonitorUpdateKeepsPrivateLocations(t *testing.T) {
	slug := keptPrivateLocationSlug()
	testAccUpdateKeepsUnmanaged(t, keepsUnmanagedCase{
		resource: "checkly_dns_monitor.test",
		config: func(name string) string {
			return privateLocationConfig(slug) + fmt.Sprintf(`
				resource "checkly_dns_monitor" "test" {
					name                      = %q
					frequency                 = 60
					activated                 = false
					locations                 = ["eu-central-1"]
					use_global_alert_settings = true
					depends_on                = [checkly_private_location.keep]

					request = {
						record_type = "A"
						query       = "welcome.checklyhq.com"
					}
				}
			`, name)
		},
		set: func(ctx context.Context, client checkly.Client, id string) error {
			monitor, err := client.GetDNSMonitor(ctx, id)
			if err != nil {
				return err
			}
			monitor.PrivateLocations = &[]string{slug}
			_, err = client.UpdateDNSMonitor(ctx, id, *monitor)
			return err
		},
		check: func(ctx context.Context, client checkly.Client, id string) error {
			monitor, err := client.GetDNSMonitor(ctx, id)
			if err != nil {
				return err
			}
			return checkKeptPrivateLocation(monitor.PrivateLocations, slug)
		},
	})
}

func TestAccICMPMonitorUpdateKeepsPrivateLocations(t *testing.T) {
	slug := keptPrivateLocationSlug()
	testAccUpdateKeepsUnmanaged(t, keepsUnmanagedCase{
		resource: "checkly_icmp_monitor.test",
		config: func(name string) string {
			return privateLocationConfig(slug) + fmt.Sprintf(`
				resource "checkly_icmp_monitor" "test" {
					name                      = %q
					frequency                 = 60
					activated                 = false
					locations                 = ["eu-central-1"]
					use_global_alert_settings = true
					depends_on                = [checkly_private_location.keep]

					request = {
						hostname = "example.com"
					}
				}
			`, name)
		},
		set: func(ctx context.Context, client checkly.Client, id string) error {
			monitor, err := client.GetICMPMonitor(ctx, id)
			if err != nil {
				return err
			}
			monitor.PrivateLocations = &[]string{slug}
			_, err = client.UpdateICMPMonitor(ctx, id, *monitor)
			return err
		},
		check: func(ctx context.Context, client checkly.Client, id string) error {
			monitor, err := client.GetICMPMonitor(ctx, id)
			if err != nil {
				return err
			}
			return checkKeptPrivateLocation(monitor.PrivateLocations, slug)
		},
	})
}

func TestAccTracerouteMonitorUpdateKeepsPrivateLocations(t *testing.T) {
	slug := keptPrivateLocationSlug()
	testAccUpdateKeepsUnmanaged(t, keepsUnmanagedCase{
		resource: "checkly_traceroute_monitor.test",
		config: func(name string) string {
			return privateLocationConfig(slug) + fmt.Sprintf(`
				resource "checkly_traceroute_monitor" "test" {
					name                      = %q
					frequency                 = 60
					activated                 = false
					locations                 = ["eu-central-1"]
					use_global_alert_settings = true
					depends_on                = [checkly_private_location.keep]

					request = {
						url = "example.com"
					}
				}
			`, name)
		},
		set: func(ctx context.Context, client checkly.Client, id string) error {
			monitor, err := client.GetTracerouteMonitor(ctx, id)
			if err != nil {
				return err
			}
			monitor.PrivateLocations = &[]string{slug}
			_, err = client.UpdateTracerouteMonitor(ctx, id, *monitor)
			return err
		},
		check: func(ctx context.Context, client checkly.Client, id string) error {
			monitor, err := client.GetTracerouteMonitor(ctx, id)
			if err != nil {
				return err
			}
			return checkKeptPrivateLocation(monitor.PrivateLocations, slug)
		},
	})
}

func TestAccPrivateLocationUpdateKeepsProxyURL(t *testing.T) {
	slug := keptPrivateLocationSlug()
	const proxyURL = "https://proxy.example.com:8080"

	testAccUpdateKeepsUnmanaged(t, keepsUnmanagedCase{
		resource: "checkly_private_location.test",
		config: func(name string) string {
			return fmt.Sprintf(`
				resource "checkly_private_location" "test" {
					name      = %q
					slug_name = %q
				}
			`, name, slug)
		},
		set: func(ctx context.Context, client checkly.Client, id string) error {
			location, err := client.GetPrivateLocation(ctx, id)
			if err != nil {
				return err
			}
			url := proxyURL
			location.ProxyURL = &url
			_, err = client.UpdatePrivateLocation(ctx, id, *location)
			return err
		},
		check: func(ctx context.Context, client checkly.Client, id string) error {
			location, err := client.GetPrivateLocation(ctx, id)
			if err != nil {
				return err
			}
			if location.ProxyURL == nil || *location.ProxyURL != proxyURL {
				return fmt.Errorf("proxyUrl = %v, want %s", location.ProxyURL, proxyURL)
			}
			return nil
		},
	})
}
