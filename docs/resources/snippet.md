# checkly_snippet
`checkly_snippet` allows users to manage Checkly snippets. Add a `checkly_snippet` resource to your resource file.

## Example Usage

```terraform
resource "checkly_snippet" "example-1" {
  name   = "Example 1"
  script   = "console.log('test');"
}
```

An alternative way to use multi-line script.

```terraform
resource "checkly_snippet" "example-2" {
  name   = "Example 2"
  script   = <<EOT
    console.log('test1');
    console.log('test2');
EOT
}
```

Or using terraform local file provider

```terraform
data "local_file" "snippet-script" {
  filename = "./snippet.js"
}

resource "checkly_snippet" "example-3" {
  name = "Example 3"
  script = data.local_file.snippet-script.content
}
```

## Argument Reference
The following arguments are supported:
* `name` - (Required) The name of the snippet.
* `script` - (Required) Your Node.js code that interacts with the API check lifecycle, or functions as a partial for browser checks.