package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
)

var (
	_ resource.Resource                 = (*StatusPageV3Resource)(nil)
	_ resource.ResourceWithConfigure    = (*StatusPageV3Resource)(nil)
	_ resource.ResourceWithUpgradeState = (*StatusPageV3Resource)(nil)
	_ resource.ResourceWithImportState  = (*StatusPageV3Resource)(nil)
)

var statusPageV3ThemeValues = allowedValues[string]{
	{Value: "AUTO"},
	{Value: "DARK"},
	{Value: "LIGHT"},
}

// The server lowercases the URL before storing it, so accepting uppercase
// here would create a permanent diff between config and state.
var statusPageV3URLRegex = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

// The server lowercases the custom domain and strips a scheme before storing
// it, so only the stored form is accepted to avoid a permanent diff. An empty
// value is accepted too: it clears the domain like leaving it out does.
var statusPageV3CustomDomainRegex = regexp.MustCompile(`^$|^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+$`)

// Same format the API accepts: #RGB or #RRGGBB, case-insensitive.
var statusPageV3HexColorRegex = regexp.MustCompile(`^#([0-9A-Fa-f]{3}|[0-9A-Fa-f]{6})$`)

// The twelve colors of one theme, in the order the API documents them.
// `field` points into a group so the same table drives the schema, the
// expand and the flatten.
var statusPageV3ThemeColorAttributes = []struct {
	key         string
	description string
	field       func(group *checkly.StatusPageV3ThemeColorGroup) *string
}{
	{"body_background_color", "The background of the page.",
		func(g *checkly.StatusPageV3ThemeColorGroup) *string { return &g.BodyBackgroundColor }},
	{"header_background_color", "The background of the page header.",
		func(g *checkly.StatusPageV3ThemeColorGroup) *string { return &g.HeaderBackgroundColor }},
	{"header_font_color", "The color of text in the page header.",
		func(g *checkly.StatusPageV3ThemeColorGroup) *string { return &g.HeaderFontColor }},
	{"title_font_color", "The color of titles and headings.",
		func(g *checkly.StatusPageV3ThemeColorGroup) *string { return &g.TitleFontColor }},
	{"body_font_color", "The color of regular body text.",
		func(g *checkly.StatusPageV3ThemeColorGroup) *string { return &g.BodyFontColor }},
	{"body_font_color_muted", "The color of de-emphasized body text, such as timestamps.",
		func(g *checkly.StatusPageV3ThemeColorGroup) *string { return &g.BodyFontColorMuted }},
	{"navigation_font_color", "The color of navigation links.",
		func(g *checkly.StatusPageV3ThemeColorGroup) *string { return &g.NavigationFontColor }},
	{"link_font_color", "The color of links in the page content.",
		func(g *checkly.StatusPageV3ThemeColorGroup) *string { return &g.LinkFontColor }},
	{"card_background_color", "The background of component and incident cards.",
		func(g *checkly.StatusPageV3ThemeColorGroup) *string { return &g.CardBackgroundColor }},
	{"border_color", "The color of borders and dividers.",
		func(g *checkly.StatusPageV3ThemeColorGroup) *string { return &g.BorderColor }},
	{"primary_button_background_color", "The background of primary buttons, such as \"Subscribe\".",
		func(g *checkly.StatusPageV3ThemeColorGroup) *string { return &g.PrimaryButtonBackgroundColor }},
	{"primary_button_font_color", "The color of text on primary buttons.",
		func(g *checkly.StatusPageV3ThemeColorGroup) *string { return &g.PrimaryButtonFontColor }},
}

var statusPageV3ThemeColorGroupAttrTypes = func() map[string]attr.Type {
	attrTypes := make(map[string]attr.Type, len(statusPageV3ThemeColorAttributes))
	for _, a := range statusPageV3ThemeColorAttributes {
		attrTypes[a.key] = types.StringType
	}
	return attrTypes
}()

var statusPageV3ThemeColorGroupObjectType = types.ObjectType{AttrTypes: statusPageV3ThemeColorGroupAttrTypes}

var statusPageV3ThemeColorsAttrTypes = map[string]attr.Type{
	"light": types.ListType{ElemType: statusPageV3ThemeColorGroupObjectType},
	"dark":  types.ListType{ElemType: statusPageV3ThemeColorGroupObjectType},
}

