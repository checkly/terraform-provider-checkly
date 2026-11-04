# checkly_status_page is deprecated; new status pages use checkly_status_page_v3, which manages a different kind of status page.
# Import an existing status page using its ID.
# The ID is returned by the Checkly API at GET /v1/status-pages.
terraform import checkly_status_page.example cd8d05a4-c292-4dc4-a78f-1dea65e5457e
