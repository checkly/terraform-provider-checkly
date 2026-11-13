package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var (
	_ resource.Resource                   = (*MaintenanceWindowsResource)(nil)
	_ resource.ResourceWithConfigure      = (*MaintenanceWindowsResource)(nil)
	_ resource.ResourceWithUpgradeState   = (*MaintenanceWindowsResource)(nil)
	_ resource.ResourceWithImportState    = (*MaintenanceWindowsResource)(nil)
	_ resource.ResourceWithMoveState      = (*MaintenanceWindowsResource)(nil)
	_ resource.ResourceWithModifyPlan     = (*MaintenanceWindowsResource)(nil)
	_ resource.ResourceWithValidateConfig = (*MaintenanceWindowsResource)(nil)
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

	deprecateAliasSchema(resp, "checkly_maintenance_windows", "checkly_maintenance_window", ", because each resource manages a single maintenance window")
}