var statusPageV3ThemeColorsObjectType = types.ObjectType{AttrTypes: statusPageV3ThemeColorsAttrTypes}

// statusPageV3StringValidator rejects string values for which check returns
// false. message is a format string receiving the attribute path (%q) and
// the rejected value (%s).
type statusPageV3StringValidator struct {
	check       func(string) bool
	message     string
	description string
}

func (v statusPageV3StringValidator) Description(ctx context.Context) string {
	return v.description
}

func (v statusPageV3StringValidator) MarkdownDescription(ctx context.Context) string {
	return v.description
}

func (v statusPageV3StringValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	if !v.check(value) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Attribute Value",
			fmt.Sprintf(v.message, req.Path.String(), value),
		)
	}
}

func validateStatusPageV3HexColor(value string) bool {
	return statusPageV3HexColorRegex.MatchString(value)
}

var statusPageV3HexColorValidator = statusPageV3StringValidator{
	check:       validateStatusPageV3HexColor,
	message:     "%q must be a hex color such as \"#FF0000\" or \"#F00\", got: %s",
	description: "value must be a hex color such as \"#FF0000\" or \"#F00\"",
}

// The API requires the complete palette when custom colors are given, so
// every color is required rather than defaulted here: the defaults belong
// to the backend and may change.
func statusPageV3ThemeColorGroupSchema(theme string) schema.ListNestedBlock {
	colors := make(map[string]schema.Attribute, len(statusPageV3ThemeColorAttributes))
	for _, a := range statusPageV3ThemeColorAttributes {
		colors[a.key] = schema.StringAttribute{
			Required:    true,
			Description: a.description + " A hex color such as \"#FF0000\" or \"#F00\".",
			Validators: []validator.String{
				statusPageV3HexColorValidator,
			},
		}
	}

	return schema.ListNestedBlock{
		Description: "The colors used when the page renders in " + theme + " mode.",
		Validators: []validator.List{
			listvalidator.IsRequired(),
			listvalidator.SizeAtMost(1),
		},
		NestedObject: schema.NestedBlockObject{
			Attributes: colors,
		},
	}
}

// NewStatusPageV3Resource returns a new checkly_status_page_v3 resource.
func NewStatusPageV3Resource() resource.Resource {
	return &StatusPageV3Resource{}
}

// StatusPageV3Resource implements the checkly_status_page_v3 resource.
type StatusPageV3Resource struct {
	client checkly.Client
}

// StatusPageV3ResourceModel describes the resource data model.
type StatusPageV3ResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	URL                types.String `tfsdk:"url"`
	CustomDomain       types.String `tfsdk:"custom_domain"`
	Description        types.String `tfsdk:"description"`
	Logo               types.String `tfsdk:"logo"`
	LogoDark           types.String `tfsdk:"logo_dark"`
	RedirectTo         types.String `tfsdk:"redirect_to"`
	Favicon            types.String `tfsdk:"favicon"`
	DefaultTheme       types.String `tfsdk:"default_theme"`
	PrivacyPolicyLink  types.String `tfsdk:"privacy_policy_link"`
	TermsOfServiceLink types.String `tfsdk:"terms_of_service_link"`
	SupportLink        types.String `tfsdk:"support_link"`
	FooterText         types.String `tfsdk:"footer_text"`
	GoogleAnalyticsTag types.String `tfsdk:"google_analytics_tag"`
	AllowIndexing      types.Bool   `tfsdk:"allow_indexing"`
	ThemeColors        types.List   `tfsdk:"theme_colors"`
}

func (r *StatusPageV3Resource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_status_page_v3"
}

// statusPageV3OptionalString is an optional string attribute that is
// stored as an empty string when left out, so that removing it from the
// configuration clears the value remotely.
func statusPageV3OptionalString(description string, validators ...validator.String) schema.StringAttribute {
	return schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Default:     stringdefault.StaticString(""),
		Description: description,
		Validators:  validators,
	}
}

