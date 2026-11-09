resource "checkly_dns_monitor" "example-dns-monitor" {
  name                      = "Example DNS monitor"
  activated                 = true
  frequency                 = 2
  use_global_alert_settings = true

  locations = [
    "eu-west-1"
  ]

  request = {
    record_type = "A"
    query       = "welcome.checklyhq.com"

    name_server = {
      host = "1.1.1.1"
      port = 53
    }

    assertions = [{
      source     = "RESPONSE_CODE"
      comparison = "EQUALS"
      target     = "NOERROR"
    }]
  }
}

# A DNS monitor that runs from a private location. The location's agents need
# version 7.0.0 or later.
resource "checkly_dns_monitor" "example-dns-monitor-private-location" {
  name                      = "Example DNS monitor on a private location"
  activated                 = true
  frequency                 = 2
  use_global_alert_settings = true

  private_locations = [
    "my-private-location"
  ]

  request = {
    record_type = "A"
    query       = "internal.example.com"
  }
}
