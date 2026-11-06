package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

var (
	_ resource.Resource                 = (*StatusPageResource)(nil)
	_ resource.ResourceWithConfigure    = (*StatusPageResource)(nil)
	_ resource.ResourceWithUpgradeState = (*StatusPageResource)(nil)
	_ resource.ResourceWithImportState  = (*StatusPageResource)(nil)
)

// NewStatusPageResource returns a new checkly_status_page resource.
func NewStatusPageResource() resource.Resource {
	return &StatusPageResource{}
}

// StatusPageResource implements the checkly_status_page resource.
type StatusPageResource struct {
	client checkly.Client
}

// StatusPageResourceModel describes the resource data model.
type StatusPageResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	URL          types.String `tfsdk:"url"`
	CustomDomain types.String `tfsdk:"custom_domain"`
	Logo         types.String `tfsdk:"logo"`
	RedirectTo   types.String `tfsdk:"redirect_to"`
	Favicon      types.String `tfsdk:"favicon"`
	DefaultTheme types.String `tfsdk:"default_theme"`
	Cards        types.List   `tfsdk:"card"`
}

// statusPageCardModel describes the card block data model.
type statusPageCardModel struct {
	Name               types.String `tfsdk:"name"`
	ServiceAttachments types.List   `tfsdk:"service_attachment"`
}

// statusPageServiceAttachmentModel describes the service_attachment block
// data model.
type statusPageServiceAttachmentModel struct {
	ServiceID types.String `tfsdk:"service_id"`
}

var statusPageServiceAttachmentAttrTypes = map[string]attr.Type{
	"service_id": types.StringType,
}

var statusPageCardAttrTypes = map[string]attr.Type{
	"name": types.StringType,
	"service_attachment": types.ListType{
		ElemType: types.ObjectType{AttrTypes: statusPageServiceAttachmentAttrTypes},
	},
}

func (r *StatusPageResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_status_page"
}

