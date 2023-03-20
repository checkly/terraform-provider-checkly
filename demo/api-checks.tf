# Simple API check
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

# Add a command line trigger for the check and create an output for the trigger URL
resource "checkly_trigger_check" "trigger-api-check-1" {
  check_id = checkly_check.api-check-1.id
}

output "trigger_api_check-1-url" {
  value = checkly_trigger_check.trigger-api-check-1.url
}

# Fully fledged API check
resource "checkly_check" "api-check-2" {
  name                   = "API check 2"
  type                   = "API"
  frequency              = 120
  activated              = true
  muted                  = true
  double_check           = true
  degraded_response_time = 15000
  max_response_time      = 30000
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
    skip_ssl         = false

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
    time_based_escalation {
      minutes_failing_threshold = 5
    }
  }
}

#  POST API Check with json body
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
  max_response_time = 18000
  muted             = true

  request {
    method           = "POST"
    url              = "https://jsonplaceholder.typicode.com/posts"
    follow_redirects = true
    skip_ssl         = false

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

# API check with empty basic_auth
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
    skip_ssl         = true
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

# Adding checks to a check group (check groups.tf for the groups code)
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
    skip_ssl         = false
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
    skip_ssl         = false
  }

  group_id    = checkly_check_group.check-group-1.id
  group_order = 2
}

# Adding alert channels to a check (check alert-channels.tf for the alert channels code)
resource "checkly_check" "check-with-alert-channels" {
  name              = "check-with-alertc-1"
  type              = "API"
  frequency         = 60
  activated         = true
  muted             = true
  double_check      = true
  max_response_time = 18000
  locations         = ["us-east-1"]

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
