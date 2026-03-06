resource "checkly_traceroute_monitor" "example-traceroute-monitor" {
  name                      = "Example Traceroute monitor"
  activated                 = true
  frequency                 = 10
  use_global_alert_settings = true

  locations = [
    "eu-west-1"
  ]

  request {
    hostname        = "example.com"
    port            = 443
    ip_family       = "IPv4"
    max_hops        = 30
    max_unknown_hops = 15
    ptr_lookup      = true
    timeout         = 10

    assertion {
      source     = "RESPONSE_TIME"
      property   = "avg"
      comparison = "LESS_THAN"
      target     = "200"
    }
  }
}
