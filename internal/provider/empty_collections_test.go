package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// The tests in this file cover optional collections: configured as empty and
// left out, they are both stored as empty, the value the API reports for an
// unset collection. The plan after each apply must be empty, so neither
// configuration drifts against the API, and an import block for either
// configuration plans no changes.

// emptyCollection names a collection attribute and the configuration that
// sets it to an empty value.
type emptyCollection struct {
	path   tfjsonpath.Path
	config string
	// empty is the expected state value, configured as empty or left out.
	empty knownvalue.Check
}

var (
	emptySet  = knownvalue.SetExact([]knownvalue.Check{})
	emptyList = knownvalue.ListExact([]knownvalue.Check{})
	emptyMap  = knownvalue.MapExact(map[string]knownvalue.Check{})
)

// testAccEmptyThenUnsetCollections applies config with every collection set
// to its empty value, then with all of them left out, and imports the
// resource at address with an import block after each. config receives the
// attribute assignments to insert where the collections belong. checks are
// further state checks for both configurations.
func testAccEmptyThenUnsetCollections(t *testing.T, address string, config func(attributes string) string, collections []emptyCollection, checks ...statecheck.StateCheck) {
	t.Helper()

	var attributes string
	stateChecks := append([]statecheck.StateCheck{}, checks...)
	for _, c := range collections {
		attributes += c.config + "\n"
		stateChecks = append(stateChecks, statecheck.ExpectKnownValue(address, c.path, c.empty))
	}

	importStep := resource.TestStep{
		ResourceName:    address,
		ImportState:     true,
		ImportStateKind: resource.ImportBlockWithID,
	}

	accTestCase(t, []resource.TestStep{
		{
			Config:            config(attributes),
			ConfigStateChecks: stateChecks,
		},
		importStep,
		{
			Config:            config(""),
			ConfigStateChecks: stateChecks,
		},
		importStep,
	})
}

// requestAssertions is the request.assertions collection of a check or
// monitor.
var requestAssertions = emptyCollection{
	path:   tfjsonpath.New("request").AtMapKey("assertions"),
	config: "assertions = []",
	empty:  emptySet,
}

func TestAccApiCheckEmptyRequestCollections(t *testing.T) {
	testAccEmptyThenUnsetCollections(t, "checkly_api_check.test", func(attributes string) string {
		return fmt.Sprintf(`
			resource "checkly_api_check" "test" {
				name      = "api-check-empty-request-collections"
				activated = false
				frequency = 60
				locations = ["eu-central-1"]
				request = {
					url = "https://api.checklyhq.com/public-stats"
					%s
				}
			}
		`, attributes)
	}, []emptyCollection{requestAssertions})
}

func TestAccCheckEmptyRequestCollections(t *testing.T) {
	testAccEmptyThenUnsetCollections(t, "checkly_check.test", func(attributes string) string {
		return fmt.Sprintf(`
			resource "checkly_check" "test" {
				name      = "check-empty-request-collections"
				type      = "API"
				activated = false
				frequency = 60
				locations = ["eu-central-1"]
				request = {
					url = "https://api.checklyhq.com/public-stats"
					%s
				}
			}
		`, attributes)
	}, []emptyCollection{requestAssertions})
}

func TestAccURLMonitorEmptyRequestCollections(t *testing.T) {
	testAccEmptyThenUnsetCollections(t, "checkly_url_monitor.test", func(attributes string) string {
		return fmt.Sprintf(`
			resource "checkly_url_monitor" "test" {
				name      = "url-monitor-empty-request-collections"
				activated = false
				frequency = 60
				locations = ["eu-central-1"]
				request = {
					url = "https://api.checklyhq.com"
					%s
				}
			}
		`, attributes)
	}, []emptyCollection{requestAssertions})
}

func TestAccDNSMonitorEmptyRequestCollections(t *testing.T) {
	testAccEmptyThenUnsetCollections(t, "checkly_dns_monitor.test", func(attributes string) string {
		return fmt.Sprintf(`
			resource "checkly_dns_monitor" "test" {
				name      = "dns-monitor-empty-request-collections"
				activated = false
				frequency = 60
				locations = ["eu-central-1"]
				request = {
					record_type = "A"
					query       = "api.checklyhq.com"
					%s
				}
			}
		`, attributes)
	}, []emptyCollection{requestAssertions})
}

