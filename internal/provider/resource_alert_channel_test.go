package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

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
				resource.TestCheckResourceAttr("checkly_alert_channel.webhook_ac", "webhook.headers.%", "0"),
				resource.TestCheckResourceAttr("checkly_alert_channel.webhook_ac", "webhook.query_parameters.%", "0"),
				resource.TestCheckNoResourceAttr("checkly_alert_channel.webhook_ac", "webhook.template"),
				resource.TestCheckNoResourceAttr("checkly_alert_channel.webhook_ac", "webhook.webhook_secret"),
				resource.TestCheckNoResourceAttr("checkly_alert_channel.webhook_ac", "webhook.webhook_type"),
			),
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
			Error: `An alert channel requires exactly one channel type attribute`,
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
			Error: `An alert channel cannot contain more than one channel type, but these are configured: email, sms.`,
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
