terraform {
  required_version = ">= 0.13"
  required_providers {
    checkly = {
      # dev/checkly/checkly is used for development only,
      # if you're using checkly provider you'll need to follow
      # installation guide instrctions found in README
      source  = "dev/checkly/checkly"
      version = "0.0.2"
    }
  }
}

variable "checkly_api_key" {
}

provider "checkly" {
  api_key = var.checkly_api_key
}
