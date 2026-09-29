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
  timezone    = "America/New_York"
  description = "Monthly database maintenance"
  silence_alerts_tags = [
    "production"
  ]
}

# Show a maintenance window on a status page
resource "checkly_status_page_service" "api" {
  name = "API"
}

resource "checkly_status_page" "example" {
  name = "Example Application"
  url  = "my-example-status-page"

  card {
    name = "Services"

    service_attachment {
      service_id = checkly_status_page_service.api.id
    }
  }
}

resource "checkly_maintenance_windows" "maintenance-2" {
  name      = "Status page maintenance"
  starts_at = "2028-08-24T00:00:00.000Z"
  ends_at   = "2028-08-24T02:00:00.000Z"
  tags = [
    "api"
  ]
  description = "We're upgrading our API servers."

  status_page_visibility {
    enabled         = true
    severity        = "MINOR"
    notify_on_start = true
    status_page_ids = [checkly_status_page.example.id]
    service_ids     = [checkly_status_page_service.api.id]
  }
}
