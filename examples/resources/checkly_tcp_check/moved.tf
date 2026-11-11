# The attributes stay the same; only the resource type changes.
resource "checkly_tcp_monitor" "example" {
  # ...
}

moved {
  from = checkly_tcp_check.example
  to   = checkly_tcp_monitor.example
}
