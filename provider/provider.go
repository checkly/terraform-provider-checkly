// Package provider exposes the Checkly Terraform provider to other Go
// modules, such as the Pulumi bridge, which cannot import the internal
// package that implements it.
package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/provider"

	internalprovider "github.com/checkly/terraform-provider-checkly/internal/provider"
)

// New returns a factory for the Checkly provider, reporting the given
// version. Downstream bridges call it, so its signature must stay stable.
func New(version string) func() provider.Provider {
	return internalprovider.New(version)
}
