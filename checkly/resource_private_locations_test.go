package checkly

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccPrivateLocationsCheckRequiredFields(t *testing.T) {
	config := `resource "checkly_private_locations" "test" {}`
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

func TestAccPrivateLocationsSuccess(t *testing.T) {
	config := `resource "checkly_private_locations" "test" {
		name     = "New Private Location"
		slug_name   = "new-private-location"
		icon       	= "location"
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config: config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_private_locations.test",
					"name",
					"New Private Location",
				),
				resource.TestCheckResourceAttr(
					"checkly_private_locations.test",
					"slug_name",
					"new-private-location",
				),
				resource.TestCheckResourceAttr(
					"checkly_private_locations.test",
					"icon",
					"location",
				),
			),
		},
	})
}
