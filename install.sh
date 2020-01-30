#!/usr/bin/env bash
set -x

# Determine architecture
if [[ $(uname -s) == Darwin ]]
then
	platform='darwin_amd64'
elif [[ $(uname -s) == Linux ]]
then
	platform='linux_amd64'
else
	echo "No supported architecture found"
	exit 1
fi

package="terraform-provider-checkly_${platform}"
jq_cmd=".assets[] | select(.name == \"${package}.gz\").browser_download_url"
# Find latest binary release URL for this platform
url="$(curl -s https://api.github.com/repos/bitfield/terraform-provider-checkly/releases/latest | jq -r "${jq_cmd}")"
# Download the tarball
curl -OL ${url}

# Rename and copy to your Terraform plugin folder
gunzip ${package}.gz
mv $package terraform-provider-checkly
chmod +x terraform-provider-checkly
PLUGIN_DIR=~/.terraform.d/plugins/$platform
mkdir -p $PLUGIN_DIR
mv terraform-provider-checkly $PLUGIN_DIR
