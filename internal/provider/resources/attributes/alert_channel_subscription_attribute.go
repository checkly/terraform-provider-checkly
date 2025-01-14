package attributes

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/checkly/terraform-provider-checkly/internal/provider/interop"
)

var (
	_ interop.Model[checkly.AlertChannelSubscription] = (*AlertChannelSubscriptionAttributeModel)(nil)
)

var AlertChannelSubscriptionAttributeSchema = schema.ListNestedAttribute{
	Description: "An array of channel IDs and whether they're activated or " +
		"not. If you don't set at least one alert subscription for your " +
		"check, we won't be able to alert you in case something goes wrong " +
		"with it.",
	Optional: true,
	Computed: true,
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

type AlertChannelSubscriptionAttributeModel struct {
	ChannelID types.Int64 `tfsdk:"channel_id"`
	Activated types.Bool  `tfsdk:"activated"`
}

var AlertChannelSubscriptionAttributeGluer = interop.GluerForListNestedAttribute[
	checkly.AlertChannelSubscription,
	AlertChannelSubscriptionAttributeModel,
](AlertChannelSubscriptionAttributeSchema)

func (m *AlertChannelSubscriptionAttributeModel) Refresh(ctx context.Context, from *checkly.AlertChannelSubscription, flags interop.RefreshFlags) diag.Diagnostics {
	m.ChannelID = types.Int64Value(from.ChannelID)
	m.Activated = types.BoolValue(from.Activated)

	return nil
}

func (m *AlertChannelSubscriptionAttributeModel) Render(ctx context.Context, into *checkly.AlertChannelSubscription) diag.Diagnostics {
	into.ChannelID = m.ChannelID.ValueInt64()
	into.Activated = m.Activated.ValueBool()

	return nil
}
