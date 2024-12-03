package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

var IDResourceAttributeSchema = schema.StringAttribute{
	Computed:    true,
	Description: "The ID of this resource.",
}

var IDDataSourceAttributeSchema = schema.StringAttribute{
	Computed:    true,
	Description: "The ID of this data source.",
}
