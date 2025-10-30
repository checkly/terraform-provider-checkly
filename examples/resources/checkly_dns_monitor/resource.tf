resource "checkly_dns_monitor" "example-dns-monitor" {
  name                      = "Example DNS monitor"
  activated                 = true
  frequency                 = 2
  use_global_alert_settings = true

  locations = [
    "eu-west-1"
  ]

  request {
    record_type = "A"
    query       = "welcome.checklyhq.com"

    name_server {
      host = "1.1.1.1"
      port = 53
    }

    assertion {
      source     = "RESPONSE_CODE"
      comparison = "EQUALS"
      target     = "NOERROR"
    }
  }
}
