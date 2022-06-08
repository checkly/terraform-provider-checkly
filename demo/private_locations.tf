resource "checkly_private_locations" "location" {
  name          = "New Private Location"
  slug_name     = "new-private-location"
  icon          = "location"
}

output "test-trigger-check-key" {
  value = checkly_private_locations.location.raw_key
}
