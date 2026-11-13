# Import an existing status page automation rule using "<status_page_id>/<rule_id>".
# Both IDs are returned by the Checkly API at GET /v3/status-pages/{status_page_id}/automation-rules.
terraform import checkly_status_page_v3_automation_rule.example e35f7e14-91b2-4d24-b7b6-e0f9e2f8e51c/5b0a9a1e-8e29-4b71-92e1-0db9b3b41f0a
