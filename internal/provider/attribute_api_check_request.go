package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	checkly "github.com/checkly/checkly-go-sdk"
)

// apiCheckRequestModel describes the request block data model.
type apiCheckRequestModel struct {
	Method          types.String `tfsdk:"method"`
	URL             types.String `tfsdk:"url"`
	FollowRedirects types.Bool   `tfsdk:"follow_redirects"`
	SkipSSL         types.Bool   `tfsdk:"skip_ssl"`
	Headers         types.Map    `tfsdk:"headers"`
	QueryParameters types.Map    `tfsdk:"query_parameters"`
	Body            types.String `tfsdk:"body"`
	BodyType        types.String `tfsdk:"body_type"`
	Assertion       types.Set    `tfsdk:"assertion"`
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
	"assertion":        types.SetType{ElemType: types.ObjectType{AttrTypes: assertionAttrTypes}},
	"basic_auth":       types.ObjectType{AttrTypes: apiCheckBasicAuthAttrTypes},
	"ip_family":        types.StringType,
}

// apiCheckRequestBlockSchemaOptions holds the settings that differ between
// the resources sharing the API check request block.
type apiCheckRequestBlockSchemaOptions struct {
	// Description is the description of the request block.
	Description string
	// URLDescription is the description of the request.url attribute.
	URLDescription string
	// Required makes the request block mandatory.
	Required bool
}

// apiCheckRequestBlockSchema returns the schema for the request block of an
// API check, which holds at most one request config.
func apiCheckRequestBlockSchema(opts apiCheckRequestBlockSchemaOptions) schema.SetNestedBlock {
	var validators []validator.Set
	if opts.Required {
		validators = append(validators, setvalidator.IsRequired())
	}
	validators = append(validators, setvalidator.SizeAtMost(1))

	return schema.SetNestedBlock{
		Description: opts.Description,
		Validators:  validators,
		NestedObject: schema.NestedBlockObject{
			// Attributes nested in this set block must not use schema
			// defaults. The framework looks up configuration values
			// for set elements by full element value; once computed
			// attributes resolve, the lookup misses, the configuration
			// is treated as null, and defaults would override values
			// the practitioner actually configured, producing a
			// perpetual diff. Default values are instead applied in
			// ModifyPlan.
			Attributes: map[string]schema.Attribute{
				"method": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "The HTTP method to use for this API check. Possible values are `GET`, `POST`, `PUT`, `HEAD`, `DELETE`, `PATCH`. (Default `GET`).",
				},
				"url": schema.StringAttribute{
					Required:    true,
					Description: opts.URLDescription,
				},
				"follow_redirects": schema.BoolAttribute{
					Optional: true,
					Computed: true,
				},
				"skip_ssl": schema.BoolAttribute{
					Optional: true,
					Computed: true,
				},
				"headers": schema.MapAttribute{
					Optional:    true,
					ElementType: types.StringType,
				},
				"query_parameters": schema.MapAttribute{
					Optional:    true,
					ElementType: types.StringType,
				},
				"body": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "The body of the request.",
				},
				"body_type": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "The `Content-Type` header of the request. Possible values `NONE`, `JSON`, `FORM`, `RAW`, and `GRAPHQL`. (Default `NONE`).",
					Validators: []validator.String{
						stringvalidator.OneOf("NONE", "JSON", "FORM", "RAW", "GRAPHQL"),
					},
				},
				"basic_auth": schema.SingleNestedAttribute{
					Optional:    true,
					Description: "Set up HTTP basic authentication (username & password).",
					Attributes: map[string]schema.Attribute{
						"username": schema.StringAttribute{
							Required: true,
						},
						"password": schema.StringAttribute{
							Required: true,
						},
					},
				},
				"ip_family": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "IP Family to be used when executing the api check. The value can be either IPv4 or IPv6. (Default `IPv4`).",
					Validators: []validator.String{
						stringvalidator.OneOf("IPv4", "IPv6"),
					},
				},
			},
			Blocks: map[string]schema.Block{
				"assertion": schema.SetNestedBlock{
					Description: "A request can have multiple assertions.",
					NestedObject: schema.NestedBlockObject{
						Attributes: map[string]schema.Attribute{
							"source": schema.StringAttribute{
								Required:    true,
								Description: "The source of the asserted value. Possible values `STATUS_CODE`, `JSON_BODY`, `HEADERS`, `TEXT_BODY`, and `RESPONSE_TIME`.",
							},
							"property": schema.StringAttribute{
								Optional: true,
								Computed: true,
							},
							"comparison": schema.StringAttribute{
								Required:    true,
								Description: "The type of comparison to be executed between expected and actual value of the assertion. Possible values `EQUALS`, `NOT_EQUALS`, `HAS_KEY`, `NOT_HAS_KEY`, `HAS_VALUE`, `NOT_HAS_VALUE`, `IS_EMPTY`, `NOT_EMPTY`, `GREATER_THAN`, `LESS_THAN`, `CONTAINS`, `NOT_CONTAINS`, `IS_NULL`, and `NOT_NULL`.",
							},
							"target": schema.StringAttribute{
								Optional: true,
								Computed: true,
							},
						},
					},
				},
			},
		},
	}
}

