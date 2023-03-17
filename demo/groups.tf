# Check group definition with minimal configurations
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

# Add a command line trigger for the group and create an output for the trigger URL
resource "checkly_trigger_group" "trigger-check-group-1" {
  group_id = checkly_check_group.check-group-1.id
}

output "trigger_check-group-1-url" {
  value = checkly_trigger_group.trigger-check-group-1.url
}

#-- Check group with minimal API defaults
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

# Check group with detailed API and alert defaults
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

  environment_variable {
    key    = "API_KEY"
    value  = "wuZeo4aeQu1ia4aezuiphookagheiwoh"
    locked = true
  }

  alert_settings {
    escalation_type = "RUN_BASED"

    run_based_escalation {
      failed_run_threshold = 1
    }

    time_based_escalation {
      minutes_failing_threshold = 5
    }

    reminders {
      amount   = 2
      interval = 5
    }
  }

  local_setup_script    = "setup-test"
  local_teardown_script = "teardown-test"
}

# Adding alert channels to a group (check alert-channels.tf for the alert channels code)
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