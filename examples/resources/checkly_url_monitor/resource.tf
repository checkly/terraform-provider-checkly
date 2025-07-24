resource "checkly_url_monitor" "example-url-monitor" {
  name                      = "Example URL monitor"
  activated                 = true
  frequency                 = 1
  use_global_alert_settings = true

  locations = [
    "us-west-1"
  ]

  request {
    url = "https://welcome.checklyhq.com"

    assertion {
      source     = "STATUS_CODE"
      comparison = "EQUALS"
      target     = "200"
    }
  }
}
