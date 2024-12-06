package provider_test

import (
	"regexp"
	"testing"

	"github.com/checkly/terraform-provider-checkly/internal/provider"
	"github.com/checkly/terraform-provider-checkly/internal/provider/globalregistry"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func protoV6ProviderFactories(options ...provider.Option) map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"checkly": providerserver.NewProtocol6WithError(provider.New("test", globalregistry.Registry, options...)()),
	}
}

func TestProviderConfigMissingAPIKey(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(
			provider.WithUseCredentialsFromEnvironment(false),
		),

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

func TestProviderConfigMissingAccountIDWithUserAPIKey(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(
			provider.WithUseCredentialsFromEnvironment(false),
		),

		Steps: []resource.TestStep{
			{
				Config: `
					provider "checkly" {
						api_key = "cu_foo"
					}

					// Must use the provider or it won't get configured.
					data "checkly_static_ips" "test" {}
				`,
				ExpectError: regexp.MustCompile("Missing Checkly Account ID"),
			},
		},
	})
}
