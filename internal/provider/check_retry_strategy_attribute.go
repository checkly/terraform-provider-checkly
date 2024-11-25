package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
)

var (
	_ ResourceModel[checkly.RetryStrategy] = (*CheckRetryStrategyAttributeModel)(nil)
)

var CheckRetryStrategyAttributeSchema = schema.SingleNestedAttribute{
	Optional: true,
	Computed: true,
	Attributes: map[string]schema.Attribute{
		"type": schema.StringAttribute{
			Required: true,
			Validators: []validator.String{
				stringvalidator.OneOf(
					"FIXED",
					"LINEAR",
					"EXPONENTIAL",
				),
			},
			Description: "Determines which type of retry strategy to use. Possible values are `FIXED`, `LINEAR`, or `EXPONENTIAL`.",
		},
		"base_backoff_seconds": schema.Int32Attribute{
			Optional:    true,
			Default:     int32default.StaticInt32(60),
			Description: "The number of seconds to wait before the first retry attempt.",
		},
		"max_retries": schema.Int32Attribute{
			Optional: true,
			Default:  int32default.StaticInt32(2),
			Validators: []validator.Int32{
				int32validator.Between(1, 10),
			},
			Description: "The maximum number of times to retry the check. Value must be between 1 and 10.",
		},
		"max_duration_seconds": schema.Int32Attribute{
			Optional: true,
			Default:  int32default.StaticInt32(600),
			Validators: []validator.Int32{
				int32validator.AtMost(600),
			},
			Description: "The total amount of time to continue retrying the check (maximum 600 seconds).",
		},
		"same_region": schema.BoolAttribute{
			Optional:    true,
			Default:     booldefault.StaticBool(true),
			Description: "Whether retries should be run in the same region as the initial check run.",
		},
	},
	Description: "A strategy for retrying failed check runs.",
}

type CheckRetryStrategyAttributeModel struct {
	Type               types.String `tfsdk:"type"`
	BaseBackoffSeconds types.Int32  `tfsdk:"base_backoff_seconds"`
	MaxRetries         types.Int32  `tfsdk:"max_retries"`
	MaxDurationSeconds types.Int32  `tfsdk:"max_duration_seconds"`
	SameRegion         types.Bool   `tfsdk:"same_region"`
}

func (m *CheckRetryStrategyAttributeModel) Refresh(ctx context.Context, from *checkly.RetryStrategy, flags RefreshFlags) diag.Diagnostics {
	m.Type = types.StringValue(from.Type)
	m.BaseBackoffSeconds = types.Int32Value(int32(from.BaseBackoffSeconds))
	m.MaxRetries = types.Int32Value(int32(from.MaxRetries))
	m.MaxDurationSeconds = types.Int32Value(int32(from.MaxDurationSeconds))
	m.SameRegion = types.BoolValue(from.SameRegion)

	return nil
}

func (m *CheckRetryStrategyAttributeModel) Render(ctx context.Context, into *checkly.RetryStrategy) diag.Diagnostics {
	into.Type = m.Type.ValueString()
	into.BaseBackoffSeconds = int(m.BaseBackoffSeconds.ValueInt32())
	into.MaxRetries = int(m.MaxRetries.ValueInt32())
	into.MaxDurationSeconds = int(m.MaxDurationSeconds.ValueInt32())
	into.SameRegion = m.SameRegion.ValueBool()

	return nil
}
