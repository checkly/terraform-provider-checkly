resource "checkly_maintenance_windows" "maintenance-1" {
  name            = "Maintenance Windows"
  starts_at       = "2014-08-24T00:00:00.000Z"
  ends_at         = "2014-08-25T00:00:00.000Z"
  repeat_unit     = "MONTH"
  repeat_ends_at  = "2014-08-24T00:00:00.000Z"
  repeat_interval = 1
  tags = [
    "production"
  ]
}
