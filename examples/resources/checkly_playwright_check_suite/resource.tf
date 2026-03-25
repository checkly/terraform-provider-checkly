# To create a Playwright Check Suite, you first need a code bundle.
#
# The code bundle is an archive file that contains the relevant code for
# your Playwright Check Suite. The same code bundle can and should be used
# for multiple Check Suites.
#
# The code bundle must at minimum contain the following files:
# - The main package.json file
# - An appropriate lockfile for your package manager
#   (e.g. package-lock.json, pnpm-lock.yaml)
# - Any files that are needed during the installation process, such as build
#   scripts and configuration files (e.g. tsconfig.json)
# - Relevant subpackages if using workspaces
# - Your Playwright configuration file(s)
# - Your Playwright spec files
# - Any files imported by the Playwright files (incl. the full import tree)
#
# When using workspaces, the bundle must include the workspace root - the
# subpackage that contains your tests is not enough. Typically, including
# those two is enough to create a working bundle, but should the packages
# depend on other packages in the workspace, then those packages must be
# included as well (recursively - repeat the process until all relevant
# workspace packages can be resolved).
resource "checkly_playwright_code_bundle" "playwright-bundle" {
  prebuilt_archive {
    file = data.archive_file.playwright-bundle.output_path
  }
}

# You can build an archive using any method you like. To construct a bundle
# within Terraform, the simplest way is to use the archive_file Data Source
# from the archive provider.
data "archive_file" "playwright-bundle" {
  type        = "tar.gz"
  output_path = "app-bundle.tar.gz"
  source_dir  = "${path.module}/app/"
  excludes = [
    ".git",
    "node_modules",
  ]
}

# Once you have a code bundle, you are ready to define Check Suites. Here's
# our recommended usage - allow auto detection to configure the runtime for
# you based on the contents of the code bundle.
resource "checkly_playwright_check_suite" "example-playwright-check" {
  name                      = "Example Playwright check"
  activated                 = true
  frequency                 = 2
  use_global_alert_settings = true

  locations = [
    "eu-west-1"
  ]

  bundle {
    id       = checkly_playwright_code_bundle.playwright-bundle.id
    metadata = checkly_playwright_code_bundle.playwright-bundle.metadata
  }
}

# Should the auto detected runtime configuration not match your needs, any
# value can be overridden manually.
resource "checkly_playwright_check_suite" "example-playwright-check-custom" {
  name                      = "Example Playwright check with overrides"
  activated                 = true
  frequency                 = 2
  use_global_alert_settings = true

  locations = [
    "eu-west-1"
  ]

  bundle {
    id       = checkly_playwright_code_bundle.playwright-bundle.id
    metadata = checkly_playwright_code_bundle.playwright-bundle.metadata
  }

  runtime {
    working_dir = "./packages/e2e"

    steps {
      test {
        command = "npx playwright test --config \"playwright.config.ts\""
      }
    }

    playwright {
      version = "1.56.1"

      device {
        type = "chromium"
      }

      device {
        type = "firefox"
      }
    }
  }
}
