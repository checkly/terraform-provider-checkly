resource "checkly_icmp_monitor" "example-icmp-monitor" {
  name                      = "Example ICMP monitor"
  activated                 = true
  frequency                 = 10
  use_global_alert_settings = true

  locations = [
    "eu-west-1"
  ]

  request {
    hostname   = "example.com"
    ip_family  = "IPv4"
    ping_count = 10

    assertion {
      source     = "LATENCY"
      property   = "avg"
      comparison = "LESS_THAN"
      target     = "200"
    }
  }
}
