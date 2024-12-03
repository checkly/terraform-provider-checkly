package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
)

var (
	_ resource.Resource                = (*CheckGroupResource)(nil)
	_ resource.ResourceWithConfigure   = (*CheckGroupResource)(nil)
	_ resource.ResourceWithImportState = (*CheckGroupResource)(nil)
)

type CheckGroupResource struct {
	client checkly.Client
}

func NewCheckGroupResource() resource.Resource {
	return &CheckGroupResource{}
}

func (r *CheckGroupResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_check_group"
}

func (r *CheckGroupResource) Schema(
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
				Description: "The name of the check group.",
			},
			"concurrency": schema.Int32Attribute{
				Required:    true,
				Description: "Determines how many checks are run concurrently when triggering a check group from CI/CD or through the API.",
			},
			"activated": schema.BoolAttribute{
				Required:    true,
				Description: "Determines if the checks in the group are running or not.",
			},
			"muted": schema.BoolAttribute{
				Optional:    true,
				Description: "Determines if any notifications will be sent out when a check in this group fails and/or recovers.",
			},
			"run_parallel": schema.BoolAttribute{
				Optional:    true,
				Description: "Determines if the checks in the group should run in all selected locations in parallel or round-robin.",
			},
			"locations":         CheckLocationsAttributeSchema,
			"private_locations": CheckPrivateLocationsAttributeSchema,
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
				Description: "A valid piece of Node.js code to run in the setup phase of an API check in this group.",
			},
			"local_teardown_script": schema.StringAttribute{
				Optional:    true,
				Description: "A valid piece of Node.js code to run in the teardown phase of an API check in this group.",
			},
			"runtime_id": schema.StringAttribute{
				Optional:    true,
				Description: "The id of the runtime to use for this group.",
			},
			"alert_channel_subscription": CheckAlertChannelSubscriptionAttributeSchema,
			"alert_settings":             CheckAlertSettingsAttributeSchema,
			"use_global_alert_settings": schema.BoolAttribute{
				Optional:    true,
				Description: "When true, the account level alert settings will be used, not the alert setting defined on this check group.",
			},
			"api_check_defaults": CheckAPICheckDefaultsAttributeSchema,
			"retry_strategy":     CheckRetryStrategyAttributeSchema,
		},
	}
}

