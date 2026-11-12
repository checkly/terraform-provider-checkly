package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccClientCertificateCheckRequiredFields(t *testing.T) {
	config := `resource "checkly_client_certificate" "test" {}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "host" is required`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "certificate" is required`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "private_key" is required`),
		},
	})
}

func TestAccClientCertificate(t *testing.T) {
	config := `
resource "checkly_client_certificate" "test" {
  host = "*.acme.com"
  path = "/partner/api/"

  certificate = <<-EOT
			-----BEGIN CERTIFICATE-----
			MIICDzCCAbagAwIBAgIUMTZlfGA7WcD8e4/zt2MqxvEgQPYwCgYIKoZIzj0EAwIw
			VDELMAkGA1UEBhMCVVMxCzAJBgNVBAgMAkNBMREwDwYDVQQHDAhUb29udG93bjES
			MBAGA1UECgwJQWNtZSBJbmMuMREwDwYDVQQDDAhhY21lLmNvbTAeFw0yNTAzMDMw
			NTQ2NTJaFw00OTEwMjMwNTQ2NTJaMHgxCzAJBgNVBAYTAlVTMQswCQYDVQQIDAJD
			QTERMA8GA1UEBwwIVG9vbnRvd24xEjAQBgNVBAoMCUFjbWUgSW5jLjEXMBUGA1UE
			AwwOV2lsZSBFLiBDb3lvdGUxHDAaBgkqhkiG9w0BCQEWDXdpbGVAYWNtZS5jb20w
			WTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAATAjjDGsKFS1qgdNqziDZoD5hamTfdH
			0P+Ukk1RIue57QYVXhQSyNzcEz15kQnwYezEqfN+FtjtTwdk/CgnAELlo0IwQDAd
			BgNVHQ4EFgQU9C9CpZqM2WMrOs3vAYsc5GbjyzswHwYDVR0jBBgwFoAUnlOyzF/N
			K7YmKQegLdbdyIOCT/UwCgYIKoZIzj0EAwIDRwAwRAIgGgSnBymlH4MkZCVk5DYH
			PdnDo2Xf5uFi1Eyn2LTYP1MCIEtiGtsf0qYv6NzIPd5uTTZoB/8hPrAgM1QzWG4O
			3C/I
			-----END CERTIFICATE-----
		EOT

  private_key = <<-EOT
			-----BEGIN ENCRYPTED PRIVATE KEY-----
			MIH0MF8GCSqGSIb3DQEFDTBSMDEGCSqGSIb3DQEFDDAkBBA5yR3aqy8mZD2wQzp1
			FH2JAgIIADAMBggqhkiG9w0CCQUAMB0GCWCGSAFlAwQBKgQQA49YCnXvfJ2CsQsV
			9C5JJwSBkNkWunSlqyeVW6OFa/+OjlLArgTGvW5ul08qu/145O9PO4Nr2CXeK5N2
			uvHwkWGfD8IVke+sgZPUjLoHsJ4h4AnyxlNHpIxgOfm0CoXT7PTaFb//d5NC6XyB
			K7ZpBzIThGlbuS/b9wp4MPmSaJn5Fci+84VG7KYK5RxU0fcU0rGSBynrZw803wnO
			FjP7qaq5bw==
			-----END ENCRYPTED PRIVATE KEY-----
		EOT

  passphrase = "secret password"

  trusted_ca = <<-EOT
			-----BEGIN CERTIFICATE-----
			MIIB/jCCAaOgAwIBAgIUZzxdNpoDYXaNiIBsh0/s++I+ZOEwCgYIKoZIzj0EAwIw
			VDELMAkGA1UEBhMCVVMxCzAJBgNVBAgMAkNBMREwDwYDVQQHDAhUb29udG93bjES
			MBAGA1UECgwJQWNtZSBJbmMuMREwDwYDVQQDDAhhY21lLmNvbTAeFw0yNTAzMDMw
			NTQzMDZaFw0yNTA0MDIwNTQzMDZaMFQxCzAJBgNVBAYTAlVTMQswCQYDVQQIDAJD
			QTERMA8GA1UEBwwIVG9vbnRvd24xEjAQBgNVBAoMCUFjbWUgSW5jLjERMA8GA1UE
			AwwIYWNtZS5jb20wWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAARDH3KGK6Vsk1A4
			yGf9ItQIS3yuAOi0n0ihmPzIOOOEN0c758ETABeUdgH55bakdx6q5KYSxf4TuXsJ
			2nCihqVVo1MwUTAdBgNVHQ4EFgQUnlOyzF/NK7YmKQegLdbdyIOCT/UwHwYDVR0j
			BBgwFoAUnlOyzF/NK7YmKQegLdbdyIOCT/UwDwYDVR0TAQH/BAUwAwEB/zAKBggq
			hkjOPQQDAgNJADBGAiEA/cJ9jV8MQz4ypQsFvUatrnbxyHO0f+pJhf09pAk6Kj8C
			IQCkSbope5r0KlVdqBeFF8wCfE3plwpelve3jqVIz6MedQ==
			-----END CERTIFICATE-----
		EOT
}`
	// A second resource for the same certificate, which the steps below
	// import. Destroying both is fine: deleting a certificate that is
	// already gone succeeds.
	importedConfig := config + strings.Replace(config,
		`resource "checkly_client_certificate" "test"`,
		`resource "checkly_client_certificate" "imported"`, 1)
	accTestCase(t, []resource.TestStep{
		{
			Config: config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_client_certificate.test",
					"host",
					"*.acme.com",
				),
				// The API drops the trailing slash; state keeps the configured
				// spelling so the plan stays empty.
				resource.TestCheckResourceAttr(
					"checkly_client_certificate.test",
					"path",
					"/partner/api/",
				),
				resource.TestCheckResourceAttr(
					"checkly_client_certificate.test",
					"certificate",
					"-----BEGIN CERTIFICATE-----\n"+
						"MIICDzCCAbagAwIBAgIUMTZlfGA7WcD8e4/zt2MqxvEgQPYwCgYIKoZIzj0EAwIw\n"+
						"VDELMAkGA1UEBhMCVVMxCzAJBgNVBAgMAkNBMREwDwYDVQQHDAhUb29udG93bjES\n"+
						"MBAGA1UECgwJQWNtZSBJbmMuMREwDwYDVQQDDAhhY21lLmNvbTAeFw0yNTAzMDMw\n"+
						"NTQ2NTJaFw00OTEwMjMwNTQ2NTJaMHgxCzAJBgNVBAYTAlVTMQswCQYDVQQIDAJD\n"+
						"QTERMA8GA1UEBwwIVG9vbnRvd24xEjAQBgNVBAoMCUFjbWUgSW5jLjEXMBUGA1UE\n"+
						"AwwOV2lsZSBFLiBDb3lvdGUxHDAaBgkqhkiG9w0BCQEWDXdpbGVAYWNtZS5jb20w\n"+
						"WTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAATAjjDGsKFS1qgdNqziDZoD5hamTfdH\n"+
						"0P+Ukk1RIue57QYVXhQSyNzcEz15kQnwYezEqfN+FtjtTwdk/CgnAELlo0IwQDAd\n"+
						"BgNVHQ4EFgQU9C9CpZqM2WMrOs3vAYsc5GbjyzswHwYDVR0jBBgwFoAUnlOyzF/N\n"+
						"K7YmKQegLdbdyIOCT/UwCgYIKoZIzj0EAwIDRwAwRAIgGgSnBymlH4MkZCVk5DYH\n"+
						"PdnDo2Xf5uFi1Eyn2LTYP1MCIEtiGtsf0qYv6NzIPd5uTTZoB/8hPrAgM1QzWG4O\n"+
						"3C/I\n"+
						"-----END CERTIFICATE-----\n",
				),
				resource.TestCheckResourceAttr(
					"checkly_client_certificate.test",
					"private_key",
					"-----BEGIN ENCRYPTED PRIVATE KEY-----\n"+
						"MIH0MF8GCSqGSIb3DQEFDTBSMDEGCSqGSIb3DQEFDDAkBBA5yR3aqy8mZD2wQzp1\n"+
						"FH2JAgIIADAMBggqhkiG9w0CCQUAMB0GCWCGSAFlAwQBKgQQA49YCnXvfJ2CsQsV\n"+
						"9C5JJwSBkNkWunSlqyeVW6OFa/+OjlLArgTGvW5ul08qu/145O9PO4Nr2CXeK5N2\n"+
						"uvHwkWGfD8IVke+sgZPUjLoHsJ4h4AnyxlNHpIxgOfm0CoXT7PTaFb//d5NC6XyB\n"+
						"K7ZpBzIThGlbuS/b9wp4MPmSaJn5Fci+84VG7KYK5RxU0fcU0rGSBynrZw803wnO\n"+
						"FjP7qaq5bw==\n"+
						"-----END ENCRYPTED PRIVATE KEY-----\n",
				),
				resource.TestCheckResourceAttr(
					"checkly_client_certificate.test",
					"passphrase",
					"secret password",
				),
				resource.TestCheckResourceAttr(
					"checkly_client_certificate.test",
					"trusted_ca",
					"-----BEGIN CERTIFICATE-----\n"+
						"MIIB/jCCAaOgAwIBAgIUZzxdNpoDYXaNiIBsh0/s++I+ZOEwCgYIKoZIzj0EAwIw\n"+
						"VDELMAkGA1UEBhMCVVMxCzAJBgNVBAgMAkNBMREwDwYDVQQHDAhUb29udG93bjES\n"+
						"MBAGA1UECgwJQWNtZSBJbmMuMREwDwYDVQQDDAhhY21lLmNvbTAeFw0yNTAzMDMw\n"+
						"NTQzMDZaFw0yNTA0MDIwNTQzMDZaMFQxCzAJBgNVBAYTAlVTMQswCQYDVQQIDAJD\n"+
						"QTERMA8GA1UEBwwIVG9vbnRvd24xEjAQBgNVBAoMCUFjbWUgSW5jLjERMA8GA1UE\n"+
						"AwwIYWNtZS5jb20wWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAARDH3KGK6Vsk1A4\n"+
						"yGf9ItQIS3yuAOi0n0ihmPzIOOOEN0c758ETABeUdgH55bakdx6q5KYSxf4TuXsJ\n"+
						"2nCihqVVo1MwUTAdBgNVHQ4EFgQUnlOyzF/NK7YmKQegLdbdyIOCT/UwHwYDVR0j\n"+
						"BBgwFoAUnlOyzF/NK7YmKQegLdbdyIOCT/UwDwYDVR0TAQH/BAUwAwEB/zAKBggq\n"+
						"hkjOPQQDAgNJADBGAiEA/cJ9jV8MQz4ypQsFvUatrnbxyHO0f+pJhf09pAk6Kj8C\n"+
						"IQCkSbope5r0KlVdqBeFF8wCfE3plwpelve3jqVIz6MedQ==\n"+
						"-----END CERTIFICATE-----\n",
				),
			),
		},
		{
			ResourceName:      "checkly_client_certificate.test",
			ImportState:       true,
			ImportStateVerify: true,
			// The API never returns the private key or its passphrase, and it
			// stores the path without the configured trailing slash.
			ImportStateVerifyIgnore: []string{"private_key", "passphrase", "path"},
			// With no prior state to keep the configured spelling from, the
			// imported path is the API's normalized one.
			ImportStateCheck: func(states []*terraform.InstanceState) error {
				if len(states) != 1 {
					return fmt.Errorf("expected 1 imported state, got %d", len(states))
				}
				if got := states[0].Attributes["path"]; got != "/partner/api" {
					return fmt.Errorf("imported path = %q, want %q", got, "/partner/api")
				}
				return nil
			},
		},
		{
			// The API never returns the private key or its passphrase, so the
			// first plan after an import stores the configured ones in place
			// instead of replacing the certificate.
			ResourceName:       "checkly_client_certificate.test",
			ImportState:        true,
			ImportStateKind:    resource.ImportBlockWithID,
			ExpectNonEmptyPlan: true,
			ImportPlanChecks: resource.ImportPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction("checkly_client_certificate.test", plancheck.ResourceActionUpdate),
				},
			},
		},
		{
			// Import the same certificate at a second address and keep it in
			// state, so the next step can apply the configuration to it.
			Config:             importedConfig,
			ResourceName:       "checkly_client_certificate.imported",
			ImportState:        true,
			ImportStatePersist: true,
			ImportStateIdFunc: func(s *terraform.State) (string, error) {
				r, ok := s.RootModule().Resources["checkly_client_certificate.test"]
				if !ok {
					return "", fmt.Errorf("checkly_client_certificate.test not found in state")
				}
				return r.Primary.ID, nil
			},
		},
		{
			// The first apply after the import records the configured key and
			// passphrase in state, after which the plan is empty.
			Config: importedConfig,
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction("checkly_client_certificate.imported", plancheck.ResourceActionUpdate),
				},
				PostApplyPostRefresh: []plancheck.PlanCheck{
					plancheck.ExpectEmptyPlan(),
				},
			},
		},
	})
}

