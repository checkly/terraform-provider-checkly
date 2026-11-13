package provider

import (
	"context"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/checkly/checkly-go-sdk/v2"
)

// sslMonitorTestRemoteBaseline is a complete security baseline, as the API
// returns it, in which every rule differs from its server default.
func sslMonitorTestRemoteBaseline() *checkly.SecurityBaseline {
	return &checkly.SecurityBaseline{
		Enabled:                 new(false),
		MinTLSVersion:           &checkly.SSLBaselineTLSRule{Value: checkly.TLSVersion13, Severity: checkly.SSLBaselineSeverityDegrade},
		MinKeySizeBits:          &checkly.SSLBaselineKeySizeRule{Value: 4096, Severity: checkly.SSLBaselineSeverityDegrade},
		WeakSignatureAlgorithm:  &checkly.SSLBaselineSeverityRule{Severity: checkly.SSLBaselineSeverityDegrade},
		WeakCipherSuite:         &checkly.SSLBaselineSeverityRule{Severity: checkly.SSLBaselineSeverityIgnore},
		KnownBadCA:              &checkly.SSLBaselineSeverityRule{Severity: checkly.SSLBaselineSeverityDegrade},
		RecommendedTLSVersion:   &checkly.SSLBaselineTLSRule{Value: checkly.TLSVersion12, Severity: checkly.SSLBaselineSeverityDegrade},
		RecommendedKeySizeBits:  &checkly.SSLBaselineKeySizeRule{Value: 8192, Severity: checkly.SSLBaselineSeverityFail},
		OCSPMustStapleRespected: &checkly.SSLBaselineSeverityRule{Severity: checkly.SSLBaselineSeverityFail},
		SCTPresent:              &checkly.SSLBaselineSeverityRule{Severity: checkly.SSLBaselineSeverityDegrade},
	}
}

