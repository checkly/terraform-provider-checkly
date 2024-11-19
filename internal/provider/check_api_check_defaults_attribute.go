package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
)

var (
	_ ResourceModel[checkly.APICheckDefaults] = (*CheckAPICheckDefaultsAttributeModel)(nil)
)

var CheckAPICheckDefaultsAttributeSchema = schema.SingleNestedAttribute{
	Optional: true,
	Computed: true,
	Attributes: map[string]schema.Attribute{
		"url": schema.StringAttribute{
			Required:    true,
			Description: "The base url for this group which you can reference with the `GROUP_BASE_URL` variable in all group checks.",
		},
		"headers":          CheckHeadersAttributeSchema,
		"query_parameters": CheckQueryParametersAttributeSchema,
		"assertion":        CheckAssertionAttributeSchema,
		"basic_auth":       CheckBasicAuthAttributeSchema,
	},
}

type CheckAPICheckDefaultsAttributeModel struct {
	URL             types.String                   `tfsdk:"url"`
	Headers         types.Map                      `tfsdk:"headers"`
	QueryParameters types.Map                      `tfsdk:"query_parameters"`
	Assertions      []CheckAssertionAttributeModel `tfsdk:"assertion"`
	BasicAuth       CheckBasicAuthAttributeModel   `tfsdk:"basic_auth"`
}

func (m *CheckAPICheckDefaultsAttributeModel) Refresh(ctx context.Context, from *checkly.APICheckDefaults, flags RefreshFlags) diag.Diagnostics {
	var diags diag.Diagnostics

	m.URL = types.StringValue(from.BaseURL)
	m.Headers = SDKKeyValuesIntoMap(&from.Headers)
	m.QueryParameters = SDKKeyValuesIntoMap(&from.QueryParameters)

	diags.Append(RefreshMany(ctx, from.Assertions, m.Assertions, flags)...)

	diags.Append(m.BasicAuth.Refresh(ctx, &from.BasicAuth, flags)...)

	return diags
}

func (m *CheckAPICheckDefaultsAttributeModel) Render(ctx context.Context, into *checkly.APICheckDefaults) diag.Diagnostics {
	var diags diag.Diagnostics

	into.BaseURL = m.URL.ValueString()
	into.Headers = SDKKeyValuesFromMap(m.Headers)
	into.QueryParameters = SDKKeyValuesFromMap(m.QueryParameters)

	diags.Append(RenderMany(ctx, m.Assertions, into.Assertions)...)

	diags.Append(m.BasicAuth.Render(ctx, &into.BasicAuth)...)

	return diags
}
