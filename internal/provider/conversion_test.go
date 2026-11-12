package provider

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

func TestInt64ValueOrNull(t *testing.T) {
	if got := int64ValueOrNull(int64(0)); !got.IsNull() {
		t.Errorf("int64ValueOrNull(int64(0)) = %s, want null", got)
	}
	if got := int64ValueOrNull(0); !got.IsNull() {
		t.Errorf("int64ValueOrNull(0) = %s, want null", got)
	}
	if got := int64ValueOrNull(int64(5)); !got.Equal(types.Int64Value(5)) {
		t.Errorf("int64ValueOrNull(int64(5)) = %s, want 5", got)
	}
	if got := int64ValueOrNull(3); !got.Equal(types.Int64Value(3)) {
		t.Errorf("int64ValueOrNull(3) = %s, want 3", got)
	}
}

func TestGroupOrderFromAPI(t *testing.T) {
	null := types.Int64Null()
	zero := types.Int64Value(0)
	five := types.Int64Value(5)
	unknown := types.Int64Unknown()

	cases := []struct {
		prior types.Int64
		value *int
		want  types.Int64
	}{
		{null, nil, null},
		{zero, nil, null},
		{five, nil, null},
		{unknown, nil, null},
		// A reported 0 stays null when nothing was stored, as after an
		// upgrade from a provider version that sent 0 for no position.
		{null, new(0), null},
		{zero, new(0), zero},
		{five, new(0), zero},
		{unknown, new(0), zero},
		{null, new(5), five},
		{zero, new(5), five},
		{five, new(5), five},
		{unknown, new(5), five},
	}
	for _, c := range cases {
		if got := groupOrderFromAPI(c.value, c.prior); !got.Equal(c.want) {
			value := "nil"
			if c.value != nil {
				value = fmt.Sprint(*c.value)
			}
			t.Errorf("groupOrderFromAPI(%s, %s) = %s, want %s", value, c.prior, got, c.want)
		}
	}
}

func TestGroupOrderToAPI(t *testing.T) {
	if got := groupOrderToAPI(types.Int64Null()); got != nil {
		t.Errorf("groupOrderToAPI(null) = %d, want nil", *got)
	}
	if got := groupOrderToAPI(types.Int64Unknown()); got != nil {
		t.Errorf("groupOrderToAPI(unknown) = %d, want nil", *got)
	}
	for _, v := range []int64{0, 5} {
		if got := groupOrderToAPI(types.Int64Value(v)); got == nil || *got != int(v) {
			t.Errorf("groupOrderToAPI(%d) = %v, want %d", v, got, v)
		}
	}
}

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

func TestSetFromStrings(t *testing.T) {
	ctx := context.Background()

	empty := types.SetValueMust(types.StringType, []attr.Value{})
	one := types.SetValueMust(types.StringType, []attr.Value{types.StringValue("a")})

	cases := map[string]struct {
		values []string
		want   types.Set
	}{
		"nil":    {nil, empty},
		"empty":  {[]string{}, empty},
		"values": {[]string{"a"}, one},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			var diags diag.Diagnostics
			if got := setFromStrings(ctx, c.values, &diags); !got.Equal(c.want) {
				t.Errorf("setFromStrings = %s, want %s", got, c.want)
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

func TestNullValueOf(t *testing.T) {
	ctx := context.Background()
	objectAttrTypes := map[string]attr.Type{"enabled": types.BoolType}

	cases := map[string]struct {
		typ  attr.Type
		want attr.Value
	}{
		"bool":   {types.BoolType, types.BoolNull()},
		"string": {types.StringType, types.StringNull()},
		"int64":  {types.Int64Type, types.Int64Null()},
		"object": {types.ObjectType{AttrTypes: objectAttrTypes}, types.ObjectNull(objectAttrTypes)},
		"set":    {types.SetType{ElemType: types.StringType}, types.SetNull(types.StringType)},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := nullValueOf(tc.typ); !got.Equal(tc.want) {
				t.Errorf("nullValueOf(%s) = %#v, want %#v", tc.typ, got, tc.want)
			}
		})
	}

	// Every member type of the objects whose null members are built with
	// nullValueOf.
	objects := map[string]map[string]attr.Type{
		"enforce_alert_settings":      checkGroupV2EnforceAlertSettingsAttrTypes,
		"enforce_locations":           checkGroupV2EnforceLocationsAttrTypes,
		"enforce_retry_strategy":      checkGroupV2EnforceRetryStrategyAttrTypes,
		"enforce_scheduling_strategy": checkGroupV2EnforceSchedulingStrategyAttrTypes,
		"alert_settings":              alertSettingsAttrTypes,
	}
	for objectName, attrTypes := range objects {
		for name, typ := range attrTypes {
			t.Run(objectName+"/"+name, func(t *testing.T) {
				got := nullValueOf(typ)
				if !got.IsNull() || !got.Type(ctx).Equal(typ) {
					t.Errorf("nullValueOf(%s) = %#v, want a null value of that type", typ, got)
				}
			})
		}
	}
}
