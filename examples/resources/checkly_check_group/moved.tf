resource "checkly_check_group_v2" "example" {
  # The attributes don't carry over as they are: translate them as shown in
  # the "Check groups" section of the v2 upgrade guide.
}

moved {
  from = checkly_check_group.example
  to   = checkly_check_group_v2.example
}
