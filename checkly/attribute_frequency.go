package checkly

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const frequencyAttributeName = "frequency"

type FrequencyAttributeSchemaOptions struct {
	Monitor            bool
	AllowHighFrequency bool
	Disclaimer         string
}

func makeFrequencyAttributeSchema(options FrequencyAttributeSchemaOptions) *schema.Schema {
	name := "check"
	if options.Monitor {
		name = "monitor"
	}

	allow := allowedValues[int]{
		{
			Value:       0,
			Description: "high frequency - use `frequency_offset` to define the actual frequency",
		},
		{
			Value:       1,
			Description: "1 minute",
		},
		{
			Value:       2,
			Description: "2 minutes",
		},
		{
			Value:       5,
			Description: "5 minutes",
		},
		{
			Value:       10,
			Description: "10 minutes",
		},
		{
			Value:       15,
			Description: "15 minutes",
		},
		{
			Value:       30,
			Description: "30 minutes",
		},
		{
			Value:       60,
			Description: "1 hour",
		},
		{
			Value:       120,
			Description: "2 hours",
		},
		{
			Value:       180,
			Description: "3 hours",
		},
		{
			Value:       360,
			Description: "6 hours",
		},
		{
			Value:       720,
			Description: "12 hours",
		},
		{
			Value:       1440,
			Description: "24 hours",
		},
	}

	if !options.AllowHighFrequency {
		allow = allow[1:]
	}

	var disclaimer string
	if options.Disclaimer != "" {
		disclaimer = options.Disclaimer + " "
	}

	return &schema.Schema{
		Description:  disclaimer + fmt.Sprintf("Controls how often the %s should run. Defined in minutes. %s", name, allow.String()),
		Type:         schema.TypeInt,
		Required:     true,
		ValidateFunc: validateOneOf(allow.Values()),
	}
}