// TestSSLMonitorSecurityBaselineFromObject asserts that the security_baseline
// attribute expands into a payload carrying every rule, and that a null
// value leaves the baseline out of the payload.
func TestSSLMonitorSecurityBaselineFromObject(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var diags diag.Diagnostics

	if got := sslMonitorSecurityBaselineFromObject(ctx, types.ObjectNull(sslMonitorSecurityBaselineAttrTypes), &diags); got != nil {
		t.Fatalf("a null baseline must expand to nil, got %+v", got)
	}

	want := sslMonitorTestRemoteBaseline()
	got := sslMonitorSecurityBaselineFromObject(ctx, sslMonitorSecurityBaselineToObject(ctx, want, &diags), &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("baseline did not round-trip (-want +got):\n%s", diff)
	}

	// The schema default expands to the server's default baseline.
	got = sslMonitorSecurityBaselineFromObject(ctx, sslMonitorSecurityBaselineToObject(ctx, nil, &diags), &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if diff := cmp.Diff(&sslMonitorDefaultSecurityBaseline, got); diff != "" {
		t.Fatalf("default baseline did not expand to the server default (-want +got):\n%s", diff)
	}
}

// TestSSLMonitorSecurityBaselineToObject asserts that a missing baseline,
// enabled flag or rule (not expected from the API) converts to its default,
// while the rules the server returns keep the server's values.
func TestSSLMonitorSecurityBaselineToObject(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	partial := sslMonitorTestRemoteBaseline()
	partial.Enabled = nil
	partial.MinKeySizeBits = nil

	partialWant := sslMonitorTestRemoteBaseline()
	partialWant.Enabled = new(true)
	partialWant.MinKeySizeBits = sslMonitorDefaultSecurityBaseline.MinKeySizeBits

	for name, tc := range map[string]struct {
		remote *checkly.SecurityBaseline
		want   *checkly.SecurityBaseline
	}{
		"nil baseline":                 {remote: nil, want: &sslMonitorDefaultSecurityBaseline},
		"nil enabled and nil key size": {remote: partial, want: partialWant},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var diags diag.Diagnostics
			got := sslMonitorSecurityBaselineFromObject(ctx, sslMonitorSecurityBaselineToObject(ctx, tc.remote, &diags), &diags)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected baseline (-want +got):\n%s", diff)
			}
		})
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
					assertions = [{
					  source     = "CERT_EXPIRES_IN_DAYS"
					  comparison = "GREATER_THAN"
					  target     = "14"
					}]
				  }
				}
			`,
			ExpectError: regexp.MustCompile(`(?s)request\.assertions\[.*\]\.source\s+value\s+must\s+be\s+one\s+of`),
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
					"request.assertions.#",
					"2",
				),
				resource.TestCheckTypeSetElemNestedAttrs(
					"checkly_ssl_monitor.test",
					"request.assertions.*",
					map[string]string{
						"source":     "CERTIFICATE",
						"property":   "daysUntilExpiry",
						"comparison": "GREATER_THAN",
						"target":     "14",
					},
				),
				resource.TestCheckTypeSetElemNestedAttrs(
					"checkly_ssl_monitor.test",
					"request.assertions.*",
					map[string]string{
						"source":     "CONNECTION",
						"property":   "hostnameVerified",
						"comparison": "EQUALS",
						"target":     "true",
					},
				),
				resource.TestCheckResourceAttr("checkly_ssl_monitor.test", "request.security_baseline.enabled", "true"),
				sslMonitorCheckBaselineRule("min_tls_version", "TLS1.3", "degrade"),
				sslMonitorCheckBaselineRule("min_key_size_bits", "3072", "degrade"),
				sslMonitorCheckBaselineRule("weak_signature_algorithm", "", "degrade"),
				sslMonitorCheckBaselineRule("weak_cipher_suite", "", "degrade"),
				sslMonitorCheckBaselineRule("known_bad_ca", "", "degrade"),
				sslMonitorCheckBaselineRule("recommended_tls_version", "TLS1.2", "degrade"),
				sslMonitorCheckBaselineRule("recommended_key_size_bits", "4096", "degrade"),
				sslMonitorCheckBaselineRule("ocsp_must_staple_respected", "", "degrade"),
				sslMonitorCheckBaselineRule("sct_present", "", "fail"),
			),
		},
		{
			// The full config enumerates every baseline rule, including the
			// severity-only and advisory ones — a clean re-plan proves each
			// rule round-trips.
			Config:             sslMonitor_full,
			PlanOnly:           true,
			ExpectNonEmptyPlan: false,
		},
		{
			// State holds the server's full baseline, so an imported monitor
			// matches the state written by create.
			ResourceName:      "checkly_ssl_monitor.test",
			ImportState:       true,
			ImportStateVerify: true,
		},
		{
			// An import block with a config listing every rule plans no
			// changes.
			ResourceName:    "checkly_ssl_monitor.test",
			ImportState:     true,
			ImportStateKind: resource.ImportBlockWithID,
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
			// state written by create: both hold the server's default
			// baseline.
			ResourceName:      "checkly_ssl_monitor.test",
			ImportState:       true,
			ImportStateVerify: true,
		},
		{
			// An import block with a config that omits security_baseline
			// plans no changes: the schema default is the server default.
			ResourceName:    "checkly_ssl_monitor.test",
			ImportState:     true,
			ImportStateKind: resource.ImportBlockWithID,
		},
	})
}

// sslMonitorCheckBaselineRule checks the value and severity of a security
// baseline rule in state. An empty value skips the value check, for rules
// that have none.
func sslMonitorCheckBaselineRule(rule, value, severity string) resource.TestCheckFunc {
	const name = "checkly_ssl_monitor.test"
	prefix := "request.security_baseline." + rule + "."

	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr(name, prefix+"severity", severity),
	}
	if value != "" {
		checks = append(checks, resource.TestCheckResourceAttr(name, prefix+"value", value))
	}

	return resource.ComposeAggregateTestCheckFunc(checks...)
}

// sslMonitorCheckDefaultBaselineExcept checks that every security baseline
// rule except the named ones holds its server default in state.
func sslMonitorCheckDefaultBaselineExcept(except ...string) resource.TestCheckFunc {
	defaults := map[string][2]string{
		"min_tls_version":            {"TLS1.2", "fail"},
		"min_key_size_bits":          {"2048", "fail"},
		"weak_signature_algorithm":   {"", "fail"},
		"weak_cipher_suite":          {"", "fail"},
		"known_bad_ca":               {"", "fail"},
		"recommended_tls_version":    {"TLS1.3", "ignore"},
		"recommended_key_size_bits":  {"3072", "ignore"},
		"ocsp_must_staple_respected": {"", "ignore"},
		"sct_present":                {"", "ignore"},
	}
	for _, rule := range except {
		delete(defaults, rule)
	}

	var checks []resource.TestCheckFunc
	for rule, def := range defaults {
		checks = append(checks, sslMonitorCheckBaselineRule(rule, def[0], def[1]))
	}

	return resource.ComposeAggregateTestCheckFunc(checks...)
}

// TestAccSSLMonitorCustomSecurityBaseline exercises the security_baseline
// lifecycle: a partial baseline applies, holds the server defaults for the
// unlisted rules and re-plans clean; an import with the same config plans
// no changes; an external change to an unlisted rule shows up as drift;
// adding a rule works; removing a rule plans a reset to its default; and
// removing the whole attribute resets the baseline entirely.
func TestAccSSLMonitorCustomSecurityBaseline(t *testing.T) {
	const name = "checkly_ssl_monitor.test"

	var id string
	captureID := func(s *terraform.State) error {
		id = s.RootModule().Resources[name].Primary.ID
		return nil
	}

	accTestCase(t, []resource.TestStep{
		{
			// Partial baseline: one rule only.
			Config: sslMonitor_partialSecurityBaseline,
			Check: resource.ComposeTestCheckFunc(
				captureID,
				sslMonitorCheckBaselineRule("min_key_size_bits", "4096", "fail"),
				resource.TestCheckResourceAttr(name, "request.security_baseline.enabled", "true"),
				sslMonitorCheckDefaultBaselineExcept("min_key_size_bits"),
			),
		},
		{
			// The partial-baseline regression check: no perpetual diff.
			Config:             sslMonitor_partialSecurityBaseline,
			PlanOnly:           true,
			ExpectNonEmptyPlan: false,
		},
		{
			// An import block with a config listing one rule plans no
			// changes: the unlisted rules plan their defaults, which is
			// what the server holds.
			ResourceName:    name,
			ImportState:     true,
			ImportStateKind: resource.ImportBlockWithID,
		},
		{
			// A rule the config does not list is still drift-checked: a
			// change made outside Terraform plans an update back to the
			// default.
			PreConfig: func() {
				ctx := context.Background()
				client := testAccClient()
				monitor, err := client.GetSSLMonitor(ctx, id)
				if err != nil {
					t.Fatal(err)
				}
				if monitor.Request.SSLConfig.SecurityBaseline == nil {
					t.Fatal("the API returned no security baseline")
				}
				monitor.Request.SSLConfig.SecurityBaseline.SCTPresent = &checkly.SSLBaselineSeverityRule{Severity: checkly.SSLBaselineSeverityDegrade}
				if _, err := client.UpdateSSLMonitor(ctx, id, *monitor); err != nil {
					t.Fatal(err)
				}
			},
			Config: sslMonitor_partialSecurityBaseline,
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(name, plancheck.ResourceActionUpdate),
					plancheck.ExpectKnownValue(name,
						tfjsonpath.New("request").AtMapKey("security_baseline").AtMapKey("sct_present").AtMapKey("severity"),
						knownvalue.StringExact("ignore")),
				},
			},
			Check: sslMonitorCheckBaselineRule("sct_present", "", "ignore"),
		},
		{
			// Add a second rule.
			Config: sslMonitor_twoRuleSecurityBaseline,
			Check: resource.ComposeTestCheckFunc(
				sslMonitorCheckBaselineRule("min_tls_version", "TLS1.3", "fail"),
				sslMonitorCheckBaselineRule("min_key_size_bits", "4096", "fail"),
			),
		},
		{
			// Remove the min_tls_version rule again: the plan resets it to
			// its default.
			Config: sslMonitor_partialSecurityBaseline,
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(name, plancheck.ResourceActionUpdate),
					plancheck.ExpectKnownValue(name,
						tfjsonpath.New("request").AtMapKey("security_baseline").AtMapKey("min_tls_version"),
						knownvalue.ObjectExact(map[string]knownvalue.Check{
							"value":    knownvalue.StringExact("TLS1.2"),
							"severity": knownvalue.StringExact("fail"),
						})),
				},
			},
			Check: resource.ComposeTestCheckFunc(
				sslMonitorCheckBaselineRule("min_key_size_bits", "4096", "fail"),
				sslMonitorCheckDefaultBaselineExcept("min_key_size_bits"),
			),
		},
		{
			Config:             sslMonitor_partialSecurityBaseline,
			PlanOnly:           true,
			ExpectNonEmptyPlan: false,
		},
		{
			// Disable baseline enforcement entirely. The server still returns
			// the full rule set with enabled=false, and the rule removed from
			// the config is reset to its default.
			Config: sslMonitor_disabledSecurityBaseline,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(name, "request.security_baseline.enabled", "false"),
				sslMonitorCheckDefaultBaselineExcept(),
			),
		},
		{
			Config:             sslMonitor_disabledSecurityBaseline,
			PlanOnly:           true,
			ExpectNonEmptyPlan: false,
		},
		{
			// Remove the whole attribute: resets to the default baseline.
			Config: sslMonitor_minimal,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(name, "request.security_baseline.enabled", "true"),
				sslMonitorCheckDefaultBaselineExcept(),
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

// sslMonitor_full sets every optional attribute. Each security baseline rule
// has a value that differs from its default.
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
			value    = "TLS1.3"
			severity = "degrade"
		  }
		  min_key_size_bits = {
			value    = 3072
			severity = "degrade"
		  }
		  weak_signature_algorithm = {
			severity = "degrade"
		  }
		  weak_cipher_suite = {
			severity = "degrade"
		  }
		  known_bad_ca = {
			severity = "degrade"
		  }
		  recommended_tls_version = {
			value    = "TLS1.2"
			severity = "degrade"
		  }
		  recommended_key_size_bits = {
			value    = 4096
			severity = "degrade"
		  }
		  ocsp_must_staple_respected = {
			severity = "degrade"
		  }
		  sct_present = {
			severity = "fail"
		  }
		}

		client_certificate = {
		  mode = "account_default"
		}

		assertions = [
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

// sslMonitor_partialSecurityBaseline overrides a single rule; every other
// rule takes its server default.
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
