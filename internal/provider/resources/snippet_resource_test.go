package resources_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSnippetCheckRequiredFields(t *testing.T) {
	config := `resource "checkly_snippet" "test" {}`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),

		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`The argument "name" is required`),
			},
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`The argument "script" is required`),
			},
		},
	})
}

func TestAccSnippetSuccess(t *testing.T) {
	config := `resource "checkly_snippet" "test" {
		name     = "foo"
		script   = "console.log('bar')"
	}`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),

		Steps: []resource.TestStep{
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
		},
	})
}
