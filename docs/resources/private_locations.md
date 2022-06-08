---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "checkly_private_locations Resource - terraform-provider-checkly"
subcategory: ""
description: |-
  
---

# checkly_private_locations (Resource)



## Example Usage

```terraform
# Simple Private Location example
resource "checkly_private_locations" "location" {
  name = "New Private Location"
  slug_name = "new-private-location"
  icon = "location"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) 
- `slug_name` (String)

### Optional

- `icon` (String) The icon that will represent the private location.

