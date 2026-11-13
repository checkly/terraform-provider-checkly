package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

var apiCheckRequestMethodValues = allowedValues[string]{
	{Value: "GET"},
	{Value: "POST"},
	{Value: "PUT"},
	{Value: "HEAD"},
	{Value: "DELETE"},
	{Value: "PATCH"},
}

var apiCheckRequestBodyTypeValues = allowedValues[string]{
	{Value: "NONE"},
	{Value: "JSON"},
	{Value: "FORM"},
	{Value: "RAW"},
	{Value: "GRAPHQL"},
}

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
	// DefaultFollowRedirects is the default of follow_redirects. checkly_api_check
	// follows redirects by default, as the web app and the CLI do; the
	// deprecated checkly_check keeps its v1.x default of not following them.
	DefaultFollowRedirects bool
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
				Description: "The HTTP method to use for this API check. " + apiCheckRequestMethodValues.String() + " Defaults to `GET`.",
				Validators: []validator.String{
					stringvalidator.OneOf(apiCheckRequestMethodValues.Values()...),
				},
			},
			"url": schema.StringAttribute{
				Required:    true,
				Description: "The URL to call. Must be between `1` and `2048` characters.",
				Validators: []validator.String{
					utf16LengthBetween(1, 2048),
				},
			},
			"follow_redirects": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(opts.DefaultFollowRedirects),
				Description: fmt.Sprintf("Whether to follow HTTP redirects. Defaults to `%t`.", opts.DefaultFollowRedirects),
			},
			"skip_ssl": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether to skip TLS certificate validation of the request. Defaults to `false`.",
			},
			"headers": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				Default:     emptyMapDefault(types.StringType),
				ElementType: types.StringType,
				Description: "HTTP headers to send with the request, as a map of header names to values. The values are not sensitive and show in plan output. To keep a credential in a header out of plan output, take it from a variable declared with `sensitive = true` or wrap it in `sensitive(...)`.",
			},
			"query_parameters": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				Default:     emptyMapDefault(types.StringType),
				ElementType: types.StringType,
				Description: "Query parameters to add to the request URL, as a map of parameter names to values. The values are not sensitive and show in plan output. To keep a credential in a parameter out of plan output, take it from a variable declared with `sensitive = true` or wrap it in `sensitive(...)`.",
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
				Description: "The `Content-Type` header of the request. " + apiCheckRequestBodyTypeValues.String() + " Defaults to `NONE`.",
				Validators: []validator.String{
					stringvalidator.OneOf(apiCheckRequestBodyTypeValues.Values()...),
				},
			},
			"assertions": schema.SetNestedAttribute{
				Optional:    true,
				Computed:    true,
				Default:     emptySetDefault(types.ObjectType{AttrTypes: assertionAttrTypes}),
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
							Description: "The source of the asserted value. Possible values are `STATUS_CODE`, `JSON_BODY`, `HEADERS`, `TEXT_BODY` and `RESPONSE_TIME`.",
						},
						"property": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "The property selecting the asserted value within the source: a JSONPath expression for `JSON_BODY`, a header name for `HEADERS`, or a regular expression for `TEXT_BODY` whose first capture group is asserted instead of the whole body. Not used for `STATUS_CODE` and `RESPONSE_TIME`.",
						},
						"comparison": schema.StringAttribute{
							Required:    true,
							Description: "The type of comparison to be executed between expected and actual value of the assertion. Possible values are `EQUALS`, `NOT_EQUALS`, `HAS_KEY`, `NOT_HAS_KEY`, `HAS_VALUE`, `NOT_HAS_VALUE`, `IS_EMPTY`, `NOT_EMPTY`, `GREATER_THAN`, `LESS_THAN`, `CONTAINS`, `NOT_CONTAINS`, `IS_NULL` and `NOT_NULL`.",
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
				Description: "IP Family to be used when executing the api check. " + ipFamilyValues.String() + " Defaults to `IPv4`.",
				Validators: []validator.String{
					stringvalidator.OneOf(ipFamilyValues.Values()...),
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
// request config to the request attribute value. prior is the previously
// stored request attribute value, which decides how empty basic auth
// credentials convert (see apiCheckBasicAuthToObject).
func apiCheckRequestToObject(ctx context.Context, r checkly.Request, prior types.Object, diags *diag.Diagnostics) types.Object {
	obj, d := types.ObjectValueFrom(ctx, apiCheckRequestAttrTypes, apiCheckRequestModel{
		Method:          types.StringValue(r.Method),
		URL:             types.StringValue(r.URL),
		FollowRedirects: types.BoolValue(r.FollowRedirects),
		SkipSSL:         types.BoolValue(r.SkipSSL),
		Headers:         stringMapFromKeyValues(ctx, r.Headers, diags),
		QueryParameters: stringMapFromKeyValues(ctx, r.QueryParameters, diags),
		Body:            types.StringValue(r.Body),
		BodyType:        types.StringValue(string(r.BodyType)),
		Assertions:      assertionsToSet(ctx, r.Assertions, diags),
		BasicAuth:       apiCheckBasicAuthToObject(ctx, r.BasicAuth, priorAttribute(prior, "basic_auth", types.ObjectNull(apiCheckBasicAuthAttrTypes)), diags),
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
// authentication credentials to the request.basic_auth attribute value.
//
// The API returns either no credentials or an empty username and password
// when basic authentication is not configured; checks created in the web app
// have the empty ones. Both convert to null, so an imported check without
// basic authentication matches a configuration without basic_auth. Empty
// credentials keep the prior value instead when it holds empty credentials
// too, so a configuration that sets them explicitly stays consistent.
func apiCheckBasicAuthToObject(ctx context.Context, ba *checkly.BasicAuth, prior types.Object, diags *diag.Diagnostics) types.Object {
	if ba == nil || (ba.Username == "" && ba.Password == "") {
		if !prior.IsNull() && !prior.IsUnknown() {
			priorAttrs := prior.Attributes()
			if stringAttrValue(priorAttrs["username"]) == "" && stringAttrValue(priorAttrs["password"]) == "" {
				return prior
			}
		}
		return types.ObjectNull(apiCheckBasicAuthAttrTypes)
	}

	obj, d := types.ObjectValueFrom(ctx, apiCheckBasicAuthAttrTypes, apiCheckBasicAuthModel{
		Username: types.StringValue(ba.Username),
		Password: types.StringValue(ba.Password),
	})
	diags.Append(d...)

	return obj
}
