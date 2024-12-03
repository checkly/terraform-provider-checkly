package interop

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	checkly "github.com/checkly/checkly-go-sdk"
)

func ClientFromProviderData(providerData any) (checkly.Client, diag.Diagnostics) {
	if providerData == nil {
		return nil, diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Missing Configure Type",
				"Expected checkly.Client, got nil. Please report this issue "+
					"to the provider developers.",
			),
		}
	}

	client, ok := providerData.(checkly.Client)
	if !ok {
		return nil, diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Unexpected Configure Type",
				fmt.Sprintf("Expected checkly.Client, got: %T. Please report "+
					"this issue to the provider developers.", providerData),
			),
		}
	}

	return client, nil
}
