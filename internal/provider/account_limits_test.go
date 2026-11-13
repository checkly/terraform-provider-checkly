package provider

import (
	"context"
	"math/big"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/checkly/checkly-go-sdk/v2"
)

// accountClient serves a fixed account, or a fixed error, and counts the
// account requests.
type accountClient struct {
	checkly.Client
	account *checkly.Account
	err     error
	calls   atomic.Int32
}

func (c *accountClient) GetAccount(ctx context.Context) (*checkly.Account, error) {
	c.calls.Add(1)
	return c.account, c.err
}

func accountWithFeatures(features ...checkly.AccountFeature) *accountClient {
	return &accountClient{account: &checkly.Account{ID: "account", Features: features}}
}

func responseTimeResources(lookup *accountLookup) map[string]resource.ResourceWithModifyPlan {
	return map[string]resource.ResourceWithModifyPlan{
		"tcp_monitor": &TCPMonitorResource{account: lookup},
		"tcp_check":   &TCPCheckResource{TCPMonitorResource{account: lookup}},
		"url_monitor": &URLMonitorResource{account: lookup},
		"api_check":   &APICheckResource{account: lookup},
		"check":       &CheckResource{account: lookup},
	}
}

// modifyResponseTimePlan runs ModifyPlan with the given planned attribute
// values, also used as the configuration and the prior state, so the values
// are unchanged. Values of attributes that the resource doesn't have are
// left out. A nil values plans a destroy.
func modifyResponseTimePlan(t *testing.T, r resource.ResourceWithModifyPlan, values map[string]tftypes.Value) resource.ModifyPlanResponse {
	t.Helper()
	ctx := context.Background()

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema
	objectType := s.Type().TerraformType(ctx).(tftypes.Object)

	raw := tftypes.NewValue(objectType, nil)
	if values != nil {
		own := map[string]tftypes.Value{}
		for name, value := range values {
			if _, ok := objectType.AttributeTypes[name]; ok {
				own[name] = value
			}
		}
		raw = objectWith(objectType, own)
	}
	plan := tfsdk.Plan{Schema: s, Raw: raw}
	config := tfsdk.Config{Schema: s, Raw: raw}
	state := tfsdk.State{Schema: s, Raw: raw}

	resp := resource.ModifyPlanResponse{Plan: plan}
	r.ModifyPlan(ctx, resource.ModifyPlanRequest{Plan: plan, Config: config, State: state}, &resp)
	return resp
}

// responseTimes returns planned response times, and for checkly_check a type
// and a script. A nil checkType leaves the type out.
func responseTimes(checkType, degraded, maxValue any) map[string]tftypes.Value {
	number := func(v any) tftypes.Value {
		if n, ok := v.(int); ok {
			return tftypes.NewValue(tftypes.Number, big.NewFloat(float64(n)))
		}
		return tftypes.NewValue(tftypes.Number, v)
	}
	values := map[string]tftypes.Value{
		"degraded_response_time": number(degraded),
		"max_response_time":      number(maxValue),
	}
	if checkType != nil {
		values["type"] = tftypes.NewValue(tftypes.String, checkType)
		values["script"] = tftypes.NewValue(tftypes.String, "x")
	}
	return values
}

