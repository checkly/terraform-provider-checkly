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

  group_id    = checkly_check_group.test-group1.id
  group_order = 1
}


resource "checkly_check" "test-check2" {
  name                   = "My test check 2"
  type                   = "API"
  activated              = true
  should_fail            = true
  frequency              = 1
  double_check           = true
  degraded_response_time = 15000
  max_response_time      = 30000

  locations = [
    "us-west-1",
    "ap-northeast-1",
    "ap-south-1",
  ]

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
  group_id    = checkly_check_group.test-group1.id
  group_order = 2
}

resource "checkly_check_group" "test-group1" {
  name      = "My test group 1"
  activated = true
  muted     = false
  tags = [
    "auto"
  ]

  locations = [
    "eu-west-1",
  ]
  concurrency = 3
  api_check_defaults {
    url = "http://example.com/"
    headers = {
      X-Test = "foo"
    }

    query_parameters = {
      query = "foo"
    }

    assertion {
      source     = "STATUS_CODE"
      property   = ""
      comparison = "EQUALS"
      target     = "200"
    }

    basic_auth {
      username = "user"
      password = "pass"
    }
  }
  environment_variables = {
    ENVTEST = "Hello world"
  }
  double_check              = true
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
      amount   = 2
      interval = 5
    }
  }
  local_setup_script    = "setup-test"
  local_teardown_script = "teardown-test"
}


# https://github.com/checkly/terraform-provider-checkly/issues/15
resource "checkly_check_group" "no-api-check-defaults" {
  name = "no-api-check-defaults"
  activated = true
  muted = false
  concurrency = 3
  locations = [
    "eu-central-1",
    "eu-west-1",
    "eu-west-2",
  ]
}

resource "checkly_check_group" "api-check-default-no-basicAuthHeaders" {
  name = "api-check-default-no-basicAuthHeaders"
  activated = true
  muted = false

  concurrency = 3
  locations = [
    "eu-central-1",
    "eu-west-1",
    "eu-west-2",
  ]
  api_check_defaults {
    url = "http://example.com/"

  }
}
