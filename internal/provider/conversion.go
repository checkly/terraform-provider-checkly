package provider

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/checkly/checkly-go-sdk/v2"
)

// stringsFromSet converts a set of strings to a Go slice. Null and unknown
// sets convert to nil.
func stringsFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) []string {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}

	var values []string
	diags.Append(set.ElementsAs(ctx, &values, false)...)

	return values
}

// stringsFromSetOrEmpty is stringsFromSet for API properties that must
// serialize as an empty JSON array rather than null: null and unknown sets
// convert to an empty slice.
func stringsFromSetOrEmpty(ctx context.Context, set types.Set, diags *diag.Diagnostics) []string {
	values := stringsFromSet(ctx, set, diags)
	if values == nil {
		return []string{}
	}

	return values
}

// setFromStrings converts a Go slice to a set of strings. A nil slice
// converts to an empty set.
func setFromStrings(ctx context.Context, values []string, diags *diag.Diagnostics) types.Set {
	if values == nil {
		values = []string{}
	}

	set, d := types.SetValueFrom(ctx, types.StringType, values)
	diags.Append(d...)

	return set
}

// stringsFromList converts a list of strings to a Go slice. Null and unknown
// lists convert to nil.
func stringsFromList(ctx context.Context, list types.List, diags *diag.Diagnostics) []string {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}

	var values []string
	diags.Append(list.ElementsAs(ctx, &values, false)...)

	return values
}

// listFromStrings converts a Go slice to a list of strings. A nil slice
// converts to an empty list.
func listFromStrings(ctx context.Context, values []string, diags *diag.Diagnostics) types.List {
	if values == nil {
		values = []string{}
	}

	list, d := types.ListValueFrom(ctx, types.StringType, values)
	diags.Append(d...)

	return list
}

// keyValuesFromStringMap converts a map of strings to the API's key-value
// pair representation, sorted by key so that requests are deterministic.
// Null and unknown maps convert to an empty, non-nil slice, which the API
// stores as no entries; payloads that leave out empty fields treat it like
// nil.
func keyValuesFromStringMap(ctx context.Context, m types.Map, diags *diag.Diagnostics) []checkly.KeyValue {
	kvs := []checkly.KeyValue{}

	if m.IsNull() || m.IsUnknown() {
		return kvs
	}

	var values map[string]string
	diags.Append(m.ElementsAs(ctx, &values, false)...)
	if diags.HasError() {
		return kvs
	}

	for _, key := range slices.Sorted(maps.Keys(values)) {
		kvs = append(kvs, checkly.KeyValue{
			Key:   key,
			Value: values[key],
		})
	}

	return kvs
}

// stringMapFromKeyValues converts the API's key-value pair representation
// to a map of strings. No pairs convert to an empty map.
func stringMapFromKeyValues(ctx context.Context, kvs []checkly.KeyValue, diags *diag.Diagnostics) types.Map {
	values := map[string]string{}
	for _, kv := range kvs {
		values[kv.Key] = kv.Value
	}

	m, d := types.MapValueFrom(ctx, types.StringType, values)
	diags.Append(d...)

	return m
}

// Optional collection attributes are computed with an empty default, so that
// a collection the configuration leaves out is stored as empty rather than
// null, the same way the API reports it. Expressions such as length() then
// work on it without a null check, and an import plans no change for it.

// emptySetDefault returns the schema default of an optional set attribute:
// an empty set of elemType.
func emptySetDefault(elemType attr.Type) defaults.Set {
	return setdefault.StaticValue(types.SetValueMust(elemType, []attr.Value{}))
}

// emptyListDefault returns the schema default of an optional list
// attribute: an empty list of elemType.
func emptyListDefault(elemType attr.Type) defaults.List {
	return listdefault.StaticValue(types.ListValueMust(elemType, []attr.Value{}))
}

// emptyMapDefault returns the schema default of an optional map attribute:
// an empty map of elemType.
func emptyMapDefault(elemType attr.Type) defaults.Map {
	return mapdefault.StaticValue(types.MapValueMust(elemType, map[string]attr.Value{}))
}

// emptySetIfNull returns an empty set of the element type of set in place of
// a null set, and set itself otherwise. It suits values carried over from a
// prior or source value that may be null, such as when nothing is stored yet,
// into an attribute that stores an empty set when unset.
func emptySetIfNull(ctx context.Context, set types.Set) types.Set {
	if set.IsNull() {
		return types.SetValueMust(set.ElementType(ctx), []attr.Value{})
	}

	return set
}

