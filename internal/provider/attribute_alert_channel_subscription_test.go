package provider

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testCheckAlertChannelSubscription checks that the alert channel
// subscriptions at blockPath of the resource named resourceName include one
// for the alert channel resource named channelResourceName, with the expected
// activation state.
func testCheckAlertChannelSubscription(resourceName, blockPath, channelResourceName, expectedActivated string) resource.TestCheckFunc {
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

		count := rs.Primary.Attributes[blockPath+".#"]
		n, _ := strconv.Atoi(count)
		for i := range n {
			prefix := fmt.Sprintf("%s.%d", blockPath, i)
			if rs.Primary.Attributes[prefix+".channel_id"] == wantChannelID {
				got := rs.Primary.Attributes[prefix+".activated"]
				if got != expectedActivated {
					return fmt.Errorf("channel %s: expected activated=%s, got %s", wantChannelID, expectedActivated, got)
				}
				return nil
			}
		}

		return fmt.Errorf("no subscription at %s with channel_id=%s found", blockPath, wantChannelID)
	}
}
