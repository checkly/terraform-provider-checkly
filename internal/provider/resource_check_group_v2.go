package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

var (
	_ resource.Resource                 = (*CheckGroupV2Resource)(nil)
	_ resource.ResourceWithConfigure    = (*CheckGroupV2Resource)(nil)
	_ resource.ResourceWithUpgradeState = (*CheckGroupV2Resource)(nil)
	_ resource.ResourceWithImportState  = (*CheckGroupV2Resource)(nil)
	_ resource.ResourceWithModifyPlan   = (*CheckGroupV2Resource)(nil)
)

// NewCheckGroupV2Resource returns a new checkly_check_group_v2 resource.
func NewCheckGroupV2Resource() resource.Resource {
	return &CheckGroupV2Resource{}
}

// CheckGroupV2Resource implements the checkly_check_group_v2 resource.
type CheckGroupV2Resource struct {
	client checkly.Client
}

// CheckGroupV2ResourceModel describes the resource data model.
type CheckGroupV2ResourceModel struct {
	ID                        types.String `tfsdk:"id"`
	Name                      types.String `tfsdk:"name"`
	Concurrency               types.Int64  `tfsdk:"concurrency"`
	Activated                 types.Bool   `tfsdk:"activated"`
	Muted                     types.Bool   `tfsdk:"muted"`
	Tags                      types.Set    `tfsdk:"tags"`
	EnvironmentVariable       types.List   `tfsdk:"environment_variable"`
	SetupScript               types.Object `tfsdk:"setup_script"`
	TeardownScript            types.Object `tfsdk:"teardown_script"`
	DefaultRuntime            types.Object `tfsdk:"default_runtime"`
	EnforceAlertSettings      types.Object `tfsdk:"enforce_alert_settings"`
	EnforceLocations          types.Object `tfsdk:"enforce_locations"`
	EnforceRetryStrategy      types.Object `tfsdk:"enforce_retry_strategy"`
	EnforceSchedulingStrategy types.Object `tfsdk:"enforce_scheduling_strategy"`
	APICheckDefaults          types.Object `tfsdk:"api_check_defaults"`
}

var checkGroupV2ScriptAttrTypes = map[string]attr.Type{
	"snippet_id":    types.Int64Type,
	"inline_script": types.StringType,
}

var checkGroupV2DefaultRuntimeAttrTypes = map[string]attr.Type{
	"runtime_id": types.StringType,
}

var checkGroupV2AlertChannelSubscriptionSetType = types.SetType{
	ElemType: types.ObjectType{AttrTypes: alertChannelSubscriptionAttrTypes},
}

var checkGroupV2EnforceAlertSettingsAttrTypes = map[string]attr.Type{
	"enabled":                             types.BoolType,
	alertSettingsAttributeName:            types.ObjectType{AttrTypes: alertSettingsAttrTypes},
	"use_global_alert_settings":           types.BoolType,
	alertChannelSubscriptionAttributeName: checkGroupV2AlertChannelSubscriptionSetType,
}

var checkGroupV2EnforceLocationsAttrTypes = map[string]attr.Type{
	"enabled":           types.BoolType,
	"locations":         types.SetType{ElemType: types.StringType},
	"private_locations": types.SetType{ElemType: types.StringType},
}

var checkGroupV2EnforceRetryStrategyAttrTypes = map[string]attr.Type{
	"enabled":                  types.BoolType,
	retryStrategyAttributeName: types.ObjectType{AttrTypes: retryStrategyAttrTypes(true)},
}

var checkGroupV2EnforceSchedulingStrategyAttrTypes = map[string]attr.Type{
	"enabled":      types.BoolType,
	"run_parallel": types.BoolType,
}

