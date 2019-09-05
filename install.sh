#!/usr/bin/env bash
set -x

# Determine architecture
if [[ $(uname -s) == Darwin ]]
then	
	package='terraform-provider-checkly_darwin_amd64.gz'
	package_unzipped='terraform-provider-checkly_darwin_amd64'
	directory='darwin_amd64'

elif [[ $(uname -s) == Linux ]]
then
	package='terraform-provider-checkly_linux_amd64.gz'
	package_unzipped='terraform-provider-checkly_linux_amd64'
	directory='linux_amd64'
else
	echo "No supported architecture found"
	exit 1
fi

# Download the binary
curl -OL https://github.com/bitfield/terraform-provider-checkly/releases/latest/download/$package

# Copy the binary to your Terraform plugin folder, unzip it and rename it to just `terraform-provider-checkly`
cp $package ~/.terraform.d/plugins/$directory
gunzip ~/.terraform.d/plugins/$directory/$package
mv ~/.terraform.d/plugins/$directory/$package_unzipped ~/.terraform.d/plugins/$directory/terraform-provider-checkly
chmod +x ~/.terraform.d/plugins/$directory/terraform-provider-checkly