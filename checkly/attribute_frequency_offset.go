package checkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const frequencyOffsetAttributeName = "frequency_offset"

type FrequencyOffsetAttributeSchemaOptions struct {
	Monitor    bool
	Disclaimer string
}

func makeFrequencyOffsetAttributeSchema(options FrequencyOffsetAttributeSchemaOptions) *schema.Schema {
	name := "check"
	if options.Monitor {
		name = "monitor"
	}

	allow := allowedValues[int]{
		{
			Value:       0,
			Description: "not set - useful with expressions",
		},
		{
			Value:       10,
			Description: "10 seconds",
		},
		{
			Value:       20,
			Description: "20 seconds",
		},
		{
			Value:       30,
			Description: "30 seconds",
		},
	}

	var disclaimer string
	if options.Disclaimer != "" {
		disclaimer = options.Disclaimer + " "
	}

	return &schema.Schema{
		Description:  disclaimer + fmt.Sprintf("When `frequency` is `0` (high frequency), `frequency_offset` is required and it alone controls how often the %s should run. Defined in seconds. %s", name, allow.String()),
		Type:         schema.TypeInt,
		Optional:     true,
		ValidateFunc: validateOneOf(allow.Values()),
	}
}

func FrequencyOffsetCustomizeDiff(_ context.Context, diff *schema.ResourceDiff, meta any) error {
	frequency := diff.Get(frequencyAttributeName).(int)
	offset := diff.Get(frequencyOffsetAttributeName).(int)

	switch {
	case frequency > 0 && offset != 0:
		return fmt.Errorf(`"frequency_offset" can only be set when "frequency" is 0`)
	case frequency == 0 && offset == 0:
		return fmt.Errorf(`"frequency_offset" is required when "frequency" is 0`)
	}

	return nil
}
