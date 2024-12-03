package provider

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
)

var (
	_ ResourceModel[checkly.AlertSettings]               = (*CheckAlertSettingsAttributeModel)(nil)
	_ ResourceModel[checkly.RunBasedEscalation]          = (*CheckRunBasedEscalationAttributeModel)(nil)
	_ ResourceModel[checkly.TimeBasedEscalation]         = (*CheckTimeBasedEscalationAttributeModel)(nil)
	_ ResourceModel[checkly.Reminders]                   = (*CheckRemindersAttributeModel)(nil)
	_ ResourceModel[checkly.ParallelRunFailureThreshold] = (*CheckParallelRunFailureThresholdAttributeModel)(nil)
	_ ResourceModel[checkly.SSLCertificates]             = (*CheckSSLCertificatesAttributeModel)(nil)
)

var CheckAlertSettingsAttributeSchema = schema.SingleNestedAttribute{
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

type CheckAlertSettingsAttributeModel struct {
	EscalationType              types.String                                   `tfsdk:"escalation_type"`
	RunBasedEscalations         CheckRunBasedEscalationAttributeModel          `tfsdk:"run_based_escalation"`
	TimeBasedEscalations        CheckTimeBasedEscalationAttributeModel         `tfsdk:"time_based_escalation"`
	Reminders                   CheckRemindersAttributeModel                   `tfsdk:"reminders"`
	ParallelRunFailureThreshold CheckParallelRunFailureThresholdAttributeModel `tfsdk:"parallel_run_failure_threshold"`
	SSLCertificates             CheckSSLCertificatesAttributeModel             `tfsdk:"ssl_certificates"`
}

func (m *CheckAlertSettingsAttributeModel) Refresh(ctx context.Context, from *checkly.AlertSettings, flags RefreshFlags) diag.Diagnostics {
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

func (m *CheckAlertSettingsAttributeModel) Render(ctx context.Context, into *checkly.AlertSettings) diag.Diagnostics {
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

type CheckRunBasedEscalationAttributeModel struct {
	FailedRunThreshold types.Int32 `tfsdk:"failed_run_threshold"`
}

func (m *CheckRunBasedEscalationAttributeModel) Refresh(ctx context.Context, from *checkly.RunBasedEscalation, flags RefreshFlags) diag.Diagnostics {
	m.FailedRunThreshold = types.Int32Value(int32(from.FailedRunThreshold))

	return nil
}

func (m *CheckRunBasedEscalationAttributeModel) Render(ctx context.Context, into *checkly.RunBasedEscalation) diag.Diagnostics {
	into.FailedRunThreshold = int(m.FailedRunThreshold.ValueInt32())

	return nil
}

type CheckTimeBasedEscalationAttributeModel struct {
	MinutesFailingThreshold types.Int32 `tfsdk:"minutes_failing_threshold"`
}

func (m *CheckTimeBasedEscalationAttributeModel) Refresh(ctx context.Context, from *checkly.TimeBasedEscalation, flags RefreshFlags) diag.Diagnostics {
	m.MinutesFailingThreshold = types.Int32Value(int32(from.MinutesFailingThreshold))

	return nil
}

func (m *CheckTimeBasedEscalationAttributeModel) Render(ctx context.Context, into *checkly.TimeBasedEscalation) diag.Diagnostics {
	into.MinutesFailingThreshold = int(m.MinutesFailingThreshold.ValueInt32())

	return nil
}

type CheckRemindersAttributeModel struct {
	Amount   types.Int32 `tfsdk:"amount"`
	Interval types.Int32 `tfsdk:"interval"`
}

func (m *CheckRemindersAttributeModel) Refresh(ctx context.Context, from *checkly.Reminders, flags RefreshFlags) diag.Diagnostics {
	m.Amount = types.Int32Value(int32(from.Amount))
	m.Interval = types.Int32Value(int32(from.Interval))

	return nil
}

func (m *CheckRemindersAttributeModel) Render(ctx context.Context, into *checkly.Reminders) diag.Diagnostics {
	into.Amount = int(m.Amount.ValueInt32())
	into.Interval = int(m.Interval.ValueInt32())

	return nil
}

type CheckParallelRunFailureThresholdAttributeModel struct {
	Enabled    types.Bool  `tfsdk:"enabled"`
	Percentage types.Int32 `tfsdk:"percentage"`
}

func (m *CheckParallelRunFailureThresholdAttributeModel) Refresh(ctx context.Context, from *checkly.ParallelRunFailureThreshold, flags RefreshFlags) diag.Diagnostics {
	m.Enabled = types.BoolValue(from.Enabled)
	m.Percentage = types.Int32Value(int32(from.Percentage))

	return nil
}

func (m *CheckParallelRunFailureThresholdAttributeModel) Render(ctx context.Context, into *checkly.ParallelRunFailureThreshold) diag.Diagnostics {
	into.Enabled = m.Enabled.ValueBool()
	into.Percentage = int(m.Percentage.ValueInt32())

	return nil
}

type CheckSSLCertificatesAttributeModel struct {
	Enabled        types.Bool  `tfsdk:"enabled"`
	AlertThreshold types.Int32 `tfsdk:"alert_threshold"`
}

func (m *CheckSSLCertificatesAttributeModel) Refresh(ctx context.Context, from *checkly.SSLCertificates, flags RefreshFlags) diag.Diagnostics {
	m.Enabled = types.BoolValue(from.Enabled)
	m.AlertThreshold = types.Int32Value(int32(from.AlertThreshold))

	return nil
}

func (m *CheckSSLCertificatesAttributeModel) Render(ctx context.Context, into *checkly.SSLCertificates) diag.Diagnostics {
	into.Enabled = m.Enabled.ValueBool()
	into.AlertThreshold = int(m.AlertThreshold.ValueInt32())

	return nil
}
