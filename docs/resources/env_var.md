# checkly_env_var
`checkly_env_var` allows users to manage checkly environment variables. Add a `checkly_env_var` resource to your resource file.

## Example Usage

```terraform
resource "checkly_env_var" "example-1" {
  key   = "API_KEY"
  value = "eNIGIXCGK1I3XtD4"
}
```

## Argument Reference
The following arguments are supported:
* `key` - (Required) The key of the environment variable.
* `value` - (Required) The value of the environment variable.