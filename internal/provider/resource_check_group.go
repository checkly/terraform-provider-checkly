package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
)

var (
	_ resource.Resource                 = (*CheckGroupResource)(nil)
	_ resource.ResourceWithConfigure    = (*CheckGroupResource)(nil)
	_ resource.ResourceWithUpgradeState = (*CheckGroupResource)(nil)
	_ resource.ResourceWithImportState  = (*CheckGroupResource)(nil)
	_ resource.ResourceWithModifyPlan   = (*CheckGroupResource)(nil)
)

// NewCheckGroupResource returns a new checkly_check_group resource.
func NewCheckGroupResource() resource.Resource {
	return &CheckGroupResource{}
}

// CheckGroupResource implements the checkly_check_group resource.
type CheckGroupResource struct {
	client checkly.Client
}

// CheckGroupResourceModel describes the resource data model.
type CheckGroupResourceModel struct {
	ID                       types.String `tfsdk:"id"`
	Name                     types.String `tfsdk:"name"`
	Concurrency              types.Int64  `tfsdk:"concurrency"`
	Activated                types.Bool   `tfsdk:"activated"`
	Muted                    types.Bool   `tfsdk:"muted"`
	RunParallel              types.Bool   `tfsdk:"run_parallel"`
	Locations                types.Set    `tfsdk:"locations"`
	PrivateLocations         types.Set    `tfsdk:"private_locations"`
	Tags                     types.Set    `tfsdk:"tags"`
	EnvironmentVariable      types.List   `tfsdk:"environment_variable"`
	SetupSnippetID           types.Int64  `tfsdk:"setup_snippet_id"`
	TeardownSnippetID        types.Int64  `tfsdk:"teardown_snippet_id"`
	LocalSetupScript         types.String `tfsdk:"local_setup_script"`
	LocalTeardownScript      types.String `tfsdk:"local_teardown_script"`
	RuntimeID                types.String `tfsdk:"runtime_id"`
	AlertChannelSubscription types.Set    `tfsdk:"alert_channel_subscription"`
	AlertSettings            types.Object `tfsdk:"alert_settings"`
	UseGlobalAlertSettings   types.Bool   `tfsdk:"use_global_alert_settings"`
	APICheckDefaults         types.Object `tfsdk:"api_check_defaults"`
	RetryStrategy            types.Object `tfsdk:"retry_strategy"`
}

func (r *CheckGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_check_group"
}

func (r *CheckGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: sdkv2SchemaVersion + 1,
		Description: "Check groups allow you to group together a set of related checks, which can also share default settings for various attributes." +
			"\n\n" +
			"The `checkly_check_group` resource type is deprecated in favor of `checkly_check_group_v2`, which does " +
			"not impose implicit defaults on its checks, and will be removed in a future version of the provider. " +
			"Both resource types manage the same check groups, so an existing group can be switched over by " +
			"importing it as a `checkly_check_group_v2` resource and removing the `checkly_check_group` resource " +
			"from state. See the v2 upgrade guide for the steps.",
		DeprecationMessage: "Use the `checkly_check_group_v2` resource instead. An existing group can be switched " +
			"over without recreating it; see the v2 upgrade guide.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the check group.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the check group.",
			},
			"concurrency": schema.Int64Attribute{
				Required:    true,
				Description: "Determines how many checks are run concurrently when triggering a check group from CI/CD or through the API.",
			},
			"activated": schema.BoolAttribute{
				Required:    true,
				Description: "Determines if the checks in the group are running or not.",
			},
			"muted": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Determines if any notifications will be sent out when a check in this group fails and/or recovers.",
			},
			"run_parallel": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Determines if the checks in the group should run in all selected locations in parallel or round-robin.",
			},
			"locations": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
				Description: "An array of one or more data center locations where to run the checks.",
			},
			"private_locations": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
				Description: "An array of one or more private locations slugs.",
			},
			"tags": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
				Description: "Tags for organizing and filtering checks.",
			},
			"setup_snippet_id": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "An ID reference to a snippet to use in the setup phase of an API check.",
			},
			"teardown_snippet_id": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "An ID reference to a snippet to use in the teardown phase of an API check.",
			},
			"local_setup_script": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "A valid piece of Node.js code to run in the setup phase of an API check in this group.",
			},
			"local_teardown_script": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "A valid piece of Node.js code to run in the teardown phase of an API check in this group.",
			},
			"runtime_id": schema.StringAttribute{
				Optional:    true,
				Description: "The id of the runtime to use for this group.",
			},
			alertSettingsAttributeName: makeAlertSettingsAttributeSchema(AlertSettingsAttributeSchemaOptions{}),
			"use_global_alert_settings": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "When true, the account level alert settings will be used, not the alert setting defined on this check group.",
			},
			apiCheckDefaultsAttributeName: makeAPICheckDefaultsAttributeSchema(),
			retryStrategyAttributeName: makeRetryStrategyAttributeSchema(RetryStrategyAttributeSchemaOptions{
				SupportsOnlyOnNetworkError: true,
			}),
		},
		Blocks: map[string]schema.Block{
			environmentVariableAttributeName: makeEnvironmentVariableSchema(EnvironmentVariableAttributeSchemaOptions{
				Description: "Insert environment variables into the runtime environment. Only relevant for browser checks. Use global environment variables whenever possible.",
			}),
			alertChannelSubscriptionAttributeName: makeAlertChannelSubscriptionSchema(AlertChannelSubscriptionAttributeSchemaOptions{}),
		},
	}
}

