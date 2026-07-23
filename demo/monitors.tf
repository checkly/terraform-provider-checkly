# gRPC monitor — invoke a unary method in BEHAVIOR mode
resource "checkly_grpc_monitor" "grpc-monitor-1" {
  name        = "gRPC monitor 1"
  activated   = true
  muted       = true
  frequency   = 5
  should_fail = false
  locations = [
    "eu-central-1",
    "us-east-2",
  ]

  request {
    host               = "grpcb.in"
    port               = 9000
    grpc_mode          = "BEHAVIOR"
    tls                = true
    service_definition = "REFLECTION"
    method             = "grpcbin.GRPCBin/DummyUnary"
    message            = jsonencode({ f_string = "hello" })

    assertion {
      source     = "GRPC_STATUS_CODE"
      property   = ""
      comparison = "EQUALS"
      target     = "0"
    }
  }

  use_global_alert_settings = true
}

# Traceroute monitor — trace the network path to a host
resource "checkly_traceroute_monitor" "traceroute-monitor-1" {
  name        = "Traceroute monitor 1"
  activated   = true
  muted       = true
  frequency   = 5
  should_fail = false
  locations = [
    "eu-central-1",
    "us-east-2",
  ]

  request {
    url      = "api.checklyhq.com"
    protocol = "ICMP"
    max_hops = 30

    assertion {
      source     = "PACKET_LOSS"
      property   = ""
      comparison = "LESS_THAN"
      target     = "10"
    }
  }

  use_global_alert_settings = true
}

# SSL monitor — watch a certificate's validity and expiry
resource "checkly_ssl_monitor" "ssl-monitor-1" {
  name        = "SSL monitor 1"
  activated   = true
  muted       = true
  frequency   = 60
  should_fail = false
  locations = [
    "eu-central-1",
    "us-east-2",
  ]

  request {
    hostname                 = "api.checklyhq.com"
    port                     = 443
    alert_days_before_expiry = 30

    assertion {
      source     = "CERTIFICATE"
      property   = "daysUntilExpiry"
      comparison = "GREATER_THAN"
      target     = "14"
    }
  }

  use_global_alert_settings = true
}
