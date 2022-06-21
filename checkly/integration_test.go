//go:build integration
// +build integration

package checkly

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

func getAccountId(t *testing.T) string {
	key := os.Getenv("CHECKLY_ACCOUNT_ID")
	if key == "" {
		t.Fatal("'CHECKLY_ACCOUNT_ID' must be set for integration tests")
	}
	return key
}

func TestChecklyTerraformIntegration(t *testing.T) {
	t.Parallel()
	terraformOptions := &terraform.Options{
		TerraformDir: "../",
		Vars: map[string]interface{}{
			"checkly_api_key":    getAPIKey(t),
			"checkly_account_id": getAccountId(t),
		},
	}
	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)
	planPath := "./test.plan"
	exit, err := terraform.GetExitCodeForTerraformCommandE(t, terraformOptions, terraform.FormatArgs(terraformOptions, "plan", "--out="+planPath, "-input=false", "-lock=true", "-detailed-exitcode")...)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(planPath)
	if exit != terraform.DefaultSuccessExitCode {
		t.Fatalf("want DefaultSuccessExitCode, got %d", exit)
	}
}
