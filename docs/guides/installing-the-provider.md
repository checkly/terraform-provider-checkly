# Installing the provider

1. If you're on MacOS, just run the  `install.sh` script:

```bash
curl https://raw.githubusercontent.com/checkly/terraform-provider-checkly/master/install.sh | sh
```

Otherwise, download the appropriate binary for your platform from the [latest tagged release](https://github.com/checkly/terraform-provider-checkly/releases).
Then copy the binary to your Terraform plugin folder, unzip it and rename it to just `terraform-provider-checkly`. Lastly, set the correct access rights.


2. Run `terraform init` and then `terraform providers`. The Checkly plugin should be listed.

```bash
terraform init

Initializing provider plugins...
Terraform has been successfully initialized!

terraform providers
.
└── provider.checkly
```

If you're having issues, please check [the Hashicorp docs on installing third party plugins.](https://www.terraform.io/docs/configuration/providers.html#third-party-plugins)
