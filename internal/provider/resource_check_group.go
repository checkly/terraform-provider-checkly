package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
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
	ID                        types.String `tfsdk:"id"`
	Name                      types.String `tfsdk:"name"`
	Concurrency               types.Int64  `tfsdk:"concurrency"`
	Activated                 types.Bool   `tfsdk:"activated"`
	Muted                     types.Bool   `tfsdk:"muted"`
	RunParallel               types.Bool   `tfsdk:"run_parallel"`
	Locations                 types.Set    `tfsdk:"locations"`
	PrivateLocations          types.Set    `tfsdk:"private_locations"`
	Tags                      types.Set    `tfsdk:"tags"`
	EnvironmentVariables      types.List   `tfsdk:"environment_variables"`
	SetupSnippetID            types.Int64  `tfsdk:"setup_snippet_id"`
	TeardownSnippetID         types.Int64  `tfsdk:"teardown_snippet_id"`
	LocalSetupScript          types.String `tfsdk:"local_setup_script"`
	LocalTeardownScript       types.String `tfsdk:"local_teardown_script"`
	RuntimeID                 types.String `tfsdk:"runtime_id"`
	AlertChannelSubscriptions types.Set    `tfsdk:"alert_channel_subscriptions"`
	AlertSettings             types.Object `tfsdk:"alert_settings"`
	UseGlobalAlertSettings    types.Bool   `tfsdk:"use_global_alert_settings"`
	DoubleCheck               types.Bool   `tfsdk:"double_check"`
	APICheckDefaults          types.Object `tfsdk:"api_check_defaults"`
	RetryStrategy             types.Object `tfsdk:"retry_strategy"`
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
			"Both resource types manage the same check groups, so an existing group can be switched over with a " +
			"`moved` block to a `checkly_check_group_v2` resource. See " +
			"[Check groups](https://registry.terraform.io/providers/checkly/checkly/latest/docs/guides/upgrading-to-v2#check-groups) in the v2 upgrade " +
			"guide for how its configuration translates.",
		DeprecationMessage: "Use the `checkly_check_group_v2` resource instead. An existing group can be switched " +
			"over with a `moved` block, without recreating it; see the v2 upgrade guide.",
		Attributes: withCommonCheckAttributes(map[string]schema.Attribute{
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
			"setup_snippet_id":    snippetIDAttributeSchema("The ID of the snippet to run in the setup phase of the group's API checks (at least `1`). When unset, no setup snippet runs."),
			"teardown_snippet_id": snippetIDAttributeSchema("The ID of the snippet to run in the teardown phase of the group's API checks (at least `1`). When unset, no teardown snippet runs."),
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
			alertSettingsAttributeName:    makeAlertSettingsAttributeSchema(AlertSettingsAttributeSchemaOptions{Subject: subjectCheckGroup}),
			apiCheckDefaultsAttributeName: makeAPICheckDefaultsAttributeSchema(),
			retryStrategyAttributeName: makeRetryStrategyAttributeSchema(RetryStrategyAttributeSchemaOptions{
				SupportsOnlyOnNetworkError: true,
				DoubleCheck:                true,
			}),
			doubleCheckAttributeName:              doubleCheckAttributeSchema(),
			alertChannelSubscriptionAttributeName: makeAlertChannelSubscriptionSchema(AlertChannelSubscriptionAttributeSchemaOptions{Subject: subjectCheckGroup}),
			environmentVariableAttributeName: makeEnvironmentVariableSchema(EnvironmentVariableAttributeSchemaOptions{
				Description: "Insert environment variables into the runtime environment. Only relevant for browser checks. Use global environment variables whenever possible.",
			}),
		}, CommonCheckAttributeSchemaOptions{Subject: subjectCheckGroup, RunParallel: true, Locations: true, RuntimeID: true}),
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
// nested in the api_check_defaults.assertions set. Schema defaults cannot be
// used for these attributes (see the attribute schema for details), so the
// plan is adjusted here instead. Substituting static values keeps every set
// element fully known and stable across plans, which set element correlation
// requires. It also keeps the stored values of a group without alert settings
// out of a newly configured alert_settings attribute; see
// planUnconfiguredAlertSettingsAsUnknown.
func (r *CheckGroupResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planNestedDefaults(ctx, req, resp, path.Root(apiCheckDefaultsAttributeName), objectWithSetDefaults("assertions", assertionsWithPropertyDefault))
	planUnconfiguredAlertSettingsAsUnknown(ctx, req, resp)
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

	result, err := r.client.CreateGroupV2(apiCtx, group)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to Create Check Group", err)
		return
	}

	// Re-read the group so that the state is based on a full representation,
	// which the create response is not guaranteed to be. The read gets its
	// own timeout, so that a slow write doesn't leave it too little time.
	readCtx, readCancel := contextWithAPITimeout(ctx)
	defer readCancel()

	created, err := r.client.GetGroupV2(readCtx, result.ID)
	if err != nil {
		// The group exists now. Record it from the create response, so
		// that Terraform tracks it (as tainted, because of the error)
		// instead of losing track of it.
		checkGroupToModel(ctx, result, &plan, &resp.Diagnostics)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		addAPIError(&resp.Diagnostics, "Unable to Read Check Group After Creation", err)
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

	group, err := r.client.GetGroupV2(apiCtx, id)
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		addAPIError(&resp.Diagnostics, "Unable to Read Check Group", err)
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

	_, err := r.client.UpdateGroupV2(apiCtx, group.ID, group)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to Update Check Group", err)
		return
	}

	// Re-read the group so that the state is based on a full representation,
	// which the update response is not guaranteed to be. The read gets its
	// own timeout, so that a slow write doesn't leave it too little time.
	readCtx, readCancel := contextWithAPITimeout(ctx)
	defer readCancel()

	updated, err := r.client.GetGroupV2(readCtx, group.ID)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to Read Check Group After Update", err)
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

	err = r.client.DeleteGroupV2(apiCtx, id)
	if err != nil {
		if isNotFoundError(err) {
			return
		}
		addAPIError(&resp.Diagnostics, "Unable to Delete Check Group", err)
		return
	}
}