func TestAccICMPMonitorEmptyRequestCollections(t *testing.T) {
	testAccEmptyThenUnsetCollections(t, "checkly_icmp_monitor.test", func(attributes string) string {
		return fmt.Sprintf(`
			resource "checkly_icmp_monitor" "test" {
				name      = "icmp-monitor-empty-request-collections"
				activated = false
				frequency = 60
				locations = ["eu-central-1"]
				request = {
					hostname = "api.checklyhq.com"
					%s
				}
			}
		`, attributes)
	}, []emptyCollection{requestAssertions})
}

func TestAccTCPMonitorEmptyRequestCollections(t *testing.T) {
	testAccEmptyThenUnsetCollections(t, "checkly_tcp_monitor.test", func(attributes string) string {
		return fmt.Sprintf(`
			resource "checkly_tcp_monitor" "test" {
				name      = "tcp-monitor-empty-request-collections"
				activated = false
				frequency = 60
				locations = ["eu-central-1"]
				request = {
					hostname = "api.checklyhq.com"
					port     = 80
					%s
				}
			}
		`, attributes)
	}, []emptyCollection{requestAssertions})
}

func TestAccGRPCMonitorEmptyRequestCollections(t *testing.T) {
	testAccEmptyThenUnsetCollections(t, "checkly_grpc_monitor.test", func(attributes string) string {
		return fmt.Sprintf(`
			resource "checkly_grpc_monitor" "test" {
				name      = "grpc-monitor-empty-request-collections"
				activated = false
				frequency = 60
				locations = ["eu-central-1"]
				request = {
					host               = "grpcb.in"
					port               = 9000
					grpc_mode          = "BEHAVIOR"
					tls                = true
					service_definition = "REFLECTION"
					method             = "grpcbin.GRPCBin/DummyUnary"
					message            = jsonencode({ f_string = "hello" })
					%s
				}
			}
		`, attributes)
	}, []emptyCollection{
		requestAssertions,
		{
			path:   tfjsonpath.New("request").AtMapKey("metadata"),
			config: "metadata = []",
			empty:  emptySet,
		},
	})
}

func TestAccSSLMonitorEmptyRequestCollections(t *testing.T) {
	testAccEmptyThenUnsetCollections(t, "checkly_ssl_monitor.test", func(attributes string) string {
		return fmt.Sprintf(`
			resource "checkly_ssl_monitor" "test" {
				name      = "ssl-monitor-empty-request-collections"
				activated = false
				frequency = 60
				locations = ["eu-central-1"]
				request = {
					hostname = "api.checklyhq.com"
					%s
				}
			}
		`, attributes)
	}, []emptyCollection{requestAssertions})
}

func TestAccTracerouteMonitorEmptyRequestCollections(t *testing.T) {
	testAccEmptyThenUnsetCollections(t, "checkly_traceroute_monitor.test", func(attributes string) string {
		return fmt.Sprintf(`
			resource "checkly_traceroute_monitor" "test" {
				name      = "traceroute-monitor-empty-request-collections"
				activated = false
				frequency = 60
				locations = ["eu-central-1"]
				request = {
					url = "api.checklyhq.com"
					%s
				}
			}
		`, attributes)
	}, []emptyCollection{requestAssertions})
}

// apiCheckDefaultsCollections are the collections of api_check_defaults on
// a check group.
var apiCheckDefaultsCollections = []emptyCollection{
	{
		path:   tfjsonpath.New(apiCheckDefaultsAttributeName).AtMapKey("headers"),
		config: "headers = {}",
		empty:  emptyMap,
	},
	{
		path:   tfjsonpath.New(apiCheckDefaultsAttributeName).AtMapKey("query_parameters"),
		config: "query_parameters = {}",
		empty:  emptyMap,
	},
	{
		path:   tfjsonpath.New(apiCheckDefaultsAttributeName).AtMapKey("assertions"),
		config: "assertions = []",
		empty:  emptySet,
	},
}

