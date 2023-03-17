# Simple Browser Check with EOT script
resource "checkly_check" "browser-check-1" {
  name                      = "A simple browser check"
  type                      = "BROWSER"
  activated                 = true
  should_fail               = false
  frequency                 = 2
  double_check              = true
  use_global_alert_settings = true
  locations = [
    "us-west-1"
  ]

  runtime_id = "2020.01"

  script = <<EOT
const assert = require("chai").assert;
const puppeteer = require("puppeteer");
const browser = await puppeteer.launch();
const page = await browser.newPage();
await page.goto("https://google.com/");
const title = await page.title();
assert.equal(title, "Google");
await browser.close();
EOT
}

# Simple Browser Check with string script
resource "checkly_check" "browser-check-2" {
  name                      = "Example check"
  type                      = "BROWSER"
  activated                 = true
  should_fail               = false
  frequency                 = 15
  double_check              = true
  use_global_alert_settings = true

  script = "console.log(process.env.URL)"

  locations = [
    "us-west-1",
    "us-east-1"
  ]

  environment_variable {
    key    = "URL"
    value  = "https://checklyhq.com"
    locked = false
  }
}