package checkly

import (
	"context"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const retryStrategyAttributeName = "retry_strategy"

var retryStrategyAttributeSchema = &schema.Schema{
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
		},
	},
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
		}
	default:
		return &checkly.RetryStrategy{
			Type:               res["type"].(string),
			BaseBackoffSeconds: res["base_backoff_seconds"].(int),
			MaxRetries:         res["max_retries"].(int),
			MaxDurationSeconds: res["max_duration_seconds"].(int),
			SameRegion:         res["same_region"].(bool),
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
			},
		}
	}
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
		return diff.SetNew(retryStrategyAttributeName, listFromRetryStrategy(nil))
	}

	_, new := diff.GetChange(retryStrategyAttributeName)

	newStrategy := retryStrategyFromList(new.([]any))
	newValue := listFromRetryStrategy(newStrategy)

	return diff.SetNew(retryStrategyAttributeName, newValue)
}
