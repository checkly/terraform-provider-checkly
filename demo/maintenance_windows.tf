# Simple Maintenance windows example
resource "checkly_maintenance_windows" "maintenance-1" {
  name      = "MW Simple"
  starts_at = "2028-08-24T00:00:00.000Z"
  ends_at   = "2028-08-25T00:00:00.000Z"
}

# Complete Maintenance windows example
resource "checkly_maintenance_windows" "maintenance-2" {
  name            = "MW Complete"
  starts_at       = "2022-05-24T00:00:00.000Z"
  ends_at         = "2022-05-25T00:00:00.000Z"
  repeat_unit     = "MONTH"
  repeat_ends_at  = "2028-08-24T00:00:00.000Z"
  repeat_interval = 1
  tags = [
    "checks",
  ]
}