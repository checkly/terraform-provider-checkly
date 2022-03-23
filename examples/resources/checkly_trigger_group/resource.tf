resource "checkly_trigger_group" "test-trigger-group" {
  group_id = "215"
}

output "test-trigger-group-url" {
  value = checkly_trigger_group.test-trigger-group.url
}