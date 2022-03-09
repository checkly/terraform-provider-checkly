# checkly_trigger_check
`checkly_trigger_check` allows users to manage Checkly trigger checks. Add a `checkly_trigger_check` resource to your resource file.

## Example Usage

Trigger check example

```terraform
resource "checkly_trigger_check" "test-trigger-check" {
   check_id = "c1ff95c5-d7f6-4a90-9ce2-1e605f117592"
}

output "test-trigger-check-url" {
  value = checkly_trigger_check.test-trigger-check.url
}
```

## Argument Reference
The following arguments are supported:
* `check_id` - The id of the check that you want to attach the trigger to.
