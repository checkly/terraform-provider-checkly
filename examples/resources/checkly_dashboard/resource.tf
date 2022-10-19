resource "checkly_dashboard" "dashboard_1" {
  custom_url      = "checkly"
  custom_domain   = "status.example.com"
  logo            = "https://www.checklyhq.com/logo.png"
  header          = "Public dashboard"
  refresh_rate    = 60
  paginate        = false
  pagination_rate = 30
  hide_tags       = false
  width           = "FULL"
  tags = [
    "production"
  ]
}

