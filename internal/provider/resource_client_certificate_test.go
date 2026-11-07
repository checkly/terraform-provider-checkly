package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
