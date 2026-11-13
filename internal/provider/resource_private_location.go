package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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
	ProxyURL types.String `tfsdk:"proxy_url"`
}

func (r *PrivateLocationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_location"
}

func (r *PrivateLocationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: sdkv2SchemaVersion + 1,
		Description: "A private location runs checks from your own infrastructure: Checkly agents that you " +
			"deploy, for example as containers in your network, run the checks and monitors assigned to it. " +
			"The agents authenticate with the location's API keys, which are only available when the " +
			"location is created; see `keys`.",
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
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"slug_name": schema.StringAttribute{
				Required: true,
				Description: "The slug name of the private location, which checks and monitors refer to the location by. " +
					"Must be between `1` and `30` characters. Must contain only letters, digits and dashes, and must not start with the ID of a public location such as `eu-central-1`. The slug can't be changed after creation: a plan that changes it fails, also with `terraform apply -replace` or `replace_triggered_by`. " +
					"To use a different slug, add a new `checkly_private_location` resource block with it, move your checks and agents to the new location, then remove the old block.",
				Validators: privateLocationSlugValidators,
				PlanModifiers: []planmodifier.String{
					privateLocationSlugUnchanged{},
				},
			},
			"icon": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("location"),
				Description: "Icon assigned to the private location. Defaults to `location`.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"keys": schema.SetAttribute{
				Computed:    true,
				Sensitive:   true,
				ElementType: types.StringType,
				Description: "The API keys that the location's agents authenticate with. The API returns a key " +
					"only when it is created, so this holds the key created with the location and is never " +
					"refreshed: keys added or revoked later, for example in the web app, don't show here. " +
					"After an import, it is null.",
				// The API returns raw key material only in the creation
				// response; subsequent reads never include it. Keep the
				// keys captured at creation time in state.
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"proxy_url": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The proxy that API checks running in the private location send their HTTP requests through, such as `https://user:password@proxy.example.com:3128`. It can hold credentials.",
			},
		},
	}
}

// privateLocationSlugUnchanged fails the plan when it changes the slug of an
// existing private location. The API ignores slugName on update, so an
// in-place change would never apply. Replacing the location instead isn't
// safe: the API refuses to delete a location that checks or groups use, and
// a new location has new API keys, which disconnects the running agents.
type privateLocationSlugUnchanged struct{}

var _ planmodifier.String = privateLocationSlugUnchanged{}

func (m privateLocationSlugUnchanged) Description(ctx context.Context) string {
	return "The slug can't be changed after the private location is created."
}

func (m privateLocationSlugUnchanged) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m privateLocationSlugUnchanged) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.PlanValue.IsUnknown() || req.PlanValue.IsNull() {
		return
	}
	if req.PlanValue.Equal(req.StateValue) {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Private Location Slug Can't Change",
		fmt.Sprintf("The slug of an existing private location can't be changed: the API keeps %s. "+
			"Replacing the location isn't supported either, also not with terraform apply -replace or replace_triggered_by: "+
			"the API refuses to delete a location that checks or groups use, and the agents of a new location need its new API keys. "+
			"To use %s, add a new checkly_private_location resource block with it, move your checks and agents to the new location, "+
			"then remove the old block. For a location that no check or group uses yet, you can instead remove its resource block, "+
			"apply, and add it back with the new slug.", req.StateValue, req.PlanValue),
	)
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
		addAPIError(&resp.Diagnostics, "Unable to Create Private Location", err)
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
		addAPIError(&resp.Diagnostics, "Unable to Read Private Location", err)
		return
	}

	privateLocationToModel(pl, &state)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PrivateLocationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PrivateLocationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	_, err := r.client.UpdatePrivateLocation(apiCtx, plan.ID.ValueString(), privateLocationFromModel(&plan))
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to Update Private Location", err)
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
		addAPIError(&resp.Diagnostics, "Unable to Delete Private Location", err)
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
		ProxyURL: optionalStringPointer(model.ProxyURL),
	}
}

// privateLocationToModel updates the model to match the API representation of
// the private location. The API never returns raw key material after
// creation, so the keys already present in the model are kept as-is.
func privateLocationToModel(pl *checkly.PrivateLocation, model *PrivateLocationResourceModel) {
	model.Name = types.StringValue(pl.Name)
	model.SlugName = types.StringValue(pl.SlugName)
	model.Icon = types.StringValue(pl.Icon)

	// The API stores an empty proxy as null, so a configured "" is kept
	// rather than refreshed to null.
	var proxyURL string
	if pl.ProxyURL != nil {
		proxyURL = *pl.ProxyURL
	}
	model.ProxyURL = stringValueOrPriorEmpty(proxyURL, model.ProxyURL)
}
