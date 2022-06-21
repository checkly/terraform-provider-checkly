terraform {
  required_version = ">= 1.1.17"
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
variable "api_key" {
}

variable "account_id" {
}

variable "api_url" {
}

provider "checkly" {
  api_key    = var.api_key
  account_id = var.account_id
  api_url    = var.api_url
}