package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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

	// expectUpdate is the address of a resource that is expected to be
	// updated in place once after the upgrade. The plan after that update
	// must be empty.
	expectUpdate string
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
				upgradeStep(c.config, c.expectUpdate),
			}
			if c.expectUpdate != "" {
				steps = append(steps, upgradeStep(c.config, ""))
			}

			resource.ParallelTest(t, resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(t)
				},
				TerraformVersionChecks: terraformVersionChecks(steps),
				Steps:                  steps,
			})
		})
	}
}

// upgradeStep plans and applies config with this provider and fails unless
// the plan was empty or, if expectUpdate is set, only updated that resource
// in place.
//
// The plan is recorded rather than checked before the apply: a step that
// fails before applying leaves state at the SDK v2 schema version, which
// the test framework then cannot read back with this provider, so it would
// skip destroying the resources.
func upgradeStep(config, expectUpdate string) resource.TestStep {
	recorder := &planChangeRecorder{}

	return resource.TestStep{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Config:                   config,
		ConfigPlanChecks: resource.ConfigPlanChecks{
			PreApply: []plancheck.PlanCheck{recorder},
		},
		Check: func(*terraform.State) error {
			changes := recorder.changes
			if expectUpdate != "" {
				expected, ok := changes[expectUpdate]
				if !ok || !expected.actions.Update() {
					return fmt.Errorf("expected an in-place update of %s, got:\n%s", expectUpdate, formatPlanChanges(changes))
				}
				delete(changes, expectUpdate)
			}
			if len(changes) > 0 {
				return fmt.Errorf("expected an empty plan, got:\n%s", formatPlanChanges(changes))
			}
			return nil
		},
	}
}

// planChangeRecorder is a plan check that never fails. It records every
// planned change by resource address.
type planChangeRecorder struct {
	changes map[string]plannedChange
}

// plannedChange is a planned change of a resource and the attributes it
// changes, formatted for a test failure message.
type plannedChange struct {
	actions tfjson.Actions
	diff    string
}

func (r *planChangeRecorder) CheckPlan(ctx context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	r.changes = map[string]plannedChange{}

	for _, rc := range req.Plan.ResourceChanges {
		if rc.Change == nil || rc.Change.Actions.NoOp() {
			continue
		}

		before, _ := rc.Change.Before.(map[string]any)
		after, _ := rc.Change.After.(map[string]any)
		beforeSensitive, _ := rc.Change.BeforeSensitive.(map[string]any)
		afterSensitive, _ := rc.Change.AfterSensitive.(map[string]any)
		afterUnknown, _ := rc.Change.AfterUnknown.(map[string]any)

		names := map[string]bool{}
		for name := range before {
			names[name] = true
		}
		for name := range after {
			names[name] = true
		}

		var lines []string
		for name := range names {
			unknown := containsTrue(afterUnknown[name])
			if reflect.DeepEqual(before[name], after[name]) && !unknown {
				continue
			}
			if containsTrue(beforeSensitive[name]) || containsTrue(afterSensitive[name]) {
				lines = append(lines, fmt.Sprintf("    %s: (sensitive)", name))
				continue
			}
			afterValue := planValueJSON(after[name])
			if unknown {
				afterValue += " (unknown: " + planValueJSON(afterUnknown[name]) + ")"
			}
			lines = append(lines, fmt.Sprintf("    %s: %s -> %s", name, planValueJSON(before[name]), afterValue))
		}
		sort.Strings(lines)

		r.changes[rc.Address] = plannedChange{actions: rc.Change.Actions, diff: strings.Join(lines, "\n")}
	}
}

// containsTrue reports whether a before_sensitive, after_sensitive or
// after_unknown value of a Terraform JSON plan marks anything: those values
// mirror the attribute's structure, with true at each marked value.
func containsTrue(marker any) bool {
	switch marker := marker.(type) {
	case bool:
		return marker
	case []any:
		for _, element := range marker {
			if containsTrue(element) {
				return true
			}
		}
	case map[string]any:
		for _, element := range marker {
			if containsTrue(element) {
				return true
			}
		}
	}
	return false
}

func planValueJSON(value any) string {
	b, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(b)
}

func formatPlanChanges(changes map[string]plannedChange) string {
	addresses := make([]string, 0, len(changes))
	for address := range changes {
		addresses = append(addresses, address)
	}
	sort.Strings(addresses)

	var b strings.Builder
	for _, address := range addresses {
		fmt.Fprintf(&b, "  %s %v\n%s\n", address, changes[address].actions, changes[address].diff)
	}
	return b.String()
}
