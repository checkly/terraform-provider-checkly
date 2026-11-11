# The attributes stay the same; only the resource type changes.
resource "checkly_maintenance_window" "example" {
  # ...
}

moved {
  from = checkly_maintenance_windows.example
  to   = checkly_maintenance_window.example
}
