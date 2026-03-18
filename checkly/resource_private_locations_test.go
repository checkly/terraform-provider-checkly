package checkly

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
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
			),
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
