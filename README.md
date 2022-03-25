<p>
  <img height="128" src="https://www.checklyhq.com/images/footer-logo.svg" align="right" />
  <h1>Checkly Terraform Provider</h1>
</p>

[![Tests](https://github.com/checkly/terraform-provider-checkly/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/checkly/terraform-provider-checkly/actions/workflows/test.yml)
[![Release](https://github.com/checkly/terraform-provider-checkly/actions/workflows/release.yml/badge.svg)](https://github.com/checkly/terraform-provider-checkly/actions/workflows/release.yml)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/checkly/terraform-provider-checkly)
![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/checkly/terraform-provider-checkly?label=Version)

> 🌐 A Terraform provider for the Checkly monitoring service

## Overview

This Terraform provider enables users to manage [Checkly](https://checklyhq.com) resources like checks, groups, snippets and more.

## Installation

Please refer to the [installation guide](https://github.com/checkly/terraform-provider-checkly/blob/main/docs/guides/getting-started.md)

If you're still using Terraform 0.12 please refer to [terraform 0.12 documentation](https://github.com/checkly/terraform-provider-checkly/blob/main/docs/guides/support-for-terraform-0.12.md)

## How to use?

For documentation and example usage see:
1. [Checkly's documentation](https://www.checklyhq.com/docs/integrations/terraform/).
2. [The official provider documentation](https://registry.terraform.io/providers/checkly/checkly/latest/docs)
3. [`test.tf`](https://github.com/checkly/terraform-provider-checkly/blob/main/test.tf).

You can also find step-by-step guides on Checkly's blog:

1. [Managing Checkly checks with Terraform](https://blog.checklyhq.com/managing-checkly-checks-with-terraform/)
2. [Scaling Puppeteer and Playwright on Checkly with Terraform](https://blog.checklyhq.com/scaling-puppeteer-playwright-on-checkly-with-terraform/)


## Run Locally

Clone the repo, build the project and add it to your Terraform plugins directory. You will need to have Go installed.

```bash
git clone git@github.com:checkly/terraform-provider-checkly.git
cd terraform-provider-checkly
go test
go build && CHECKLY_API_KEY=xxx CHECKLY_ACCOUNT_ID=xxx  go test -tags=integration
```

> We recommend to use [tfswitch](https://tfswitch.warrensbox.com/) to easily manage different Terraform versions in your local environment.

## License

[MIT](https://github.com/checkly/terraform-checkly-provider/blob/main/LICENSE)

<br>


<p align="center">
  <a href="https://checklyhq.com?utm_source=github&utm_medium=sponsor-logo-github&utm_campaign=headless-recorder" target="_blank">
  <img width="100px" src="https://www.checklyhq.com/images/text_racoon_logo.svg" alt="Checkly" />
  </a>
  <br />
  <i><sub>Delightful Active Monitoring for Developers</sub></i>
  <br>
  <b><sub>From Checkly with ♥️</sub></b>
<p>
