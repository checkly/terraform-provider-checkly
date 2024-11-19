package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ ResourceModel[[]string] = (*CheckLocationsAttributeModel)(nil)
)

var CheckLocationsAttributeSchema = schema.SetAttribute{
	Description: "An array of one or more data center locations where to run the checks.",
	ElementType: types.StringType,
	Optional:    true,
}

type CheckLocationsAttributeModel types.Set

func (m *CheckLocationsAttributeModel) Refresh(ctx context.Context, from *[]string, flags RefreshFlags) diag.Diagnostics {
	*m = CheckLocationsAttributeModel(IntoUntypedStringSet(from))

	return nil
}

func (m *CheckLocationsAttributeModel) Render(ctx context.Context, into *[]string) diag.Diagnostics {
	*into = FromUntypedStringSet(types.Set(*m))

	return nil
}
