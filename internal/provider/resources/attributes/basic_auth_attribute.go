package attributes

import (
	"context"

	"github.com/checkly/checkly-go-sdk"
	"github.com/checkly/terraform-provider-checkly/internal/provider/interop"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ interop.Model[checkly.BasicAuth] = (*BasicAuthAttributeModel)(nil)
)

var BasicAuthAttributeSchema = schema.SingleNestedAttribute{
	Description: "Credentials for Basic HTTP authentication.",
	Optional:    true,
	Computed:    true,
	Attributes: map[string]schema.Attribute{
		"username": schema.StringAttribute{
			Required: true,
		},
		"password": schema.StringAttribute{
			Required:  true,
			Sensitive: true,
		},
	},
}

type BasicAuthAttributeModel struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

var BasicAuthAttributeGluer = interop.GluerForSingleNestedAttribute[
	checkly.BasicAuth,
	BasicAuthAttributeModel,
](BasicAuthAttributeSchema)

func (m *BasicAuthAttributeModel) Refresh(ctx context.Context, from *checkly.BasicAuth, flags interop.RefreshFlags) diag.Diagnostics {
	m.Username = types.StringValue(from.Username)
	m.Password = types.StringValue(from.Password)

	return nil
}

func (m *BasicAuthAttributeModel) Render(ctx context.Context, into *checkly.BasicAuth) diag.Diagnostics {
	into.Username = m.Username.ValueString()
	into.Password = m.Password.ValueString()

	return nil
}
