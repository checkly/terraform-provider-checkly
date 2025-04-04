resource "checkly_status_page_service" "api" {
  name = "API"
}

resource "checkly_status_page_service" "database" {
  name = "Database"
}

resource "checkly_status_page" "example" {
  name          = "Example Application"
  url           = "my-example-status-page"
  default_theme = "DARK"

  card {
    name = "Services"

    service_attachment {
      service_id = checkly_status_page_service.api.id
    }

    service_attachment {
      service_id = checkly_status_page_service.database.id
    }
  }
}
