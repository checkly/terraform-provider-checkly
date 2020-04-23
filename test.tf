variable "checkly_api_key" {
}

provider "checkly" {
  api_key = var.checkly_api_key
}

resource "checkly_check" "test-check1" {
  name                      = "My test check 1"
  type                      = "API"
  activated                 = true
  should_fail               = true
  frequency                 = 1
  ssl_check_domain          = "example.com"
  double_check              = true
  degraded_response_time    = 15000
  max_response_time         = 30000
  use_global_alert_settings = false

  alert_settings {
    escalation_type = "RUN_BASED"

    run_based_escalation {
      failed_run_threshold = 1
    }

    time_based_escalation {
      minutes_failing_threshold = 5
    }

    ssl_certificates {
      enabled         = true
      alert_threshold = 30
    }

    reminders {
      amount = 1
    }
  }

  locations = [
    "us-west-1",
    "ap-northeast-1",
    "ap-south-1",
  ]

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
      property   = ""
      comparison = "EQUALS"
      target     = "401"
    }

    basic_auth {
      username = ""
      password = ""
    }
  }
}

resource "checkly_check" "test-check2" {
  name                      = "My test check 2"
  type                      = "API"
  activated                 = true
  should_fail               = true
  frequency                 = 1
  ssl_check_domain          = "example.com"
  double_check              = true
  degraded_response_time    = 15000
  max_response_time         = 30000
  use_global_alert_settings = true

  locations = [
    "us-west-1",
    "ap-northeast-1",
    "ap-south-1",
  ]

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
      property   = ""
      comparison = "EQUALS"
      target     = "401"
    }

    basic_auth {
      username = ""
      password = ""
    }
  }
}
