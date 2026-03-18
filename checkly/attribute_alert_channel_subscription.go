package checkly

import (
	"fmt"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const alertChannelSubscriptionAttributeName = "alert_channel_subscription"

type AlertChannelSubscriptionAttributeSchemaOptions struct {
	Description string
	Monitor     bool
}

func makeAlertChannelSubscriptionAttributeSchema(options AlertChannelSubscriptionAttributeSchemaOptions) *schema.Schema {
	name := "check"
	if options.Monitor {
		name = "monitor"
	}

	description := options.Description
	if description == "" {
		description = fmt.Sprintf("An array of channel IDs and whether "+
			"they're activated or not. If you don't set at least one alert "+
			"channel subscription for your %s, we won't be able to alert "+
			"you even if it starts failing.",
			name,
		)
	}

	return &schema.Schema{
		Description: description,
		Type:        schema.TypeSet,
		Optional:    true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"channel_id": {
					Description: "The ID of the alert channel.",
					Type:        schema.TypeInt,
					Required:    true,
				},
				"activated": {
					Description: "Whether an alert should be sent to this channel.",
					Type:        schema.TypeBool,
					Required:    true,
				},
			},
		},
	}
}

func alertChannelSubscriptionsFromSet(s *schema.Set) []checkly.AlertChannelSubscription {
	res := []checkly.AlertChannelSubscription{}
	for _, it := range s.List() {
		tm := it.(tfMap)
		res = append(res, checkly.AlertChannelSubscription{
			ChannelID: int64(tm["channel_id"].(int)),
			Activated: tm["activated"].(bool),
		})
	}
	return res
}

func setFromAlertChannelSubscriptions(subs []checkly.AlertChannelSubscription) []tfMap {
	res := make([]tfMap, len(subs))
	for i, sub := range subs {
		res[i] = tfMap{
			"channel_id": sub.ChannelID,
			"activated":  sub.Activated,
		}
	}

	return res
}
