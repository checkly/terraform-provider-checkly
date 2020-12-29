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

This Terraform provider enables users to manage [Checkly](https://checklyhq.com) resources like checks. You can read detailed tutorials and explanations of the Checkly Terraform provider here:

1. [Managing Checkly checks with Terraform](https://blog.checklyhq.com/managing-checkly-checks-with-terraform/)
2. [Scaling Puppeteer and Playwright on Checkly with Terraform](https://blog.checklyhq.com/scaling-puppeteer-playwright-on-checkly-with-terraform/)

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
For documentation and example usage see:
1. [Checkly's documentation](https://www.checklyhq.com/docs/integrations/terraform/).
2. [docs/resources](https://github.com/checkly/terraform-provider-checkly/tree/master/docs/resources)
3. [`test.tf`](https://github.com/checkly/terraform-provider-checkly/blob/master/test.tf).

## Developing the provider

Clone the repo, build the project and add it to your Terraform plugins directory. You will need to have Go installed.

```bash
git clone git@github.com:checkly/terraform-provider-checkly.git
cd terraform-provider-checkly
go test
go build && CHECKLY_API_KEY=XXX go test -tags=integration
```
