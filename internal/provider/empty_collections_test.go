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

// The tests in this file cover optional collections nested in an object
// attribute: configured as empty, they are stored as empty; left out, they
// are stored as null. The plan after each apply must be empty, so neither
// value drifts against the API, which reports an unset collection as empty.

// emptyCollection names a collection attribute and the configuration that
// sets it to an empty value.
type emptyCollection struct {
	path   tfjsonpath.Path
	config string
	// empty is the expected state value when configured as empty.
	empty knownvalue.Check
}

var (
	emptySet = knownvalue.SetExact([]knownvalue.Check{})
	emptyMap = knownvalue.MapExact(map[string]knownvalue.Check{})
)

// testAccEmptyThenUnsetCollections applies config with every collection set
// to its empty value, then with all of them left out. config receives the
// attribute assignments to insert into the object that holds them.
func testAccEmptyThenUnsetCollections(t *testing.T, address string, config func(attributes string) string, collections []emptyCollection) {
	t.Helper()

	var attributes string
	var emptyChecks, nullChecks []statecheck.StateCheck
	for _, c := range collections {
		attributes += c.config + "\n"
		emptyChecks = append(emptyChecks, statecheck.ExpectKnownValue(address, c.path, c.empty))
		nullChecks = append(nullChecks, statecheck.ExpectKnownValue(address, c.path, knownvalue.Null()))
	}

	accTestCase(t, []resource.TestStep{
		{
			Config:            config(attributes),
			ConfigStateChecks: emptyChecks,
		},
		{
			Config:            config(""),
			ConfigStateChecks: nullChecks,
		},
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
