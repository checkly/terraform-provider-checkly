package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	frequencyAttributeName       = "frequency"
	frequencyOffsetAttributeName = "frequency_offset"
)

// defaultFrequency is the frequency, in minutes, that applies when the
// frequency attribute is not configured. It matches the Checkly API default.
const defaultFrequency int64 = 10

// FrequencyAttributeSchemaOptions customizes the frequency attribute schema
// for a specific resource.
type FrequencyAttributeSchemaOptions struct {
	// Subject names what the resource schedules: subjectCheck or
	// subjectMonitor.
	Subject CheckSubject
	// AllowHighFrequency permits the value 0, which enables sub-minute
	// scheduling controlled by the frequency_offset attribute.
	AllowHighFrequency bool
}

// makeFrequencyAttributeSchema returns the schema for the frequency
// attribute, which defaults to defaultFrequency.
func makeFrequencyAttributeSchema(options FrequencyAttributeSchemaOptions) schema.Int64Attribute {
	allow := allowedValues[int64]{
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

	return schema.Int64Attribute{
		Optional:    true,
		Computed:    true,
		Default:     int64default.StaticInt64(defaultFrequency),
		Description: fmt.Sprintf("Controls how often the %s should run. Defined in minutes. %s (Default `%d`).", options.Subject, allow.String(), defaultFrequency),
		Validators: []validator.Int64{
			int64validator.OneOf(allow.Values()...),
		},
	}
}

// FrequencyOffsetAttributeSchemaOptions customizes the frequency_offset
// attribute schema for a specific resource.
type FrequencyOffsetAttributeSchemaOptions struct {
	// Subject names what the resource schedules: subjectCheck or
	// subjectMonitor.
	Subject CheckSubject
	// Disclaimer is prepended to the attribute description.
	Disclaimer string
}

// makeFrequencyOffsetAttributeSchema returns the schema for the
// frequency_offset attribute.
func makeFrequencyOffsetAttributeSchema(options FrequencyOffsetAttributeSchemaOptions) schema.Int64Attribute {
	allow := allowedValues[int64]{
		{
			Value:       0,
			Description: "disabled - use `frequency` to define the actual frequency",
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

	return schema.Int64Attribute{
		Optional:    true,
		Description: disclaimer + fmt.Sprintf("When `frequency` is `0` (high frequency), `frequency_offset` is required and it alone controls how often the %s should run. Defined in seconds. %s", options.Subject, allow.String()),
		Validators: []validator.Int64{
			int64validator.OneOf(allow.Values()...),
		},
	}
}

// frequencyOffsetFromAPI resolves the frequency_offset attribute from an API
// response, for every resource that has the attribute. The configured offset
// only takes effect with high frequency scheduling (frequency 0); for a regular
// frequency the API assigns an offset of its own to spread the runs, so the API
// value is used only for frequency 0. For a regular frequency a prior (planned
// or previously stored) value of 0, which is equivalent to unset, or null is
// preserved; any other value is cleared, so that a frequency changed outside
// Terraform also shows the configured offset as a change.
func frequencyOffsetFromAPI(frequency, offset int, prior types.Int64) types.Int64 {
	if frequency == 0 {
		return types.Int64Value(int64(offset))
	}
	// ValueInt64 is 0 for a null value, so null is preserved as well.
	if prior.IsUnknown() || prior.ValueInt64() != 0 {
		return types.Int64Null()
	}

	return prior
}

var _ resource.ConfigValidator = (*frequencyOffsetConfigValidator)(nil)

// frequencyOffsetConfigValidator validates that frequency_offset is set when
// frequency is 0 (high frequency scheduling), and not set otherwise.
type frequencyOffsetConfigValidator struct{}

// newFrequencyOffsetConfigValidator returns a resource config validator for
// resources that have both the frequency and frequency_offset attributes.
func newFrequencyOffsetConfigValidator() resource.ConfigValidator {
	return &frequencyOffsetConfigValidator{}
}

func (v *frequencyOffsetConfigValidator) Description(ctx context.Context) string {
	return "frequency_offset can only be set, and must be set, when frequency is 0"
}

func (v *frequencyOffsetConfigValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *frequencyOffsetConfigValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var frequency, offset types.Int64

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(frequencyAttributeName), &frequency)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(frequencyOffsetAttributeName), &offset)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if frequency.IsUnknown() || offset.IsUnknown() {
		return
	}

	// An unset frequency validates against the schema default.
	frequencyValue := defaultFrequency
	if !frequency.IsNull() {
		frequencyValue = frequency.ValueInt64()
	}

	// A frequency_offset of 0 is equivalent to not setting it at all.
	offsetDisabled := offset.IsNull() || offset.ValueInt64() == 0

	switch {
	case frequencyValue > 0 && !offsetDisabled:
		resp.Diagnostics.AddAttributeError(
			path.Root(frequencyOffsetAttributeName),
			"Invalid Attribute Combination",
			`"frequency_offset" can only be set when "frequency" is 0`,
		)
	case frequencyValue == 0 && offsetDisabled:
		resp.Diagnostics.AddAttributeError(
			path.Root(frequencyOffsetAttributeName),
			"Missing Required Attribute",
			`"frequency_offset" is required when "frequency" is 0`,
		)
	}
}
