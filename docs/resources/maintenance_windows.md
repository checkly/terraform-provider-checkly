# checkly_maintenance_windows
`checkly_maintenance_windows` allows users to manage Checkly maintenance windows. Add a `checkly_maintenance_windows` resource to your resource file.

## Example Usage

Minimal maintenance windows example

```terraform
resource "checkly_maintenance_windows" "maintenance-1" {
  name            = "string"
  starts_at       = "2014-08-24T00:00:00.000Z"
  ends_at         = "2014-08-25T00:00:00.000Z"
  repeat_unit     = "MONTH"
  tags = [
    "string",
  ]
}
```

Full maintenance windows example (includes optional fields)

```terraform
resource "checkly_maintenance_windows" "maintenance-1" {
  name            = "string"
  starts_at       = "2014-08-24T00:00:00.000Z"
  ends_at         = "2014-08-25T00:00:00.000Z"
  repeat_unit     = "MONTH"
  repeat_ends_at  = "2014-08-24T00:00:00.000Z"
  repeat_interval = 1
  tags = [
    "string",
  ]
}
```

## Argument Reference
The following arguments are supported:
* `name` - (Required) The maintenance window name.
* `starts_at` - (Required) The start date of the maintenance window.
* `ends_at` - (Required) The end date of the maintenance window.
* `repeat_unit` - (Optional) The repeat strategy for the maintenance window.
* `repeat_ends_at` - (Required) The end date where the maintenance window should stop repeating.
* `repeat_interval` - (Optional) The repeat interval of the maintenance window from the first occurance.
* `tags` - (Required) The names of the checks and groups maintenance window should apply to.
