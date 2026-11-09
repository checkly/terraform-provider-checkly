package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// emoji is a character outside the Basic Multilingual Plane: one code point,
// four UTF-8 bytes and two UTF-16 code units.
const emoji = "😀"

// validateString runs validators on value at path p.
func validateString(validators []validator.String, p path.Path, value types.String) diag.Diagnostics {
	req := validator.StringRequest{Path: p, ConfigValue: value}
	var resp validator.StringResponse
	for _, v := range validators {
		v.ValidateString(context.Background(), req, &resp)
	}
	return resp.Diagnostics
}

// validateStringAttribute runs the validators of the top-level string
// attribute name of r on value.
func validateStringAttribute(t *testing.T, r resource.Resource, name string, value types.String) diag.Diagnostics {
	t.Helper()

	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	attribute, ok := schemaResp.Schema.Attributes[name].(schema.StringAttribute)
	if !ok {
		t.Fatalf("%s is not a string attribute", name)
	}
	return validateString(attribute.Validators, path.Root(name), value)
}

func TestUTF16LengthBetween(t *testing.T) {
	cases := map[string]struct {
		value types.String
		valid bool
	}{
		"null":                            {types.StringNull(), true},
		"unknown":                         {types.StringUnknown(), true},
		"empty":                           {types.StringValue(""), false},
		"the longest ASCII":               {types.StringValue(strings.Repeat("a", 2000)), true},
		"ASCII too long":                  {types.StringValue(strings.Repeat("a", 2001)), false},
		"the longest two-byte characters": {types.StringValue(strings.Repeat("ä", 2000)), true},
		"the longest emoji":               {types.StringValue(strings.Repeat(emoji, 1000)), true},
		"emoji too long":                  {types.StringValue(strings.Repeat(emoji, 1000) + "a"), false},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			diags := validateString([]validator.String{utf16LengthBetween(1, 2000)}, path.Root("goal"), c.value)
			if diags.HasError() == c.valid {
				t.Errorf("valid = %v, diagnostics: %v", c.valid, diags)
			}
		})
	}
}

func TestUTF16LengthBetweenErrorGivesTheCount(t *testing.T) {
	diags := validateString([]validator.String{utf16LengthBetween(1, 3)}, path.Root("goal"), types.StringValue(emoji+emoji))
	if diags.ErrorsCount() != 1 || !strings.Contains(diags.Errors()[0].Detail(), "got: 4") {
		t.Errorf("diagnostics = %v", diags)
	}
}

func TestIsAPIWhitespace(t *testing.T) {
	cases := map[rune]bool{
		' ':     true,
		'\t':    true,
		'\n':    true,
		'\r':    true,
		'\v':    true,
		'\f':    true,
		0x00A0:  true,
		0xFEFF:  true,
		0x2028:  true,
		0x2029:  true,
		0x3000:  true,
		0x2003:  true,
		0x0085:  false,
		0x200B:  false,
		'a':     false,
		'-':     false,
		0x1F600: false,
		0x180E:  false,
	}

	for r, want := range cases {
		if got := isAPIWhitespace(r); got != want {
			t.Errorf("isAPIWhitespace(%U) = %v, want %v", r, got, want)
		}
	}
}

func TestPrivateLocationSlugValidator(t *testing.T) {
	cases := map[string]bool{
		"my-loc":           true,
		"lab-eu-central-1": true,
		"US-EAST-1":        true,
		"eu-central":       true,
		"MyLocation1":      true,
		"":                 false,
		"a_b":              false,
		"my loc":           false,
		"us-east-1":        false,
		"us-east-1x":       false,
		"eu-central-1-lab": false,
		"us-gov-west-1":    false,
		"ap-southeast-2":   false,
	}
	cases[strings.Repeat("a", 30)] = true
	cases[strings.Repeat("a", 31)] = false

	for value, valid := range cases {
		t.Run(value, func(t *testing.T) {
			diags := validateStringAttribute(t, NewPrivateLocationResource(), "slug_name", types.StringValue(value))
			if diags.HasError() == valid {
				t.Errorf("valid = %v, diagnostics: %v", valid, diags)
			}
		})
	}
}

