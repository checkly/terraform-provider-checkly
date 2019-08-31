#!/usr/bin/env bash
set -x

# Download the binary
curl -OL https://github.com/bitfield/terraform-provider-checkly/releases/latest/download/terraform-provider-checkly_darwin_amd64.gz

# Copy the binary to your Terraform plugin folder, unzip it and rename it to just `terraform-provider-checkly`

cp terraform-provider-checkly_darwin_amd64.gz ~/.terraform.d/plugins/darwin_amd64
gunzip ~/.terraform.d/plugins/darwin_amd64/terraform-provider-checkly_darwin_amd64.gz
mv ~/.terraform.d/plugins/darwin_amd64/terraform-provider-checkly_darwin_amd64 ~/.terraform.d/plugins/darwin_amd64/terraform-provider-checkly
chmod +x ~/.terraform.d/plugins/darwin_amd64/terraform-provider-checkly
