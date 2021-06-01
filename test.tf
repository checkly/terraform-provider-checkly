#-- This file provides example terraform resource definitions that create checkly resources

################################# API CHECKS #################################

#----------------------------EXAMPLE----------------------------#
#-- a very simple API check definition

resource "checkly_check" "api-check-1" {
  name              = "API check 1"
  type              = "API"
  frequency         = 60
  activated         = true
  muted             = true
  double_check      = true
  max_response_time = 18000
  locations = [
    "eu-central-1",
    "us-east-2",
  ]

  request {
    method = "GET"
    url    = "https://api.checklyhq.com/public-stats"

    assertion {
      comparison = "EQUALS"
      property   = ""
      source     = "STATUS_CODE"
      target     = "200"
    }
  }

  use_global_alert_settings = true

}

#----------------------------EXAMPLE----------------------------#
#-- a fully fledged API check

resource "checkly_check" "api-check-2" {
  name                   = "API check 2"
  type                   = "API"
  frequency              = 10
  activated              = true
  muted                  = true
  double_check           = true
  ssl_check              = false
  degraded_response_time = 15000
  max_response_time      = 30000
  environment_variables  = null
  locations = [
    "eu-central-1",
    "us-east-2",
    "ap-northeast-1"
  ]

  tags = ["checks", "api"]

  request {
    method           = "GET"
    url              = "https://api.checklyhq.com/public-stats"
    follow_redirects = true

    headers = {
      X-CUSTOM-1 = 1
      X-CUSTOM-2 = "foo"
    }

    query_parameters = {
      param1 = 123
      param2 = "bar"
    }

    basic_auth {
      username = "user"
      password = "pass"

      # secret data shouldn't be put as plain text here, it should be injected usinng TF variables:
      # username = var.username
      # password = var.password
      # another alternative is using git encrypted secret files
    }

    assertion {
      comparison = "EQUALS"
      property   = ""
      source     = "STATUS_CODE"
      target     = "200"
    }

    assertion {
      comparison = "EQUALS"
      property   = "cache-control"
      source     = "HEADERS"
      target     = "no-cache"
    }

    assertion {
      comparison = "GREATER_THAN"
      property   = "$.apiCheckResults"
      source     = "JSON_BODY"
      target     = "100"
    }
  }

  alert_settings {
    escalation_type = "RUN_BASED"
    reminders {
      amount   = 0
      interval = 5
    }
    run_based_escalation {
      failed_run_threshold = 1
    }
    ssl_certificates {
      alert_threshold = 30
      enabled         = true
    }
    time_based_escalation {
      minutes_failing_threshold = 5
    }
  }
}

#----------------------------EXAMPLE----------------------------#
#-- a POST API Check with json body

resource "checkly_check" "api-check-3" {
  name         = "Canonical API check 3"
  type         = "API"
  activated    = true
  double_check = true
  frequency    = 720
  locations = [
    "eu-central-1",
    "us-east-2",
  ]
  max_response_time     = 18000
  muted                 = true
  environment_variables = null

  request {
    method           = "POST"
    url              = "https://jsonplaceholder.typicode.com/posts"
    follow_redirects = true

    headers = {
      Content-type = "application/json; charset=UTF-8"
    }

    body      = "{\"message\":\"hello checkly\",\"messageId\":1}"
    body_type = "JSON"

    assertion {
      comparison = "EQUALS"
      property   = ""
      source     = "STATUS_CODE"
      target     = "201"
    }

    assertion {
      comparison = "EQUALS"
      source     = "JSON_BODY"
      property   = "$.message"
      target     = "hello checkly"
    }

    assertion {
      comparison = "EQUALS"
      source     = "JSON_BODY"
      property   = "$.messageId"
      target     = 1
    }

  }

  use_global_alert_settings = true

}

#----------------------------EXAMPLE----------------------------#
#-- an api check with empty basic_auth
resource "checkly_check" "api-check-4" {
  name                   = "api check with empty basic_auth"
  type                   = "API"
  activated              = true
  should_fail            = false
  frequency              = 1
  degraded_response_time = 3000
  max_response_time      = 6000
  tags = [
    "testing",
    "bug"
  ]

  locations = [
    "eu-central-1"
  ]

  request {
    follow_redirects = false
    url              = "https://api.checklyhq.com/public-stats"

    basic_auth {
      username = ""
      password = ""
    }

    assertion {
      source     = "STATUS_CODE"
      property   = ""
      comparison = "EQUALS"
      target     = "200"
    }
  }
}


################################# BROWSER CHECKS #################################

#----------------------------EXAMPLE----------------------------#
#-- a Browser Check which runs E2E test

resource "checkly_check" "browser-check-2" {
  name                      = "Example check"
  type                      = "BROWSER"
  activated                 = true
  should_fail               = false
  frequency                 = 15
  double_check              = true
  ssl_check                 = true
  use_global_alert_settings = true

  script = "console.log('test')"

  locations = [
    "us-west-1"
  ]
}

