# Basic gRPC monitor using a health check (HEALTH mode)
resource "checkly_grpc_monitor" "example-grpc-monitor" {
  name                      = "Example gRPC monitor"
  activated                 = true
  should_fail               = false
  frequency                 = 1
  use_global_alert_settings = true

  locations = [
    "us-west-1"
  ]

  request {
    host      = "grpcb.in"
    port      = 9000
    grpc_mode = "HEALTH"
    service   = "grpcbin.GRPCBin"
  }
}

# A more complex example invoking a unary method (BEHAVIOR mode) with
# metadata, assertions and an alert/retry policy
resource "checkly_grpc_monitor" "example-grpc-monitor-2" {
  name        = "Example gRPC monitor 2"
  activated   = true
  should_fail = false
  frequency   = 5

  locations = [
    "us-west-1",
    "eu-central-1",
  ]

  alert_settings {
    escalation_type = "RUN_BASED"

    run_based_escalation {
      failed_run_threshold = 1
    }

    reminders {
      amount = 1
    }
  }

  retry_strategy {
    type                 = "FIXED"
    base_backoff_seconds = 60
    max_duration_seconds = 600
    max_retries          = 3
    same_region          = false
  }

  request {
    host               = "grpcb.in"
    port               = 9000
    grpc_mode          = "BEHAVIOR"
    tls                = true
    timeout            = 30
    service_definition = "REFLECTION"
    method             = "grpcbin.GRPCBin/DummyUnary"
    message            = jsonencode({ f_string = "hello" })

    metadata {
      key   = "x-api-key"
      value = "supersecret"
    }

    assertion {
      source     = "GRPC_STATUS_CODE"
      property   = ""
      comparison = "EQUALS"
      target     = "0"
    }

    assertion {
      source     = "RESPONSE_TIME"
      property   = ""
      comparison = "LESS_THAN"
      target     = "1000"
    }
  }
}
