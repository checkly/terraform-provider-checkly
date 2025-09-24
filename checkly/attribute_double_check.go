package checkly

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const doubleCheckAttributeName = "double_check"

var doubleCheckAttributeSchema = &schema.Schema{
	Type:          schema.TypeBool,
	Optional:      true,
	Default:       false,
	Description:   "Setting this to `true` will trigger a retry when a check fails from the failing region and another, randomly selected region before marking the check as failed. (Default `false`).",
	Deprecated:    fmt.Sprintf("The property `double_check` is deprecated and will be removed in a future version. To enable retries for failed check runs, use the `%v` property instead.", retryStrategyAttributeName),
	ConflictsWith: []string{retryStrategyAttributeName},
}
