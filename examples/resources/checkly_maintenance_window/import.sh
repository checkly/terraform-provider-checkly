# Import an existing maintenance window using its numeric ID.
# The ID is returned by the Checkly API at GET /v1/maintenance-windows.
terraform import checkly_maintenance_window.example 12345
