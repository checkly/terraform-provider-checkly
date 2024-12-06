package resources_test

import (
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/checkly/terraform-provider-checkly/internal/provider"
	"github.com/checkly/terraform-provider-checkly/internal/provider/globalregistry"
)

func protoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"checkly": providerserver.NewProtocol6WithError(provider.New("test", globalregistry.Registry)()),
	}
}
