# Terraform 0.13+ uses the Terraform Registry:

terraform {
  required_version = "= 1.1.17"
  required_providers {
    checkly = {
      source  = "checkly/checkly"
      version = "1.4.3"
    }
  }
}

# Configure the Checkly provider
variable "checkly_api_key" {
}

variable "checkly_account_id" {
}

provider "checkly" {
  api_key = var.checkly_api_key
  account_id = var.checkly_account_id
}
