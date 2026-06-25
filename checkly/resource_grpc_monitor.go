package checkly

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceGRPCMonitor() *schema.Resource {
	return &schema.Resource{
		Create: resourceGRPCMonitorCreate,
		Read:   resourceGRPCMonitorRead,
		Update: resourceGRPCMonitorUpdate,
		Delete: resourceGRPCMonitorDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "gRPC monitors allow you to monitor gRPC services, either via a health check (HEALTH mode) or by invoking a unary method (BEHAVIOR mode).",
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the monitor.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A description of the monitor.",
			},
			frequencyAttributeName: makeFrequencyAttributeSchema(FrequencyAttributeSchemaOptions{
				Monitor:            true,
				AllowHighFrequency: true,
			}),
			frequencyOffsetAttributeName: makeFrequencyOffsetAttributeSchema(FrequencyOffsetAttributeSchemaOptions{
				Monitor: true,
			}),
			"activated": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Determines if the monitor is running or not. Possible values `true`, and `false`.",
			},
			"muted": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Determines if any notifications will be sent out when a monitor fails/degrades/recovers.",
			},
			"should_fail": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Allows to invert the behaviour of when a monitor is considered to fail.",
			},
			"run_parallel": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Determines if the monitor should run in all selected locations in parallel or round-robin.",
			},
			"locations": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "An array of one or more data center locations where to run this monitor. (Default [\"us-east-1\"])",
			},
			"degraded_response_time": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      4000,
				ValidateFunc: validateBetween(0, 30000),
				Description:  "The response time in milliseconds starting from which a monitor should be considered degraded. Possible values are between 0 and 30000. (Default `4000`).",
			},
			"max_response_time": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      5000,
				ValidateFunc: validateBetween(0, 30000),
				Description:  "The response time in milliseconds starting from which a monitor should be considered failing. Possible values are between 0 and 30000. (Default `5000`).",
			},
			"tags": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "A list of tags for organizing and filtering checks and monitors.",
			},
			"runtime_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     nil,
				Description: "The ID of the runtime to use for this monitor.",
			},
			alertChannelSubscriptionAttributeName: makeAlertChannelSubscriptionAttributeSchema(AlertChannelSubscriptionAttributeSchemaOptions{
				Monitor: true,
			}),
			"private_locations": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				DefaultFunc: func() (interface{}, error) {
					return []tfMap{}, nil
				},
				Description: "An array of one or more private locations slugs.",
			},
			alertSettingsAttributeName: makeAlertSettingsAttributeSchema(AlertSettingsAttributeSchemaOptions{
				Monitor: true,
			}),
			"use_global_alert_settings": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "When true, the account level alert settings will be used, not the alert setting defined on this monitor.",
			},
			"request": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"host": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The host to connect to. Do not include a scheme or a port in this value.",
						},
						"port": {
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validateBetween(1, 65535),
							Description:  "The port number to connect to. Possible values are between 1 and 65535.",
						},
						"ip_family": {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "IPv4",
							ValidateFunc: validateOneOf([]string{"IPv4", "IPv6"}),
							Description:  "The IP family to use when executing the gRPC check. The value can be either `IPv4` or `IPv6`. (Default `IPv4`).",
						},
						"grpc_mode": {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "BEHAVIOR",
							ValidateFunc: validateOneOf([]string{"BEHAVIOR", "HEALTH"}),
							Description:  "The gRPC monitoring mode. `BEHAVIOR` invokes a unary method (requires `method`); `HEALTH` queries the standard gRPC health-check service (allows `service`). (Default `BEHAVIOR`).",
						},
						"tls": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Whether to use a TLS-encrypted connection to the gRPC server. (Default `true`).",
						},
						"skip_ssl": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Whether to skip SSL certificate validation when `tls` is enabled. (Default `false`).",
						},
						"timeout": {
							Type:         schema.TypeInt,
							Optional:     true,
							Computed:     true,
							ValidateFunc: validateBetween(1, 180),
							Description:  "The number of seconds to wait for the gRPC call to complete before timing out. Possible values are between 1 and 180. (Default `60`).",
						},
						"store_response_body": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Whether to store the gRPC response body with the check result. (Default `true`).",
						},
						"service": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The service name to query in `HEALTH` mode. An empty value queries overall server health. Forbidden in `BEHAVIOR` mode.",
						},
						"service_definition": {
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							ValidateFunc: validateOneOf([]string{"REFLECTION", "PROTO_FILE"}),
							Description:  "How the service definition is resolved in `BEHAVIOR` mode: `REFLECTION` uses server reflection; `PROTO_FILE` uses the inline `proto_content`. (Default `REFLECTION`).",
						},
						"proto_content": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The inline `.proto` file source used when `service_definition = \"PROTO_FILE\"` in `BEHAVIOR` mode.",
						},
						"method": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The fully-qualified gRPC method to invoke in `BEHAVIOR` mode (e.g. `package.Service/Method`). Required in `BEHAVIOR` mode; forbidden in `HEALTH` mode.",
						},
						"message": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The JSON request message sent as the gRPC call payload in `BEHAVIOR` mode.",
						},
						"metadata": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"key": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "The gRPC metadata (header) key.",
									},
									"value": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "The gRPC metadata (header) value.",
									},
								},
							},
							Description: "gRPC metadata (request headers) sent with the call.",
						},
						"assertion": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"source": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "The source of the asserted value. Possible values are `RESPONSE_TIME`, `GRPC_RESPONSE`, `GRPC_METADATA`, `GRPC_HEALTHCHECK_STATUS`, and `GRPC_STATUS_CODE`.",
									},
									"property": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"comparison": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "The type of comparison to be executed between expected and actual value of the assertion. Possible values are `EQUALS`, `NOT_EQUALS`, `HAS_KEY`, `NOT_HAS_KEY`, `HAS_VALUE`, `NOT_HAS_VALUE`, `IS_EMPTY`, `NOT_EMPTY`, `GREATER_THAN`, `LESS_THAN`, `CONTAINS`, `NOT_CONTAINS`, `IS_NULL`, and `NOT_NULL`.",
									},
									"target": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
							Description: "A request can have multiple assertions.",
						},
					},
				},
				Description: "The parameters for the gRPC connection.",
			},
			"group_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The id of the check group this monitor is part of.",
			},
			"group_order": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The position of this monitor in a check group. It determines in what order checks and monitors are run when a group is triggered from the API or from CI/CD.",
			},
			retryStrategyAttributeName: makeRetryStrategyAttributeSchema(RetryStrategyAttributeSchemaOptions{
				Required: false,
				Computed: true,
			}),
			"trigger_incident": triggerIncidentAttributeSchema,
		},
		CustomizeDiff: customdiff.Sequence(
			RetryStrategyCustomizeDiff,
			FrequencyOffsetCustomizeDiff,
		),
	}
}