func TestPrivateLocationSlugErrorNamesPublicLocations(t *testing.T) {
	diags := validateStringAttribute(t, NewPrivateLocationResource(), "slug_name", types.StringValue("eu-central-1-lab"))
	if !diags.HasError() || !strings.Contains(diags.Errors()[0].Detail(), "must not start with the ID of a public location") {
		t.Errorf("diagnostics = %v", diags)
	}
}

func TestStatusPageURLValidator(t *testing.T) {
	cases := map[string]bool{
		"a":              true,
		"my-status":      true,
		"status-page-1":  true,
		"":               false,
		"MyStatus":       false,
		"-status":        false,
		"status-":        false,
		"status_page":    false,
		"status.example": false,
	}
	cases[strings.Repeat("a", 63)] = true
	cases[strings.Repeat("a", 64)] = false

	for _, r := range []resource.Resource{NewStatusPageResource(), NewStatusPageV3Resource()} {
		for value, valid := range cases {
			t.Run(value, func(t *testing.T) {
				diags := validateStringAttribute(t, r, "url", types.StringValue(value))
				if diags.HasError() == valid {
					t.Errorf("valid = %v, diagnostics: %v", valid, diags)
				}
			})
		}
	}
}

func TestMaintenanceWindowLengthLimits(t *testing.T) {
	cases := []struct {
		attribute string
		value     types.String
		valid     bool
	}{
		{"name", types.StringValue(strings.Repeat("a", 200)), true},
		{"name", types.StringValue(strings.Repeat(emoji, 100)), true},
		{"name", types.StringValue(strings.Repeat("a", 201)), false},
		{"name", types.StringValue(strings.Repeat(emoji, 100) + "a"), false},
		{"name", types.StringValue(""), false},
		{"description", types.StringNull(), true},
		{"description", types.StringValue(""), true},
		{"description", types.StringValue(strings.Repeat("a", 2000)), true},
		{"description", types.StringValue(strings.Repeat("a", 2001)), false},
	}

	for _, r := range []resource.Resource{NewMaintenanceWindowResource(), NewMaintenanceWindowsResource()} {
		for _, c := range cases {
			if diags := validateStringAttribute(t, r, c.attribute, c.value); diags.HasError() == c.valid {
				t.Errorf("%T %s of %d bytes: valid = %v, diagnostics: %v", r, c.attribute, len(c.value.ValueString()), c.valid, diags)
			}
		}
	}
}

func TestClientCertificateLengthLimits(t *testing.T) {
	cases := []struct {
		attribute string
		value     string
		valid     bool
	}{
		{"host", strings.Repeat("a", 255), true},
		{"host", strings.Repeat("ä", 255), true},
		{"host", strings.Repeat("a", 256), false},
		{"passphrase", strings.Repeat("a", 95), true},
		{"passphrase", strings.Repeat("a", 96), false},
		// The passphrase limit counts UTF-8 bytes.
		{"passphrase", strings.Repeat("ä", 48), false},
	}

	for _, c := range cases {
		if diags := validateStringAttribute(t, NewClientCertificateResource(), c.attribute, types.StringValue(c.value)); diags.HasError() == c.valid {
			t.Errorf("%s of %d bytes: valid = %v, diagnostics: %v", c.attribute, len(c.value), c.valid, diags)
		}
	}
}

func TestClientCertificatePassphraseErrorOmitsValue(t *testing.T) {
	passphrase := strings.Repeat("s", 96)
	diags := validateStringAttribute(t, NewClientCertificateResource(), "passphrase", types.StringValue(passphrase))
	if diags.ErrorsCount() != 1 || strings.Contains(diags.Errors()[0].Detail(), passphrase) {
		t.Errorf("diagnostics = %v", diags)
	}
}

func TestStatusPageV3ComponentNameValidators(t *testing.T) {
	cases := map[string]struct {
		value string
		valid bool
	}{
		"a name":            {"API", true},
		"the longest":       {strings.Repeat("a", 255), true},
		"the longest emoji": {strings.Repeat(emoji, 127) + "a", true},
		"too long":          {strings.Repeat("a", 256), false},
		"emoji too long":    {strings.Repeat(emoji, 128), false},
		"empty":             {"", false},
		"a trailing space":  {"API ", false},
		"a leading newline": {"\nAPI", false},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			diags := validateStringAttribute(t, NewStatusPageV3ComponentResource(), "name", types.StringValue(c.value))
			if diags.HasError() == c.valid {
				t.Errorf("valid = %v, diagnostics: %v", c.valid, diags)
			}
		})
	}
}

