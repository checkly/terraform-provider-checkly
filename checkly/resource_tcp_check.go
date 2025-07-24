package checkly

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

func resourceTCPCheck() *schema.Resource {
	monitorResource := resourceTCPMonitor()

	monitorResource.Description = "" +
		"The `checkly_tcp_check` resource has been renamed to `checkly_tcp_monitor` to better reflect its position " +
		"in the Checkly product lineup." +
		"\n\n" +
		"The old resource type will not be deprecated until the Checkly provider is updated to the Terraform Plugin " +
		"Framework, which makes it possible to easily move resources between resource types." +
		"\n\n" +
		"We recommend using the `checkly_tcp_monitor` resource type for any new resources."

	return monitorResource
}
