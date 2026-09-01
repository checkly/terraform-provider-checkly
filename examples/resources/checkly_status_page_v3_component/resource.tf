resource "checkly_status_page_v3" "example" {
  name = "Example Application"
  url  = "my-example-status-page"
}

resource "checkly_status_page_v3_component" "services" {
  status_page_id = checkly_status_page_v3.example.id
  type           = "GROUP"
  name           = "Services"
  display_order  = 0
}

resource "checkly_status_page_v3_component" "api" {
  status_page_id = checkly_status_page_v3.example.id
  name           = "API"
  description    = "The public API"
  display_order  = 1
  parent_id      = checkly_status_page_v3_component.services.id
}
