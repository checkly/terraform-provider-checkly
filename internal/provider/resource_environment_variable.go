package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

var (
	_ resource.Resource                 = (*EnvironmentVariableResource)(nil)
	_ resource.ResourceWithConfigure    = (*EnvironmentVariableResource)(nil)
	_ resource.ResourceWithUpgradeState = (*EnvironmentVariableResource)(nil)
	_ resource.ResourceWithImportState  = (*EnvironmentVariableResource)(nil)
	_ resource.ResourceWithModifyPlan   = (*EnvironmentVariableResource)(nil)
)

// NewEnvironmentVariableResource returns a new checkly_environment_variable
// resource.
func NewEnvironmentVariableResource() resource.Resource {
	return &EnvironmentVariableResource{}
}

// EnvironmentVariableResource implements the checkly_environment_variable
// resource.
type EnvironmentVariableResource struct {
	client checkly.Client
}

// EnvironmentVariableResourceModel describes the resource data model.
type EnvironmentVariableResourceModel struct {
	ID     types.String `tfsdk:"id"`
	Key    types.String `tfsdk:"key"`
	Value  types.String `tfsdk:"value"`
	Locked types.Bool   `tfsdk:"locked"`
	Secret types.Bool   `tfsdk:"secret"`
}

func (r *EnvironmentVariableResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_variable"
}

func (r *EnvironmentVariableResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     sdkv2SchemaVersion + 1,
		Description: "Global environment variables that can be referenced from check scripts.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the environment variable. Identical to the key.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"key": schema.StringAttribute{
				Required:    true,
				Description: "The name of the environment variable. Changing the name forces the environment variable to be replaced.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				Required:  true,
				Sensitive: true,
				Description: "The value of the environment variable, which must not be empty. " + environmentVariableValueDescription + " " +
					"The API never returns the value of a secret variable, so the provider keeps the configured value: " +
					"a change made outside Terraform, for example in the web app, doesn't show in the plan. " +
					"An imported secret variable has a null value, so the first plan after the import shows an update, " +
					"and applying it overwrites the value in Checkly with the configured one.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					environmentVariableValueValidator{},
				},
			},
			"locked": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "If true, the value is hidden in the Checkly UI by default, but it can be revealed. (Default `false`).",
			},
			"secret": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				Description: "If true, the value can never be revealed after creation, and the API never returns it; see `value`. " +
					"The API can't turn a secret variable back into a regular one, so changing `secret` from `true` to `false` " +
					"forces the environment variable to be replaced. (Default `false`).",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplaceIf(
						func(ctx context.Context, req planmodifier.BoolRequest, resp *boolplanmodifier.RequiresReplaceIfFuncResponse) {
							// An unknown plan value may turn out false, which
							// can only be applied by a replacement.
							resp.RequiresReplace = req.StateValue.ValueBool() && (req.PlanValue.IsUnknown() || !req.PlanValue.ValueBool())
						},
						"Changing secret from true to false forces the environment variable to be replaced.",
						"Changing `secret` from `true` to `false` forces the environment variable to be replaced.",
					),
				},
			},
		},
	}
}

// ModifyPlan checks the length of a secret value. Every create and update of
// an environment variable sends the value, and the API then checks it, so a
// short value is an error when the plan changes anything. A plan without
// changes only warns about it, since the API keeps the stored value.
func (r *EnvironmentVariableResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan EnvironmentVariableResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plan.Value.IsUnknown() || plan.Secret.IsUnknown() || !plan.Secret.ValueBool() {
		return
	}

	sent := req.State.Raw.IsNull() || !req.Plan.Raw.Equal(req.State.Raw)
	checkSecretEnvironmentVariableValue(&resp.Diagnostics, path.Root("value"), plan.Value.ValueString(), sent)
}

// UpgradeState upgrades state written by the provider's SDK v2
// implementation; see sdkv2StateUpgraders.
func (r *EnvironmentVariableResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return sdkv2StateUpgraders(ctx, r)
}

func (r *EnvironmentVariableResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *EnvironmentVariableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EnvironmentVariableResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	_, err := r.client.CreateEnvironmentVariable(apiCtx, environmentVariableFromModel(&plan))
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to Create Environment Variable", err)
		return
	}

	plan.ID = plan.Key

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EnvironmentVariableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state EnvironmentVariableResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	envVar, err := r.client.GetEnvironmentVariable(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		addAPIError(&resp.Diagnostics, "Unable to Read Environment Variable", err)
		return
	}

	environmentVariableToModel(envVar, &state)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *EnvironmentVariableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan EnvironmentVariableResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	_, err := r.client.UpdateEnvironmentVariable(apiCtx, plan.ID.ValueString(), environmentVariableFromModel(&plan))
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to Update Environment Variable", err)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EnvironmentVariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EnvironmentVariableResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	err := r.client.DeleteEnvironmentVariable(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			return
		}
		addAPIError(&resp.Diagnostics, "Unable to Delete Environment Variable", err)
		return
	}
}

func (r *EnvironmentVariableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	// The ID of an environment variable is its key.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("key"), req.ID)...)
}

func environmentVariableFromModel(model *EnvironmentVariableResourceModel) checkly.EnvironmentVariable {
	return checkly.EnvironmentVariable{
		Key:    model.Key.ValueString(),
		Value:  model.Value.ValueString(),
		Locked: model.Locked.ValueBool(),
		Secret: model.Secret.ValueBool(),
	}
}

// environmentVariableToModel updates the model to match the API
// representation of the environment variable. The value of a secret
// environment variable is never returned by the API, so the value already
// present in the model is preserved for secrets.
func environmentVariableToModel(envVar *checkly.EnvironmentVariable, model *EnvironmentVariableResourceModel) {
	model.ID = types.StringValue(envVar.Key)
	model.Key = types.StringValue(envVar.Key)
	if !envVar.Secret {
		model.Value = types.StringValue(envVar.Value)
	}
	model.Locked = types.BoolValue(envVar.Locked)
	model.Secret = types.BoolValue(envVar.Secret)
}
