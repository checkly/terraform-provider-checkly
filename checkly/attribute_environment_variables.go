package checkly

import (
	"fmt"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const deprecatedEnvironmentVariablesAttributeName = "environment_variables"

type DeprecatedEnvironmentVariablesAttributeSchemaOptions struct {
}

func makeDeprecatedEnvironmentVariablesAttributeSchema(options DeprecatedEnvironmentVariablesAttributeSchemaOptions) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeMap,
		Optional:    true,
		Deprecated:  "This attribute is deprecated and will be removed in a future version. Use the `environment_variable` attribute instead.",
		Description: "Key/value pairs of environment variables to insert into the runtime environment.",
	}
}

func mapFromDeprecatedEnvironmentVariables(evs []checkly.EnvironmentVariable) tfMap {
	var s = tfMap{}
	for _, ev := range evs {
		s[ev.Key] = ev.Value
	}
	return s
}

func deprecatedEnvironmentVariablesFromMap(m map[string]any) []checkly.EnvironmentVariable {
	r := make([]checkly.EnvironmentVariable, 0, len(m))
	for k, v := range m {
		s, ok := v.(string)
		if !ok {
			panic(fmt.Errorf("could not convert environment variable value %v to string", v))
		}

		r = append(r, checkly.EnvironmentVariable{
			Key:   k,
			Value: s,
		})

	}
	return r
}

func deprecatedEnvironmentVariablesFromResourceData(d *schema.ResourceData) ([]checkly.EnvironmentVariable, error) {
	val, ok := d.GetOk(deprecatedEnvironmentVariablesAttributeName)
	if !ok {
		return nil, nil
	}

	vars := deprecatedEnvironmentVariablesFromMap(val.(tfMap))

	return vars, nil
}
