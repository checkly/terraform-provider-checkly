package attributes

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/checkly/terraform-provider-checkly/internal/provider/interop"
	"github.com/checkly/terraform-provider-checkly/internal/sdkutil"
)

var (
	_ interop.Model[checkly.APICheckDefaults] = (*APICheckDefaultsAttributeModel)(nil)
)

var APICheckDefaultsAttributeSchema = schema.SingleNestedAttribute{
	Optional: true,
	Computed: true,
	Attributes: map[string]schema.Attribute{
		"url": schema.StringAttribute{
			Description: "The base url for this group which you can reference with the `GROUP_BASE_URL` variable in all group checks.",
			Required:    true,
		},
		"headers":          HeadersAttributeSchema,
		"query_parameters": QueryParametersAttributeSchema,
		"assertion":        AssertionAttributeSchema,
		"basic_auth":       BasicAuthAttributeSchema,
	},
}

type APICheckDefaultsAttributeModel struct {
	URL             types.String `tfsdk:"url"`
	Headers         types.Map    `tfsdk:"headers"`
	QueryParameters types.Map    `tfsdk:"query_parameters"`
	Assertions      types.List   `tfsdk:"assertions"`
	BasicAuth       types.Object `tfsdk:"basic_auth"`
}

var APICheckDefaultsAttributeGluer = interop.GluerForSingleNestedAttribute[
	checkly.APICheckDefaults,
	APICheckDefaultsAttributeModel,
](APICheckDefaultsAttributeSchema)

func (m *APICheckDefaultsAttributeModel) Refresh(ctx context.Context, from *checkly.APICheckDefaults, flags interop.RefreshFlags) diag.Diagnostics {
	var diags diag.Diagnostics

	m.URL = types.StringValue(from.BaseURL)
	m.Headers = sdkutil.KeyValuesIntoMap(&from.Headers)
	m.QueryParameters = sdkutil.KeyValuesIntoMap(&from.QueryParameters)

	m.Assertions, _, diags = AssertionAttributeGluer.RefreshToList(ctx, &from.Assertions, flags)
	if diags.HasError() {
		return diags
	}

	m.BasicAuth, _, diags = BasicAuthAttributeGluer.RefreshToObject(ctx, &from.BasicAuth, flags)
	if diags.HasError() {
		return diags
	}

	return diags
}

func (m *APICheckDefaultsAttributeModel) Render(ctx context.Context, into *checkly.APICheckDefaults) diag.Diagnostics {
	var diags diag.Diagnostics

	into.BaseURL = m.URL.ValueString()
	into.Headers = sdkutil.KeyValuesFromMap(m.Headers)
	into.QueryParameters = sdkutil.KeyValuesFromMap(m.QueryParameters)

	into.Assertions, _, diags = AssertionAttributeGluer.RenderFromList(ctx, m.Assertions)
	if diags.HasError() {
		return diags
	}

	into.BasicAuth, _, diags = BasicAuthAttributeGluer.RenderFromObject(ctx, m.BasicAuth)
	if diags.HasError() {
		return diags
	}

	return diags
}
