package provider

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
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
		values []string
		prior  types.Set
		want   types.Set
	}{
		"empty with a null prior":     {nil, null, null},
		"empty with an unknown prior": {nil, unknown, null},
		"empty with an empty prior":   {nil, empty, empty},
		"values":                      {[]string{"a"}, null, one},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			var diags diag.Diagnostics
			if got := stringSetFromAPINullIfUnset(ctx, c.values, c.prior, &diags); !got.Equal(c.want) {
				t.Errorf("stringSetFromAPINullIfUnset = %s, want %s", got, c.want)
			}
			if diags.HasError() {
				t.Fatal(diags)
			}
		})
	}
}

func TestKeyValuesFromStringMap(t *testing.T) {
	ctx := context.Background()

	cases := map[string]struct {
		m    types.Map
		want []checkly.KeyValue
	}{
		"null":    {types.MapNull(types.StringType), []checkly.KeyValue{}},
		"unknown": {types.MapUnknown(types.StringType), []checkly.KeyValue{}},
		"sorted by key": {
			types.MapValueMust(types.StringType, map[string]attr.Value{"b": types.StringValue("2"), "a": types.StringValue("1")}),
			[]checkly.KeyValue{{Key: "a", Value: "1"}, {Key: "b", Value: "2"}},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			var diags diag.Diagnostics
			got := keyValuesFromStringMap(ctx, c.m, &diags)
			if diags.HasError() {
				t.Fatal(diags)
			}
			// The API stores an empty slice as no entries, so it must not
			// become nil.
			if got == nil || !reflect.DeepEqual(got, c.want) {
				t.Errorf("keyValuesFromStringMap = %#v, want %#v", got, c.want)
			}
		})
	}
}

func TestKnownPriorAttribute(t *testing.T) {
	attrTypes := map[string]attr.Type{"b": types.BoolType}
	null := types.BoolNull()

	cases := map[string]struct {
		prior types.Object
		want  types.Bool
	}{
		"null prior":    {types.ObjectNull(attrTypes), null},
		"unknown prior": {types.ObjectUnknown(attrTypes), null},
		"unknown value": {types.ObjectValueMust(attrTypes, map[string]attr.Value{"b": types.BoolUnknown()}), null},
		"known value":   {types.ObjectValueMust(attrTypes, map[string]attr.Value{"b": types.BoolValue(true)}), types.BoolValue(true)},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if got := knownPriorAttribute(c.prior, "b", null); !got.Equal(c.want) {
				t.Errorf("knownPriorAttribute = %s, want %s", got, c.want)
			}
		})
	}
}
