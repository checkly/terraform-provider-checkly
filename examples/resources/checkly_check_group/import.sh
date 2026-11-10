# checkly_check_group is deprecated; to start managing an existing resource, import it as checkly_check_group_v2 instead.
# A group already managed as checkly_check_group switches to checkly_check_group_v2 with a moved block; see the v2 upgrade guide.
# Import an existing check group using its numeric ID.
# The ID is returned by the Checkly API at GET /v1/check-groups.
terraform import checkly_check_group.example 12345
