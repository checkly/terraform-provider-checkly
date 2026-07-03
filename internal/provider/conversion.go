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
// attributes of resources that have not been created yet.
func valueContainsUnknown(value attr.Value) bool {
	if value.IsUnknown() {
		return true
	}

	if value.IsNull() {
		return false
	}

	switch v := value.(type) {
	case types.Object:
		for _, nested := range v.Attributes() {
			if valueContainsUnknown(nested) {
				return true
			}
		}
	case types.List:
		for _, nested := range v.Elements() {
			if valueContainsUnknown(nested) {
				return true
			}
		}
	case types.Set:
		for _, nested := range v.Elements() {
			if valueContainsUnknown(nested) {
				return true
			}
		}
	case types.Map:
		for _, nested := range v.Elements() {
			if valueContainsUnknown(nested) {
				return true
			}
		}
	case types.Tuple:
		for _, nested := range v.Elements() {
			if valueContainsUnknown(nested) {
				return true
			}
		}
	}

	return false
}
