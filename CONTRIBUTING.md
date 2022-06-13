# Contributing to Checkly Terraform Provider

> Learn about our [Commitment to Open Source](https://checklyhq.com/oss).

Hi! We are really excited that you are interested in contributing to Checkly Terraform Provider, and we really appreciate your commitment. Before submitting your contribution, please make sure to take a moment and read through the following guidelines:

- [Code of Conduct](./CODE_OF_CONDUCT.md)
- [Issue Reporting Guidelines](#issue-reporting-guidelines)
- [Pull Request Guidelines](#pull-request-guidelines)
- [Development Setup](#development-setup)
- [Scripts](#scripts)
- [Project Structure](#project-structure)
- [Release Process](#releases-process)

## Issue Reporting Guidelines

- Always use [GitHub issues](https://github.com/checkly/terraform-provider-checkly/issues/new/choose) to create new issues and select the corresponding issue template.

## Pull Request Guidelines

- Checkout a topic branch from a base branch, e.g. `main`, and merge back against that branch.

- [Make sure to tick the "Allow edits from maintainers" box](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/working-with-forks/allowing-changes-to-a-pull-request-branch-created-from-a-fork). This allows us to directly make minor edits / refactors and saves a lot of time.

- Add accompanying documentation, usage samples & test cases
- Add/update demo files to showcase your changes.
- Use existing resources as templates and ensure that each property has a corresponding `description` field.
- Each PR should be linked with an issue, use [GitHub keywords](https://docs.github.com/en/get-started/writing-on-github/working-with-advanced-formatting/using-keywords-in-issues-and-pull-requests) for that.
- Be sure to follow up project code styles (`$ make fmt`)

- If adding a new feature:
  - Provide a convincing reason to add this feature. Ideally, you should open a "feature request" issue first and have it approved before working on it (it should has the label "state: confirmed")
  - Each new feature should be linked to

- If fixing a bug:
  - Provide a detailed description of the bug in the PR. A working demo would be great!

- It's OK to have multiple small commits as you work on the PR - GitHub can automatically squash them before merging.

- Make sure tests pass!

- Commit messages must follow the [semantic commit messages](https://gist.github.com/joshbuchea/6f47e86d2510bce28f8e7f42ae84c716) so that changelogs can be automatically generated.

## Development Setup

The development branch is `main`. This is the branch that all pull requests should be made against.

> ⚠️ Before you start, is recommended to have a good understanding on how the provider works, the resources it has and its different configurations. Here are the ["getting started"](https://github.com/checkly/terraform-provider-checkly#getting-started) guides.

### Pre-requirements
- [Terraform](https://learn.hashicorp.com/tutorials/terraform/install-cli) >= v1.2.2
- [Go](https://go.dev/doc/install) >= 1.18.2

After you have installed Terraform and Go, you can clone the repository and start working on the `main` branch.

1. [Fork](https://help.github.com/articles/fork-a-repo/) this repository to your own GitHub account and then [clone](https://help.github.com/articles/cloning-a-repository/) it to your local device.

  > If you don't need the whole git history, you can clone with depth 1 to reduce the download size:

  ```sh
  $ git clone --depth=1 git@github.com:checkly/terraform-provider-checkly.git
  ```

2. Navigate into the project and create a new branch:
  ```sh
  cd terraform-provider-checkly && git checkout -b MY_BRANCH_NAME
  ```

3. Download go packages
  ```sh
  $ go mod tidy
  ```

4. Run build process to ensure that everyhing is on place
  ```sh
  $ go build
  ```


> 💡 We recommend to use [tfswitch](https://tfswitch.warrensbox.com/) to easily manage different Terraform versions in your local environment.

You are ready to go, take a look at the [`dev` script](###`make-dev`) so you can start testing your local changes.

## Scripts

In order to facilitate the development process, we have a `GNUmakefile` with a few scripts that are used to build, test and run working examples.

### `make dev`
This command builds your local project and sets up the provider binary to be used in the `demo` directory.
```sh
$ make dev
```

After you run the `dev` script, you should be able to run terraform with the `demo` sample project:
  1. Navigate to the: `$ cd demo`
  1. Start a new terraform project `terraform init`
  1. Run `terraform plan` and/or `terraform apply`

Other useful tip for local testing/development is to override the `demo/versions.tf` with configuration pointing to your local environments:
  ```tf
  terraform {
    required_providers {
      checkly = {
        source  = "dev/checkly/checkly"
        version = "0.0.0-canary"
      }
    }
  }

  provider "checkly" {
    api_key = "xxx"
    account_id = "xxx"
    api_url="http://localhost:3000" # sorry, this is only for Checkly employees 😅
  }
  ```

### `make local-sdk`
If you are also performing updates in the [Checkly Go SDK](https://github.com/checkly/checkly-go-sdk) (ensure to have both projects located  in the same directory), you can use this script to point the provider to the local version of the Checkly Go SDK.
```sh
$ make local-sdk
```

### `make fmt`
Format the code using native terraform and go formatting tools.
```sh
$ make fmt
```

### `make doc`
Use [terraform-plugin-docs](https://github.com/hashicorp/terraform-plugin-docs) to automatically generate resources documentation.
```sh
$ make doc
```

### `make testacc`
Run acceptance tests (all test must pass in order to merge a PR).
```sh
$ make testacc
```

## Project Structure

- `./.github`: contains github (and github actions) related files.

- `./checkly`: contains the provider itself. This is the most important directory and where you will be working most of the time. The file `provider` is the main entry point for the provider, then you will find some helpers and the resource files (i.e. `resource_checkly.go`). Note that each resource has its own `_test` file (i.e. `resource_checkly_test.go`).

> If you want to learn more about Terraform Providers and resources, [here](https://www.terraform.io/cdktf/concepts/providers-and-resources) is a good place to start.

- `./demo`: it is a full working sample of the provider with all the different resources. It's also used to easily try out your local changes.

- `./docs`: this directoy is automatically generated, please do not apply manual changes here. Run `make doc` in order to re-generate documentation.

- `./examples`: here you will find resource and provider exampels (in `.tf` files). This will be use to generate docs and samples.

- `./templates`: contains the templates used to generate the documentation layouts.

- `./tools`: contains tools, scripts or utilities for development workflow.


## Release Process
The release process is automatically handled with [goreleaser](https://goreleaser.com/) and GitHub `release` action.
To trigger a new release you need to create a new git tag, using [SemVer](https://semver.org) pattenr and then push it to the `main` branch.

Remember to create releases candidates releases and spend some time testing in production before publishin a final version. You can also tag the release as "Pre-Release" on GitHub until you consider it mature enough.