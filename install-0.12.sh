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

jq_cmd=".assets[] | select(.name | endswith(\"${platform}.gz\")).browser_download_url"
# Find latest binary release URL for this platform
url="$(curl -s https://api.github.com/repos/checkly/terraform-provider-checkly/releases/latest | jq -r "${jq_cmd}")"
# Download the tarball
curl -OL ${url}
# Rename and copy to your Terraform plugin folder
filename=$(basename $url)
gunzip ${filename}
filename=${filename%.gz}
chmod +x ${filename}
PLUGIN_DIR=~/.terraform.d/plugins/$platform
mkdir -p $PLUGIN_DIR
mv $filename ${PLUGIN_DIR}/${filename%_${platform}}