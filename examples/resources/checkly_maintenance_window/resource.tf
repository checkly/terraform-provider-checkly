resource "checkly_maintenance_window" "maintenance-1" {
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

resource "checkly_maintenance_window" "maintenance-2" {
  name      = "Status page maintenance"
  starts_at = "2028-08-24T00:00:00.000Z"
  ends_at   = "2028-08-24T02:00:00.000Z"
  tags = [
    "api"
  ]
  description = "We're upgrading our API servers."

  status_page_visibility = {
    enabled         = true
    severity        = "MINOR"
    notify_on_start = true
    status_page_ids = [checkly_status_page.example.id]
    service_ids     = [checkly_status_page_service.api.id]
  }
}

# Show a maintenance window on a v3 status page, with the status each
# affected component shows while the maintenance is in progress
resource "checkly_status_page_v3" "example" {
  name = "Example Application"
  url  = "my-example-status-page-v3"
}

resource "checkly_status_page_v3_component" "api" {
  status_page_id = checkly_status_page_v3.example.id
  name           = "API"
  display_order  = 0
}

resource "checkly_status_page_v3_component" "database" {
  status_page_id = checkly_status_page_v3.example.id
  name           = "Database"
  display_order  = 1
}

resource "checkly_maintenance_window" "maintenance-3" {
  name        = "Database upgrade"
  starts_at   = "2028-09-24T00:00:00.000Z"
  ends_at     = "2028-09-24T02:00:00.000Z"
  description = "We're upgrading our database servers."

  status_page_visibility = {
    enabled         = true
    severity        = "MEDIUM"
    status_page_ids = [checkly_status_page_v3.example.id]

    component_configs = [{
      status_page_id = checkly_status_page_v3.example.id
      components = [
        {
          component_id = checkly_status_page_v3_component.database.id
        },
        {
          component_id = checkly_status_page_v3_component.api.id
          status       = "DEGRADED_PERFORMANCE"
        },
      ]
    }]
  }
}
