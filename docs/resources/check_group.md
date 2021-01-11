# checkly_check_group
The `checkly_check_group` resource allows users to manage checkly Check Groups.  

Checkly's groups feature allows you to group together a set of related checks, which can also share default settings for various attributes. Here is an example check group:

## Example usage
```terraform
resource "checkly_check_group" "test-group1" {
  name      = "My test group 1"
  activated = true
  muted     = false
  tags = [
    "auto"
  ]

  locations = [
    "eu-west-1",
  ]
  concurrency = 3
  api_check_defaults {
    url = "http://example.com/"
    headers = {
      X-Test = "foo"
    }

    query_parameters = {
      query = "foo"
    }

    assertion {
      source     = "STATUS_CODE"
      property   = ""
      comparison = "EQUALS"
      target     = "200"
    }

    assertion {
      source     = "TEXT_BODY"
      property   = ""
      comparison = "CONTAINS"
      target     = "welcome"
    }

    basic_auth {
      username = "user"
      password = "pass"
    }
  }
  environment_variables = {
    ENVTEST = "Hello world"
  }
  double_check              = true
  use_global_alert_settings = false

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
      amount   = 2
      interval = 5
    }
  }
  local_setup_script    = "setup-test"
  local_teardown_script = "teardown-test"
}
```  
To add a check to a group, set its `group_id` attribute to the ID of the group. For example:

```terraform
resource "checkly_check" "test-check1" {
  name                      = "My test check 1"
  ...
  group_id    = checkly_check_group.test-group1.id
  group_order = 1
}
```

## Example Usage - With Alert channels
first define an alert channel
```terraform 
resource "checkly_alert_channel" "email_ac1" {
  email {
    address = "info@example.com"
  }
}
resource "checkly_alert_channel" "email_ac2" {
  email {
    address = "info2@example.com"
  }
}
```

then connect the check group to the alert channel
```terraform
resource "checkly_check_group" "test-group1" {
  name      = "My test group 1"
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
* `name` (Required) The name of the check group.  
* `activated` (Required) Determines if the checks in the group are running or not.  
* `muted` (Optional) Determines if any notifications will be send out when a check in this group fails and/or recovers.  
* `locations` (Required) An array of one or more data center locations where to run the checks.  
* `concurrency` (Required) Determines how many checks are invoked concurrently when triggering a check group from CI/CD or through the API.  
* `environment_variables` (Optional)  Key/value pairs for setting environment variables during check execution. These are only relevant for Browser checks. Use global environment variables whenever possible.  
* `double_check` (Optional) Setting this to "true" will trigger a retry when a check fails from the failing region and another, randomly selected region before marking the check as failed.  
* `tags` (Optional) Tags for organizing and filtering checks.  
* `local_setup_script` (Optional) A valid piece of Node.js code to run in the setup phase of an API check in this group.  
* `local_teardown_script` (Optional) A valid piece of Node.js code to run in the teardown phase of an API check in this group.  
* `use_global_alert_settings` (Optional) When true, the account level alert setting will be used, not the alert setting defined on this check group.  
* `api_check_defaults` (Optional) Default configs to use for all api checks belonging to this group. Supported values documented below.  
* `alert_settings` (Optional). Supported values documented below. 


### Argument Reference: api_check_defaults
The `api_check_defaults` section supports the following:  
* `url` (Required) The base url for this group which you can reference with the {{GROUP_BASE_URL}} variable in all group checks.  
* `headers` (Optional).  
* `query_parameters` (Optional).  
* `assertion` (Optional). Possible arguments:
  * `source` (Required) Possible values `STATUS_CODE`, `JSON_BODY`, `HEADERS`, `TEXT_BODY`, and `RESPONSE_TIME`.  
  * `property` (Optional).  
  * `comparison` (Required) Possible values `EQUALS`, `NOT_EQUALS`, `HAS_KEY`, `NOT_HAS_KEY`, `HAS_VALUE`, `NOT_HAS_VALUE`, `IS_EMPTY`, `NOT_EMPTY`, `GREATER_THAN`, `LESS_THAN`, `CONTAINS`, `NOT_CONTAINS`, `IS_NULL`, and `NOT_NULL`.  
  * `target` (Required)
* `basic_auth` (Optional). Possible arguments
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