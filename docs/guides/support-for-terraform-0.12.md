# Support for Terraform 0.12
If you're still using Terraform 0.12 and are not ready to upgrade yet, you can still still use checkly provider, please follow the following instructions to set it up:

1. Make sure terraform 0.12 is installed on your system

2. Install checkly provider

```bash
curl https://raw.githubusercontent.com/checkly/terraform-provider-checkly/main/install-0.12.sh | sh
```

2. Run `terraform providers`. The Checkly plugin should be listed.

```bash
terraform providers
.
└── provider.checkly
```

If you're having issues, please check [the Hashicorp docs on installing third party plugins.](https://www.terraform.io/docs/configuration/providers.html#third-party-plugins)


### Authentication

To use the provider with your Checkly account, you will need an API Key for your Checkly user. Go to the [User Settings: API Keys page](https://app.checklyhq.com/settings/user/api-keys) and click 'Create API Key'.

Now expose the API key as an environment variable in your shell:

```bash
export TF_VAR_checkly_api_key=<my_api_key>
export TF_VAR_checkly_account_id=<my_account_id>
```

### Usage
create a tf config file, for example, `example.tf`

```terraform
variable "checkly_api_key" {
}

variable "checkly_account_id" {
}

provider "checkly" {
  api_key = var.checkly_api_key
  account_id = var.checkly_account_id
}

resource "checkly_group" "group1" {
  name        = "Group 1"
  activated   = true
  muted       = false
  concurrency = 3
  locations = [ "eu-west-1" ]
}
```

Then you are ready to run `terraform init`, `terraform plan` and `terraform apply`.