# Checkly Terraform provider

[![CircleCI](https://circleci.com/gh/checkly/terraform-provider-checkly/tree/master.svg?style=svg)](https://circleci.com/gh/checkly/terraform-provider-checkly/tree/master)

* [Introduction](#introduction)
* [Supported resource](#supported-resources)
* [Installation](#installing-the-provider)
* [Usage](#using-the-provider)
	* [Checks](#checks)
	* [Check groups](#check-groups)
* [Development](#developing-the-provider)

## Introduction

This Terraform provider enables users to manage [Checkly](https://checklyhq.com) resources like checks. You can read a detailed tutorial and explanation of the Checkly Terraform provider here:

* [Managing Checkly checks with Terraform](https://blog.checklyhq.com/managing-checkly-checks-with-terraform/)

## Supported resources

- [x] Checks
- [x] Check groups
- [ ] Alert channels
- [ ] Snippets
- [ ] Environment variables


## Installing the provider

1. If you're on MacOS, just run the  `install.sh` script:

```bash
curl https://raw.githubusercontent.com/checkly/terraform-provider-checkly/master/install.sh | sh
```

Otherwise, download the appropriate binary for your platform from the [latest tagged release](https://github.com/checkly/terraform-provider-checkly/releases).
Then copy the binary to your Terraform plugin folder, unzip it and rename it to just `terraform-provider-checkly`. Lastly, set the correct access rights.


1. Run `terraform init` and then `terraform providers`. The Checkly plugin should be listed.

```bash
terraform init

Initializing provider plugins...
Terraform has been successfully initialized!

terraform providers
.
└── provider.checkly
```

If you're having issues, please check [the Hashicorp docs on installing third party plugins.](https://www.terraform.io/docs/configuration/providers.html#third-party-plugins)

### Authentication

To use the provider with your Checkly account, you will need an API Key for the account. Go to the [Account Settings: API Keys page](https://app.checklyhq.com/account/api-keys) and click 'Create API Key'.

Now expose the API key as an environment variable in your shell:

```bash
export TF_VAR_checkly_api_key=<my_api_key>
```

## Using the provider
For usage examples please look into `test.tf`(https://github.com/checkly/terraform-provider-checkly/blob/master/test.tf) to run these examples, set env variable `TF_CHECKLY_API_KEY` to your checkly API key, then run `make plan` followed by `make apply`.  

Before we have full Terraform-style documentation, make sure to also reference the [Checkly public API documentation](https://www.checklyhq.com/docs/api) as the Terraform provider *talks* to this API.

We are still working on more fleshed out examples and documentation at the moment.


## Developing the provider

Clone the repo, build the project and add it to your Terraform plugins directory. You will need to have Go installed.

```bash
git clone git@github.com:checkly/terraform-provider-checkly.git
cd terraform-provider-checkly
go test
go build && CHECKLY_API_KEY=XXX go test -tags=integration
```
