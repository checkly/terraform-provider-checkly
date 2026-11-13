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
var descriptionMarker = regexp.MustCompile("(?i)defaults to `[^`]*`|must be (at least|at most|between) [^.(]*|(the only )?possible values?( are| is)?:? [^.(]*")

// hasDescription reports whether description says what an attribute or
// block is, beyond the constraint markers that descriptionMarker matches.
func hasDescription(description string) bool {
	rest := descriptionMarker.ReplaceAllString(description, "")
	return strings.ContainsFunc(rest, func(r rune) bool {
		return ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z')
	})
}

// walkDescriptions fails for every attribute in block, at any depth, whose
// description is empty or only states a default, range or possible values.
// No schema of the provider has nested blocks (TestNoBlocks covers resources
// and data sources), so attributes are all there is to walk. It walks the
// protocol schema, the form Terraform and the documentation generator see, so
// the provider, resource and data source schemas share one walker whatever
// framework package defined them.
func walkDescriptions(t *testing.T, where string, block *tfprotov6.SchemaBlock) {
	t.Helper()

	if block == nil {
		return
	}

	for _, a := range block.Attributes {
		walkAttributeDescriptions(t, where+"."+a.Name, a)
	}
}

// walkAttributeDescriptions fails when the attribute at path, or any
// attribute nested in it, has no description beyond constraint markers, or
// has a description with a gap that missingWord matches.
func walkAttributeDescriptions(t *testing.T, path string, a *tfprotov6.SchemaAttribute) {
	t.Helper()

	if !hasDescription(a.Description) {
		t.Errorf("%s: attribute has no description: %q", path, a.Description)
	}
	if gap := missingWord.FindString(a.Description); gap != "" {
		t.Errorf("%s: description has a gap (%q), as left by an empty interpolated word: %q", path, gap, a.Description)
	}
	for _, s := range descriptionStyleViolations {
		if s.pattern.MatchString(a.Description) {
			t.Errorf("%s: description %s (see Attribute descriptions in CONTRIBUTING.md): %q", path, s.problem, a.Description)
		}
	}

	if a.NestedType == nil {
		return
	}

	for _, nested := range a.NestedType.Attributes {
		walkAttributeDescriptions(t, path+"."+nested.Name, nested)
	}
}

// descriptionStyleViolations match wording that the attribute description
// convention in CONTRIBUTING.md replaces, so that new descriptions follow it.
var descriptionStyleViolations = []struct {
	pattern *regexp.Regexp
	problem string
}{
	{regexp.MustCompile("\\(Default `"), "states a default as \"(Default `X`)\" instead of \"Defaults to `X`.\""},
	{regexp.MustCompile("The allowed values are|The only allowed value is"), "lists values as \"allowed values\" instead of \"Possible values are\""},
	{regexp.MustCompile("(?i)possible values `"), "lists values without \"are\" after \"Possible values\""},
	{regexp.MustCompile("(?i)value must be (between|one of)"), "states a range or values as \"Value must be\" instead of \"Possible values are\""},
	{regexp.MustCompile("(?i)the value can be either"), "lists values as \"The value can be either\" instead of \"Possible values are\""},
	{regexp.MustCompile("`, (and|or) `"), "has a comma before the last element of a list"},
	{regexp.MustCompile("Defaults to .*([Pp]ossible values? (are|is)|[Mm]ust be (at least|at most|between))"), "states a restriction after the default"},
	{regexp.MustCompile("When unset|When not set"), "states a conditional default as \"When unset\" instead of \"If not set\""},
}

// TestEveryAttributeHasDescription keeps the generated documentation
// complete: every attribute and block of the provider, resource and data
// source schemas, at any depth, has a description that says more than its
// default, range or possible values, and no description has the gap that an
// empty interpolated word leaves.
func TestEveryAttributeHasDescription(t *testing.T) {
	ctx := context.Background()

	p := New("test")()
	server := providerserver.NewProtocol6(p)()

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
	// The protocol schemas are built from the provider's Resources list;
	// matching the count makes sure every registered resource is checked.
	if want := len(p.Resources(ctx)); len(resp.ResourceSchemas) != want {
		t.Fatalf("GetProviderSchema returned %d resource schemas, want %d", len(resp.ResourceSchemas), want)
	}
	for name, s := range resp.ResourceSchemas {
		walkDescriptions(t, name, s.Block)
	}
	for name, s := range resp.DataSourceSchemas {
		walkDescriptions(t, name, s.Block)
	}
}

// missingWord matches the gaps that an empty interpolated word leaves in a
// sentence: a doubled space ("how often the  should run") or a space before
// punctuation ("the policy for the .").
var missingWord = regexp.MustCompile(`  | [.,;:)]`)
