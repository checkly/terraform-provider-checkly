package checkly

import (
	"testing"
)

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

// Credentials must be marked Sensitive so Terraform redacts them in plan
// output. A trigger url embeds its token, so it is a credential too.
func TestSensitiveAttributes(t *testing.T) {
	p := Provider()

	if !p.Schema["api_key"].Sensitive {
		t.Errorf("provider attribute api_key is not Sensitive")
	}

	cases := map[string][]string{
		"checkly_client_certificate": {"private_key", "passphrase"},
		"checkly_trigger_check":      {"token", "url"},
		"checkly_trigger_group":      {"token", "url"},
	}

	for resourceName, attrNames := range cases {
		resource, ok := p.ResourcesMap[resourceName]
		if !ok {
			t.Errorf("resource %s not found", resourceName)
			continue
		}
		for _, attrName := range attrNames {
			attr, ok := resource.Schema[attrName]
			if !ok {
				t.Errorf("%s: attribute %s not found", resourceName, attrName)
				continue
			}
			if !attr.Sensitive {
				t.Errorf("%s: attribute %s is not Sensitive", resourceName, attrName)
			}
		}
	}
}
