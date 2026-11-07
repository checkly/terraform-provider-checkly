# Import an existing alert channel using its numeric ID.
# The ID is returned by the Checkly API at GET /v1/alert-channels.
terraform import checkly_alert_channel.example 12345
