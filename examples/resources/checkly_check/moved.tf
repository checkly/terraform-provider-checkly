# Each check moves to the resource of its type. Change the resource type,
# remove the attributes the new type doesn't have, and add a moved block.

# A check with type = "API".
resource "checkly_api_check" "api" {
  # ...
}

moved {
  from = checkly_check.api
  to   = checkly_api_check.api
}

# A check with type = "BROWSER".
resource "checkly_browser_check" "browser" {
  # ...
}

moved {
  from = checkly_check.browser
  to   = checkly_browser_check.browser
}

# A check with type = "MULTI_STEP".
resource "checkly_multistep_check" "multistep" {
  # ...
}

moved {
  from = checkly_check.multistep
  to   = checkly_multistep_check.multistep
}
