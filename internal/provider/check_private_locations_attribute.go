package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ ResourceModel[[]string] = (*CheckPrivateLocationsAttributeModel)(nil)
)

var CheckPrivateLocationsAttributeSchema = schema.SetAttribute{
	Description: "An array of one or more private locations slugs.",
	ElementType: types.StringType,
	Optional:    true,
}

type CheckPrivateLocationsAttributeModel types.Set

func (m *CheckPrivateLocationsAttributeModel) Refresh(ctx context.Context, from *[]string, flags RefreshFlags) diag.Diagnostics {
	*m = CheckPrivateLocationsAttributeModel(IntoUntypedStringSet(from))

	return nil
}

func (m *CheckPrivateLocationsAttributeModel) Render(ctx context.Context, into *[]string) diag.Diagnostics {
	*into = FromUntypedStringSet(types.Set(*m))

	return nil
}
