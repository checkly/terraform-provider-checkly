# The attributes stay the same; only the resource type changes.
resource "checkly_heartbeat_monitor" "example" {
  # ...
}

moved {
  from = checkly_heartbeat.example
  to   = checkly_heartbeat_monitor.example
}
