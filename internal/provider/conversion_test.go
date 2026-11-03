package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValueContainsUnknown(t *testing.T) {
	ctx := context.Background()

	elemAttrTypes := map[string]attr.Type{"name": types.StringType}
	elemType := types.ObjectType{AttrTypes: elemAttrTypes}
	setWith := func(name types.String) types.Set {
		return types.SetValueMust(elemType, []attr.Value{
			types.ObjectValueMust(elemAttrTypes, map[string]attr.Value{"name": name}),
		})
	}

	cases := map[string]struct {
		value attr.Value
		want  bool
	}{
		"null object":                 {types.ObjectNull(elemAttrTypes), false},
		"unknown object":              {types.ObjectUnknown(elemAttrTypes), true},
		"known set":                   {setWith(types.StringValue("a")), false},
		"unknown inside a set object": {setWith(types.StringUnknown()), true},
		"null inside a set object":    {setWith(types.StringNull()), false},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if got := valueContainsUnknown(ctx, c.value); got != c.want {
				t.Errorf("valueContainsUnknown = %v, want %v", got, c.want)
			}
		})
	}
}

func TestStringSetFromAPI(t *testing.T) {
	ctx := context.Background()

	null := types.SetNull(types.StringType)
	unknown := types.SetUnknown(types.StringType)
	empty := types.SetValueMust(types.StringType, []attr.Value{})
	one := types.SetValueMust(types.StringType, []attr.Value{types.StringValue("a")})

	cases := map[string]struct {
		values                      []string
		prior                       types.Set
		nullIfUnset, emptyIfUnknown types.Set
	}{
		"empty with a null prior":     {nil, null, null, null},
		"empty with an unknown prior": {nil, unknown, null, empty},
		"empty with an empty prior":   {nil, empty, empty, empty},
		"values":                      {[]string{"a"}, null, one, one},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			var diags diag.Diagnostics
			if got := stringSetFromAPINullIfUnset(ctx, c.values, c.prior, &diags); !got.Equal(c.nullIfUnset) {
				t.Errorf("stringSetFromAPINullIfUnset = %s, want %s", got, c.nullIfUnset)
			}
			if got := stringSetFromAPIEmptyIfUnknown(ctx, c.values, c.prior, &diags); !got.Equal(c.emptyIfUnknown) {
				t.Errorf("stringSetFromAPIEmptyIfUnknown = %s, want %s", got, c.emptyIfUnknown)
			}
			if diags.HasError() {
				t.Fatal(diags)
			}
		})
	}
}
