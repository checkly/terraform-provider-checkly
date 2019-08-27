# Checkly Terraform provider

This Terraform provider enables users to manage [Checkly](https://checklyhq.com) resources like checks.

## Using the provider



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
cp terraform-provider-checkly ~/.terraform.d/plugins/
```