provider "checkly" {
  # api_key = "YOUR CHECKLY API KEY"
}

resource "checkly_check" "test-check2" {
  name      = "My test check 2"
  url       = "http://example.com/"
  type      = "BROWSER"
  activated = true
  frequency = 5
  locations = ["eu-central-1"]
}