// UpgradeState upgrades state written by the provider's SDK v2
// implementation; see sdkv2StateUpgraders.
func (r *CheckGroupResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return sdkv2StateUpgraders(ctx, r)
}

func (r *CheckGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

// ModifyPlan applies default values to the optional+computed attributes
// nested in the api_check_defaults.assertion set. Schema defaults cannot be
// used for these attributes (see the attribute schema for details), so the
// plan is adjusted here instead. Substituting static values keeps every set
// element fully known and stable across plans, which set element correlation
// requires.
func (r *CheckGroupResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// A destroy plan has no planned state to default values in.
	if req.Plan.Raw.IsNull() {
		return
	}

	var configDefaults types.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(apiCheckDefaultsAttributeName), &configDefaults)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// When the configuration contains unknown values (references to
	// attributes of resources that have not been created yet), null values
	// cannot be distinguished from values that will resolve at apply time.
	// Leave the plan untouched; the apply-time re-plan applies the defaults
	// once the configuration values are known.
	if valueContainsUnknown(configDefaults) {
		return
	}

	var defaults types.Object
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root(apiCheckDefaultsAttributeName), &defaults)...)
	if resp.Diagnostics.HasError() {
		return
	}

	defaults = apiCheckDefaultsWithDefaults(ctx, defaults, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root(apiCheckDefaultsAttributeName), defaults)...)
}

