package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// utf16LengthValidator rejects strings whose length in UTF-16 code units is
// outside [min, max]. The API validates lengths in JavaScript, whose string
// length counts UTF-16 code units, so a character outside the Basic
// Multilingual Plane, such as most emoji, counts as two.
type utf16LengthValidator struct {
	min, max int
}

var _ validator.String = utf16LengthValidator{}

// utf16LengthBetween returns a validator that accepts strings of min to max
// UTF-16 code units.
func utf16LengthBetween(min, max int) validator.String {
	return utf16LengthValidator{min: min, max: max}
}

// utf16Length returns the number of UTF-16 code units that encode s.
func utf16Length(s string) int {
	n := 0
	for _, r := range s {
		n += utf16.RuneLen(r)
	}
	return n
}

func (v utf16LengthValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("value must be %d to %d UTF-16 code units long", v.min, v.max)
}

func (v utf16LengthValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v utf16LengthValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	if n := utf16Length(req.ConfigValue.ValueString()); n < v.min || n > v.max {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Attribute Value Length",
			fmt.Sprintf("%s must be %d to %d UTF-16 code units long, got: %d. Most characters count as one; emoji and other characters outside the Basic Multilingual Plane count as two.", req.Path, v.min, v.max, n),
		)
	}
}

// isAPIWhitespace reports whether JavaScript's String.prototype.trim, which
// the API uses, removes r: the WhiteSpace and LineTerminator characters of
// ECMAScript. Unlike unicode.IsSpace, it includes U+FEFF and excludes U+0085.
func isAPIWhitespace(r rune) bool {
	switch r {
	case '\t', '\v', '\f', ' ', '\u00A0', '\uFEFF', '\n', '\r', '\u2028', '\u2029':
		return true
	}
	return unicode.Is(unicode.Zs, r)
}

// trimmedStringValidator rejects leading and trailing whitespace as the API's
// trim defines it (see isAPIWhitespace). The API trims these values, so an
// untrimmed one, such as a heredoc with its final newline, would read back
// different from the configuration.
type trimmedStringValidator struct{}

var _ validator.String = trimmedStringValidator{}

func (v trimmedStringValidator) Description(ctx context.Context) string {
	return "value must not have leading or trailing whitespace"
}

func (v trimmedStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v trimmedStringValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	if trimmed := strings.TrimFunc(value, isAPIWhitespace); value != trimmed {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Attribute Value",
			fmt.Sprintf("%s must not have leading or trailing whitespace, which the API removes. Use trimspace() to remove it, for example on a heredoc.", req.Path),
		)
	}
}

// noControlCharactersRegex is the API's rule for host names sent to a
// network probe: no C0 control characters (U+0000 to U+001F) and no DEL
// (U+007F). C1 control characters, such as U+0085, are accepted.
var noControlCharactersRegex = regexp.MustCompile(`^[^\x00-\x1f\x7f]*$`)

// noControlCharacters returns a validator that rejects strings with
// characters that noControlCharactersRegex excludes.
func noControlCharacters() validator.String {
	return stringvalidator.RegexMatches(noControlCharactersRegex, "must not contain control characters")
}

// noNULCharacterValidator rejects strings that contain U+0000 (NUL), which
// the API rejects in some free-text fields where other control characters,
// such as tabs and newlines, are valid. The error doesn't repeat the value,
// which can be long, such as the source of a .proto file, or secret, such as
// a metadata value. Nor does it name the path: the path of an attribute in a
// set element includes the whole element, values included.
type noNULCharacterValidator struct{}

var _ validator.String = noNULCharacterValidator{}

func (v noNULCharacterValidator) Description(ctx context.Context) string {
	return "value must not contain NUL characters (U+0000)"
}

func (v noNULCharacterValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v noNULCharacterValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	if strings.ContainsRune(req.ConfigValue.ValueString(), 0) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Attribute Value",
			"The value must not contain a NUL character (U+0000), which the API rejects.",
		)
	}
}

// The API's private location slug rule: 1 to 30 letters, digits and dashes,
// not starting with the ID of a public location, such as "eu-central-1". Go
// regular expressions have no lookahead, so the rule is two expressions,
// checked by privateLocationSlugValidators.
var (
	privateLocationSlugRegex             = regexp.MustCompile(`^[a-zA-Z0-9-]{1,30}$`)
	privateLocationPublicLocationIDRegex = regexp.MustCompile(`^(us(-gov)?|ap|ca|cn|eu|sa|af|me)-(central|(north|south)?(east|west)?)-\d+`)
)

