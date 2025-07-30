resource "checkly_heartbeat_monitor" "example-heartbeat-monitor" {
  name      = "Example heartbeat monitor"
  activated = true
  heartbeat {
    period      = 7
    period_unit = "days"
    grace       = 1
    grace_unit  = "days"
  }
  use_global_alert_settings = true
}
