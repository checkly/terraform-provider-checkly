package checkly

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	checkly "github.com/checkly/checkly-go-sdk"
)

func resourceCheckGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceCheckGroupCreate,
		Read:   resourceCheckGroupRead,
		Update: resourceCheckGroupUpdate,
		Delete: resourceCheckGroupDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "Check groups allow  you to group together a set of related checks, which can also share default settings for various attributes.",
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the check group.",
			},
			"concurrency": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Determines how many checks are run concurrently when triggering a check group from CI/CD or through the API.",
			},
			"activated": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Determines if the checks in the group are running or not.",
			},
			"muted": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Determines if any notifications will be sent out when a check in this group fails and/or recovers.",
			},
			"run_parallel": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Determines if the checks in the group should run in all selected locations in parallel or round-robin.",
			},
			"locations": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "An array of one or more data center locations where to run the checks.",
			},
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
			deprecatedEnvironmentVariablesAttributeName: makeDeprecatedEnvironmentVariablesAttributeSchema(DeprecatedEnvironmentVariablesAttributeSchemaOptions{}),
			environmentVariableAttributeName: makeEnvironmentVariableAttributeSchema(EnvironmentVariableAttributeSchemaOptions{
				Description: "Insert environment variables into the runtime environment. Only relevant for browser checks. Use global environment variables whenever possible.",
			}),
			doubleCheckAttributeName: doubleCheckAttributeSchema,
			"tags": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "Tags for organizing and filtering checks.",
			},
			"setup_snippet_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "An ID reference to a snippet to use in the setup phase of an API check.",
			},
			"teardown_snippet_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "An ID reference to a snippet to use in the teardown phase of an API check.",
			},
			"local_setup_script": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A valid piece of Node.js code to run in the setup phase of an API check in this group.",
			},
			"local_teardown_script": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A valid piece of Node.js code to run in the teardown phase of an API check in this group.",
			},
			"runtime_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     nil,
				Description: "The id of the runtime to use for this group.",
			},
			"alert_channel_subscription": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"channel_id": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"activated": {
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
			},
			alertSettingsAttributeName: makeAlertSettingsAttributeSchema(AlertSettingsAttributeSchemaOptions{
				EnableSSLCertificates: true,
			}),
			"use_global_alert_settings": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "When true, the account level alert settings will be used, not the alert setting defined on this check group.",
			},
			apiCheckDefaultsAttributeName: makeAPICheckDefaultsAttributeSchema(),
			retryStrategyAttributeName: makeRetryStrategyAttributeSchema(RetryStrategyAttributeSchemaOptions{
				SupportsOnlyOnNetworkError: true,
			}),
		},
		CustomizeDiff: customdiff.Sequence(
			RetryStrategyCustomizeDiff,
		),
	}
}

