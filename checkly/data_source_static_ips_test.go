package checkly

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func checkResourceAttrGreaterThanOrEqualTo(n int) resource.CheckResourceAttrWithFunc {
	return func(value string) error {
		i, err := strconv.Atoi(value)
		if err != nil {
			return err
		}

		if i < n {
			return fmt.Errorf("%v must be greater than or equal to %v", i, n)
		}

		return nil
	}
}

func TestAccStaticIPsAll(t *testing.T) {
	config := `data "checkly_static_ips" "test" {}`

	accTestCase(t, []resource.TestStep{
		{
			Config: config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrWith(
					"data.checkly_static_ips.test",
					"addresses.#",
					checkResourceAttrGreaterThanOrEqualTo(1),
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
				resource.TestCheckResourceAttrWith(
					"data.checkly_static_ips.test",
					"addresses.#",
					checkResourceAttrGreaterThanOrEqualTo(1),
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
				resource.TestCheckResourceAttrWith(
					"data.checkly_static_ips.test",
					"addresses.#",
					checkResourceAttrGreaterThanOrEqualTo(1),
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
				resource.TestCheckResourceAttrWith(
					"data.checkly_static_ips.test",
					"addresses.#",
					checkResourceAttrGreaterThanOrEqualTo(1),
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
				resource.TestCheckResourceAttrWith(
					"data.checkly_static_ips.test",
					"addresses.#",
					checkResourceAttrGreaterThanOrEqualTo(1),
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
				resource.TestCheckResourceAttrWith(
					"data.checkly_static_ips.test",
					"addresses.#",
					checkResourceAttrGreaterThanOrEqualTo(1),
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
