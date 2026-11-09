# Basic API check
resource "checkly_api_check" "example_check" {
  name                      = "Example check"
  activated                 = true
  should_fail               = false
  frequency                 = 1
  use_global_alert_settings = true

  locations = [
    "us-west-1"
  ]

  request = {
    url              = "https://api.example.com/"
    follow_redirects = true
    skip_ssl         = false
    assertion = [{
      source     = "STATUS_CODE"
      comparison = "EQUALS"
      target     = "200"
    }]
  }
}

# A more complex example using more assertions and setting alerts
resource "checkly_api_check" "example_check_2" {
  name                   = "Example API check 2"
  activated              = true
  should_fail            = true
  frequency              = 1
  degraded_response_time = 5000
  max_response_time      = 10000

  locations = [
    "us-west-1",
    "ap-northeast-1",
    "ap-south-1",
  ]

  alert_settings = {
    escalation_type = "RUN_BASED"

    run_based_escalation = {
      failed_run_threshold = 1
    }

    reminders = {
      amount = 1
    }
  }

  retry_strategy = {
    type                 = "FIXED"
    base_backoff_seconds = 60
    max_duration_seconds = 600
    max_retries          = 3
    same_region          = false
  }

  request = {
    follow_redirects = true
    skip_ssl         = false
    url              = "http://api.example.com/"

    query_parameters = {
      search = "foo"
    }

    headers = {
      X-Bogus = "bogus"
    }

    assertion = [
      {
        source     = "JSON_BODY"
        property   = "code"
        comparison = "HAS_VALUE"
        target     = "authentication.failed"
      },
      {
        source     = "STATUS_CODE"
        property   = ""
        comparison = "EQUALS"
        target     = "401"
      },
    ]

    basic_auth = {
      username = ""
      password = ""
    }
  }
}

# A high frequency API check running every 30 seconds
resource "checkly_api_check" "high_frequency_check" {
  name             = "High frequency check"
  activated        = true
  frequency        = 0
  frequency_offset = 30

  locations = [
    "us-west-1"
  ]

  request = {
    url = "https://api.example.com/health"
    assertion = [{
      source     = "STATUS_CODE"
      comparison = "EQUALS"
      target     = "200"
    }]
  }
}

# Connecting checks to alert channels
resource "checkly_alert_channel" "email_ac1" {
  email = {
    address = "info1@example.com"
  }
}

resource "checkly_alert_channel" "email_ac2" {
  email = {
    address = "info2@example.com"
  }
}

resource "checkly_api_check" "example_check_3" {
  name = "Example check"
  # ...

  request = {
    url = "https://api.example.com/"
  }

  alert_channel_subscriptions = [
    {
      channel_id = checkly_alert_channel.email_ac1.id
      activated  = true
    },
    {
      channel_id = checkly_alert_channel.email_ac2.id
      activated  = true
    },
  ]
}

# Migrating an existing checkly_check resource with type = "API" without
# recreating the underlying check
resource "checkly_api_check" "migrated_check" {
  name      = "Migrated check"
  frequency = 10

  locations = [
    "us-west-1"
  ]

  request = {
    url = "https://api.example.com/"
  }
}

moved {
  from = checkly_check.migrated_check
  to   = checkly_api_check.migrated_check
}
