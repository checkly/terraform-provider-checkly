package provider

import (
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var LastUpdatedAttributeSchema = schema.StringAttribute{
	Computed:    true,
	Description: "When the resource was last updated by the provider.",
}

func LastUpdatedNow() types.String {
	return types.StringValue(time.Now().Format(time.RFC850))
}
