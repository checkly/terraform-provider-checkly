package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
)

func TestWebhookTypeValidation(t *testing.T) {
	v := stringvalidator.OneOf(alertChannelWebhookTypes.Values()...)

	validate := func(value string) []error {
		req := validator.StringRequest{
			Path:        path.Root("webhook_type"),
			ConfigValue: types.StringValue(value),
		}
		var resp validator.StringResponse
		v.ValidateString(context.Background(), req, &resp)

		var errs []error
		for _, d := range resp.Diagnostics.Errors() {
			errs = append(errs, fmt.Errorf("%s: %s", d.Summary(), d.Detail()))
		}
		return errs
	}

	valid := []string{
		"WEBHOOK_DISCORD",
		"WEBHOOK_FIREHYDRANT",
		"WEBHOOK_GITLAB_ALERT",
		"WEBHOOK_ROOTLY",
		"WEBHOOK_SPIKESH",
		"WEBHOOK_SPLUNK",
		"WEBHOOK_MSTEAMS",
		"WEBHOOK_TELEGRAM",
	}
	for _, value := range valid {
		if errs := validate(value); len(errs) > 0 {
			t.Errorf("expected %q to be valid, got errors: %v", value, errs)
		}
	}

	if errs := validate("WEBHOOK_UNKNOWN"); len(errs) == 0 {
		t.Error("expected 'WEBHOOK_UNKNOWN' to be invalid, but got no errors")
	}
}

func TestAlertChannelConfigValidatorsExactlyOneChannel(t *testing.T) {
	ctx := context.Background()

	var schemaResp fwresource.SchemaResponse
	NewAlertChannelResource().Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	objectType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)

	email := objectWith(objectType.AttributeTypes["email"].(tftypes.Object), map[string]tftypes.Value{
		"address": tftypes.NewValue(tftypes.String, "info@example.com"),
	})
	sms := objectWith(objectType.AttributeTypes["sms"].(tftypes.Object), map[string]tftypes.Value{
		"name":   tftypes.NewValue(tftypes.String, "smsalerts"),
		"number": tftypes.NewValue(tftypes.String, "+4917512345678"),
	})
	unknownSlack := tftypes.NewValue(objectType.AttributeTypes["slack"], tftypes.UnknownValue)
	emailPath := path.Root("email")
	const wantDetail = "Exactly one of these attributes must be configured: [email,slack,slack_app,sms,call,webhook,opsgenie,pagerduty]"

	cases := map[string]struct {
		values      map[string]tftypes.Value
		wantSummary string
		wantPath    *path.Path
	}{
		"none": {
			wantSummary: "Missing Attribute Configuration",
		},
		"one": {
			values: map[string]tftypes.Value{"email": email},
		},
		"two": {
			values:      map[string]tftypes.Value{"email": email, "sms": sms},
			wantSummary: "Invalid Attribute Combination",
			wantPath:    &emailPath,
		},
		"two and unknown": {
			values:      map[string]tftypes.Value{"email": email, "sms": sms, "slack": unknownSlack},
			wantSummary: "Invalid Attribute Combination",
			wantPath:    &emailPath,
		},
		"only unknown": {
			values: map[string]tftypes.Value{"slack": unknownSlack},
		},
		"one and unknown": {
			values: map[string]tftypes.Value{"email": email, "slack": unknownSlack},
		},
	}

	r := NewAlertChannelResource().(fwresource.ResourceWithConfigValidators)
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			config := tfsdk.Config{Schema: schemaResp.Schema, Raw: objectWith(objectType, tc.values)}

			var diags []string
			var resp fwresource.ValidateConfigResponse
			for _, v := range r.ConfigValidators(ctx) {
				v.ValidateResource(ctx, fwresource.ValidateConfigRequest{Config: config}, &resp)
			}
			for _, d := range resp.Diagnostics.Errors() {
				diags = append(diags, d.Summary()+": "+d.Detail())
			}

			if tc.wantSummary == "" {
				if len(diags) != 0 {
					t.Fatalf("unexpected errors: %v", diags)
				}
				return
			}
			if len(diags) != 1 {
				t.Fatalf("got %d errors, want 1: %v", len(diags), diags)
			}
			got := resp.Diagnostics.Errors()[0]
			if got.Summary() != tc.wantSummary || got.Detail() != wantDetail {
				t.Errorf("got %q: %q, want %q: %q", got.Summary(), got.Detail(), tc.wantSummary, wantDetail)
			}
			if tc.wantPath != nil {
				withPath, ok := got.(diag.DiagnosticWithPath)
				if !ok || !withPath.Path().Equal(*tc.wantPath) {
					t.Errorf("diagnostic %v is not anchored at %s", got, *tc.wantPath)
				}
			}
		})
	}
}

