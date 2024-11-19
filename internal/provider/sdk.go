package provider

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
)

type SDKIdentifier struct {
	Path  path.Path
	Title string
}

func (i *SDKIdentifier) FromString(id types.String) (int64, diag.Diagnostics) {
	if id.IsUnknown() {
		return 0, diag.Diagnostics{
			diag.NewAttributeErrorDiagnostic(
				i.Path,
				"Unknown "+i.Title,
				"", // TODO
			),
		}
	}

	if id.IsNull() {
		return 0, diag.Diagnostics{
			diag.NewAttributeErrorDiagnostic(
				i.Path,
				"Missing "+i.Title,
				"", // TODO
			),
		}
	}

	val, err := strconv.ParseInt(id.ValueString(), 10, 64)
	if err != nil {
		return 0, diag.Diagnostics{
			diag.NewAttributeErrorDiagnostic(
				i.Path,
				"Invalid "+i.Title,
				"Value must be numeric, but was not: "+err.Error(),
			),
		}
	}

	return val, nil
}

func (i *SDKIdentifier) IntoString(id int64) types.String {
	return types.StringValue(fmt.Sprintf("%d", id))
}

func SDKKeyValuesFromMap(m types.Map) []checkly.KeyValue {
	if m.IsNull() {
		return nil
	}

	var values []checkly.KeyValue
	for key, val := range m.Elements() {
		values = append(values, checkly.KeyValue{
			Key:   key,
			Value: val.(types.String).ValueString(),
		})
	}

	return values
}

func SDKKeyValuesIntoMap(values *[]checkly.KeyValue) types.Map {
	if values == nil {
		return types.MapNull(types.StringType)
	}

	mapValues := make(map[string]attr.Value, len(*values))
	for _, kv := range *values {
		mapValues[kv.Key] = types.StringValue(kv.Value)
	}

	return types.MapValueMust(types.StringType, mapValues)
}