// checkGroupV2DisabledEnforcementObject returns the canonical attribute
// value for a disabled enforcement: enabled is false and all other members
// are null. Enforcement attributes default to this value, which means that
// removing an enforcement from the configuration plans a change back to the
// disabled enforcement.
func checkGroupV2DisabledEnforcementObject(attrTypes map[string]attr.Type) types.Object {
	attrValues := make(map[string]attr.Value, len(attrTypes))

	for name, attrType := range attrTypes {
		if name == "enabled" {
			attrValues[name] = types.BoolValue(false)
			continue
		}

		switch t := attrType.(type) {
		case types.ObjectType:
			attrValues[name] = types.ObjectNull(t.AttrTypes)
		case types.SetType:
			attrValues[name] = types.SetNull(t.ElemType)
		default:
			switch attrType {
			case types.BoolType:
				attrValues[name] = types.BoolNull()
			default:
				panic(fmt.Sprintf("unsupported enforcement attribute type %T", attrType))
			}
		}
	}

	return types.ObjectValueMust(attrTypes, attrValues)
}

func (r *CheckGroupV2Resource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_check_group_v2"
}

func (r *CheckGroupV2Resource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	scriptAttributes := map[string]schema.Attribute{
		"snippet_id": schema.Int64Attribute{
			Optional:    true,
			Description: "The ID of a code snippet. Code snippets are not available for new plans.",
			Validators: []validator.Int64{
				int64validator.AtLeast(1),
			},
		},
		"inline_script": schema.StringAttribute{
			Optional:    true,
			Description: "A valid piece of Node.js code.",
		},
	}

	resp.Schema = schema.Schema{
		Version: sdkv2SchemaVersion + 1,
		Description: "Check groups organize related checks together and allow optional shared configuration. " +
			"\n\n" +
			"Unlike `checkly_check_group` (v1), which always imposed implicit defaults for various settings, " +
			"`checkly_check_group_v2` does not override any check settings by default. " +
			"A group can be as minimal as just a name. " +
			"Use the optional `enforce_*` attributes only when you explicitly want the group to override individual " +
			"check settings." +
			"\n\n" +
			"The new `checkly_check_group_v2` resource should always be preferred over the old " +
			"`checkly_check_group` resource.",
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
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(1),
				Description: "Determines the number of checks to run concurrently when triggering the check group " +
					"via CI/CD or the API. " +
					"(Default `1`).",
			},
			"activated": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
				Description: "Determines whether activated checks in the group should run or not. Deactivating the " +
					"group will prevent all checks in the group from running, regardless of their activated state. " +
					"(Default `true`).",
			},
			"muted": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				Description: "Determines if any notifications will be sent out when a check in this group fails " +
					"and/or recovers. Muting the group will deactivate all notifications for all checks in the " +
					"group, regardless of their muted state. " +
					"(Default `false`).",
			},
			"tags": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
				Description: "Additional tags to append to all checks in the group.",
			},
			"enforce_alert_settings": schema.SingleNestedAttribute{
				Optional: true,
				Computed: true,
				Default:  objectdefault.StaticValue(checkGroupV2DisabledEnforcementObject(checkGroupV2EnforceAlertSettingsAttrTypes)),
				PlanModifiers: []planmodifier.Object{
					checkGroupV2DisabledEnforcementIgnoredFields,
				},
				Description: "Enforces alert settings for the whole group. " +
					"Overrides check configuration.",
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Required:    true,
						Description: "Determines whether the enforced alert settings should be active.",
					},
					alertSettingsAttributeName: makeAlertSettingsAttributeSchema(AlertSettingsAttributeSchemaOptions{}),
					"use_global_alert_settings": schema.BoolAttribute{
						Optional: true,
						Computed: true,
						Default:  booldefault.StaticBool(false),
						Description: "Whether to use account level alert settings instead of the group's alert " +
							"settings." +
							"Default (`false`).",
					},
					alertChannelSubscriptionAttributeName: schema.SetNestedAttribute{
						Optional: true,
						Description: "An array of channel IDs and whether they're activated or not. If you don't " +
							"set at least one alert channel subscription for your group, we won't be able to alert " +
							"you even if a check in the group starts failing.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"channel_id": schema.Int64Attribute{
									Required:    true,
									Description: "The ID of the alert channel.",
								},
								"activated": schema.BoolAttribute{
									Required:    true,
									Description: "Whether an alert should be sent to this channel.",
								},
							},
						},
					},
				},
			},
			"enforce_locations": schema.SingleNestedAttribute{
				Optional: true,
				Computed: true,
				Default:  objectdefault.StaticValue(checkGroupV2DisabledEnforcementObject(checkGroupV2EnforceLocationsAttrTypes)),
				PlanModifiers: []planmodifier.Object{
					checkGroupV2DisabledEnforcementIgnoredFields,
				},
				Description: "Enforces public and private locations for the whole group. " +
					"Overrides check configuration.",
				Validators: []validator.Object{
					checkGroupV2EnforceLocationsValidator{},
				},
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Required:    true,
						Description: "Determines whether the enforced locations should be active.",
					},
					"locations": schema.SetAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Description: "An array of one or more data center locations where to run the checks.",
					},
					"private_locations": schema.SetAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Description: "An array of one or more private locations slugs.",
					},
				},
			},
			"enforce_retry_strategy": schema.SingleNestedAttribute{
				Optional: true,
				Computed: true,
				Default:  objectdefault.StaticValue(checkGroupV2DisabledEnforcementObject(checkGroupV2EnforceRetryStrategyAttrTypes)),
				PlanModifiers: []planmodifier.Object{
					checkGroupV2DisabledEnforcementIgnoredFields,
				},
				Description: "Enforces a retry strategy for the whole group. " +
					"Overrides check configuration.",
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Required:    true,
						Description: "Determines whether the enforced retry strategy should be active.",
					},
					retryStrategyAttributeName: makeRetryStrategyAttributeSchema(RetryStrategyAttributeSchemaOptions{
						SupportsOnlyOnNetworkError: true,
						Required:                   true,
					}),
				},
			},
			"enforce_scheduling_strategy": schema.SingleNestedAttribute{
				Optional: true,
				Computed: true,
				Default:  objectdefault.StaticValue(checkGroupV2DisabledEnforcementObject(checkGroupV2EnforceSchedulingStrategyAttrTypes)),
				PlanModifiers: []planmodifier.Object{
					checkGroupV2DisabledEnforcementIgnoredFields,
				},
				Description: "Enforces a scheduling strategy for the whole group. " +
					"Overrides check configuration.",
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Required:    true,
						Description: "Determines whether the enforced scheduling strategy should be active.",
					},
					"run_parallel": schema.BoolAttribute{
						Required:    true,
						Description: "Determines if the checks in the group should run in all selected locations in parallel or round-robin.",
					},
				},
			},
			apiCheckDefaultsAttributeName: makeAPICheckDefaultsAttributeSchema(),
			"setup_script": schema.SingleNestedAttribute{
				Optional: true,
				Description: "A script to run in the setup phase of an API check. " +
					"Runs in addition to the check's own setup script.",
				Validators: []validator.Object{
					checkGroupV2ScriptValidator{},
				},
				Attributes: scriptAttributes,
			},
			"teardown_script": schema.SingleNestedAttribute{
				Optional: true,
				Description: "A script to run in the teardown phase of an API check. " +
					"Runs in addition to the check's own teardown script.",
				Validators: []validator.Object{
					checkGroupV2ScriptValidator{},
				},
				Attributes: scriptAttributes,
			},
			"default_runtime": schema.SingleNestedAttribute{
				Optional: true,
				Description: "Sets a default runtime for the group. " +
					"Used as a fallback when a check belonging to the group has no runtime set. " +
					"Takes precedence over the account default runtime.",
				Attributes: map[string]schema.Attribute{
					"runtime_id": schema.StringAttribute{
						Required:    true,
						Description: "The runtime ID.",
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			environmentVariableAttributeName: makeEnvironmentVariableSchema(EnvironmentVariableAttributeSchemaOptions{
				Description: "Insert environment variables into the runtime environment of checks in the group. " +
					"Only relevant for check types that support environment variables. " +
					"Use global environment variables whenever possible.",
			}),
		},
	}
}

