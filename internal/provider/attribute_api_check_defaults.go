package provider

import (
	"context"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

const apiCheckDefaultsAttributeName = "api_check_defaults"

// apiCheckDefaultsModel describes the api_check_defaults attribute data
// model.
type apiCheckDefaultsModel struct {
	URL             types.String `tfsdk:"url"`
	Headers         types.Map    `tfsdk:"headers"`
	QueryParameters types.Map    `tfsdk:"query_parameters"`
	Assertions      types.Set    `tfsdk:"assertions"`
	BasicAuth       types.Object `tfsdk:"basic_auth"`
}

var apiCheckDefaultsAttrTypes = map[string]attr.Type{
	"url":              types.StringType,
	"headers":          types.MapType{ElemType: types.StringType},
	"query_parameters": types.MapType{ElemType: types.StringType},
	"assertions":       types.SetType{ElemType: types.ObjectType{AttrTypes: assertionAttrTypes}},
	"basic_auth":       types.ObjectType{AttrTypes: apiCheckBasicAuthAttrTypes},
}

// apiCheckDefaultsDefaultObject returns the canonical attribute value for a
// group without API check defaults: an empty base URL, no headers or query
// parameters, no assertions and no basic auth credentials. This matches what
// the Checkly API stores when no defaults are configured, so removing the
// attribute from the configuration plans a change back to this value.
func apiCheckDefaultsDefaultObject() types.Object {
	return types.ObjectValueMust(apiCheckDefaultsAttrTypes, map[string]attr.Value{
		"url":              types.StringValue(""),
		"headers":          types.MapNull(types.StringType),
		"query_parameters": types.MapNull(types.StringType),
		"assertions":       types.SetNull(types.ObjectType{AttrTypes: assertionAttrTypes}),
		"basic_auth":       types.ObjectNull(apiCheckBasicAuthAttrTypes),
	})
}

// makeAPICheckDefaultsAttributeSchema returns the schema for the
// api_check_defaults attribute. The attribute is optional+computed and
// defaults to empty API check defaults.
func makeAPICheckDefaultsAttributeSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
		Computed:    true,
		Default:     objectdefault.StaticValue(apiCheckDefaultsDefaultObject()),
		Description: "Default settings for API checks in this group.",
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Required:    true,
				Description: "The base url for this group which you can reference with the `GROUP_BASE_URL` variable in all group checks.",
			},
			"headers": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Default headers for API checks in this group.",
			},
			"query_parameters": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Default query parameters for API checks in this group.",
			},
			"assertions": schema.SetNestedAttribute{
				Optional:    true,
				Description: "Default assertions for API checks in this group.",
				NestedObject: schema.NestedAttributeObject{
					// Attributes nested in this set attribute must not use
					// schema defaults. The framework looks up configuration
					// values for set elements by full element value; once
					// computed attributes resolve, the lookup misses, the
					// configuration is treated as null, and defaults would
					// override values the practitioner actually configured,
					// producing a perpetual diff. Default values are instead
					// applied in ModifyPlan of the resources using this
					// attribute.
					Attributes: map[string]schema.Attribute{
						"source": schema.StringAttribute{
							Required:    true,
							Description: "The source of the asserted value. Possible values `STATUS_CODE`, `JSON_BODY`, `HEADERS`, `TEXT_BODY`, and `RESPONSE_TIME`.",
						},
						"property": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "The property of the asserted value, for example a JSON path or header name.",
						},
						"comparison": schema.StringAttribute{
							Required:    true,
							Description: "The type of comparison to be executed between expected and actual value of the assertion. Possible values `EQUALS`, `NOT_EQUALS`, `HAS_KEY`, `NOT_HAS_KEY`, `HAS_VALUE`, `NOT_HAS_VALUE`, `IS_EMPTY`, `NOT_EMPTY`, `GREATER_THAN`, `LESS_THAN`, `CONTAINS`, `NOT_CONTAINS`, `IS_NULL`, and `NOT_NULL`.",
						},
						"target": schema.StringAttribute{
							Required:    true,
							Description: "The value to compare against.",
						},
					},
				},
			},
			"basic_auth": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Default basic auth credentials for API checks in this group.",
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
		},
	}
}

