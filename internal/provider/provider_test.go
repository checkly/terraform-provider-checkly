package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// resourceSchemas returns the schema of every resource the provider
// registers, keyed by resource type name.
func resourceSchemas(t *testing.T) map[string]schema.Schema {
	t.Helper()

	ctx := context.Background()
	p := New("test")()

	var providerMetadata provider.MetadataResponse
	p.Metadata(ctx, provider.MetadataRequest{}, &providerMetadata)

	schemas := make(map[string]schema.Schema)
	for _, newResource := range p.Resources(ctx) {
		r := newResource()

		var metadata resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: providerMetadata.TypeName}, &metadata)

		var resp resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("%s: schema diagnostics: %v", metadata.TypeName, resp.Diagnostics)
		}

		schemas[metadata.TypeName] = resp.Schema
	}

	return schemas
}

func TestResourceSchemasAreValid(t *testing.T) {
	ctx := context.Background()

	for name, s := range resourceSchemas(t) {
		if diags := s.ValidateImplementation(ctx); diags.HasError() {
			t.Errorf("%s: invalid schema: %v", name, diags)
		}
	}
}

// Credentials must be marked Sensitive so Terraform redacts them in plan
// output.
func TestSensitiveAttributes(t *testing.T) {
	ctx := context.Background()

	var providerSchema provider.SchemaResponse
	New("test")().Schema(ctx, provider.SchemaRequest{}, &providerSchema)
	if !providerSchema.Schema.Attributes["api_key"].IsSensitive() {
		t.Errorf("provider attribute api_key is not Sensitive")
	}

	schemas := resourceSchemas(t)

	cases := map[string][]string{
		"checkly_client_certificate": {"private_key", "passphrase"},
	}

	for resourceName, attrNames := range cases {
		s, ok := schemas[resourceName]
		if !ok {
			t.Errorf("resource %s not found", resourceName)
			continue
		}
		for _, attrName := range attrNames {
			attr, ok := s.Attributes[attrName]
			if !ok {
				t.Errorf("%s: attribute %s not found", resourceName, attrName)
				continue
			}
			if !attr.IsSensitive() {
				t.Errorf("%s: attribute %s is not Sensitive", resourceName, attrName)
			}
		}
	}
}
