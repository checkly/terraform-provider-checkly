# Simple public dashboard example
resource "checkly_dashboard" "dashboard-1" {
  custom_url = "tf-demo-status"
  tags       = ["checks"]
}

# Complete public dashboard example
resource "checkly_dashboard" "dashboard-2" {
  custom_url      = "tf-demo-status-complete"
  logo            = "https://www.checklyhq.com/images/text_racoon_logo.svg"
  favicon         = "https://www.checklyhq.com/images/text_racoon_logo.svg"
  link            = "https://www.checklyhq.com"
  description     = "This is a demo dashboard"
  header          = "TF Demo Status"
  refresh_rate    = 60
  paginate        = false
  pagination_rate = 30
  hide_tags       = false
  width           = "960PX"
  tags            = ["checks"]
}