func TestValidateResponseTimeLimits(t *testing.T) {
	type want struct {
		errorsOn   []string
		warningsOn []string
		calls      int32
	}
	cases := map[string]struct {
		resource string
		client   *accountClient
		values   map[string]tftypes.Value
		want     want
	}{
		"extended url 40000":     {"url_monitor", accountWithFeatures(checkly.AccountFeatureExtendedResponseTimeLimits), responseTimes(nil, 3000, 40000), want{calls: 1}},
		"extended tcp 45000":     {"tcp_monitor", accountWithFeatures(checkly.AccountFeatureExtendedResponseTimeLimits), responseTimes(nil, 45000, 45000), want{calls: 1}},
		"standard url 40000":     {"url_monitor", accountWithFeatures(), responseTimes(nil, 3000, 40000), want{errorsOn: []string{"max_response_time"}, calls: 1}},
		"standard url 30000":     {"url_monitor", accountWithFeatures(), responseTimes(nil, 30000, 30000), want{}},
		"standard tcp 6000":      {"tcp_monitor", accountWithFeatures(), responseTimes(nil, 6000, 6000), want{errorsOn: []string{"degraded_response_time", "max_response_time"}, calls: 1}},
		"standard tcp 5000":      {"tcp_monitor", accountWithFeatures(), responseTimes(nil, 4000, 5000), want{}},
		"standard tcp check":     {"tcp_check", accountWithFeatures(), responseTimes(nil, 4000, 6000), want{errorsOn: []string{"max_response_time"}, calls: 1}},
		"standard api check":     {"api_check", accountWithFeatures(), responseTimes(nil, 40000, 40000), want{errorsOn: []string{"degraded_response_time", "max_response_time"}, calls: 1}},
		"standard api 30000":     {"api_check", accountWithFeatures(), responseTimes(nil, 5000, 30000), want{}},
		"standard check API":     {"check", accountWithFeatures(), responseTimes("API", 15000, 40000), want{errorsOn: []string{"max_response_time"}, calls: 1}},
		"extended check API":     {"check", accountWithFeatures(checkly.AccountFeatureExtendedResponseTimeLimits), responseTimes("API", 15000, 40000), want{calls: 1}},
		"standard check BROWSER": {"check", accountWithFeatures(), responseTimes("BROWSER", 15000, 40000), want{}},
		"unknown type check":     {"check", accountWithFeatures(), responseTimes(tftypes.UnknownValue, 15000, 40000), want{}},
		"unknown value":          {"url_monitor", accountWithFeatures(), responseTimes(nil, tftypes.UnknownValue, tftypes.UnknownValue), want{}},
		"lookup fails": {"url_monitor", &accountClient{err: &checkly.APIError{Method: http.MethodGet, Path: "/v1/accounts/me", StatusCode: http.StatusForbidden}},
			responseTimes(nil, 3000, 40000), want{warningsOn: []string{"max_response_time"}, calls: 1}},
		"destroy": {"url_monitor", accountWithFeatures(), nil, want{}},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			r := responseTimeResources(&accountLookup{client: c.client})[c.resource]
			resp := modifyResponseTimePlan(t, r, c.values)

			var errorsOn, warningsOn []string
			for _, d := range resp.Diagnostics {
				withPath, ok := d.(interface{ Path() path.Path })
				if !ok {
					t.Fatalf("diagnostic without a path: %v", d)
				}
				switch d.Severity() {
				case diag.SeverityError:
					if d.Summary() != "Response Time Above Account Limit" {
						t.Errorf("unexpected error: %s: %s", d.Summary(), d.Detail())
					}
					errorsOn = append(errorsOn, withPath.Path().String())
				default:
					warningsOn = append(warningsOn, withPath.Path().String())
				}
			}
			if strings.Join(errorsOn, ",") != strings.Join(c.want.errorsOn, ",") {
				t.Errorf("errors on %v, want %v: %v", errorsOn, c.want.errorsOn, resp.Diagnostics)
			}
			if strings.Join(warningsOn, ",") != strings.Join(c.want.warningsOn, ",") {
				t.Errorf("warnings on %v, want %v: %v", warningsOn, c.want.warningsOn, resp.Diagnostics)
			}
			if got := c.client.calls.Load(); got != c.want.calls {
				t.Errorf("account read %d times, want %d", got, c.want.calls)
			}
		})
	}
}

func TestValidateResponseTimeLimitsErrorText(t *testing.T) {
	r := &TCPMonitorResource{account: &accountLookup{client: accountWithFeatures()}}
	resp := modifyResponseTimePlan(t, r, responseTimes(nil, 4000, 6000))

	errs := resp.Diagnostics.Errors()
	if len(errs) != 1 {
		t.Fatalf("want one error: %v", resp.Diagnostics)
	}
	want := "`max_response_time` is `6000`, but can be at most `5000` on this account. Values up to `45000` need extended limits to be approved for your account; contact Checkly support to request them."
	if got := errs[0].Detail(); got != want {
		t.Errorf("detail = %q, want %q", got, want)
	}
}

// The account is read once per provider, however many resources and
// attributes need it.
func TestValidateResponseTimeLimitsReadsAccountOnce(t *testing.T) {
	client := accountWithFeatures()
	lookup := &accountLookup{client: client}

	errorCount := 0
	for _, r := range responseTimeResources(lookup) {
		resp := modifyResponseTimePlan(t, r, responseTimes("API", 40000, 40000))
		errorCount += resp.Diagnostics.ErrorsCount()
	}

	if errorCount != 10 {
		t.Errorf("errors = %d, want 2 for each of 5 resources", errorCount)
	}
	if got := client.calls.Load(); got != 1 {
		t.Errorf("account read %d times, want 1", got)
	}
}