func TestAccEmail(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `resource "checkly_alert_channel" "t1" {
				email = {
					address = "info@example.com"
				}
				send_recovery = false
				send_failure  = false
				send_degraded = false
				ssl_expiry    = false
				ssl_expiry_threshold = 10
				}`,
			Check: resource.TestCheckResourceAttr("checkly_alert_channel.t1", "email.address", "info@example.com"),
		},
	})
}

func TestAccSlack(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `resource "checkly_alert_channel" "slack_ac" {
				slack = {
					channel = "checkly_alerts"
					url     = "https://hooks.slack.com/services/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"
				}
				send_recovery        = true
				send_failure         = true
				send_degraded        = false
				ssl_expiry           = true
				ssl_expiry_threshold = 11
			}`,
		},
	})
}

func TestAccSlackApp(t *testing.T) {
	accTestCaseWithErrorCheck(t, []resource.TestStep{
		{
			Config: `resource "checkly_alert_channel" "slack_app_ac" {
				slack_app = {
					slack_channels = ["#terraform-e2e-slack-app-alert-target"]
				}
				send_recovery        = true
				send_failure         = true
				send_degraded        = false
				ssl_expiry           = true
				ssl_expiry_threshold = 11
			}`,
		},
	}, func(err error) error {
		// A SLACK_APP alert channel can only be created on accounts that have
		// connected the Checkly Slack App. Without it the API responds with a
		// slack_integration_not_connected error, which is account setup rather
		// than a provider defect, so skip instead of failing.
		if err != nil && strings.Contains(err.Error(), "slack_integration_not_connected") {
			t.Skipf("Slack App integration is not connected on this account; skipping: %v", err)
		}
		return err
	})
}

func TestAccSMS(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `resource "checkly_alert_channel" "sms_ac" {
				sms = {
					name   = "smsalerts"
					number = "+4917512345678"
				}
			}`,
		},
	})
}

func TestAccOpsgenie(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `resource "checkly_alert_channel" "opsgenie_ac" {
				opsgenie = {
					name     = "opsalert"
					api_key  = "key1"
					region   = "EU"
					priority = "P1"
				}
			}`,
		},
	})
}