// validateResourceConfig validates config, a configuration of the resource
// r of type typeName, through the provider server, so that every schema
// validator runs with the whole configuration. It returns the error
// diagnostics whose path starts at the attribute or block named root; the
// configurations lack required attributes, so there are errors about others
// too.
func validateResourceConfig(t *testing.T, r resource.Resource, typeName, root string, config func(objectType tftypes.Object) tftypes.Value) []*tfprotov6.Diagnostic {
	t.Helper()
	ctx := context.Background()

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	objectType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)

	value, err := tfprotov6.NewDynamicValue(objectType, config(objectType))
	if err != nil {
		t.Fatal(err)
	}

	server := providerserver.NewProtocol6(New("test")())()
	if _, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{}); err != nil {
		t.Fatal(err)
	}
	resp, err := server.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{TypeName: typeName, Config: &value})
	if err != nil {
		t.Fatal(err)
	}

	var errors []*tfprotov6.Diagnostic
	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError && d.Attribute != nil && len(d.Attribute.Steps()) > 0 &&
			d.Attribute.Steps()[0] == tftypes.AttributeName(root) {
			errors = append(errors, d)
		}
	}
	return errors
}

// environmentVariableValueCase is a value and flags of one environment
// variable. A flag is passed to tftypes.NewValue, so nil is null and
// tftypes.UnknownValue is unknown.
type environmentVariableValueCase struct {
	value          any
	secret, locked any
	valid          bool
}

var environmentVariableValueCases = map[string]environmentVariableValueCase{
	"the longest value":                {strings.Repeat("a", 64000), false, false, true},
	"a value too long":                 {strings.Repeat("a", 64001), false, false, false},
	"null flags":                       {strings.Repeat("a", 64000), nil, nil, true},
	"multi-byte characters as bytes":   {strings.Repeat("ä", 32001), nil, nil, false},
	"the longest secret":               {strings.Repeat("a", 31744), true, false, true},
	"a secret too long":                {strings.Repeat("a", 31745), true, false, false},
	"the longest locked value":         {strings.Repeat("a", 31744), false, true, true},
	"a locked value too long":          {strings.Repeat("a", 31745), nil, true, false},
	"unknown flags allow 64000":        {strings.Repeat("a", 64000), tftypes.UnknownValue, tftypes.UnknownValue, true},
	"unknown flags reject 64001":       {strings.Repeat("a", 64001), tftypes.UnknownValue, tftypes.UnknownValue, false},
	"a secret with an unknown lock":    {strings.Repeat("a", 31745), true, tftypes.UnknownValue, false},
	"an unknown secret with no lock":   {strings.Repeat("a", 31745), tftypes.UnknownValue, false, true},
	"a locked value of unknown secret": {strings.Repeat("a", 31745), tftypes.UnknownValue, true, false},
	"an unknown value":                 {tftypes.UnknownValue, true, nil, true},
}

// checkEnvironmentVariableValueErrors checks that errors match c and that
// an error is on wantPath and doesn't contain the value.
func checkEnvironmentVariableValueErrors(t *testing.T, c environmentVariableValueCase, errors []*tfprotov6.Diagnostic, wantPath *tftypes.AttributePath) {
	t.Helper()

	if (len(errors) == 0) != c.valid {
		t.Fatalf("valid = %v, errors: %v", c.valid, errors)
	}
	if c.valid {
		return
	}
	if !errors[0].Attribute.Equal(wantPath) {
		t.Errorf("error path = %s, want %s", errors[0].Attribute, wantPath)
	}
	if value, ok := c.value.(string); ok && strings.Contains(errors[0].Detail, value) {
		t.Errorf("error detail contains the value: %s", errors[0].Detail)
	}
}

