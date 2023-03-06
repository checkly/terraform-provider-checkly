package checkly

import (
	"regexp"
	"testing"

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
	config := `resource "checkly_private_location" "test" {
		name     = "New Private Location"
		slug_name   = "new-private-location"
		icon       	= "bell-fill"
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config: config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_private_location.test",
					"name",
					"New Private Location",
				),
				resource.TestCheckResourceAttr(
					"checkly_private_location.test",
					"slug_name",
					"new-private-location",
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
	config := `resource "checkly_private_location" "without_icon" {
		name     = "New Private Location"
		slug_name   = "new-private-location"
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config: config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_private_location.without_icon",
					"name",
					"New Private Location",
				),
				resource.TestCheckResourceAttr(
					"checkly_private_location.without_icon",
					"slug_name",
					"new-private-location",
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
