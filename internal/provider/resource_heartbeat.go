package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var (
	_ resource.Resource                = (*HeartbeatResource)(nil)
	_ resource.ResourceWithConfigure   = (*HeartbeatResource)(nil)
	_ resource.ResourceWithImportState = (*HeartbeatResource)(nil)
)

// NewHeartbeatResource returns a new checkly_heartbeat resource, a deprecated
// alias of checkly_heartbeat_monitor.
func NewHeartbeatResource() resource.Resource {
	return &HeartbeatResource{}
}

// HeartbeatResource implements the checkly_heartbeat resource. It shares its
// entire implementation with HeartbeatMonitorResource, differing only in type
// name and deprecation status.
type HeartbeatResource struct {
	HeartbeatMonitorResource
}

func (r *HeartbeatResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_heartbeat"
}

func (r *HeartbeatResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	r.HeartbeatMonitorResource.Schema(ctx, req, resp)

	resp.Schema.Description = "" +
		"The `checkly_heartbeat` resource has been renamed to `checkly_heartbeat_monitor` to better reflect its " +
		"position in the Checkly product lineup. The `checkly_heartbeat` resource type is deprecated and will be " +
		"removed in a future version of the provider." +
		"\n\n" +
		"The schemas of both resource types are identical, and existing `checkly_heartbeat` resources can be " +
		"migrated to `checkly_heartbeat_monitor` using a `moved` block." +
		"\n\n" +
		"We recommend using the `checkly_heartbeat_monitor` resource type for any new resources."

	resp.Schema.DeprecationMessage = "" +
		"Use the `checkly_heartbeat_monitor` resource instead. The schemas are identical, and existing resources " +
		"can be migrated using a `moved` block."
}
