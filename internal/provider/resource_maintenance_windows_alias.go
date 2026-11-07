package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var (
	_ resource.Resource                = (*MaintenanceWindowsResource)(nil)
	_ resource.ResourceWithConfigure   = (*MaintenanceWindowsResource)(nil)
	_ resource.ResourceWithImportState = (*MaintenanceWindowsResource)(nil)
)

// NewMaintenanceWindowsResource returns a new checkly_maintenance_windows
// resource, a deprecated alias of checkly_maintenance_window.
func NewMaintenanceWindowsResource() resource.Resource {
	return &MaintenanceWindowsResource{}
}

// MaintenanceWindowsResource implements the checkly_maintenance_windows
// resource. It shares its entire implementation with
// MaintenanceWindowResource, differing only in type name and deprecation
// status.
type MaintenanceWindowsResource struct {
	MaintenanceWindowResource
}

func (r *MaintenanceWindowsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_maintenance_windows"
}

func (r *MaintenanceWindowsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	r.MaintenanceWindowResource.Schema(ctx, req, resp)

	resp.Schema.Description = "" +
		"The `checkly_maintenance_windows` resource has been renamed to `checkly_maintenance_window`, because " +
		"each resource manages a single maintenance window. The `checkly_maintenance_windows` resource type is " +
		"deprecated and will be removed in a future version of the provider." +
		"\n\n" +
		"The schemas of both resource types are identical, and existing `checkly_maintenance_windows` resources " +
		"can be migrated to `checkly_maintenance_window` using a `moved` block." +
		"\n\n" +
		"We recommend using the `checkly_maintenance_window` resource type for any new resources."

	resp.Schema.DeprecationMessage = "" +
		"Use the `checkly_maintenance_window` resource instead. The schemas are identical, and existing resources " +
		"can be migrated using a `moved` block."
}
