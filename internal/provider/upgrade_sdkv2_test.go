package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// sdkv2ProviderVersion is the last release of the provider built on the
// Terraform Plugin SDK v2.
const sdkv2ProviderVersion = "1.32.0"

// sdkv2UpgradeCase describes the same objects twice: as configured for the
// SDK v2 release, and in the current syntax, which models single nested
// objects as attributes (`alert_settings = { ... }`) rather than blocks.
type sdkv2UpgradeCase struct {
	sdkv2Config string
	config      string

	// planChecks replace the expectation of an empty plan for cases that
	// are expected to change once. A further step then expects the plan
	// to be empty.
	planChecks []plancheck.PlanCheck
}

// TestAccUpgradeFromSDKv2 applies each case with the SDK v2 release, then
// plans the equivalent configuration with this provider. The plan must be
// empty: the old state has to upgrade, refresh and match the configuration
// without updates or replacements.
func TestAccUpgradeFromSDKv2(t *testing.T) {
	// State written by the registry release belongs to
	// registry.terraform.io/checkly/checkly. The in-process provider must be
	// registered under the same address to take that state over, rather than
	// under the default hashicorp namespace. Setting it here, before the
	// subtests run in parallel, keeps it set until they have all finished.
	t.Setenv(resource.EnvTfAccProviderNamespace, "checkly")

	for name, c := range sdkv2UpgradeCases {
		t.Run(name, func(t *testing.T) {
			emptyPlanStep := resource.TestStep{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   c.config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			}

			steps := []resource.TestStep{
				{
					ExternalProviders: map[string]resource.ExternalProvider{
						"checkly": {
							Source:            "checkly/checkly",
							VersionConstraint: sdkv2ProviderVersion,
						},
					},
					Config: c.sdkv2Config,
				},
			}
			if c.planChecks == nil {
				steps = append(steps, emptyPlanStep)
			} else {
				steps = append(steps, resource.TestStep{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Config:                   c.config,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: c.planChecks,
					},
				}, emptyPlanStep)
			}

			resource.ParallelTest(t, resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(t)
				},
				Steps: steps,
			})
		})
	}
}
