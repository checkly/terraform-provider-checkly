package provider

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/checkly/checkly-go-sdk/v2"
)

func TestAccPrivateLocationCheckRequiredFields(t *testing.T) {
	config := `resource "checkly_private_location" "test" {}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "name" is required`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "slug_name" is required`),
		},
	})
}

func TestAccPrivateLocationInvalidSlug(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `resource "checkly_private_location" "test" {
				name      = "Invalid slug"
				slug_name = "eu-central-1-lab"
			}`,
			PlanOnly:    true,
			ExpectError: regexp.MustCompile(`must\s+not\s+start\s+with\s+the\s+ID\s+of\s+a\s+public\s+location`),
		},
	})
}

func TestAccPrivateLocationSuccess(t *testing.T) {
	rInt := acctest.RandInt()
	// Slug names are limited to 30 alphanumeric characters. We use modulo to
	// keep the random suffix short while still avoiding collisions in parallel runs.
	slug := fmt.Sprintf("tf-test-pl-s-%d", rInt%100000)
	accTestCase(t, []resource.TestStep{
		{
			Config: fmt.Sprintf(`
				resource "checkly_private_location" "test" {
					name      = "Private Location %d"
					slug_name = "%s"
					icon      = "bell-fill"
				}
			`, rInt, slug),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_private_location.test",
					"name",
					fmt.Sprintf("Private Location %d", rInt),
				),
				resource.TestCheckResourceAttr(
					"checkly_private_location.test",
					"slug_name",
					slug,
				),
				resource.TestCheckResourceAttr(
					"checkly_private_location.test",
					"icon",
					"bell-fill",
				),
				resource.TestCheckResourceAttr(
					"checkly_private_location.test",
					"keys.#",
					"1",
				),
			),
		},
		{
			ResourceName:      "checkly_private_location.test",
			ImportState:       true,
			ImportStateVerify: true,
			// The API returns raw keys only in the create response; reads return
			// masked keys, so an imported private location has none.
			ImportStateVerifyIgnore: []string{"keys"},
		},
	})
}

func TestAccPrivateLocationDefaultIcon(t *testing.T) {
	rInt := acctest.RandInt()
	// Slug names are limited to 30 alphanumeric characters. We use modulo to
	// keep the random suffix short while still avoiding collisions in parallel runs.
	slug := fmt.Sprintf("tf-test-pl-d-%d", rInt%100000)
	accTestCase(t, []resource.TestStep{
		{
			Config: fmt.Sprintf(`
				resource "checkly_private_location" "without_icon" {
					name      = "Private Location %d"
					slug_name = "%s"
				}
			`, rInt, slug),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_private_location.without_icon",
					"name",
					fmt.Sprintf("Private Location %d", rInt),
				),
				resource.TestCheckResourceAttr(
					"checkly_private_location.without_icon",
					"slug_name",
					slug,
				),
				resource.TestCheckResourceAttr(
					"checkly_private_location.without_icon",
					"icon",
					"location",
				),
			),
		},
	})
}

// An empty proxy_url removes the proxy like null does, and stays empty on
// refresh instead of turning into null.
func TestPrivateLocationProxyURLRoundTrip(t *testing.T) {
	t.Parallel()

	const proxyURL = "http://proxy.example.com:3128"

	tests := []struct {
		name      string
		value     types.String
		wantSent  *string
		fromAPI   *string
		wantState types.String
	}{
		{"set", types.StringValue(proxyURL), new(proxyURL), new(proxyURL), types.StringValue(proxyURL)},
		{"empty", types.StringValue(""), new(""), nil, types.StringValue("")},
		{"null", types.StringNull(), nil, nil, types.StringNull()},
		{"removed outside Terraform", types.StringValue(proxyURL), new(proxyURL), nil, types.StringNull()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := PrivateLocationResourceModel{ProxyURL: tt.value}

			sent := privateLocationFromModel(&model).ProxyURL
			if (sent == nil) != (tt.wantSent == nil) || (sent != nil && *sent != *tt.wantSent) {
				t.Errorf("sent proxyUrl = %v, want %v", sent, tt.wantSent)
			}

			privateLocationToModel(&checkly.PrivateLocation{ProxyURL: tt.fromAPI}, &model)
			if !model.ProxyURL.Equal(tt.wantState) {
				t.Errorf("proxy_url = %s, want %s", model.ProxyURL, tt.wantState)
			}
		})
	}
}

func TestAccPrivateLocationProxyURL(t *testing.T) {
	const resourceName = "checkly_private_location.test"
	const proxyURL = "http://proxy.example.com:3128"
	const changedProxyURL = "https://proxy.example.com:8080"

	rInt := acctest.RandInt()
	// Slug names are limited to 30 alphanumeric characters. We use modulo to
	// keep the random suffix short while still avoiding collisions in parallel runs.
	slug := fmt.Sprintf("tf-test-pl-p-%d", rInt%100000)
	config := func(attributes string) string {
		return fmt.Sprintf(`
			resource "checkly_private_location" "test" {
				name      = "Private Location %d"
				slug_name = %q
				%s
			}
		`, rInt, slug, attributes)
	}

	var id string
	captureID := func(s *terraform.State) error {
		id = s.RootModule().Resources[resourceName].Primary.ID
		return nil
	}
	stored := func(want *string) resource.TestCheckFunc {
		return func(*terraform.State) error {
			location, err := testAccClient().GetPrivateLocation(context.Background(), id)
			if err != nil {
				return err
			}
			got := location.ProxyURL
			// The API reports no proxy as null.
			if (got == nil) != (want == nil) || (got != nil && *got != *want) {
				return fmt.Errorf("proxyUrl = %v, want %v", got, want)
			}
			return nil
		}
	}

	accTestCase(t, []resource.TestStep{
		{
			Config: config(fmt.Sprintf("proxy_url = %q", proxyURL)),
			Check: resource.ComposeTestCheckFunc(
				captureID,
				resource.TestCheckResourceAttr(resourceName, "proxy_url", proxyURL),
				stored(new(proxyURL)),
			),
		},
		{
			Config: config(fmt.Sprintf("proxy_url = %q", changedProxyURL)),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(resourceName, "proxy_url", changedProxyURL),
				stored(new(changedProxyURL)),
			),
		},
		{
			ResourceName:      resourceName,
			ImportState:       true,
			ImportStateVerify: true,
			// The API returns raw keys only in the create response; reads return
			// masked keys, so an imported private location has none.
			ImportStateVerifyIgnore: []string{"keys"},
		},
		{
			// An empty proxy_url removes the proxy and stays empty in the state.
			Config: config(`proxy_url = ""`),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(resourceName, "proxy_url", ""),
				stored(nil),
			),
		},
		{
			Config: config(fmt.Sprintf("proxy_url = %q", proxyURL)),
			Check:  stored(new(proxyURL)),
		},
		{
			Config: config(""),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(resourceName, "proxy_url"),
				stored(nil),
			),
		},
		{
			// A proxy set outside Terraform shows up as a removal.
			PreConfig: func() {
				location, err := testAccClient().GetPrivateLocation(context.Background(), id)
				if err != nil {
					t.Fatal(err)
				}
				location.ProxyURL = new(proxyURL)
				if _, err := testAccClient().UpdatePrivateLocation(context.Background(), id, *location); err != nil {
					t.Fatal(err)
				}
			},
			Config: config(""),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
				},
			},
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(resourceName, "proxy_url"),
				stored(nil),
			),
		},
	})
}

// monitorPrivateLocationsCase runs the private_locations attribute of one
// monitor resource against the API.
type monitorPrivateLocationsCase struct {
	resource string
	// config returns the configuration of a monitor with the given name and
	// further attributes.
	config func(name, attributes string) string
	set    func(ctx context.Context, client checkly.Client, id string, privateLocations []string) error
	read   func(ctx context.Context, client checkly.Client, id string) (*[]string, error)
}

func testAccMonitorPrivateLocations(t *testing.T, c monitorPrivateLocationsCase) {
	t.Helper()

	rInt := acctest.RandInt()
	name := fmt.Sprintf("tf-acc-private-locations-%d", rInt)
	// Slug names are limited to 30 alphanumeric characters. We use modulo to
	// keep the random suffix short while still avoiding collisions in parallel runs.
	slugA := fmt.Sprintf("tf-test-pl-a-%d", rInt%100000)
	slugB := fmt.Sprintf("tf-test-pl-b-%d", rInt%100000)

	// The API refuses to delete a private location that a monitor still
	// runs from, so the monitor depends on both locations even when it
	// references neither, and is destroyed first.
	config := func(attributes string) string {
		return fmt.Sprintf(`
			resource "checkly_private_location" "a" {
				name      = %[1]q
				slug_name = %[1]q
			}

			resource "checkly_private_location" "b" {
				name      = %[2]q
				slug_name = %[2]q
			}
		`, slugA, slugB) + c.config(name, attributes+`
			depends_on = [checkly_private_location.a, checkly_private_location.b]
		`)
	}

	var id string
	captureID := func(s *terraform.State) error {
		id = s.RootModule().Resources[c.resource].Primary.ID
		return nil
	}
	stored := func(want ...string) resource.TestCheckFunc {
		return func(*terraform.State) error {
			got, err := c.read(context.Background(), testAccClient(), id)
			if err != nil {
				return err
			}
			var gotSlugs []string
			if got != nil {
				gotSlugs = *got
			}
			// The API reports no private locations as [].
			if !slices.Equal(gotSlugs, want) {
				return fmt.Errorf("privateLocations = %v, want %v", gotSlugs, want)
			}
			return nil
		}
	}

	accTestCase(t, []resource.TestStep{
		{
			Config: config(`private_locations = [checkly_private_location.a.slug_name]`),
			Check: resource.ComposeTestCheckFunc(
				captureID,
				resource.TestCheckResourceAttr(c.resource, "private_locations.#", "1"),
				resource.TestCheckTypeSetElemAttr(c.resource, "private_locations.*", slugA),
				stored(slugA),
			),
		},
		{
			Config: config(`private_locations = [checkly_private_location.b.slug_name]`),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(c.resource, "private_locations.#", "1"),
				resource.TestCheckTypeSetElemAttr(c.resource, "private_locations.*", slugB),
				stored(slugB),
			),
		},
		{
			ResourceName:      c.resource,
			ImportState:       true,
			ImportStateVerify: true,
		},
		{
			Config: config(""),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(c.resource, "private_locations.#"),
				stored(),
			),
		},
		{
			// Private locations set outside Terraform show up as a removal.
			PreConfig: func() {
				if err := c.set(context.Background(), testAccClient(), id, []string{slugA}); err != nil {
					t.Fatal(err)
				}
			},
			Config: config(""),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(c.resource, plancheck.ResourceActionUpdate),
				},
			},
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(c.resource, "private_locations.#"),
				stored(),
			),
		},
	})
}
