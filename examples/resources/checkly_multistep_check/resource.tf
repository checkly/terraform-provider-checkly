# Basic multi-step check
resource "checkly_multistep_check" "example" {
  name                      = "Example multi-step check"
  frequency                 = 10
  use_global_alert_settings = true

  locations = [
    "us-west-1"
  ]

  runtime_id = "2025.04"

  script = <<EOT
import { test, expect } from '@playwright/test'

const baseUrl = process.env.ENVIRONMENT_URL || 'https://api.checklyhq.com'

test('public stats', async ({ request }) => {
  const stats = await test.step('get stats', async () => {
    const response = await request.get(`$${baseUrl}/public-stats`)
    expect(response).toBeOK()
    return response.json()
  })

  await test.step('verify stats', async () => {
    expect(stats.apiCheckResults).toBeGreaterThan(0)
  })
})
EOT
}

# An editor-style multi-file script with dependency files
resource "checkly_multistep_check" "example_2" {
  name      = "Example multi-step check 2"
  frequency = 10

  locations = [
    "us-west-1"
  ]

  runtime_id = "2025.04"

  script_path = "__checks__/example.spec.ts"
  script      = <<EOT
import { test } from '@playwright/test'
import { getStats } from './util.ts'

test('public stats', async ({ request }) => {
    await getStats(request)
})
EOT

  dependencies = [
    {
      path    = "__checks__/util.ts"
      content = <<EOT
import { expect } from '@playwright/test'

export async function getStats(request) {
    const response = await request.get('https://api.checklyhq.com/public-stats')
    expect(response).toBeOK()
    return response.json()
}
EOT
    },
  ]
}

# Migrating an existing checkly_check of type MULTI_STEP without recreating it
resource "checkly_multistep_check" "example_3" {
  name      = "Example multi-step check 3"
  frequency = 10

  locations = [
    "us-west-1"
  ]

  runtime_id = "2025.04"
  script     = "console.log('test')"
}

moved {
  from = checkly_check.example_3
  to   = checkly_multistep_check.example_3
}
