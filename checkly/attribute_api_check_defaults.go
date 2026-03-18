package checkly

import (
	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const apiCheckDefaultsAttributeName = "api_check_defaults"

func makeAPICheckDefaultsAttributeSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		MaxItems: 1,
		Optional: true,
		Computed: true,
		DefaultFunc: func() (interface{}, error) {
			return []tfMap{
				{
					"url":              "",
					"headers":          []tfMap{},
					"query_parameters": []tfMap{},
					"basic_auth":       tfMap{},
				}}, nil
		},
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"url": {
					Type:        schema.TypeString,
					Required:    true,
					Description: "The base url for this group which you can reference with the `GROUP_BASE_URL` variable in all group checks.",
				},
				"headers": {
					Type:     schema.TypeMap,
					Optional: true,
					Computed: true,
					DefaultFunc: func() (interface{}, error) {
						return []tfMap{}, nil
					},
				},
				"query_parameters": {
					Type:     schema.TypeMap,
					Optional: true,
					Computed: true,
					DefaultFunc: func() (interface{}, error) {
						return []tfMap{}, nil
					},
				},
				"assertion": {
					Type:     schema.TypeSet,
					Optional: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"source": {
								Type:        schema.TypeString,
								Required:    true,
								Description: "The source of the asserted value. Possible values `STATUS_CODE`, `JSON_BODY`, `HEADERS`, `TEXT_BODY`, and `RESPONSE_TIME`.",
							},
							"property": {
								Type:     schema.TypeString,
								Optional: true,
							},
							"comparison": {
								Type:        schema.TypeString,
								Required:    true,
								Description: "The type of comparison to be executed between expected and actual value of the assertion. Possible values `EQUALS`, `NOT_EQUALS`, `HAS_KEY`, `NOT_HAS_KEY`, `HAS_VALUE`, `NOT_HAS_VALUE`, `IS_EMPTY`, `NOT_EMPTY`, `GREATER_THAN`, `LESS_THAN`, `CONTAINS`, `NOT_CONTAINS`, `IS_NULL`, and `NOT_NULL`.",
							},
							"target": {
								Type:     schema.TypeString,
								Required: true,
							},
						},
					},
				},
				"basic_auth": {
					Type:     schema.TypeSet,
					MaxItems: 1,
					Optional: true,
					Computed: true,
					DefaultFunc: func() (interface{}, error) {
						return []tfMap{}, nil
					},
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"username": {
								Type:     schema.TypeString,
								Required: true,
							},
							"password": {
								Type:     schema.TypeString,
								Required: true,
							},
						},
					},
				},
			},
		},
	}
}

func apiCheckDefaultsFromSet(s *schema.Set) checkly.APICheckDefaults {
	if s.Len() == 0 {
		return checkly.APICheckDefaults{}
	}
	res := s.List()[0].(tfMap)

	return checkly.APICheckDefaults{
		BaseURL:         res["url"].(string),
		Headers:         keyValuesFromMap(res["headers"].(tfMap)),
		QueryParameters: keyValuesFromMap(res["query_parameters"].(tfMap)),
		Assertions:      assertionsFromSet(res["assertion"].(*schema.Set)),
		BasicAuth:       checkGroupBasicAuthFromSet(res["basic_auth"].(*schema.Set)),
	}
}

func setFromAPICheckDefaults(a checkly.APICheckDefaults) []tfMap {
	s := tfMap{}
	s["url"] = a.BaseURL
	s["headers"] = mapFromKeyValues(a.Headers)
	s["query_parameters"] = mapFromKeyValues(a.QueryParameters)
	s["assertion"] = setFromAssertions(a.Assertions)
	s["basic_auth"] = checkGroupSetFromBasicAuth(a.BasicAuth)
	return []tfMap{s}
}

func checkGroupSetFromBasicAuth(b checkly.BasicAuth) []tfMap {
	if b.Username == "" && b.Password == "" {
		return []tfMap{}
	}
	return []tfMap{
		{
			"username": b.Username,
			"password": b.Password,
		},
	}
}

func checkGroupBasicAuthFromSet(s *schema.Set) checkly.BasicAuth {
	if s.Len() == 0 {
		return checkly.BasicAuth{
			Username: "",
			Password: "",
		}
	}
	res := s.List()[0].(tfMap)
	return checkly.BasicAuth{
		Username: res["username"].(string),
		Password: res["password"].(string),
	}
}
