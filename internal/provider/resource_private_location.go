package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

var (
	_ resource.Resource                 = (*PrivateLocationResource)(nil)
	_ resource.ResourceWithConfigure    = (*PrivateLocationResource)(nil)
	_ resource.ResourceWithUpgradeState = (*PrivateLocationResource)(nil)
	_ resource.ResourceWithImportState  = (*PrivateLocationResource)(nil)
)

// NewPrivateLocationResource returns a new checkly_private_location resource.
func NewPrivateLocationResource() resource.Resource {
	return &PrivateLocationResource{}
}

// PrivateLocationResource implements the checkly_private_location resource.
type PrivateLocationResource struct {
	client checkly.Client
}

// PrivateLocationResourceModel describes the resource data model.
type PrivateLocationResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	SlugName types.String `tfsdk:"slug_name"`
	Icon     types.String `tfsdk:"icon"`
	Keys     types.Set    `tfsdk:"keys"`
}

func (r *PrivateLocationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_location"
}

func (r *PrivateLocationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: sdkv2SchemaVersion + 1,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the private location.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The private location name.",
			},
			"slug_name": schema.StringAttribute{
				Required:    true,
				Description: "Valid slug name.",
			},
			"icon": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("location"),
				Description: "Icon assigned to the private location.",
			},
			"keys": schema.SetAttribute{
				Computed:    true,
				Sensitive:   true,
				ElementType: types.StringType,
				Description: "Private location API keys.",
				// The API returns raw key material only in the creation
				// response; subsequent reads never include it. Keep the
				// keys captured at creation time in state.
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// UpgradeState upgrades state written by the provider's SDK v2
// implementation; see sdkv2StateUpgraders.
func (r *PrivateLocationResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return sdkv2StateUpgraders(ctx, r)
}

func (r *PrivateLocationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *PrivateLocationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PrivateLocationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.CreatePrivateLocation(apiCtx, privateLocationFromModel(&plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Private Location",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	plan.ID = types.StringValue(result.ID)

	// Raw key material is only available in the creation response; later
	// reads only return masked keys.
	var rawKeys []string
	for _, key := range result.Keys {
		if key.RawKey != "" {
			rawKeys = append(rawKeys, key.RawKey)
		}
	}
	plan.Keys = setFromStrings(ctx, rawKeys, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PrivateLocationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PrivateLocationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	pl, err := r.client.GetPrivateLocation(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Read Private Location",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	state.Name = types.StringValue(pl.Name)
	state.SlugName = types.StringValue(pl.SlugName)
	state.Icon = types.StringValue(pl.Icon)
	// The API never returns raw key material after creation, so the keys
	// already present in state are kept as-is.

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PrivateLocationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PrivateLocationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	stored, err := readForUpdate(ctx, r.client.GetPrivateLocation, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Private Location",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	// Terraform has no attribute for the proxy, and the update would remove it.
	location := privateLocationFromModel(&plan)
	location.ProxyURL = stored.ProxyURL

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	_, err = r.client.UpdatePrivateLocation(apiCtx, plan.ID.ValueString(), location)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Private Location",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PrivateLocationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PrivateLocationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	err := r.client.DeletePrivateLocation(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Delete Private Location",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}
}

func (r *PrivateLocationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func privateLocationFromModel(model *PrivateLocationResourceModel) checkly.PrivateLocation {
	return checkly.PrivateLocation{
		Name:     model.Name.ValueString(),
		SlugName: model.SlugName.ValueString(),
		Icon:     model.Icon.ValueString(),
	}
}
