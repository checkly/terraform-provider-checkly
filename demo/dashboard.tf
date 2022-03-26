# Public dashboard example
resource "checkly_dashboard" "dashboard-1" {
  custom_url      = "testurl"
  custom_domain   = "testdomain"
  logo            = "logo"
  header          = "header"
  refresh_rate    = 60
  paginate        = false
  pagination_rate = 30
  hide_tags       = false
  width           = "FULL"
  tags = [
    "string",
  ]
}
