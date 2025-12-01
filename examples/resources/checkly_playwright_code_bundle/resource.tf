# Construct a new bundle from source files
data "archive_file" "playwright-bundle" {
  type        = "tar.gz"
  output_path = "app-bundle.tar.gz"
  source_dir  = "${path.module}/app/"
  excludes = [
    ".git",
    "node_modules",
  ]
}

resource "checkly_playwright_code_bundle" "example-1" {
  prebuilt_archive {
    file = data.archive_file.playwright-bundle.output_path
  }
}

# Use an existing bundle archive
resource "checkly_playwright_code_bundle" "example-2" {
  prebuilt_archive {
    file = "${path.module}/existing-playwright-bundle.tar.gz"
  }
}
