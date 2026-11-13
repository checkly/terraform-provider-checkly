package provider

import (
	"context"
	"regexp"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSnippetCheckRequiredFields(t *testing.T) {
	config := `resource "checkly_snippet" "test" {}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "name" is required`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "script" is required`),
		},
	})
}

func TestAccSnippetSuccess(t *testing.T) {
	config := `resource "checkly_snippet" "test" {
		name     = "foo"
		script   = "console.log('bar')"
	}`
	accTestCase(t, []resource.TestStep{
		{
			Config: config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_snippet.test",
					"name",
					"foo",
				),
				resource.TestCheckResourceAttr(
					"checkly_snippet.test",
					"script",
					"console.log('bar')",
				),
			),
		},
		{
			ResourceName:      "checkly_snippet.test",
			ImportState:       true,
			ImportStateVerify: true,
		},
	})
}

func TestSnippetAttributeValidators(t *testing.T) {
	var schemaResp fwresource.SchemaResponse
	NewSnippetResource().Schema(context.Background(), fwresource.SchemaRequest{}, &schemaResp)

	runAttributeValidationCases(t, schemaResp.Schema, []attributeValidationCase{
		{[]string{"name"}, types.StringValue("Snippet"), true},
		{[]string{"name"}, types.StringValue(""), false},
		{[]string{"script"}, types.StringValue("console.log(1)"), true},
		{[]string{"script"}, types.StringValue(""), false},
	})
}
