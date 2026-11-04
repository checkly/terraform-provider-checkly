package provider

import (
	"context"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	checkly "github.com/checkly/checkly-go-sdk"
)

// sslMonitorTestBaseline returns a request.security_baseline attribute value
// with the given enabled flag and minimum key size rule (when keySize is
// non-nil). Every other rule is null.
func sslMonitorTestBaseline(t *testing.T, enabled bool, keySize *SSLMonitorBaselineKeySizeRuleModel) types.Object {
	t.Helper()

	ctx := context.Background()

	keySizeRule := types.ObjectNull(sslMonitorBaselineKeySizeRuleAttrTypes)
	if keySize != nil {
		var d diag.Diagnostics
		keySizeRule, d = types.ObjectValueFrom(ctx, sslMonitorBaselineKeySizeRuleAttrTypes, *keySize)
		if d.HasError() {
			t.Fatalf("unexpected diagnostics: %v", d)
		}
	}

	obj, d := types.ObjectValueFrom(ctx, sslMonitorSecurityBaselineAttrTypes, SSLMonitorSecurityBaselineModel{
		Enabled:                 types.BoolValue(enabled),
		MinTLSVersion:           types.ObjectNull(sslMonitorBaselineTLSRuleAttrTypes),
		MinKeySizeBits:          keySizeRule,
		WeakSignatureAlgorithm:  types.ObjectNull(sslMonitorBaselineSeverityRuleAttrTypes),
		WeakCipherSuite:         types.ObjectNull(sslMonitorBaselineSeverityRuleAttrTypes),
		KnownBadCA:              types.ObjectNull(sslMonitorBaselineSeverityRuleAttrTypes),
		RecommendedTLSVersion:   types.ObjectNull(sslMonitorBaselineTLSRuleAttrTypes),
		RecommendedKeySizeBits:  types.ObjectNull(sslMonitorBaselineKeySizeRuleAttrTypes),
		OCSPMustStapleRespected: types.ObjectNull(sslMonitorBaselineSeverityRuleAttrTypes),
		SCTPresent:              types.ObjectNull(sslMonitorBaselineSeverityRuleAttrTypes),
	})
	if d.HasError() {
		t.Fatalf("unexpected diagnostics: %v", d)
	}

	return obj
}

