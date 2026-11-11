package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/checkly/checkly-go-sdk/v2"
)

// apiCheckRequestModel describes the request attribute data model.
type apiCheckRequestModel struct {
	Method          types.String `tfsdk:"method"`
	URL             types.String `tfsdk:"url"`
	FollowRedirects types.Bool   `tfsdk:"follow_redirects"`
	SkipSSL         types.Bool   `tfsdk:"skip_ssl"`
	Headers         types.Map    `tfsdk:"headers"`
	QueryParameters types.Map    `tfsdk:"query_parameters"`
	Body            types.String `tfsdk:"body"`
	BodyType        types.String `tfsdk:"body_type"`
	Assertions      types.Set    `tfsdk:"assertions"`
	BasicAuth       types.Object `tfsdk:"basic_auth"`
	IPFamily        types.String `tfsdk:"ip_family"`
}

// apiCheckBasicAuthModel describes the basic_auth attribute data model of a
// request and of api_check_defaults.
type apiCheckBasicAuthModel struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

var apiCheckBasicAuthAttrTypes = map[string]attr.Type{
	"username": types.StringType,
	"password": types.StringType,
}

var apiCheckRequestAttrTypes = map[string]attr.Type{
	"method":           types.StringType,
	"url":              types.StringType,
	"follow_redirects": types.BoolType,
	"skip_ssl":         types.BoolType,
	"headers":          types.MapType{ElemType: types.StringType},
	"query_parameters": types.MapType{ElemType: types.StringType},
	"body":             types.StringType,
	"body_type":        types.StringType,
	"assertions":       types.SetType{ElemType: types.ObjectType{AttrTypes: assertionAttrTypes}},
	"basic_auth":       types.ObjectType{AttrTypes: apiCheckBasicAuthAttrTypes},
	"ip_family":        types.StringType,
}

// apiCheckRequestAttributeSchemaOptions holds the settings that differ
// between the resources sharing the API check request attribute.
type apiCheckRequestAttributeSchemaOptions struct {
	// Description is the description of the request attribute.
	Description string
	// Required makes the request attribute mandatory.
	Required bool
}

// apiCheckRequestAttributeSchema returns the schema for the request
// attribute of an API check.
func apiCheckRequestAttributeSchema(opts apiCheckRequestAttributeSchemaOptions) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Required:    opts.Required,
		Optional:    !opts.Required,
		Description: opts.Description,
		Attributes: map[string]schema.Attribute{
			"method": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("GET"),
				Description: "The HTTP method to use for this API check. Possible values are `GET`, `POST`, `PUT`, `HEAD`, `DELETE`, `PATCH`. (Default `GET`).",
			},
			"url": schema.StringAttribute{
				Required:    true,
				Description: "The URL to call (1 to 2048 characters).",
				Validators: []validator.String{
					utf16LengthBetween(1, 2048),
				},
			},
			"follow_redirects": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether to follow HTTP redirects. (Default `false`).",
			},
			"skip_ssl": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether to skip TLS certificate validation of the request. (Default `false`).",
			},
			"headers": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "HTTP headers to send with the request, as a map of header names to values.",
			},
			"query_parameters": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Query parameters to add to the request URL, as a map of parameter names to values.",
			},
			"body": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "The body of the request.",
			},
			"body_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("NONE"),
				Description: "The `Content-Type` header of the request. Possible values `NONE`, `JSON`, `FORM`, `RAW`, and `GRAPHQL`. (Default `NONE`).",
				Validators: []validator.String{
					stringvalidator.OneOf("NONE", "JSON", "FORM", "RAW", "GRAPHQL"),
				},
			},
			"assertions": schema.SetNestedAttribute{
				Optional:    true,
				Description: "A request can have multiple assertions.",
				NestedObject: schema.NestedAttributeObject{
					// Attributes nested in this set attribute must not use
					// schema defaults. The framework looks up configuration
					// values for set elements by full element value; once
					// computed attributes resolve, the lookup misses, the
					// configuration is treated as null, and defaults would
					// override values the practitioner actually configured,
					// producing a perpetual diff. Default values are instead
					// applied in ModifyPlan.
					Attributes: map[string]schema.Attribute{
						"source": schema.StringAttribute{
							Required:    true,
							Description: "The source of the asserted value. Possible values `STATUS_CODE`, `JSON_BODY`, `HEADERS`, `TEXT_BODY`, and `RESPONSE_TIME`.",
						},
						"property": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "The property selecting the asserted value within the source: a JSONPath expression for `JSON_BODY`, a header name for `HEADERS`, or a regular expression for `TEXT_BODY` whose first capture group is asserted instead of the whole body. Not used for `STATUS_CODE` and `RESPONSE_TIME`.",
						},
						"comparison": schema.StringAttribute{
							Required:    true,
							Description: "The type of comparison to be executed between expected and actual value of the assertion. Possible values `EQUALS`, `NOT_EQUALS`, `HAS_KEY`, `NOT_HAS_KEY`, `HAS_VALUE`, `NOT_HAS_VALUE`, `IS_EMPTY`, `NOT_EMPTY`, `GREATER_THAN`, `LESS_THAN`, `CONTAINS`, `NOT_CONTAINS`, `IS_NULL`, and `NOT_NULL`.",
						},
						"target": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "The value to compare against, e.g. `200` for `STATUS_CODE` or a number of milliseconds for `RESPONSE_TIME`. Leave empty for comparisons that take no value, such as `IS_EMPTY` and `NOT_NULL`.",
						},
					},
				},
			},
			"basic_auth": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Set up HTTP basic authentication (username & password).",
				Attributes: map[string]schema.Attribute{
					"username": schema.StringAttribute{
						Required:    true,
						Description: "The basic auth username.",
					},
					"password": schema.StringAttribute{
						Required:    true,
						Sensitive:   true,
						Description: "The basic auth password.",
					},
				},
			},
			"ip_family": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("IPv4"),
				Description: "IP Family to be used when executing the api check. The value can be either IPv4 or IPv6. (Default `IPv4`).",
				Validators: []validator.String{
					stringvalidator.OneOf("IPv4", "IPv6"),
				},
			},
		},
	}
}

