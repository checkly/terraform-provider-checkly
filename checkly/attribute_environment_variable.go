package checkly

import (
	"errors"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const environmentVariableAttributeName = "environment_variable"

type EnvironmentVariableAttributeSchemaOptions struct {
	Description string
}

func makeEnvironmentVariableAttributeSchema(options EnvironmentVariableAttributeSchemaOptions) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		Description: options.Description,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"key": {
					Description: "The name of the environment variable or secret.",
					Type:        schema.TypeString,
					Required:    true,
				},
				"value": {
					Description: "The value of the environment variable or secret.",
					Type:        schema.TypeString,
					Required:    true,
				},
				"locked": {
					Description: "If true, the value is not shown by default, but it can be accessed. (Default `false`).",
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
				},
				"secret": {
					Description: "If true, the value will never be visible. (Default `false`).",
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
				},
			},
		},
	}
}

var environmentVariableAttributeSchema = &schema.Schema{
	Description: "Create and resolve an incident based on the alert configuration. Useful for status page automation.",
	Type:        schema.TypeSet,
	MaxItems:    1,
	Optional:    true,
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			"service_id": {
				Description: "The status page service that this incident will be associated with.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"severity": {
				Description:  "The severity level of the incident. Possible values are `MINOR`, `MEDIUM`, `MAJOR`, and `CRITICAL`.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateOneOf([]string{"MINOR", "MEDIUM", "MAJOR", "CRITICAL"}),
			},
			"name": {
				Description: "The name of the incident.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"description": {
				Description: "A detailed description of the incident.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"notify_subscribers": {
				Description: "Whether to notify subscribers when the incident is triggered.",
				Type:        schema.TypeBool,
				Required:    true,
			},
		},
	},
}

func environmentVariablesFromList(s []any) []checkly.EnvironmentVariable {
	res := []checkly.EnvironmentVariable{}
	if len(s) == 0 {
		return res
	}

	for _, ev := range s {
		tm := ev.(tfMap)
		key := tm["key"].(string)
		value := tm["value"].(string)
		locked := tm["locked"].(bool)
		secret := tm["secret"].(bool)

		res = append(res, checkly.EnvironmentVariable{
			Key:    key,
			Value:  value,
			Locked: locked,
			Secret: secret,
		})
	}

	return res
}

func listFromEnvironmentVariables(ev []checkly.EnvironmentVariable) []tfMap {
	if ev == nil {
		return []tfMap{}
	}

	res := []tfMap{}
	for _, v := range ev {
		res = append(res, tfMap{
			"key":    v.Key,
			"value":  v.Value,
			"locked": v.Locked,
			"secret": v.Secret,
		})
	}

	return res
}

func environmentVariablesFromResourceData(d *schema.ResourceData) ([]checkly.EnvironmentVariable, error) {
	val, ok := d.GetOk(environmentVariableAttributeName)
	if !ok {
		return nil, nil
	}

	vars := environmentVariablesFromList(val.([]any))

	return vars, nil
}

func compatEnvironmentVariablesFromResourceData(d *schema.ResourceData) ([]checkly.EnvironmentVariable, error) {
	oldVars, err := deprecatedEnvironmentVariablesFromResourceData(d)
	if err != nil {
		return nil, err
	}

	newVars, err := environmentVariablesFromResourceData(d)
	if err != nil {
		return nil, err
	}

	switch {
	case len(newVars) > 0 && len(oldVars) > 0:
		return nil, errors.New(`must not use the deprecated "environment_variables" attribute together with the "environment_variable" attribute`)
	case len(oldVars) > 0:
		return oldVars, nil
	default:
		return newVars, nil
	}
}

func updateCompatEnvironmentVariablesResourceData(d *schema.ResourceData, evs []checkly.EnvironmentVariable) error {
	oldVars, err := deprecatedEnvironmentVariablesFromResourceData(d)
	if err != nil {
		return err
	}

	switch {
	// If the resource defines old-style environment variables, update those
	// and clear the new-style variables.
	case len(oldVars) > 0:
		err := d.Set(deprecatedEnvironmentVariablesAttributeName, mapFromDeprecatedEnvironmentVariables(evs))
		if err != nil {
			return err
		}

		err = d.Set(environmentVariableAttributeName, []tfMap{})
		if err != nil {
			return err
		}

		return nil
	// Otherwise, always update new-style variables and clear the old-style
	// variables.
	default:
		err := d.Set(environmentVariableAttributeName, listFromEnvironmentVariables(evs))
		if err != nil {
			return err
		}

		// Clear old-style variables.
		err = d.Set(deprecatedEnvironmentVariablesAttributeName, tfMap{})
		if err != nil {
			return err
		}

		return nil
	}
}
