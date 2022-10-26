# Define a variable to hold the API Key and Account ID
variable "checkly_api_key" {}
variable "checkly_account_id" {}

# Specify the Checkly provider
terraform {
  required_providers {
    checkly = {
      source = "checkly/checkly"
      version = "1.4.3"
    }
  }
}

# Pass the API Key environment variable to the provider
provider "checkly" {
  api_key = var.checkly_api_key
  account_id = var.checkly_account_id
}

# Create your first API Check
resource "checkly_check" "example_check" {
  name                      = "Example API check"
  type                      = "API"
  activated                 = true
  should_fail               = false
  frequency                 = 10
  double_check              = true
  use_global_alert_settings = true

  locations = [
    "us-west-1"
  ]

  request {
    url              = "https://api.example.com/"
    follow_redirects = true
    assertion {
      source     = "STATUS_CODE"
      comparison = "EQUALS"
      target     = "200"
    }
  }
}