// apiCheckRequestFromObject converts the request attribute value to its API
// representation. A null or unknown value converts to a zero-valued request.
func apiCheckRequestFromObject(ctx context.Context, obj types.Object, diags *diag.Diagnostics) checkly.Request {
	model := objectAs[apiCheckRequestModel](ctx, obj, diags)
	if model == nil {
		return checkly.Request{}
	}

	return checkly.Request{
		Method:          model.Method.ValueString(),
		URL:             model.URL.ValueString(),
		FollowRedirects: model.FollowRedirects.ValueBool(),
		SkipSSL:         model.SkipSSL.ValueBool(),
		Body:            model.Body.ValueString(),
		BodyType:        checkly.BodyType(model.BodyType.ValueString()),
		Headers:         keyValuesFromStringMap(ctx, model.Headers, diags),
		QueryParameters: keyValuesFromStringMap(ctx, model.QueryParameters, diags),
		Assertions:      assertionsFromSet(ctx, model.Assertions, diags),
		BasicAuth:       apiCheckBasicAuthFromObject(ctx, model.BasicAuth, diags),
		IPFamily:        checkly.IPFamily(model.IPFamily.ValueString()),
	}
}

// apiCheckRequestToObject converts the API representation of an API check
// request config to the request attribute value. The prior attribute value
// (the previously stored state) is consulted so that headers, query
// parameters and assertions omitted from the configuration stay null instead
// of drifting to empty collections.
func apiCheckRequestToObject(ctx context.Context, r checkly.Request, prior types.Object, diags *diag.Diagnostics) types.Object {
	priorModel := apiCheckRequestModel{
		Headers:         types.MapNull(types.StringType),
		QueryParameters: types.MapNull(types.StringType),
		Assertions:      types.SetNull(types.ObjectType{AttrTypes: assertionAttrTypes}),
	}
	if !prior.IsNull() && !prior.IsUnknown() {
		diags.Append(prior.As(ctx, &priorModel, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return types.ObjectNull(apiCheckRequestAttrTypes)
		}
	}

	obj, d := types.ObjectValueFrom(ctx, apiCheckRequestAttrTypes, apiCheckRequestModel{
		Method:          types.StringValue(r.Method),
		URL:             types.StringValue(r.URL),
		FollowRedirects: types.BoolValue(r.FollowRedirects),
		SkipSSL:         types.BoolValue(r.SkipSSL),
		Headers:         stringMapFromAPINullIfUnset(ctx, r.Headers, priorModel.Headers, diags),
		QueryParameters: stringMapFromAPINullIfUnset(ctx, r.QueryParameters, priorModel.QueryParameters, diags),
		Body:            types.StringValue(r.Body),
		BodyType:        types.StringValue(string(r.BodyType)),
		Assertions:      assertionsToSet(ctx, r.Assertions, priorModel.Assertions, diags),
		BasicAuth:       apiCheckBasicAuthToObject(ctx, r.BasicAuth, diags),
		IPFamily:        types.StringValue(string(r.IPFamily)),
	})
	diags.Append(d...)

	return obj
}

// apiCheckBasicAuthFromObject converts the request.basic_auth attribute
// value to its API representation. Null and unknown values convert to nil,
// which disables basic authentication.
func apiCheckBasicAuthFromObject(ctx context.Context, obj types.Object, diags *diag.Diagnostics) *checkly.BasicAuth {
	model := objectAs[apiCheckBasicAuthModel](ctx, obj, diags)
	if model == nil {
		return nil
	}

	return &checkly.BasicAuth{
		Username: model.Username.ValueString(),
		Password: model.Password.ValueString(),
	}
}

// apiCheckBasicAuthToObject converts the API representation of basic
// authentication credentials to the request.basic_auth attribute value. The
// API returns no credentials when basic authentication is not configured,
// which converts to null.
func apiCheckBasicAuthToObject(ctx context.Context, ba *checkly.BasicAuth, diags *diag.Diagnostics) types.Object {
	if ba == nil {
		return types.ObjectNull(apiCheckBasicAuthAttrTypes)
	}

	obj, d := types.ObjectValueFrom(ctx, apiCheckBasicAuthAttrTypes, apiCheckBasicAuthModel{
		Username: types.StringValue(ba.Username),
		Password: types.StringValue(ba.Password),
	})
	diags.Append(d...)

	return obj
}
