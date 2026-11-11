package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// addAPIError adds an error diagnostic for a failed Checkly API request,
// with summary naming the failed operation and the API error as detail.
func addAPIError(diags *diag.Diagnostics, summary string, err error) {
	diags.AddError(summary, fmt.Sprintf("Checkly API error: %s", err))
}
