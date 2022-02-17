# checkly_dashboard
`checkly_dashboard` allows users to manage Checkly dashboards. Add a `checkly_dashboard` resource to your resource file.

## Example Usage

Minimal dashboard example

```terraform
resource "checkly_dashboard" "dashboard-1" {
  custom_url      = "checkly"
  custom_domain   = "status.example.com"
  logo            = "https://www.checklyhq.com/logo.png"
  header          = "Public dashboard"
  refresh_rate    = 60
  paginate        = false
  pagination_rate = 30
  hide_tags       = false
}
```

Full dashboard example (includes optional fields)

```terraform
resource "checkly_dashboard" "dashboard-1" {
  custom_url      = "checkly"
  custom_domain   = "status.example.com"
  logo            = "https://www.checklyhq.com/logo.png"
  header          = "Public dashboard"
  refresh_rate    = 60
  paginate        = false
  pagination_rate = 30
  hide_tags       = false
  width           = "FULL"
  tags = [
    "auto",
  ]
}
```

## Argument Reference
The following arguments are supported:
* `custom_url` - (Required) A subdomain name under "checklyhq.com". Needs to be unique across all users.
* `custom_domain` - (Required) A custom user domain, e.g. "status.example.com". See the docs on updating your DNS and SSL usage.
* `logo` - (Required) A URL pointing to an image file.
* `header` - (Required) A piece of text displayed at the top of your dashboard.
* `refresh_rate` - (Required) How often to refresh the dashboard in seconds. Possible values `30`, `60` and `600`.
* `paginate` - (Required) Determines of pagination is on or off.
* `pagination_rate` - (Required) How often to trigger pagination in seconds. Possible values `30`, `60` and `300`.
* `hide_tags` - (Required) Show or hide the tags on the dashboard.
* `width` - (Optional) Determines whether to use the full screen or focus in the center. Possible values `FULL` and `960PX`.
* `tags` - (Optional) A list of one or more tags that filter which checks to display on the dashboard.