func TestAccCheckGroupEmptyAPICheckDefaultsCollections(t *testing.T) {
	testAccEmptyThenUnsetCollections(t, "checkly_check_group.test", func(attributes string) string {
		return fmt.Sprintf(`
			resource "checkly_check_group" "test" {
				name        = "group-empty-api-check-defaults-collections"
				activated   = true
				concurrency = 1
				locations   = ["eu-central-1"]
				api_check_defaults = {
					url = "https://api.example.com/"
					%s
				}
			}
		`, attributes)
	}, apiCheckDefaultsCollections)
}

func TestAccCheckGroupV2EmptyAPICheckDefaultsCollections(t *testing.T) {
	testAccEmptyThenUnsetCollections(t, "checkly_check_group_v2.test", func(attributes string) string {
		return fmt.Sprintf(`
			resource "checkly_check_group_v2" "test" {
				name = "group-v2-empty-api-check-defaults-collections"
				api_check_defaults = {
					url = "https://api.example.com/"
					%s
				}
			}
		`, attributes)
	}, apiCheckDefaultsCollections)
}

func TestAccAlertChannelWebhookEmptyCollections(t *testing.T) {
	name := fmt.Sprintf("webhook-empty-collections-%d", acctest.RandInt())
	testAccEmptyThenUnsetCollections(t, "checkly_alert_channel.test", func(attributes string) string {
		return fmt.Sprintf(`
			resource "checkly_alert_channel" "test" {
				webhook = {
					name = %q
					url  = "https://example.com/webhook"
					%s
				}
			}
		`, name, attributes)
	}, []emptyCollection{
		{
			path:   tfjsonpath.New("webhook").AtMapKey("headers"),
			config: "headers = {}",
			empty:  emptyMap,
		},
		{
			path:   tfjsonpath.New("webhook").AtMapKey("query_parameters"),
			config: "query_parameters = {}",
			empty:  emptyMap,
		},
	})
}

// tagsLengthOutput is an output that applies length() to the tags of
// address, which fails for a null value.
func tagsLengthOutput(address string) string {
	return fmt.Sprintf(`
		output "tags_length" {
			value = length(%s.tags)
		}
	`, address)
}

func TestAccApiCheckEmptyCollections(t *testing.T) {
	channel := fmt.Sprintf("api-check-empty-collections-%d@example.com", acctest.RandInt())
	testAccEmptyThenUnsetCollections(t, "checkly_api_check.test", func(attributes string) string {
		return fmt.Sprintf(`
			resource "checkly_alert_channel" "email" {
				email = {
					address = %q
				}
			}

			resource "checkly_api_check" "test" {
				name      = "api-check-empty-collections"
				activated = false
				frequency = 60
				locations = ["eu-central-1"]
				request = {
					url = "https://api.checklyhq.com/public-stats"
				}
				intent = {
					goal = "The public stats endpoint responds."
				}
				%s
			}
		`, channel, attributes) + tagsLengthOutput("checkly_api_check.test")
	}, []emptyCollection{
		{path: tfjsonpath.New("tags"), config: "tags = []", empty: emptySet},
		{path: tfjsonpath.New("private_locations"), config: "private_locations = []", empty: emptySet},
		{path: tfjsonpath.New("alert_channel_subscriptions"), config: "alert_channel_subscriptions = []", empty: emptySet},
	},
		statecheck.ExpectKnownOutputValue("tags_length", knownvalue.Int64Exact(0)),
		statecheck.ExpectKnownValue("checkly_api_check.test", tfjsonpath.New("intent").AtMapKey("required_outcomes"), emptyList),
		statecheck.ExpectKnownValue("checkly_api_check.test", tfjsonpath.New("intent").AtMapKey("must_preserve"), emptyList),
		statecheck.ExpectKnownValue("checkly_api_check.test", tfjsonpath.New("request").AtMapKey("headers"), emptyMap),
		statecheck.ExpectKnownValue("checkly_api_check.test", tfjsonpath.New("request").AtMapKey("query_parameters"), emptyMap),
	)
}

func TestAccBrowserCheckEmptyCollections(t *testing.T) {
	testAccEmptyThenUnsetCollections(t, "checkly_browser_check.test", func(attributes string) string {
		return fmt.Sprintf(`
			resource "checkly_browser_check" "test" {
				name      = "browser-check-empty-collections"
				activated = false
				frequency = 720
				locations = ["us-east-1"]
				script    = "console.log('test')"
				%s
			}
		`, attributes)
	}, []emptyCollection{
		{path: tfjsonpath.New("environment_variables"), config: "environment_variables = []", empty: emptyList},
		{path: tfjsonpath.New("dependencies"), config: "dependencies = []", empty: emptyList},
	})
}

