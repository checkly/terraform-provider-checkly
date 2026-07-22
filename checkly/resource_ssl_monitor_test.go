package checkly

import (
	"regexp"
	"testing"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestSecurityBaselineFromList is a unit test (no TF_ACC) asserting that the
// security_baseline block expands into a payload carrying exactly the
// configured rules: the API replaces the stored baseline wholesale on every
// write, so a rule absent from the payload is reset to its server default.
func TestSecurityBaselineFromList(t *testing.T) {
	if securityBaselineFromList(nil) != nil {
		t.Fatal("an absent block must expand to nil so the server default baseline applies")
	}

	baseline := securityBaselineFromList([]any{tfMap{
		"enabled":                    false,
		"min_key_size_bits":          []any{tfMap{"value": 4096, "severity": "degrade"}},
		"min_tls_version":            []any{},
		"weak_signature_algorithm":   []any{},
		"weak_cipher_suite":          []any{},
		"known_bad_ca":               []any{},
		"recommended_tls_version":    []any{},
		"recommended_key_size_bits":  []any{},
		"ocsp_must_staple_respected": []any{},
		"sct_present":                []any{},
	}})
	if baseline == nil || baseline.Enabled == nil || *baseline.Enabled {
		t.Fatalf("expected enabled=false to survive expansion, got %+v", baseline)
	}
	if baseline.MinKeySizeBits == nil || baseline.MinKeySizeBits.Value != 4096 || baseline.MinKeySizeBits.Severity != "degrade" {
		t.Fatalf("expected minKeySizeBits {4096 degrade}, got %+v", baseline.MinKeySizeBits)
	}
	if baseline.MinTLSVersion != nil || baseline.KnownBadCA != nil {
		t.Fatalf("rules absent from config must be omitted from the payload, got %+v", baseline)
	}
}

// TestSetFromSecurityBaseline asserts the read-side projection: the server
// normalizes every baseline to the full rule set, and only the rules present
// in prior state (i.e. in the config as of the last apply) are written back —
// otherwise a config with a partial baseline would diff forever.
func TestSetFromSecurityBaseline(t *testing.T) {
	enabled := true
	remote := &checkly.SecurityBaseline{
		Enabled:        &enabled,
		MinTLSVersion:  &checkly.SSLBaselineTLSRule{Value: "TLS1.2", Severity: "fail"},
		MinKeySizeBits: &checkly.SSLBaselineKeySizeRule{Value: 4096, Severity: "fail"},
		KnownBadCA:     &checkly.SSLBaselineSeverityRule{Severity: "fail"},
	}

	if got := setFromSecurityBaseline(remote, nil); got != nil {
		t.Fatalf("with no prior state (import), the baseline must not be projected into state, got %+v", got)
	}
	if got := setFromSecurityBaseline(remote, tfMap{"security_baseline": []any{}}); got != nil {
		t.Fatalf("with no prior baseline block, the server default must not be projected into state, got %+v", got)
	}

	prior := tfMap{"security_baseline": []any{tfMap{
		"enabled":           true,
		"min_key_size_bits": []any{tfMap{"value": 2048, "severity": "fail"}},
	}}}
	got := setFromSecurityBaseline(remote, prior)
	if len(got) != 1 {
		t.Fatalf("expected a projected baseline block, got %+v", got)
	}
	if _, ok := got[0]["min_tls_version"]; ok {
		t.Fatalf("rule absent from prior state must not be projected, got %+v", got[0])
	}
	rule, ok := got[0]["min_key_size_bits"].([]tfMap)
	if !ok || len(rule) != 1 || rule[0]["value"] != 4096 {
		t.Fatalf("configured rule must carry the remote (drifted) value 4096, got %+v", got[0]["min_key_size_bits"])
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
		{
			// The full config enumerates every baseline rule, including the
			// severity-only and advisory ones — a clean re-plan proves each
			// rule's read-side projection round-trips.
			Config:             sslMonitor_full,
			PlanOnly:           true,
			ExpectNonEmptyPlan: false,
		},
	})
}

// TestAccSSLMonitorMinimalCleanReplan asserts anti-pattern B is avoided: a
// config omitting every optional field (including the security_baseline and
// client_certificate blocks) applies, then re-plans with no diff.
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

