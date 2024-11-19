package provider

import (
	"context"

	"github.com/checkly/checkly-go-sdk"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ ResourceModel[checkly.BasicAuth] = (*CheckBasicAuthAttributeModel)(nil)
)

var CheckBasicAuthAttributeSchema = schema.SingleNestedAttribute{
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

type CheckBasicAuthAttributeModel struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

func (m *CheckBasicAuthAttributeModel) Refresh(ctx context.Context, from *checkly.BasicAuth, flags RefreshFlags) diag.Diagnostics {
	m.Username = types.StringValue(from.Username)
	m.Password = types.StringValue(from.Password)

	return nil
}

func (m *CheckBasicAuthAttributeModel) Render(ctx context.Context, into *checkly.BasicAuth) diag.Diagnostics {
	into.Username = m.Username.ValueString()
	into.Password = m.Password.ValueString()

	return nil
}
