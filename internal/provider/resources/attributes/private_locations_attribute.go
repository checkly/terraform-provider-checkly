package attributes

import (
	"context"

	"github.com/checkly/terraform-provider-checkly/internal/provider/interop"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ interop.Model[string] = (*PrivateLocationsAttributeModel)(nil)
)

var PrivateLocationsAttributeSchema = schema.SetAttribute{
	Description: "An array of one or more private locations slugs.",
	ElementType: types.StringType,
	Optional:    true,
}

type PrivateLocationsAttributeModel string

var PrivateLocationsAttributeGluer = interop.GluerForSetAttribute[
	string,
	PrivateLocationsAttributeModel,
](PrivateLocationsAttributeSchema)

func (m *PrivateLocationsAttributeModel) Refresh(ctx context.Context, from *string, flags interop.RefreshFlags) diag.Diagnostics {
	*m = PrivateLocationsAttributeModel(*from)

	return nil
}

func (m *PrivateLocationsAttributeModel) Render(ctx context.Context, into *string) diag.Diagnostics {
	*into = string(*m)

	return nil
}
