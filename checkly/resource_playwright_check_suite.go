package checkly

import (
	"context"
	"encoding/base64"
	"fmt"
	"slices"
	"sort"
	"strings"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var defaultPlaywrightBrowsers = []string{"chromium", "firefox", "webkit"}

var defaultTestCommand = map[string]string{
	"npm":  "npx playwright test",
	"pnpm": "pnpm playwright test",
	"yarn": "yarn playwright test",
	"bun":  "bunx playwright test",
}

func resourcePlaywrightCheckSuite() *schema.Resource {
	return &schema.Resource{
		Create: resourcePlaywrightCheckSuiteCreate,
		Read:   resourcePlaywrightCheckSuiteRead,
		Update: resourcePlaywrightCheckSuiteUpdate,
		Delete: resourcePlaywrightCheckSuiteDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "Creates a Playwright check from a code bundle.",
		Schema: map[string]*schema.Schema{
			"name": {
				Description: "The name of the check.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"description": {
				Description: "A description of the check.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			frequencyAttributeName: makeFrequencyAttributeSchema(FrequencyAttributeSchemaOptions{
				Monitor:            false,
				AllowHighFrequency: false,
			}),
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
			environmentVariableAttributeName: makeEnvironmentVariableAttributeSchema(EnvironmentVariableAttributeSchemaOptions{
				Description: "Insert environment variables into the execution environment.",
			}),
			"tags": {
				Description: "A list of tags for organizing and filtering checks and monitors.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			alertChannelSubscriptionAttributeName: makeAlertChannelSubscriptionAttributeSchema(AlertChannelSubscriptionAttributeSchemaOptions{}),
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
						metadataAttributeName: {
							Description: "The generated metadata of the code bundle.",
							Type:        schema.TypeString,
							Required:    true,
						},
					},
				},
			},
			"runtime": {
				Description: "Configure the runtime environment of the Playwright check.",
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"auto_detect": {
							Description: "Whether to automatically detect appropriate runtime environment " +
								"configuration from the bundle. " +
								"(Default `true`).",
							Type:     schema.TypeBool,
							Default:  true,
							Optional: true,
						},
						"working_dir": {
							Description: "The working directory in which runtime commands are executed. " +
								"This is useful for monorepos or workspaces where the Playwright " +
								"project is in a subdirectory. Use \".\" to explicitly specify the root.",
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
							ValidateFunc: func(val any, key string) (warns []string, errs []error) {
								if val.(string) == "" {
									errs = append(errs, fmt.Errorf("%q must not be empty; use \".\" for the root directory", key))
								}
								return
							},
						},
						"steps": {
							Description: "Customize the actions taken during test execution.",
							Type:        schema.TypeList,
							Optional:    true,
							Computed:    true,
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
										Computed:    true,
										MaxItems:    1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"command": {
													Description: "The command used to run Playwright. The default " +
														"value is the appropriate exec command for your package " +
														"manager (e.g. `npx playwright test` for `npm`).",
													Type:     schema.TypeString,
													Optional: true,
													Computed: true,
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
							Computed: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"version": {
										Description: "The Playwright version to use. Defaults to the " +
											"version detected from the code bundle's lockfile.",
										Type:     schema.TypeString,
										Optional: true,
										Computed: true,
									},
									"device": {
										Description: "The list of devices that should be made available for Playwright. " +
											"Defaults to chromium, firefox, and webkit.",
										Type:     schema.TypeSet,
										Optional: true,
										Computed: true,
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
						"engine": {
							Description: "The JavaScript engine used to run the Playwright tests.",
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Description:  `The engine name. Valid values are "node" or "bun".`,
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validateOneOf([]string{"node", "bun"}),
									},
									"version": {
										Description:  `The engine version. Valid values: "22", "24" for node; "1.3" for bun.`,
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validateOneOf([]string{"22", "24", "1.3"}),
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
		CustomizeDiff: customdiff.Sequence(
			func(ctx context.Context, diff *schema.ResourceDiff, meta any) error {
				runtimeListAttr := diff.GetRawConfig().GetAttr("runtime")

				var isRuntimeBlockPresent bool
				var isRuntimeWorkingDirPresent bool
				var isRuntimePlaywrightBlockPresent bool
				var isRuntimePlaywrightVersionPresent bool
				var isRuntimePlaywrightDeviceBlockPresent bool
				var isRuntimeStepsBlockPresent bool
				var isRuntimeStepsInstallBlockPresent bool
				var isRuntimeStepsTestBlockPresent bool
				var isRuntimeStepsTestCommandPresent bool

				runtimeIt := runtimeListAttr.ElementIterator()
				if runtimeIt.Next() {
					isRuntimeBlockPresent = true

					_, configRuntimeAttr := runtimeIt.Element()

					workingDirAttr := configRuntimeAttr.GetAttr("working_dir")
					isRuntimeWorkingDirPresent = !workingDirAttr.IsNull()

					playwrightListAttr := configRuntimeAttr.GetAttr("playwright")

					playwrightIt := playwrightListAttr.ElementIterator()
					if playwrightIt.Next() {
						isRuntimePlaywrightBlockPresent = true

						_, playwrightAttr := playwrightIt.Element()

						versionAttr := playwrightAttr.GetAttr("version")

						isRuntimePlaywrightVersionPresent = !versionAttr.IsNull()

						deviceListAttr := playwrightAttr.GetAttr("device")

						deviceIt := deviceListAttr.ElementIterator()
						if deviceIt.Next() {
							isRuntimePlaywrightDeviceBlockPresent = true
						}
					}

					stepsListAttr := configRuntimeAttr.GetAttr("steps")

					stepsIt := stepsListAttr.ElementIterator()
					if stepsIt.Next() {
						isRuntimeStepsBlockPresent = true

						_, stepsAttr := stepsIt.Element()

						installListAttr := stepsAttr.GetAttr("install")

						installIt := installListAttr.ElementIterator()
						if installIt.Next() {
							isRuntimeStepsInstallBlockPresent = true
						}

						testListAttr := stepsAttr.GetAttr("test")

						testIt := testListAttr.ElementIterator()
						if testIt.Next() {
							isRuntimeStepsTestBlockPresent = true

							_, testAttr := testIt.Element()

							commandAttr := testAttr.GetAttr("command")

							isRuntimeStepsTestCommandPresent = !commandAttr.IsNull()
						}
					}
				}

				bundleAttr, err := PlaywrightCheckSuiteBundleAttributeFromList(diff.Get("bundle").([]any))
				if err != nil {
					return err
				}

				runtimeAttr, err := PlaywrightCheckSuiteRuntimeAttributeFromList(diff.Get("runtime").([]any))
				if err != nil {
					return fmt.Errorf("failed to convert state runtime attribute to struct: %w\n", err)
				}

				if runtimeAttr == nil {
					runtimeAttr = &PlaywrightCheckSuiteRuntimeAttribute{}
				}

				var overrideRuntime bool

				if !isRuntimeBlockPresent {
					runtimeAttr.AutoDetect = true
					overrideRuntime = true
				}

				if !isRuntimeWorkingDirPresent {
					if runtimeAttr.WorkingDir != "" {
						runtimeAttr.WorkingDir = ""
						overrideRuntime = true
					}
				}

				if !isRuntimePlaywrightBlockPresent {
					if runtimeAttr.Playwright != nil {
						runtimeAttr.Playwright = nil
						overrideRuntime = true
					}
				}

				if !isRuntimePlaywrightVersionPresent {
					if runtimeAttr.Playwright != nil && runtimeAttr.Playwright.Version != "" {
						runtimeAttr.Playwright.Version = ""
						overrideRuntime = true
					}
				}

				if !isRuntimePlaywrightDeviceBlockPresent {
					if runtimeAttr.Playwright != nil && runtimeAttr.Playwright.Devices != nil {
						runtimeAttr.Playwright.Devices = nil
						overrideRuntime = true
					}
				}

				if !isRuntimeStepsBlockPresent {
					if runtimeAttr.Steps != nil {
						runtimeAttr.Steps = nil
						overrideRuntime = true
					}
				}

				if !isRuntimeStepsInstallBlockPresent {
					if runtimeAttr.Steps != nil && runtimeAttr.Steps.Install != nil {
						runtimeAttr.Steps.Install = nil
						overrideRuntime = true
					}
				}

				if !isRuntimeStepsTestBlockPresent {
					if runtimeAttr.Steps != nil && runtimeAttr.Steps.Test != nil {
						runtimeAttr.Steps.Test = nil
						overrideRuntime = true
					}
				}

				if !isRuntimeStepsTestCommandPresent {
					if runtimeAttr.Steps != nil && runtimeAttr.Steps.Test != nil && runtimeAttr.Steps.Test.Command != "" {
						runtimeAttr.Steps.Test.Command = ""
						overrideRuntime = true
					}
				}

				if runtimeAttr.AutoDetect && bundleAttr != nil {
					if !isRuntimeWorkingDirPresent {
						runtimeAttr.WorkingDir = bundleAttr.Metadata.WorkingDir
						overrideRuntime = true
					}

					playwrightAttr := runtimeAttr.Playwright
					if runtimeAttr.Playwright == nil {
						playwrightAttr = &PlaywrightCheckSuiteRuntimePlaywrightAttribute{}
						runtimeAttr.Playwright = playwrightAttr
					}

					if playwrightAttr.Version == "" {
						playwrightAttr.Version = bundleAttr.Metadata.PlaywrightVersion
						overrideRuntime = true
					}

					if playwrightAttr.Devices == nil || len(*playwrightAttr.Devices) == 0 {
						var devices PlaywrightCheckSuiteRuntimePlaywrightDeviceAttributes
						for _, device := range defaultPlaywrightBrowsers {
							devices = append(devices, PlaywrightCheckSuiteRuntimePlaywrightDeviceAttribute{
								Type: device,
							})
						}

						playwrightAttr.Devices = &devices
						overrideRuntime = true
					}

					stepsAttr := runtimeAttr.Steps
					if stepsAttr == nil {
						stepsAttr = &PlaywrightCheckSuiteRuntimeStepsAttribute{}
						runtimeAttr.Steps = stepsAttr
					}

					testAttr := stepsAttr.Test
					if testAttr == nil {
						testAttr = &PlaywrightCheckSuiteRuntimeStepsTestAttribute{}
						stepsAttr.Test = testAttr
					}

					if testAttr.Command == "" {
						if defaultTestCommand, ok := defaultTestCommand[bundleAttr.Metadata.PackageManager]; ok {
							testAttr.Command = defaultTestCommand
							overrideRuntime = true
						}
					}

					// Just assume NPM if we couldn't detect a package manager.
					if testAttr.Command == "" {
						if defaultTestCommand, ok := defaultTestCommand["npm"]; ok {
							testAttr.Command = defaultTestCommand
							overrideRuntime = true
						}
					}
				}

				if runtimeAttr.Playwright == nil || runtimeAttr.Playwright.Version == "" {
					if runtimeAttr.AutoDetect {
						return fmt.Errorf(
							"unable to detect Playwright version from the code bundle's lockfile; " +
								"set \"runtime.playwright.version\" explicitly or ensure the archive " +
								"contains a package-lock.json, pnpm-lock.yaml, yarn.lock, or bun.lock with @playwright/test",
						)
					} else {
						return fmt.Errorf(`"runtime.playwright.version" is required when "runtime.auto_detect" is false`)
					}
				}

				if runtimeAttr.Playwright == nil || runtimeAttr.Playwright.Devices == nil || len(*runtimeAttr.Playwright.Devices) == 0 {
					if runtimeAttr.AutoDetect {
						return fmt.Errorf(
							"unable to detect default devices from the code bundle; " +
								"set \"runtime.playwright.device\" blocks explicitly",
						)
					} else {
						return fmt.Errorf(`at least one "runtime.playwright.device" block is required when "runtime.auto_detect" is false`)
					}
				}

				if runtimeAttr.Steps == nil || runtimeAttr.Steps.Test == nil || runtimeAttr.Steps.Test.Command == "" {
					if runtimeAttr.AutoDetect {
						return fmt.Errorf(
							"unable to detect an appropriate test command for the code bundle; " +
								"set \"runtime.steps.test.command\" explicitly or ensure the archive " +
								"contains a supported lockfile",
						)
					} else {
						return fmt.Errorf(`"runtime.steps.test.command" is required when "runtime.auto_detect" is false`)
					}
				}

				if overrideRuntime {
					diff.SetNew("runtime", runtimeAttr.ToList())
				}

				return nil
			},
		),
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
		Description:               optionalStringPointerFromResourceData(d, "description"),
		Frequency:                 d.Get(frequencyAttributeName).(int),
		Activated:                 d.Get("activated").(bool),
		Muted:                     d.Get("muted").(bool),
		RunParallel:               d.Get("run_parallel").(bool),
		Locations:                 stringsFromSet(d.Get("locations").(*schema.Set)),
		Tags:                      stringsFromSet(d.Get("tags").(*schema.Set)),
		UseGlobalAlertSettings:    d.Get("use_global_alert_settings").(bool),
		GroupID:                   int64(d.Get("group_id").(int)),
		GroupOrder:                d.Get("group_order").(int),
		AlertChannelSubscriptions: alertChannelSubscriptionsFromSet(d.Get(alertChannelSubscriptionAttributeName).(*schema.Set)),
		TriggerIncident:           triggerIncidentFromSet(d.Get("trigger_incident").(*schema.Set)),
	}

	envVars, err := environmentVariablesFromResourceData(d)
	if err != nil {
		return PlaywrightCheckSuiteResource{}, err
	}

	check.EnvironmentVariables = envVars

	alertSettings := alertSettingsFromSet(d.Get("alert_settings").([]any))
	check.AlertSettings = &alertSettings

	privateLocations := stringsFromSet(d.Get("private_locations").(*schema.Set))
	check.PrivateLocations = &privateLocations

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

		check.CacheHash = bundleAttr.Metadata.CacheHash
	}

	runtimeAttr, err := PlaywrightCheckSuiteRuntimeAttributeFromList(d.Get("runtime").([]any))
	if err != nil {
		return PlaywrightCheckSuiteResource{}, err
	}

	if runtimeAttr != nil {
		if runtimeAttr.WorkingDir != "" {
			check.WorkingDir = &runtimeAttr.WorkingDir
		}

		if runtimeAttr.Steps != nil {
			if runtimeAttr.Steps.Test != nil && runtimeAttr.Steps.Test.Command != "" {
				check.TestCommand = &runtimeAttr.Steps.Test.Command
			}

			if runtimeAttr.Steps.Install != nil && runtimeAttr.Steps.Install.Command != "" {
				check.InstallCommand = &runtimeAttr.Steps.Install.Command
			}
		}

		if runtimeAttr.Playwright != nil {
			check.PlaywrightVersion = &runtimeAttr.Playwright.Version

			if runtimeAttr.Playwright.Devices != nil {
				var browsers []string
				for _, device := range *runtimeAttr.Playwright.Devices {
					browsers = append(browsers, device.Type)
				}

				check.Browsers = browsers
			}
		}

		if runtimeAttr.Engine != nil {
			check.Engine = &runtimeAttr.Engine.Name
			check.EngineVersion = &runtimeAttr.Engine.Version
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

	existingRuntimeAttr, err := PlaywrightCheckSuiteRuntimeAttributeFromList(d.Get("runtime").([]any))
	if err != nil {
		return PlaywrightCheckSuiteResource{}, err
	}

	runtimeAttr := PlaywrightCheckSuiteRuntimeAttribute{
		AutoDetect: existingRuntimeAttr.AutoDetect,
	}

	if check.WorkingDir != nil {
		runtimeAttr.WorkingDir = *check.WorkingDir
	}

	if check.TestCommand != nil || check.InstallCommand != nil {
		runtimeAttr.Steps = new(PlaywrightCheckSuiteRuntimeStepsAttribute)

		if check.InstallCommand != nil && *check.InstallCommand != "" {
			runtimeAttr.Steps.Install = &PlaywrightCheckSuiteRuntimeStepsInstallAttribute{
				Command: *check.InstallCommand,
			}
		}

		if check.TestCommand != nil && *check.TestCommand != "" {
			runtimeAttr.Steps.Test = &PlaywrightCheckSuiteRuntimeStepsTestAttribute{
				Command: *check.TestCommand,
			}
		}
	}

	if len(check.Browsers) != 0 || check.PlaywrightVersion != nil {
		runtimeAttr.Playwright = new(PlaywrightCheckSuiteRuntimePlaywrightAttribute)

		if check.PlaywrightVersion != nil {
			runtimeAttr.Playwright.Version = *check.PlaywrightVersion
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

	if check.Engine != nil && *check.Engine != "" {
		runtimeAttr.Engine = &PlaywrightCheckSuiteRuntimeEngineAttribute{
			Name: *check.Engine,
		}
		if check.EngineVersion != nil {
			runtimeAttr.Engine.Version = *check.EngineVersion
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
	d.Set("description", r.Description)
	d.Set("activated", r.Activated)
	d.Set("muted", r.Muted)
	d.Set("run_parallel", r.RunParallel)
	d.Set("locations", r.Locations)
	d.Set("private_locations", r.PrivateLocations)

	d.Set(environmentVariableAttributeName, listFromEnvironmentVariables(r.EnvironmentVariables))

	sort.Strings(r.Tags)
	d.Set("tags", r.Tags)

	d.Set(frequencyAttributeName, r.Frequency)

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
	d.Set(alertChannelSubscriptionAttributeName, setFromAlertChannelSubscriptions(r.AlertChannelSubscriptions))
	d.Set("trigger_incident", setFromTriggerIncident(r.TriggerIncident))
	d.SetId(d.Id())
	return nil
}

type PlaywrightCheckSuiteBundleAttribute struct {
	ID       string
	Metadata *PlaywrightCodeBundleMetadata
}

func PlaywrightCheckSuiteBundleAttributeFromList(
	list []any,
) (*PlaywrightCheckSuiteBundleAttribute, error) {
	if len(list) == 0 {
		return nil, nil
	}

	m := list[0].(tfMap)

	data, err := PlaywrightCodeBundleMetadataFromString(m[metadataAttributeName].(string))
	if err != nil {
		return nil, err
	}

	a := PlaywrightCheckSuiteBundleAttribute{
		ID:       m["id"].(string),
		Metadata: data,
	}

	return &a, nil
}

func (a *PlaywrightCheckSuiteBundleAttribute) ToList() []tfMap {
	if a == nil {
		return []tfMap{}
	}

	return []tfMap{
		{
			"id":                  a.ID,
			metadataAttributeName: a.Metadata.EncodeToString(),
		},
	}
}

type PlaywrightCheckSuiteRuntimeAttribute struct {
	AutoDetect bool
	WorkingDir string
	Steps      *PlaywrightCheckSuiteRuntimeStepsAttribute
	Playwright *PlaywrightCheckSuiteRuntimePlaywrightAttribute
	Engine     *PlaywrightCheckSuiteRuntimeEngineAttribute
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

	engineAttr, err := PlaywrightCheckSuiteRuntimeEngineAttributeFromList(m["engine"].([]any))
	if err != nil {
		return nil, err
	}

	a := PlaywrightCheckSuiteRuntimeAttribute{
		AutoDetect: m["auto_detect"].(bool),
		WorkingDir: m["working_dir"].(string),
		Steps:      stepsAttr,
		Playwright: playwrightAttr,
		Engine:     engineAttr,
	}

	return &a, nil
}

func (a *PlaywrightCheckSuiteRuntimeAttribute) ToList() []tfMap {
	if a == nil {
		return []tfMap{}
	}

	return []tfMap{
		{
			"auto_detect": a.AutoDetect,
			"working_dir": a.WorkingDir,
			"steps":       a.Steps.ToList(),
			"playwright":  a.Playwright.ToList(),
			"engine":      a.Engine.ToList(),
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

type PlaywrightCheckSuiteRuntimeEngineAttribute struct {
	Name    string
	Version string
}

func PlaywrightCheckSuiteRuntimeEngineAttributeFromList(list []any) (*PlaywrightCheckSuiteRuntimeEngineAttribute, error) {
	if len(list) == 0 {
		return nil, nil
	}
	m := list[0].(tfMap)
	return &PlaywrightCheckSuiteRuntimeEngineAttribute{
		Name:    m["name"].(string),
		Version: m["version"].(string),
	}, nil
}

func (a *PlaywrightCheckSuiteRuntimeEngineAttribute) ToList() []tfMap {
	if a == nil {
		return []tfMap{}
	}
	return []tfMap{{
		"name":    a.Name,
		"version": a.Version,
	}}
}
