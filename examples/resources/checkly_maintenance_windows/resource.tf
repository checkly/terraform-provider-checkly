resource "checkly_maintenance_windows" "maintenance-1" {
  name            = "Maintenance Windows"
  starts_at       = "2014-08-24T00:00:00.000Z"
  ends_at         = "2014-08-25T00:00:00.000Z"
  repeat_unit     = "MONTH"
  repeat_ends_at  = "2014-08-24T00:00:00.000Z"
  repeat_interval = 1
  timezone        = "America/New_York"
  tags = [
    "production"
  ]

  # Silence alerts for a wider set of checks than the ones being paused.
  silence_alerts_tags = [
    "production",
    "staging"
  ]
}

# Pause and silence every check in the account, ignoring tags entirely.
resource "checkly_maintenance_windows" "maintenance-account-wide" {
  name               = "Account-wide maintenance"
  starts_at          = "2014-08-24T00:00:00.000Z"
  ends_at            = "2014-08-25T00:00:00.000Z"
  timezone           = "Europe/Berlin"
  pause_all_checks   = true
  silence_all_alerts = true
}
