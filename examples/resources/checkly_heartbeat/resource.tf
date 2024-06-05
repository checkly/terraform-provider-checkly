resource "checkly_heartbeat" "example-heartbeat" {
  name                      = "Example heartbeat"
  activated                 = true
  heartbeat {
    period                  = 7
    period_unit             = "days"
    grace                   = 1
    grace_unit              = "days"
  }
  use_global_alert_settings = true
}