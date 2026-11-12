package provider

import (
	"context"
	"fmt"
	"net/netip"
	"net/url"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// urlValidator rejects strings that are not absolute URLs with one of
// schemes, as the API checks them (Joi's string().uri() with a scheme list):
//
//   - only the characters RFC 3986 allows in a URI, with every "%" starting
//     a two-digit hex escape;
//   - "[" and "]" only around an IPv6 address as the host, such as
//     "http://[::1]/", and no "#" after the one that starts the fragment;
//   - a lowercase scheme from schemes, which the API compares
//     case-sensitively;
//   - for http and https, "//" and a host; for other schemes, such as
//     mailto, a non-empty remainder after the colon.
//
// The http and https rule is stricter than RFC 3986, which also allows
// "http:path" and an empty host, because such a URL leads nowhere. So is
// the host rule: RFC 3986 also allows a future IP version ("[v1.x]") in
// brackets, which no URL in practice uses. An empty
// string is rejected unless allowEmpty is set. The error does not repeat the
// value, which may hold credentials.
type urlValidator struct {
	schemes    []string
	allowEmpty bool
}

var _ validator.String = urlValidator{}

// httpURL returns a validator that accepts http and https URLs.
func httpURL() urlValidator {
	return urlValidator{schemes: []string{"http", "https"}}
}

// urlWithSchemes returns a validator that accepts URLs with one of schemes,
// which must be lowercase.
func urlWithSchemes(schemes ...string) urlValidator {
	return urlValidator{schemes: schemes}
}

// orEmpty returns a copy of v that also accepts the empty string, for
// attributes where an empty value means "none".
func (v urlValidator) orEmpty() urlValidator {
	v.allowEmpty = true
	return v
}

func (v urlValidator) schemeList() string {
	if len(v.schemes) == 1 {
		return v.schemes[0]
	}
	return strings.Join(v.schemes[:len(v.schemes)-1], ", ") + " or " + v.schemes[len(v.schemes)-1]
}

func (v urlValidator) Description(ctx context.Context) string {
	description := fmt.Sprintf("value must be an absolute %s URL", v.schemeList())
	if v.allowEmpty {
		description += " or empty"
	}
	return description
}

func (v urlValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v urlValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	if value == "" && v.allowEmpty {
		return
	}

	if problem := v.check(value); problem != "" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid URL",
			fmt.Sprintf("%s must be an absolute %s URL: %s.", req.Path, v.schemeList(), problem),
		)
	}
}

// check returns why value is not an accepted URL, or "" when it is.
func (v urlValidator) check(value string) string {
	if value == "" {
		return "the value is empty"
	}

	for i := 0; i < len(value); i++ {
		c := value[i]
		if c == '%' {
			if i+2 >= len(value) || !isHexDigit(value[i+1]) || !isHexDigit(value[i+2]) {
				return `every "%" must start a two-digit hex escape, such as "%20"`
			}
			continue
		}
		if !isURIChar(c) {
			return "it contains a character that URLs don't allow, such as a space or a non-ASCII character; percent-encode it"
		}
	}

	scheme, rest, found := strings.Cut(value, ":")
	if !found || !isURIScheme(scheme) {
		return `it must start with a scheme, such as "https://"`
	}
	if !slices.Contains(v.schemes, scheme) {
		return fmt.Sprintf("the scheme must be %s, in lowercase", v.schemeList())
	}

	if _, fragment, hasFragment := strings.Cut(rest, "#"); hasFragment && strings.Contains(fragment, "#") {
		return `it has more than one "#"; percent-encode the others as "%23"`
	}
	if problem := checkURIBrackets(rest); problem != "" {
		return problem
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return "it cannot be parsed"
	}

	if scheme == "http" || scheme == "https" {
		if !strings.HasPrefix(rest, "//") || parsed.Hostname() == "" {
			return fmt.Sprintf(`it must have a host, as in "%s://example.com/"`, scheme)
		}
		return ""
	}

	if rest == "" {
		return fmt.Sprintf(`it must have something after "%s:"`, scheme)
	}
	return ""
}

// checkURIBrackets returns why the part of a URI after the scheme's colon
// has "[" or "]" where RFC 3986 doesn't allow them, or "" when it has
// none or only around an IPv6 address as the host.
func checkURIBrackets(rest string) string {
	const problem = `"[" and "]" are only allowed around an IPv6 address as the host, such as "http://[::1]/"; percent-encode them elsewhere`

	outside := rest
	if authority, ok := strings.CutPrefix(rest, "//"); ok {
		if end := strings.IndexAny(authority, "/?#"); end >= 0 {
			authority = authority[:end]
		}
		hostPort := authority
		if at := strings.LastIndexByte(authority, '@'); at >= 0 {
			hostPort = authority[at+1:]
		}
		if literal, afterOpen := strings.CutPrefix(hostPort, "["); afterOpen {
			address, port, closed := strings.Cut(literal, "]")
			if !closed || (port != "" && !strings.HasPrefix(port, ":")) {
				return problem
			}
			if addr, err := netip.ParseAddr(address); err != nil || !addr.Is6() || addr.Zone() != "" {
				return "the host in brackets must be an IPv6 address"
			}
			outside = strings.Replace(rest, "["+address+"]", "", 1)
		}
	}
	if strings.ContainsAny(outside, "[]") {
		return problem
	}
	return ""
}

// isURIScheme reports whether s is an RFC 3986 scheme in lowercase: a
// letter followed by letters, digits, "+", "-" and ".".
func isURIScheme(s string) bool {
	if s == "" || s[0] < 'a' || s[0] > 'z' {
		return false
	}
	for i := 1; i < len(s); i++ {
		switch c := s[i]; {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '+', c == '-', c == '.':
		default:
			return false
		}
	}
	return true
}

// isURIChar reports whether c is an RFC 3986 unreserved or reserved
// character. "%" is handled separately, as the start of an escape.
func isURIChar(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	}
	return strings.IndexByte("-._~:/?#[]@!$&'()*+,;=", c) >= 0
}

func isHexDigit(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}