func (r *StatusPageV3Resource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: sdkv2SchemaVersion + 1,
		Description: "Checkly status pages allow you to easily communicate " +
			"the uptime and health of your applications and services to your " +
			"customers. A v3 status page has no cards or services: its " +
			"structure is managed with `checkly_status_page_v3_component` " +
			"resources, and incidents can be automated with " +
			"`checkly_status_page_v3_automation_rule` resources. The " +
			"`checkly_status_page_v3` resource should always be preferred " +
			"over the old `checkly_status_page` resource.",
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
				Description: "The unique subdomain of the status page: lowercase alphanumeric characters and dashes, no leading or trailing dash, at most 63 characters.",
				Validators: []validator.String{
					statusPageV3StringValidator{
						check:       statusPageV3URLRegex.MatchString,
						message:     "%q can only contain lowercase alphanumeric characters and dashes, with no leading or trailing dash, got: %s",
						description: "value must be lowercase alphanumeric characters and dashes, with no leading or trailing dash",
					},
				},
			},
			"custom_domain": statusPageV3OptionalString(
				"A custom user domain, e.g. \"status.example.com\": a lowercase hostname without a scheme. See the docs on updating your DNS and SSL usage.",
				statusPageV3StringValidator{
					check:       statusPageV3CustomDomainRegex.MatchString,
					message:     "%q must be a lowercase hostname without a scheme, such as \"status.example.com\", got: %s",
					description: "value must be a lowercase hostname without a scheme",
				},
			),
			"description": statusPageV3OptionalString("A short description shown on the status page."),
			"logo":        statusPageV3OptionalString("A URL to an image file to use as the logo for the status page."),
			"logo_dark":   statusPageV3OptionalString("A URL to an image file to use as the logo in dark mode."),
			"redirect_to": statusPageV3OptionalString("The URL the user should be redirected to when clicking the logo."),
			"favicon":     statusPageV3OptionalString("A URL to an image file to use as the favicon of the status page."),
			"default_theme": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("AUTO"),
				Description: "The default theme of the status page. " + statusPageV3ThemeValues.String(),
				Validators: []validator.String{
					stringvalidator.OneOf(statusPageV3ThemeValues.Values()...),
				},
			},
			"privacy_policy_link":   statusPageV3OptionalString("A link to your privacy policy, shown in the page footer."),
			"terms_of_service_link": statusPageV3OptionalString("A link to your terms of service, shown in the page footer."),
			"support_link":          statusPageV3OptionalString("A support contact link (http, https or mailto), shown in the page footer."),
			"footer_text":           statusPageV3OptionalString("Free-form footer text."),
			"google_analytics_tag":  statusPageV3OptionalString("A Google Analytics tag ID (e.g. \"G-XXXXXXXXXX\") to embed on the public page."),
			"allow_indexing": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether search engines may index the public page. (Default `true`).",
			},
		},
		Blocks: map[string]schema.Block{
			"theme_colors": schema.ListNestedBlock{
				Description: "Custom colors for the light and dark theme of the page. Requires custom " +
					"theme colors to be part of your plan. The complete palette must be given: both " +
					"`light` and `dark`, each with every color. Leave the block out to use Checkly's " +
					"default colors; the default palette the API then reports is not tracked, and the " +
					"block is not populated when importing a page.",
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Blocks: map[string]schema.Block{
						"light": statusPageV3ThemeColorGroupSchema("light"),
						"dark":  statusPageV3ThemeColorGroupSchema("dark"),
					},
				},
			},
		},
	}
}

// UpgradeState upgrades state written by the provider's SDK v2
// implementation; see sdkv2StateUpgraders.
func (r *StatusPageV3Resource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return sdkv2StateUpgraders(ctx, r)
}

