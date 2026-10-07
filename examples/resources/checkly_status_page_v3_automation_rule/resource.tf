resource "checkly_status_page_v3" "example" {
  name = "Example Application"
  url  = "my-example-status-page"
}

resource "checkly_status_page_v3_component" "api" {
  status_page_id = checkly_status_page_v3.example.id
  name           = "API"
  display_order  = 0
}

# When a check tagged "api" fails, an incident is opened on the page marking
# the API component as a major outage, and resolved when the check recovers.
resource "checkly_status_page_v3_automation_rule" "api_outage" {
  status_page_id = checkly_status_page_v3.example.id
  name           = "API outage"
  first_update   = "We are investigating an issue with the API."
  last_update    = "The issue has been resolved."
  tags           = ["api"]

  component {
    component_id  = checkly_status_page_v3_component.api.id
    target_impact = "MAJOR_OUTAGE"
  }
}
