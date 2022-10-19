resource "checkly_trigger_group" "test_trigger_group" {
  group_id = "215"
}

output "test_trigger_group_url" {
  value = checkly_trigger_group.test_trigger_group.url
}