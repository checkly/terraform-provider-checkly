provider "checkly" {
  # api_key = "YOUR CHECKLY API KEY"
}

resource "checkly_check" "test-check2" {
  name             = "My test check 2"
  url              = "http://example.com/"
  type             = "BROWSER"
  activated        = true
  should_fail      = true
  follow_redirects = true
  frequency        = 1

  locations = [
    "us-west-1",
    "ap-northeast-1",
    "ap-south-1",
  ]

  # assertion {
  #   source     = "JSON_BODY"
  #   property   = "code"
  #   comparison = "HAS_VALUE"
  #   target     = "authentication.failed"
  # }

  # assertion {
  #   source     = "STATUS_CODE"
  #   comparison = "EQUALS"
  #   target     = "401"
  # }
}
