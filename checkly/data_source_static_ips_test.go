package checkly

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccStaticIPsAll(t *testing.T) {
	config := `data "checkly_static_ips" "test" {}`

	accTestCase(t, []resource.TestStep{
		{
			Config: config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"data.checkly_static_ips.test",
					"addresses.#",
					"162",
				),
			),
		},
	})
}

func TestAccStaticIPsTwoRegionsOnly(t *testing.T) {
	config := `data "checkly_static_ips" "test" {
    locations = ["us-east-1","ap-southeast-1"]
  }`

	accTestCase(t, []resource.TestStep{
		{
			Config: config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"data.checkly_static_ips.test",
					"addresses.#",
					"20",
				),
			),
		},
	})
}

func TestAccStaticIPsIPv6Only(t *testing.T) {
	config := `data "checkly_static_ips" "test" {
    ip_family = "IPv6"
  }`

	accTestCase(t, []resource.TestStep{
		{
			Config: config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"data.checkly_static_ips.test",
					"addresses.#",
					"22",
				),
			),
		},
	})
}

func TestAccStaticIPsIPv4Only(t *testing.T) {
	config := `data "checkly_static_ips" "test" {
    ip_family = "IPv4"
  }`

	accTestCase(t, []resource.TestStep{
		{
			Config: config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"data.checkly_static_ips.test",
					"addresses.#",
					"140",
				),
			),
		},
	})
}

func TestAccStaticIPsIPv6AndOneRegionOnly(t *testing.T) {
	config := `data "checkly_static_ips" "test" {
    ip_family = "IPv6"
    locations = ["us-east-1"]
  }`

	accTestCase(t, []resource.TestStep{
		{
			Config: config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"data.checkly_static_ips.test",
					"addresses.#",
					"1",
				),
			),
		},
	})
}

func TestAccStaticIPsIPv4AndOneRegionOnly(t *testing.T) {
	config := `data "checkly_static_ips" "test" {
    ip_family = "IPv4"
    locations = ["us-east-1"]
  }`

	accTestCase(t, []resource.TestStep{
		{
			Config: config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"data.checkly_static_ips.test",
					"addresses.#",
					"12",
				),
			),
		},
	})
}

func TestAccStaticIPsInvalidIPFamily(t *testing.T) {
	config := `data "checkly_static_ips" "test" {
    ip_family = "invalid"
  }`

	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`"ip_family" must be either "IPv6" or "IPv4"`),
		},
	})
}
