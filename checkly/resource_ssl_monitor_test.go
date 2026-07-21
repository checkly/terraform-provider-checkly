package checkly

import (
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// TestSSLRequestSecurityBaselineUnknownKeyRejected is a unit test (no TF_ACC)
// asserting that an unknown security_baseline rule key produces an error.
// Silently dropping it — as plain json.Unmarshal would — means a misspelled
// SSL security control is never sent to the API and thus never enforced.
func TestSSLRequestSecurityBaselineUnknownKeyRejected(t *testing.T) {
	request := func(baseline string) []any {
		return []any{tfMap{
			"hostname":                 "example.com",
			"port":                     443,
			"ip_family":                "IPv4",
			"skip_chain_validation":    false,
			"handshake_timeout_ms":     10000,
			"alert_days_before_expiry": 20,
			"security_baseline":        baseline,
			"assertion":                schema.NewSet(func(any) int { return 0 }, nil),
		}}
	}

	_, err := sslRequestFromList(request(`{"minKeySizeBts":{"value":2048,"severity":"fail"}}`))
	if err == nil || !strings.Contains(err.Error(), "security_baseline") {
		t.Fatalf("expected a security_baseline decode error for an unknown rule key, got: %v", err)
	}

	if _, err := sslRequestFromList(request(`{"minKeySizeBits":{"value":2048,"severity":"fail"}}`)); err != nil {
		t.Fatalf("expected a valid baseline to decode, got: %v", err)
	}
}

func TestAccSSLMonitorRequiredFields(t *testing.T) {
	config := `resource "checkly_ssl_monitor" "test" {}`
	accTestCase(t, []resource.TestStep{
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "name" is required, but no definition was found.`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "activated" is required, but no definition was found.`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`The argument "frequency" is required, but no definition was found.`),
		},
		{
			Config:      config,
			ExpectError: regexp.MustCompile(`At least 1 "request" blocks are required.`),
		},
	})
}

func TestAccSSLMonitorBasic(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: sslMonitor_basic,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_ssl_monitor.test",
					"name",
					"SSL Monitor 1",
				),
				resource.TestCheckResourceAttr(
					"checkly_ssl_monitor.test",
					"activated",
					"true",
				),
				resource.TestCheckResourceAttr(
					"checkly_ssl_monitor.test",
					"frequency",
					"60",
				),
				testCheckResourceAttrExpr(
					"checkly_ssl_monitor.test",
					"locations.*",
					"us-east-1",
				),
				testCheckResourceAttrExpr(
					"checkly_ssl_monitor.test",
					"request.*.hostname",
					"api.checklyhq.com",
				),
				testCheckResourceAttrExpr(
					"checkly_ssl_monitor.test",
					"request.*.alert_days_before_expiry",
					"30",
				),
			),
		},
	})
}

func TestAccSSLMonitorFull(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: sslMonitor_full,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_ssl_monitor.test",
					"name",
					"sslMonitor_full",
				),
				testCheckResourceAttrExpr(
					"checkly_ssl_monitor.test",
					`"locations.#"`,
					"2",
				),
				testCheckResourceAttrExpr(
					"checkly_ssl_monitor.test",
					"request.*.server_name",
					"api.checklyhq.com",
				),
				testCheckResourceAttrExpr(
					"checkly_ssl_monitor.test",
					"request.*.handshake_timeout_ms",
					"10000",
				),
				resource.TestCheckResourceAttr(
					"checkly_ssl_monitor.test",
					"degraded_response_time",
					"3000",
				),
				testCheckResourceAttrExpr(
					"checkly_ssl_monitor.test",
					"request.*.client_certificate.*.mode",
					"account_default",
				),
				resource.TestCheckResourceAttr(
					"checkly_ssl_monitor.test",
					"request.0.assertion.#",
					"2",
				),
				resource.TestCheckTypeSetElemNestedAttrs(
					"checkly_ssl_monitor.test",
					"request.0.assertion.*",
					map[string]string{
						"source":     "CERTIFICATE",
						"property":   "daysUntilExpiry",
						"comparison": "GREATER_THAN",
						"target":     "14",
					},
				),
				resource.TestCheckTypeSetElemNestedAttrs(
					"checkly_ssl_monitor.test",
					"request.0.assertion.*",
					map[string]string{
						"source":     "CONNECTION",
						"property":   "hostnameVerified",
						"comparison": "EQUALS",
						"target":     "true",
					},
				),
			),
		},
	})
}

// TestAccSSLMonitorMinimalCleanReplan asserts anti-pattern B is avoided: a
// config omitting every optional field (including the computed security_baseline
// and client_certificate blocks) applies, then re-plans with no diff.
func TestAccSSLMonitorMinimalCleanReplan(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: sslMonitor_minimal,
		},
		{
			Config:             sslMonitor_minimal,
			PlanOnly:           true,
			ExpectNonEmptyPlan: false,
		},
	})
}

