resource "checkly_snippet" "example-1" {
  name   = "Example 1"
  script   = "console.log('test');"
}

# An alternative way to use multi-line script.
resource "checkly_snippet" "example-2" {
  name   = "Example 2"
  script   = <<EOT
    console.log('test1');
    console.log('test2');
EOT
}

# Using local file provider
# data "local_file" "snippet-script" {
#   filename = "./snippet.js"
# }

# resource "checkly_snippet" "example-3" {
#   name = "Example 3"
#   script = data.local_file.snippet-script.content
# }
