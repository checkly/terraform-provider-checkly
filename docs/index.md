# Checkly Provider
This Terraform provider enables users to manage [Checkly](https://checklyhq.com) resources like checks. 

## Supported resources
The Checkly provider provides the following resources to interact with [Checklyhq.com](https://checklyhq.com), we'll be adding more soon. 
- [x] Checks
- [x] Check groups
- [ ] Alert channels
- [ ] Snippets
- [ ] Environment variables

## Authentication
To use the provider with your Checkly account, you will need an API Key for the account. Go to the [Account Settings: API Keys](https://app.checklyhq.com/account/api-keys) page to create a new API key or to use an existing one.

Now expose the API key as an environment variable in your shell:

`$ export TF_VAR_checkly_api_key="your-api-key"`

## Example Usage
```terraform
# define a variable to hold the API Key
variable "checkly_api_key" {}


provider "checkly" {
  # pass the API Key environment variable to the provider
  api_key = "${var.checkly_api_key}"
}

# define an API check
resource "checkly_check" "example-check" {
  name                      = "Example API check"
  type                      = "API"
  activated                 = true
  should_fail               = false
  frequency                 = 10
  double_check              = true
  ssl_check                 = true
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
```