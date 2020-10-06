# Installing the provider and getting started

1. Make sure terraform 0.13 is installed on your system

2. Create a `versions.tf` terraform config file in your project:
```terraform
terraform {
  required_version = ">= 0.13"
  required_providers {
    checkly = {
      source = "checkly/checkly"
      version = "0.7.1"
    }
  }
}

variable "checkly_api_key" {
}

provider "checkly" {
  api_key = var.checkly_api_key
}
```

3. To use the provider with your Checkly account, you will need an API Key for the account. Go to the [Account Settings: API Keys page](https://app.checklyhq.com/account/api-keys) and click 'Create API Key'. Get your api key and add it to your env `export TF_VAR_checkly_api_key=XXXXXX`

4. Run `terraform providers`. The Checkly plugin should be listed.

```bash
terraform providers
.
└── provider[registry.terraform.io/checkly/checkly] ***
```

5. Create a tf resource config file, for example `example.tf`:
```terraform
resource "checkly_check_group" "first-group" {
  name        = "Group 1"
  activated   = true
  muted       = false
  concurrency = 3
  locations = [ "eu-west-1" ]
}
```

6. Now run `terraform init` and then `terraform plan` followed by `terraform apply`