func TestClientCertificatePathRequiresReplace(t *testing.T) {
	ctx := context.Background()

	var schemaResp fwresource.SchemaResponse
	NewClientCertificateResource().Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	pathAttr := schemaResp.Schema.Attributes["path"].(schema.StringAttribute)

	cases := []struct {
		state, plan types.String
		want        bool
	}{
		{types.StringValue("/partner/api"), types.StringValue("/partner/api/"), false},
		{types.StringValue("/partner/api/"), types.StringValue("/partner/api"), false},
		{types.StringValue("/partner/api"), types.StringValue("/partner"), true},
		{types.StringNull(), types.StringValue("/partner"), true},
		{types.StringValue("/partner"), types.StringNull(), true},
		{types.StringNull(), types.StringUnknown(), true},
	}

	for _, c := range cases {
		req := planmodifier.StringRequest{
			StateValue:  c.state,
			PlanValue:   c.plan,
			ConfigValue: c.plan,
			State:       tfsdk.State{Raw: tftypes.NewValue(tftypes.Object{}, map[string]tftypes.Value{})},
			Plan:        tfsdk.Plan{Raw: tftypes.NewValue(tftypes.Object{}, map[string]tftypes.Value{})},
		}
		resp := planmodifier.StringResponse{PlanValue: c.plan}
		for _, m := range pathAttr.PlanModifiers {
			m.PlanModifyString(ctx, req, &resp)
		}
		if resp.RequiresReplace != c.want {
			t.Errorf("path %s -> %s: RequiresReplace = %t, want %t", c.state, c.plan, resp.RequiresReplace, c.want)
		}
	}
}

