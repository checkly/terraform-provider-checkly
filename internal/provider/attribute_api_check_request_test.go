package provider

import (
	"context"
	"testing"

	"github.com/checkly/checkly-go-sdk/v2"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAPICheckBasicAuthToObject(t *testing.T) {
	ctx := context.Background()

	credentials := func(username, password string) types.Object {
		return types.ObjectValueMust(apiCheckBasicAuthAttrTypes, map[string]attr.Value{
			"username": types.StringValue(username),
			"password": types.StringValue(password),
		})
	}
	null := types.ObjectNull(apiCheckBasicAuthAttrTypes)

	cases := map[string]struct {
		api   *checkly.BasicAuth
		prior types.Object
		want  types.Object
	}{
		"no credentials":                   {api: nil, prior: null, want: null},
		"empty credentials":                {api: &checkly.BasicAuth{}, prior: null, want: null},
		"empty credentials, prior empty":   {api: &checkly.BasicAuth{}, prior: credentials("", ""), want: credentials("", "")},
		"no credentials, prior empty":      {api: nil, prior: credentials("", ""), want: credentials("", "")},
		"empty credentials, prior set":     {api: &checkly.BasicAuth{}, prior: credentials("user", "pass"), want: null},
		"empty credentials, prior unknown": {api: &checkly.BasicAuth{}, prior: types.ObjectUnknown(apiCheckBasicAuthAttrTypes), want: null},
		"credentials":                      {api: &checkly.BasicAuth{Username: "user", Password: "pass"}, prior: null, want: credentials("user", "pass")},
		"username only":                    {api: &checkly.BasicAuth{Username: "user"}, prior: credentials("", ""), want: credentials("user", "")},
		"credentials changed outside":      {api: &checkly.BasicAuth{Username: "new", Password: "pass"}, prior: credentials("user", "pass"), want: credentials("new", "pass")},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			var diags diag.Diagnostics
			got := apiCheckBasicAuthToObject(ctx, c.api, c.prior, &diags)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if !got.Equal(c.want) {
				t.Errorf("got %s, want %s", got, c.want)
			}
		})
	}
}
