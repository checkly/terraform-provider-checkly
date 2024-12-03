package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
)

var (
	_ resource.Resource                = (*CheckResource)(nil)
	_ resource.ResourceWithConfigure   = (*CheckResource)(nil)
	_ resource.ResourceWithImportState = (*CheckResource)(nil)
)

type CheckResource struct {
	client checkly.Client
}

func NewCheckResource() resource.Resource {
	return &CheckResource{}
}

func (r *CheckResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_check"
}

func (r *CheckResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description: "Check groups allow you to group together a set of related checks, which can also share default settings for various attributes.",
		Attributes: map[string]schema.Attribute{
			"id":           IDResourceAttributeSchema,
			"last_updated": LastUpdatedAttributeSchema,
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the check.",
			},
			"type": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						"API",
						"BROWSER",
						"MULTI_STEP",
					),
				},
				Description: "The type of the check. Possible values are `API`, `BROWSER`, and `MULTI_STEP`.",
			},
			"frequency": schema.Int32Attribute{
				Required: true,
				Validators: []validator.Int32{
					int32validator.OneOf(0, 1, 2, 5, 10, 15, 30, 60, 120, 180, 360, 720, 1440),
					// TODO: can only be 0 if type == API
				},
				Description: "The frequency in minutes to run the check. Possible values are `0`, `1`, `2`, `5`, `10`, `15`, `30`, `60`, `120`, `180`, `360`, `720`, and `1440`.",
			},
			"frequency_offset": schema.Int32Attribute{
				Optional: true,
				Validators: []validator.Int32{
					int32validator.OneOf(10, 20, 30), // TODO: Are these the only values?
					// TODO: can only be set if frequency == 0
				},
				Description: "This property is only valid for high frequency API checks. To create a high frequency check, the property `frequency` must be `0` and `frequency_offset` could be `10`, `20` or `30`.",
			},
			"activated": schema.BoolAttribute{
				Required:    true,
				Description: "Determines if the check is running or not. Possible values `true`, and `false`.",
			},
			"muted": schema.BoolAttribute{
				Optional:    true,
				Description: "Determines if any notifications will be sent out when a check fails/degrades/recovers.",
			},
			"should_fail": schema.BoolAttribute{
				Optional:    true,
				Description: "Allows to invert the behaviour of when a check is considered to fail. Allows for validating error status like 404.",
			},
			"run_parallel": schema.BoolAttribute{
				Optional:    true,
				Description: "Determines if the check should run in all selected locations in parallel or round-robin.",
			},
			"locations": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "An array of one or more data center locations where to run the checks.",
			},
			"private_locations": CheckPrivateLocationsAttributeSchema,
			"script": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "A valid piece of Node.js JavaScript code describing a browser interaction with the Puppeteer/Playwright framework or a reference to an external JavaScript file.",
			},
			"degraded_response_time": schema.Int32Attribute{
				Optional: true,
				Computed: true,
				Default:  int32default.StaticInt32(15000),
				Validators: []validator.Int32{
					int32validator.Between(0, 30000),
				},
				Description: "The response time in milliseconds starting from which a check should be considered degraded. Possible values are between 0 and 30000. (Default `15000`).",
			},
			"max_response_time": schema.Int32Attribute{
				Optional: true,
				Computed: true,
				Default:  int32default.StaticInt32(30000),
				Validators: []validator.Int32{
					int32validator.Between(0, 30000),
				},
				Description: "The response time in milliseconds starting from which a check should be considered failing. Possible values are between 0 and 30000. (Default `30000`).",
			},
			"environment_variables": schema.MapAttribute{
				ElementType: types.StringType,
				Validators: []validator.Map{
					mapvalidator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("environment_variable"),
					),
				},
				Optional:           true,
				Description:        "Key/value pairs for setting environment variables during check execution. These are only relevant for browser checks. Use global environment variables whenever possible.",
				DeprecationMessage: "The property `environment_variables` is deprecated and will be removed in a future version. Consider using the new `environment_variable` list.",
			},
			"environment_variable": CheckEnvironmentVariableAttributeSchema,
			"double_check": schema.BoolAttribute{
				Optional:           true,
				Description:        "Setting this to `true` will trigger a retry when a check fails from the failing region and another, randomly selected region before marking the check as failed.",
				DeprecationMessage: "The property `double_check` is deprecated and will be removed in a future version. To enable retries for failed check runs, use the `retry_strategy` property instead.",
			},
			"tags": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Tags for organizing and filtering checks.",
			},
			"ssl_check": schema.BoolAttribute{
				Optional:           true,
				Description:        "Determines if the SSL certificate should be validated for expiry.",
				DeprecationMessage: "The property `ssl_check` is deprecated and it's ignored by the Checkly Public API. It will be removed in a future version.",
			},
			"ssl_check_domain": schema.StringAttribute{
				Optional:   true,
				Validators: []validator.String{
					// TODO: can only be set if type == BROWSER
				},
				Description: "A valid fully qualified domain name (FQDN) to check its SSL certificate.",
			},
			"setup_snippet_id": schema.Int64Attribute{
				Optional:    true,
				Description: "An ID reference to a snippet to use in the setup phase of an API check.",
			},
			"teardown_snippet_id": schema.Int64Attribute{
				Optional:    true,
				Description: "An ID reference to a snippet to use in the teardown phase of an API check.",
			},
			"local_setup_script": schema.StringAttribute{
				Optional:    true,
				Description: "A valid piece of Node.js code to run in the setup phase.",
			},
			"local_teardown_script": schema.StringAttribute{
				Optional:    true,
				Description: "A valid piece of Node.js code to run in the teardown phase.",
			},
			"runtime_id": schema.StringAttribute{
				Optional:    true,
				Description: "The id of the runtime to use for this check.",
				// TODO: If type == MULTI_STEP, use GetRuntime to check whether
				// the runtime supports MULTI_STEP
			},
			"alert_channel_subscription": CheckAlertChannelSubscriptionAttributeSchema,
			"alert_settings":             CheckAlertSettingsAttributeSchema,
			"use_global_alert_settings": schema.BoolAttribute{
				Optional:    true,
				Description: "When true, the account level alert settings will be used, not the alert setting defined on this check.",
			},
			"request": CheckRequestAttributeSchema, // TODO: can only be set if type == API
			"group_id": schema.Int64Attribute{
				Optional:    true,
				Description: "The id of the check group this check is part of.",
			},
			"group_order": schema.Int32Attribute{
				Optional:    true,
				Description: "The position of this check in a check group. It determines in what order checks are run when a group is triggered from the API or from CI/CD.",
			},
			"retry_strategy": CheckRetryStrategyAttributeSchema,
		},
	}
}

