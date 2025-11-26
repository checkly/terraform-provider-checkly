package checkly

import (
	"context"
	"encoding/base64"
	"fmt"
	"slices"
	"sort"
	"strings"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourcePlaywrightCheckSuite() *schema.Resource {
	return &schema.Resource{
		Create: resourcePlaywrightCheckSuiteCreate,
		Read:   resourcePlaywrightCheckSuiteRead,
		Update: resourcePlaywrightCheckSuiteUpdate,
		Delete: resourcePlaywrightCheckSuiteDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "Creates a DNS Monitor to check DNS record availability and response times.",
		Schema: map[string]*schema.Schema{
			"name": {
				Description: "The name of the check.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"frequency": {
				Description:  "How often the check should run in minutes. Possible values are `1`, `2`, `5`, `10`, `15`, `30`, `60`, `120`, `180`, `360`, `720`, and `1440`.",
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validateOneOf([]int{1, 2, 5, 10, 15, 30, 60, 120, 180, 360, 720, 1440}),
			},
			"activated": {
				Description: "Determines whether the check will run periodically or not after being deployed.",
				Type:        schema.TypeBool,
				Required:    true,
			},
			"muted": {
				Description: "Determines if any notifications will be sent out when the check fails and/or recovers. (Default `false`).",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"run_parallel": {
				Description: "Determines whether the check should run on all selected locations in parallel or round-robin. (Default `false`).",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"locations": {
				Description: "An array of one or more data center locations where to run the this check.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"tags": {
				Description: "A list of tags for organizing and filtering checks and monitors.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"alert_channel_subscription": {
				Description: "An array of channel IDs and whether they're activated or not. If you don't set at least one alert subscription for your check, we won't be able to alert you.",
				Type:        schema.TypeList,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"channel_id": {
							Description: "The ID of the alert channel.",
							Type:        schema.TypeInt,
							Required:    true,
						},
						"activated": {
							Description: "Whether an alert should be sent to this channel.",
							Type:        schema.TypeBool,
							Required:    true,
						},
					},
				},
			},
			"private_locations": {
				Description: "An array of one or more private locations slugs.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			alertSettingsAttributeName: makeAlertSettingsAttributeSchema(AlertSettingsAttributeSchemaOptions{
				Monitor: false,
			}),
			"use_global_alert_settings": {
				Description: "When true, the account level alert settings will be used, not the alert setting defined on this check. (Default `true`).",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
			},
			"bundle": {
				Description: "Attaches a code bundle to the check.",
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Description: "The ID of the code bundle.",
							Type:        schema.TypeString,
							Required:    true,
						},
						"data": {
							Description: "The auxiliary data of the code bundle.",
							Type:        schema.TypeString,
							Required:    true,
						},
					},
				},
			},
			"runtime": {
				Description: "Configure the runtime environment of the Playwright check.",
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"steps": {
							Description: "Customize the actions taken during test execution.",
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"install": {
										Description: "Customize the install step, which is used to initialize the " +
											"environment prior to starting the test run.",
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"command": {
													Description: "The command used to install dependencies prior to " +
														"running Playwright. The default value is the appropriate " +
														"install command for your package manager (e.g. " +
														"`npm install` for `npm`).",
													Type:     schema.TypeString,
													Optional: true,
												},
											},
										},
									},
									"test": {
										Description: "Customize the test step.",
										Type:        schema.TypeList,
										Optional:    true,
										MaxItems:    1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"command": {
													Description: "The command used to run Playwright. The default " +
														"value is the appropriate exec command for your package " +
														"manager (e.g. `npx playwright test` for `npm`).",
													Type:     schema.TypeString,
													Optional: true,
												},
											},
										},
									},
								},
							},
						},
						"playwright": {
							Description: "Configure the Playwright capabilities that should be made available " +
								"to the runtime environment.",
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"version": {
										Description: "The Playwright version to use.",
										Type:        schema.TypeString,
										Optional:    true,
									},
									"device": {
										Description: "The list of devices that should be made available for Playwright.",
										Type:        schema.TypeSet,
										Optional:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"type": {
													Description: "The type of the device.",
													Type:        schema.TypeString,
													Required:    true,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"group_id": {
				Description: "The ID of the check group that this check is part of.",
				Type:        schema.TypeInt,
				Optional:    true,
			},
			"group_order": {
				Description: "The position of the check in the check group. It determines in what order checks and monitors are run when a group is triggered from the API or from CI/CD.",
				Type:        schema.TypeInt,
				Optional:    true,
			},
			"trigger_incident": triggerIncidentAttributeSchema,
		},
	}
}