// emptyListIfNull is emptySetIfNull for lists.
func emptyListIfNull(ctx context.Context, list types.List) types.List {
	if list.IsNull() {
		return types.ListValueMust(list.ElementType(ctx), []attr.Value{})
	}

	return list
}

// priorAttribute returns the attribute called name of prior, the planned or
// previously stored value of an object attribute, or null when prior is null
// or unknown.
func priorAttribute[T attr.Value](prior types.Object, name string, null T) T {
	if !prior.IsNull() && !prior.IsUnknown() {
		if value, ok := prior.Attributes()[name].(T); ok {
			return value
		}
	}

	return null
}

// knownPriorAttribute is priorAttribute, except that an unknown attribute
// value also resolves to null. It suits values carried over from prior into
// state, which cannot hold unknown values.
func knownPriorAttribute[T attr.Value](prior types.Object, name string, null T) T {
	if value := priorAttribute(prior, name, null); !value.IsUnknown() {
		return value
	}

	return null
}

// optionalStringPointer converts an optional string value to a *string,
// returning nil for null and unknown values.
func optionalStringPointer(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	return value.ValueStringPointer()
}

// stringValueOrNull converts a possibly-empty string from the API to a
// string value, mapping the empty string to null.
func stringValueOrNull(value string) types.String {
	if value == "" {
		return types.StringNull()
	}

	return types.StringValue(value)
}

// int64ValueOrNull converts a reference or a position from the API, where 0
// stands for none, to an int64 value, mapping 0 to null.
func int64ValueOrNull[T ~int | ~int64](value T) types.Int64 {
	if value == 0 {
		return types.Int64Null()
	}

	return types.Int64Value(int64(value))
}

// groupOrderToAPI converts the group position of a check or monitor to its
// API representation: nil for no position, which clears a stored one.
func groupOrderToAPI(value types.Int64) *int {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	order := int(value.ValueInt64())
	return &order
}

// groupOrderFromAPI converts the group position the API reports for a check
// or monitor: no position is null and any other position is that number,
// except that a reported 0 stays null when the prior value is null.
//
// Provider versions before v2.0.0 sent position 0 for every API, browser and
// multi-step check and TCP monitor they created, grouped or not, and their
// state, which held 0 for an unset position, upgrades to null. Reading those
// positions faithfully would plan a change from 0 to null for each of these
// checks after the upgrade although nothing is configured. A configured 0
// reads back as 0, and a 0 that replaces another prior position shows as a
// change.
func groupOrderFromAPI(value *int, prior types.Int64) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}
	if *value == 0 && prior.IsNull() {
		return prior
	}

	return types.Int64Value(int64(*value))
}

// stringValueOrPriorEmpty converts an optional string from the API like
// stringValueOrNull, except that an empty string stays empty when the prior
// value is empty. The API reports "" for an unset field, so an attribute
// configured as "" would otherwise refresh to null and differ from its
// configuration.
func stringValueOrPriorEmpty(value string, prior types.String) types.String {
	if value == "" && !prior.IsNull() && !prior.IsUnknown() && prior.ValueString() == "" {
		return prior
	}

	return stringValueOrNull(value)
}

// stringOrNullWhenUnknown resolves an unknown value to null. Useful when an
// optional+computed value turns out to have no value tracked by the API.
func stringOrNullWhenUnknown(value types.String) types.String {
	if value.IsUnknown() {
		return types.StringNull()
	}

	return value
}

// valueContainsUnknown reports whether the value, or any value nested within
// it, is unknown. Configuration values are unknown when they reference
// attributes of resources that have not been created yet. A value that cannot
// be converted to its Terraform representation is reported as unknown, so
// that callers err on the side of leaving the plan untouched.
func valueContainsUnknown(ctx context.Context, value attr.Value) bool {
	tfValue, err := value.ToTerraformValue(ctx)
	if err != nil {
		return true
	}

	return !tfValue.IsFullyKnown()
}

// nullValueOf returns the null value of an attribute type. It panics when the
// type cannot produce a null value, which only a malformed schema type can
// cause.
func nullValueOf(t attr.Type) attr.Value {
	ctx := context.Background()

	value, err := t.ValueFromTerraform(ctx, tftypes.NewValue(t.TerraformType(ctx), nil))
	if err != nil {
		panic(fmt.Sprintf("cannot build a null value of type %s: %s", t, err))
	}

	return value
}
