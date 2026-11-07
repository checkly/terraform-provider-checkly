# Basic browser check
resource "checkly_browser_check" "example" {
  name                      = "Example browser check"
  frequency                 = 10
  use_global_alert_settings = true

  locations = [
    "us-west-1"
  ]

  runtime_id = "2025.04"

  script = <<EOT
const { expect, test } = require('@playwright/test')

test.use({ actionTimeout: 10000 })

test('visit page and take screenshot', async ({ page }) => {
    const response = await page.goto(process.env.ENVIRONMENT_URL || 'https://checklyhq.com')
    await page.screenshot({ path: 'screenshot.jpg' })
    expect(response.status(), 'should respond with correct status code').toBeLessThan(400)
})
EOT
}

# A more complex example using alert settings and a retry strategy
resource "checkly_browser_check" "example_2" {
  name         = "Example browser check 2"
  frequency    = 15
  run_parallel = true

  locations = [
    "us-west-1",
    "ap-northeast-1",
    "ap-south-1",
  ]

  alert_settings = {
    escalation_type = "RUN_BASED"

    run_based_escalation = {
      failed_run_threshold = 1
    }

    reminders = {
      amount = 1
    }
  }

  retry_strategy = {
    type                 = "FIXED"
    base_backoff_seconds = 60
    max_duration_seconds = 600
    max_retries          = 3
    same_region          = false
  }

  script = "console.log('test')"
}

# An editor-style multi-file script with dependency files
resource "checkly_browser_check" "example_3" {
  name      = "Example browser check 3"
  frequency = 10

  locations = [
    "us-west-1"
  ]

  script_path = "__checks__/example.spec.ts"
  script      = <<EOT
import { test } from '@playwright/test'
import { visitPage } from './util.ts'

test('visit page', async ({ page }) => {
    await visitPage(page, 'https://checklyhq.com')
})
EOT

  dependency {
    path    = "__checks__/util.ts"
    content = <<EOT
import { expect } from '@playwright/test'

export async function visitPage(page, url) {
    const response = await page.goto(url)
    expect(response.status()).toBeLessThan(400)
}
EOT
  }
}

# An alternative syntax for adding the script is by referencing an external file
data "local_file" "browser_script" {
  filename = "${path.module}/browser-script.js"
}

resource "checkly_browser_check" "example_4" {
  name      = "Example browser check 4"
  frequency = 10

  locations = [
    "us-west-1"
  ]

  script = data.local_file.browser_script.content
}

# Migrating an existing checkly_check of type BROWSER without recreating it
resource "checkly_browser_check" "example_5" {
  name      = "Example browser check 5"
  frequency = 10

  locations = [
    "us-west-1"
  ]

  script = "console.log('test')"
}

moved {
  from = checkly_check.example_5
  to   = checkly_browser_check.example_5
}

# A check with an intent, which guides root cause analysis and automatic repair
resource "checkly_browser_check" "example_6" {
  name      = "Example browser check 6"
  frequency = 10

  locations = [
    "us-west-1"
  ]

  intent = {
    goal = "Customers can sign in to their account."

    required_outcomes = [
      "The dashboard is shown after signing in.",
    ]

    must_preserve = [
      "The check signs in through the public login page.",
    ]
  }

  ai_auto_repair_enabled = true

  script = "console.log('test')"

  # The configured script would otherwise replace a saved repair on the next
  # apply. Changes to script in this configuration are then ignored too.
  lifecycle {
    ignore_changes = [script]
  }
}
