package checkly

import (
	"context"
	"fmt"
	"sort"
	"strings"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	defaultRuntimeAttributeName            = "default_runtime"
	enforceAlertSettingsAttributeName      = "enforce_alert_settings"
	enforceLocationsAttributeName          = "enforce_locations"
	enforceRetryStrategyAttributeName      = "enforce_retry_strategy"
	enforceSchedulingStrategyAttributeName = "enforce_scheduling_strategy"
	setupScriptAttributeName               = "setup_script"
	teardownScriptAttributeName            = "teardown_script"
)

func resourceCheckGroupV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceCheckGroupV2Create,
		Read:   resourceCheckGroupV2Read,
		Update: resourceCheckGroupV2Update,
		Delete: resourceCheckGroupV2Delete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "Check groups organize related checks together and allow optional shared configuration. " +
			"\n\n" +
			"Unlike `checkly_check_group` (v1), which always imposed implicit defaults for various settings, " +
			"`checkly_check_group_v2` does not override any check settings by default. " +
			"A group can be as minimal as just a name. " +
			"Use the optional `enforce_*` blocks only when you explicitly want the group to override individual " +
			"check settings." +
			"\n\n" +
			"The new `checkly_check_group_v2` resource should always be preferred over the old " +
			"`checkly_check_group` resource.",
		Schema: map[string]*schema.Schema{
			"name": {
				Description: "The name of the check group.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"concurrency": {
				Description: "Determines the number of checks to run concurrently when triggering the check group " +
					"via CI/CD or the API. " +
					"(Default `1`).",
				Type:     schema.TypeInt,
				Optional: true,
				Default:  1,
			},
			"activated": {
				Description: "Determines whether activated checks in the group should run or not. Deactivating the " +
					"group will prevent all checks in the group from running, regardless of their activated state. " +
					"(Default `true`).",
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"muted": {
				Description: "Determines if any notifications will be sent out when a check in this group fails " +
					"and/or recovers. Muting the group will deactivate all notifications for all checks in the " +
					"group, regardless of their muted state. " +
					"(Default `false`).",
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			environmentVariableAttributeName: makeEnvironmentVariableAttributeSchema(EnvironmentVariableAttributeSchemaOptions{
				Description: "Insert environment variables into the runtime environment of checks in the group. " +
					"Only relevant for check types that support environment variables. " +
					"Use global environment variables whenever possible.",
			}),
			"tags": {
				Description: "Additional tags to append to all checks in the group.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			setupScriptAttributeName: {
				Description: "A script to run in the setup phase of an API check. " +
					"Runs in addition to the check's own setup script.",
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"snippet_id": {
							Description: "The ID of a code snippet. Code snippets are not available for new plans.",
							Type:        schema.TypeInt,
							Optional:    true,
							ExactlyOneOf: []string{
								fmt.Sprintf("%s.0.snippet_id", setupScriptAttributeName),
								fmt.Sprintf("%s.0.inline_script", setupScriptAttributeName),
							},
						},
						"inline_script": {
							Description: "A valid piece of Node.js code.",
							Type:        schema.TypeString,
							Optional:    true,
							ExactlyOneOf: []string{
								fmt.Sprintf("%s.0.snippet_id", setupScriptAttributeName),
								fmt.Sprintf("%s.0.inline_script", setupScriptAttributeName),
							},
						},
					},
				},
			},
			teardownScriptAttributeName: {
				Description: "A script to run in the teardown phase of an API check. " +
					"Runs in addition to the check's own teardown script.",
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"snippet_id": {
							Description: "The ID of a code snippet. Code snippets are not available for new plans.",
							Type:        schema.TypeInt,
							Optional:    true,
							ExactlyOneOf: []string{
								fmt.Sprintf("%s.0.snippet_id", teardownScriptAttributeName),
								fmt.Sprintf("%s.0.inline_script", teardownScriptAttributeName),
							},
						},
						"inline_script": {
							Description: "A valid piece of Node.js code.",
							Type:        schema.TypeString,
							Optional:    true,
							ExactlyOneOf: []string{
								fmt.Sprintf("%s.0.snippet_id", teardownScriptAttributeName),
								fmt.Sprintf("%s.0.inline_script", teardownScriptAttributeName),
							},
						},
					},
				},
			},
			defaultRuntimeAttributeName: {
				Description: "Sets a default runtime for the group. " +
					"Used as a fallback when a check belonging to the group has no runtime set. " +
					"Takes precedence over the account default runtime.",
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"runtime_id": {
							Description: "The runtime ID.",
							Type:        schema.TypeString,
							Required:    true,
						},
					},
				},
			},
			enforceAlertSettingsAttributeName: {
				Description: "Enforces alert settings for the whole group. " +
					"Overrides check configuration.",
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Description: "Determines whether the enforced alert settings should be active.",
							Type:        schema.TypeBool,
							Required:    true,
						},
						alertSettingsAttributeName: makeAlertSettingsAttributeSchema(AlertSettingsAttributeSchemaOptions{
							EnableSSLCertificates: true,
						}),
						"use_global_alert_settings": {
							Description: "Whether to use account level alert settings instead of the group's alert " +
								"settings." +
								"Default (`false`).",
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
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
					},
				},
			},
			enforceLocationsAttributeName: {
				Description: "Enforces public and private locations for the whole group. " +
					"Overrides check configuration.",
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Description: "Determines whether the enforced locations should be active.",
							Type:        schema.TypeBool,
							Required:    true,
						},
						"locations": {
							Description: "An array of one or more data center locations where to run the checks.",
							Type:        schema.TypeSet,
							Optional:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							AtLeastOneOf: []string{
								fmt.Sprintf("%s.0.locations", enforceLocationsAttributeName),
								fmt.Sprintf("%s.0.private_locations", enforceLocationsAttributeName),
							},
						},
						"private_locations": {
							Description: "An array of one or more private locations slugs.",
							Type:        schema.TypeSet,
							Optional:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							AtLeastOneOf: []string{
								fmt.Sprintf("%s.0.locations", enforceLocationsAttributeName),
								fmt.Sprintf("%s.0.private_locations", enforceLocationsAttributeName),
							},
						},
					},
				},
			},
			enforceRetryStrategyAttributeName: {
				Description: "Enforces a retry strategy for the whole group. " +
					"Overrides check configuration.",
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Description: "Determines whether the enforced retry strategy should be active.",
							Type:        schema.TypeBool,
							Required:    true,
						},
						retryStrategyAttributeName: makeRetryStrategyAttributeSchema(RetryStrategyAttributeSchemaOptions{
							SupportsOnlyOnNetworkError: true,
							Required:                   true,
							Computed:                   false,
						}),
					},
				},
			},
			enforceSchedulingStrategyAttributeName: {
				Description: "Enforces a scheduling strategy for the whole group. " +
					"Overrides check configuration.",
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Description: "Determines whether the enforced scheduling strategy should be active.",
							Type:        schema.TypeBool,
							Required:    true,
						},
						"run_parallel": {
							Description: "Determines if the checks in the group should run in all selected locations in parallel or round-robin.",
							Type:        schema.TypeBool,
							Required:    true,
						},
					},
				},
			},
			apiCheckDefaultsAttributeName: makeAPICheckDefaultsAttributeSchema(),
		},
		CustomizeDiff: customdiff.Sequence(
			makeEnabledCustomizeDiffFunc(enforceAlertSettingsAttributeName, func(old, new []any) ([]tfMap, bool) {
				return nil, false
			}),
			makeEnabledCustomizeDiffFunc(enforceLocationsAttributeName, func(old, new []any) ([]tfMap, bool) {
				return nil, false
			}),
			makeEnabledCustomizeDiffFunc(enforceRetryStrategyAttributeName, func(old, new []any) ([]tfMap, bool) {
				retryStrategy := new[0].(tfMap)[retryStrategyAttributeName].([]any)

				return []tfMap{
					{
						"enabled":                  true,
						retryStrategyAttributeName: listFromRetryStrategy(retryStrategyFromList(retryStrategy)),
					},
				}, true
			}),
			makeEnabledCustomizeDiffFunc(enforceSchedulingStrategyAttributeName, func(old, new []any) ([]tfMap, bool) {
				return nil, false
			}),
		),
	}
}

