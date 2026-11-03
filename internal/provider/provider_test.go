package provider

import (
	"context"
	"strings"
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
// output. Attributes of nested objects and blocks are given as dotted paths.
func TestSensitiveAttributes(t *testing.T) {
	ctx := context.Background()

	var providerSchema provider.SchemaResponse
	New("test")().Schema(ctx, provider.SchemaRequest{}, &providerSchema)
	if !providerSchema.Schema.Attributes["api_key"].IsSensitive() {
		t.Errorf("provider attribute api_key is not Sensitive")
	}

	schemas := resourceSchemas(t)

	cases := map[string][]string{
		"checkly_client_certificate":   {"private_key", "passphrase"},
		"checkly_environment_variable": {"value"},
		"checkly_dashboard":            {"key"},
		"checkly_private_location":     {"keys"},
		"checkly_alert_channel": {
			"slack.url",
			"webhook.url",
			"webhook.webhook_secret",
			"opsgenie.api_key",
			"pagerduty.service_key",
		},
		"checkly_api_check":              {"request.basic_auth.password"},
		"checkly_check":                  {"request.basic_auth.password", "environment_variable.value"},
		"checkly_check_group":            {"api_check_defaults.basic_auth.password", "environment_variable.value"},
		"checkly_check_group_v2":         {"api_check_defaults.basic_auth.password", "environment_variable.value"},
		"checkly_browser_check":          {"environment_variable.value"},
		"checkly_multistep_check":        {"environment_variable.value"},
		"checkly_playwright_check_suite": {"environment_variable.value"},
		"checkly_heartbeat_monitor":      {"heartbeat.ping_token"},
		"checkly_heartbeat":              {"heartbeat.ping_token"},
	}

	for resourceName, attrPaths := range cases {
		s, ok := schemas[resourceName]
		if !ok {
			t.Errorf("resource %s not found", resourceName)
			continue
		}
		for _, attrPath := range attrPaths {
			attr, ok := schemaAttributeAtPath(s, strings.Split(attrPath, "."))
			if !ok {
				t.Errorf("%s: attribute %s not found", resourceName, attrPath)
				continue
			}
			if !attr.IsSensitive() {
				t.Errorf("%s: attribute %s is not Sensitive", resourceName, attrPath)
			}
		}
	}
}

// schemaAttributeAtPath looks up an attribute by the names of the attributes
// and blocks that lead to it, skipping over list and set elements.
func schemaAttributeAtPath(s schema.Schema, names []string) (schema.Attribute, bool) {
	attributes, blocks := s.Attributes, s.Blocks
	for i, name := range names {
		if attr, ok := attributes[name]; ok {
			if i == len(names)-1 {
				return attr, true
			}
			switch attr := attr.(type) {
			case schema.SingleNestedAttribute:
				attributes, blocks = attr.Attributes, nil
			case schema.ListNestedAttribute:
				attributes, blocks = attr.NestedObject.Attributes, nil
			case schema.SetNestedAttribute:
				attributes, blocks = attr.NestedObject.Attributes, nil
			default:
				return nil, false
			}
			continue
		}

		switch block := blocks[name].(type) {
		case schema.SingleNestedBlock:
			attributes, blocks = block.Attributes, block.Blocks
		case schema.ListNestedBlock:
			attributes, blocks = block.NestedObject.Attributes, block.NestedObject.Blocks
		case schema.SetNestedBlock:
			attributes, blocks = block.NestedObject.Attributes, block.NestedObject.Blocks
		default:
			return nil, false
		}
	}

	return nil, false
}