func (r *StatusPageV3Resource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

// readInto fetches the status page and updates the model to match it.
// notFound is true when the page no longer exists remotely.
func (r *StatusPageV3Resource) readInto(ctx context.Context, model *StatusPageV3ResourceModel, diags *diag.Diagnostics) (notFound bool, err error) {
	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	statusPage, err := r.client.GetStatusPageV3(apiCtx, model.ID.ValueString())
	if err != nil {
		return isNotFoundError(err), err
	}

	statusPageV3ToModel(ctx, statusPage, model, diags)

	return false, nil
}

func (r *StatusPageV3Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan StatusPageV3ResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	statusPage := statusPageV3FromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.CreateStatusPageV3(apiCtx, statusPage)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Status Page V3",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	plan.ID = types.StringValue(result.ID)

	if _, err := r.readInto(ctx, &plan, &resp.Diagnostics); err != nil {
		// Keep the created page in state so that it is not orphaned.
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.AddError(
			"Unable to Read Status Page V3",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *StatusPageV3Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state StatusPageV3ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	notFound, err := r.readInto(ctx, &state, &resp.Diagnostics)
	if err != nil {
		if notFound {
			// Deleted remotely: mark the resource as gone.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Read Status Page V3",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *StatusPageV3Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan StatusPageV3ResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	statusPage := statusPageV3FromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	_, err := r.client.UpdateStatusPageV3(apiCtx, statusPage.ID, statusPage)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Status Page V3",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	if _, err := r.readInto(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Status Page V3",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *StatusPageV3Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state StatusPageV3ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	err := r.client.DeleteStatusPageV3(apiCtx, state.ID.ValueString())
	if err != nil {
		// Already gone (deleted out-of-band): nothing left to do.
		if isNotFoundError(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Delete Status Page V3",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}
}

func (r *StatusPageV3Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// statusPageV3FromModel converts the resource model to its API
// representation.
func statusPageV3FromModel(ctx context.Context, model *StatusPageV3ResourceModel, diags *diag.Diagnostics) checkly.StatusPageV3 {
	return checkly.StatusPageV3{
		ID:                 model.ID.ValueString(),
		Name:               model.Name.ValueString(),
		URL:                model.URL.ValueString(),
		CustomDomain:       model.CustomDomain.ValueString(),
		Description:        model.Description.ValueString(),
		Logo:               model.Logo.ValueString(),
		LogoDark:           model.LogoDark.ValueString(),
		RedirectTo:         model.RedirectTo.ValueString(),
		Favicon:            model.Favicon.ValueString(),
		DefaultTheme:       checkly.StatusPageTheme(model.DefaultTheme.ValueString()),
		PrivacyPolicyLink:  model.PrivacyPolicyLink.ValueString(),
		TermsOfServiceLink: model.TermsOfServiceLink.ValueString(),
		SupportLink:        model.SupportLink.ValueString(),
		FooterText:         model.FooterText.ValueString(),
		GoogleAnalyticsTag: model.GoogleAnalyticsTag.ValueString(),
		AllowIndexing:      model.AllowIndexing.ValueBool(),
		ThemeColors:        statusPageV3ThemeColorsFromList(model.ThemeColors),
	}
}

// statusPageV3ToModel updates the model to match the API representation of
// the status page.
func statusPageV3ToModel(ctx context.Context, p *checkly.StatusPageV3, model *StatusPageV3ResourceModel, diags *diag.Diagnostics) {
	model.ID = types.StringValue(p.ID)
	model.Name = types.StringValue(p.Name)
	model.URL = types.StringValue(p.URL)
	model.CustomDomain = types.StringValue(p.CustomDomain)
	model.Description = types.StringValue(p.Description)
	model.Logo = types.StringValue(p.Logo)
	model.LogoDark = types.StringValue(p.LogoDark)
	model.RedirectTo = types.StringValue(p.RedirectTo)
	model.Favicon = types.StringValue(p.Favicon)
	model.DefaultTheme = types.StringValue(string(p.DefaultTheme))
	model.PrivacyPolicyLink = types.StringValue(p.PrivacyPolicyLink)
	model.TermsOfServiceLink = types.StringValue(p.TermsOfServiceLink)
	model.SupportLink = types.StringValue(p.SupportLink)
	model.FooterText = types.StringValue(p.FooterText)
	model.GoogleAnalyticsTag = types.StringValue(p.GoogleAnalyticsTag)
	model.AllowIndexing = types.BoolValue(p.AllowIndexing)

	// The API always reports a palette, the defaults when none is set, so
	// the colors are only tracked when configured. Tracking them regardless
	// would make every plan of an unconfigured page try to remove them.
	if statusPageV3ThemeColorsConfigured(model.ThemeColors) {
		model.ThemeColors = statusPageV3ThemeColorsToList(p.ThemeColors, model.ThemeColors, diags)
	} else {
		model.ThemeColors = types.ListValueMust(statusPageV3ThemeColorsObjectType, []attr.Value{})
	}
}

// statusPageV3ThemeColorsConfigured reports whether the theme_colors block
// value holds a palette.
func statusPageV3ThemeColorsConfigured(list types.List) bool {
	return !list.IsNull() && !list.IsUnknown() && len(list.Elements()) > 0
}

// statusPageV3FirstObject returns the single object of a MaxItems:1 block
// list, or false when there is none.
func statusPageV3FirstObject(value attr.Value) (types.Object, bool) {
	list, ok := value.(types.List)
	if !ok || list.IsNull() || list.IsUnknown() || len(list.Elements()) == 0 {
		return types.Object{}, false
	}

	obj, ok := list.Elements()[0].(types.Object)
	if !ok || obj.IsNull() || obj.IsUnknown() {
		return types.Object{}, false
	}

	return obj, true
}

func statusPageV3ThemeColorGroupFromList(value attr.Value) checkly.StatusPageV3ThemeColorGroup {
	var group checkly.StatusPageV3ThemeColorGroup

	obj, ok := statusPageV3FirstObject(value)
	if !ok {
		return group
	}

	attrs := obj.Attributes()
	for _, a := range statusPageV3ThemeColorAttributes {
		if color, ok := attrs[a.key].(types.String); ok {
			*a.field(&group) = color.ValueString()
		}
	}

	return group
}

func statusPageV3ThemeColorsFromList(list types.List) *checkly.StatusPageV3ThemeColors {
	obj, ok := statusPageV3FirstObject(list)
	if !ok {
		return nil
	}

	attrs := obj.Attributes()

	return &checkly.StatusPageV3ThemeColors{
		Light: statusPageV3ThemeColorGroupFromList(attrs["light"]),
		Dark:  statusPageV3ThemeColorGroupFromList(attrs["dark"]),
	}
}

// statusPageV3ThemeColorGroupToList converts a color group to its block
// value. Hex colors are case-insensitive, so a prior value that only differs
// from the API's in case is kept to avoid a spurious diff.
func statusPageV3ThemeColorGroupToList(group *checkly.StatusPageV3ThemeColorGroup, prior attr.Value, diags *diag.Diagnostics) types.List {
	var priorAttrs map[string]attr.Value
	if obj, ok := statusPageV3FirstObject(prior); ok {
		priorAttrs = obj.Attributes()
	}

	colors := make(map[string]attr.Value, len(statusPageV3ThemeColorAttributes))
	for _, a := range statusPageV3ThemeColorAttributes {
		color := *a.field(group)
		if p, ok := priorAttrs[a.key].(types.String); ok && !p.IsNull() && !p.IsUnknown() && strings.EqualFold(p.ValueString(), color) {
			color = p.ValueString()
		}
		colors[a.key] = types.StringValue(color)
	}

	obj, d := types.ObjectValue(statusPageV3ThemeColorGroupAttrTypes, colors)
	diags.Append(d...)

	list, d := types.ListValue(statusPageV3ThemeColorGroupObjectType, []attr.Value{obj})
	diags.Append(d...)

	return list
}

// statusPageV3ThemeColorsToList converts the API palette to the
// theme_colors block value, using prior to keep the configured spelling of
// each color.
func statusPageV3ThemeColorsToList(themeColors *checkly.StatusPageV3ThemeColors, prior types.List, diags *diag.Diagnostics) types.List {
	if themeColors == nil {
		return types.ListValueMust(statusPageV3ThemeColorsObjectType, []attr.Value{})
	}

	var priorAttrs map[string]attr.Value
	if obj, ok := statusPageV3FirstObject(prior); ok {
		priorAttrs = obj.Attributes()
	}

	obj, d := types.ObjectValue(statusPageV3ThemeColorsAttrTypes, map[string]attr.Value{
		"light": statusPageV3ThemeColorGroupToList(&themeColors.Light, priorAttrs["light"], diags),
		"dark":  statusPageV3ThemeColorGroupToList(&themeColors.Dark, priorAttrs["dark"], diags),
	})
	diags.Append(d...)

	list, d := types.ListValue(statusPageV3ThemeColorsObjectType, []attr.Value{obj})
	diags.Append(d...)

	return list
}