func (r *CheckResource) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	client, diags := ClientFromProviderData(req.ProviderData)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	r.client = client
}

func (r *CheckResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *CheckResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan CheckResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var desiredModel checkly.Check
	resp.Diagnostics.Append(plan.Render(ctx, &desiredModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realizedModel, err := r.client.CreateCheck(ctx, desiredModel)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Checkly Check",
			fmt.Sprintf("Could not create check, unexpected error: %s", err),
		)

		return
	}

	resp.Diagnostics.Append(plan.Refresh(ctx, realizedModel, ModelCreated)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *CheckResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state CheckResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteCheck(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Checkly Check",
			fmt.Sprintf("Could not delete check, unexpected error: %s", err),
		)

		return
	}
}

func (r *CheckResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state CheckResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realizedModel, err := r.client.GetCheck(ctx, state.ID.ValueString())
	if err != nil {
		if SDKIsHTTPNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading Checkly Check",
			fmt.Sprintf("Could not retrieve check, unexpected error: %s", err),
		)

		return
	}

	resp.Diagnostics.Append(state.Refresh(ctx, realizedModel, ModelLoaded)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *CheckResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan CheckResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var desiredModel checkly.Check
	resp.Diagnostics.Append(plan.Render(ctx, &desiredModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realizedModel, err := r.client.UpdateCheck(
		ctx,
		plan.ID.ValueString(),
		desiredModel,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Checkly Check",
			fmt.Sprintf("Could not update check, unexpected error: %s", err),
		)

		return
	}

	resp.Diagnostics.Append(plan.Refresh(ctx, realizedModel, ModelUpdated)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

var (
	_ ResourceModel[checkly.Check] = (*CheckResourceModel)(nil)
)

type CheckResourceModel struct {
	ID                        types.String                                  `tfsdk:"id"`
	LastUpdated               types.String                                  `tfsdk:"last_updated"` // FIXME: Keep this? Old code did not have it.
	Name                      types.String                                  `tfsdk:"name"`
	Type                      types.String                                  `tfsdk:"type"`
	Frequency                 types.Int32                                   `tfsdk:"frequency"`
	FrequencyOffset           types.Int32                                   `tfsdk:"frequency_offset"`
	Activated                 types.Bool                                    `tfsdk:"activated"`
	Muted                     types.Bool                                    `tfsdk:"muted"`
	ShouldFail                types.Bool                                    `tfsdk:"should_fail"`
	RunParallel               types.Bool                                    `tfsdk:"run_parallel"`
	Locations                 CheckLocationsAttributeModel                  `tfsdk:"locations"`
	PrivateLocations          CheckPrivateLocationsAttributeModel           `tfsdk:"private_locations"`
	Script                    types.String                                  `tfsdk:"script"`
	DegradedResponseTime      types.Int32                                   `tfsdk:"degraded_response_time"`
	MaxResponseTime           types.Int32                                   `tfsdk:"max_response_time"`
	EnvironmentVariables      types.Map                                     `tfsdk:"environment_variables"`
	EnvironmentVariable       []CheckEnvironmentVariableAttributeModel      `tfsdk:"environment_variable"`
	DoubleCheck               types.Bool                                    `tfsdk:"double_check"`
	Tags                      types.Set                                     `tfsdk:"tags"`
	SSLCheck                  types.Bool                                    `tfsdk:"ssl_check"`
	SSLCheckDomain            types.String                                  `tfsdk:"ssl_check_domain"`
	SetupSnippetID            types.Int64                                   `tfsdk:"setup_snippet_id"`
	TearDownSnippetID         types.Int64                                   `tfsdk:"teardown_snippet_id"`
	LocalSetupScript          types.String                                  `tfsdk:"local_setup_script"`
	LocalTearDownScript       types.String                                  `tfsdk:"local_teardown_script"`
	RuntimeID                 types.String                                  `tfsdk:"runtime_id"`
	AlertChannelSubscriptions []CheckAlertChannelSubscriptionAttributeModel `tfsdk:"alert_channel_subscription"`
	AlertSettings             CheckAlertSettingsAttributeModel              `tfsdk:"alert_settings"`
	UseGlobalAlertSettings    types.Bool                                    `tfsdk:"use_global_alert_settings"`
	Request                   CheckRequestAttributeModel                    `tfsdk:"request"`
	GroupID                   types.Int64                                   `tfsdk:"group_id"`
	GroupOrder                types.Int32                                   `tfsdk:"group_order"`
	RetryStrategy             *CheckRetryStrategyAttributeModel             `tfsdk:"retry_strategy"`
}

func (m *CheckResourceModel) Refresh(ctx context.Context, from *checkly.Check, flags RefreshFlags) diag.Diagnostics {
	if flags.Created() {
		m.ID = types.StringValue(from.ID)
	}

	if flags.Created() || flags.Updated() {
		m.LastUpdated = LastUpdatedNow()
	}

	m.Name = types.StringValue(from.Name)
	m.Type = types.StringValue(from.Type)
	m.Frequency = types.Int32Value(int32(from.Frequency))
	m.FrequencyOffset = types.Int32Value(int32(from.FrequencyOffset))
	m.Activated = types.BoolValue(from.Activated)
	m.Muted = types.BoolValue(from.Muted)
	m.ShouldFail = types.BoolValue(from.ShouldFail)
	m.RunParallel = types.BoolValue(from.RunParallel)

	diags := m.Locations.Refresh(ctx, &from.Locations, flags)
	if diags.HasError() {
		return diags
	}

	diags = m.PrivateLocations.Refresh(ctx, from.PrivateLocations, flags)
	if diags.HasError() {
		return diags
	}

	m.Script = types.StringValue(from.Script)
	m.DegradedResponseTime = types.Int32Value(int32(from.DegradedResponseTime))
	m.MaxResponseTime = types.Int32Value(int32(from.MaxResponseTime))

	if !m.EnvironmentVariables.IsNull() {
		// Deprecated mode.
		m.EnvironmentVariables = types.MapNull(types.StringType)

		// TODO either implement backwards compat or remove.
	} else {
		m.EnvironmentVariable = nil

		diags := RefreshMany(ctx, from.EnvironmentVariables, m.EnvironmentVariable, flags)
		if diags.HasError() {
			return diags
		}
	}

	m.DoubleCheck = types.BoolValue(from.DoubleCheck)

	m.Tags = IntoUntypedStringSet(&from.Tags)

	m.SSLCheck = types.BoolValue(from.SSLCheck)
	m.SSLCheckDomain = types.StringValue(from.SSLCheckDomain)

	m.SetupSnippetID = types.Int64Value(from.SetupSnippetID)
	m.TearDownSnippetID = types.Int64Value(from.TearDownSnippetID)
	m.LocalSetupScript = types.StringValue(from.LocalSetupScript)
	m.LocalTearDownScript = types.StringValue(from.LocalTearDownScript)

	if from.RuntimeID != nil {
		m.RuntimeID = types.StringValue(*from.RuntimeID)
	} else {
		m.RuntimeID = types.StringNull()
	}

	diags = RefreshMany(ctx, from.AlertChannelSubscriptions, m.AlertChannelSubscriptions, flags)
	if diags.HasError() {
		return diags
	}

	diags = m.AlertSettings.Refresh(ctx, &from.AlertSettings, flags)
	if diags.HasError() {
		return diags
	}

	m.UseGlobalAlertSettings = types.BoolValue(from.UseGlobalAlertSettings)

	diags = m.Request.Refresh(ctx, &from.Request, flags)
	if diags.HasError() {
		return diags
	}

	m.GroupID = types.Int64Value(from.GroupID)
	m.GroupOrder = types.Int32Value(int32(from.GroupOrder))

	if from.RetryStrategy != nil {
		diags = m.RetryStrategy.Refresh(ctx, from.RetryStrategy, flags)
		if diags.HasError() {
			return diags
		}
	} else {
		m.RetryStrategy = nil
	}

	return nil
}

func (m *CheckResourceModel) Render(ctx context.Context, into *checkly.Check) diag.Diagnostics {
	into.Name = m.Name.ValueString()
	into.Type = m.Type.ValueString()
	into.Frequency = int(m.Frequency.ValueInt32())
	into.FrequencyOffset = int(m.FrequencyOffset.ValueInt32())
	into.Activated = m.Activated.ValueBool()
	into.Muted = m.Muted.ValueBool()
	into.ShouldFail = m.ShouldFail.ValueBool()
	into.RunParallel = m.RunParallel.ValueBool()

	diags := m.Locations.Render(ctx, &into.Locations)
	if diags.HasError() {
		return diags
	}

	diags = m.PrivateLocations.Render(ctx, into.PrivateLocations)
	if diags.HasError() {
		return diags
	}

	into.Script = m.Script.ValueString()
	into.DegradedResponseTime = int(m.DegradedResponseTime.ValueInt32())
	into.MaxResponseTime = int(m.MaxResponseTime.ValueInt32())

	if !m.EnvironmentVariables.IsNull() {
		// Deprecated mode.
		into.EnvironmentVariables = nil

		// TODO either implement backwards compat or remove.
	} else {
		into.EnvironmentVariables = nil

		diags := RenderMany(ctx, m.EnvironmentVariable, into.EnvironmentVariables)
		if diags.HasError() {
			return diags
		}
	}

	into.DoubleCheck = m.DoubleCheck.ValueBool()

	into.Tags = FromUntypedStringSet(m.Tags)

	into.SSLCheck = m.SSLCheck.ValueBool()
	into.SSLCheckDomain = m.SSLCheckDomain.ValueString()

	into.SetupSnippetID = m.SetupSnippetID.ValueInt64()
	into.TearDownSnippetID = m.TearDownSnippetID.ValueInt64()
	into.LocalSetupScript = m.LocalSetupScript.ValueString()
	into.LocalTearDownScript = m.LocalTearDownScript.ValueString()

	if !m.RuntimeID.IsNull() {
		value := m.RuntimeID.ValueString()
		into.RuntimeID = &value
	} else {
		into.RuntimeID = nil
	}

	diags = RenderMany(ctx, m.AlertChannelSubscriptions, into.AlertChannelSubscriptions)
	if diags.HasError() {
		return diags
	}

	diags = m.AlertSettings.Render(ctx, &into.AlertSettings)
	if diags.HasError() {
		return diags
	}

	into.UseGlobalAlertSettings = m.UseGlobalAlertSettings.ValueBool()

	diags = m.Request.Render(ctx, &into.Request)
	if diags.HasError() {
		return diags
	}

	into.GroupID = m.GroupID.ValueInt64()
	into.GroupOrder = int(m.GroupOrder.ValueInt32())

	if m.RetryStrategy != nil {
		diags = m.RetryStrategy.Render(ctx, into.RetryStrategy)
		if diags.HasError() {
			return diags
		}
	} else {
		into.RetryStrategy = nil
	}

	return nil
}