func TestClientCertificateKeyRequiresReplace(t *testing.T) {
	ctx := context.Background()

	var schemaResp fwresource.SchemaResponse
	NewClientCertificateResource().Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)

	// statePrivateKey is the private key in prior state for passphrase rows;
	// private_key rows use their own state value.
	cases := []struct {
		name                string
		attribute           string
		state, plan         types.String
		statePrivateKey     types.String
		wantRequiresReplace bool
	}{
		{"changed private key", "private_key", types.StringValue("old key"), types.StringValue("new key"), types.String{}, true},
		{"unknown private key", "private_key", types.StringValue("old key"), types.StringUnknown(), types.String{}, true},
		{"private key after import", "private_key", types.StringNull(), types.StringValue("key"), types.String{}, false},
		{"unknown private key after import", "private_key", types.StringNull(), types.StringUnknown(), types.String{}, false},
		{"private key after a v1.x import", "private_key", types.StringValue(""), types.StringValue("key"), types.String{}, false},
		{"added passphrase", "passphrase", types.StringNull(), types.StringValue("x"), types.StringValue("key"), true},
		{"changed passphrase", "passphrase", types.StringValue("x"), types.StringValue("y"), types.StringValue("key"), true},
		{"emptied passphrase", "passphrase", types.StringNull(), types.StringValue(""), types.StringValue("key"), false},
		{"passphrase after import", "passphrase", types.StringNull(), types.StringValue("x"), types.StringNull(), false},
		{"passphrase after a v1.x import", "passphrase", types.StringNull(), types.StringValue("x"), types.StringValue(""), false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			model := ClientCertificateResourceModel{
				ID:          types.StringValue("1"),
				Host:        types.StringValue("acme.com"),
				Path:        types.StringNull(),
				Certificate: types.StringValue("certificate"),
				PrivateKey:  c.state,
				Passphrase:  types.StringNull(),
				TrustedCA:   types.StringNull(),
			}
			if c.attribute == "passphrase" {
				model.PrivateKey = c.statePrivateKey
				model.Passphrase = c.state
			}
			state := tfsdk.State{Schema: schemaResp.Schema}
			if diags := state.Set(ctx, &model); diags.HasError() {
				t.Fatalf("setting state: %v", diags)
			}
			plan := tfsdk.Plan{Schema: schemaResp.Schema, Raw: state.Raw}

			req := planmodifier.StringRequest{
				Path:        path.Root(c.attribute),
				StateValue:  c.state,
				PlanValue:   c.plan,
				ConfigValue: c.plan,
				State:       state,
				Plan:        plan,
			}
			resp := planmodifier.StringResponse{PlanValue: c.plan}
			for _, m := range schemaResp.Schema.Attributes[c.attribute].(schema.StringAttribute).PlanModifiers {
				m.PlanModifyString(ctx, req, &resp)
			}
			if resp.Diagnostics.HasError() {
				t.Fatalf("plan modifiers: %v", resp.Diagnostics)
			}
			if resp.RequiresReplace != c.wantRequiresReplace {
				t.Errorf("RequiresReplace = %t, want %t", resp.RequiresReplace, c.wantRequiresReplace)
			}
		})
	}
}

