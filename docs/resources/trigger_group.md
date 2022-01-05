# checkly_trigger_group
`checkly_trigger_group` allows users to manage Checkly trigger groups. Add a `checkly_trigger_group` resource to your resource file.

## Example Usage

Trigger group example

```terraform
resource "checkly_trigger_group" "test-trigger-group" {
   group_id = "215"
}

output "test-trigger-group-url" {
  value = checkly_trigger_group.test-trigger-group.url
}
```

## Argument Reference
The following arguments are supported:
* `group_id` - The id of the group that you want to attach the trigger to.
