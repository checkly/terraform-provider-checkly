package attributes

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

var IDAttributeSchema = schema.StringAttribute{
	Computed:    true,
	Description: "The ID of this data source.",
}
