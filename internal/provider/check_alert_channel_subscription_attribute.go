package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
)

var (
	_ ResourceModel[checkly.AlertChannelSubscription] = (*CheckAlertChannelSubscriptionAttributeModel)(nil)
)

var CheckAlertChannelSubscriptionAttributeSchema = schema.ListNestedAttribute{
	Description: "An array of channel IDs and whether they're activated or " +
		"not. If you don't set at least one alert subscription for your " +
		"check, we won't be able to alert you in case something goes wrong " +
		"with it.",
	Optional: true,
	NestedObject: schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"channel_id": schema.Int64Attribute{
				Required: true,
			},
			"activated": schema.BoolAttribute{
				Required: true,
			},
		},
	},
}

type CheckAlertChannelSubscriptionAttributeModel struct {
	ChannelID types.Int64 `tfsdk:"channel_id"`
	Activated types.Bool  `tfsdk:"activated"`
}

func (m *CheckAlertChannelSubscriptionAttributeModel) Refresh(ctx context.Context, from *checkly.AlertChannelSubscription, flags RefreshFlags) diag.Diagnostics {
	m.ChannelID = types.Int64Value(from.ChannelID)
	m.Activated = types.BoolValue(from.Activated)

	return nil
}

func (m *CheckAlertChannelSubscriptionAttributeModel) Render(ctx context.Context, into *checkly.AlertChannelSubscription) diag.Diagnostics {
	into.ChannelID = m.ChannelID.ValueInt64()
	into.Activated = m.Activated.ValueBool()

	return nil
}