func TestClientCertificatePathProblem(t *testing.T) {
	valid := []string{
		"",
		"/partner",
		"/partner/api",
		"/partner/api/",
		"/partner%20api",
		"/caf%C3%A9",
		"/a-b_c.d~e/f!$&'()*+,;=:@g",
		"/v1.2/...x/.hidden",
		"/a|b/[c]",
		"/a%zz",
		"/50%",
		"/" + strings.Repeat("a", clientCertificatePathMaxLength-1),
	}
	for _, p := range valid {
		if problem := clientCertificatePathProblem(p); problem != "" {
			t.Errorf("%q: unexpected problem %q", p, problem)
		}
	}

	invalid := map[string]string{
		"partner":      `must start with "/"`,
		" /partner":    `must start with "/"`,
		"/":            `must name a path below "/"`,
		"//":           `must not contain empty segments`,
		"/a//b":        `must not contain empty segments`,
		"/a//":         `must not contain empty segments`,
		"/a/./b":       `must not contain "." or ".." segments`,
		"/a/..":        `must not contain "." or ".." segments`,
		"/a/%2E%2e/b":  `must not contain "." or ".." segments`,
		"/partner api": "must not contain whitespace",
		"/partner ":    "must not contain whitespace",
		"/a\tb":        "must not contain whitespace",
		"/a\x7fb":      "must not contain whitespace",
		"/a?b":         "must not contain whitespace",
		"/a#b":         "must not contain whitespace",
		"/a\\b":        "must not contain whitespace",
		"/café":        "percent-encoded",
		"/{id}":        "percent-encoded",
		"/a^b":         "percent-encoded",
		"/a`b":         "percent-encoded",
		"/a\"b":        "percent-encoded",
		"/a<b>":        "percent-encoded",
		"/" + strings.Repeat("a", clientCertificatePathMaxLength): "must be at most",
	}
	for p, want := range invalid {
		if problem := clientCertificatePathProblem(p); !strings.Contains(problem, want) {
			t.Errorf("%q: problem = %q, want it to contain %q", p, problem, want)
		}
	}
}

func TestClientCertificateAttributeValidators(t *testing.T) {
	var schemaResp fwresource.SchemaResponse
	NewClientCertificateResource().Schema(context.Background(), fwresource.SchemaRequest{}, &schemaResp)

	runAttributeValidationCases(t, schemaResp.Schema, []attributeValidationCase{
		{[]string{"certificate"}, types.StringValue(""), false},
		{[]string{"private_key"}, types.StringValue(""), false},
		{[]string{"path"}, types.StringValue("/partner/api/"), true},
		{[]string{"path"}, types.StringValue("/partner api"), false},
	})
}