// TestAccSSLMonitorCustomSecurityBaseline exercises the security_baseline
// nested-block lifecycle: a PARTIAL baseline applies and re-plans clean (the
// server fills in the unlisted rules, which must not be projected into state),
// adding a rule works, removing a rule produces a diff and resets it to the
// server default (the API replaces the baseline wholesale on writes), and
// removing the whole block resets the baseline entirely.
func TestAccSSLMonitorCustomSecurityBaseline(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			// Partial baseline: one rule only.
			Config: sslMonitor_partialSecurityBaseline,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_ssl_monitor.test",
					"request.0.security_baseline.0.min_key_size_bits.0.value",
					"4096",
				),
				resource.TestCheckResourceAttr(
					"checkly_ssl_monitor.test",
					"request.0.security_baseline.0.min_key_size_bits.0.severity",
					"fail",
				),
				resource.TestCheckResourceAttr(
					"checkly_ssl_monitor.test",
					"request.0.security_baseline.0.enabled",
					"true",
				),
				// Rules the config does not mention must not be projected
				// into state, or the partial baseline would diff forever.
				resource.TestCheckNoResourceAttr(
					"checkly_ssl_monitor.test",
					"request.0.security_baseline.0.min_tls_version.0.value",
				),
			),
		},
		{
			// The partial-baseline footgun regression check: no perpetual diff.
			Config:             sslMonitor_partialSecurityBaseline,
			PlanOnly:           true,
			ExpectNonEmptyPlan: false,
		},
		{
			// Add a second rule.
			Config: sslMonitor_twoRuleSecurityBaseline,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_ssl_monitor.test",
					"request.0.security_baseline.0.min_tls_version.0.value",
					"TLS1.3",
				),
				resource.TestCheckResourceAttr(
					"checkly_ssl_monitor.test",
					"request.0.security_baseline.0.min_key_size_bits.0.value",
					"4096",
				),
			),
		},
		{
			// Remove the min_tls_version rule again: the plan must show the
			// removal, and afterwards the rule must be gone from state (the
			// server reset it to its default).
			Config: sslMonitor_partialSecurityBaseline,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr(
					"checkly_ssl_monitor.test",
					"request.0.security_baseline.0.min_tls_version.0.value",
				),
			),
		},
		{
			Config:             sslMonitor_partialSecurityBaseline,
			PlanOnly:           true,
			ExpectNonEmptyPlan: false,
		},
		{
			// Disable baseline enforcement entirely. The server still returns
			// the full rule set with enabled=false; only enabled may be
			// projected into state.
			Config: sslMonitor_disabledSecurityBaseline,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_ssl_monitor.test",
					"request.0.security_baseline.0.enabled",
					"false",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_ssl_monitor.test",
					"request.0.security_baseline.0.min_key_size_bits.0.value",
				),
			),
		},
		{
			Config:             sslMonitor_disabledSecurityBaseline,
			PlanOnly:           true,
			ExpectNonEmptyPlan: false,
		},
		{
			// Remove the whole block: resets to the account default baseline.
			Config: sslMonitor_minimal,
			Check: resource.TestCheckResourceAttr(
				"checkly_ssl_monitor.test",
				"request.0.security_baseline.#",
				"0",
			),
		},
		{
			Config:             sslMonitor_minimal,
			PlanOnly:           true,
			ExpectNonEmptyPlan: false,
		},
	})
}

// TestAccSSLMonitorSecurityBaselineUnknownRule asserts that a misspelled rule
// is rejected at plan time by the schema instead of being silently dropped
// (which would leave the intended security control unenforced).
func TestAccSSLMonitorSecurityBaselineUnknownRule(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config:      sslMonitor_unknownBaselineRule,
			ExpectError: regexp.MustCompile(`Blocks of type "min_key_size_bts" are not expected here`),
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

		security_baseline {
		  enabled = true
		  min_tls_version {
			value    = "TLS1.2"
			severity = "fail"
		  }
		  min_key_size_bits {
			value    = 2048
			severity = "fail"
		  }
		  weak_signature_algorithm {
			severity = "fail"
		  }
		  weak_cipher_suite {
			severity = "fail"
		  }
		  known_bad_ca {
			severity = "fail"
		  }
		  recommended_tls_version {
			value    = "TLS1.3"
			severity = "ignore"
		  }
		  recommended_key_size_bits {
			value    = 3072
			severity = "ignore"
		  }
		  ocsp_must_staple_respected {
			severity = "ignore"
		  }
		  sct_present {
			severity = "ignore"
		  }
		}

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

// sslMonitor_minimal omits every optional attribute; the server-side defaults
// (including the default security baseline) must not produce a re-plan diff.
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

// sslMonitor_partialSecurityBaseline overrides a single rule; the server fills
// in the rest, which must stay out of state (projection) so this re-plans clean.
const sslMonitor_partialSecurityBaseline = `
	resource "checkly_ssl_monitor" "test" {
	  name      = "ssl-custom-baseline"
	  activated = true
	  frequency = 60
	  locations = ["us-east-1"]
	  request {
		hostname = "api.checklyhq.com"
		security_baseline {
		  min_key_size_bits {
			value = 4096
		  }
		}
	  }
	}
`

const sslMonitor_twoRuleSecurityBaseline = `
	resource "checkly_ssl_monitor" "test" {
	  name      = "ssl-custom-baseline"
	  activated = true
	  frequency = 60
	  locations = ["us-east-1"]
	  request {
		hostname = "api.checklyhq.com"
		security_baseline {
		  min_key_size_bits {
			value = 4096
		  }
		  min_tls_version {
			value = "TLS1.3"
		  }
		}
	  }
	}
`

// sslMonitor_unknownBaselineRule misspells min_key_size_bits; the nested-block
// schema must reject it at plan time.
const sslMonitor_disabledSecurityBaseline = `
	resource "checkly_ssl_monitor" "test" {
	  name      = "ssl-custom-baseline"
	  activated = true
	  frequency = 60
	  locations = ["us-east-1"]
	  request {
		hostname = "api.checklyhq.com"
		security_baseline {
		  enabled = false
		}
	  }
	}
`

const sslMonitor_unknownBaselineRule = `
	resource "checkly_ssl_monitor" "test" {
	  name      = "ssl-unknown-baseline-rule"
	  activated = true
	  frequency = 60
	  locations = ["us-east-1"]
	  request {
		hostname = "api.checklyhq.com"
		security_baseline {
		  min_key_size_bts {
			value = 2048
		  }
		}
	  }
	}
`
