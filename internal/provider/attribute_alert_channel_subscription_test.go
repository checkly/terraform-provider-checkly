package provider

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/checkly/checkly-go-sdk/v2"
)

// testCheckAlertChannelSubscription checks that the alert channel
// subscriptions at attrPath of the resource named resourceName include one
// for the alert channel resource named channelResourceName, with the expected
// activation state.
func testCheckAlertChannelSubscription(resourceName, attrPath, channelResourceName, expectedActivated string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		channelRS, ok := s.RootModule().Resources[channelResourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", channelResourceName)
		}
		wantChannelID := channelRS.Primary.Attributes["id"]

		count := rs.Primary.Attributes[attrPath+".#"]
		n, _ := strconv.Atoi(count)
		for i := range n {
			prefix := fmt.Sprintf("%s.%d", attrPath, i)
			if rs.Primary.Attributes[prefix+".channel_id"] == wantChannelID {
				got := rs.Primary.Attributes[prefix+".activated"]
				if got != expectedActivated {
					return fmt.Errorf("channel %s: expected activated=%s, got %s", wantChannelID, expectedActivated, got)
				}
				return nil
			}
		}

		return fmt.Errorf("no subscription at %s with channel_id=%s found", attrPath, wantChannelID)
	}
}

// The API reports no subscriptions as an empty array, which converts to null
// unless an empty set was configured.
func TestAlertChannelSubscriptionsToSet(t *testing.T) {
	ctx := context.Background()
	elemType := types.ObjectType{AttrTypes: alertChannelSubscriptionAttrTypes}
	empty := types.SetValueMust(elemType, nil)

	for name, c := range map[string]struct {
		subs  []checkly.AlertChannelSubscription
		prior types.Set
		want  types.Set
	}{
		"none, prior null":    {prior: types.SetNull(elemType), want: types.SetNull(elemType)},
		"none, prior unknown": {prior: types.SetUnknown(elemType), want: types.SetNull(elemType)},
		"none, prior empty":   {prior: empty, want: empty},
		"none, prior non-empty": {
			prior: alertChannelSubscriptionsSet(1, true),
			want:  empty,
		},
		"one, prior null": {
			subs:  []checkly.AlertChannelSubscription{{ChannelID: 1, Activated: true}},
			prior: types.SetNull(elemType),
			want:  alertChannelSubscriptionsSet(1, true),
		},
	} {
		t.Run(name, func(t *testing.T) {
			var diags diag.Diagnostics
			got := alertChannelSubscriptionsToSet(ctx, c.subs, c.prior, &diags)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if !got.Equal(c.want) {
				t.Errorf("got %s, want %s", got, c.want)
			}
		})
	}
}