func (r *CheckGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CheckGroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group := checkGroupFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.CreateGroup(apiCtx, group)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Check Group",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	// Re-read the group so that the state is based on a full representation,
	// which the create response is not guaranteed to be.
	created, err := r.client.GetGroup(apiCtx, result.ID)
	if err != nil {
		// The group exists now. Record it from the create response, so
		// that Terraform tracks it (as tainted, because of the error)
		// instead of losing track of it.
		checkGroupToModel(ctx, result, &plan, &resp.Diagnostics)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.AddError(
			"Unable to Read Check Group After Creation",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	checkGroupToModel(ctx, created, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CheckGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CheckGroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := decodeNumericID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Check Group ID",
			fmt.Sprintf("Check group ID must be numeric: %s", err),
		)
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	group, err := r.client.GetGroup(apiCtx, id)
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Read Check Group",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	checkGroupToModel(ctx, group, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *CheckGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CheckGroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group := checkGroupFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	_, err := r.client.UpdateGroup(apiCtx, group.ID, group)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Check Group",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	// Re-read the group so that the state is based on a full representation,
	// which the update response is not guaranteed to be.
	updated, err := r.client.GetGroup(apiCtx, group.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Check Group After Update",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	checkGroupToModel(ctx, updated, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CheckGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CheckGroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := decodeNumericID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Check Group ID",
			fmt.Sprintf("Check group ID must be numeric: %s", err),
		)
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	err = r.client.DeleteGroup(apiCtx, id)
	if err != nil {
		if isNotFoundError(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Delete Check Group",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}
}

func (r *CheckGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// checkGroupStrings ensures a non-nil slice so that the value serializes as
// an empty JSON array rather than null, which is what the Checkly API
// expects for unset string collections.
func checkGroupStrings(values []string) []string {
	if values == nil {
		return []string{}
	}

	return values
}

// checkGroupFromModel converts the resource data model to its API
// representation.
func checkGroupFromModel(ctx context.Context, model *CheckGroupResourceModel, diags *diag.Diagnostics) checkly.Group {
	var id int64
	if !model.ID.IsNull() && !model.ID.IsUnknown() {
		id, _ = decodeNumericID(model.ID.ValueString())
	}

	group := checkly.Group{
		ID:                        id,
		Name:                      model.Name.ValueString(),
		Concurrency:               int(model.Concurrency.ValueInt64()),
		Activated:                 model.Activated.ValueBool(),
		Muted:                     model.Muted.ValueBool(),
		RunParallel:               model.RunParallel.ValueBool(),
		Locations:                 checkGroupStrings(stringsFromSet(ctx, model.Locations, diags)),
		Tags:                      checkGroupStrings(stringsFromSet(ctx, model.Tags, diags)),
		EnvironmentVariables:      environmentVariablesFromList(ctx, model.EnvironmentVariable, diags),
		SetupSnippetID:            model.SetupSnippetID.ValueInt64(),
		TearDownSnippetID:         model.TeardownSnippetID.ValueInt64(),
		LocalSetupScript:          model.LocalSetupScript.ValueString(),
		LocalTearDownScript:       model.LocalTeardownScript.ValueString(),
		AlertSettings:             alertSettingsFromObject(ctx, model.AlertSettings, diags),
		UseGlobalAlertSettings:    model.UseGlobalAlertSettings.ValueBool(),
		APICheckDefaults:          apiCheckDefaultsFromObject(ctx, model.APICheckDefaults, diags),
		AlertChannelSubscriptions: alertChannelSubscriptionsFromSet(ctx, model.AlertChannelSubscription, diags),
		RetryStrategy:             retryStrategyFromObject(ctx, model.RetryStrategy, diags),
	}

	// An empty runtime ID means the group uses the account default runtime.
	if runtimeID := model.RuntimeID.ValueString(); runtimeID != "" {
		group.RuntimeID = &runtimeID
	}

	privateLocations := checkGroupStrings(stringsFromSet(ctx, model.PrivateLocations, diags))
	group.PrivateLocations = &privateLocations

	return group
}

// checkGroupToModel updates the model to match the API representation of the
// check group. Values that the API does not track are resolved using the
// values already present in the model (the plan on create/update, the state
// on read).
func checkGroupToModel(ctx context.Context, group *checkly.Group, model *CheckGroupResourceModel, diags *diag.Diagnostics) {
	model.ID = types.StringValue(encodeNumericID(group.ID))
	model.Name = types.StringValue(group.Name)
	model.Concurrency = types.Int64Value(int64(group.Concurrency))
	model.Activated = types.BoolValue(group.Activated)
	model.Muted = types.BoolValue(group.Muted)
	model.RunParallel = types.BoolValue(group.RunParallel)
	model.Locations = setFromStrings(ctx, group.Locations, diags)

	if group.PrivateLocations != nil {
		model.PrivateLocations = setFromStrings(ctx, *group.PrivateLocations, diags)
	} else {
		model.PrivateLocations = setFromStrings(ctx, nil, diags)
	}

	model.Tags = setFromStrings(ctx, group.Tags, diags)
	model.EnvironmentVariable = environmentVariablesToList(ctx, group.EnvironmentVariables, model.EnvironmentVariable, diags)
	model.SetupSnippetID = types.Int64Value(group.SetupSnippetID)
	model.TeardownSnippetID = types.Int64Value(group.TearDownSnippetID)
	model.LocalSetupScript = types.StringValue(group.LocalSetupScript)
	model.LocalTeardownScript = types.StringValue(group.LocalTearDownScript)

	// The API reports no runtime ID when the group uses the account default
	// runtime. In that case the value already present in the model (null or
	// an empty string) is preserved.
	if group.RuntimeID != nil {
		model.RuntimeID = types.StringValue(*group.RuntimeID)
	} else {
		model.RuntimeID = stringOrNullWhenUnknown(model.RuntimeID)
	}

	model.AlertChannelSubscription = alertChannelSubscriptionsToSet(ctx, group.AlertChannelSubscriptions, diags)
	model.AlertSettings = alertSettingsToObject(ctx, group.AlertSettings, model.AlertSettings, diags)
	model.UseGlobalAlertSettings = types.BoolValue(group.UseGlobalAlertSettings)
	model.APICheckDefaults = apiCheckDefaultsToObject(ctx, group.APICheckDefaults, model.APICheckDefaults, diags)
	model.RetryStrategy = retryStrategyToObject(ctx, group.RetryStrategy, model.RetryStrategy, diags)
}
