package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func protoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"checkly": providerserver.NewProtocol6WithError(New("test")()),
	}
}

func TestProviderConfigMissingAPIKey(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),

		Steps: []resource.TestStep{
			{
				Config: `
					provider "checkly" {
					}

					// Must use the provider or it won't get configured.
					data "checkly_static_ips" "test" {}
				`,
				ExpectError: regexp.MustCompile("Missing Checkly API Key"),
			},
		},
	})
}