func makeEnabledCustomizeDiffFunc(attrName string, f func(old, new []any) ([]tfMap, bool)) schema.CustomizeDiffFunc {
	return func(_ context.Context, diff *schema.ResourceDiff, meta any) error {
		rawOldBlock, rawNewBlock := diff.GetChange(attrName)

		oldBlock := rawOldBlock.([]any)
		newBlock := rawNewBlock.([]any)

		// If both the old value and the new value are empty, then don't show
		// the diff. This prevents the attribute from being shown in the diff
		// when the resource is being created.
		//
		// Note that when the block is set, but then removed by the user, TF
		// somehow still returns a non-empty list for the new block. That case
		// is handled later via GetRawPlan().
		if len(oldBlock) == 0 && len(newBlock) == 0 {
			err := diff.Clear(attrName)
			if err != nil {
				return err
			}

			return nil
		}

		planBlock := diff.GetRawPlan().GetAttr(attrName)

		it := planBlock.ElementIterator()

		// If the attribute is not in the plan, it has been removed.
		if !it.Next() {
			err := diff.SetNew(attrName, []tfMap{
				{
					"enabled": false,
				},
			})
			if err != nil {
				return err
			}

			return nil
		}

		_, first := it.Element()

		enabled := first.GetAttr("enabled").True()

		// If the block is present, but has not been enabled, then simplify
		// the diff.
		if !enabled {
			err := diff.SetNew(attrName, []tfMap{
				{
					"enabled": false,
				},
			})
			if err != nil {
				return err
			}

			return nil
		}

		value, ok := f(oldBlock, newBlock)
		if ok {
			err := diff.SetNew(attrName, value)
			if err != nil {
				return err
			}
		}

		return nil
	}
}