// apiCheckRequestWithDefaults returns the request block value with default
// values substituted for null and unknown optional+computed attributes.
func apiCheckRequestWithDefaults(ctx context.Context, set types.Set, diags *diag.Diagnostics) types.Set {
	if set.IsNull() || set.IsUnknown() {
		return set
	}

	var models []apiCheckRequestModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return set
	}

	for i, model := range models {
		models[i].Method = stringOrDefault(model.Method, "GET")
		models[i].FollowRedirects = boolOrDefault(model.FollowRedirects, false)
		models[i].SkipSSL = boolOrDefault(model.SkipSSL, false)
		models[i].Body = stringOrDefault(model.Body, "")
		models[i].BodyType = stringOrDefault(model.BodyType, "NONE")
		models[i].IPFamily = stringOrDefault(model.IPFamily, "IPv4")
		models[i].Assertion = assertionsWithPropertyAndTargetDefaults(ctx, model.Assertion, diags)
	}

	result, d := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: apiCheckRequestAttrTypes}, models)
	diags.Append(d...)

	return result
}

// apiCheckRequestFromSet converts the request block value to its API
// representation. An empty, null or unknown value converts to a zero-valued
// request.
func apiCheckRequestFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) checkly.Request {
	if set.IsNull() || set.IsUnknown() {
		return checkly.Request{}
	}

	var models []apiCheckRequestModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() || len(models) == 0 {
		return checkly.Request{}
	}

	model := models[0]

	return checkly.Request{
		Method:          model.Method.ValueString(),
		URL:             model.URL.ValueString(),
		FollowRedirects: model.FollowRedirects.ValueBool(),
		SkipSSL:         model.SkipSSL.ValueBool(),
		Body:            model.Body.ValueString(),
		BodyType:        model.BodyType.ValueString(),
		Headers:         apiCheckKeyValuesFromMap(ctx, model.Headers, diags),
		QueryParameters: apiCheckKeyValuesFromMap(ctx, model.QueryParameters, diags),
		Assertions:      assertionsFromSet(ctx, model.Assertion, diags),
		BasicAuth:       apiCheckBasicAuthFromObject(ctx, model.BasicAuth, diags),
		IPFamily:        model.IPFamily.ValueString(),
	}
}

// apiCheckRequestToSet converts the API representation of an API check
// request config to the request block value. The prior block value (the
// previously stored state) is consulted so that collections omitted from the
// configuration stay null instead of drifting to empty values.
func apiCheckRequestToSet(ctx context.Context, r checkly.Request, prior types.Set, diags *diag.Diagnostics) types.Set {
	elementType := types.ObjectType{AttrTypes: apiCheckRequestAttrTypes}

	priorModel := apiCheckRequestModel{
		Headers:         types.MapNull(types.StringType),
		QueryParameters: types.MapNull(types.StringType),
	}
	if !prior.IsNull() && !prior.IsUnknown() {
		var priorModels []apiCheckRequestModel
		diags.Append(prior.ElementsAs(ctx, &priorModels, false)...)
		if diags.HasError() {
			return types.SetNull(elementType)
		}
		if len(priorModels) > 0 {
			priorModel = priorModels[0]
		}
	}

	model := apiCheckRequestModel{
		Method:          types.StringValue(r.Method),
		URL:             types.StringValue(r.URL),
		FollowRedirects: types.BoolValue(r.FollowRedirects),
		SkipSSL:         types.BoolValue(r.SkipSSL),
		Headers:         apiCheckMapFromKeyValues(ctx, r.Headers, priorModel.Headers, diags),
		QueryParameters: apiCheckMapFromKeyValues(ctx, r.QueryParameters, priorModel.QueryParameters, diags),
		Body:            types.StringValue(r.Body),
		BodyType:        types.StringValue(r.BodyType),
		Assertion:       assertionsToSet(ctx, r.Assertions, diags),
		BasicAuth:       apiCheckBasicAuthToObject(ctx, r.BasicAuth, diags),
		IPFamily:        types.StringValue(r.IPFamily),
	}

	set, d := types.SetValueFrom(ctx, elementType, []apiCheckRequestModel{model})
	diags.Append(d...)

	return set
}

// apiCheckKeyValuesFromMap converts a map of strings to the API's key-value
// pair representation. Null and unknown maps convert to an empty (non-nil)
// slice, which the API stores as "no entries".
func apiCheckKeyValuesFromMap(ctx context.Context, m types.Map, diags *diag.Diagnostics) []checkly.KeyValue {
	kvs := []checkly.KeyValue{}

	if m.IsNull() || m.IsUnknown() {
		return kvs
	}

	var values map[string]string
	diags.Append(m.ElementsAs(ctx, &values, false)...)
	if diags.HasError() {
		return kvs
	}

	for key, value := range values {
		kvs = append(kvs, checkly.KeyValue{
			Key:   key,
			Value: value,
		})
	}

	return kvs
}

// apiCheckMapFromKeyValues converts the API's key-value pair representation
// to a map of strings. An empty value converts to null when the prior value
// is null, so that an omitted map does not cause drift against an empty API
// response.
func apiCheckMapFromKeyValues(ctx context.Context, kvs []checkly.KeyValue, prior types.Map, diags *diag.Diagnostics) types.Map {
	if len(kvs) == 0 && prior.IsNull() {
		return prior
	}

	values := map[string]string{}
	for _, kv := range kvs {
		values[kv.Key] = kv.Value
	}

	m, d := types.MapValueFrom(ctx, types.StringType, values)
	diags.Append(d...)

	return m
}

// apiCheckBasicAuthFromObject converts the request.basic_auth attribute
// value to its API representation. Null and unknown values convert to nil,
// which disables basic authentication.
func apiCheckBasicAuthFromObject(ctx context.Context, obj types.Object, diags *diag.Diagnostics) *checkly.BasicAuth {
	if obj.IsNull() || obj.IsUnknown() {
		return nil
	}

	var model apiCheckBasicAuthModel
	diags.Append(obj.As(ctx, &model, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
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
