resource "checkly_check_group" "test_group1" {
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

    assertion {
      source     = "TEXT_BODY"
      property   = ""
      comparison = "CONTAINS"
      target     = "welcome"
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

    reminders {
      amount   = 2
      interval = 5
    }
  }
  local_setup_script    = "setup-test"
  local_teardown_script = "teardown-test"
}

# Add a check to a group
resource "checkly_check" "test_check1" {
  name                      = "My test check 1"

  group_id    = checkly_check_group.test_group1.id
  group_order = 1
}


# Using with alert channels
resource "checkly_alert_channel" "email_ac1" {
  email {
    address = "info@example.com"
  }
}

resource "checkly_alert_channel" "email_ac2" {
  email {
    address = "info2@example.com"
  }
}


# Connect the check group to the alert channels
resource "checkly_check_group" "test_group1" {
  name      = "My test group 1"

  alert_channel_subscription {
    channel_id = checkly_alert_channel.email_ac1.id
    activated  = true
  }

  alert_channel_subscription {
    channel_id = checkly_alert_channel.email_ac2.id
    activated  = true
  }
}