// TestAccSSLMonitorCustomSecurityBaseline applies a config with an explicit
// jsonencode-d security_baseline and re-plans clean — proving the jsonencode key
// order is stable across the create/read round-trip (no spurious diff).
func TestAccSSLMonitorCustomSecurityBaseline(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: sslMonitor_customSecurityBaseline,
			Check: resource.ComposeTestCheckFunc(
				testCheckResourceAttrExpr(
					"checkly_ssl_monitor.test",
					"request.*.security_baseline",
					"minKeySize",
				),
				testCheckResourceAttrExpr(
					"checkly_ssl_monitor.test",
					"request.*.security_baseline",
					"TLS1.2",
				),
			),
		},
		{
			Config:             sslMonitor_customSecurityBaseline,
			PlanOnly:           true,
			ExpectNonEmptyPlan: false,
		},
	})
}

const sslMonitor_basic = `
	resource "checkly_ssl_monitor" "test" {
	  name      = "SSL Monitor 1"
	  activated = true
	  frequency = 60
	  locations = ["us-east-1"]
	  request {
		hostname                 = "api.checklyhq.com"
		port                     = 443
		alert_days_before_expiry = 30
	  }
	}
`

const sslMonitor_full = `
	resource "checkly_ssl_monitor" "test" {
	  name                   = "sslMonitor_full"
	  activated              = true
	  frequency              = 60
	  muted                  = true
	  degraded_response_time = 3000
	  max_response_time      = 10000
	  locations = [
		"us-east-1",
		"eu-central-1",
	  ]
	  request {
		hostname                 = "api.checklyhq.com"
		port                     = 443
		server_name              = "api.checklyhq.com"
		ip_family                = "IPv4"
		skip_chain_validation    = false
		handshake_timeout_ms     = 10000
		alert_days_before_expiry = 20

		security_baseline = jsonencode({
		  enabled                 = true
		  minTLSVersion           = { value = "TLS1.2", severity = "fail" }
		  minKeySizeBits          = { value = 2048, severity = "fail" }
		  weakSignatureAlgorithm  = { severity = "fail" }
		  weakCipherSuite         = { severity = "fail" }
		  knownBadCA              = { severity = "fail" }
		  recommendedTLSVersion   = { value = "TLS1.3", severity = "ignore" }
		  recommendedKeySizeBits  = { value = 3072, severity = "ignore" }
		  ocspMustStapleRespected = { severity = "ignore" }
		  sctPresent              = { severity = "ignore" }
		})

		client_certificate {
		  mode = "account_default"
		}

		assertion {
		  source     = "CERTIFICATE"
		  property   = "daysUntilExpiry"
		  comparison = "GREATER_THAN"
		  target     = "14"
		}

		assertion {
		  source     = "CONNECTION"
		  property   = "hostnameVerified"
		  comparison = "EQUALS"
		  target     = "true"
		}
	  }

	  alert_settings {
		escalation_type = "RUN_BASED"
		reminders {
		  amount   = 1
		  interval = 5
		}
		run_based_escalation {
		  failed_run_threshold = 1
		}
	  }
	}
`

// sslMonitor_minimal omits every optional attribute; security_baseline and
// client_certificate are computed and must not produce a re-plan diff.
const sslMonitor_minimal = `
	resource "checkly_ssl_monitor" "test" {
	  name      = "ssl-minimal"
	  activated = true
	  frequency = 60
	  locations = ["us-east-1"]
	  request {
		hostname = "api.checklyhq.com"
	  }
	}
`

// sslMonitor_customSecurityBaseline supplies the COMPLETE canonical baseline.
// The server normalizes any partial baseline by filling in the remaining rules,
// so a clean re-plan requires the config to enumerate every rule. The keys here
// are intentionally in a different order than the SDK's struct/wire order to
// prove the security_baseline diff suppressor handles jsonencode key ordering.
const sslMonitor_customSecurityBaseline = `
	resource "checkly_ssl_monitor" "test" {
	  name      = "ssl-custom-baseline"
	  activated = true
	  frequency = 60
	  locations = ["us-east-1"]
	  request {
		hostname = "api.checklyhq.com"
		security_baseline = jsonencode({
		  minKeySizeBits          = { value = 2048, severity = "fail" }
		  minTLSVersion           = { value = "TLS1.2", severity = "fail" }
		  recommendedKeySizeBits  = { value = 3072, severity = "ignore" }
		  recommendedTLSVersion   = { value = "TLS1.3", severity = "ignore" }
		  weakSignatureAlgorithm  = { severity = "fail" }
		  weakCipherSuite         = { severity = "fail" }
		  knownBadCA              = { severity = "fail" }
		  ocspMustStapleRespected = { severity = "ignore" }
		  sctPresent              = { severity = "ignore" }
		  enabled                 = true
		})
	  }
	}
`
