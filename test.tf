variable "checkly_api_key" {}

provider "checkly" {
  api_key = "${var.checkly_api_key}"
}

resource "checkly_check" "test-check2" {
  name             = "My test check 2"
  type             = "API"
  activated        = true
  should_fail      = true
  frequency        = 1
  ssl_check_domain = "example.com"
  double_check     = true

  locations = [
    "us-west-1",
    "ap-northeast-1",
    "ap-south-1",
  ]

  alert_channels {
    email {
      address = "alert@example.com"
    }
  }

  alert_settings {
    escalation_type = "RUN_BASED"

    ssl_certificates {
      enabled         = true
      alert_threshold = 30
    }
  }

  request {
    follow_redirects = true
    url              = "http://example.com/"

    query_parameters = {
      search = "foo"
    }

    headers = {
      X-Bogus = "bogus"
    }

    assertion {
      source     = "JSON_BODY"
      property   = "code"
      comparison = "HAS_VALUE"
      target     = "authentication.failed"
    }

    assertion {
      source     = "STATUS_CODE"
      comparison = "EQUALS"
      target     = "401"
    }
  }
}
