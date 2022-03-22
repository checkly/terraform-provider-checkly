resource "checkly_alert_channel" "webhook_ac" {
  webhook {
    name   = "webhhookalerts"
    method = "get"
    headers = {
      X-HEADER-1 = "foo"
    }
    query_parameters = {
      query1 = "bar"
    }
    template       = "tmpl"
    url            = "https://example.com/webhook"
    webhook_secret = "foo-secret"
  }
}