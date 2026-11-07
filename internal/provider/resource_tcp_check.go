package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var (
	_ resource.Resource                     = (*TCPCheckResource)(nil)
	_ resource.ResourceWithConfigure        = (*TCPCheckResource)(nil)
	_ resource.ResourceWithConfigValidators = (*TCPCheckResource)(nil)
	_ resource.ResourceWithImportState      = (*TCPCheckResource)(nil)
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

	resp.Schema.Description = "" +
		"The `checkly_tcp_check` resource has been renamed to `checkly_tcp_monitor` to better reflect its position " +
		"in the Checkly product lineup. The `checkly_tcp_check` resource type is deprecated and will be removed in " +
		"a future version of the provider." +
		"\n\n" +
		"The schemas of both resource types are identical, and existing `checkly_tcp_check` resources can be " +
		"migrated to `checkly_tcp_monitor` using a `moved` block." +
		"\n\n" +
		"We recommend using the `checkly_tcp_monitor` resource type for any new resources."

	resp.Schema.DeprecationMessage = "" +
		"Use the `checkly_tcp_monitor` resource instead. The schemas are identical, and existing resources can be " +
		"migrated using a `moved` block."
}