// Each resource takes the account lookup from the provider data in
// Configure and checks its response times against it.
func TestResponseTimeLimitsConfigureWiring(t *testing.T) {
	newResources := map[string]func() resource.Resource{
		"tcp_monitor": NewTCPMonitorResource,
		"tcp_check":   NewTCPCheckResource,
		"url_monitor": NewURLMonitorResource,
		"api_check":   NewAPICheckResource,
		"check":       NewCheckResource,
	}
	accounts := map[string]struct {
		features  []checkly.AccountFeature
		wantError bool
	}{
		"standard": {wantError: true},
		"extended": {features: []checkly.AccountFeature{checkly.AccountFeatureExtendedResponseTimeLimits}},
	}

	for resourceName, newResource := range newResources {
		for accountName, a := range accounts {
			t.Run(resourceName+" "+accountName, func(t *testing.T) {
				client := accountWithFeatures(a.features...)
				r := newResource().(resource.ResourceWithModifyPlan)

				var configureResp resource.ConfigureResponse
				r.(resource.ResourceWithConfigure).Configure(context.Background(), resource.ConfigureRequest{ProviderData: newProviderData(client)}, &configureResp)
				if configureResp.Diagnostics.HasError() {
					t.Fatal(configureResp.Diagnostics)
				}

				resp := modifyResponseTimePlan(t, r, responseTimes("API", 4000, 40000))
				if got := resp.Diagnostics.HasError(); got != a.wantError {
					t.Errorf("error reported = %v, want %v: %v", got, a.wantError, resp.Diagnostics)
				}
				if got := client.calls.Load(); got != 1 {
					t.Errorf("account read %d times, want 1", got)
				}
			})
		}
	}
}

// An unconfigured provider, as during some validation runs, leaves the
// account lookup nil.
func TestValidateResponseTimeLimitsUnconfigured(t *testing.T) {
	for name, r := range responseTimeResources(nil) {
		t.Run(name, func(t *testing.T) {
			resp := modifyResponseTimePlan(t, r, responseTimes("API", 40000, 40000))
			if resp.Diagnostics.HasError() || resp.Diagnostics.WarningsCount() > 0 {
				t.Errorf("diagnostics: %v", resp.Diagnostics)
			}
		})
	}
}

// Response times above the standard limits apply on an account with
// extended response time limits and fail the plan on any other account. The
// test reads the account to find out which applies.
func TestAccResponseTimeAccountLimits(t *testing.T) {
	if os.Getenv(tfresource.EnvTfAcc) == "" {
		t.Skipf("acceptance test, set %s to run it", tfresource.EnvTfAcc)
	}
	testAccPreCheck(t)

	ctx, cancel := contextWithAPITimeout(context.Background())
	defer cancel()
	account, err := testAccClient().GetAccount(ctx)
	if err != nil {
		t.Fatalf("reading the account: %v", err)
	}
	extended := account.HasFeature(checkly.AccountFeatureExtendedResponseTimeLimits)
	t.Logf("account has extended response time limits: %v", extended)

	config := `
		resource "checkly_url_monitor" "test" {
			name              = "URL Monitor Extended Response Time"
			activated         = false
			locations         = ["eu-central-1"]
			max_response_time = 40000
			request = {
				url = "https://api.checklyhq.com"
			}
		}

		resource "checkly_tcp_monitor" "test" {
			name              = "TCP Monitor Extended Response Time"
			activated         = false
			locations         = ["eu-central-1"]
			max_response_time = 6000
			request = {
				hostname = "api.checklyhq.com"
				port     = 443
			}
		}
	`

	step := tfresource.TestStep{Config: config}
	if extended {
		step.ConfigStateChecks = []statecheck.StateCheck{
			statecheck.ExpectKnownValue("checkly_url_monitor.test", tfjsonpath.New("max_response_time"), knownvalue.Int64Exact(40000)),
			statecheck.ExpectKnownValue("checkly_tcp_monitor.test", tfjsonpath.New("max_response_time"), knownvalue.Int64Exact(6000)),
		}
	} else {
		step.ExpectError = regexp.MustCompile("Response Time Above Account Limit")
	}
	accTestCase(t, []tfresource.TestStep{step})
}
