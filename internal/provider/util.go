package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func IntoUntypedStringSet(slice *[]string) types.Set {
	if slice == nil {
		return types.SetNull(types.StringType)
	}

	var values []attr.Value
	for _, value := range *slice {
		values = append(values, types.StringValue(value))
	}

	return types.SetValueMust(types.StringType, values)
}

func FromUntypedStringSet(set types.Set) []string {
	if set.IsNull() {
		return nil
	}

	var slice []string
	for _, el := range set.Elements() {
		slice = append(slice, el.(types.String).ValueString())
	}

	return slice
}