func resourcePlaywrightCheckSuiteCreate(d *schema.ResourceData, client any) error {
	r, err := PlaywrightCheckSuiteResourceFromResourceData(d)
	if err != nil {
		return fmt.Errorf("failed to load Playwright check suite from resource data: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()

	newCheck, err := client.(checkly.Client).CreatePlaywrightCheck(ctx, *r.PlaywrightCheck)
	if err != nil {
		return fmt.Errorf("failed to create Playwright check suite: %w", err)
	}

	d.SetId(newCheck.ID)

	return resourcePlaywrightCheckSuiteRead(d, client)
}

func resourcePlaywrightCheckSuiteRead(d *schema.ResourceData, client any) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()

	check, err := client.(checkly.Client).GetPlaywrightCheck(ctx, d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("failed to retrieve Playwright check suite %q: %w", d.Id(), err)
	}

	resource, err := PlaywrightCheckSuiteResourceFromAPIModel(check, d)
	if err != nil {
		return fmt.Errorf("failed to convert API response to Playwright check suite resource: %w", err)
	}

	err = resource.StoreResourceData(d)
	if err != nil {
		return fmt.Errorf("failed to store Playwright check suite %q state: %w", d.Id(), err)
	}

	return nil
}

func resourcePlaywrightCheckSuiteUpdate(d *schema.ResourceData, client any) error {
	r, err := PlaywrightCheckSuiteResourceFromResourceData(d)
	if err != nil {
		return fmt.Errorf("failed to load Playwright check suite from resource data: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()

	_, err = client.(checkly.Client).UpdatePlaywrightCheck(ctx, r.ID, *r.PlaywrightCheck)
	if err != nil {
		return fmt.Errorf("failed to update Playwright check suite %q: %w", d.Id(), err)
	}

	d.SetId(r.ID)

	return resourcePlaywrightCheckSuiteRead(d, client)
}

func resourcePlaywrightCheckSuiteDelete(d *schema.ResourceData, client any) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()

	if err := client.(checkly.Client).DeletePlaywrightCheck(ctx, d.Id()); err != nil {
		return fmt.Errorf("failed to delete Playwright check suite %q: %w", d.Id(), err)
	}

	return nil
}

type PlaywrightCheckSuiteResource struct {
	*checkly.PlaywrightCheck
	Bundle  *PlaywrightCheckSuiteBundleAttribute
	Runtime *PlaywrightCheckSuiteRuntimeAttribute
}

func PlaywrightCheckSuiteResourceFromResourceData(
	d *schema.ResourceData,
) (PlaywrightCheckSuiteResource, error) {
	check := checkly.PlaywrightCheck{
		ID:                        d.Id(),
		Name:                      d.Get("name").(string),
		Frequency:                 d.Get("frequency").(int),
		Activated:                 d.Get("activated").(bool),
		Muted:                     d.Get("muted").(bool),
		RunParallel:               d.Get("run_parallel").(bool),
		Locations:                 stringsFromSet(d.Get("locations").(*schema.Set)),
		Tags:                      stringsFromSet(d.Get("tags").(*schema.Set)),
		UseGlobalAlertSettings:    d.Get("use_global_alert_settings").(bool),
		GroupID:                   int64(d.Get("group_id").(int)),
		GroupOrder:                d.Get("group_order").(int),
		AlertChannelSubscriptions: alertChannelSubscriptionsFromSet(d.Get("alert_channel_subscription").([]any)),
		TriggerIncident:           triggerIncidentFromSet(d.Get("trigger_incident").(*schema.Set)),
	}

	alertSettings := alertSettingsFromSet(d.Get("alert_settings").([]any))
	check.AlertSettings = &alertSettings

	bundleAttr, err := PlaywrightCheckSuiteBundleAttributeFromList(d.Get("bundle").([]any))
	if err != nil {
		return PlaywrightCheckSuiteResource{}, err
	}

	if bundleAttr != nil {
		bundlePath, err := base64.StdEncoding.DecodeString(bundleAttr.ID)
		if err != nil {
			return PlaywrightCheckSuiteResource{}, fmt.Errorf("invalid code bundle identifier %q: %w", bundleAttr.ID, err)
		}

		check.CodeBundlePath = string(bundlePath)

		// We may want to make this configurable in the future, but for now
		// this will do.
		check.CacheHash = checksumSha256(strings.NewReader(bundleAttr.Data.ChecksumSha256))
	}

	runtimeAttr, err := PlaywrightCheckSuiteRuntimeAttributeFromList(d.Get("runtime").([]any))
	if err != nil {
		return PlaywrightCheckSuiteResource{}, err
	}

	if runtimeAttr != nil {
		if runtimeAttr.Steps != nil {
			if runtimeAttr.Steps.Test != nil {
				check.TestCommand = runtimeAttr.Steps.Test.Command
			}

			if runtimeAttr.Steps.Install != nil {
				check.InstallCommand = &runtimeAttr.Steps.Install.Command
			}
		}

		if runtimeAttr.Playwright != nil {
			check.PlaywrightVersion = runtimeAttr.Playwright.Version

			var browsers []string
			for _, device := range *runtimeAttr.Playwright.Devices {
				browsers = append(browsers, device.Type)
			}

			check.Browsers = browsers
		}
	}

	resource := PlaywrightCheckSuiteResource{
		PlaywrightCheck: &check,
		Bundle:          bundleAttr,
		Runtime:         runtimeAttr,
	}

	return resource, nil
}

func PlaywrightCheckSuiteResourceFromAPIModel(
	check *checkly.PlaywrightCheck,
	d *schema.ResourceData,
) (PlaywrightCheckSuiteResource, error) {
	// Use the current state as a template so that we don't have to access
	// the archive file path manually (it needs to be kept because it only
	// exists on the client).
	bundleAttr, err := PlaywrightCheckSuiteBundleAttributeFromList(d.Get("bundle").([]any))
	if err != nil {
		return PlaywrightCheckSuiteResource{}, err
	}

	var runtimeAttr PlaywrightCheckSuiteRuntimeAttribute

	if check.TestCommand != "" || check.InstallCommand != nil {
		runtimeAttr.Steps = new(PlaywrightCheckSuiteRuntimeStepsAttribute)

		if check.InstallCommand != nil {
			runtimeAttr.Steps.Install = &PlaywrightCheckSuiteRuntimeStepsInstallAttribute{
				Command: *check.InstallCommand,
			}
		}

		if check.TestCommand != "" {
			runtimeAttr.Steps.Test = &PlaywrightCheckSuiteRuntimeStepsTestAttribute{
				Command: check.TestCommand,
			}
		}
	}

	if len(check.Browsers) != 0 || check.PlaywrightVersion != "" {
		runtimeAttr.Playwright = new(PlaywrightCheckSuiteRuntimePlaywrightAttribute)

		if check.PlaywrightVersion != "" {
			runtimeAttr.Playwright.Version = check.PlaywrightVersion
		}

		if len(check.Browsers) != 0 {
			slices.Sort(check.Browsers)

			devices := make(PlaywrightCheckSuiteRuntimePlaywrightDeviceAttributes, 0, len(check.Browsers))

			for _, deviceType := range check.Browsers {
				devices = append(devices, PlaywrightCheckSuiteRuntimePlaywrightDeviceAttribute{
					Type: deviceType,
				})
			}

			runtimeAttr.Playwright.Devices = &devices
		}
	}

	resource := PlaywrightCheckSuiteResource{
		PlaywrightCheck: check,
		Bundle:          bundleAttr,
		Runtime:         &runtimeAttr,
	}

	return resource, nil
}

func (r *PlaywrightCheckSuiteResource) StoreResourceData(
	d *schema.ResourceData,
) error {
	d.Set("name", r.Name)
	d.Set("activated", r.Activated)
	d.Set("muted", r.Muted)
	d.Set("run_parallel", r.RunParallel)
	d.Set("locations", r.Locations)
	d.Set("private_locations", r.PrivateLocations)

	sort.Strings(r.Tags)
	d.Set("tags", r.Tags)

	d.Set("frequency", r.Frequency)

	if err := d.Set("alert_settings", setFromAlertSettings(*r.AlertSettings)); err != nil {
		return fmt.Errorf("error setting alert settings for resource %s: %w", d.Id(), err)
	}
	d.Set("use_global_alert_settings", r.UseGlobalAlertSettings)

	err := d.Set("bundle", r.Bundle.ToList())
	if err != nil {
		return err
	}

	err = d.Set("runtime", r.Runtime.ToList())
	if err != nil {
		return fmt.Errorf("error setting runtime for resource %s: %w", d.Id(), err)
	}

	d.Set("group_id", r.GroupID)
	d.Set("group_order", r.GroupOrder)
	d.Set("alert_channel_subscription", r.AlertChannelSubscriptions)
	d.Set("trigger_incident", setFromTriggerIncident(r.TriggerIncident))
	d.SetId(d.Id())
	return nil
}

type PlaywrightCheckSuiteBundleAttribute struct {
	ID   string
	Data *PlaywrightCodeBundleData
}

func PlaywrightCheckSuiteBundleAttributeFromList(
	list []any,
) (*PlaywrightCheckSuiteBundleAttribute, error) {
	if len(list) == 0 {
		return nil, nil
	}

	m := list[0].(tfMap)

	data, err := PlaywrightCodeBundleDataFromString(m["data"].(string))
	if err != nil {
		return nil, err
	}

	a := PlaywrightCheckSuiteBundleAttribute{
		ID:   m["id"].(string),
		Data: data,
	}

	return &a, nil
}

func (a *PlaywrightCheckSuiteBundleAttribute) ToList() []tfMap {
	if a == nil {
		return []tfMap{}
	}

	return []tfMap{
		{
			"id":   a.ID,
			"data": a.Data.EncodeToString(),
		},
	}
}

type PlaywrightCheckSuiteRuntimeAttribute struct {
	Steps      *PlaywrightCheckSuiteRuntimeStepsAttribute
	Playwright *PlaywrightCheckSuiteRuntimePlaywrightAttribute
}

func PlaywrightCheckSuiteRuntimeAttributeFromList(
	list []any,
) (*PlaywrightCheckSuiteRuntimeAttribute, error) {
	if len(list) == 0 {
		return nil, nil
	}

	m := list[0].(tfMap)

	stepsAttr, err := PlaywrightCheckSuiteRuntimeStepsAttributeFromList(m["steps"].([]any))
	if err != nil {
		return nil, err
	}

	playwrightAttr, err := PlaywrightCheckSuiteRuntimePlaywrightAttributeFromList(m["playwright"].([]any))
	if err != nil {
		return nil, err
	}

	a := PlaywrightCheckSuiteRuntimeAttribute{
		Steps:      stepsAttr,
		Playwright: playwrightAttr,
	}

	return &a, nil
}

func (a *PlaywrightCheckSuiteRuntimeAttribute) ToList() []tfMap {
	if a == nil {
		return []tfMap{}
	}

	return []tfMap{
		{
			"steps":      a.Steps.ToList(),
			"playwright": a.Playwright.ToList(),
		},
	}
}

type PlaywrightCheckSuiteRuntimeStepsAttribute struct {
	Install *PlaywrightCheckSuiteRuntimeStepsInstallAttribute
	Test    *PlaywrightCheckSuiteRuntimeStepsTestAttribute
}

func PlaywrightCheckSuiteRuntimeStepsAttributeFromList(
	list []any,
) (*PlaywrightCheckSuiteRuntimeStepsAttribute, error) {
	if len(list) == 0 {
		return nil, nil
	}

	m := list[0].(tfMap)

	installAttr, err := PlaywrightCheckSuiteRuntimeStepsInstallAttributeFromList(m["install"].([]any))
	if err != nil {
		return nil, err
	}

	testAttr, err := PlaywrightCheckSuiteRuntimeStepsTestAttributeFromList(m["test"].([]any))
	if err != nil {
		return nil, err
	}

	a := PlaywrightCheckSuiteRuntimeStepsAttribute{
		Install: installAttr,
		Test:    testAttr,
	}

	return &a, nil
}

func (a *PlaywrightCheckSuiteRuntimeStepsAttribute) ToList() []tfMap {
	if a == nil {
		return []tfMap{}
	}

	return []tfMap{
		{
			"install": a.Install.ToList(),
			"test":    a.Test.ToList(),
		},
	}
}

type PlaywrightCheckSuiteRuntimeStepsInstallAttribute struct {
	Command string
}

func PlaywrightCheckSuiteRuntimeStepsInstallAttributeFromList(
	list []any,
) (*PlaywrightCheckSuiteRuntimeStepsInstallAttribute, error) {
	if len(list) == 0 {
		return nil, nil
	}

	m := list[0].(tfMap)

	a := PlaywrightCheckSuiteRuntimeStepsInstallAttribute{
		Command: m["command"].(string),
	}

	return &a, nil
}

func (a *PlaywrightCheckSuiteRuntimeStepsInstallAttribute) ToList() []tfMap {
	if a == nil {
		return []tfMap{}
	}

	return []tfMap{
		{
			"command": a.Command,
		},
	}
}

type PlaywrightCheckSuiteRuntimeStepsTestAttribute struct {
	Command string
}

func PlaywrightCheckSuiteRuntimeStepsTestAttributeFromList(
	list []any,
) (*PlaywrightCheckSuiteRuntimeStepsTestAttribute, error) {
	if len(list) == 0 {
		return nil, nil
	}

	m := list[0].(tfMap)

	a := PlaywrightCheckSuiteRuntimeStepsTestAttribute{
		Command: m["command"].(string),
	}

	return &a, nil
}

func (a *PlaywrightCheckSuiteRuntimeStepsTestAttribute) ToList() []tfMap {
	if a == nil {
		return []tfMap{}
	}

	return []tfMap{
		{
			"command": a.Command,
		},
	}
}

func PlaywrightCheckSuiteRuntimePlaywrightAttributeFromList(
	list []any,
) (*PlaywrightCheckSuiteRuntimePlaywrightAttribute, error) {
	if len(list) == 0 {
		return nil, nil
	}

	m := list[0].(tfMap)

	devices, err := PlaywrightCheckSuiteRuntimePlaywrightDeviceAttributesFromList(m["device"].(*schema.Set).List())
	if err != nil {
		return nil, err
	}

	a := PlaywrightCheckSuiteRuntimePlaywrightAttribute{
		Version: m["version"].(string),
		Devices: devices,
	}

	return &a, nil
}

func (a *PlaywrightCheckSuiteRuntimePlaywrightAttribute) ToList() []tfMap {
	if a == nil {
		return []tfMap{}
	}

	return []tfMap{
		{
			"version": a.Version,
			"device":  a.Devices.ToList(),
		},
	}
}

type PlaywrightCheckSuiteRuntimePlaywrightAttribute struct {
	Version string
	Devices *PlaywrightCheckSuiteRuntimePlaywrightDeviceAttributes
}

type PlaywrightCheckSuiteRuntimePlaywrightDeviceAttribute struct {
	Type string
}

type PlaywrightCheckSuiteRuntimePlaywrightDeviceAttributes []PlaywrightCheckSuiteRuntimePlaywrightDeviceAttribute

func PlaywrightCheckSuiteRuntimePlaywrightDeviceAttributesFromList(
	list []any,
) (*PlaywrightCheckSuiteRuntimePlaywrightDeviceAttributes, error) {
	if len(list) == 0 {
		return nil, nil
	}

	a := make(PlaywrightCheckSuiteRuntimePlaywrightDeviceAttributes, 0, len(list))

	for _, device := range list {
		a = append(a, PlaywrightCheckSuiteRuntimePlaywrightDeviceAttribute{
			Type: device.(tfMap)["type"].(string),
		})
	}

	return &a, nil
}

func (a *PlaywrightCheckSuiteRuntimePlaywrightDeviceAttributes) ToList() []tfMap {
	if a == nil {
		return []tfMap{}
	}

	m := []tfMap{}

	for _, device := range *a {
		m = append(m, tfMap{
			"type": device.Type,
		})
	}

	return m
}
