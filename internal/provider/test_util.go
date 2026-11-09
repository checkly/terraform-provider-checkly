package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/checkly/checkly-go-sdk/v2"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every
// Terraform CLI command executed to create a provider server to which the
// CLI can reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"checkly": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccClient returns an API client configured like the
// provider under test, from the acceptance test environment.
func testAccClient() checkly.Client {
	apiURL := os.Getenv("CHECKLY_API_URL")
	if apiURL == "" {
		apiURL = "https://api.checklyhq.com"
	}
	client, err := checkly.NewClient(
		checkly.WithBaseURL(apiURL),
		checkly.WithAuth(os.Getenv("CHECKLY_API_KEY"), os.Getenv("CHECKLY_ACCOUNT_ID")),
	)
	if err != nil {
		panic(err)
	}
	return client
}

func testAccPreCheck(t *testing.T) {
	if os.Getenv("CHECKLY_API_KEY") == "" {
		t.Fatal("CHECKLY_API_KEY must be set for acceptance tests")
	}

	if os.Getenv("CHECKLY_ACCOUNT_ID") == "" {
		t.Fatal("CHECKLY_ACCOUNT_ID must be set for acceptance tests")
	}
}

func accTestCase(t *testing.T, steps []resource.TestStep) {
	accTestCaseWithErrorCheck(t, steps, nil)
}

// accTestCaseWithErrorCheck behaves like accTestCase but routes any step error
// through errorCheck before the framework reports it. This lets a test skip
// when the account lacks setup the test depends on (e.g. a connected
// integration) rather than failing.
func accTestCaseWithErrorCheck(t *testing.T, steps []resource.TestStep, errorCheck resource.ErrorCheckFunc) {
	accTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             nil,
		ErrorCheck:               errorCheck,
		TerraformVersionChecks:   terraformVersionChecks(steps),
		Steps:                    steps,
	})
}

// alertChannelResource matches a configuration that declares an alert
// channel, but not one that only subscribes to an existing channel.
var alertChannelResource = regexp.MustCompile(`(?m)^\s*resource\s+"checkly_alert_channel"\s`)

// createsAlertChannel reports whether any step declares an alert channel.
func createsAlertChannel(steps []resource.TestStep) bool {
	for _, step := range steps {
		if alertChannelResource.MatchString(step.Config) {
			return true
		}
	}

	return false
}

// accTest runs testCase in parallel with other tests, unless its steps
// create an alert channel. Creating an alert channel subscribes it to every
// group in the account and to every check outside a group, so a check or
// group test running at the same time would find a subscription it never
// configured. Go runs the sequential top-level tests one at a time before it
// starts the parallel ones, so no other test of this package has resources
// in the account while a channel is created. Acceptance tests must run
// through accTest, not call resource.ParallelTest directly.
func accTest(t *testing.T, testCase resource.TestCase) {
	if createsAlertChannel(testCase.Steps) {
		resource.Test(t, testCase)
	} else {
		resource.ParallelTest(t, testCase)
	}
}

// configBlockVersionChecks lists the configuration blocks that need a newer
// Terraform than the provider's minimum of 1.0, with a check that skips the
// versions which cannot run them as the tests use them.
var configBlockVersionChecks = []struct {
	block *regexp.Regexp
	check tfversion.TerraformVersionCheck
}{
	// moved blocks between resource types.
	{regexp.MustCompile(`(?m)^\s*moved\s*\{`), tfversion.SkipBelow(tfversion.Version1_8_0)},
	{regexp.MustCompile(`(?m)^\s*removed\s*\{`), tfversion.SkipBelow(tfversion.Version1_7_0)},
	// import blocks whose id refers to another resource.
	{regexp.MustCompile(`(?m)^\s*import\s*\{`), tfversion.SkipBelow(tfversion.Version1_6_0)},
	// The built-in resource whose output is known only at apply time.
	{regexp.MustCompile(`(?m)^\s*resource\s+"terraform_data"`), tfversion.SkipBelow(tfversion.Version1_4_0)},
	// ignore_changes paths into a nested attribute, which Terraform before
	// 1.2.7 rejects as an unsupported attribute.
	{regexp.MustCompile(`(?m)^\s*ignore_changes\s*=\s*\[[^\]]*\w\.\w`), tfversion.SkipBelow(version.Must(version.NewVersion("1.2.7")))},
}

// terraformVersionChecks skips a test on Terraform versions that cannot run
// the configuration blocks its steps use.
func terraformVersionChecks(steps []resource.TestStep) []tfversion.TerraformVersionCheck {
	var checks []tfversion.TerraformVersionCheck
	for _, b := range configBlockVersionChecks {
		for _, step := range steps {
			if b.block.MatchString(step.Config) {
				checks = append(checks, b.check)
				break
			}
		}
	}

	return checks
}

// testCheckResourceAttrExpr tests resource attributes using regular
// expressions applied to the JSON-marshaled attribute map. This helps testing
// collections which have irregular indices.
func testCheckResourceAttrExpr(resource, attrExpr, value string) func(s *terraform.State) error {
	return func(s *terraform.State) error {
		if len(s.Modules) < 1 {
			return fmt.Errorf("testCheckResourceAttrExpr: state has no modules")
		}
		if _, ok := s.Modules[0].Resources[resource]; !ok {
			return fmt.Errorf("Resource not found: %s", resource)
		}
		marshaled, _ := json.Marshal(s.Modules[0].Resources[resource].Primary.Attributes)
		r, err := regexp.Compile(attrExpr)
		if err != nil {
			return err
		}
		if !r.MatchString(string(marshaled)) {
			return fmt.Errorf(`Couldn't find [%s: "%s"] in %s`, attrExpr, value, string(marshaled))
		}
		return err
	}
}
