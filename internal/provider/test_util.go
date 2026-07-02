package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every
// Terraform CLI command executed to create a provider server to which the
// CLI can reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"checkly": providerserver.NewProtocol6WithError(New("test")()),
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
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             nil,
		ErrorCheck:               errorCheck,
		Steps:                    steps,
	})
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
