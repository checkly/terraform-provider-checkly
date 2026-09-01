resource "checkly_status_page_v3" "example" {
  name          = "Example Application"
  url           = "my-example-status-page"
  default_theme = "DARK"
}

# The page structure is declared with components: GROUPs contain SERVICEs.
resource "checkly_status_page_v3_component" "services" {
  status_page_id = checkly_status_page_v3.example.id
  type           = "GROUP"
  name           = "Services"
  display_order  = 0
}

resource "checkly_status_page_v3_component" "api" {
  status_page_id = checkly_status_page_v3.example.id
  name           = "API"
  display_order  = 1
  parent_id      = checkly_status_page_v3_component.services.id
}

resource "checkly_status_page_v3_component" "database" {
  status_page_id = checkly_status_page_v3.example.id
  name           = "Database"
  display_order  = 2
  parent_id      = checkly_status_page_v3_component.services.id
}
