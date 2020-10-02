# checkly_check
`checkly_check` allows users to manage checkly checks. Add a `checkly_check` resource to your resource file. 
 
## Example Usage - API checks
This first example is a very minimal **API check**.

```terraform
resource "checkly_check" "example-check" {
  name                      = "Example check"
  type                      = "API"
  activated                 = true
  should_fail               = false
  frequency                 = 1
  double_check              = true
  ssl_check                 = true
  use_global_alert_settings = true

  locations = [
    "us-west-1"
  ]

  request {
    url              = "https://api.example.com/"
    follow_redirects = true
    assertion {
      source     = "STATUS_CODE"
      comparison = "EQUALS"
      target     = "200"
    }
  }
}
```

Here's something a little more complicated, to show what you can do:

```terraform
resource "checkly_check" "example-check-2" {
  name                   = "Example API check 2"
  type                   = "API"
  activated              = true
  should_fail            = true
  frequency              = 1
  double_check           = true
  degraded_response_time = 5000
  max_response_time      = 10000

  locations = [
    "us-west-1",
    "ap-northeast-1",
    "ap-south-1",
  ]

  alert_settings {
    escalation_type = "RUN_BASED"

    run_based_escalation {
      failed_run_threshold = 1
    }

    time_based_escalation {
      minutes_failing_threshold = 5
    }

    ssl_certificates {
      enabled         = true
      alert_threshold = 30
    }

    reminders {
      amount = 1
    }
  }

  request {
    follow_redirects = true
    url              = "http://api.example.com/"

    query_parameters = {
      search = "foo"
    }

    headers = {
      X-Bogus = "bogus"
    }

    assertion {
      source     = "JSON_BODY"
      property   = "code"
      comparison = "HAS_VALUE"
      target     = "authentication.failed"
    }

    assertion {
      source     = "STATUS_CODE"
      property   = ""
      comparison = "EQUALS"
      target     = "401"
    }

    basic_auth {
      username = ""
      password = ""
    }
  }
}
```  


## Example Usage - Browser checks
A **browser** check is similar, but a bit simpler as it has less options. Notice the multi line string syntax with `EOT`. Terraform also gives you the option to insert content from external files which would be useful for larger scripts or when you want to manage your browser check scripts as separate Javascript files. See the partial examples below

```terraform
resource "checkly_check" "browser-check-1" {
  name                      = "Example check"
  type                      = "BROWSER"
  activated                 = true
  should_fail               = false
  frequency                 = 10
  double_check              = true
  ssl_check                 = true
  use_global_alert_settings = true
  locations = [
    "us-west-1"
  ]

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
```

An alternative syntax for add the script is by referencing an external file

```terraform
data "local_file" "browser-script" {
  filename = "${path.module}/browser-script.js"
}

resource "checkly_check" "browser-check-1" {
  ...
  script = data.local_file.browser-script.content
}
```

## Example Usage - With Alert channels
first define an alert channel
```terraform 
resource "checkly_alert_channel" "email_ac1" {
  email {
    address = "info1@example.com"
  }
}

resource "checkly_alert_channel" "email_ac2" {
  email {
    address = "info2@example.com"
  }
}
```

then connect the check to the alert channel
```terraform
resource "checkly_check" "example-check" {
  name                      = "Example check"
  ....

  alert_channel_subscription {
    channel_id = checkly_alert_channel.email_ac1.id
    activated  = true
  }

  alert_channel_subscription {
    channel_id = checkly_alert_channel.email_ac2.id
    activated  = true
  }

}
```
 


