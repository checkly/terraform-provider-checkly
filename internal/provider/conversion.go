package provider

import (
	"context"
	"maps"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

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

// stringSetFromAPINullIfUnset converts a string slice from the API to a set
// of strings, null when empty and unset before (see setNullIfUnset).
func stringSetFromAPINullIfUnset(ctx context.Context, values []string, prior types.Set, diags *diag.Diagnostics) types.Set {
	return setNullIfUnset(ctx, setFromStrings(ctx, values, diags), prior)
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

// stringListFromAPINullIfUnset converts a string slice from the API to a list
// of strings. The API reports an unset collection as an empty array, so an
// empty value converts to null when the prior value (planned or previously
// stored) is null or unknown. An optional attribute absent from the
// configuration thus stays null instead of drifting to an empty list.
func stringListFromAPINullIfUnset(ctx context.Context, values []string, prior types.List, diags *diag.Diagnostics) types.List {
	if len(values) == 0 && (prior.IsNull() || prior.IsUnknown()) {
		return types.ListNull(types.StringType)
	}

	return listFromStrings(ctx, values, diags)
}

// setNullIfUnset returns a null set in place of an empty set when the prior
// value (planned or previously stored) is null or unknown. The API reports an
// unset collection as empty, so an optional attribute absent from the
// configuration thus stays null instead of drifting to an empty set.
func setNullIfUnset(ctx context.Context, set types.Set, prior types.Set) types.Set {
	if len(set.Elements()) == 0 && (prior.IsNull() || prior.IsUnknown()) {
		return types.SetNull(set.ElementType(ctx))
	}

	return set
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

// stringMapFromAPINullIfUnset converts the API's key-value pair
// representation to a map of strings. No pairs convert to null when prior is
// null or unknown, by the same rule as setNullIfUnset.
func stringMapFromAPINullIfUnset(ctx context.Context, kvs []checkly.KeyValue, prior types.Map, diags *diag.Diagnostics) types.Map {
	if len(kvs) == 0 && (prior.IsNull() || prior.IsUnknown()) {
		return types.MapNull(types.StringType)
	}

	values := map[string]string{}
	for _, kv := range kvs {
		values[kv.Key] = kv.Value
	}

	m, d := types.MapValueFrom(ctx, types.StringType, values)
	diags.Append(d...)

	return m
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