// apiCheckDefaultsKeyValuesFromMap converts a map of strings to its API
// representation. Null and unknown maps convert to nil. The result is sorted
// by key for determinism.
func apiCheckDefaultsKeyValuesFromMap(ctx context.Context, m types.Map, diags *diag.Diagnostics) []checkly.KeyValue {
	if m.IsNull() || m.IsUnknown() {
		return nil
	}

	var values map[string]string
	diags.Append(m.ElementsAs(ctx, &values, false)...)
	if diags.HasError() {
		return nil
	}

	keyValues := make([]checkly.KeyValue, 0, len(values))
	for key, value := range values {
		keyValues = append(keyValues, checkly.KeyValue{
			Key:   key,
			Value: value,
		})
	}

	sort.Slice(keyValues, func(i, j int) bool {
		return keyValues[i].Key < keyValues[j].Key
	})

	return keyValues
}

// apiCheckDefaultsFromObject converts the api_check_defaults attribute value
// to its API representation. Null and unknown values convert to empty API
// check defaults.
func apiCheckDefaultsFromObject(ctx context.Context, obj types.Object, diags *diag.Diagnostics) checkly.APICheckDefaults {
	model := objectAs[apiCheckDefaultsModel](ctx, obj, diags)
	if model == nil {
		return checkly.APICheckDefaults{}
	}

	defaults := checkly.APICheckDefaults{
		BaseURL:         model.URL.ValueString(),
		Headers:         apiCheckDefaultsKeyValuesFromMap(ctx, model.Headers, diags),
		QueryParameters: apiCheckDefaultsKeyValuesFromMap(ctx, model.QueryParameters, diags),
		Assertions:      assertionsFromSet(ctx, model.Assertions, diags),
	}

	if basicAuth := apiCheckBasicAuthFromObject(ctx, model.BasicAuth, diags); basicAuth != nil {
		defaults.BasicAuth = *basicAuth
	}

	return defaults
}

// apiCheckDefaultsToObject converts the API representation of API check
// defaults to the api_check_defaults attribute value.
//
// The Checkly API reports empty basic auth credentials both when no basic
// auth is configured and when it is configured with empty strings. When the
// API reports empty credentials, the basic_auth value from prior (the planned
// or previously stored attribute value) is preserved when it also holds empty
// credentials, so that the resulting value stays consistent with the plan.
func apiCheckDefaultsToObject(ctx context.Context, defaults checkly.APICheckDefaults, prior types.Object, diags *diag.Diagnostics) types.Object {
	basicAuth := types.ObjectNull(apiCheckBasicAuthAttrTypes)

	if defaults.BasicAuth.Username != "" || defaults.BasicAuth.Password != "" {
		basicAuth = types.ObjectValueMust(apiCheckBasicAuthAttrTypes, map[string]attr.Value{
			"username": types.StringValue(defaults.BasicAuth.Username),
			"password": types.StringValue(defaults.BasicAuth.Password),
		})
	} else if !prior.IsNull() && !prior.IsUnknown() {
		if priorBasicAuth, ok := prior.Attributes()["basic_auth"].(types.Object); ok &&
			!priorBasicAuth.IsNull() && !priorBasicAuth.IsUnknown() {
			priorAttrs := priorBasicAuth.Attributes()
			if stringAttrValue(priorAttrs["username"]) == "" && stringAttrValue(priorAttrs["password"]) == "" {
				basicAuth = priorBasicAuth
			}
		}
	}

	obj, d := types.ObjectValue(apiCheckDefaultsAttrTypes, map[string]attr.Value{
		"url":              types.StringValue(defaults.BaseURL),
		"headers":          stringMapFromAPINullIfUnset(ctx, defaults.Headers, priorAttribute(prior, "headers", types.MapNull(types.StringType)), diags),
		"query_parameters": stringMapFromAPINullIfUnset(ctx, defaults.QueryParameters, priorAttribute(prior, "query_parameters", types.MapNull(types.StringType)), diags),
		"assertions":       assertionsToSet(ctx, defaults.Assertions, priorAssertions(prior), diags),
		"basic_auth":       basicAuth,
	})
	diags.Append(d...)

	return obj
}
