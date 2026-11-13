package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/checkly/checkly-go-sdk/v2"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every
// Terraform CLI command executed to create a provider server to which the
// CLI can reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"checkly": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccClient returns an API client configured like the
// provider under test, from the acceptance test environment.
func testAccClient() checkly.Client {
	apiURL := os.Getenv("CHECKLY_API_URL")
	if apiURL == "" {
		apiURL = "https://api.checklyhq.com"
	}
	client, err := checkly.NewClient(
		checkly.WithBaseURL(apiURL),
		checkly.WithAuth(os.Getenv("CHECKLY_API_KEY"), os.Getenv("CHECKLY_ACCOUNT_ID")),
	)
	if err != nil {
		panic(err)
	}
	return client
}

func testAccPreCheck(t *testing.T) {
	if os.Getenv("CHECKLY_API_KEY") == "" {
		t.Fatal("CHECKLY_API_KEY must be set for acceptance tests")
	}

	if os.Getenv("CHECKLY_ACCOUNT_ID") == "" {
		t.Fatal("CHECKLY_ACCOUNT_ID must be set for acceptance tests")
	}
}

func accTestCase(t *testing.T, steps []resource.TestStep) {
	accTestCaseWithErrorCheck(t, steps, nil)
}

// accTestCaseWithErrorCheck behaves like accTestCase but routes any step error
// through errorCheck before the framework reports it. This lets a test skip
// when the account lacks setup the test depends on (e.g. a connected
// integration) rather than failing.
func accTestCaseWithErrorCheck(t *testing.T, steps []resource.TestStep, errorCheck resource.ErrorCheckFunc) {
	accTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             nil,
		ErrorCheck:               errorCheck,
		Steps:                    steps,
	})
}

// accTest runs testCase in parallel with the other acceptance tests. Each
// test creates its own resources, and none of them changes another test's:
// creating an alert channel through the public API does not subscribe
// existing checks or groups (only the web app and the Vercel integration
// do), and the SDK creates checks, monitors and groups with
// autoAssignAlerts=false, so they don't subscribe to other tests' channels.
// Acceptance tests must run through accTest, not call resource.Test or
// resource.ParallelTest directly, so this is the one place to change if
// that stops being true.
func accTest(t *testing.T, testCase resource.TestCase) {
	resource.ParallelTest(t, testCase)
}
