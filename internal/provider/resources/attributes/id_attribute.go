package attributes

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

var IDAttributeSchema = schema.StringAttribute{
	Computed:    true,
	Description: "The ID of this resource.",
}
