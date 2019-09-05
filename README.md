# Checkly Terraform provider

[![CircleCI](https://circleci.com/gh/bitfield/terraform-provider-checkly/tree/master.svg?style=svg)](https://circleci.com/gh/bitfield/terraform-provider-checkly/tree/master)

This Terraform provider enables users to manage [Checkly](https://checklyhq.com) resources like checks.

## Using the provider

1. Download the binary applicable for your platform from the [latest tagged release](https://github.com/bitfield/terraform-provider-checkly/releases). 
Then copy the binary to your Terraform plugin folder, unzip it and rename it to just `terraform-provider-checkly`. Lastly, set the correct access rights.


I you're on MacOS, just run the script `install.sh` script

```bash
curl https://raw.githubusercontent.com/bitfield/terraform-provider-checkly/master/install.sh | sh
```


2. Run `terraform init` and then `terraform providers`. The Checkly plugin should be listed.

```bash
terraform init

Initializing provider plugins...
Terraform has been successfully initialized!

terraform providers
.
└── provider.checkly
```

If you're having issues, please check [the Hashicorp docs on installing third party plugins.](https://www.terraform.io/docs/configuration/providers.html#third-party-plugins)

### Authentication

To use the provider with your Checkly account, you will need an API Key for the account. Go to the [Account Settings: API Keys page](https://app.checklyhq.com/account/api-keys) and click 'Create API Key'.

Now expose the API key as an environment variable in your shell:

```bash
export TF_VAR_checkly_api_key=<my_api_key>
```

### Usage examples

Add a `checkly_check` resource to your resource file. 

This first example is a very minimal API check.

```terraform
variable "checkly_api_key" {}

provider "checkly" {
  api_key = "${var.checkly_api_key}"
}

resource "checkly_check" "checkly-public-stats" {
  name                      = "public-stats"
  type                      = "API"
  activated                 = true
  should_fail               = true
  frequency                 = 1
  double_check              = true
  ssl_check                 = true
  ssl_check_domain          = "api.checklyhq.com"
  use_global_alert_settings = true

  locations = [
    "us-west-1"
  ]

  request {
    url              = "https://api.checklyhq.com/public-stats"
    follow_redirects = true
    assertion {
      source     = "STATUS_CODE"
      comparison = "EQUALS"
      target     = "200"
    }
  }
}
```

## Developing the provider

Clone the repo, build the project and add it to the Terraform plugins directory. You will need to have Go installed.

```bash
git clone git@github.com:bitfield/terraform-provider-checkly.git
cd terraform-provider-checkly
go build
```