
# Minimal example: just a folder to organize checks.
resource "checkly_check_group_v2" "just-a-folder" {
  name = "Just a Folder"
}

# Full example: a group with enforced settings that override individual checks.
resource "checkly_check_group_v2" "production-api" {
  name        = "Production API"
  activated   = true
  muted       = false
  concurrency = 5
  tags        = ["production", "api"]

  # Enforce locations: all checks in this group will run in these locations,
  # regardless of what's configured on the individual check.
  enforce_locations {
    enabled           = true
    locations         = ["us-east-1", "eu-west-1", "ap-southeast-1"]
    private_locations = ["my-datacenter"]
  }

  # Enforce retry strategy: all checks in this group will use this retry
  # strategy, overriding any retry settings on the individual check.
  enforce_retry_strategy {
    enabled = true

    retry_strategy {
      type                 = "LINEAR"
      base_backoff_seconds = 30
      max_retries          = 3
      max_duration_seconds = 300
      same_region          = true
    }
  }

  # Enforce scheduling strategy: all checks run in parallel across locations.
  enforce_scheduling_strategy {
    enabled      = true
    run_parallel = true
  }

  # Enforce alert settings: controls when alerts fire for all checks in the group.
  enforce_alert_settings {
    enabled = true

    alert_settings {
      escalation_type = "RUN_BASED"

      run_based_escalation {
        failed_run_threshold = 2
      }

      reminders {
        amount   = 3
        interval = 10
      }

      parallel_run_failure_threshold {
        enabled    = true
        percentage = 50
      }
    }

    # Subscribe alert channels to this group.
    alert_channel_subscription {
      channel_id = 1234
      activated  = true
    }
  }

  # Environment variables available to all checks in the group.
  environment_variable {
    key    = "BASE_URL"
    value  = "https://api.example.com"
    locked = false
  }

  environment_variable {
    key    = "API_TOKEN"
    value  = "super-secret-token"
    locked = true
    secret = true
  }

  # Default runtime for checks that don't specify their own.
  default_runtime {
    runtime_id = "2024.02"
  }

  # API check defaults: shared base URL, headers, and assertions.
  api_check_defaults {
    url = "https://api.example.com/"

    headers = {
      "X-Api-Key"    = "{{API_TOKEN}}"
      "Content-Type" = "application/json"
    }

    query_parameters = {
      "version" = "v2"
    }

    assertion {
      source     = "STATUS_CODE"
      comparison = "LESS_THAN"
      target     = "400"
    }

    assertion {
      source     = "RESPONSE_TIME"
      comparison = "LESS_THAN"
      target     = "3000"
    }

    basic_auth {
      username = "admin"
      password = "pass"
    }
  }

  # Setup and teardown scripts run in addition to each check's own scripts.
  setup_script {
    inline_script = <<-EOF
      // Runs before every API check in this group
      console.log("Setting up...")
    EOF
  }

  teardown_script {
    inline_script = <<-EOF
      // Runs after every API check in this group
      console.log("Tearing down...")
    EOF
  }

}

# Example: enforce only retry strategy with network-error-only retries.
resource "checkly_check_group_v2" "retry-on-network-errors" {
  name = "Retry on Network Errors Only"

  enforce_retry_strategy {
    enabled = true

    retry_strategy {
      type                 = "EXPONENTIAL"
      base_backoff_seconds = 10
      max_retries          = 5
      max_duration_seconds = 600
      same_region          = false

      # Only retry when the failure is a network error, not an assertion failure.
      only_on {
        network_error = true
      }
    }
  }
}

# Example: enforce time-based alert escalation.
resource "checkly_check_group_v2" "time-based-alerts" {
  name = "Time-Based Alert Escalation"

  enforce_alert_settings {
    enabled = true

    alert_settings {
      escalation_type = "TIME_BASED"

      time_based_escalation {
        minutes_failing_threshold = 10
      }

      reminders {
        amount   = 5
        interval = 15
      }
    }
  }
}

# Example: use global alert settings instead of custom ones.
resource "checkly_check_group_v2" "use-global-alerts" {
  name = "Use Global Alert Settings"

  enforce_alert_settings {
    enabled                   = true
    use_global_alert_settings = true
  }
}

# Example: enforce no retries.
resource "checkly_check_group_v2" "no-retries" {
  name = "No Retries Group"

  enforce_retry_strategy {
    enabled = true

    retry_strategy {
      type = "NO_RETRIES"
    }
  }
}

# Example: enforce a single retry.
resource "checkly_check_group_v2" "single-retry" {
  name = "Single Retry Group"

  enforce_retry_strategy {
    enabled = true

    retry_strategy {
      type                 = "SINGLE_RETRY"
      base_backoff_seconds = 10
      same_region          = true
    }
  }
}
