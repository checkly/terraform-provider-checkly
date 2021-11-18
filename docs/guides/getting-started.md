# Installing the provider and getting started

1. Create a `versions.tf` terraform config file in your project:
```terraform
terraform {
  required_providers {
    checkly = {
      source = "checkly/checkly"
      version = "1.3.0"
    }
  }
}

variable "checkly_api_key" {
}

variable "checkly_account_id" {
}

provider "checkly" {
  api_key = var.checkly_api_key
  account_id = var.checkly_account_id
}
```

2. To use the provider with your Checkly account, you will need an API Key for your Checkly user. Go to the [User Settings: API Keys page](https://app.checklyhq.com/settings/user/api-keys) and click 'Create API Key'. Get your User API Key and add it to your env:
```bash
$ export TF_VAR_checkly_api_key=cu_xxx
```

1. You also need to set your target Account ID, you can find the Checkly Account ID under your [account settings](https://app.checklyhq.com/settings/account/general). If you don't have access to account settings, please contact your account owner/admin.

```bash
$ export TF_VAR_checkly_account_id=xxx
```

> ⚠️ If you are still using legacy Account API Keys, you can skip this step. Notice that Account API keys will be deprecated soon.

1. Run `$ terraform providers` in your terminal. The Checkly plugin should be listed.

```bash
terraform providers
.
└── provider[registry.terraform.io/checkly/checkly] ***
```

5. Create a TF resource config file, for example `main.tf`:
```terraform
resource "checkly_check_group" "first-group" {
  name        = "Group 1"
  activated   = true
  muted       = false
  concurrency = 3
  locations = [ "eu-west-1" ]
}
```

5. Now run `$ terraform init` to setup a new project. Then use `$ terraform plan` followed by `$ terraform apply` to deploy your monitoring.
