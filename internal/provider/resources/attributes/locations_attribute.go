package attributes

import (
	"context"

	"github.com/checkly/terraform-provider-checkly/internal/provider/interop"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ interop.Model[string] = (*LocationsAttributeModel)(nil)
)

var LocationsAttributeSchema = schema.SetAttribute{
	Description: "An array of one or more data center locations where to run the checks.",
	ElementType: types.StringType,
	Optional:    true,
}

type LocationsAttributeModel string

var LocationsAttributeGluer = interop.GluerForSetAttribute[
	string,
	LocationsAttributeModel,
](LocationsAttributeSchema)

func (m *LocationsAttributeModel) Refresh(ctx context.Context, from *string, flags interop.RefreshFlags) diag.Diagnostics {
	*m = LocationsAttributeModel(*from)

	return nil
}

func (m *LocationsAttributeModel) Render(ctx context.Context, into *string) diag.Diagnostics {
	*into = string(*m)

	return nil
}
