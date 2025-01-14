package resources_test

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/checkly/terraform-provider-checkly/internal/provider"
	"github.com/checkly/terraform-provider-checkly/internal/provider/globalregistry"
)

func protoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"checkly": providerserver.NewProtocol6WithError(provider.New("test", globalregistry.Registry)()),
	}
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
