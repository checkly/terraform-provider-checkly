package main

import (
	"github.com/checkly/terraform-provider-checkly/checkly"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: checkly.Provider,
	})
}