func resourceCheckGroupV2Create(d *schema.ResourceData, client interface{}) error {
	r, err := CheckGroupV2ResourceFromResourceData(d)
	if err != nil {
		return fmt.Errorf("failed to load check group v2 from resource data: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()

	newGroup, err := client.(checkly.Client).CreateGroupV2(ctx, *r.GroupV2)
	if err != nil {
		return fmt.Errorf("failed to create check group (v2): %w", err)
	}

	d.SetId(encodeNumericID(newGroup.ID))

	return resourceCheckGroupV2Read(d, client)
}

func resourceCheckGroupV2Read(d *schema.ResourceData, client interface{}) error {
	id, err := decodeNumericID(d.Id())
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()

	group, err := client.(checkly.Client).GetGroupV2(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("failed to retrieve check group (v2) %q: %w", d.Id(), err)
	}

	resource, err := CheckGroupV2ResourceFromAPIModel(group, d)
	if err != nil {
		return fmt.Errorf("failed to convert API response to check group (v2) resource: %w", err)
	}

	err = resource.StoreResourceData(d)
	if err != nil {
		return fmt.Errorf("failed to store check group (v2) %q state: %w", d.Id(), err)
	}

	return nil
}

func resourceCheckGroupV2Update(d *schema.ResourceData, client interface{}) error {
	r, err := CheckGroupV2ResourceFromResourceData(d)
	if err != nil {
		return fmt.Errorf("failed to load check group (v2) %q from resource data: %w", d.Id(), err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()

	_, err = client.(checkly.Client).UpdateGroupV2(ctx, r.ID, *r.GroupV2)
	if err != nil {
		return fmt.Errorf("failed to update check group (v2) %q: %w", d.Id(), err)
	}

	d.SetId(encodeNumericID(r.ID))

	return resourceCheckGroupV2Read(d, client)
}

func resourceCheckGroupV2Delete(d *schema.ResourceData, client interface{}) error {
	id, err := decodeNumericID(d.Id())
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()

	if err := client.(checkly.Client).DeleteGroupV2(ctx, id); err != nil {
		return fmt.Errorf("failed to delete check group (v2) %q: %w", id, err)
	}

	return nil
}

type CheckGroupV2ScriptAttribute struct {
	SnippetID    *int64
	InlineScript *string
}

func CheckGroupV2ScriptAttributeFromList(
	list []any,
) (*CheckGroupV2ScriptAttribute, error) {
	if len(list) == 0 {
		return nil, nil
	}

	m := list[0].(tfMap)

	a := CheckGroupV2ScriptAttribute{}

	if raw, ok := m["snippet_id"]; ok {
		if raw != nil {
			id := int64(raw.(int))
			if id != 0 {
				a.SnippetID = &id
			}
		}
	}

	if raw, ok := m["inline_script"]; ok {
		if raw != nil {
			val := raw.(string)
			a.InlineScript = &val
		}
	}

	return &a, nil
}

func (a *CheckGroupV2ScriptAttribute) ToList() []tfMap {
	if a == nil {
		return []tfMap{}
	}

	return []tfMap{
		{
			"snippet_id":    a.SnippetID,
			"inline_script": a.InlineScript,
		},
	}
}

type CheckGroupV2DefaultRuntimeAttribute struct {
	RuntimeID string
}

func CheckGroupV2DefaultRuntimeAttributeFromList(
	list []any,
) (*CheckGroupV2DefaultRuntimeAttribute, error) {
	if len(list) == 0 {
		return nil, nil
	}

	m := list[0].(tfMap)

	a := CheckGroupV2DefaultRuntimeAttribute{
		RuntimeID: m["runtime_id"].(string),
	}

	return &a, nil
}

func (a *CheckGroupV2DefaultRuntimeAttribute) ToList() []tfMap {
	if a == nil {
		return []tfMap{}
	}

	return []tfMap{
		{
			"runtime_id": a.RuntimeID,
		},
	}
}

type CheckGroupV2EnforceAlertSettingsAttribute struct {
	Enabled                   bool
	AlertSettings             *checkly.AlertSettings
	UseGlobalAlertSettings    *bool
	AlertChannelSubscriptions []checkly.AlertChannelSubscription
}

func CheckGroupV2EnforceAlertSettingsAttributeFromList(
	list []any,
) (*CheckGroupV2EnforceAlertSettingsAttribute, error) {
	if len(list) == 0 {
		return nil, nil
	}

	m := list[0].(tfMap)

	a := CheckGroupV2EnforceAlertSettingsAttribute{
		Enabled: m["enabled"].(bool),
	}

	if raw, ok := m[alertSettingsAttributeName]; ok {
		if raw != nil {
			val := alertSettingsFromSet(raw.([]any))
			a.AlertSettings = &val
		}
	}

	if raw, ok := m["use_global_alert_settings"]; ok {
		if raw != nil {
			val := raw.(bool)
			a.UseGlobalAlertSettings = &val
		}
	}

	if raw, ok := m["alert_channel_subscription"]; ok {
		if raw != nil {
			a.AlertChannelSubscriptions = alertChannelSubscriptionsFromSet(raw.([]any))
		}
	}

	return &a, nil
}

func (a *CheckGroupV2EnforceAlertSettingsAttribute) ToList() []tfMap {
	if a == nil {
		return []tfMap{}
	}

	if !a.Enabled {
		return []tfMap{
			{
				"enabled": false,
			},
		}
	}

	var alertSettings []tfMap
	if a.AlertSettings != nil {
		alertSettings = setFromAlertSettings(*a.AlertSettings)
	}

	return []tfMap{
		{
			"enabled":                    true,
			alertSettingsAttributeName:   alertSettings,
			"use_global_alert_settings":  a.UseGlobalAlertSettings,
			"alert_channel_subscription": a.AlertChannelSubscriptions,
		},
	}
}

type CheckGroupV2EnforceLocationsAttribute struct {
	Enabled          bool
	Locations        []string
	PrivateLocations []string
}

func CheckGroupV2EnforceLocationsAttributeFromList(
	list []any,
) (*CheckGroupV2EnforceLocationsAttribute, error) {
	if len(list) == 0 {
		return nil, nil
	}

	m := list[0].(tfMap)

	a := CheckGroupV2EnforceLocationsAttribute{
		Enabled:          m["enabled"].(bool),
		Locations:        stringsFromSet(m["locations"].(*schema.Set)),
		PrivateLocations: stringsFromSet(m["private_locations"].(*schema.Set)),
	}

	return &a, nil
}

func (a *CheckGroupV2EnforceLocationsAttribute) ToList() []tfMap {
	if a == nil {
		return []tfMap{}
	}

	if !a.Enabled {
		return []tfMap{
			{
				"enabled": false,
			},
		}
	}

	return []tfMap{
		{
			"enabled":           true,
			"locations":         a.Locations,
			"private_locations": a.PrivateLocations,
		},
	}
}

type CheckGroupV2EnforceRetryStrategyAttribute struct {
	Enabled       bool
	RetryStrategy *checkly.RetryStrategy
}

func CheckGroupV2EnforceRetryStrategyAttributeFromList(
	list []any,
) (*CheckGroupV2EnforceRetryStrategyAttribute, error) {
	if len(list) == 0 {
		return nil, nil
	}

	m := list[0].(tfMap)

	a := CheckGroupV2EnforceRetryStrategyAttribute{
		Enabled:       m["enabled"].(bool),
		RetryStrategy: retryStrategyFromList(m[retryStrategyAttributeName].([]any)),
	}

	return &a, nil
}

func (a *CheckGroupV2EnforceRetryStrategyAttribute) ToList() []tfMap {
	if a == nil {
		return []tfMap{}
	}

	if !a.Enabled {
		return []tfMap{
			{
				"enabled": false,
			},
		}
	}

	return []tfMap{
		{
			"enabled":                  true,
			retryStrategyAttributeName: listFromRetryStrategy(a.RetryStrategy),
		},
	}
}

type CheckGroupV2EnforceSchedulingStrategyAttribute struct {
	Enabled     bool
	RunParallel bool
}

func CheckGroupV2EnforceSchedulingStrategyAttributeFromList(
	list []any,
) (*CheckGroupV2EnforceSchedulingStrategyAttribute, error) {
	if len(list) == 0 {
		return nil, nil
	}

	m := list[0].(tfMap)

	a := CheckGroupV2EnforceSchedulingStrategyAttribute{
		Enabled:     m["enabled"].(bool),
		RunParallel: m["run_parallel"].(bool),
	}

	return &a, nil
}

func (a *CheckGroupV2EnforceSchedulingStrategyAttribute) ToList() []tfMap {
	if a == nil {
		return []tfMap{}
	}

	if !a.Enabled {
		return []tfMap{
			{
				"enabled": false,
			},
		}
	}

	return []tfMap{
		{
			"enabled":      true,
			"run_parallel": a.RunParallel,
		},
	}
}

type CheckGroupV2Resource struct {
	*checkly.GroupV2
	DefaultRuntime            *CheckGroupV2DefaultRuntimeAttribute
	EnforceAlertSettings      *CheckGroupV2EnforceAlertSettingsAttribute
	EnforceLocations          *CheckGroupV2EnforceLocationsAttribute
	EnforceRetryStrategy      *CheckGroupV2EnforceRetryStrategyAttribute
	EnforceSchedulingStrategy *CheckGroupV2EnforceSchedulingStrategyAttribute
	SetupScript               *CheckGroupV2ScriptAttribute
	TeardownScript            *CheckGroupV2ScriptAttribute
}

func CheckGroupV2ResourceFromResourceData(
	d *schema.ResourceData,
) (*CheckGroupV2Resource, error) {
	// We might not have an ID yet, so be lenient.
	id, _ := decodeNumericID(d.Id())

	group := checkly.GroupV2{
		ID:               id,
		Name:             d.Get("name").(string),
		Concurrency:      d.Get("concurrency").(int),
		Activated:        d.Get("activated").(bool),
		Muted:            d.Get("muted").(bool),
		Tags:             stringsFromSet(d.Get("tags").(*schema.Set)),
		APICheckDefaults: apiCheckDefaultsFromSet(d.Get(apiCheckDefaultsAttributeName).(*schema.Set)),
	}

	defaultRuntimeAttr, err := CheckGroupV2DefaultRuntimeAttributeFromList(d.Get(defaultRuntimeAttributeName).([]any))
	if err != nil {
		return nil, err
	}

	if defaultRuntimeAttr != nil {
		group.RuntimeID = &defaultRuntimeAttr.RuntimeID
	}

	enforceAlertSettingsAttr, err := CheckGroupV2EnforceAlertSettingsAttributeFromList(d.Get(enforceAlertSettingsAttributeName).([]any))
	if err != nil {
		return nil, err
	}

	if enforceAlertSettingsAttr != nil && enforceAlertSettingsAttr.Enabled {
		group.AlertSettings = enforceAlertSettingsAttr.AlertSettings
		group.UseGlobalAlertSettings = enforceAlertSettingsAttr.UseGlobalAlertSettings
		group.AlertChannelSubscriptions = enforceAlertSettingsAttr.AlertChannelSubscriptions
	}

	enforceLocationsAttr, err := CheckGroupV2EnforceLocationsAttributeFromList(d.Get(enforceLocationsAttributeName).([]any))
	if err != nil {
		return nil, err
	}

	if enforceLocationsAttr != nil && enforceLocationsAttr.Enabled {
		group.Locations = enforceLocationsAttr.Locations
		group.PrivateLocations = &enforceLocationsAttr.PrivateLocations
	}

	enforceRetryStrategyAttr, err := CheckGroupV2EnforceRetryStrategyAttributeFromList(d.Get(enforceRetryStrategyAttributeName).([]any))
	if err != nil {
		return nil, err
	}

	if enforceRetryStrategyAttr != nil && enforceRetryStrategyAttr.Enabled {
		group.RetryStrategy = enforceRetryStrategyAttr.RetryStrategy
	} else {
		group.RetryStrategy = &checkly.RetryStrategy{
			Type: "FALLBACK",
		}
	}

	enforceSchedulingStrategyAttr, err := CheckGroupV2EnforceSchedulingStrategyAttributeFromList(d.Get(enforceSchedulingStrategyAttributeName).([]any))
	if err != nil {
		return nil, err
	}

	if enforceSchedulingStrategyAttr != nil && enforceSchedulingStrategyAttr.Enabled {
		group.RunParallel = &enforceSchedulingStrategyAttr.RunParallel
	}

	setupScriptAttr, err := CheckGroupV2ScriptAttributeFromList(d.Get(setupScriptAttributeName).([]any))
	if err != nil {
		return nil, err
	}

	if setupScriptAttr != nil {
		group.SetupSnippetID = setupScriptAttr.SnippetID
		group.LocalSetupScript = setupScriptAttr.InlineScript
	}

	teardownScriptAttr, err := CheckGroupV2ScriptAttributeFromList(d.Get(teardownScriptAttributeName).([]any))
	if err != nil {
		return nil, err
	}

	if teardownScriptAttr != nil {
		group.TearDownSnippetID = teardownScriptAttr.SnippetID
		group.LocalTearDownScript = teardownScriptAttr.InlineScript
	}

	envVars, err := environmentVariablesFromResourceData(d)
	if err != nil {
		return nil, err
	}

	group.EnvironmentVariables = envVars

	resource := CheckGroupV2Resource{
		GroupV2:                   &group,
		DefaultRuntime:            defaultRuntimeAttr,
		EnforceAlertSettings:      enforceAlertSettingsAttr,
		EnforceLocations:          enforceLocationsAttr,
		EnforceRetryStrategy:      enforceRetryStrategyAttr,
		EnforceSchedulingStrategy: enforceSchedulingStrategyAttr,
		SetupScript:               setupScriptAttr,
		TeardownScript:            teardownScriptAttr,
	}

	return &resource, nil
}

func CheckGroupV2ResourceFromAPIModel(
	group *checkly.GroupV2,
	d *schema.ResourceData,
) (*CheckGroupV2Resource, error) {
	var defaultRuntimeAttr *CheckGroupV2DefaultRuntimeAttribute
	if group.RuntimeID != nil {
		defaultRuntimeAttr = &CheckGroupV2DefaultRuntimeAttribute{
			RuntimeID: *group.RuntimeID,
		}
	}

	var enforceAlertSettingsAttr *CheckGroupV2EnforceAlertSettingsAttribute
	if group.AlertSettings != nil || group.UseGlobalAlertSettings != nil ||
		len(group.AlertChannelSubscriptions) != 0 {
		enforceAlertSettingsAttr = &CheckGroupV2EnforceAlertSettingsAttribute{
			Enabled:                   true,
			AlertSettings:             group.AlertSettings,
			UseGlobalAlertSettings:    group.UseGlobalAlertSettings,
			AlertChannelSubscriptions: group.AlertChannelSubscriptions,
		}
	} else {
		enforceAlertSettingsAttr = &CheckGroupV2EnforceAlertSettingsAttribute{
			Enabled: false,
		}
	}

	var enforceLocationsAttr *CheckGroupV2EnforceLocationsAttribute
	if len(group.Locations) != 0 || (group.PrivateLocations != nil && len(*group.PrivateLocations) != 0) {
		enforceLocationsAttr = &CheckGroupV2EnforceLocationsAttribute{
			Enabled:   true,
			Locations: group.Locations,
		}

		if group.PrivateLocations != nil {
			enforceLocationsAttr.PrivateLocations = *group.PrivateLocations
		}
	} else {
		enforceLocationsAttr = &CheckGroupV2EnforceLocationsAttribute{
			Enabled: false,
		}
	}

	var enforceRetryStrategyAttr *CheckGroupV2EnforceRetryStrategyAttribute
	if group.RetryStrategy == nil || group.RetryStrategy.Type != "FALLBACK" {
		enforceRetryStrategyAttr = &CheckGroupV2EnforceRetryStrategyAttribute{
			Enabled:       true,
			RetryStrategy: group.RetryStrategy,
		}
	} else {
		enforceRetryStrategyAttr = &CheckGroupV2EnforceRetryStrategyAttribute{
			Enabled: false,
		}
	}

	var enforceSchedulingStrategyAttr *CheckGroupV2EnforceSchedulingStrategyAttribute
	if group.RunParallel != nil {
		enforceSchedulingStrategyAttr = &CheckGroupV2EnforceSchedulingStrategyAttribute{
			Enabled:     true,
			RunParallel: *group.RunParallel,
		}
	} else {
		enforceSchedulingStrategyAttr = &CheckGroupV2EnforceSchedulingStrategyAttribute{
			Enabled: false,
		}
	}

	var setupScriptAttr *CheckGroupV2ScriptAttribute
	if group.SetupSnippetID != nil || group.LocalSetupScript != nil {
		setupScriptAttr = &CheckGroupV2ScriptAttribute{
			SnippetID:    group.SetupSnippetID,
			InlineScript: group.LocalSetupScript,
		}
	}

	var teardownScriptAttr *CheckGroupV2ScriptAttribute
	if group.TearDownSnippetID != nil || group.LocalTearDownScript != nil {
		teardownScriptAttr = &CheckGroupV2ScriptAttribute{
			SnippetID:    group.TearDownSnippetID,
			InlineScript: group.LocalTearDownScript,
		}
	}

	resource := CheckGroupV2Resource{
		GroupV2:                   group,
		DefaultRuntime:            defaultRuntimeAttr,
		EnforceAlertSettings:      enforceAlertSettingsAttr,
		EnforceLocations:          enforceLocationsAttr,
		EnforceRetryStrategy:      enforceRetryStrategyAttr,
		EnforceSchedulingStrategy: enforceSchedulingStrategyAttr,
		SetupScript:               setupScriptAttr,
		TeardownScript:            teardownScriptAttr,
	}

	return &resource, nil
}

func (r *CheckGroupV2Resource) StoreResourceData(
	d *schema.ResourceData,
) error {
	d.Set("name", r.Name)
	d.Set("concurrency", r.Concurrency)
	d.Set("activated", r.Activated)
	d.Set("muted", r.Muted)

	sort.Strings(r.Tags)
	d.Set("tags", r.Tags)

	err := d.Set(defaultRuntimeAttributeName, r.DefaultRuntime.ToList())
	if err != nil {
		return fmt.Errorf("failed to set %q for resource %s: %w", defaultRuntimeAttributeName, d.Id(), err)
	}

	err = d.Set(enforceAlertSettingsAttributeName, r.EnforceAlertSettings.ToList())
	if err != nil {
		return fmt.Errorf("failed to set %q for resource %s: %w", enforceAlertSettingsAttributeName, d.Id(), err)
	}

	err = d.Set(enforceLocationsAttributeName, r.EnforceLocations.ToList())
	if err != nil {
		return fmt.Errorf("failed to set %q for resource %s: %w", enforceLocationsAttributeName, d.Id(), err)
	}

	err = d.Set(enforceRetryStrategyAttributeName, r.EnforceRetryStrategy.ToList())
	if err != nil {
		return fmt.Errorf("failed to set %q for resource %s: %w", enforceRetryStrategyAttributeName, d.Id(), err)
	}

	err = d.Set(enforceSchedulingStrategyAttributeName, r.EnforceSchedulingStrategy.ToList())
	if err != nil {
		return fmt.Errorf("failed to set %q for resource %s: %w", enforceSchedulingStrategyAttributeName, d.Id(), err)
	}

	err = d.Set(setupScriptAttributeName, r.SetupScript.ToList())
	if err != nil {
		return fmt.Errorf("failed to set %q for resource %s: %w", setupScriptAttributeName, d.Id(), err)
	}

	err = d.Set(teardownScriptAttributeName, r.TeardownScript.ToList())
	if err != nil {
		return fmt.Errorf("failed to set %q for resource %s: %w", teardownScriptAttributeName, d.Id(), err)
	}

	err = d.Set(environmentVariableAttributeName, listFromEnvironmentVariables(r.EnvironmentVariables))
	if err != nil {
		return err
	}

	err = d.Set(apiCheckDefaultsAttributeName, setFromAPICheckDefaults(r.APICheckDefaults))
	if err != nil {
		return fmt.Errorf("failed to set %q for resource %s: %w", apiCheckDefaultsAttributeName, d.Id(), err)
	}

	return nil
}
