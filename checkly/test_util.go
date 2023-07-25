package checkly

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var testAccProviders map[string]*schema.Provider

func init() {
	testAccProviders = map[string]*schema.Provider{
		"checkly": Provider(),
	}
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
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: nil,
		Steps:        steps,
	})
}

// test resource using regular expressions
// this helps testing arrays which have irregular indices;
// needed because we get things like "alert_settings.2888461220.escalation_type": "RUN_BASED"
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
