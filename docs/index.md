# Checkly provider
This Terraform provider enables users to manage [Checkly](https://checklyhq.com) resources like checks, groups and snippets. 

You can find a quick [step-by-step guide](https://www.checklyhq.com/docs/integrations/terraform/) in Checkly's documentation.

## Authentication
To use the provider with your Checkly account, you will need an API Key for the account. Go to the [Account Settings: API Keys](https://app.checklyhq.com/account/api-keys) page to create a new API key or to use an existing one.

Now expose the API key as an environment variable in your shell:

`$ export TF_VAR_checkly_api_key="your-api-key"`

## Example usage

```terraform
# define a variable to hold the API Key
variable "checkly_api_key" {}

# specify the Checkly provider
terraform {
  required_providers {
    checkly = {
      source = "checkly/checkly"
      version = "0.7.1"
    }
  }
}

# pass the API Key environment variable to the provider
provider "checkly" {
  api_key = var.checkly_api_key
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

For additional documentation and examples, see the Guides and Resources sections.