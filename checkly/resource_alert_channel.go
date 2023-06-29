package checkly

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/checkly/checkly-go-sdk"
)

const (
	AcFieldEmail                = "email"
	AcFieldEmailAddress         = "address"
	AcFieldSlack                = "slack"
	AcFieldSlackURL             = "url"
	AcFieldSlackChannel         = "channel"
	AcFieldSMS                  = "sms"
	AcFieldSMSName              = "name"
	AcFieldSMSNumber            = "number"
	AcFieldWebhook              = "webhook"
	AcFieldWebhookName          = "name"
	AcFieldWebhookMethod        = "method"
	AcFieldWebhookHeaders       = "headers"
	AcFieldWebhookQueryParams   = "query_parameters"
	AcFieldWebhookTemplate      = "template"
	AcFieldWebhookURL           = "url"
	AcFieldWebhookSecret        = "webhook_secret"
	AcFieldWebhookType          = "webhook_type"
	AcFieldOpsgenie             = "opsgenie"
	AcFieldOpsgenieName         = "name"
	AcFieldOpsgenieAPIKey       = "api_key"
	AcFieldOpsgenieRegion       = "region"
	AcFieldOpsgeniePriority     = "priority"
	AcFieldPagerduty            = "pagerduty"
	AcFieldPagerdutyAccount     = "account"
	AcFieldPagerdutyServiceKey  = "service_key"
	AcFieldPagerdutyServiceName = "service_name"
	AcFieldSendRecovery         = "send_recovery"
	AcFieldSendFailure          = "send_failure"
	AcFieldSendDegraded         = "send_degraded"
	AcFieldSSLExpiry            = "ssl_expiry"
	AcFieldSSLExpiryThreshold   = "ssl_expiry_threshold"
	AcFieldCall                 = "call"
	AcFieldCallName             = "name"
	AcFieldCallNumber           = "number"
)

func resourceAlertChannel() *schema.Resource {
	return &schema.Resource{
		Description: "Allows you to define alerting channels for the checks and groups in your account",
		Create:      resourceAlertChannelCreate,
		Read:        resourceAlertChannelRead,
		Update:      resourceAlertChannelUpdate,
		Delete:      resourceAlertChannelDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			AcFieldEmail: {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						AcFieldEmailAddress: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The email address of this email alert channel.",
						},
					},
				},
			},
			AcFieldSlack: {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						AcFieldSlackURL: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The Slack webhook URL",
						},
						AcFieldSlackChannel: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The name of the alert's Slack channel",
						},
					},
				},
			},
			AcFieldSMS: {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						AcFieldSMSName: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The name of this alert channel",
						},
						AcFieldSMSNumber: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The mobile number to receive the alerts",
						},
					},
				},
			},
			AcFieldCall: {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						AcFieldCallName: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The name of this alert channel",
						},
						AcFieldCallNumber: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The mobile number to receive the alerts",
						},
					},
				},
			},
			AcFieldWebhook: {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						AcFieldWebhookName: {
							Type:     schema.TypeString,
							Required: true,
						},
						AcFieldWebhookMethod: {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "POST",
							Description: "(Default `POST`)",
						},
						AcFieldWebhookHeaders: {
							Type:     schema.TypeMap,
							Optional: true,
							Computed: true,
							DefaultFunc: func() (interface{}, error) {
								return []tfMap{}, nil
							},
						},
						AcFieldWebhookQueryParams: {
							Type:     schema.TypeMap,
							Optional: true,
							Computed: true,
							DefaultFunc: func() (interface{}, error) {
								return []tfMap{}, nil
							},
						},
						AcFieldWebhookTemplate: {
							Type:     schema.TypeString,
							Optional: true,
						},
						AcFieldWebhookURL: {
							Type:     schema.TypeString,
							Required: true,
						},
						AcFieldWebhookSecret: {
							Type:     schema.TypeString,
							Optional: true,
						},
						AcFieldWebhookType: {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Type of the webhook. Possible values are 'WEBHOOK_DISCORD', 'WEBHOOK_FIREHYDRANT', 'WEBHOOK_GITLAB_ALERT', 'WEBHOOK_SPIKESH', 'WEBHOOK_SPLUNK', 'WEBHOOK_MSTEAMS' and 'WEBHOOK_TELEGRAM'.",
							ValidateFunc: func(value interface{}, key string) (warns []string, errs []error) {
								v := value.(string)
								isValid := false
								options := []string{"WEBHOOK_DISCORD", "WEBHOOK_FIREHYDRANT", "WEBHOOK_GITLAB_ALERT", "WEBHOOK_SPIKESH", "WEBHOOK_SPLUNK", "WEBHOOK_MSTEAMS", "WEBHOOK_TELEGRAM"}
								for _, option := range options {
									if v == option {
										isValid = true
									}
								}
								if !isValid {
									errs = append(errs, fmt.Errorf("%q must be one of %v, got %s", key, options, v))
								}
								return warns, errs
							},
						},
					},
				},
			},
			AcFieldOpsgenie: {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						AcFieldOpsgenieName: {
							Type:     schema.TypeString,
							Required: true,
						},
						AcFieldOpsgenieAPIKey: {
							Type:     schema.TypeString,
							Required: true,
						},
						AcFieldOpsgenieRegion: {
							Type:     schema.TypeString,
							Required: true,
						},
						AcFieldOpsgeniePriority: {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			AcFieldPagerduty: {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						AcFieldPagerdutyServiceKey: {
							Type:     schema.TypeString,
							Required: true,
						},
						AcFieldPagerdutyServiceName: {
							Type:     schema.TypeString,
							Optional: true,
						},
						AcFieldPagerdutyAccount: {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			AcFieldSendRecovery: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "(Default `true`)",
			},
			AcFieldSendFailure: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "(Default `true`)",
			},
			AcFieldSendDegraded: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "(Default `false`)",
			},
			AcFieldSSLExpiry: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "(Default `false`)",
			},
			AcFieldSSLExpiryThreshold: {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  30,
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					min := 1
					max := 30
					v := val.(int)
					if v < min || v > max {
						errs = append(errs, fmt.Errorf("%q must be between %d and  %d, got: %d", key, min, max, v))
					}
					return warns, errs
				},
				Description: "Value must be between 1 and 30 (Default `30`)",
			},
		},
	}
}

