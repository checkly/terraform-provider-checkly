package checkly

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccEmail(t *testing.T) {
	t.Parallel()
	accTestCase(t, []resource.TestStep{
		{
			Config: `resource "checkly_alert_channel" "t1" {
				email {
					address = "info@example.com"
				}
				send_recovery = false
				send_failure  = false
				send_degraded = false
				ssl_expiry    = false
				ssl_expiry_threshold = 10
				}`,
		},
	})
}

func TestAccSlack(t *testing.T) {
	t.Parallel()
	accTestCase(t, []resource.TestStep{
		{
			Config: `resource "checkly_alert_channel" "slack_ac" {
				slack {
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

func TestAccSMS(t *testing.T) {
	t.Parallel()
	accTestCase(t, []resource.TestStep{
		{
			Config: `resource "checkly_alert_channel" "sms_ac" {
				sms {
					name   = "smsalerts"
					number = "4917512345678"
				}
			}`,
		},
	})
}

func TestAccOpsgenie(t *testing.T) {
	t.Parallel()
	accTestCase(t, []resource.TestStep{
		{
			Config: `resource "checkly_alert_channel" "opsgenie_ac" {
				opsgenie {
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
	t.Parallel()
	accTestCase(t, []resource.TestStep{
		{
			Config: `resource "checkly_alert_channel" "pagerduty_ac" {
				pagerduty {
					account      = "checkly"
					service_key  = "key1"
					service_name = "pdalert"
				}
			}`,
		},
	})
}

func TestAccWebhook(t *testing.T) {
	t.Parallel()
	accTestCase(t, []resource.TestStep{
		{
			Config: `resource "checkly_alert_channel" "webhook_ac" {
				webhook {
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
	})
}

func TestAccFail(t *testing.T) {
	t.Parallel()
	cases := []struct {
		Config string
		Error  string
	}{
		{
			Config: `resource "checkly_alert_channel" "t1" {
				email {	}
			}`,
			Error: `The argument "address" is required`,
		},
		{
			Config: `resource "checkly_alert_channel" "t1" {
				sms {
				}
			}`,
			Error: `The argument "number" is required`,
		},
		{
			Config: `resource "checkly_alert_channel" "t1" {
				slack {
				}
			}`,
			Error: `The argument "channel" is required`,
		},
		{
			Config: `resource "checkly_alert_channel" "t1" {
				slack {
				}
			}`,
			Error: `Missing required argument`,
		},
		{
			Config: `resource "checkly_alert_channel" "t1" {
				webhook {
				}
			}`,
			Error: `The argument "name" is required`,
		},
		{
			Config: `resource "checkly_alert_channel" "t1" {
				webhook {
				}
			}`,
			Error: `The argument "url" is required`,
		},
		{
			Config: `resource "checkly_alert_channel" "t1" {
				opsgenie {
				}
			}`,
			Error: `The argument "api_key" is required`,
		},
		{
			Config: `resource "checkly_alert_channel" "t1" {
				opsgenie {
				}
			}`,
			Error: `The argument "priority" is required`,
		},
		{
			Config: `resource "checkly_alert_channel" "t1" {
				opsgenie {
				}
			}`,
			Error: `The argument "region" is required`,
		},
		{
			Config: `resource "checkly_alert_channel" "t1" {
				pagerduty {
				}
			}`,
			Error: `The argument "service_key" is required`,
		},
	}
	for key, tc := range cases {
		t.Run(fmt.Sprintf("%d", key), func(t *testing.T) {
			accTestCase(t, []resource.TestStep{
				{
					Config:      tc.Config,
					ExpectError: regexp.MustCompile(tc.Error),
				},
			})
		})
	}
}
