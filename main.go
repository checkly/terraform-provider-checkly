package main

import (
	"flag"

	"github.com/checkly/terraform-provider-checkly/checkly"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

func main() {
	var debugMode bool

	flag.BoolVar(&debugMode, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := &plugin.ServeOpts{
		Debug:        debugMode,
		ProviderAddr: "registry.terraform.io/checkly/checkly",
		ProviderFunc: checkly.Provider,
	}

	plugin.Serve(opts)
}