// privateLocationSlugValidators reject slugs the API doesn't accept for a
// private location.
var privateLocationSlugValidators = []validator.String{
	stringvalidator.RegexMatches(privateLocationSlugRegex, "must be 1 to 30 letters, digits and dashes"),
	predicateStringValidator{
		check:       func(value string) bool { return !privateLocationPublicLocationIDRegex.MatchString(value) },
		message:     "%q must not start with the ID of a public location, such as \"eu-central-1\", got: %s",
		description: "value must not start with the ID of a public location, such as \"eu-central-1\"",
	},
}

const (
	// maxEnvironmentVariables is the most environment variables the API
	// accepts on a check or check group.
	maxEnvironmentVariables = 200
	// maxEnvironmentVariableValueBytes is the API's limit on the UTF-8
	// length of an environment variable value.
	maxEnvironmentVariableValueBytes = 64000
	// maxSecretEnvironmentVariableValueBytes is the API's limit on the
	// UTF-8 length of a secret or locked environment variable value.
	maxSecretEnvironmentVariableValueBytes = 31 * 1024
	// minSecretEnvironmentVariableValueCharacters is the API's minimum
	// length of a secret environment variable value, in Unicode code
	// points. Runners mask every occurrence of a secret value in the check
	// output, so a shorter value would mask most of it.
	minSecretEnvironmentVariableValueCharacters = 3
)

// environmentVariableValueDescription states the API's limits on an
// environment variable value, for attribute descriptions.
var environmentVariableValueDescription = fmt.Sprintf("At most %d bytes in UTF-8, or %d bytes for a secret or locked variable. A secret value must have at least %d characters.", maxEnvironmentVariableValueBytes, maxSecretEnvironmentVariableValueBytes, minSecretEnvironmentVariableValueCharacters)

// environmentVariableValueValidator rejects an environment variable value
// longer than the API accepts. The sibling secret and locked attributes pick
// the limit: a flag that is null counts as false, and one that is unknown
// counts as false too, so only the larger limit is enforced until it is
// known. The error gives lengths only, because the value is sensitive.
//
// The minimum length of a secret value is checked at plan time instead (see
// checkSecretEnvironmentVariableValue), because whether the API checks it
// depends on the prior state.
type environmentVariableValueValidator struct{}

var _ validator.String = environmentVariableValueValidator{}

func (v environmentVariableValueValidator) Description(ctx context.Context) string {
	return environmentVariableValueDescription
}

func (v environmentVariableValueValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v environmentVariableValueValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	var secret, locked types.Bool
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, req.Path.ParentPath().AtName("secret"), &secret)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, req.Path.ParentPath().AtName("locked"), &locked)...)
	if resp.Diagnostics.HasError() {
		return
	}

	limit, kind := maxEnvironmentVariableValueBytes, "an environment variable"
	if secret.ValueBool() || locked.ValueBool() {
		limit, kind = maxSecretEnvironmentVariableValueBytes, "a secret or locked environment variable"
	}
	if n := len(req.ConfigValue.ValueString()); n > limit {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Environment Variable Value Too Long",
			fmt.Sprintf("%s is %d bytes long in UTF-8, but the API accepts at most %d bytes for the value of %s.", req.Path, n, limit, kind),
		)
	}
}

// checkSecretEnvironmentVariableValue reports a secret value at p that is
// shorter than the API accepts. The API counts code points, as
// utf8.RuneCountInString does, and an empty secret value is too short too.
// It checks the value only when it writes it: sent is true when the plan
// creates the variable or sends a new value, and the short value is then an
// error. Otherwise the API keeps the stored value as it is, so the plan only
// warns, and the same configuration still fails when it creates the variable
// anew. The diagnostics give lengths only, because the value is sensitive.
func checkSecretEnvironmentVariableValue(diags *diag.Diagnostics, p path.Path, value string, sent bool) {
	n := utf8.RuneCountInString(value)
	if n >= minSecretEnvironmentVariableValueCharacters {
		return
	}

	const summary = "Secret Environment Variable Value Too Short"
	reason := fmt.Sprintf("%s is %d characters long, but the API requires at least %d characters for the value of a secret environment variable, because runners mask every occurrence of a secret value in the check output.", p, n, minSecretEnvironmentVariableValueCharacters)
	if sent {
		diags.AddAttributeError(p, summary, reason)
		return
	}
	diags.AddAttributeWarning(p, summary, reason+" The API keeps a stored value equal to it, but rejects it on create, when it changes, and when the value stored in Checkly differs, for example after a change outside Terraform, which the plan can't see. Use a longer value.")
}
