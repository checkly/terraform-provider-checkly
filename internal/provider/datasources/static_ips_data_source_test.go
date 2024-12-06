package datasources_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/checkly/terraform-provider-checkly/internal/provider"
	"github.com/checkly/terraform-provider-checkly/internal/provider/globalregistry"
)

func protoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"checkly": providerserver.NewProtocol6WithError(provider.New("test", globalregistry.Registry)()),
	}
}

func TestAccStaticIPsAll(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),

		Steps: []resource.TestStep{
			{
				Config: `
					data "checkly_static_ips" "test" {}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.checkly_static_ips.test",
						"addresses.#",
						"162",
					),
				),
			},
		},
	})
}

func TestAccStaticIPsTwoRegionsOnly(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),

		Steps: []resource.TestStep{
			{
				Config: `
					data "checkly_static_ips" "test" {
    					locations = ["us-east-1","ap-southeast-1"]
  					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.checkly_static_ips.test",
						"addresses.#",
						"20",
					),
				),
			},
		},
	})
}

func TestAccStaticIPsIPv6Only(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),

		Steps: []resource.TestStep{
			{
				Config: `
					data "checkly_static_ips" "test" {
						ip_family = "IPv6"
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.checkly_static_ips.test",
						"addresses.#",
						"22",
					),
				),
			},
		},
	})
}

func TestAccStaticIPsIPv4Only(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),

		Steps: []resource.TestStep{
			{
				Config: `
					data "checkly_static_ips" "test" {
						ip_family = "IPv4"
 					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.checkly_static_ips.test",
						"addresses.#",
						"140",
					),
				),
			},
		},
	})
}

func TestAccStaticIPsIPv6AndOneRegionOnly(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),

		Steps: []resource.TestStep{
			{
				Config: `
					data "checkly_static_ips" "test" {
						ip_family = "IPv6"
						locations = ["us-east-1"]
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.checkly_static_ips.test",
						"addresses.#",
						"1",
					),
				),
			},
		},
	})
}

func TestAccStaticIPsIPv4AndOneRegionOnly(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),

		Steps: []resource.TestStep{
			{
				Config: `
					data "checkly_static_ips" "test" {
						ip_family = "IPv4"
						locations = ["us-east-1"]
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.checkly_static_ips.test",
						"addresses.#",
						"12",
					),
				),
			},
		},
	})
}

func TestAccStaticIPsInvalidIPFamily(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),

		Steps: []resource.TestStep{
			{
				Config: `
					data "checkly_static_ips" "test" {
						ip_family = "invalid"
					}
				`,
				ExpectError: regexp.MustCompile(`Invalid Attribute Value Match`),
			},
		},
	})
}