func (r *CheckGroupResource) Configure(
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

func (r *CheckGroupResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *CheckGroupResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan CheckGroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var desiredModel checkly.Group
	resp.Diagnostics.Append(plan.Render(ctx, &desiredModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realizedModel, err := r.client.CreateGroup(ctx, desiredModel)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Checkly Check Group",
			fmt.Sprintf("Could not create check group, unexpected error: %s", err),
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

func (r *CheckGroupResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state CheckGroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, diags := CheckGroupID.FromString(state.ID)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	err := r.client.DeleteGroup(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Checkly Check Group",
			fmt.Sprintf("Could not delete check group, unexpected error: %s", err),
		)

		return
	}
}

func (r *CheckGroupResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state CheckGroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, diags := CheckGroupID.FromString(state.ID)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	realizedModel, err := r.client.GetGroup(ctx, id)
	if err != nil {
		if SDKIsHTTPNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading Checkly Check Group",
			fmt.Sprintf("Could not retrieve check group, unexpected error: %s", err),
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

func (r *CheckGroupResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan CheckGroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, diags := CheckGroupID.FromString(plan.ID)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	var desiredModel checkly.Group
	resp.Diagnostics.Append(plan.Render(ctx, &desiredModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realizedModel, err := r.client.UpdateGroup(
		ctx,
		id,
		desiredModel,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Checkly Check Group",
			fmt.Sprintf("Could not update check group, unexpected error: %s", err),
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

var CheckGroupID = SDKIdentifier{
	Path:  path.Root("id"),
	Title: "Checkly Check Group ID",
}

var (
	_ ResourceModel[checkly.Group] = (*CheckGroupResourceModel)(nil)
)

type CheckGroupResourceModel struct {
	ID                        types.String                                  `tfsdk:"id"`
	LastUpdated               types.String                                  `tfsdk:"last_updated"` // FIXME: Keep this? Old code did not have it.
	Name                      types.String                                  `tfsdk:"name"`
	Concurrency               types.Int32                                   `tfsdk:"concurrency"`
	Activated                 types.Bool                                    `tfsdk:"activated"`
	Muted                     types.Bool                                    `tfsdk:"muted"`
	RunParallel               types.Bool                                    `tfsdk:"run_parallel"`
	Locations                 CheckLocationsAttributeModel                  `tfsdk:"locations"`
	PrivateLocations          CheckPrivateLocationsAttributeModel           `tfsdk:"private_locations"`
	EnvironmentVariables      types.Map                                     `tfsdk:"environment_variables"`
	EnvironmentVariable       []CheckEnvironmentVariableAttributeModel      `tfsdk:"environment_variable"`
	DoubleCheck               types.Bool                                    `tfsdk:"double_check"`
	Tags                      types.Set                                     `tfsdk:"tags"`
	SetupSnippetID            types.Int64                                   `tfsdk:"setup_snippet_id"`
	TearDownSnippetID         types.Int64                                   `tfsdk:"teardown_snippet_id"`
	LocalSetupScript          types.String                                  `tfsdk:"local_setup_script"`
	LocalTearDownScript       types.String                                  `tfsdk:"local_teardown_script"`
	RuntimeID                 types.String                                  `tfsdk:"runtime_id"`
	AlertChannelSubscriptions []CheckAlertChannelSubscriptionAttributeModel `tfsdk:"alert_channel_subscription"`
	AlertSettings             CheckAlertSettingsAttributeModel              `tfsdk:"alert_settings"`
	UseGlobalAlertSettings    types.Bool                                    `tfsdk:"use_global_alert_settings"`
	APICheckDefaults          CheckAPICheckDefaultsAttributeModel           `tfsdk:"api_check_defaults"`
	RetryStrategy             *CheckRetryStrategyAttributeModel             `tfsdk:"retry_strategy"`
}

func (m *CheckGroupResourceModel) Refresh(ctx context.Context, from *checkly.Group, flags RefreshFlags) diag.Diagnostics {
	if flags.Created() {
		m.ID = CheckGroupID.IntoString(from.ID)
	}

	if flags.Created() || flags.Updated() {
		m.LastUpdated = LastUpdatedNow()
	}

	m.Name = types.StringValue(from.Name)
	m.Concurrency = types.Int32Value(int32(from.Concurrency))
	m.Activated = types.BoolValue(from.Activated)
	m.Muted = types.BoolValue(from.Muted)
	m.RunParallel = types.BoolValue(from.RunParallel)

	diags := m.Locations.Refresh(ctx, &from.Locations, flags)
	if diags != nil {
		return diags
	}

	diags = m.PrivateLocations.Refresh(ctx, from.PrivateLocations, flags)
	if diags != nil {
		return diags
	}

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

	diags = m.APICheckDefaults.Refresh(ctx, &from.APICheckDefaults, flags)
	if diags.HasError() {
		return diags
	}

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

func (m *CheckGroupResourceModel) Render(ctx context.Context, into *checkly.Group) diag.Diagnostics {
	into.Name = m.Name.ValueString()
	into.Concurrency = int(m.Concurrency.ValueInt32())
	into.Activated = m.Activated.ValueBool()
	into.Muted = m.Muted.ValueBool()
	into.RunParallel = m.RunParallel.ValueBool()

	diags := m.Locations.Render(ctx, &into.Locations)
	if diags.HasError() {
		return diags
	}

	diags = m.PrivateLocations.Render(ctx, into.PrivateLocations)
	if diags.HasError() {
		return diags
	}

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

	diags = m.APICheckDefaults.Render(ctx, &into.APICheckDefaults)
	if diags.HasError() {
		return diags
	}

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
