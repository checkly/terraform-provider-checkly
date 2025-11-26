data "archive_file" "playwright-bundle" {
  type        = "tar.gz"
  output_path = "test-bundle.tar.gz"
  source_dir  = "${path.module}/"
}

resource "checkly_playwright_code_bundle" "playwright-bundle" {
  source_archive {
    file = data.archive_file.playwright-bundle.output_path
  }
}

resource "checkly_playwright_check_suite" "example-playwright-check" {
  name                      = "Example Playwright check"
  activated                 = true
  frequency                 = 2
  use_global_alert_settings = true

  locations = [
    "eu-west-1"
  ]

  bundle {
    id   = checkly_playwright_code_bundle.playwright-bundle.id
    data = checkly_playwright_code_bundle.playwright-bundle.data
  }

  runtime {
    steps {
      install {
        command = "pnpm i"
      }

      test {
        command = "pnpm playwright test"
      }
    }

    playwright {
      version = "1.56.1"

      device {
        type = "chromium"
      }
    }
  }
}