// UpgradeState upgrades state written by the provider's SDK v2
// implementation; see sdkv2StateUpgraders.
func (r *CheckGroupV2Resource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return sdkv2StateUpgraders(ctx, r)
}

func (r *CheckGroupV2Resource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

// ModifyPlan applies default values to the optional+computed attributes
// nested in the api_check_defaults.assertion set. Schema defaults cannot be
// used for these attributes (see the attribute schema for details), so the
// plan is adjusted here instead. Substituting static values keeps every set
// element fully known and stable across plans, which set element correlation
// requires.
func (r *CheckGroupV2Resource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planNestedDefaults(ctx, req, resp, path.Root(apiCheckDefaultsAttributeName), objectWithSetDefaults("assertion", assertionsWithPropertyDefault))
}

func (r *CheckGroupV2Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CheckGroupV2ResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group := checkGroupV2FromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.CreateGroupV2(apiCtx, group)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Check Group (v2)",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	// Re-read the group so that the state is based on a full representation,
	// which the create response is not guaranteed to be.
	created, err := r.client.GetGroupV2(apiCtx, result.ID)
	if err != nil {
		// The group exists now. Record it from the create response, so
		// that Terraform tracks it (as tainted, because of the error)
		// instead of losing track of it.
		checkGroupV2ToModel(ctx, result, &plan, &resp.Diagnostics)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.AddError(
			"Unable to Read Check Group (v2) After Creation",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	checkGroupV2ToModel(ctx, created, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CheckGroupV2Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CheckGroupV2ResourceModel

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
		resp.Diagnostics.AddError(
			"Unable to Read Check Group (v2)",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	checkGroupV2ToModel(ctx, group, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *CheckGroupV2Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CheckGroupV2ResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group := checkGroupV2FromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	_, err := r.client.UpdateGroupV2(apiCtx, group.ID, group)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Check Group (v2)",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	// Re-read the group so that the state is based on a full representation,
	// which the update response is not guaranteed to be.
	updated, err := r.client.GetGroupV2(apiCtx, group.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Check Group (v2) After Update",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	checkGroupV2ToModel(ctx, updated, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CheckGroupV2Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CheckGroupV2ResourceModel

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
		resp.Diagnostics.AddError(
			"Unable to Delete Check Group (v2)",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}
}

func (r *CheckGroupV2Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// checkGroupV2ScriptValidator validates that a setup_script/teardown_script
// object configures exactly one of snippet_id and inline_script.
type checkGroupV2ScriptValidator struct{}

func (v checkGroupV2ScriptValidator) Description(ctx context.Context) string {
	return "exactly one of \"snippet_id\" or \"inline_script\" must be specified"
}

func (v checkGroupV2ScriptValidator) MarkdownDescription(ctx context.Context) string {
	return "exactly one of `snippet_id` or `inline_script` must be specified"
}

func (v checkGroupV2ScriptValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	attrs := req.ConfigValue.Attributes()
	snippetID := attrs["snippet_id"]
	inlineScript := attrs["inline_script"]

	// Unknown values may turn out to be either null or set; there is nothing
	// to validate yet.
	if snippetID.IsUnknown() || inlineScript.IsUnknown() {
		return
	}

	configured := 0
	if !snippetID.IsNull() {
		configured++
	}
	if !inlineScript.IsNull() {
		configured++
	}

	if configured != 1 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Script",
			"Exactly one of \"snippet_id\" or \"inline_script\" must be specified.",
		)
	}
}

// checkGroupV2EnforceLocationsValidator validates that a configured
// enforce_locations attribute specifies at least one location in locations or
// private_locations. Empty sets are rejected because the Checkly API treats a
// group without locations as not enforcing locations at all.
type checkGroupV2EnforceLocationsValidator struct{}

func (v checkGroupV2EnforceLocationsValidator) Description(ctx context.Context) string {
	return "at least one of \"locations\" or \"private_locations\" must contain a location"
}

func (v checkGroupV2EnforceLocationsValidator) MarkdownDescription(ctx context.Context) string {
	return "at least one of `locations` or `private_locations` must contain a location"
}

func (v checkGroupV2EnforceLocationsValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	attrs := req.ConfigValue.Attributes()
	locations := attrs["locations"]
	privateLocations := attrs["private_locations"]

	// Unknown values may turn out to be either null or set; there is nothing
	// to validate yet.
	if locations.IsUnknown() || privateLocations.IsUnknown() {
		return
	}

	hasLocation := func(value attr.Value) bool {
		set, ok := value.(types.Set)
		return ok && !set.IsNull() && len(set.Elements()) != 0
	}

	if !hasLocation(locations) && !hasLocation(privateLocations) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Missing Locations",
			"At least one of \"locations\" or \"private_locations\" must contain a location.",
		)
	}
}

// checkGroupV2EnforcementEnabled reports whether an enforcement attribute
// value is present and has its enabled flag set.
func checkGroupV2EnforcementEnabled(obj types.Object) bool {
	if obj.IsNull() || obj.IsUnknown() {
		return false
	}

	return boolAttrValue(obj.Attributes()["enabled"])
}

// checkGroupV2ScriptFromObject converts a setup_script/teardown_script
// attribute value to its API representation.
func checkGroupV2ScriptFromObject(obj types.Object) (snippetID *int64, inlineScript *string) {
	attrs := obj.Attributes()

	if v, ok := attrs["snippet_id"].(types.Int64); ok && !v.IsNull() && !v.IsUnknown() {
		id := v.ValueInt64()
		if id != 0 {
			snippetID = &id
		}
	}

	if v, ok := attrs["inline_script"].(types.String); ok {
		inlineScript = optionalStringPointer(v)
	}

	return snippetID, inlineScript
}

// checkGroupV2ObjectWithUnknownsNulled returns a copy of an object value with
// unknown top-level members resolved to null, preserving the object type.
// Unknown values cannot be stored in state, so a preserved planned object
// whose optional+computed members lack defaults must have those members
// resolved before storing. Only top-level members need resolving: deeper
// optional+computed members all have static defaults, so they are known
// whenever their parent is known.
func checkGroupV2ObjectWithUnknownsNulled(ctx context.Context, obj types.Object) types.Object {
	if obj.IsNull() || obj.IsUnknown() {
		return obj
	}

	attrTypes := obj.AttributeTypes(ctx)
	attrValues := make(map[string]attr.Value, len(attrTypes))

	for name, value := range obj.Attributes() {
		if !value.IsUnknown() {
			attrValues[name] = value
			continue
		}

		switch t := attrTypes[name].(type) {
		case types.ObjectType:
			attrValues[name] = types.ObjectNull(t.AttrTypes)
		case types.SetType:
			attrValues[name] = types.SetNull(t.ElemType)
		default:
			switch attrTypes[name] {
			case types.BoolType:
				attrValues[name] = types.BoolNull()
			case types.StringType:
				attrValues[name] = types.StringNull()
			case types.Int64Type:
				attrValues[name] = types.Int64Null()
			default:
				panic(fmt.Sprintf("unsupported attribute type %T", attrTypes[name]))
			}
		}
	}

	return types.ObjectValueMust(attrTypes, attrValues)
}

// checkGroupV2PriorMember returns the named member of a prior enforcement
// value, or nil when the prior value itself is null or unknown.
func checkGroupV2PriorMember(prior types.Object, name string) attr.Value {
	if prior.IsNull() || prior.IsUnknown() {
		return nil
	}

	return prior.Attributes()[name]
}

// checkGroupV2FromModel converts the resource data model to its API
// representation.
func checkGroupV2FromModel(ctx context.Context, model *CheckGroupV2ResourceModel, diags *diag.Diagnostics) checkly.GroupV2 {
	var id int64
	if !model.ID.IsNull() && !model.ID.IsUnknown() {
		id, _ = decodeNumericID(model.ID.ValueString())
	}

	group := checkly.GroupV2{
		ID:                   id,
		Name:                 model.Name.ValueString(),
		Concurrency:          int(model.Concurrency.ValueInt64()),
		Activated:            model.Activated.ValueBool(),
		Muted:                model.Muted.ValueBool(),
		Tags:                 stringsFromSetOrEmpty(ctx, model.Tags, diags),
		APICheckDefaults:     apiCheckDefaultsFromObject(ctx, model.APICheckDefaults, diags),
		EnvironmentVariables: environmentVariablesFromList(ctx, model.EnvironmentVariable, diags),
	}

	if runtime := model.DefaultRuntime; !runtime.IsNull() && !runtime.IsUnknown() {
		if v, ok := runtime.Attributes()["runtime_id"].(types.String); ok {
			group.RuntimeID = optionalStringPointer(v)
		}
	}

	if script := model.SetupScript; !script.IsNull() && !script.IsUnknown() {
		group.SetupSnippetID, group.LocalSetupScript = checkGroupV2ScriptFromObject(script)
	}

	if script := model.TeardownScript; !script.IsNull() && !script.IsUnknown() {
		group.TearDownSnippetID, group.LocalTearDownScript = checkGroupV2ScriptFromObject(script)
	}

	if checkGroupV2EnforcementEnabled(model.EnforceAlertSettings) {
		attrs := model.EnforceAlertSettings.Attributes()

		if v, ok := attrs[alertSettingsAttributeName].(types.Object); ok && !v.IsNull() && !v.IsUnknown() {
			settings := alertSettingsFromObject(ctx, v, diags)
			group.AlertSettings = &settings
		}

		if v, ok := attrs["use_global_alert_settings"].(types.Bool); ok && !v.IsNull() && !v.IsUnknown() {
			val := v.ValueBool()
			group.UseGlobalAlertSettings = &val
		}

		if v, ok := attrs[alertChannelSubscriptionAttributeName].(types.Set); ok && !v.IsNull() && !v.IsUnknown() {
			group.AlertChannelSubscriptions = alertChannelSubscriptionsFromSet(ctx, v, diags)
		}
	}

	if checkGroupV2EnforcementEnabled(model.EnforceLocations) {
		attrs := model.EnforceLocations.Attributes()

		if v, ok := attrs["locations"].(types.Set); ok {
			group.Locations = stringsFromSetOrEmpty(ctx, v, diags)
		}

		if v, ok := attrs["private_locations"].(types.Set); ok {
			privateLocations := stringsFromSetOrEmpty(ctx, v, diags)
			group.PrivateLocations = &privateLocations
		}
	}

	if checkGroupV2EnforcementEnabled(model.EnforceRetryStrategy) {
		retryStrategy, _ := model.EnforceRetryStrategy.Attributes()[retryStrategyAttributeName].(types.Object)
		group.RetryStrategy = retryStrategyFromObject(ctx, retryStrategy, diags)
	} else {
		// The FALLBACK strategy tells the API that the group does not
		// enforce a retry strategy and checks use their own.
		group.RetryStrategy = &checkly.RetryStrategy{
			Type: "FALLBACK",
		}
	}

	if checkGroupV2EnforcementEnabled(model.EnforceSchedulingStrategy) {
		if v, ok := model.EnforceSchedulingStrategy.Attributes()["run_parallel"].(types.Bool); ok && !v.IsNull() && !v.IsUnknown() {
			val := v.ValueBool()
			group.RunParallel = &val
		}
	}

	return group
}

// checkGroupV2ToModel updates the model to match the API representation of
// the check group. Values that the API does not track are resolved using the
// values already present in the model (the plan on create/update, the state
// on read).
func checkGroupV2ToModel(ctx context.Context, group *checkly.GroupV2, model *CheckGroupV2ResourceModel, diags *diag.Diagnostics) {
	model.ID = types.StringValue(encodeNumericID(group.ID))
	model.Name = types.StringValue(group.Name)
	model.Concurrency = types.Int64Value(int64(group.Concurrency))
	model.Activated = types.BoolValue(group.Activated)
	model.Muted = types.BoolValue(group.Muted)
	model.Tags = setFromStrings(ctx, group.Tags, diags)
	model.EnvironmentVariable = environmentVariablesToList(ctx, group.EnvironmentVariables, model.EnvironmentVariable, diags)
	model.SetupScript = checkGroupV2ScriptToObject(group.SetupSnippetID, group.LocalSetupScript)
	model.TeardownScript = checkGroupV2ScriptToObject(group.TearDownSnippetID, group.LocalTearDownScript)
	model.DefaultRuntime = checkGroupV2DefaultRuntimeToObject(group.RuntimeID)
	model.EnforceAlertSettings = checkGroupV2EnforceAlertSettingsToObject(ctx, group, model.EnforceAlertSettings, diags)
	model.EnforceLocations = checkGroupV2EnforceLocationsToObject(ctx, group, model.EnforceLocations, diags)
	model.EnforceRetryStrategy = checkGroupV2EnforceRetryStrategyToObject(ctx, group, model.EnforceRetryStrategy, diags)
	model.EnforceSchedulingStrategy = checkGroupV2EnforceSchedulingStrategyToObject(ctx, group, model.EnforceSchedulingStrategy, diags)
	model.APICheckDefaults = apiCheckDefaultsToObject(ctx, group.APICheckDefaults, model.APICheckDefaults, diags)
}

// checkGroupV2ScriptToObject converts the API representation of a
// setup/teardown script to a setup_script/teardown_script attribute value,
// which is null when the group has no script.
func checkGroupV2ScriptToObject(snippetID *int64, inlineScript *string) types.Object {
	if snippetID == nil && inlineScript == nil {
		return types.ObjectNull(checkGroupV2ScriptAttrTypes)
	}

	return types.ObjectValueMust(checkGroupV2ScriptAttrTypes, map[string]attr.Value{
		"snippet_id":    types.Int64PointerValue(snippetID),
		"inline_script": types.StringPointerValue(inlineScript),
	})
}

// checkGroupV2DefaultRuntimeToObject converts the API representation of a
// group runtime to a default_runtime attribute value, which is null when the
// group has no default runtime.
func checkGroupV2DefaultRuntimeToObject(runtimeID *string) types.Object {
	if runtimeID == nil {
		return types.ObjectNull(checkGroupV2DefaultRuntimeAttrTypes)
	}

	return types.ObjectValueMust(checkGroupV2DefaultRuntimeAttrTypes, map[string]attr.Value{
		"runtime_id": types.StringPointerValue(runtimeID),
	})
}

// checkGroupV2EnforceAlertSettingsToObject converts the API representation
// of a group's alert settings to the enforce_alert_settings attribute value.
// Members that the API does not track (for example, details of a disabled
// enforcement) are preserved from prior; unknown prior values resolve to
// null.
func checkGroupV2EnforceAlertSettingsToObject(ctx context.Context, group *checkly.GroupV2, prior types.Object, diags *diag.Diagnostics) types.Object {
	priorAlertSettings, ok := checkGroupV2PriorMember(prior, alertSettingsAttributeName).(types.Object)
	if !ok || priorAlertSettings.IsUnknown() {
		priorAlertSettings = types.ObjectNull(alertSettingsAttrTypes)
	}

	useGlobal, ok := checkGroupV2PriorMember(prior, "use_global_alert_settings").(types.Bool)
	if !ok || useGlobal.IsUnknown() {
		useGlobal = types.BoolNull()
	}

	subscriptions, ok := checkGroupV2PriorMember(prior, alertChannelSubscriptionAttributeName).(types.Set)
	if !ok || subscriptions.IsUnknown() {
		subscriptions = types.SetNull(checkGroupV2AlertChannelSubscriptionSetType.ElemType)
	}

	enforced := group.AlertSettings != nil ||
		group.UseGlobalAlertSettings != nil ||
		len(group.AlertChannelSubscriptions) != 0

	// A known prior alert_settings object may still have unknown members
	// (optional+computed attributes without defaults, such as reminders, that
	// were absent from the plan). When the prior object is preserved because
	// the API tracks no value for it, those members resolve to null.
	alertSettings := checkGroupV2ObjectWithUnknownsNulled(ctx, priorAlertSettings)

	if enforced {
		if group.AlertSettings != nil {
			alertSettings = alertSettingsToObject(ctx, *group.AlertSettings, priorAlertSettings, diags)
		}

		if group.UseGlobalAlertSettings != nil {
			useGlobal = types.BoolValue(*group.UseGlobalAlertSettings)
		}

		if len(group.AlertChannelSubscriptions) != 0 {
			subscriptions = alertChannelSubscriptionsToSet(ctx, group.AlertChannelSubscriptions, diags)
		} else if !subscriptions.IsNull() && len(subscriptions.Elements()) != 0 {
			// The API reports no subscriptions, so the prior non-empty value
			// is stale.
			subscriptions = types.SetNull(checkGroupV2AlertChannelSubscriptionSetType.ElemType)
		}
	}

	obj, d := types.ObjectValue(checkGroupV2EnforceAlertSettingsAttrTypes, map[string]attr.Value{
		"enabled":                             types.BoolValue(enforced),
		alertSettingsAttributeName:            alertSettings,
		"use_global_alert_settings":           useGlobal,
		alertChannelSubscriptionAttributeName: subscriptions,
	})
	diags.Append(d...)

	return obj
}

// checkGroupV2EnforceLocationsToObject converts the API representation of a
// group's locations to the enforce_locations attribute value. Members that
// the API does not track (for example, details of a disabled enforcement)
// are preserved from prior; unknown prior values resolve to null.
func checkGroupV2EnforceLocationsToObject(ctx context.Context, group *checkly.GroupV2, prior types.Object, diags *diag.Diagnostics) types.Object {
	locations, ok := checkGroupV2PriorMember(prior, "locations").(types.Set)
	if !ok || locations.IsUnknown() {
		locations = types.SetNull(types.StringType)
	}

	privateLocations, ok := checkGroupV2PriorMember(prior, "private_locations").(types.Set)
	if !ok || privateLocations.IsUnknown() {
		privateLocations = types.SetNull(types.StringType)
	}

	hasLocations := len(group.Locations) != 0
	hasPrivateLocations := group.PrivateLocations != nil && len(*group.PrivateLocations) != 0
	enforced := hasLocations || hasPrivateLocations

	if enforced {
		if hasLocations {
			locations = setFromStrings(ctx, group.Locations, diags)
		} else if !locations.IsNull() && len(locations.Elements()) != 0 {
			locations = types.SetNull(types.StringType)
		}

		if hasPrivateLocations {
			privateLocations = setFromStrings(ctx, *group.PrivateLocations, diags)
		} else if !privateLocations.IsNull() && len(privateLocations.Elements()) != 0 {
			privateLocations = types.SetNull(types.StringType)
		}
	}

	obj, d := types.ObjectValue(checkGroupV2EnforceLocationsAttrTypes, map[string]attr.Value{
		"enabled":           types.BoolValue(enforced),
		"locations":         locations,
		"private_locations": privateLocations,
	})
	diags.Append(d...)

	return obj
}

// checkGroupV2EnforceRetryStrategyToObject converts the API representation
// of a group's retry strategy to the enforce_retry_strategy attribute value.
// A FALLBACK strategy means the group does not enforce a retry strategy. The
// retry strategy of a disabled enforcement is preserved from prior; unknown
// prior values resolve to null.
func checkGroupV2EnforceRetryStrategyToObject(ctx context.Context, group *checkly.GroupV2, prior types.Object, diags *diag.Diagnostics) types.Object {
	priorRetryStrategy, ok := checkGroupV2PriorMember(prior, retryStrategyAttributeName).(types.Object)
	if !ok || priorRetryStrategy.IsUnknown() {
		priorRetryStrategy = types.ObjectNull(retryStrategyAttrTypes(true))
	}

	enforced := group.RetryStrategy == nil || group.RetryStrategy.Type != "FALLBACK"

	retryStrategy := priorRetryStrategy
	if enforced {
		retryStrategy = retryStrategyToObject(ctx, group.RetryStrategy, priorRetryStrategy, diags)
	}

	obj, d := types.ObjectValue(checkGroupV2EnforceRetryStrategyAttrTypes, map[string]attr.Value{
		"enabled":                  types.BoolValue(enforced),
		retryStrategyAttributeName: retryStrategy,
	})
	diags.Append(d...)

	return obj
}

// checkGroupV2EnforceSchedulingStrategyToObject converts the API
// representation of a group's scheduling strategy to the
// enforce_scheduling_strategy attribute value. The run_parallel value of a
// disabled enforcement is preserved from prior; unknown prior values resolve
// to null.
func checkGroupV2EnforceSchedulingStrategyToObject(ctx context.Context, group *checkly.GroupV2, prior types.Object, diags *diag.Diagnostics) types.Object {
	runParallel, ok := checkGroupV2PriorMember(prior, "run_parallel").(types.Bool)
	if !ok || runParallel.IsUnknown() {
		runParallel = types.BoolNull()
	}

	enforced := group.RunParallel != nil
	if enforced {
		runParallel = types.BoolValue(*group.RunParallel)
	}

	obj, d := types.ObjectValue(checkGroupV2EnforceSchedulingStrategyAttrTypes, map[string]attr.Value{
		"enabled":      types.BoolValue(enforced),
		"run_parallel": runParallel,
	})
	diags.Append(d...)

	return obj
}
