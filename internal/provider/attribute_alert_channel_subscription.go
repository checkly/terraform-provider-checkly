package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

const alertChannelSubscriptionAttributeName = "alert_channel_subscriptions"

// AlertChannelSubscriptionModel describes an element of the
// alert_channel_subscriptions attribute.
type AlertChannelSubscriptionModel struct {
	ChannelID types.Int64 `tfsdk:"channel_id"`
	Activated types.Bool  `tfsdk:"activated"`
}

var alertChannelSubscriptionAttrTypes = map[string]attr.Type{
	"channel_id": types.Int64Type,
	"activated":  types.BoolType,
}

// AlertChannelSubscriptionAttributeSchemaOptions customizes the
// alert_channel_subscriptions attribute schema for a specific resource.
type AlertChannelSubscriptionAttributeSchemaOptions struct {
	// Subject names what the resource manages. For subjectCheckGroup the
	// descriptions talk about the checks and monitors in the group.
	Subject CheckSubject
}

// makeAlertChannelSubscriptionSchema returns the schema for the
// alert_channel_subscriptions attribute. Leaving it out stores an empty set.
func makeAlertChannelSubscriptionSchema(options AlertChannelSubscriptionAttributeSchemaOptions) schema.SetNestedAttribute {
	name := string(options.Subject)

	description := fmt.Sprintf("A set of channel IDs and whether "+
		"they're activated or not. If you don't set at least one alert "+
		"channel subscription for your %s, we won't be able to alert "+
		"you even if it starts failing.",
		name,
	)
	if options.Subject == subjectCheckGroup {
		description = "A set of channel IDs and whether they're activated or not. If you don't set at least " +
			"one alert channel subscription for your group, we won't be able to alert you even if a check or " +
			"monitor in the group starts failing."
	}

	return schema.SetNestedAttribute{
		Optional:    true,
		Computed:    true,
		Default:     emptySetDefault(types.ObjectType{AttrTypes: alertChannelSubscriptionAttrTypes}),
		Description: description,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"channel_id": schema.Int64Attribute{
					Required:    true,
					Description: "The ID of the alert channel.",
				},
				"activated": schema.BoolAttribute{
					Required:    true,
					Description: "Whether an alert should be sent to this channel.",
				},
			},
		},
	}
}

// alertChannelSubscriptionsFromSet converts the alert_channel_subscriptions
// attribute value to its API representation. Null converts to no
// subscriptions, which clears them in Checkly.
func alertChannelSubscriptionsFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) []checkly.AlertChannelSubscription {
	res := []checkly.AlertChannelSubscription{}

	if set.IsNull() || set.IsUnknown() {
		return res
	}

	var models []AlertChannelSubscriptionModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return res
	}

	for _, model := range models {
		res = append(res, checkly.AlertChannelSubscription{
			ChannelID: model.ChannelID.ValueInt64(),
			Activated: model.Activated.ValueBool(),
		})
	}

	return res
}

// alertChannelSubscriptionsToSet converts the API representation of alert
// channel subscriptions to the alert_channel_subscriptions attribute value.
// No subscriptions convert to an empty set.
func alertChannelSubscriptionsToSet(ctx context.Context, subs []checkly.AlertChannelSubscription, diags *diag.Diagnostics) types.Set {
	models := make([]AlertChannelSubscriptionModel, len(subs))
	for i, sub := range subs {
		models[i] = AlertChannelSubscriptionModel{
			ChannelID: types.Int64Value(sub.ChannelID),
			Activated: types.BoolValue(sub.Activated),
		}
	}

	set, d := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: alertChannelSubscriptionAttrTypes}, models)
	diags.Append(d...)

	return set
}
