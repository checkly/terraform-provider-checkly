package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var (
	_ resource.Resource                     = (*TCPCheckResource)(nil)
	_ resource.ResourceWithConfigure        = (*TCPCheckResource)(nil)
	_ resource.ResourceWithUpgradeState     = (*TCPCheckResource)(nil)
	_ resource.ResourceWithConfigValidators = (*TCPCheckResource)(nil)
	_ resource.ResourceWithImportState      = (*TCPCheckResource)(nil)
	_ resource.ResourceWithMoveState        = (*TCPCheckResource)(nil)
	_ resource.ResourceWithModifyPlan       = (*TCPCheckResource)(nil)
)

// NewTCPCheckResource returns a new checkly_tcp_check resource, a deprecated
// alias of checkly_tcp_monitor.
func NewTCPCheckResource() resource.Resource {
	return &TCPCheckResource{}
}

// TCPCheckResource implements the checkly_tcp_check resource. It shares its
// entire implementation with TCPMonitorResource, differing only in type name
// and deprecation status.
type TCPCheckResource struct {
	TCPMonitorResource
}

func (r *TCPCheckResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tcp_check"
}

func (r *TCPCheckResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	r.TCPMonitorResource.Schema(ctx, req, resp)

	deprecateAliasSchema(resp, "checkly_tcp_check", "checkly_tcp_monitor", " to better reflect its position in the Checkly product lineup")
}
