package checkly

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
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
	})
}