func resourceCheckGroupCreate(d *schema.ResourceData, client interface{}) error {
	group, err := checkGroupFromResourceData(d)
	if err != nil {
		return fmt.Errorf("translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	gotGroup, err := client.(checkly.Client).CreateGroup(ctx, group)
	if err != nil {
		return fmt.Errorf("API error11: %w", err)
	}
	d.SetId(fmt.Sprintf("%d", gotGroup.ID))
	return resourceCheckGroupRead(d, client)
}

func resourceCheckGroupRead(d *schema.ResourceData, client interface{}) error {
	ID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return fmt.Errorf("ID %s is not numeric: %w", d.Id(), err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	group, err := client.(checkly.Client).GetGroup(ctx, ID)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			//if resource is deleted remotely, then mark it as
			//successfully gone by unsetting it's ID
			d.SetId("")
			return nil
		}
		return fmt.Errorf("API error12: %w", err)
	}
	return resourceDataFromCheckGroup(group, d)
}

func resourceCheckGroupUpdate(d *schema.ResourceData, client interface{}) error {
	group, err := checkGroupFromResourceData(d)
	if err != nil {
		return fmt.Errorf("translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	_, err = client.(checkly.Client).UpdateGroup(ctx, group.ID, group)
	if err != nil {
		return fmt.Errorf("API error13: %w", err)
	}
	d.SetId(fmt.Sprintf("%d", group.ID))
	return resourceCheckGroupRead(d, client)
}

func resourceCheckGroupDelete(d *schema.ResourceData, client interface{}) error {
	ID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return fmt.Errorf("ID %s is not numeric: %w", d.Id(), err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	if err := client.(checkly.Client).DeleteGroup(ctx, ID); err != nil {
		return fmt.Errorf("API error14: %w", err)
	}
	return nil
}

func resourceDataFromCheckGroup(g *checkly.Group, d *schema.ResourceData) error {
	d.Set("name", g.Name)
	d.Set("concurrency", g.Concurrency)
	d.Set("activated", g.Activated)
	d.Set("muted", g.Muted)
	d.Set("run_parallel", g.RunParallel)
	d.Set("locations", g.Locations)
	d.Set(doubleCheckAttributeName, g.DoubleCheck)
	d.Set("setup_snippet_id", g.SetupSnippetID)
	d.Set("teardown_snippet_id", g.TearDownSnippetID)
	d.Set("local_setup_script", g.LocalSetupScript)
	d.Set("local_teardown_script", g.LocalTearDownScript)
	d.Set("alert_channel_subscription", g.AlertChannelSubscriptions)
	d.Set("private_locations", g.PrivateLocations)

	sort.Strings(g.Tags)
	d.Set("tags", g.Tags)

	if err := updateCompatEnvironmentVariablesResourceData(d, g.EnvironmentVariables); err != nil {
		return fmt.Errorf("error setting environment variables for resource %s: %s", d.Id(), err)
	}

	if g.RuntimeID != nil {
		d.Set("runtime_id", *g.RuntimeID)
	}

	if err := d.Set("alert_settings", setFromAlertSettings(g.AlertSettings)); err != nil {
		return fmt.Errorf("error setting alert settings for resource %s: %s", d.Id(), err)
	}
	d.Set("use_global_alert_settings", g.UseGlobalAlertSettings)

	if err := d.Set(apiCheckDefaultsAttributeName, setFromAPICheckDefaults(g.APICheckDefaults)); err != nil {
		return fmt.Errorf("error setting request for resource %s: %s", d.Id(), err)
	}

	d.Set(retryStrategyAttributeName, listFromRetryStrategy(g.RetryStrategy))

	d.SetId(d.Id())
	return nil
}

func checkGroupFromResourceData(d *schema.ResourceData) (checkly.Group, error) {
	ID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		if d.Id() != "" {
			return checkly.Group{}, err
		}
		ID = 0
	}

	group := checkly.Group{
		ID:                        ID,
		Name:                      d.Get("name").(string),
		Concurrency:               d.Get("concurrency").(int),
		Activated:                 d.Get("activated").(bool),
		Muted:                     d.Get("muted").(bool),
		RunParallel:               d.Get("run_parallel").(bool),
		Locations:                 stringsFromSet(d.Get("locations").(*schema.Set)),
		DoubleCheck:               d.Get(doubleCheckAttributeName).(bool),
		Tags:                      stringsFromSet(d.Get("tags").(*schema.Set)),
		SetupSnippetID:            int64(d.Get("setup_snippet_id").(int)),
		TearDownSnippetID:         int64(d.Get("teardown_snippet_id").(int)),
		LocalSetupScript:          d.Get("local_setup_script").(string),
		LocalTearDownScript:       d.Get("local_teardown_script").(string),
		AlertSettings:             alertSettingsFromSet(d.Get("alert_settings").([]interface{})),
		UseGlobalAlertSettings:    d.Get("use_global_alert_settings").(bool),
		APICheckDefaults:          apiCheckDefaultsFromSet(d.Get(apiCheckDefaultsAttributeName).(*schema.Set)),
		AlertChannelSubscriptions: alertChannelSubscriptionsFromSet(d.Get("alert_channel_subscription").([]interface{})),
		RetryStrategy:             retryStrategyFromList(d.Get(retryStrategyAttributeName).([]any)),
	}

	runtimeId := d.Get("runtime_id").(string)
	if runtimeId == "" {
		group.RuntimeID = nil
	} else {
		group.RuntimeID = &runtimeId
	}

	environmentVariables, err := compatEnvironmentVariablesFromResourceData(d)
	if err != nil {
		return checkly.Group{}, err
	}
	group.EnvironmentVariables = environmentVariables

	privateLocations := stringsFromSet(d.Get("private_locations").(*schema.Set))
	group.PrivateLocations = &privateLocations

	return group, nil
}

