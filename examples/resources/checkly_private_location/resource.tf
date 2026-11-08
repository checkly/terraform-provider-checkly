resource "checkly_private_location" "location" {
  name      = "New Private Location"
  slug_name = "new-private-location"
}

# The proxy URL can hold credentials, so it comes from a sensitive variable.
variable "proxy_url" {
  type      = string
  sensitive = true
}

# API checks running in this location send their HTTP requests through the
# proxy.
resource "checkly_private_location" "location-with-proxy" {
  name      = "Private Location behind a proxy"
  slug_name = "private-location-behind-a-proxy"
  proxy_url = var.proxy_url
}
