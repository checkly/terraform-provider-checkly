# checkly_status_page_service is deprecated together with checkly_status_page; services of v3 status pages are checkly_status_page_v3_component resources.
# Import an existing status page service using its ID.
# The ID is returned by the Checkly API at GET /v1/status-pages/services.
terraform import checkly_status_page_service.example 8a894b49-467f-4af6-9230-cd4d3bd452f4