func (r *CheckGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// checkGroupFromModel converts the resource data model to its API
// representation. Every update replaces the whole group, so settings the group
// does not have are sent as nil to clear them.
//
// double_check is not sent: the API's default of false clears a stored flag,
// and when double_check is true the planned retry_strategy already holds the
// strategy it stands for.
func checkGroupFromModel(ctx context.Context, model *CheckGroupResourceModel, diags *diag.Diagnostics) checkly.GroupV2 {
	var id int64
	if !model.ID.IsNull() && !model.ID.IsUnknown() {
		id, _ = decodeNumericID(model.ID.ValueString())
	}

	group := checkly.GroupV2{
		ID:                        id,
		Name:                      model.Name.ValueString(),
		Concurrency:               int(model.Concurrency.ValueInt64()),
		Activated:                 model.Activated.ValueBool(),
		Muted:                     model.Muted.ValueBool(),
		RunParallel:               model.RunParallel.ValueBoolPointer(),
		Locations:                 stringsFromSetOrEmpty(ctx, model.Locations, diags),
		Tags:                      stringsFromSetOrEmpty(ctx, model.Tags, diags),
		EnvironmentVariables:      environmentVariablesFromList(ctx, model.EnvironmentVariables, diags),
		SetupSnippetID:            pointerOrNilWhenZero(model.SetupSnippetID.ValueInt64()),
		TearDownSnippetID:         pointerOrNilWhenZero(model.TeardownSnippetID.ValueInt64()),
		LocalSetupScript:          pointerOrNilWhenZero(model.LocalSetupScript.ValueString()),
		LocalTearDownScript:       pointerOrNilWhenZero(model.LocalTeardownScript.ValueString()),
		UseGlobalAlertSettings:    model.UseGlobalAlertSettings.ValueBoolPointer(),
		APICheckDefaults:          apiCheckDefaultsFromObject(ctx, model.APICheckDefaults, diags),
		AlertChannelSubscriptions: alertChannelSubscriptionsFromSet(ctx, model.AlertChannelSubscriptions, diags),
		RetryStrategy:             retryStrategyFromObject(ctx, model.RetryStrategy, diags),
	}

	// An empty escalation_type is what reading a group without alert settings
	// stores. Nil keeps the group that way, and the API rejects an empty
	// object.
	if alertSettings := alertSettingsFromObject(ctx, model.AlertSettings, diags); alertSettings.EscalationType != "" {
		group.AlertSettings = &alertSettings
	}

	// An empty runtime ID means the group uses the account default runtime.
	if runtimeID := model.RuntimeID.ValueString(); runtimeID != "" {
		group.RuntimeID = &runtimeID
	}

	privateLocations := stringsFromSetOrEmpty(ctx, model.PrivateLocations, diags)
	group.PrivateLocations = &privateLocations

	return group
}

// checkGroupToModel updates the model to match the API representation of the
// check group. Values that the API does not track are resolved using the
// values already present in the model (the plan on create/update, the state
// on read).
func checkGroupToModel(ctx context.Context, group *checkly.GroupV2, model *CheckGroupResourceModel, diags *diag.Diagnostics) {
	model.ID = types.StringValue(encodeNumericID(group.ID))
	model.Name = types.StringValue(group.Name)
	model.Concurrency = types.Int64Value(int64(group.Concurrency))
	model.Activated = types.BoolValue(group.Activated)
	model.Muted = types.BoolValue(group.Muted)
	model.RunParallel = types.BoolValue(valueOrZero(group.RunParallel))
	model.Locations = stringSetFromAPINullIfUnset(ctx, group.Locations, model.Locations, diags)
	model.PrivateLocations = stringSetFromAPINullIfUnset(ctx, valueOrZero(group.PrivateLocations), model.PrivateLocations, diags)
	model.Tags = stringSetFromAPINullIfUnset(ctx, group.Tags, model.Tags, diags)
	model.EnvironmentVariables = environmentVariablesToList(ctx, group.EnvironmentVariables, model.EnvironmentVariables, diags)
	model.SetupSnippetID = int64ValueOrNull(valueOrZero(group.SetupSnippetID))
	model.TeardownSnippetID = int64ValueOrNull(valueOrZero(group.TearDownSnippetID))
	model.LocalSetupScript = types.StringValue(valueOrZero(group.LocalSetupScript))
	model.LocalTeardownScript = types.StringValue(valueOrZero(group.LocalTearDownScript))

	// The API reports no runtime ID when the group uses the account default
	// runtime. In that case the value already present in the model (null or
	// an empty string) is preserved.
	if group.RuntimeID != nil {
		model.RuntimeID = types.StringValue(*group.RuntimeID)
	} else {
		model.RuntimeID = stringOrNullWhenUnknown(model.RuntimeID)
	}

	model.AlertChannelSubscriptions = alertChannelSubscriptionsToSet(ctx, group.AlertChannelSubscriptions, model.AlertChannelSubscriptions, diags)
	model.AlertSettings = alertSettingsToObject(ctx, valueOrZero(group.AlertSettings), model.AlertSettings, diags)
	model.UseGlobalAlertSettings = types.BoolValue(valueOrZero(group.UseGlobalAlertSettings))

	// The SDK's GroupV2 has no double_check, so the value in the model
	// stands, and a value that was never set, as on import, is false. A
	// change made outside Terraform shows up through retry_strategy.
	model.DoubleCheck = types.BoolValue(model.DoubleCheck.ValueBool())

	model.APICheckDefaults = apiCheckDefaultsToObject(ctx, group.APICheckDefaults, model.APICheckDefaults, diags)
	model.RetryStrategy = retryStrategyToObject(ctx, group.RetryStrategy, model.RetryStrategy, diags)
}

// pointerOrNilWhenZero returns a pointer to value, or nil for the zero value
// of its type.
func pointerOrNilWhenZero[T comparable](value T) *T {
	var zero T
	if value == zero {
		return nil
	}

	return &value
}

// valueOrZero returns the value of pointer, or the zero value of its type
// when pointer is nil.
func valueOrZero[T any](pointer *T) T {
	if pointer == nil {
		var zero T
		return zero
	}

	return *pointer
}
