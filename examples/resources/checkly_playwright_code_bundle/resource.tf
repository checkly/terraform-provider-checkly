data "archive_file" "playwright-bundle" {
  type        = "tar.gz"
  output_path = "example-playwright-bundle.tar.gz"
  source_dir  = "${path.module}/"
}

resource "checkly_playwright_code_bundle" "example-1" {
  source_archive {
    file = data.archive_file.playwright-bundle.output_path
  }
}

resource "checkly_playwright_code_bundle" "example-2" {
  source_archive {
    file = "${path.module}/existing-playwright-bundle.tar.gz"
  }
}