func resourceGRPCMonitorCreate(d *schema.ResourceData, client interface{}) error {
	monitor, err := grpcCheckFromResourceData(d)
	if err != nil {
		return fmt.Errorf("translation error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	newMonitor, err := client.(checkly.Client).CreateGRPCMonitor(ctx, monitor)

	if err != nil {
		checkJSON, _ := json.Marshal(monitor)
		return fmt.Errorf("API error 1: %w, Check: %s", err, string(checkJSON))
	}
	d.SetId(newMonitor.ID)
	return resourceGRPCMonitorRead(d, client)
}

func resourceGRPCMonitorRead(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	monitor, err := client.(checkly.Client).GetGRPCMonitor(ctx, d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			//if resource is deleted remotely, then mark it as
			//successfully gone by unsetting it's ID
			d.SetId("")
			return nil
		}
		return fmt.Errorf("API error 2: %w", err)
	}
	return resourceDataFromGRPCMonitor(monitor, d)
}

func resourceGRPCMonitorUpdate(d *schema.ResourceData, client interface{}) error {
	monitor, err := grpcCheckFromResourceData(d)

	if err != nil {
		return fmt.Errorf("translation error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	_, err = client.(checkly.Client).UpdateGRPCMonitor(ctx, monitor.ID, monitor)
	if err != nil {
		checkJSON, _ := json.Marshal(monitor)
		return fmt.Errorf("API error 3: Couldn't update check, Error: %w, \nCheck: %s", err, checkJSON)
	}
	d.SetId(monitor.ID)
	return resourceGRPCMonitorRead(d, client)
}

func resourceGRPCMonitorDelete(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	if err := client.(checkly.Client).DeleteGRPCMonitor(ctx, d.Id()); err != nil {
		return fmt.Errorf("API error 4: Couldn't delete Check %s, Error: %w", d.Id(), err)
	}
	return nil
}

func resourceDataFromGRPCMonitor(c *checkly.GRPCMonitor, d *schema.ResourceData) error {
	d.Set("name", c.Name)
	d.Set("description", c.Description)
	d.Set("activated", c.Activated)
	d.Set("muted", c.Muted)
	d.Set("should_fail", c.ShouldFail)
	d.Set("run_parallel", c.RunParallel)
	d.Set("locations", c.Locations)
	d.Set("degraded_response_time", c.DegradedResponseTime)
	d.Set("max_response_time", c.MaxResponseTime)

	sort.Strings(c.Tags)
	d.Set("tags", c.Tags)

	d.Set(frequencyAttributeName, c.Frequency)
	if c.Frequency == 0 {
		d.Set(frequencyOffsetAttributeName, c.FrequencyOffset)
	} else {
		d.Set(frequencyOffsetAttributeName, nil)
	}

	if c.RuntimeID != nil {
		d.Set("runtime_id", *c.RuntimeID)
	}

	if err := d.Set("alert_settings", setFromAlertSettings(*c.AlertSettings)); err != nil {
		return fmt.Errorf("error setting alert settings for resource %s: %w", d.Id(), err)
	}
	d.Set("use_global_alert_settings", c.UseGlobalAlertSettings)

	err := d.Set("request", setFromGRPCRequest(c.Request))
	if err != nil {
		return fmt.Errorf("error setting request for resource %s: %w", d.Id(), err)
	}
	d.Set("group_id", c.GroupID)
	d.Set("group_order", c.GroupOrder)
	d.Set("private_locations", c.PrivateLocations)
	d.Set(alertChannelSubscriptionAttributeName, setFromAlertChannelSubscriptions(c.AlertChannelSubscriptions))
	d.Set(retryStrategyAttributeName, listFromRetryStrategy(c.RetryStrategy))
	d.Set("trigger_incident", setFromTriggerIncident(c.TriggerIncident))
	d.SetId(d.Id())
	return nil
}

func setFromGRPCRequest(r checkly.GRPCRequest) []tfMap {
	s := tfMap{}
	s["host"] = r.URL
	s["port"] = grpcPortToInt(r.Port)
	s["ip_family"] = r.IPFamily
	s["skip_ssl"] = r.SkipSSL
	s["timeout"] = r.Timeout
	s["assertion"] = setFromAssertions(r.Assertions)
	s["grpc_mode"] = r.GRPCConfig.Mode
	s["tls"] = r.GRPCConfig.TLS
	s["store_response_body"] = r.GRPCConfig.StoreResponseBody
	s["service"] = r.GRPCConfig.Service
	s["service_definition"] = r.GRPCConfig.ServiceDefinition
	s["proto_content"] = r.GRPCConfig.ProtoContent
	s["method"] = r.GRPCConfig.Method
	s["message"] = r.GRPCConfig.Message
	s["metadata"] = setFromGRPCMetadata(r.GRPCConfig.Metadata)
	return []tfMap{s}
}

func grpcCheckFromResourceData(d *schema.ResourceData) (checkly.GRPCMonitor, error) {
	monitor := checkly.GRPCMonitor{
		ID:                        d.Id(),
		Name:                      d.Get("name").(string),
		Description:               optionalStringPointerFromResourceData(d, "description"),
		Frequency:                 d.Get(frequencyAttributeName).(int),
		Activated:                 d.Get("activated").(bool),
		Muted:                     d.Get("muted").(bool),
		ShouldFail:                d.Get("should_fail").(bool),
		RunParallel:               d.Get("run_parallel").(bool),
		Locations:                 stringsFromSet(d.Get("locations").(*schema.Set)),
		DegradedResponseTime:      d.Get("degraded_response_time").(int),
		MaxResponseTime:           d.Get("max_response_time").(int),
		Tags:                      stringsFromSet(d.Get("tags").(*schema.Set)),
		UseGlobalAlertSettings:    d.Get("use_global_alert_settings").(bool),
		GroupID:                   int64(d.Get("group_id").(int)),
		GroupOrder:                d.Get("group_order").(int),
		AlertChannelSubscriptions: alertChannelSubscriptionsFromSet(d.Get(alertChannelSubscriptionAttributeName).(*schema.Set)),
		RetryStrategy:             retryStrategyFromList(d.Get(retryStrategyAttributeName).([]any)),
		TriggerIncident:           triggerIncidentFromSet(d.Get("trigger_incident").(*schema.Set)),
	}

	alertSettings := alertSettingsFromSet(d.Get("alert_settings").([]interface{}))
	monitor.AlertSettings = &alertSettings

	runtimeId := d.Get("runtime_id").(string)
	if runtimeId == "" {
		monitor.RuntimeID = nil
	} else {
		monitor.RuntimeID = &runtimeId
	}

	privateLocations := stringsFromSet(d.Get("private_locations").(*schema.Set))
	monitor.PrivateLocations = &privateLocations

	monitor.Request = grpcRequestFromList(d.Get("request").([]any))

	monitor.FrequencyOffset = d.Get(frequencyOffsetAttributeName).(int)

	return monitor, nil
}

func grpcRequestFromList(s []any) checkly.GRPCRequest {
	if len(s) == 0 || s[0] == nil {
		return checkly.GRPCRequest{}
	}
	res := s[0].(tfMap)
	return checkly.GRPCRequest{
		URL:        res["host"].(string),
		Port:       res["port"].(int),
		IPFamily:   res["ip_family"].(string),
		SkipSSL:    res["skip_ssl"].(bool),
		Timeout:    res["timeout"].(int),
		Assertions: assertionsFromSet(res["assertion"].(*schema.Set)),
		GRPCConfig: checkly.GRPCConfig{
			Mode:              res["grpc_mode"].(string),
			TLS:               res["tls"].(bool),
			StoreResponseBody: res["store_response_body"].(bool),
			Service:           res["service"].(string),
			ServiceDefinition: res["service_definition"].(string),
			ProtoContent:      res["proto_content"].(string),
			Method:            res["method"].(string),
			Message:           res["message"].(string),
			Metadata:          grpcMetadataFromSet(res["metadata"].(*schema.Set)),
		},
	}
}

func grpcMetadataFromSet(s *schema.Set) []checkly.GRPCMetadata {
	res := make([]checkly.GRPCMetadata, s.Len())
	for i, item := range s.List() {
		m := item.(tfMap)
		res[i] = checkly.GRPCMetadata{
			Key:   m["key"].(string),
			Value: m["value"].(string),
		}
	}
	return res
}

func setFromGRPCMetadata(metadata []checkly.GRPCMetadata) []tfMap {
	s := make([]tfMap, len(metadata))
	for i, m := range metadata {
		s[i] = tfMap{
			"key":   m.Key,
			"value": m.Value,
		}
	}
	return s
}

// grpcPortToInt coerces the SDK's open `interface{}` port (a JSON number decodes
// to float64, a templated port stays a string) back into the schema's TypeInt.
// A non-numeric template value yields 0, which the user's config overrides.
func grpcPortToInt(p interface{}) int {
	switch v := p.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}
