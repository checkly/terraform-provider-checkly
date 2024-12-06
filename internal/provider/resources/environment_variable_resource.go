package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/checkly/terraform-provider-checkly/internal/provider/interop"
	"github.com/checkly/terraform-provider-checkly/internal/provider/resources/attributes"
	"github.com/checkly/terraform-provider-checkly/internal/sdkutil"
)

var (
	_ resource.Resource                = (*EnvironmentVariableResource)(nil)
	_ resource.ResourceWithConfigure   = (*EnvironmentVariableResource)(nil)
	_ resource.ResourceWithImportState = (*EnvironmentVariableResource)(nil)
)

type EnvironmentVariableResource struct {
	client checkly.Client
}

func NewEnvironmentVariableResource() resource.Resource {
	return &EnvironmentVariableResource{}
}

func (r *EnvironmentVariableResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_environment_variable"
}

func (r *EnvironmentVariableResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":           attributes.IDAttributeSchema,
			"last_updated": attributes.LastUpdatedAttributeSchema,
			"key": schema.StringAttribute{
				Required:    true,
				Description: "", // TODO
			},
			"value": schema.StringAttribute{
				Required:    true,
				Sensitive:   true, // FIXME: Keep sensitive? Old code did not set it.
				Description: "",   // TODO
			},
			"locked": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "", // TODO
			},
			"secret": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "", // TODO
			},
		},
	}
}

func (r *EnvironmentVariableResource) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	client, diags := interop.ClientFromProviderData(req.ProviderData)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	r.client = client
}

func (r *EnvironmentVariableResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *EnvironmentVariableResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan EnvironmentVariableResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var desiredModel checkly.EnvironmentVariable
	resp.Diagnostics.Append(plan.Render(ctx, &desiredModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realizedModel, err := r.client.CreateEnvironmentVariable(
		ctx,
		desiredModel,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Checkly Environment Variable",
			fmt.Sprintf("Could not create environment variable, unexpected error: %s", err),
		)

		return
	}

	resp.Diagnostics.Append(plan.Refresh(ctx, realizedModel, interop.Created)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *EnvironmentVariableResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state EnvironmentVariableResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteEnvironmentVariable(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Checkly Environment Variable",
			fmt.Sprintf("Could not delete environment variable, unexpected error: %s", err),
		)

		return
	}
}

func (r *EnvironmentVariableResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state EnvironmentVariableResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realizedModel, err := r.client.GetEnvironmentVariable(
		ctx,
		state.ID.ValueString(),
	)
	if err != nil {
		if sdkutil.IsHTTPNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading Checkly Environment Variable",
			fmt.Sprintf("Could not retrieve environment variable, unexpected error: %s", err),
		)

		return
	}

	resp.Diagnostics.Append(state.Refresh(ctx, realizedModel, interop.Loaded)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *EnvironmentVariableResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan EnvironmentVariableResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var desiredModel checkly.EnvironmentVariable
	resp.Diagnostics.Append(plan.Render(ctx, &desiredModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realizedModel, err := r.client.UpdateEnvironmentVariable(
		ctx,
		plan.ID.ValueString(),
		desiredModel,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Checkly Environment Variable",
			fmt.Sprintf("Could not update environment variable, unexpected error: %s", err),
		)

		return
	}

	resp.Diagnostics.Append(plan.Refresh(ctx, realizedModel, interop.Updated)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

var (
	_ interop.Model[checkly.EnvironmentVariable] = (*EnvironmentVariableResourceModel)(nil)
)

type EnvironmentVariableResourceModel struct {
	ID          types.String `tfsdk:"id"`
	LastUpdated types.String `tfsdk:"last_updated"` // FIXME: Keep this? Old code did not have it.
	Key         types.String `tfsdk:"key"`
	Value       types.String `tfsdk:"value"`
	Locked      types.Bool   `tfsdk:"locked"`
	Secret      types.Bool   `tfsdk:"secret"`
}

func (m *EnvironmentVariableResourceModel) Refresh(ctx context.Context, from *checkly.EnvironmentVariable, flags interop.RefreshFlags) diag.Diagnostics {
	if flags.Created() {
		m.ID = types.StringValue(from.Key)
	}

	if flags.Created() || flags.Updated() {
		m.LastUpdated = attributes.LastUpdatedNow()
	}

	m.Key = types.StringValue(from.Key)
	m.Locked = types.BoolValue(from.Locked)
	m.Secret = types.BoolValue(from.Secret)

	// We can never receive a secret value back from the server. Just assume
	// the value is still unchanged and only update state if we're not dealing
	// with a secret.
	if !from.Secret {
		m.Value = types.StringValue(from.Value)
	}

	return nil
}

func (m *EnvironmentVariableResourceModel) Render(ctx context.Context, into *checkly.EnvironmentVariable) diag.Diagnostics {
	into.Key = m.Key.ValueString()
	into.Value = m.Value.ValueString()
	into.Locked = m.Locked.ValueBool()
	into.Secret = m.Secret.ValueBool()

	return nil
}