func TestAccPagerduty(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `resource "checkly_alert_channel" "pagerduty_ac" {
				pagerduty = {
					account      = "checkly"
					service_key  = "key1"
					service_name = "pdalert"
				}
			}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_alert_channel.pagerduty_ac", "pagerduty.account", "checkly"),
				resource.TestCheckResourceAttr("checkly_alert_channel.pagerduty_ac", "pagerduty.service_name", "pdalert"),
			),
		},
	})
}

func TestAccWebhook(t *testing.T) {
	accTestCase(t, []resource.TestStep{
		{
			Config: `resource "checkly_alert_channel" "webhook_ac" {
				webhook = {
				  name   = "webhhookalerts"
				  method = "get"
				  headers  = {
					X-HEADER-1 = "foo"
				  }
				  query_parameters = {
					query1 = "bar"
				  }
				  template       = "tmpl"
				  url            = "https://example.com/webhook"
				  webhook_secret = "foo-secret"
				}
			  }`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_alert_channel.webhook_ac", "webhook.name", "webhhookalerts"),
				resource.TestCheckResourceAttr("checkly_alert_channel.webhook_ac", "webhook.method", "get"),
				resource.TestCheckResourceAttr("checkly_alert_channel.webhook_ac", "webhook.headers.X-HEADER-1", "foo"),
				resource.TestCheckResourceAttr("checkly_alert_channel.webhook_ac", "webhook.query_parameters.query1", "bar"),
				resource.TestCheckResourceAttr("checkly_alert_channel.webhook_ac", "webhook.template", "tmpl"),
				resource.TestCheckNoResourceAttr("checkly_alert_channel.webhook_ac", "email"),
			),
		},
		// Unset optional attributes take their defaults or stay null.
		{
			Config: `resource "checkly_alert_channel" "webhook_ac" {
				webhook = {
				  name = "webhhookalerts"
				  url  = "https://example.com/webhook"
				}
			  }`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_alert_channel.webhook_ac", "webhook.method", "POST"),
				resource.TestCheckNoResourceAttr("checkly_alert_channel.webhook_ac", "webhook.template"),
				resource.TestCheckNoResourceAttr("checkly_alert_channel.webhook_ac", "webhook.webhook_secret"),
				resource.TestCheckNoResourceAttr("checkly_alert_channel.webhook_ac", "webhook.webhook_type"),
			),
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue("checkly_alert_channel.webhook_ac", tfjsonpath.New("webhook").AtMapKey("headers"), emptyMap),
				statecheck.ExpectKnownValue("checkly_alert_channel.webhook_ac", tfjsonpath.New("webhook").AtMapKey("query_parameters"), emptyMap),
			},
		},
		// Optional strings configured as "" stay empty instead of refreshing
		// to null.
		{
			Config: `resource "checkly_alert_channel" "webhook_ac" {
				webhook = {
				  name           = "webhhookalerts"
				  url            = "https://example.com/webhook"
				  template       = ""
				  webhook_secret = ""
				}
			  }`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("checkly_alert_channel.webhook_ac", "webhook.template", ""),
				resource.TestCheckResourceAttr("checkly_alert_channel.webhook_ac", "webhook.webhook_secret", ""),
			),
		},
		{
			Config: `resource "checkly_alert_channel" "webhook_ac" {
				webhook = {
				  name = "webhhookalerts"
				  url  = "https://example.com/webhook"
				}
			  }`,
		},
		{
			ResourceName:      "checkly_alert_channel.webhook_ac",
			ImportState:       true,
			ImportStateVerify: true,
		},
	})
}

func TestAccFail(t *testing.T) {
	cases := []struct {
		Config string
		Error  string
	}{
		{
			Config: `resource "checkly_alert_channel" "t1" {
				email = {}
			}`,
			Error: `Inappropriate value for attribute "email": attribute "address" is required.`,
		},
		{
			Config: `resource "checkly_alert_channel" "t1" {
				sms = {}
			}`,
			Error: `Inappropriate value for attribute "sms": attributes "name" and "number" are required.`,
		},
		{
			Config: `resource "checkly_alert_channel" "t1" {
				slack = {}
			}`,
			Error: `Inappropriate value for attribute "slack": attributes "channel" and "url" are required.`,
		},
		{
			Config: `resource "checkly_alert_channel" "t1" {
				webhook = {}
			}`,
			Error: `Inappropriate value for attribute "webhook": attributes "name" and "url" are required.`,
		},
		{
			Config: `resource "checkly_alert_channel" "t1" {
				opsgenie = {}
			}`,
			Error: `Inappropriate value for attribute "opsgenie": attributes "api_key", "name", "priority", and "region" are required.`,
		},
		{
			Config: `resource "checkly_alert_channel" "t1" {
				pagerduty = {}
			}`,
			Error: `Inappropriate value for attribute "pagerduty": attribute "service_key" is required.`,
		},
		{
			Config: `resource "checkly_alert_channel" "t1" {
			}`,
			Error: `Exactly one of these attributes must be configured: [email,slack,slack_app,sms,call,webhook,opsgenie,pagerduty]`,
		},
		{
			Config: `resource "checkly_alert_channel" "t1" {
				email = {
					address = "info@example.com"
				}
				sms = {
					name   = "smsalerts"
					number = "+4917512345678"
				}
			}`,
			Error: `Invalid Attribute Combination`,
		},
	}
	for key, tc := range cases {
		t.Run(fmt.Sprintf("%d", key), func(t *testing.T) {
			accTestCase(t, []resource.TestStep{
				{
					Config:      tc.Config,
					ExpectError: wrappedMessageRegexp(tc.Error),
				},
			})
		})
	}
}

// wrappedMessageRegexp matches message in Terraform's diagnostic output,
// which wraps long lines at arbitrary spaces.
func wrappedMessageRegexp(message string) *regexp.Regexp {
	words := strings.Fields(message)
	for i, word := range words {
		words[i] = regexp.QuoteMeta(word)
	}
	return regexp.MustCompile(strings.Join(words, `\s+`))
}