func TestAccMaintenanceWindowEmptyCollections(t *testing.T) {
	name := fmt.Sprintf("maintenance-window-empty-collections-%d", acctest.RandInt())
	testAccEmptyThenUnsetCollections(t, "checkly_maintenance_window.test", func(attributes string) string {
		return fmt.Sprintf(`
			resource "checkly_maintenance_window" "test" {
				name      = %q
				starts_at = "2099-01-01T09:00:00.000Z"
				ends_at   = "2099-01-01T10:00:00.000Z"
				%s
			}
		`, name, attributes) + tagsLengthOutput("checkly_maintenance_window.test")
	}, []emptyCollection{
		{path: tfjsonpath.New("tags"), config: "tags = []", empty: emptySet},
		{path: tfjsonpath.New("silence_alerts_tags"), config: "silence_alerts_tags = []", empty: emptySet},
	}, statecheck.ExpectKnownOutputValue("tags_length", knownvalue.Int64Exact(0)))
}

// The components of a component_configs entry are defaulted in the plan, as
// they are nested in a set element. An empty set means the same as leaving
// them out, so it is accepted together with affect_all_components = true.
func TestAccMaintenanceWindowEmptyStatusPageVisibilityCollections(t *testing.T) {
	// Requires an account with the status page maintenance windows
	// entitlement.
	rInt := acctest.RandInt()
	const r = "checkly_maintenance_window.test"
	visibility := tfjsonpath.New("status_page_visibility")
	testAccEmptyThenUnsetCollections(t, r, func(attributes string) string {
		return fmt.Sprintf(`
			resource "checkly_status_page_v3" "test" {
				name = "Maintenance Window Empty Collections %[1]d"
				url  = "mw-empty-collections-%[1]d"
			}

			resource "checkly_maintenance_window" "test" {
				name      = "Maintenance Window Empty Collections %[1]d"
				starts_at = "2099-01-01T09:00:00.000Z"
				ends_at   = "2099-01-01T10:00:00.000Z"

				status_page_visibility = {
					enabled         = true
					status_page_ids = [checkly_status_page_v3.test.id]

					component_configs = [{
						status_page_id        = checkly_status_page_v3.test.id
						affect_all_components = true
						%[2]s
					}]
				}
			}
		`, rInt, attributes)
	}, []emptyCollection{
		{path: visibility.AtMapKey("component_configs").AtSliceIndex(0).AtMapKey("components"), config: "components = []", empty: emptySet},
	},
		statecheck.ExpectKnownValue(r, visibility.AtMapKey("reminder_minutes_before"), emptySet),
		statecheck.ExpectKnownValue(r, visibility.AtMapKey("service_ids"), emptySet),
	)
}

// A group that enforces only public locations stores its private locations
// as empty, so importing it plans no change.
func TestAccCheckGroupV2EmptyCollections(t *testing.T) {
	name := fmt.Sprintf("group-v2-empty-collections-%d", acctest.RandInt())
	const r = "checkly_check_group_v2.test"
	testAccEmptyThenUnsetCollections(t, r, func(attributes string) string {
		return fmt.Sprintf(`
			resource "checkly_check_group_v2" "test" {
				name = %q

				enforce_locations = {
					enabled   = true
					locations = ["eu-central-1"]
				}
				%s
			}
		`, name, attributes) + tagsLengthOutput(r)
	}, []emptyCollection{
		{path: tfjsonpath.New("tags"), config: "tags = []", empty: emptySet},
		{path: tfjsonpath.New("environment_variables"), config: "environment_variables = []", empty: emptyList},
	},
		statecheck.ExpectKnownOutputValue("tags_length", knownvalue.Int64Exact(0)),
		statecheck.ExpectKnownValue(r, tfjsonpath.New("enforce_locations").AtMapKey("private_locations"), emptySet),
		statecheck.ExpectKnownValue(r, tfjsonpath.New("enforce_alert_settings").AtMapKey("alert_channel_subscriptions"), emptySet),
	)
}
