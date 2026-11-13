package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestHTTPURLValidator(t *testing.T) {
	cases := map[string]struct {
		value types.String
		valid bool
	}{
		"null":                     {types.StringNull(), true},
		"unknown":                  {types.StringUnknown(), true},
		"empty":                    {types.StringValue(""), false},
		"https":                    {types.StringValue("https://example.com"), true},
		"http with path and query": {types.StringValue("http://example.com/a/b.png?x=1&y=%20#top"), true},
		"port and credentials":     {types.StringValue("https://user:pass@example.com:8443/"), true},
		"IPv6 host":                {types.StringValue("http://[::1]:8080/"), true},
		"IPv6 host with userinfo":  {types.StringValue("http://user@[2001:db8::1]/a"), true},
		"IPv4 in brackets":         {types.StringValue("http://[127.0.0.1]/"), false},
		"bracketed name":           {types.StringValue("http://[example.com]/"), false},
		"unclosed bracket":         {types.StringValue("http://[::1/"), false},
		"text after the bracket":   {types.StringValue("http://[::1]x/"), false},
		"bracket in path":          {types.StringValue("https://example.com/a[1]"), false},
		"bracket in query":         {types.StringValue("https://example.com/?a[]=1"), false},
		"bracket in fragment":      {types.StringValue("https://example.com/#[x]"), false},
		"fragment":                 {types.StringValue("https://example.com/#a/b?c"), true},
		"two fragments":            {types.StringValue("https://example.com/#a#b"), false},
		"reserved characters":      {types.StringValue("https://example.com/!$&'()*+,;=:@~"), true},
		"uppercase scheme":         {types.StringValue("HTTPS://example.com"), false},
		"other scheme":             {types.StringValue("ftp://example.com"), false},
		"mailto":                   {types.StringValue("mailto:a@example.com"), false},
		"no scheme":                {types.StringValue("example.com/logo.png"), false},
		"relative":                 {types.StringValue("/logo.png"), false},
		"no slashes":               {types.StringValue("https:example.com"), false},
		"no host":                  {types.StringValue("https:///logo.png"), false},
		"only a port":              {types.StringValue("https://:443/"), false},
		"space":                    {types.StringValue("https://example.com/a b.png"), false},
		"non-ASCII":                {types.StringValue("https://exämple.com/"), false},
		"bad escape":               {types.StringValue("https://example.com/%zz"), false},
		"truncated escape":         {types.StringValue("https://example.com/%2"), false},
		"quote":                    {types.StringValue(`https://example.com/"x"`), false},
		"not a URL":                {types.StringValue("not a url"), false},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			diags := validateString([]validator.String{httpURL()}, path.Root("logo"), c.value)
			if diags.HasError() == c.valid {
				t.Errorf("valid = %v, diagnostics: %v", c.valid, diags)
			}
		})
	}
}

func TestURLValidatorOrEmpty(t *testing.T) {
	v := httpURL().orEmpty()
	if diags := validateString([]validator.String{v}, path.Root("logo"), types.StringValue("")); diags.HasError() {
		t.Errorf("empty rejected: %v", diags)
	}
	if diags := validateString([]validator.String{v}, path.Root("logo"), types.StringValue("logo.png")); !diags.HasError() {
		t.Error("a value without a scheme was accepted")
	}
	// orEmpty returns a copy: the original still rejects the empty string.
	if diags := validateString([]validator.String{httpURL()}, path.Root("logo"), types.StringValue("")); !diags.HasError() {
		t.Error("httpURL() accepted the empty string")
	}
}

func TestURLValidatorWithSchemes(t *testing.T) {
	v := urlWithSchemes("http", "https", "mailto")
	cases := map[string]bool{
		"mailto:support@example.com": true,
		"https://example.com/help":   true,
		"mailto:":                    false,
		"MAILTO:support@example.com": false,
		"tel:+123456789":             false,
	}
	for value, valid := range cases {
		t.Run(value, func(t *testing.T) {
			diags := validateString([]validator.String{v}, path.Root("support_link"), types.StringValue(value))
			if diags.HasError() == valid {
				t.Errorf("valid = %v, diagnostics: %v", valid, diags)
			}
		})
	}

	if got, want := v.Description(context.Background()), "value must be an absolute http, https or mailto URL"; got != want {
		t.Errorf("Description() = %q, want %q", got, want)
	}
}

func TestURLValidatorErrorOmitsTheValue(t *testing.T) {
	diags := validateString([]validator.String{httpURL()}, path.Root("logo"), types.StringValue("ftp://user:secret@example.com"))
	if diags.ErrorsCount() != 1 {
		t.Fatalf("diagnostics = %v", diags)
	}
	if detail := diags.Errors()[0].Detail(); strings.Contains(detail, "secret") {
		t.Errorf("the error repeats the value: %s", detail)
	}
}
