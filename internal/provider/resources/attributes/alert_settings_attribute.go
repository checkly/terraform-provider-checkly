package attributes

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/checkly/terraform-provider-checkly/internal/provider/interop"
)

func init() {
	defaultAlertSettings := checkly.AlertSettings{
		EscalationType: "RUN_BASED",
		RunBasedEscalation: checkly.RunBasedEscalation{
			FailedRunThreshold: 5,
		},
	}

	defaultAlertSettingsObject, _, diags := AlertSettingsAttributeGluer.RefreshToObject(
		context.Background(),
		&defaultAlertSettings,
		interop.Loaded,
	)
	if diags.HasError() {
		panic(diags)
	}

	// defaultRunBasedEscalationObject, _, diags := RunBasedEscalationAttributeGluer.RefreshToObject(
	// 	context.Background(),
	// 	&defaultAlertSettings.RunBasedEscalation,
	// 	interop.Loaded,
	// )
	// if diags.HasError() {
	// 	panic(diags)
	// }

	AlertSettingsAttributeSchema.Default = objectdefault.StaticValue(defaultAlertSettingsObject)
	// RunBasedEscalationAttributeSchema.Default = objectdefault.StaticValue(defaultRunBasedEscalationObject)
}

var RunBasedEscalationAttributeSchema = schema.SingleNestedAttribute{
	Optional: true,
	Computed: true,
	Attributes: map[string]schema.Attribute{
		"failed_run_threshold": schema.Int32Attribute{
			Description: "After how many failed consecutive check runs an alert notification should be sent. Possible values are between 1 and 5. (Default `1`).",
			Optional:    true,
			Computed:    true,
			Default:     int32default.StaticInt32(1),
			Validators: []validator.Int32{
				int32validator.Between(1, 5),
			},
		},
	},
	Default: objectdefault.StaticValue(
		types.ObjectValueMust(
			map[string]attr.Type{
				"failed_run_threshold": types.Int32Type,
			},
			map[string]attr.Value{
				"failed_run_threshold": types.Int32Value(3),
			},
		),
	),
}

var TimeBasedEscalationAttributeSchema = schema.SingleNestedAttribute{
	Optional: true,
	Computed: true,
	Attributes: map[string]schema.Attribute{
		"minutes_failing_threshold": schema.Int32Attribute{
			Description: "After how many minutes after a check starts failing an alert should be sent. Possible values are `5`, `10`, `15`, and `30`. (Default `5`).",
			Optional:    true,
			Computed:    true,
			Default:     int32default.StaticInt32(5),
			Validators: []validator.Int32{
				int32validator.OneOf(5, 10, 15, 30),
			},
		},
	},
}

var RemindersAttributeSchema = schema.SingleNestedAttribute{
	Optional: true,
	Computed: true,
	Attributes: map[string]schema.Attribute{
		"amount": schema.Int32Attribute{
			Description: "How many reminders to send out after the initial alert notification. Possible values are `0`, `1`, `2`, `3`, `4`, `5`, and `100000`",
			Optional:    true,
			Validators: []validator.Int32{
				int32validator.OneOf(0, 1, 2, 3, 4, 5, 100000),
			},
		},
		"interval": schema.Int32Attribute{
			Description: "Possible values are `5`, `10`, `15`, and `30`. (Default `5`).",
			Optional:    true,
			Computed:    true,
			Default:     int32default.StaticInt32(5),
			Validators: []validator.Int32{
				int32validator.OneOf(5, 10, 15, 30),
			},
		},
	},
}

var ParallelRunFailureThresholdAttributeSchema = schema.SingleNestedAttribute{
	Optional: true,
	Computed: true,
	Attributes: map[string]schema.Attribute{
		"enabled": schema.BoolAttribute{
			Description: "Applicable only for checks scheduled in parallel in multiple locations.",
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(false),
		},
		"percentage": schema.Int32Attribute{
			Description: "Possible values are `10`, `20`, `30`, `40`, `50`, `60`, `70`, `80`, `90`, and `100`. (Default `10`).",
			Optional:    true,
			Computed:    true,
			Default:     int32default.StaticInt32(10),
			Validators: []validator.Int32{
				int32validator.OneOf(10, 20, 30, 40, 50, 60, 70, 80, 90, 100),
			},
		},
	},
}

var SSLCertificatesAttributeSchema = schema.SingleNestedAttribute{
	Description:        "At what interval the reminders should be sent.",
	Optional:           true,
	DeprecationMessage: "This property is deprecated and it's ignored by the Checkly Public API. It will be removed in a future version.",
	Attributes: map[string]schema.Attribute{
		"enabled": schema.BoolAttribute{
			Description: "Determines if alert notifications should be sent for expiring SSL certificates. Possible values `true`, and `false`. (Default `false`).",
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(false),
		},
		"alert_threshold": schema.Int32Attribute{
			Description: "How long before SSL certificate expiry to send alerts. Possible values `3`, `7`, `14`, `30`. (Default `3`).",
			Optional:    true,
			Computed:    true,
			Default:     int32default.StaticInt32(3),
			Validators: []validator.Int32{
				int32validator.OneOf(3, 7, 14, 30),
			},
		},
	},
}

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
		"run_based_escalation":           RunBasedEscalationAttributeSchema,
		"time_based_escalation":          TimeBasedEscalationAttributeSchema,
		"reminders":                      RemindersAttributeSchema,
		"parallel_run_failure_threshold": ParallelRunFailureThresholdAttributeSchema,
		"ssl_certificates":               SSLCertificatesAttributeSchema,
	},
}