resource "checkly_check" "browser-check-1" {
  name                      = "A simple browser check"
  type                      = "BROWSER"
  activated                 = true
  should_fail               = false
  frequency                 = 10
  double_check              = true
  ssl_check                 = true
  use_global_alert_settings = true
  locations = [
    "us-west-1"
  ]

  script = <<EOT
const assert = require("chai").assert;
const puppeteer = require("puppeteer");

const browser = await puppeteer.launch();
const page = await browser.newPage();
await page.goto("https://google.com/");
const title = await page.title();

assert.equal(title, "Google");
await browser.close();
EOT
}


################################# CHECK GROUPS #################################
## Checkly's groups feature allows you to group together a set of related checks,
## which can also share default settings for various attributes. Here is an
## example check group:

#----------------------------EXAMPLE----------------------------#
#-- a check group definition with minimal configurations

resource "checkly_check_group" "check-group-1" {
  name        = "Check Group 1"
  activated   = true
  muted       = false
  concurrency = 3
  locations = [
    "eu-west-1",
    "eu-west-2",
  ]
}

#----------------------------EXAMPLE----------------------------#
#-- a check group with minimal API defaults

resource "checkly_check_group" "check-group-2" {
  name        = "Check Group 2 with minimal api check defaults"
  activated   = true
  muted       = false
  concurrency = 3
  locations = [
    "eu-west-1",
    "eu-west-2",
  ]
  api_check_defaults {
    url = "http://example.com/"
  }
}


#----------------------------EXAMPLE----------------------------#
#-- a check group with more defaults

resource "checkly_check_group" "check-group-3" {
  name                      = "Check Group 3 with more defaults"
  activated                 = true
  muted                     = false
  concurrency               = 3
  double_check              = true
  use_global_alert_settings = false
  locations = [
    "eu-west-1",
    "eu-west-2",
  ]

  api_check_defaults {
    url = "http://example.com/"

    headers = {
      X-Test = "fooheader"
    }

    query_parameters = {
      query = "foo-val"
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


#----------------------------EXAMPLE----------------------------#
#--- Adding a check to a check group

resource "checkly_check" "api-check-group-1_1" {
  name      = "API check 1 belonging to group 1"
  type      = "API"
  activated = true
  muted     = true
  frequency = 720
  locations = [
    "eu-central-1",
    "us-east-2",
  ]
  request {
    method           = "GET"
    url              = "https://api.checklyhq.com/public-stats"
    follow_redirects = true
  }

  group_id    = checkly_check_group.check-group-1.id
  group_order = 1 #The `group_order` attribute specifies in which order the checks will be executed: 1, 2, 3, etc.

}

resource "checkly_check" "api-check-group-1_2" {
  name      = "API check 2 belonging to group 1"
  type      = "API"
  activated = true
  muted     = true
  frequency = 720
  locations = [
    "eu-central-1",
    "us-east-2",
  ]
  request {
    method           = "GET"
    url              = "https://api.checklyhq.com/public-stats"
    follow_redirects = true
  }

  group_id    = checkly_check_group.check-group-1.id
  group_order = 2
}

################################# ALERT CHANNELS #################################

resource "checkly_alert_channel" "email_ac" {
  email {
    address = "info@example.com"
  }
  send_recovery = true
  send_failure  = false
  send_degraded = true
  ssl_expiry    = false
}

resource "checkly_alert_channel" "slack_ac" {
  slack {
    channel = "checkly_alerts"
    url     = "https://slack.com/webhookurl"
  }
  send_recovery        = true
  send_failure         = true
  send_degraded        = false
  ssl_expiry           = true
  ssl_expiry_threshold = 11
}

resource "checkly_alert_channel" "sms_ac" {
  sms {
    name   = "smsalerts"
    number = "4917512345678"
  }
}


resource "checkly_alert_channel" "opsgenie_ac" {
  opsgenie {
    name     = "opsalert"
    api_key  = "key1"
    region   = "EU"
    priority = "P1"
  }
}

resource "checkly_alert_channel" "pagerduty_ac" {
  pagerduty {
    account      = "checkly"
    service_key  = "key1"
    service_name = "pdalert"
  }
}

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

resource "checkly_check" "check-with-alert-channels" {
  name              = "check-with-alertc-1"
  type              = "API"
  frequency         = 60
  activated         = true
  muted             = true
  double_check      = true
  max_response_time = 18000
  locations         = []

  request {
    method = "GET"
    url    = "https://api.checklyhq.com/public-stats"

    assertion {
      comparison = "EQUALS"
      property   = ""
      source     = "STATUS_CODE"
      target     = "200"
    }
  }

  alert_channel_subscription {
    channel_id = checkly_alert_channel.email_ac.id
    activated  = true
  }

  alert_channel_subscription {
    channel_id = checkly_alert_channel.sms_ac.id
    activated  = true
  }
}

resource "checkly_check_group" "group-with-alert-channels" {
  name        = "group-with-alert"
  activated   = true
  muted       = false
  concurrency = 3
  locations = [
    "eu-west-1",
    "eu-west-2",
  ]

  alert_channel_subscription {
    channel_id = checkly_alert_channel.email_ac.id
    activated  = true
  }
  alert_channel_subscription {
    channel_id = checkly_alert_channel.sms_ac.id
    activated  = false
  }
}