func TestEnvironmentVariableBlockValueLimit(t *testing.T) {
	for name, c := range environmentVariableValueCases {
		t.Run(name, func(t *testing.T) {
			errors := validateResourceConfig(t, NewCheckGroupV2Resource(), "checkly_check_group_v2", environmentVariableAttributeName, func(objectType tftypes.Object) tftypes.Value {
				listType := objectType.AttributeTypes[environmentVariableAttributeName].(tftypes.List)
				return objectWith(objectType, map[string]tftypes.Value{
					environmentVariableAttributeName: tftypes.NewValue(listType, []tftypes.Value{
						objectWith(listType.ElementType.(tftypes.Object), map[string]tftypes.Value{
							"key":    tftypes.NewValue(tftypes.String, "KEY"),
							"value":  tftypes.NewValue(tftypes.String, c.value),
							"secret": tftypes.NewValue(tftypes.Bool, c.secret),
							"locked": tftypes.NewValue(tftypes.Bool, c.locked),
						}),
					}),
				})
			})
			checkEnvironmentVariableValueErrors(t, c, errors, tftypes.NewAttributePath().WithAttributeName(environmentVariableAttributeName).WithElementKeyInt(0).WithAttributeName("value"))
		})
	}
}

func TestEnvironmentVariableResourceValueLimit(t *testing.T) {
	for name, c := range environmentVariableValueCases {
		t.Run(name, func(t *testing.T) {
			errors := validateResourceConfig(t, NewEnvironmentVariableResource(), "checkly_environment_variable", "value", func(objectType tftypes.Object) tftypes.Value {
				return objectWith(objectType, map[string]tftypes.Value{
					"key":    tftypes.NewValue(tftypes.String, "KEY"),
					"value":  tftypes.NewValue(tftypes.String, c.value),
					"secret": tftypes.NewValue(tftypes.Bool, c.secret),
					"locked": tftypes.NewValue(tftypes.Bool, c.locked),
				})
			})
			checkEnvironmentVariableValueErrors(t, c, errors, tftypes.NewAttributePath().WithAttributeName("value"))
		})
	}
}

// validateResourceConfigWithEnvironmentVariables validates a configuration
// of the resource r of type typeName that sets only attributes and n
// environment_variable blocks with the given value. It returns the error
// diagnostics about the environment_variable blocks.
func validateResourceConfigWithEnvironmentVariables(t *testing.T, r resource.Resource, typeName string, attributes map[string]tftypes.Value, n int, value string) []*tfprotov6.Diagnostic {
	t.Helper()

	return validateResourceConfig(t, r, typeName, environmentVariableAttributeName, func(objectType tftypes.Object) tftypes.Value {
		listType := objectType.AttributeTypes[environmentVariableAttributeName].(tftypes.List)
		variables := make([]tftypes.Value, n)
		for i := range variables {
			variables[i] = objectWith(listType.ElementType.(tftypes.Object), map[string]tftypes.Value{
				"key":   tftypes.NewValue(tftypes.String, "KEY"),
				"value": tftypes.NewValue(tftypes.String, value),
			})
		}
		values := map[string]tftypes.Value{environmentVariableAttributeName: tftypes.NewValue(listType, variables)}
		for name, value := range attributes {
			values[name] = value
		}
		return objectWith(objectType, values)
	})
}

func TestEnvironmentVariableBlockLimits(t *testing.T) {
	cases := []struct {
		resource   resource.Resource
		typeName   string
		attributes map[string]tftypes.Value
	}{
		{NewCheckResource(), "checkly_check", map[string]tftypes.Value{"type": tftypes.NewValue(tftypes.String, "BROWSER")}},
		{NewBrowserCheckResource(), "checkly_browser_check", nil},
		{NewCheckGroupResource(), "checkly_check_group", nil},
		{NewCheckGroupV2Resource(), "checkly_check_group_v2", nil},
		{NewMultistepCheckResource(), "checkly_multistep_check", nil},
		{NewPlaywrightCheckSuiteResource(), "checkly_playwright_check_suite", nil},
	}

	for _, c := range cases {
		t.Run(c.typeName, func(t *testing.T) {
			if errors := validateResourceConfigWithEnvironmentVariables(t, c.resource, c.typeName, c.attributes, 200, "value"); len(errors) > 0 {
				t.Errorf("200 variables: %v", errors[0])
			}
			if errors := validateResourceConfigWithEnvironmentVariables(t, c.resource, c.typeName, c.attributes, 201, "value"); len(errors) == 0 {
				t.Error("201 variables: no error")
			}
			if errors := validateResourceConfigWithEnvironmentVariables(t, c.resource, c.typeName, c.attributes, 1, strings.Repeat("a", 64001)); len(errors) == 0 {
				t.Error("a value too long: no error")
			}
		})
	}
}
