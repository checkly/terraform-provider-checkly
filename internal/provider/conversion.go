package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

// stringSetFromAPINullIfUnset converts a string slice from the API to a set of strings.
// The API reports an unset collection as an empty array, so an empty value
// converts to null when the prior value (planned or previously stored) is
// null or unknown. An optional attribute absent from the configuration thus
// stays null instead of drifting to an empty set.
func stringSetFromAPINullIfUnset(ctx context.Context, values []string, prior types.Set, diags *diag.Diagnostics) types.Set {
	if len(values) == 0 && (prior.IsNull() || prior.IsUnknown()) {
		return types.SetNull(types.StringType)
	}

	return setFromStrings(ctx, values, diags)
}

// stringSetFromAPIEmptyIfUnknown converts a string slice from the API to a
// set of strings. An empty value keeps the prior value when that is null, so
// that an omitted collection does not cause drift against an empty API
// response. Unlike stringSetFromAPINullIfUnset, an unknown prior value converts to an
// empty set.
func stringSetFromAPIEmptyIfUnknown(ctx context.Context, values []string, prior types.Set, diags *diag.Diagnostics) types.Set {
	if len(values) == 0 && prior.IsNull() {
		return prior
	}

	return setFromStrings(ctx, values, diags)
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

// int64OrNullWhenUnknown resolves an unknown value to null. Useful when an
// optional+computed value turns out to have no value tracked by the API.
func int64OrNullWhenUnknown(value types.Int64) types.Int64 {
	if value.IsUnknown() {
		return types.Int64Null()
	}

	return value
}

// boolOrNullWhenUnknown resolves an unknown value to null. Useful when an
// optional+computed value turns out to have no value tracked by the API.
func boolOrNullWhenUnknown(value types.Bool) types.Bool {
	if value.IsUnknown() {
		return types.BoolNull()
	}

	return value
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
