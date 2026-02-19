package checkly

import (
	"context"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const retryStrategyAttributeName = "retry_strategy"

type RetryStrategyAttributeSchemaOptions struct {
	SupportsOnlyOnNetworkError bool
}

func makeRetryStrategyAttributeSchema(options RetryStrategyAttributeSchemaOptions) *schema.Schema {
	onlyOnSchema := map[string]*schema.Schema{}

	if options.SupportsOnlyOnNetworkError {
		onlyOnSchema["network_error"] = &schema.Schema{
			Description: "When `true`, retry only if the cause of the failure is a network error. (Default `false`).",
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
		}
	}

	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		Computed:    true,
		MaxItems:    1,
		Description: "A strategy for retrying failed check/monitor runs.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"type": {
					Description:  "Determines which type of retry strategy to use. Possible values are `FIXED`, `LINEAR`, `EXPONENTIAL`, `SINGLE_RETRY`, and `NO_RETRIES`.",
					Type:         schema.TypeString,
					Required:     true,
					ValidateFunc: validateOneOf([]string{"FIXED", "LINEAR", "EXPONENTIAL", "SINGLE_RETRY", "NO_RETRIES"}),
				},
				"base_backoff_seconds": {
					Description: "The number of seconds to wait before the first retry attempt. (Default `60`).",
					Type:        schema.TypeInt,
					Optional:    true,
					Default:     60,
				},
				"max_retries": {
					Description:  "The maximum number of times to retry the check/monitor. Value must be between `1` and `10`. Available when `type` is `FIXED`, `LINEAR`, or `EXPONENTIAL`. (Default `2`).",
					Type:         schema.TypeInt,
					Optional:     true,
					Default:      2,
					ValidateFunc: validateBetween(1, 10),
				},
				"max_duration_seconds": {
					Description:  "The total amount of time to continue retrying the check/monitor (maximum 600 seconds). Available when `type` is `FIXED`, `LINEAR`, or `EXPONENTIAL`. (Default `600`).",
					Type:         schema.TypeInt,
					Optional:     true,
					Default:      600,
					ValidateFunc: validateBetween(0, 600),
				},
				"same_region": {
					Description: "Whether retries should be run in the same region as the initial check/monitor run. (Default `true`).",
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     true,
				},
				"only_on": {
					Description: "Apply the retry strategy only if the defined conditions match.",
					Type:        schema.TypeList,
					MaxItems:    1,
					Optional:    true,
					Elem: &schema.Resource{
						Schema: onlyOnSchema,
					},
				},
			},
		},
	}
}

func retryStrategyFromList(s []any) *checkly.RetryStrategy {
	if len(s) == 0 {
		return nil
	}

	res := s[0].(tfMap)

	kind := res["type"].(string)

	switch kind {
	case "NO_RETRIES":
		return nil
	case "SINGLE_RETRY":
		return &checkly.RetryStrategy{
			Type:               res["type"].(string),
			BaseBackoffSeconds: res["base_backoff_seconds"].(int),
			SameRegion:         res["same_region"].(bool),
			OnlyOn:             retryStrategyOnlyOnFromList(res["only_on"].([]any)),
		}
	default:
		return &checkly.RetryStrategy{
			Type:               res["type"].(string),
			BaseBackoffSeconds: res["base_backoff_seconds"].(int),
			MaxRetries:         res["max_retries"].(int),
			MaxDurationSeconds: res["max_duration_seconds"].(int),
			SameRegion:         res["same_region"].(bool),
			OnlyOn:             retryStrategyOnlyOnFromList(res["only_on"].([]any)),
		}
	}
}

func listFromRetryStrategy(rs *checkly.RetryStrategy) []tfMap {
	if rs == nil || rs.Type == "NO_RETRIES" {
		return []tfMap{
			{
				"type": "NO_RETRIES",
			},
		}
	}

	switch rs.Type {
	case "SINGLE_RETRY":
		return []tfMap{
			{
				"type":                 rs.Type,
				"base_backoff_seconds": rs.BaseBackoffSeconds,
				"same_region":          rs.SameRegion,
				"only_on":              listFromRetryStrategyOnlyOn(rs.OnlyOn),
			},
		}
	default:
		return []tfMap{
			{
				"type":                 rs.Type,
				"base_backoff_seconds": rs.BaseBackoffSeconds,
				"max_retries":          rs.MaxRetries,
				"max_duration_seconds": rs.MaxDurationSeconds,
				"same_region":          rs.SameRegion,
				"only_on":              listFromRetryStrategyOnlyOn(rs.OnlyOn),
			},
		}
	}
}

func retryStrategyOnlyOnFromList(s []any) []string {
	var conditions []string

	for _, item := range s {
		res := item.(tfMap)

		if v, ok := res["network_error"].(bool); ok && v {
			conditions = append(conditions, "NETWORK_ERROR")
		}
	}

	return conditions
}

func listFromRetryStrategyOnlyOn(conditions []string) []tfMap {
	switch {
	case len(conditions) == 1 && conditions[0] == "NETWORK_ERROR":
		return []tfMap{
			{
				"network_error": true,
			},
		}
	default:
		return []tfMap{}
	}
}

var doubleCheckEquivalentRetryStrategy = checkly.RetryStrategy{
	Type:               "FIXED",
	MaxRetries:         1,
	BaseBackoffSeconds: 0,
	MaxDurationSeconds: 600,
	SameRegion:         false,
}

// RetryStrategy is a seemingly simple yet deceivingly complex attribute.
// Allowed values change depending on the retry strategy type. This
// function is used to normalize the value so we don't get into a diff loop.
func RetryStrategyCustomizeDiff(_ context.Context, diff *schema.ResourceDiff, meta any) error {
	// There's a weird issue where removing the `retry_strategy {}` attribute
	// is not seen as a change. It seems to be because HasChange, GetChange
	// and friends consult the state first. We can work around this by checking
	// whether the plan has a value for the attribute or not.
	empty := diff.GetRawPlan().GetAttr(retryStrategyAttributeName).LengthInt() == 0
	if empty {
		// If this resource has double_check and it's set to true, then we
		// need the change the default to the equivalent retry strategy to
		// avoid diff loops.
		doubleCheck := diff.GetRawPlan().Type().HasAttribute(doubleCheckAttributeName) &&
			diff.GetRawPlan().GetAttr(doubleCheckAttributeName).True()

		if doubleCheck {
			return diff.SetNew(retryStrategyAttributeName, listFromRetryStrategy(&doubleCheckEquivalentRetryStrategy))
		}

		return diff.SetNew(retryStrategyAttributeName, listFromRetryStrategy(nil))
	}

	_, new := diff.GetChange(retryStrategyAttributeName)

	newStrategy := retryStrategyFromList(new.([]any))
	newValue := listFromRetryStrategy(newStrategy)

	return diff.SetNew(retryStrategyAttributeName, newValue)
}
