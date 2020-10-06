# Checkly Terraform provider

[![CircleCI](https://circleci.com/gh/checkly/terraform-provider-checkly/tree/master.svg?style=svg)](https://circleci.com/gh/checkly/terraform-provider-checkly/tree/master)

* [Introduction](#introduction)
* [Supported resource](#supported-resources)
* [Installation](#installing-the-provider)
* [Usage](#using-the-provider)
	* [Checks](#checks)
	  * [API checks](#api-checks)
	  * [Browser checks](#browser-checks)	  
	* [Check groups](#check-groups)
* [Development](#developing-the-provider)

## Introduction

This Terraform provider enables users to manage [Checkly](https://checklyhq.com) resources like checks. You can read a detailed tutorial and explanation of the Checkly Terraform provider here:

* [Managing Checkly checks with Terraform](https://blog.checklyhq.com/managing-checkly-checks-with-terraform/)

## Supported resources

- [x] Checks
- [x] Check groups
- [ ] Alert channels
- [X] Snippets
- [ ] Environment variables


## Installing the provider

Please refer to the [installation guide](https://github.com/checkly/terraform-provider-checkly/blob/master/docs/guides/installing-the-provider-and-getting-started.md)

If you're still using Terraform 0.12 please refer to [terraform 0.12 documentation](https://github.com/checkly/terraform-provider-checkly/blob/master/docs/guides/support-for-terraform-0.12.md)

## Using the provider
For documentation and example usage look into [docs/resources](https://github.com/checkly/terraform-provider-checkly/tree/master/docs/resources). More examples can be found here `test.tf`(https://github.com/checkly/terraform-provider-checkly/blob/master/test.tf).  


### Checks

Add a `checkly_check` resource to your resource file. You can add **API checks** and **browser checks**, either individually or as part of a **check group**.

#### API checks
This first example is a very minimal **API check**.

```terraform
terraform {
  required_version = ">= 0.13"
  required_providers {
    checkly = {
      source  = "checkly/checkly"
      version = "~> 0.6.9"
    }
  }
}

variable "checkly_api_key" {}

provider "checkly" {
  api_key = "${var.checkly_api_key}"
}

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
resource "checkly_check" "example-check2" {
  name                   = "Example check 2"
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
#### Browser checks
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

### Check Groups

Checkly's groups feature allows you to group together a set of related checks, which can also share default settings for various attributes. Here is an example check group:

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

The `group_order` attribute specifies in which order the checks will be executed: 1, 2, 3, etc.



## Developing the provider

Clone the repo, build the project and add it to your Terraform plugins directory. You will need to have Go installed.

```bash
git clone git@github.com:checkly/terraform-provider-checkly.git
cd terraform-provider-checkly
go test
go build && CHECKLY_API_KEY=XXX go test -tags=integration
```
