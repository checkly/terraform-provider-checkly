package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var (
	_ resource.Resource                 = (*HeartbeatResource)(nil)
	_ resource.ResourceWithConfigure    = (*HeartbeatResource)(nil)
	_ resource.ResourceWithUpgradeState = (*HeartbeatResource)(nil)
	_ resource.ResourceWithImportState  = (*HeartbeatResource)(nil)
	_ resource.ResourceWithMoveState    = (*HeartbeatResource)(nil)
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

	deprecateAliasSchema(resp, "checkly_heartbeat", "checkly_heartbeat_monitor", " to better reflect its position in the Checkly product lineup")
}
