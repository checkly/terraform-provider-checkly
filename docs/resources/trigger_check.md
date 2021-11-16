# checkly_trigger_check
`checkly_trigger_check` allows users to manage Checkly trigger checks. Add a `checkly_trigger_check` resource to your resource file.

## Example Usage

Trigger check example

```terraform
resource "checkly_trigger_check" "test-trigger-check" {
   check_id      = "215"
   token         = "N0mXBTcks4IW"
}
```

## Argument Reference
The following arguments are supported:
* `check_id` - The id of the check that you want to attach the trigger to.
* `token` - The token created with the trigger, needed for trigger removal.
