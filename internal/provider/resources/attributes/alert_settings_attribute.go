package attributes

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/checkly/terraform-provider-checkly/internal/provider/interop"
)

var (
	_ interop.Model[checkly.AlertSettings]               = (*AlertSettingsAttributeModel)(nil)
	_ interop.Model[checkly.RunBasedEscalation]          = (*RunBasedEscalationAttributeModel)(nil)
	_ interop.Model[checkly.TimeBasedEscalation]         = (*TimeBasedEscalationAttributeModel)(nil)
	_ interop.Model[checkly.Reminders]                   = (*RemindersAttributeModel)(nil)
	_ interop.Model[checkly.ParallelRunFailureThreshold] = (*ParallelRunFailureThresholdAttributeModel)(nil)
	_ interop.Model[checkly.SSLCertificates]             = (*SSLCertificatesAttributeModel)(nil)
)

var AlertSettingsAttributeSchema = schema.SingleNestedAttribute{
	Optional: true,
	Computed: true,
	Attributes: map[string]schema.Attribute{
		"escalation_type": schema.StringAttribute{
			Optional: true,
			Computed: true,
			Default:  stringdefault.StaticString("RUN_BASED"),
			Validators: []validator.String{
				stringvalidator.OneOf("RUN_BASED", "TIME_BASED"),
			},
			Description: "Determines what type of escalation to use. Possible values are `RUN_BASED` or `TIME_BASED`.",
		},
		"run_based_escalation": schema.SingleNestedAttribute{
			Optional: true,
			Computed: true,
			Attributes: map[string]schema.Attribute{
				"failed_run_threshold": schema.Int32Attribute{
					Optional: true,
					Computed: true,
					Default:  int32default.StaticInt32(1),
					Validators: []validator.Int32{
						int32validator.Between(1, 5),
					},
					Description: "After how many failed consecutive check runs an alert notification should be sent. Possible values are between 1 and 5. (Default `1`).",
				},
			},
		},
		"time_based_escalation": schema.SingleNestedAttribute{
			Optional: true,
			Computed: true,
			Attributes: map[string]schema.Attribute{
				"minutes_failing_threshold": schema.Int32Attribute{
					Optional: true,
					Computed: true,
					Default:  int32default.StaticInt32(5),
					Validators: []validator.Int32{
						int32validator.OneOf(5, 10, 15, 30),
					},
					Description: "After how many minutes after a check starts failing an alert should be sent. Possible values are `5`, `10`, `15`, and `30`. (Default `5`).",
				},
			},
		},
		"reminders": schema.SingleNestedAttribute{
			Optional: true,
			Computed: true,
			Attributes: map[string]schema.Attribute{
				"amount": schema.Int32Attribute{
					Optional: true,
					Validators: []validator.Int32{
						int32validator.OneOf(0, 1, 2, 3, 4, 5, 100000),
					},
					Description: "How many reminders to send out after the initial alert notification. Possible values are `0`, `1`, `2`, `3`, `4`, `5`, and `100000`",
				},
				"interval": schema.Int32Attribute{
					Optional: true,
					Computed: true,
					Default:  int32default.StaticInt32(5),
					Validators: []validator.Int32{
						int32validator.OneOf(5, 10, 15, 30),
					},
					Description: "Possible values are `5`, `10`, `15`, and `30`. (Default `5`).",
				},
			},
		},
		"parallel_run_failure_threshold": schema.SingleNestedAttribute{
			Optional: true,
			Computed: true,
			Attributes: map[string]schema.Attribute{
				"enabled": schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(false),
					Description: "Applicable only for checks scheduled in parallel in multiple locations.",
				},
				"percentage": schema.Int32Attribute{
					Optional: true,
					Computed: true,
					Default:  int32default.StaticInt32(10),
					Validators: []validator.Int32{
						int32validator.OneOf(10, 20, 30, 40, 50, 60, 70, 80, 90, 100),
					},
					Description: "Possible values are `10`, `20`, `30`, `40`, `50`, `60`, `70`, `80`, `90`, and `100`. (Default `10`).",
				},
			},
		},
		"ssl_certificates": schema.SingleNestedAttribute{
			Optional:           true,
			DeprecationMessage: "This property is deprecated and it's ignored by the Checkly Public API. It will be removed in a future version.",
			Attributes: map[string]schema.Attribute{
				"enabled": schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(false),
					Description: "Determines if alert notifications should be sent for expiring SSL certificates. Possible values `true`, and `false`. (Default `false`).",
				},
				"alert_threshold": schema.Int32Attribute{
					Optional: true,
					Computed: true,
					Default:  int32default.StaticInt32(3),
					Validators: []validator.Int32{
						int32validator.OneOf(3, 7, 14, 30),
					},
					Description: "How long before SSL certificate expiry to send alerts. Possible values `3`, `7`, `14`, `30`. (Default `3`).",
				},
			},
			Description: "At what interval the reminders should be sent.",
		},
	},
}

type AlertSettingsAttributeModel struct {
	EscalationType              types.String                              `tfsdk:"escalation_type"`
	RunBasedEscalations         RunBasedEscalationAttributeModel          `tfsdk:"run_based_escalation"`
	TimeBasedEscalations        TimeBasedEscalationAttributeModel         `tfsdk:"time_based_escalation"`
	Reminders                   RemindersAttributeModel                   `tfsdk:"reminders"`
	ParallelRunFailureThreshold ParallelRunFailureThresholdAttributeModel `tfsdk:"parallel_run_failure_threshold"`
	SSLCertificates             SSLCertificatesAttributeModel             `tfsdk:"ssl_certificates"`
}