// TestSSLMonitorSecurityBaselineFromObject asserts that the security_baseline
// attribute expands into a payload carrying exactly the configured rules: the
// API replaces the stored baseline wholesale on every write, so a rule absent
// from the payload is reset to its server default.
func TestSSLMonitorSecurityBaselineFromObject(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var diags diag.Diagnostics

	if got := sslMonitorSecurityBaselineFromObject(ctx, types.ObjectNull(sslMonitorSecurityBaselineAttrTypes), &diags); got != nil {
		t.Fatalf("an absent baseline must expand to nil so the server default baseline applies, got %+v", got)
	}

	baseline := sslMonitorSecurityBaselineFromObject(ctx, sslMonitorTestBaseline(t, false, &SSLMonitorBaselineKeySizeRuleModel{
		Value:    types.Int64Value(4096),
		Severity: types.StringValue("degrade"),
	}), &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
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

// TestSSLMonitorSecurityBaselineToObject asserts the read-side projection:
// the server normalizes every baseline to the full rule set, and only the
// rules present in prior state (i.e. in the config as of the last apply) are
// written back. Otherwise a config with a partial baseline would diff forever.
func TestSSLMonitorSecurityBaselineToObject(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var diags diag.Diagnostics

	enabled := true
	remote := &checkly.SecurityBaseline{
		Enabled:        &enabled,
		MinTLSVersion:  &checkly.SSLBaselineTLSRule{Value: "TLS1.2", Severity: "fail"},
		MinKeySizeBits: &checkly.SSLBaselineKeySizeRule{Value: 4096, Severity: "fail"},
		KnownBadCA:     &checkly.SSLBaselineSeverityRule{Severity: "fail"},
	}

	if got := sslMonitorSecurityBaselineToObject(ctx, remote, types.ObjectNull(sslMonitorSecurityBaselineAttrTypes), &diags); !got.IsNull() {
		t.Fatalf("without a prior baseline (unconfigured, or a fresh import), the server default must not be projected into state, got %v", got)
	}

	prior := sslMonitorTestBaseline(t, true, &SSLMonitorBaselineKeySizeRuleModel{
		Value:    types.Int64Value(2048),
		Severity: types.StringValue("fail"),
	})
	got := sslMonitorSecurityBaselineToObject(ctx, remote, prior, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	var model SSLMonitorSecurityBaselineModel
	diags.Append(got.As(ctx, &model, basetypes.ObjectAsOptions{})...)
	if diags.HasError() || got.IsNull() {
		t.Fatalf("expected a projected baseline, got %v (%v)", got, diags)
	}
	if !model.MinTLSVersion.IsNull() {
		t.Fatalf("rule absent from prior state must not be projected, got %v", model.MinTLSVersion)
	}

	var rule SSLMonitorBaselineKeySizeRuleModel
	diags.Append(model.MinKeySizeBits.As(ctx, &rule, basetypes.ObjectAsOptions{})...)
	if diags.HasError() || rule.Value.ValueInt64() != 4096 {
		t.Fatalf("configured rule must carry the remote (drifted) value 4096, got %v", model.MinKeySizeBits)
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
			// Terraform only reports the missing request attribute once
			// the other required attributes are present.
			Config: `
				resource "checkly_ssl_monitor" "test" {
					name      = "test-required-fields"
					activated = true
					frequency = 60
				}
			`,
			ExpectError: regexp.MustCompile(`The argument "request" is required, but no definition was found.`),
		},
	})
}

// TestAccSSLMonitorInvalidAssertionSource asserts a removed-grammar (or
// mistyped) assertion source fails at plan time instead of surfacing as an
// API error at apply time.
func TestAccSSLMonitorInvalidAssertionSource(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `
				resource "checkly_ssl_monitor" "test" {
				  name      = "ssl-invalid-assertion-source"
				  activated = true
				  frequency = 60
				  locations = ["us-east-1"]
				  request = {
					hostname = "api.checklyhq.com"
					assertion = [{
					  source     = "CERT_EXPIRES_IN_DAYS"
					  comparison = "GREATER_THAN"
					  target     = "14"
					}]
				  }
				}
			`,
			ExpectError: regexp.MustCompile(`(?s)request\.assertion\[.*\]\.source\s+value\s+must\s+be\s+one\s+of`),
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
				resource.TestCheckTypeSetElemAttr(
					"checkly_ssl_monitor.test",
					"locations.*",
					"us-east-1",
				),
				resource.TestCheckResourceAttr(
					"checkly_ssl_monitor.test",
					"request.hostname",
					"api.checklyhq.com",
				),
				resource.TestCheckResourceAttr(
					"checkly_ssl_monitor.test",
					"request.alert_days_before_expiry",
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
				resource.TestCheckResourceAttr(
					"checkly_ssl_monitor.test",
					"locations.#",
					"2",
				),
				resource.TestCheckResourceAttr(
					"checkly_ssl_monitor.test",
					"request.server_name",
					"api.checklyhq.com",
				),
				resource.TestCheckResourceAttr(
					"checkly_ssl_monitor.test",
					"request.handshake_timeout_ms",
					"10000",
				),
				resource.TestCheckResourceAttr(
					"checkly_ssl_monitor.test",
					"degraded_response_time",
					"3000",
				),
				resource.TestCheckResourceAttr(
					"checkly_ssl_monitor.test",
					"request.client_certificate.mode",
					"account_default",
				),
				resource.TestCheckResourceAttr(
					"checkly_ssl_monitor.test",
					"request.assertion.#",
					"2",
				),
				resource.TestCheckTypeSetElemNestedAttrs(
					"checkly_ssl_monitor.test",
					"request.assertion.*",
					map[string]string{
						"source":     "CERTIFICATE",
						"property":   "daysUntilExpiry",
						"comparison": "GREATER_THAN",
						"target":     "14",
					},
				),
				resource.TestCheckTypeSetElemNestedAttrs(
					"checkly_ssl_monitor.test",
					"request.assertion.*",
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
		{
			// Import has no prior baseline to project the server's onto, so
			// the configured baseline is the only difference.
			ResourceName:            "checkly_ssl_monitor.test",
			ImportState:             true,
			ImportStateVerify:       true,
			ImportStateVerifyIgnore: []string{"request.security_baseline"},
		},
	})
}

// TestAccSSLMonitorMinimalCleanReplan asserts that a config omitting every
// optional field (including the security_baseline and client_certificate
// attributes) applies, then re-plans with no diff.
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
		{
			// An imported monitor without a configured baseline matches the
			// state written by create: the server's default baseline is not
			// projected into state.
			ResourceName:      "checkly_ssl_monitor.test",
			ImportState:       true,
			ImportStateVerify: true,
		},
	})
}

