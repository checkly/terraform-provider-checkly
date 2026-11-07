package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

var (
	_ resource.Resource                 = (*EnvironmentVariableResource)(nil)
	_ resource.ResourceWithConfigure    = (*EnvironmentVariableResource)(nil)
	_ resource.ResourceWithUpgradeState = (*EnvironmentVariableResource)(nil)
	_ resource.ResourceWithImportState  = (*EnvironmentVariableResource)(nil)
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "The value of the environment variable.",
			},
			"locked": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "If true, the value is hidden in the Checkly UI by default, but it can be revealed. (Default `false`).",
			},
			"secret": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "If true, the value can never be revealed after creation. (Default `false`).",
			},
		},
	}
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
		resp.Diagnostics.AddError(
			"Unable to Create Environment Variable",
			fmt.Sprintf("Checkly API error: %s", err),
		)
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
		resp.Diagnostics.AddError(
			"Unable to Read Environment Variable",
			fmt.Sprintf("Checkly API error: %s", err),
		)
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
		resp.Diagnostics.AddError(
			"Unable to Update Environment Variable",
			fmt.Sprintf("Checkly API error: %s", err),
		)
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
		resp.Diagnostics.AddError(
			"Unable to Delete Environment Variable",
			fmt.Sprintf("Checkly API error: %s", err),
		)
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
