---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "checkly_status_page Resource - terraform-provider-checkly"
subcategory: ""
description: |-
  Checkly status pages allow you to easily communicate the uptime and health of your applications and services to your customers.
---

# checkly_status_page (Resource)

Checkly status pages allow you to easily communicate the uptime and health of your applications and services to your customers.

## Example Usage

```terraform
resource "checkly_status_page_service" "api" {
  name = "API"
}

resource "checkly_status_page_service" "database" {
  name = "Database"
}

resource "checkly_status_page" "example" {
  name          = "Example Application"
  url           = "my-example-status-page"
  default_theme = "DARK"

  card {
    name = "Services"

    service_attachment {
      service_id = checkly_status_page_service.api.id
    }

    service_attachment {
      service_id = checkly_status_page_service.database.id
    }
  }
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `card` (Block List, Min: 1) A list of cards to include on the status page. (see [below for nested schema](#nestedblock--card))
- `name` (String) The name of the status page.
- `url` (String) The URL of the status page.

### Optional

- `custom_domain` (String) A custom user domain, e.g. "status.example.com". See the docs on updating your DNS and SSL usage.
- `default_theme` (String) Possible values are `AUTO`, `DARK`, and `LIGHT`. (Default `AUTO`).
- `favicon` (String) A URL to an image file to use as the favicon of the status page.
- `logo` (String) A URL to an image file to use as the logo for the status page.
- `redirect_to` (String) The URL the user should be redirected to when clicking the logo.

### Read-Only

- `id` (String) The ID of this resource.

<a id="nestedblock--card"></a>
### Nested Schema for `card`

Required:

- `name` (String) The name of the card.
- `service_attachment` (Block List, Min: 1) A list of services to attach to the card. (see [below for nested schema](#nestedblock--card--service_attachment))

<a id="nestedblock--card--service_attachment"></a>
### Nested Schema for `card.service_attachment`

Required:

- `service_id` (String) The ID of the service.