// TestAccSSLMonitorCustomSecurityBaseline exercises the security_baseline
// lifecycle: a PARTIAL baseline applies and re-plans clean (the server fills
// in the unlisted rules, which must not be projected into state),
// adding a rule works, removing a rule produces a diff and resets it to the
// server default (the API replaces the baseline wholesale on writes), and
// removing the whole attribute resets the baseline entirely.
func TestAccSSLMonitorCustomSecurityBaseline(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			// Partial baseline: one rule only.
			Config: sslMonitor_partialSecurityBaseline,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"checkly_ssl_monitor.test",
					"request.security_baseline.min_key_size_bits.value",
					"4096",
				),
				resource.TestCheckResourceAttr(
					"checkly_ssl_monitor.test",
					"request.security_baseline.min_key_size_bits.severity",
					"fail",
				),
				resource.TestCheckResourceAttr(
					"checkly_ssl_monitor.test",
					"request.security_baseline.enabled",
					"true",
				),
				// Rules the config does not mention must not be projected
				// into state, or the partial baseline would diff forever.
				resource.TestCheckNoResourceAttr(
					"checkly_ssl_monitor.test",
					"request.security_baseline.min_tls_version.value",
				),
			),
		},
		{
			// The partial-baseline regression check: no perpetual diff.
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
					"request.security_baseline.min_tls_version.value",
					"TLS1.3",
				),
				resource.TestCheckResourceAttr(
					"checkly_ssl_monitor.test",
					"request.security_baseline.min_key_size_bits.value",
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
					"request.security_baseline.min_tls_version.value",
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
					"request.security_baseline.enabled",
					"false",
				),
				resource.TestCheckNoResourceAttr(
					"checkly_ssl_monitor.test",
					"request.security_baseline.min_key_size_bits.value",
				),
			),
		},
		{
			Config:             sslMonitor_disabledSecurityBaseline,
			PlanOnly:           true,
			ExpectNonEmptyPlan: false,
		},
		{
			// Remove the whole attribute: resets to the account default
			// baseline.
			Config: sslMonitor_minimal,
			Check: resource.TestCheckNoResourceAttr(
				"checkly_ssl_monitor.test",
				"request.security_baseline.enabled",
			),
		},
		{
			Config:             sslMonitor_minimal,
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
	  request = {
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
	  request = {
		hostname                 = "api.checklyhq.com"
		port                     = 443
		server_name              = "api.checklyhq.com"
		ip_family                = "IPv4"
		skip_chain_validation    = false
		handshake_timeout_ms     = 10000
		alert_days_before_expiry = 20

		security_baseline = {
		  enabled = true
		  min_tls_version = {
			value    = "TLS1.2"
			severity = "fail"
		  }
		  min_key_size_bits = {
			value    = 2048
			severity = "fail"
		  }
		  weak_signature_algorithm = {
			severity = "fail"
		  }
		  weak_cipher_suite = {
			severity = "fail"
		  }
		  known_bad_ca = {
			severity = "fail"
		  }
		  recommended_tls_version = {
			value    = "TLS1.3"
			severity = "ignore"
		  }
		  recommended_key_size_bits = {
			value    = 3072
			severity = "ignore"
		  }
		  ocsp_must_staple_respected = {
			severity = "ignore"
		  }
		  sct_present = {
			severity = "ignore"
		  }
		}

		client_certificate = {
		  mode = "account_default"
		}

		assertion = [
		  {
			source     = "CERTIFICATE"
			property   = "daysUntilExpiry"
			comparison = "GREATER_THAN"
			target     = "14"
		  },
		  {
			source     = "CONNECTION"
			property   = "hostnameVerified"
			comparison = "EQUALS"
			target     = "true"
		  },
		]
	  }

	  alert_settings = {
		escalation_type = "RUN_BASED"
		reminders = {
		  amount   = 1
		  interval = 5
		}
		run_based_escalation = {
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
	  request = {
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
	  request = {
		hostname = "api.checklyhq.com"
		security_baseline = {
		  min_key_size_bits = {
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
	  request = {
		hostname = "api.checklyhq.com"
		security_baseline = {
		  min_key_size_bits = {
			value = 4096
		  }
		  min_tls_version = {
			value = "TLS1.3"
		  }
		}
	  }
	}
`

// sslMonitor_disabledSecurityBaseline disables the baseline without listing
// any rule.
const sslMonitor_disabledSecurityBaseline = `
	resource "checkly_ssl_monitor" "test" {
	  name      = "ssl-custom-baseline"
	  activated = true
	  frequency = 60
	  locations = ["us-east-1"]
	  request = {
		hostname = "api.checklyhq.com"
		security_baseline = {
		  enabled = false
		}
	  }
	}
`
