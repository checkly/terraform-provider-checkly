package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// attributeValidationErrors runs the validators of the attribute that names
// leads to in s on value, and returns the summaries and details of the
// errors. It covers string and list attributes whose validators read only
// the value itself.
func attributeValidationErrors(t *testing.T, s schema.Schema, names []string, value attr.Value) []string {
	t.Helper()
	ctx := context.Background()

	a, ok := schemaAttributeAtPath(s, names)
	if !ok {
		t.Fatalf("no attribute at %v", names)
	}
	p := path.Root(names[0])
	for _, name := range names[1:] {
		p = p.AtName(name)
	}

	var errs []string
	collect := func(summary, detail string) { errs = append(errs, summary+": "+detail) }
	switch a := a.(type) {
	case schema.StringAttribute:
		for _, v := range a.Validators {
			var resp validator.StringResponse
			v.ValidateString(ctx, validator.StringRequest{Path: p, ConfigValue: value.(types.String)}, &resp)
			for _, d := range resp.Diagnostics.Errors() {
				collect(d.Summary(), d.Detail())
			}
		}
	case schema.ListAttribute:
		for _, v := range a.Validators {
			var resp validator.ListResponse
			v.ValidateList(ctx, validator.ListRequest{Path: p, ConfigValue: value.(types.List)}, &resp)
			for _, d := range resp.Diagnostics.Errors() {
				collect(d.Summary(), d.Detail())
			}
		}
	default:
		t.Fatalf("attribute at %v is a %T", names, a)
	}

	return errs
}

// attributeValidationCase is a value of an attribute and whether its
// validators accept it.
type attributeValidationCase struct {
	names []string
	value attr.Value
	valid bool
}

func runAttributeValidationCases(t *testing.T, s schema.Schema, cases []attributeValidationCase) {
	t.Helper()
	for _, c := range cases {
		errs := attributeValidationErrors(t, s, c.names, c.value)
		if c.valid && len(errs) > 0 {
			t.Errorf("%v = %s: unexpected errors: %v", c.names, c.value, errs)
		}
		if !c.valid && len(errs) == 0 {
			t.Errorf("%v = %s: expected an error", c.names, c.value)
		}
	}
}
