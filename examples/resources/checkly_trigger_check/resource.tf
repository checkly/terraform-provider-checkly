resource "checkly_trigger_check" "test_trigger_check" {
  check_id = "c1ff95c5-d7f6-4a90-9ce2-1e605f117592"
}

output "test_trigger_check_url" {
  value = checkly_trigger_check.test_trigger_check.url
}