func (r *StatusPageResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: sdkv2SchemaVersion + 1,
		Description: "Checkly status pages allow you to easily communicate " +
			"the uptime and health of your applications and services to your customers. " +
			"This resource creates a legacy v2 status page." +
			"\n\n" +
			"The `checkly_status_page` resource type is deprecated in favor of `checkly_status_page_v3` and will " +
			"be removed in a future version of the provider. v3 status pages are separate objects, so an " +
			"existing page cannot be converted: create a v3 page alongside it instead. Maintenance windows can " +
			"be shown on v3 pages too, but `status_page_visibility` on `checkly_maintenance_window` can only link " +
			"them to v2 status pages for now.",
		DeprecationMessage: "Use the `checkly_status_page_v3` resource instead. v3 status pages are separate " +
			"objects, so an existing page cannot be converted; see the v2 upgrade guide. Keep the page if a maintenance " +
			"window managed by Terraform appears on it: `status_page_visibility` can't link a window to a v3 page yet.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the status page.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the status page.",
			},
			"url": schema.StringAttribute{
				Required:    true,
				Description: "The URL of the status page.",
			},
			"custom_domain": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "A custom user domain, e.g. \"status.example.com\". See the docs on updating your DNS and SSL usage.",
				Validators: []validator.String{
					statusPageCustomDomainValidator{},
				},
			},
			"logo": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "A URL to an image file to use as the logo for the status page.",
			},
			"redirect_to": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "The URL the user should be redirected to when clicking the logo.",
			},
			"favicon": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "A URL to an image file to use as the favicon of the status page.",
			},
			"default_theme": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("AUTO"),
				Description: "The color theme the status page is shown in. `AUTO` follows the visitor's system preference. Possible values are `AUTO`, `DARK`, and `LIGHT`. (Default `AUTO`).",
				Validators: []validator.String{
					stringvalidator.OneOf("AUTO", "DARK", "LIGHT"),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"card": schema.ListNestedBlock{
				Description: "A list of cards to include on the status page.",
				Validators: []validator.List{
					listvalidator.IsRequired(),
					listvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "The name of the card.",
						},
					},
					Blocks: map[string]schema.Block{
						"service_attachment": schema.ListNestedBlock{
							Description: "A list of services to attach to the card.",
							Validators: []validator.List{
								listvalidator.IsRequired(),
								listvalidator.SizeAtLeast(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"service_id": schema.StringAttribute{
										Required:    true,
										Description: "The ID of the service.",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// UpgradeState upgrades state written by the provider's SDK v2
// implementation; see sdkv2StateUpgraders.
func (r *StatusPageResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return sdkv2StateUpgraders(ctx, r)
}

func (r *StatusPageResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *StatusPageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan StatusPageResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	statusPage := statusPageFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.CreateStatusPage(apiCtx, statusPage)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Status Page",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	statusPageToModel(ctx, result, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *StatusPageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state StatusPageResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	statusPage, err := r.client.GetStatusPage(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Read Status Page",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	statusPageToModel(ctx, statusPage, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *StatusPageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan StatusPageResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	statusPage := statusPageFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.UpdateStatusPage(apiCtx, statusPage.ID, statusPage)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Status Page",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	statusPageToModel(ctx, result, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *StatusPageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state StatusPageResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	err := r.client.DeleteStatusPage(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Delete Status Page",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}
}

func (r *StatusPageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// statusPageFromModel converts the resource model to its API representation.
func statusPageFromModel(ctx context.Context, model *StatusPageResourceModel, diags *diag.Diagnostics) checkly.StatusPage {
	return checkly.StatusPage{
		ID:           model.ID.ValueString(),
		Name:         model.Name.ValueString(),
		URL:          model.URL.ValueString(),
		CustomDomain: model.CustomDomain.ValueString(),
		Logo:         model.Logo.ValueString(),
		RedirectTo:   model.RedirectTo.ValueString(),
		Favicon:      model.Favicon.ValueString(),
		DefaultTheme: checkly.StatusPageTheme(model.DefaultTheme.ValueString()),
		Cards:        statusPageCardsFromList(ctx, model.Cards, diags),
	}
}

// statusPageCardsFromList converts the card block value to its API
// representation.
func statusPageCardsFromList(ctx context.Context, list types.List, diags *diag.Diagnostics) []checkly.StatusPageCard {
	res := []checkly.StatusPageCard{}

	if list.IsNull() || list.IsUnknown() {
		return res
	}

	var models []statusPageCardModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return res
	}

	for _, model := range models {
		var attachmentModels []statusPageServiceAttachmentModel
		diags.Append(model.ServiceAttachments.ElementsAs(ctx, &attachmentModels, false)...)
		if diags.HasError() {
			return res
		}

		services := []checkly.StatusPageService{}
		for _, attachment := range attachmentModels {
			services = append(services, checkly.StatusPageService{
				ID: attachment.ServiceID.ValueString(),
			})
		}

		res = append(res, checkly.StatusPageCard{
			Name:     model.Name.ValueString(),
			Services: services,
		})
	}

	return res
}

// statusPageCardsToList converts the API representation of cards to the card
// block value.
func statusPageCardsToList(ctx context.Context, cards []checkly.StatusPageCard, diags *diag.Diagnostics) types.List {
	cardType := types.ObjectType{AttrTypes: statusPageCardAttrTypes}

	models := make([]statusPageCardModel, 0, len(cards))
	for _, card := range cards {
		attachmentModels := make([]statusPageServiceAttachmentModel, 0, len(card.Services))
		for _, service := range card.Services {
			attachmentModels = append(attachmentModels, statusPageServiceAttachmentModel{
				ServiceID: types.StringValue(service.ID),
			})
		}

		attachments, d := types.ListValueFrom(
			ctx,
			types.ObjectType{AttrTypes: statusPageServiceAttachmentAttrTypes},
			attachmentModels,
		)
		diags.Append(d...)
		if diags.HasError() {
			return types.ListNull(cardType)
		}

		models = append(models, statusPageCardModel{
			Name:               types.StringValue(card.Name),
			ServiceAttachments: attachments,
		})
	}

	list, d := types.ListValueFrom(ctx, cardType, models)
	diags.Append(d...)

	return list
}

// statusPageToModel updates the model to match the API representation of the
// status page.
func statusPageToModel(ctx context.Context, statusPage *checkly.StatusPage, model *StatusPageResourceModel, diags *diag.Diagnostics) {
	model.ID = types.StringValue(statusPage.ID)
	model.Name = types.StringValue(statusPage.Name)
	model.URL = types.StringValue(statusPage.URL)
	model.CustomDomain = types.StringValue(statusPage.CustomDomain)
	model.Logo = types.StringValue(statusPage.Logo)
	model.RedirectTo = types.StringValue(statusPage.RedirectTo)
	model.Favicon = types.StringValue(statusPage.Favicon)
	model.DefaultTheme = types.StringValue(string(statusPage.DefaultTheme))
	model.Cards = statusPageCardsToList(ctx, statusPage.Cards, diags)
}

var _ validator.String = statusPageCustomDomainValidator{}

// statusPageCustomDomainValidator rejects custom domains under reserved
// documentation domains (example.com, example.net, example.org), which the
// Checkly API does not support.
type statusPageCustomDomainValidator struct{}

func (v statusPageCustomDomainValidator) Description(ctx context.Context) string {
	return "custom domains ending in example.com, example.net or example.org are not supported"
}

func (v statusPageCustomDomainValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v statusPageCustomDomainValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	unsupportedSuffixes := []string{
		"example.com",
		"example.net",
		"example.org",
	}

	value := strings.ToLower(req.ConfigValue.ValueString())
	for _, suffix := range unsupportedSuffixes {
		if strings.HasSuffix(value, suffix) {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Unsupported Custom Domain",
				fmt.Sprintf("custom domains ending in %s are not supported", suffix),
			)
			return
		}
	}
}