type AlertSettingsAttributeModel struct {
	EscalationType              types.String `tfsdk:"escalation_type"`
	RunBasedEscalation          types.Object `tfsdk:"run_based_escalation"`
	TimeBasedEscalation         types.Object `tfsdk:"time_based_escalation"`
	Reminders                   types.Object `tfsdk:"reminders"`
	ParallelRunFailureThreshold types.Object `tfsdk:"parallel_run_failure_threshold"`
	SSLCertificates             types.Object `tfsdk:"ssl_certificates"`
}

var AlertSettingsAttributeGluer = interop.GluerForSingleNestedAttribute[
	checkly.AlertSettings,
	AlertSettingsAttributeModel,
](AlertSettingsAttributeSchema)

func (m *AlertSettingsAttributeModel) Refresh(ctx context.Context, from *checkly.AlertSettings, flags interop.RefreshFlags) diag.Diagnostics {
	var diags diag.Diagnostics

	m.EscalationType = types.StringValue(from.EscalationType)

	runBasedEscalation := &from.RunBasedEscalation
	timeBasedEscalation := &from.TimeBasedEscalation

	// switch from.EscalationType {
	// case checkly.RunBased:
	// 	timeBasedEscalation = nil
	// case checkly.TimeBased:
	// 	runBasedEscalation = nil
	// }

	m.RunBasedEscalation, _, diags = RunBasedEscalationAttributeGluer.RefreshToObject(ctx, runBasedEscalation, flags)
	if diags.HasError() {
		return diags
	}

	m.TimeBasedEscalation, _, diags = TimeBasedEscalationAttributeGluer.RefreshToObject(ctx, timeBasedEscalation, flags)
	if diags.HasError() {
		return diags
	}

	m.Reminders, _, diags = RemindersAttributeGluer.RefreshToObject(ctx, &from.Reminders, flags)
	if diags.HasError() {
		return diags
	}

	m.ParallelRunFailureThreshold, _, diags = ParallelRunFailureThresholdAttributeGluer.RefreshToObject(ctx, &from.ParallelRunFailureThreshold, flags)
	if diags.HasError() {
		return diags
	}

	sslCertificates := &from.SSLCertificates
	if !sslCertificates.Enabled && sslCertificates.AlertThreshold == 0 {
		sslCertificates = nil
	}

	m.SSLCertificates, _, diags = SSLCertificatesAttributeGluer.RefreshToObject(ctx, sslCertificates, flags)
	if diags.HasError() {
		return diags
	}

	return diags
}

func (m *AlertSettingsAttributeModel) Render(ctx context.Context, into *checkly.AlertSettings) diag.Diagnostics {
	var diags diag.Diagnostics

	switch m.EscalationType.ValueString() {
	case checkly.RunBased:
		into.EscalationType = checkly.RunBased
		into.RunBasedEscalation, _, diags = RunBasedEscalationAttributeGluer.RenderFromObject(ctx, m.RunBasedEscalation)
		if diags.HasError() {
			return diags
		}
	case checkly.TimeBased:
		into.EscalationType = checkly.TimeBased
		into.TimeBasedEscalation, _, diags = TimeBasedEscalationAttributeGluer.RenderFromObject(ctx, m.TimeBasedEscalation)
		if diags.HasError() {
			return diags
		}
	default:
		panic("OTHER")
		// TODO diags
	}

	into.Reminders, _, diags = RemindersAttributeGluer.RenderFromObject(ctx, m.Reminders)
	if diags.HasError() {
		return diags
	}

	into.ParallelRunFailureThreshold, _, diags = ParallelRunFailureThresholdAttributeGluer.RenderFromObject(ctx, m.ParallelRunFailureThreshold)
	if diags.HasError() {
		return diags
	}

	into.SSLCertificates, _, diags = SSLCertificatesAttributeGluer.RenderFromObject(ctx, m.SSLCertificates)
	if diags.HasError() {
		return diags
	}

	return diags
}

type RunBasedEscalationAttributeModel struct {
	FailedRunThreshold types.Int32 `tfsdk:"failed_run_threshold"`
}

var RunBasedEscalationAttributeGluer = interop.GluerForSingleNestedAttribute[
	checkly.RunBasedEscalation,
	RunBasedEscalationAttributeModel,
](RunBasedEscalationAttributeSchema)

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

var TimeBasedEscalationAttributeGluer = interop.GluerForSingleNestedAttribute[
	checkly.TimeBasedEscalation,
	TimeBasedEscalationAttributeModel,
](TimeBasedEscalationAttributeSchema)

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

var RemindersAttributeGluer = interop.GluerForSingleNestedAttribute[
	checkly.Reminders,
	RemindersAttributeModel,
](RemindersAttributeSchema)

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

var ParallelRunFailureThresholdAttributeGluer = interop.GluerForSingleNestedAttribute[
	checkly.ParallelRunFailureThreshold,
	ParallelRunFailureThresholdAttributeModel,
](ParallelRunFailureThresholdAttributeSchema)

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

var SSLCertificatesAttributeGluer = interop.GluerForSingleNestedAttribute[
	checkly.SSLCertificates,
	SSLCertificatesAttributeModel,
](SSLCertificatesAttributeSchema)

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