## Argument Reference  
The following arguments are supported:
* `name` - (Required) The name of the check.  
* `type` - (Required) The type of the check. Possible values are `API`, and `BROWSER`.  
* `frequency` (Required) The frequency in minutes to run the check. Possible values are `1`, `5`, `10`, `15`, `30`, `60`, `720`, and `1440`.
* `activated` (Required) Determines if the check is running or not. Possible values `true`, and `false`.  
* `muted` (Optional) Determines if any notifications will be sent out when a check fails and/or recovers. Possible values `true`, and `false`.  
* `double_check` (Optional) Setting this to "true" will trigger a retry when a check fails from the failing region and another, randomly selected region before marking the check as failed. Possible values `true`, and `false`.  
* `ssl_check` (Optional) Determines if the SSL certificate should be validated for expiry. Possible values `true`, and `false`.  
* `should_fail` (Optional) Allows to invert the behaviour of when a check is considered to fail. Allows for validating error status like 404. Possible values `true`, and `false`.  
* `locations` (Required) An array of one or more data center locations where to run the this check.  
* `script` (Optional) A valid piece of Node.js javascript code describing a browser interaction with the Puppeteer framework or a terraform reference to an external javascript file.  
* `environment_variables` (Optional) Key/value pairs for setting environment variables during check execution. These are only relevant for Browser checks. Use global environment variables whenever possible.  
* `Tags` (Optional) A list of Tags for organizing and filtering checks. 
* `setup_snippet_id` (Optional) An ID reference to a snippet to use in the setup phase of an API check.  
* `teardown_snippet_id` (Optional) An ID reference to a snippet to use in the teardown phase of an API check.  
* `local_setup_script` (Optional) A valid piece of Node.js code to run in the setup phase.  
* `local_teardown_script` (Optional) A valid piece of Node.js code to run in the teardown phase.
* `use_global_alert_settings` (Optional) When true, the account level alert setting will be used, not the alert setting defined on this check. Possible values `true`, and `false`.  
* `degraded_response_time` (Optional) The response time in milliseconds where a check should be considered degraded. Possible values are between 0 and 30000. Defaults to `15000`.  
* `max_response_time` (Optional) The response time in milliseconds where a check should be considered failing. Possible values are between 0 and 30000. Defaults to `30000`.  
* `group_id` (Optional). The id of the check group this check is part of.
* `group_order` (Optional) The position of this check in a check group. It determines in what order checks are run when a group is triggered from the API or from CI/CD.  
* `request` (Optional). An API check might have one request config. Supported values documented below.    
* `alert_settings` (Optional). Supported values documented below.  

### Argument Reference: request
The `request` section is added to API checks and supports the following:  
* `method` (Optional) The HTTP method to use for this API check. Possible values are `GET`, `POST`, `PUT`, `HEAD`, `DELETE`, `PATCH`. Defaults to `GET`.
* `url` (Required) .
* `follow_redirects` (Optional) .
* `headers` (Optional) .
* `query_parameters` (Optional).
* `body` (Optional)
* `body_type` (Optional) Possible values `NONE`, `JSON`, `FORM`, `RAW`, and `GRAPHQL`.
* `assertion` (Optional) A request can have multiple assetions. Assertion has the following arguments:  
  * `source` (Required) Possible values `STATUS_CODE`, `JSON_BODY`, `HEADERS`, `TEXT_BODY`, and `RESPONSE_TIME`.  
  * `property` (Optional).  
  * `comparison` (Required) Possible values `EQUALS`, `NOT_EQUALS`, `HAS_KEY`, `NOT_HAS_KEY`, `HAS_VALUE`, `NOT_HAS_VALUE`, `IS_EMPTY`, `NOT_EMPTY`, `GREATER_THAN`, `LESS_THAN`, `CONTAINS`, `NOT_CONTAINS`, `IS_NULL`, and `NOT_NULL`.  
  * `target` (Required).  
* `basic_auth` (Optional) A request might have one basic_auth config. basic_auth has two arguments:
  * `username` (Required)  
  * `password` (Required)  


### Argument Reference: alert_settings
The `alert_Settings` section supports the following:  
* `escalation_type` (Optional) Determines what type of escalation to use. Possible values are `RUN_BASED` or `TIME_BASED`.  
* `run_based_escalation` (Optional). Possible arguments:
  * `failed_run_threshold` (Optional) After how many failed consecutive check runs an alert notification should be send. Possible values are between 1 and 5. Defaults to `1`.  
* `time_based_escalation` (Optional). Possible arguments:
  * `minutes_failing_threshold` (Optional) After how many minutes after a check starts failing an alert should be send. Possible values are `5`, `10`, `15`, and `30`. Defaults to `5`.  
* `reminders` (Optional). Possible arguments:
  * `amount` (Optional) How many reminders to send out after the initial alert notification. Possible values are `0`, `1`, `2`, `3`, `4`, `5`, and `100000`
  * `interval` (Optional). Possible values are `5`, `10`, `15`, and `30`. Defaults to `5`.  
* `ssl_certificates` (Optional) At what interval the reminders should be send.  Possible arguments:
  * `enabled` (Optional) Determines if alert notifications should be send for expiring SSL certificates. Possible values `true`, and `false`. Defaults to `true`.  
  * `alert_threshold` (Optional) At what moment in time to start alerting on SSL certificates. Possible values `3`, `7`, `14`, `30`. Defaults to `3`.  