func resourceAlertChannelCreate(d *schema.ResourceData, client interface{}) error {
	ac, err := alertChannelFromResourceData(d)
	if err != nil {
		return makeError("resourceAlertChannelCreate.1", &ErrorLog{"err": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	resp, err := client.(checkly.Client).CreateAlertChannel(ctx, ac)
	if err != nil {
		cjson, _ := json.Marshal(ac.GetConfig())
		return makeError("resourceAlertChannelCreate.2", &ErrorLog{
			"err":    err.Error(),
			"type":   ac.Type,
			"config": string(cjson),
		})
	}
	d.SetId(fmt.Sprintf("%d", resp.ID))
	return resourceAlertChannelRead(d, client)
}

func resourceAlertChannelRead(d *schema.ResourceData, client interface{}) error {
	ID, err := resourceIDToInt(d.Id())
	if err != nil {
		return makeError("resourceAlertChannelRead.1", &ErrorLog{"err": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	ac, err := client.(checkly.Client).GetAlertChannel(ctx, ID)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			//if resource is deleted remotely, then mark it as
			//successfully gone by unsetting it's ID
			d.SetId("")
			return nil
		}
		return makeError("resourceAlertChannelRead.2", &ErrorLog{"err": err.Error()})
	}
	return resourceDataFromAlertChannel(ac, d)
}

func resourceAlertChannelUpdate(d *schema.ResourceData, client interface{}) error {
	ac, err := alertChannelFromResourceData(d)
	if err != nil {
		return makeError("resourceAlertChannelUpdate.1", &ErrorLog{"err": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	_, err = client.(checkly.Client).UpdateAlertChannel(ctx, ac.ID, ac)
	if err != nil {
		return makeError("resourceAlertChannelUpdate.2", &ErrorLog{"err": err.Error()})
	}
	d.SetId(fmt.Sprintf("%d", ac.ID))
	return resourceAlertChannelRead(d, client)
}

func resourceAlertChannelDelete(d *schema.ResourceData, client interface{}) error {
	ID, err := resourceIDToInt(d.Id())
	if err != nil {
		return makeError("resourceAlertChannelDelete.1", &ErrorLog{"err": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	if err := client.(checkly.Client).DeleteAlertChannel(ctx, ID); err != nil {
		return makeError("resourceAlertChannelDelete.2", &ErrorLog{"err": err.Error()})
	}
	return nil
}

func resourceDataFromAlertChannel(it *checkly.AlertChannel, d *schema.ResourceData) error {
	d.Set(AcFieldEmail, setFromEmail(it.Email))
	d.Set(AcFieldSMS, setFromSMS(it.SMS))
	d.Set(AcFieldCall, setFromCall(it.CALL))
	d.Set(AcFieldSlack, setFromSlack(it.Slack))
	d.Set(AcFieldWebhook, setFromWebhook(it.Webhook))
	d.Set(AcFieldOpsgenie, setFromOpsgenie(it.Opsgenie))
	d.Set(AcFieldPagerduty, setFromPagerduty(it.Pagerduty))
	if it.SendRecovery != nil {
		d.Set(AcFieldSendRecovery, *it.SendRecovery)
	}
	if it.SendFailure != nil {
		d.Set(AcFieldSendFailure, *it.SendFailure)
	}
	if it.SendDegraded != nil {
		d.Set(AcFieldSendDegraded, *it.SendDegraded)
	}
	if it.SSLExpiry != nil {
		d.Set(AcFieldSSLExpiry, *it.SSLExpiry)
	}
	if it.SSLExpiryThreshold != nil {
		d.Set(AcFieldSSLExpiryThreshold, *it.SSLExpiryThreshold)
	}
	return nil
}

func alertChannelFromResourceData(d *schema.ResourceData) (checkly.AlertChannel, error) {
	ac := checkly.AlertChannel{}
	ID, err := resourceIDToInt(d.Id())
	if err != nil {
		return ac, makeError("alertChannelFromResourceData.1", &ErrorLog{"err": err.Error()})
	}
	if err == nil {
		ac.ID = ID
	}

	sendRecovery := d.Get(AcFieldSendRecovery).(bool)
	ac.SendRecovery = &sendRecovery

	sendFailure := d.Get(AcFieldSendFailure).(bool)
	ac.SendFailure = &sendFailure

	sndDegraded := d.Get(AcFieldSendDegraded).(bool)
	ac.SendDegraded = &sndDegraded

	sslExpiry := d.Get(AcFieldSSLExpiry).(bool)
	ac.SSLExpiry = &sslExpiry

	if v, ok := d.GetOk(AcFieldSSLExpiryThreshold); ok {
		i := v.(int)
		ac.SSLExpiryThreshold = &i
	}

	fields := []string{AcFieldEmail, AcFieldSMS, AcFieldCall, AcFieldSlack, AcFieldWebhook, AcFieldOpsgenie, AcFieldPagerduty}
	setCount := 0
	for _, field := range fields {
		cfgSet := (d.Get(field)).(*schema.Set)
		if cfgSet.Len() > 0 {
			ac.Type = strings.ToUpper(field)
			c, err := alertChannelConfigFromSet(ac.Type, cfgSet)
			if err != nil {
				return ac, makeError("alertChannelFromResourceData.2", &ErrorLog{"err": err.Error()})
			}
			ac.SetConfig(c)
			setCount++
		}
	}
	if setCount > 1 {
		return ac, makeError("Alert-Channel config can't contain more than one Channel", nil)
	}
	return ac, nil
}

func alertChannelConfigFromSet(channelType string, s *schema.Set) (interface{}, error) {
	if s.Len() == 0 {
		return nil, nil
	}
	cfg := s.List()[0].(map[string]interface{})
	switch channelType {
	case checkly.AlertTypeEmail:
		return &checkly.AlertChannelEmail{
			Address: cfg[AcFieldEmailAddress].(string),
		}, nil
	case checkly.AlertTypeSMS:
		return &checkly.AlertChannelSMS{
			Name:   cfg[AcFieldSMSName].(string),
			Number: cfg[AcFieldSMSNumber].(string),
		}, nil
	case checkly.AlertTypeCall:
		return &checkly.AlertChannelCall{
			Name:   cfg[AcFieldCallName].(string),
			Number: cfg[AcFieldCallNumber].(string),
		}, nil
	case checkly.AlertTypeSlack:
		return &checkly.AlertChannelSlack{
			Channel:    cfg[AcFieldSlackChannel].(string),
			WebhookURL: cfg[AcFieldSlackURL].(string),
		}, nil
	case checkly.AlertTypeOpsgenie:
		return &checkly.AlertChannelOpsgenie{
			Name:     cfg[AcFieldOpsgenieName].(string),
			APIKey:   cfg[AcFieldOpsgenieAPIKey].(string),
			Region:   cfg[AcFieldOpsgenieRegion].(string),
			Priority: cfg[AcFieldOpsgeniePriority].(string),
		}, nil
	case checkly.AlertTypePagerduty:
		return &checkly.AlertChannelPagerduty{
			Account:     cfg[AcFieldPagerdutyAccount].(string),
			ServiceKey:  cfg[AcFieldPagerdutyServiceKey].(string),
			ServiceName: cfg[AcFieldPagerdutyServiceName].(string),
		}, nil
	case checkly.AlertTypeWebhook:
		return &checkly.AlertChannelWebhook{
			Name:            cfg[AcFieldWebhookName].(string),
			Method:          cfg[AcFieldWebhookMethod].(string),
			Template:        cfg[AcFieldWebhookTemplate].(string),
			URL:             cfg[AcFieldWebhookURL].(string),
			WebhookSecret:   cfg[AcFieldWebhookSecret].(string),
			WebhookType:     cfg[AcFieldWebhookType].(string),
			Headers:         keyValuesFromMap(cfg[AcFieldWebhookHeaders].(tfMap)),
			QueryParameters: keyValuesFromMap(cfg[AcFieldWebhookQueryParams].(tfMap)),
		}, nil
	}
	return nil, makeError("alertChannelConfigFromSet:unkownType", nil)
}

func setFromEmail(cfg *checkly.AlertChannelEmail) []tfMap {
	if cfg == nil {
		return []tfMap{}
	}
	return []tfMap{
		{
			AcFieldEmailAddress: cfg.Address,
		},
	}
}

func setFromSMS(cfg *checkly.AlertChannelSMS) []tfMap {
	if cfg == nil {
		return []tfMap{}
	}
	return []tfMap{
		{
			AcFieldSMSName:   cfg.Name,
			AcFieldSMSNumber: cfg.Number,
		},
	}
}

func setFromCall(cfg *checkly.AlertChannelCall) []tfMap {
	if cfg == nil {
		return []tfMap{}
	}
	return []tfMap{
		{
			AcFieldCallName:   cfg.Name,
			AcFieldCallNumber: cfg.Number,
		},
	}
}

func setFromSlack(cfg *checkly.AlertChannelSlack) []tfMap {
	if cfg == nil {
		return []tfMap{}
	}
	return []tfMap{
		{
			AcFieldSlackChannel: cfg.Channel,
			AcFieldSlackURL:     cfg.WebhookURL,
		},
	}
}

func setFromWebhook(cfg *checkly.AlertChannelWebhook) []tfMap {
	if cfg == nil {
		return []tfMap{}
	}
	return []tfMap{
		{
			AcFieldWebhookName:        cfg.Name,
			AcFieldWebhookMethod:      cfg.Method,
			AcFieldWebhookHeaders:     mapFromKeyValues(cfg.Headers),
			AcFieldWebhookQueryParams: mapFromKeyValues(cfg.QueryParameters),
			AcFieldWebhookTemplate:    cfg.Template,
			AcFieldWebhookURL:         cfg.URL,
			AcFieldWebhookSecret:      cfg.WebhookSecret,
			AcFieldWebhookType:        cfg.WebhookType,
		},
	}
}

func setFromOpsgenie(cfg *checkly.AlertChannelOpsgenie) []tfMap {
	if cfg == nil {
		return []tfMap{}
	}
	return []tfMap{
		{
			AcFieldOpsgenieName:     cfg.Name,
			AcFieldOpsgenieAPIKey:   cfg.APIKey,
			AcFieldOpsgenieRegion:   cfg.Region,
			AcFieldOpsgeniePriority: cfg.Priority,
		},
	}
}

func setFromPagerduty(cfg *checkly.AlertChannelPagerduty) []tfMap {
	if cfg == nil {
		return []tfMap{}
	}
	return []tfMap{
		{
			AcFieldPagerdutyAccount:     cfg.Account,
			AcFieldPagerdutyServiceKey:  cfg.ServiceKey,
			AcFieldPagerdutyServiceName: cfg.ServiceName,
		},
	}
}