func (m *AlertSettingsAttributeModel) Refresh(ctx context.Context, from *checkly.AlertSettings, flags interop.RefreshFlags) diag.Diagnostics {
	var diags diag.Diagnostics

	m.EscalationType = types.StringValue(from.EscalationType)

	switch from.EscalationType {
	case checkly.RunBased:
		diags.Append(m.RunBasedEscalations.Refresh(ctx, &from.RunBasedEscalation, flags)...)
	case checkly.TimeBased:
		diags.Append(m.TimeBasedEscalations.Refresh(ctx, &from.TimeBasedEscalation, flags)...)
	default:
		// TODO diags
	}

	diags.Append(m.Reminders.Refresh(ctx, &from.Reminders, flags)...)
	diags.Append(m.ParallelRunFailureThreshold.Refresh(ctx, &from.ParallelRunFailureThreshold, flags)...)
	diags.Append(m.SSLCertificates.Refresh(ctx, &from.SSLCertificates, flags)...)

	return diags
}

func (m *AlertSettingsAttributeModel) Render(ctx context.Context, into *checkly.AlertSettings) diag.Diagnostics {
	var diags diag.Diagnostics

	switch m.EscalationType.ValueString() {
	case checkly.RunBased:
		into.EscalationType = checkly.RunBased
		diags.Append(m.RunBasedEscalations.Render(ctx, &into.RunBasedEscalation)...)
	case checkly.TimeBased:
		into.EscalationType = checkly.TimeBased
		diags.Append(m.TimeBasedEscalations.Render(ctx, &into.TimeBasedEscalation)...)
	default:
		// TODO diags
	}

	diags.Append(m.Reminders.Render(ctx, &into.Reminders)...)
	diags.Append(m.ParallelRunFailureThreshold.Render(ctx, &into.ParallelRunFailureThreshold)...)
	diags.Append(m.SSLCertificates.Render(ctx, &into.SSLCertificates)...)

	return diags
}

type RunBasedEscalationAttributeModel struct {
	FailedRunThreshold types.Int32 `tfsdk:"failed_run_threshold"`
}

func (m *RunBasedEscalationAttributeModel) Refresh(ctx context.Context, from *checkly.RunBasedEscalation, flags interop.RefreshFlags) diag.Diagnostics {
	m.FailedRunThreshold = types.Int32Value(int32(from.FailedRunThreshold))

	return nil
}

func (m *RunBasedEscalationAttributeModel) Render(ctx context.Context, into *checkly.RunBasedEscalation) diag.Diagnostics {
	into.FailedRunThreshold = int(m.FailedRunThreshold.ValueInt32())

	return nil
}

type TimeBasedEscalationAttributeModel struct {
	MinutesFailingThreshold types.Int32 `tfsdk:"minutes_failing_threshold"`
}

func (m *TimeBasedEscalationAttributeModel) Refresh(ctx context.Context, from *checkly.TimeBasedEscalation, flags interop.RefreshFlags) diag.Diagnostics {
	m.MinutesFailingThreshold = types.Int32Value(int32(from.MinutesFailingThreshold))

	return nil
}

func (m *TimeBasedEscalationAttributeModel) Render(ctx context.Context, into *checkly.TimeBasedEscalation) diag.Diagnostics {
	into.MinutesFailingThreshold = int(m.MinutesFailingThreshold.ValueInt32())

	return nil
}

type RemindersAttributeModel struct {
	Amount   types.Int32 `tfsdk:"amount"`
	Interval types.Int32 `tfsdk:"interval"`
}

func (m *RemindersAttributeModel) Refresh(ctx context.Context, from *checkly.Reminders, flags interop.RefreshFlags) diag.Diagnostics {
	m.Amount = types.Int32Value(int32(from.Amount))
	m.Interval = types.Int32Value(int32(from.Interval))

	return nil
}

func (m *RemindersAttributeModel) Render(ctx context.Context, into *checkly.Reminders) diag.Diagnostics {
	into.Amount = int(m.Amount.ValueInt32())
	into.Interval = int(m.Interval.ValueInt32())

	return nil
}

type ParallelRunFailureThresholdAttributeModel struct {
	Enabled    types.Bool  `tfsdk:"enabled"`
	Percentage types.Int32 `tfsdk:"percentage"`
}

func (m *ParallelRunFailureThresholdAttributeModel) Refresh(ctx context.Context, from *checkly.ParallelRunFailureThreshold, flags interop.RefreshFlags) diag.Diagnostics {
	m.Enabled = types.BoolValue(from.Enabled)
	m.Percentage = types.Int32Value(int32(from.Percentage))

	return nil
}

func (m *ParallelRunFailureThresholdAttributeModel) Render(ctx context.Context, into *checkly.ParallelRunFailureThreshold) diag.Diagnostics {
	into.Enabled = m.Enabled.ValueBool()
	into.Percentage = int(m.Percentage.ValueInt32())

	return nil
}

type SSLCertificatesAttributeModel struct {
	Enabled        types.Bool  `tfsdk:"enabled"`
	AlertThreshold types.Int32 `tfsdk:"alert_threshold"`
}

func (m *SSLCertificatesAttributeModel) Refresh(ctx context.Context, from *checkly.SSLCertificates, flags interop.RefreshFlags) diag.Diagnostics {
	m.Enabled = types.BoolValue(from.Enabled)
	m.AlertThreshold = types.Int32Value(int32(from.AlertThreshold))

	return nil
}

func (m *SSLCertificatesAttributeModel) Render(ctx context.Context, into *checkly.SSLCertificates) diag.Diagnostics {
	into.Enabled = m.Enabled.ValueBool()
	into.AlertThreshold = int(m.AlertThreshold.ValueInt32())

	return nil
}
