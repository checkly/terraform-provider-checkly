resource "checkly_trigger_check" "test-trigger-check" {
  check_id = "c1ff95c5-d7f6-4a90-9ce2-1e605f117592"
}

output "test-trigger-check-url" {
  value = checkly_trigger_check.test-trigger-check.url
}
