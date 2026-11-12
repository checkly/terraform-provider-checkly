# Import an existing Playwright check suite using its check ID.
# You can find the ID in the check's URL in the Checkly dashboard.
#
# The code bundle is not imported: the API doesn't return the bundle's
# metadata, so the first plan after the import shows an in-place update of
# `bundle`, and the apply sends the bundle from the configuration.
terraform import checkly_playwright_check_suite.example 3ffee447-5c86-4c56-b95a-2e1b4d1a1234
