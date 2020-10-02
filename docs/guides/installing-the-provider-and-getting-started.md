# Installing the provider and getting started

1. Make sure terraform 0.13 is installed on your system

2. Install checkly provider:

```bash
curl https://raw.githubusercontent.com/checkly/terraform-provider-checkly/master/install-0.13.sh | sh
```

3. Create a project directory & `cd` into it

4. Create a `versions.tf` terraform config file:
```terraform
terraform {
  required_version = ">= 0.13"
  required_providers {
    checkly = {
      source = "local/checkly/checkly"
      version = "0.6.7"
    }
  }
}

variable "checkly_api_key" {
}

provider "checkly" {
  api_key = var.checkly_api_key
}
```

5. To use the provider with your Checkly account, you will need an API Key for the account. Go to the [Account Settings: API Keys page](https://app.checklyhq.com/account/api-keys) and click 'Create API Key'. Get your api key and add it to your env `export TF_VAR_checkly_api_key=XXXXXX`

6. Run `terraform providers`. The Checkly plugin should be listed.

```bash
terraform providers
.
└── provider[local/checkly/checkly] ***
```

7. Create a tf resource config file, for example `example.tf`:
```terraform
resource "checkly_check_group" "first-group" {
  name        = "Group 1"
  activated   = true
  muted       = false
  concurrency = 3
  locations = [ "eu-west-1" ]
}
```

8. Now run `terraform init` and then `terraform plan` followed by `terraform apply`
