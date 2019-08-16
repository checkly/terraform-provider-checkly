// +build integration

package main

import (
	"os"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
)

func getAPIKey(t *testing.T) string {
	key := os.Getenv("CHECKLY_API_KEY")
	if key == "" {
		t.Fatal("'CHECKLY_API_KEY' must be set for integration tests")
	}
	return key
}

func TestChecklyTerraformIntegration(t *testing.T) {
	t.Parallel()
	terraformOptions := &terraform.Options{
		TerraformDir: ".",
		Vars: map[string]interface{}{
			"checkly_api_key": getAPIKey(t),
		},
	}
	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)
}
