package attributes

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/checkly/terraform-provider-checkly/internal/provider/interop"
	"github.com/checkly/terraform-provider-checkly/internal/sdkutil"
)

var (
	_ interop.Model[checkly.Request] = (*RequestAttributeModel)(nil)
)

var RequestAttributeSchema = schema.SingleNestedAttribute{
	Optional: true,
	Attributes: map[string]schema.Attribute{
		"method": schema.StringAttribute{
			Optional: true,
			Computed: true,
			Default:  stringdefault.StaticString("GET"),
			Validators: []validator.String{
				stringvalidator.OneOf(
					"GET",
					"POST",
					"PUT",
					"HEAD",
					"DELETE",
					"PATCH",
				),
			},
			Description: "The HTTP method to use for this API check. Possible values are `GET`, `POST`, `PUT`, `HEAD`, `DELETE`, `PATCH`. (Default `GET`).",
		},
		"url": schema.StringAttribute{
			Required: true,
		},
		"follow_redirects": schema.BoolAttribute{
			Optional: true,
		},
		"skip_ssl": schema.BoolAttribute{
			Optional: true,
		},
		"headers":          HeadersAttributeSchema,
		"query_parameters": QueryParametersAttributeSchema,
		"body": schema.StringAttribute{
			Optional:    true,
			Description: "The body of the request.",
		},
		"body_type": schema.StringAttribute{
			Optional: true,
			Computed: true,
			Default:  stringdefault.StaticString("NONE"),
			Validators: []validator.String{
				stringvalidator.OneOf(
					"NONE",
					"JSON",
					"FORM",
					"RAW",
					"GRAPHQL",
				),
			},
			Description: "The `Content-Type` header of the request. Possible values `NONE`, `JSON`, `FORM`, `RAW`, and `GRAPHQL`.",
		},
		"assertion":  AssertionAttributeSchema,
		"basic_auth": BasicAuthAttributeSchema,
		"ip_family": schema.StringAttribute{
			Optional: true,
			Computed: true,
			Default:  stringdefault.StaticString("IPv4"),
			Validators: []validator.String{
				stringvalidator.OneOf(
					"IPv4",
					"IPv6",
				),
			},
			Description: "IP Family to be used when executing the api check. The value can be either IPv4 or IPv6.",
		},
	},
}

type RequestAttributeModel struct {
	Method          types.String              `tfsdk:"method"`
	URL             types.String              `tfsdk:"url"`
	FollowRedirects types.Bool                `tfsdk:"follow_redirects"`
	SkipSSL         types.Bool                `tfsdk:"skip_ssl"`
	Headers         types.Map                 `tfsdk:"headers"`
	QueryParameters types.Map                 `tfsdk:"query_parameters"`
	Body            types.String              `tfsdk:"body"`
	BodyType        types.String              `tfsdk:"body_type"`
	Assertions      []AssertionAttributeModel `tfsdk:"assertion"`
	BasicAuth       BasicAuthAttributeModel   `tfsdk:"basic_auth"`
	IPFamily        types.String              `tfsdk:"ip_family"`
}

func (m *RequestAttributeModel) Refresh(ctx context.Context, from *checkly.Request, flags interop.RefreshFlags) diag.Diagnostics {
	m.Method = types.StringValue(from.Method)
	m.URL = types.StringValue(from.URL)
	m.FollowRedirects = types.BoolValue(from.FollowRedirects)
	m.SkipSSL = types.BoolValue(from.SkipSSL)
	m.Headers = sdkutil.KeyValuesIntoMap(&from.Headers)
	m.QueryParameters = sdkutil.KeyValuesIntoMap(&from.QueryParameters)
	m.Body = types.StringValue(from.Body)
	m.BodyType = types.StringValue(from.BodyType)

	diags := interop.RefreshMany(ctx, from.Assertions, m.Assertions, flags)
	if diags.HasError() {
		return diags
	}

	diags = m.BasicAuth.Refresh(ctx, from.BasicAuth, flags)
	if diags.HasError() {
		return diags
	}

	m.IPFamily = types.StringValue(from.IPFamily)

	return nil
}

func (m *RequestAttributeModel) Render(ctx context.Context, into *checkly.Request) diag.Diagnostics {
	into.Method = m.Method.ValueString()
	into.URL = m.URL.ValueString()
	into.FollowRedirects = m.FollowRedirects.ValueBool()
	into.SkipSSL = m.SkipSSL.ValueBool()
	into.Headers = sdkutil.KeyValuesFromMap(m.Headers)
	into.QueryParameters = sdkutil.KeyValuesFromMap(m.QueryParameters)
	into.Body = m.Body.ValueString()
	into.BodyType = m.Body.ValueString()

	diags := interop.RenderMany(ctx, m.Assertions, into.Assertions)
	if diags.HasError() {
		return diags
	}

	diags = m.BasicAuth.Render(ctx, into.BasicAuth)
	if diags.HasError() {
		return diags
	}

	into.IPFamily = m.IPFamily.ValueString()

	return nil
}
