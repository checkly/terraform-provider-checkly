terraform {
  required_version = "<= 1.1.17"
  required_providers {
    checkly = {
      # dev/checkly/checkly is used for development only,
      # if you're using checkly provider you'll need to follow
      # installation guide instrctions found in README
      source  = "dev/checkly/checkly"
      version = "0.0.0-canary"
    }
  }
}
variable "checkly_api_key" {
}

variable "checkly_account_id" {
}

provider "checkly" {
  api_key = var.checkly_api_key
  account_id = var.checkly_account_id
}