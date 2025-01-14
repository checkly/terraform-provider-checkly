package resources_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAlertChannelEmail(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),

		Steps: []resource.TestStep{
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
			},
		},
	})
}

func TestAccAlertChannelSlack(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),

		Steps: []resource.TestStep{
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
		},
	})
}

func TestAccAlertChannelSMS(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),

		Steps: []resource.TestStep{
			{
				Config: `resource "checkly_alert_channel" "sms_ac" {
				sms = {
					name   = "smsalerts"
					number = "4917512345678"
				}
			}`,
			},
		},
	})
}

func TestAccAlertChannelOpsgenie(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),

		Steps: []resource.TestStep{
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
		},
	})
}

func TestAccAlertChannelPagerduty(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),

		Steps: []resource.TestStep{
			{
				Config: `resource "checkly_alert_channel" "pagerduty_ac" {
				pagerduty = {
					account      = "checkly"
					service_key  = "key1"
					service_name = "pdalert"
				}
			}`,
			},
		},
	})
}

func TestAccAlertChannelWebhook(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),

		Steps: []resource.TestStep{
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
			},
		},
	})
}

func TestAccAlertChannelFail(t *testing.T) {
	cases := []struct {
		Config string
		Error  string
	}{
		{
			Config: `resource "checkly_alert_channel" "t1" {
				email = {	}
			}`,
			Error: `attribute[\s\S]+"address"[\s\S]+required`,
		},
		{
			Config: `resource "checkly_alert_channel" "t1" {
				sms = {
				}
			}`,
			Error: `attribute[\s\S]+"number"[\s\S]+required`,
		},
		{
			Config: `resource "checkly_alert_channel" "t1" {
				slack = {
				}
			}`,
			Error: `attribute[\s\S]+"url"[\s\S]+required`,
		},
		{
			Config: `resource "checkly_alert_channel" "t1" {
				slack = {
				}
			}`,
			Error: `attribute[\s\S]+"channel"[\s\S]+required`,
		},
		{
			Config: `resource "checkly_alert_channel" "t1" {
				webhook = {
				}
			}`,
			Error: `attribute[\s\S]+"name"[\s\S]+required`,
		},
		{
			Config: `resource "checkly_alert_channel" "t1" {
				webhook = {
				}
			}`,
			Error: `attribute[\s\S]+"url"[\s\S]+required`,
		},
		{
			Config: `resource "checkly_alert_channel" "t1" {
				opsgenie = {
				}
			}`,
			Error: `attribute[\s\S]+"api_key"[\s\S]+required`,
		},
		{
			Config: `resource "checkly_alert_channel" "t1" {
				opsgenie = {
				}
			}`,
			Error: `attribute[\s\S]+"priority"[\s\S]+required`,
		},
		{
			Config: `resource "checkly_alert_channel" "t1" {
				opsgenie = {
				}
			}`,
			Error: `attribute[\s\S]+"region"[\s\S]+required`,
		},
		{
			Config: `resource "checkly_alert_channel" "t1" {
				pagerduty = {
				}
			}`,
			Error: `attribute[\s\S]+"service_key"[\s\S]+required`,
		},
	}
	for key, tc := range cases {
		t.Run(fmt.Sprintf("%d", key), func(t *testing.T) {
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: protoV6ProviderFactories(),

				Steps: []resource.TestStep{
					{
						Config:      tc.Config,
						ExpectError: regexp.MustCompile(tc.Error),
					},
				},
			})
		})
	}
}
