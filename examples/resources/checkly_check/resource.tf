# Basic API Check
resource "checkly_check" "example-check" {
  name                      = "Example check"
  type                      = "API"
  activated                 = true
  should_fail               = false
  frequency                 = 1
  double_check              = true
  ssl_check                 = true
  use_global_alert_settings = true

  locations = [
    "us-west-1"
  ]

  request {
    url              = "https://api.example.com/"
    follow_redirects = true
    skip_ssl         = false
    assertion {
      source     = "STATUS_CODE"
      comparison = "EQUALS"
      target     = "200"
    }
  }
}

# A more complex example using more assertions and setting alerts
resource "checkly_check" "example-check-2" {
  name                   = "Example API check 2"
  type                   = "API"
  activated              = true
  should_fail            = true
  frequency              = 1
  double_check           = true
  degraded_response_time = 5000
  max_response_time      = 10000

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
    skip_ssl         = false
    url              = "http://api.example.com/"

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

# Basic Browser  Check
resource "checkly_check" "browser-check-1" {
  name                      = "Example check"
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

  runtime_id = "2021.06"

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

# An alternative syntax for add the script is by referencing an external file
data "local_file" "browser-script" {
  filename = "${path.module}/browser-script.js"
}

resource "checkly_check" "browser-check-1" {
  name                      = "Example check"
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

  runtime_id = "2021.06"
  script = data.local_file.browser-script.content
}

## Connection checks with alert channels
resource "checkly_alert_channel" "email_ac1" {
  email {
    address = "info1@example.com"
  }
}

resource "checkly_alert_channel" "email_ac2" {
  email {
    address = "info2@example.com"
  }
}

resource "checkly_check" "example-check" {
  name                      = "Example check"
  ....

  alert_channel_subscription {
    channel_id = checkly_alert_channel.email_ac1.id
    activated  = true
  }

  alert_channel_subscription {
    channel_id = checkly_alert_channel.email_ac2.id
    activated  = true
  }

}
