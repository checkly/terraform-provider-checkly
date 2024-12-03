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
			Required:    true,
			Description: "The base url for this group which you can reference with the `GROUP_BASE_URL` variable in all group checks.",
		},
		"headers":          HeadersAttributeSchema,
		"query_parameters": QueryParametersAttributeSchema,
		"assertion":        AssertionAttributeSchema,
		"basic_auth":       BasicAuthAttributeSchema,
	},
}

type APICheckDefaultsAttributeModel struct {
	URL             types.String              `tfsdk:"url"`
	Headers         types.Map                 `tfsdk:"headers"`
	QueryParameters types.Map                 `tfsdk:"query_parameters"`
	Assertions      []AssertionAttributeModel `tfsdk:"assertion"`
	BasicAuth       BasicAuthAttributeModel   `tfsdk:"basic_auth"`
}

func (m *APICheckDefaultsAttributeModel) Refresh(ctx context.Context, from *checkly.APICheckDefaults, flags interop.RefreshFlags) diag.Diagnostics {
	var diags diag.Diagnostics

	m.URL = types.StringValue(from.BaseURL)
	m.Headers = sdkutil.KeyValuesIntoMap(&from.Headers)
	m.QueryParameters = sdkutil.KeyValuesIntoMap(&from.QueryParameters)

	diags.Append(interop.RefreshMany(ctx, from.Assertions, m.Assertions, flags)...)

	diags.Append(m.BasicAuth.Refresh(ctx, &from.BasicAuth, flags)...)

	return diags
}

func (m *APICheckDefaultsAttributeModel) Render(ctx context.Context, into *checkly.APICheckDefaults) diag.Diagnostics {
	var diags diag.Diagnostics

	into.BaseURL = m.URL.ValueString()
	into.Headers = sdkutil.KeyValuesFromMap(m.Headers)
	into.QueryParameters = sdkutil.KeyValuesFromMap(m.QueryParameters)

	diags.Append(interop.RenderMany(ctx, m.Assertions, into.Assertions)...)

	diags.Append(m.BasicAuth.Render(ctx, &into.BasicAuth)...)

	return diags
}
