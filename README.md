# Checkly Terraform provider

[![CircleCI](https://circleci.com/gh/tnolet/terraform-provider-checkly/tree/master.svg?style=svg)](https://circleci.com/gh/tnolet/terraform-provider-checkly/tree/master)

This Terraform provider enables users to manage [Checkly](https://checklyhq.com) resources like checks.

## Using the provider

1. Download the binary applicable for your platform from the [latest tagged release](https://github.com/tnolet/terraform-provider-checkly/releases)

```bash

curl -OL https://github.com/tnolet/terraform-provider-checkly/releases/latest/download/terraform-provider-checkly_darwin_amd64.gz
```

2. Copy the binary to your Terraform plugin folder, unzip it and rename it to just `terraform-provider-checkly`

```bash
cp terraform-provider-checkly_darwin_amd64.gz ~/.terraform.d/plugins/darwin_amd64
gunzip ~/.terraform.d/plugins/darwin_amd64/terraform-provider-checkly_darwin_amd64.gz
mv ~/.terraform.d/plugins/darwin_amd64/terraform-provider-checkly_darwin_amd64 ~/.terraform.d/plugins/darwin_amd64/terraform-provider-checkly
```

3. Run `terraform init` and then `terraform providers`. The Checkly plugin should be listed.

```bash
terraform init

Initializing provider plugins...
Terraform has been successfully initialized!

terraform providers
.
└── provider.checkly
```

If you're having issues, please check [the Hashicorp docs on installing third party plugins.](https://www.terraform.io/docs/configuration/providers.html#third-party-plugins)

### Usage examples

Add a `checkly_check` resource to your resource file

```terraform
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
go install
cp terraform-provider-checkly ~/.terraform.d/plugins/darwin_amd64
```