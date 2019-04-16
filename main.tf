provider "uptimerobot" {
  # api_key = "YOUR UPTIME ROBOT API KEY"
}

resource "uptimerobot_monitor" "test-monitor" {
  friendly_name = "My test monitor"
  url           = "http://bitfieldconsulting.com/"
  type          = "HTTP"
  alert_contact = ["2416450"]
}
