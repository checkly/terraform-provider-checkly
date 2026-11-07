package provider

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// descriptionMarker matches the clauses that state a constraint on an
// attribute's value rather than what the attribute is: a default, a range,
// or a list of possible values.
var descriptionMarker = regexp.MustCompile("(?i)\\(default [^)]*\\)|value must be (between|one of) [^.(]*|(the )?(possible|allowed) values( are| is)?:? [^.(]*")

// hasDescription reports whether description says what an attribute or
// block is, beyond the constraint markers that descriptionMarker matches.
func hasDescription(description string) bool {
	rest := descriptionMarker.ReplaceAllString(description, "")
	return strings.ContainsFunc(rest, func(r rune) bool {
		return ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z')
	})
}

// walkDescriptions fails for every attribute and block in block, at any
// depth, whose description is empty or only states a default, range or
// possible values. It walks the protocol schema, the
// form Terraform and the documentation generator see, so the provider,
// resource and data source schemas share one walker whatever framework
// package defined them.
func walkDescriptions(t *testing.T, where string, block *tfprotov6.SchemaBlock) {
	t.Helper()

	if block == nil {
		return
	}

	for _, a := range block.Attributes {
		walkAttributeDescriptions(t, where+"."+a.Name, a)
	}

	for _, b := range block.BlockTypes {
		path := where + "." + b.TypeName
		if b.Block == nil || !hasDescription(b.Block.Description) {
			t.Errorf("%s: block has no description", path)
		}
		walkDescriptions(t, path, b.Block)
	}
}

// walkAttributeDescriptions fails when the attribute at path, or any
// attribute nested in it, has no description beyond constraint markers.
func walkAttributeDescriptions(t *testing.T, path string, a *tfprotov6.SchemaAttribute) {
	t.Helper()

	if !hasDescription(a.Description) {
		t.Errorf("%s: attribute has no description: %q", path, a.Description)
	}

	if a.NestedType == nil {
		return
	}

	for _, nested := range a.NestedType.Attributes {
		walkAttributeDescriptions(t, path+"."+nested.Name, nested)
	}
}

// TestEveryAttributeHasDescription keeps the generated documentation
// complete: every attribute and block of the provider, resource and data
// source schemas, at any depth, has a description that says more than its
// default, range or possible values.
func TestEveryAttributeHasDescription(t *testing.T) {
	ctx := context.Background()

	server := providerserver.NewProtocol6(New("test")())()

	resp, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("GetProviderSchema: %v", err)
	}
	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("GetProviderSchema: %s: %s", d.Summary, d.Detail)
		}
	}

	if resp.Provider == nil {
		t.Fatal("GetProviderSchema returned no provider schema")
	}
	walkDescriptions(t, "provider", resp.Provider.Block)

	if len(resp.ResourceSchemas) == 0 || len(resp.DataSourceSchemas) == 0 {
		t.Fatalf("GetProviderSchema returned %d resource and %d data source schemas; want both", len(resp.ResourceSchemas), len(resp.DataSourceSchemas))
	}
	for name, s := range resp.ResourceSchemas {
		walkDescriptions(t, name, s.Block)
	}
	for name, s := range resp.DataSourceSchemas {
		walkDescriptions(t, name, s.Block)
	}
}
