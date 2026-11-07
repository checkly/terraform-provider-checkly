package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	tfpath "github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/checkly/checkly-go-sdk/v2"
)

var playwrightCheckSuiteDefaultBrowsers = []string{"chromium", "firefox", "webkit"}

var playwrightCheckSuiteDefaultTestCommands = map[string]string{
	"npm":  "npx playwright test",
	"pnpm": "pnpm playwright test",
	"yarn": "yarn playwright test",
	"bun":  "bunx playwright test",
}

var (
	_ resource.Resource                 = (*PlaywrightCheckSuiteResource)(nil)
	_ resource.ResourceWithConfigure    = (*PlaywrightCheckSuiteResource)(nil)
	_ resource.ResourceWithUpgradeState = (*PlaywrightCheckSuiteResource)(nil)
	_ resource.ResourceWithImportState  = (*PlaywrightCheckSuiteResource)(nil)
)

// NewPlaywrightCheckSuiteResource returns a new checkly_playwright_check_suite
// resource.
func NewPlaywrightCheckSuiteResource() resource.Resource {
	return &PlaywrightCheckSuiteResource{}
}

// PlaywrightCheckSuiteResource implements the checkly_playwright_check_suite
// resource.
type PlaywrightCheckSuiteResource struct {
	client checkly.Client
}

// PlaywrightCheckSuiteResourceModel describes the resource data model.
type PlaywrightCheckSuiteResourceModel struct {
	ID                       types.String `tfsdk:"id"`
	Name                     types.String `tfsdk:"name"`
	Description              types.String `tfsdk:"description"`
	Frequency                types.Int64  `tfsdk:"frequency"`
	Activated                types.Bool   `tfsdk:"activated"`
	Muted                    types.Bool   `tfsdk:"muted"`
	RunParallel              types.Bool   `tfsdk:"run_parallel"`
	Locations                types.Set    `tfsdk:"locations"`
	EnvironmentVariable      types.List   `tfsdk:"environment_variable"`
	Tags                     types.Set    `tfsdk:"tags"`
	AlertChannelSubscription types.Set    `tfsdk:"alert_channel_subscription"`
	PrivateLocations         types.Set    `tfsdk:"private_locations"`
	AlertSettings            types.Object `tfsdk:"alert_settings"`
	UseGlobalAlertSettings   types.Bool   `tfsdk:"use_global_alert_settings"`
	Bundle                   types.Object `tfsdk:"bundle"`
	Runtime                  types.Object `tfsdk:"runtime"`
	GroupID                  types.Int64  `tfsdk:"group_id"`
	GroupOrder               types.Int64  `tfsdk:"group_order"`
	TriggerIncident          types.Object `tfsdk:"trigger_incident"`
}

// PlaywrightCheckSuiteBundleModel describes the bundle attribute data model.
type PlaywrightCheckSuiteBundleModel struct {
	ID       types.String `tfsdk:"id"`
	Metadata types.String `tfsdk:"metadata"`
}

var playwrightCheckSuiteBundleAttrTypes = map[string]attr.Type{
	"id":       types.StringType,
	"metadata": types.StringType,
}

// PlaywrightCheckSuiteRuntimeModel describes the runtime attribute data
// model.
type PlaywrightCheckSuiteRuntimeModel struct {
	AutoDetect types.Bool   `tfsdk:"auto_detect"`
	WorkingDir types.String `tfsdk:"working_dir"`
	Steps      types.Object `tfsdk:"steps"`
	Playwright types.Object `tfsdk:"playwright"`
	Engine     types.Object `tfsdk:"engine"`
}

// PlaywrightCheckSuiteRuntimeStepsModel describes the runtime.steps attribute
// data model.
type PlaywrightCheckSuiteRuntimeStepsModel struct {
	Install types.Object `tfsdk:"install"`
	Test    types.Object `tfsdk:"test"`
}

// PlaywrightCheckSuiteRuntimeCommandModel describes the runtime.steps.install
// and runtime.steps.test attribute data models.
type PlaywrightCheckSuiteRuntimeCommandModel struct {
	Command types.String `tfsdk:"command"`
}

// PlaywrightCheckSuiteRuntimePlaywrightModel describes the runtime.playwright
// attribute data model.
type PlaywrightCheckSuiteRuntimePlaywrightModel struct {
	Version types.String `tfsdk:"version"`
	Device  types.Set    `tfsdk:"device"`
}

// PlaywrightCheckSuiteRuntimeDeviceModel describes the
// runtime.playwright.device element data model.
type PlaywrightCheckSuiteRuntimeDeviceModel struct {
	Type types.String `tfsdk:"type"`
}

// PlaywrightCheckSuiteRuntimeEngineModel describes the runtime.engine
// attribute data model.
type PlaywrightCheckSuiteRuntimeEngineModel struct {
	Name    types.String `tfsdk:"name"`
	Version types.String `tfsdk:"version"`
}

var playwrightCheckSuiteRuntimeCommandAttrTypes = map[string]attr.Type{
	"command": types.StringType,
}

var playwrightCheckSuiteRuntimeStepsAttrTypes = map[string]attr.Type{
	"install": types.ObjectType{AttrTypes: playwrightCheckSuiteRuntimeCommandAttrTypes},
	"test":    types.ObjectType{AttrTypes: playwrightCheckSuiteRuntimeCommandAttrTypes},
}

var playwrightCheckSuiteRuntimeDeviceAttrTypes = map[string]attr.Type{
	"type": types.StringType,
}

var playwrightCheckSuiteRuntimeDeviceElementType = types.ObjectType{
	AttrTypes: playwrightCheckSuiteRuntimeDeviceAttrTypes,
}

var playwrightCheckSuiteRuntimePlaywrightAttrTypes = map[string]attr.Type{
	"version": types.StringType,
	"device":  types.SetType{ElemType: playwrightCheckSuiteRuntimeDeviceElementType},
}

var playwrightCheckSuiteRuntimeEngineAttrTypes = map[string]attr.Type{
	"name":    types.StringType,
	"version": types.StringType,
}

var playwrightCheckSuiteRuntimeAttrTypes = map[string]attr.Type{
	"auto_detect": types.BoolType,
	"working_dir": types.StringType,
	"steps":       types.ObjectType{AttrTypes: playwrightCheckSuiteRuntimeStepsAttrTypes},
	"playwright":  types.ObjectType{AttrTypes: playwrightCheckSuiteRuntimePlaywrightAttrTypes},
	"engine":      types.ObjectType{AttrTypes: playwrightCheckSuiteRuntimeEngineAttrTypes},
}

func (r *PlaywrightCheckSuiteResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_playwright_check_suite"
}

func (r *PlaywrightCheckSuiteResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     sdkv2SchemaVersion + 1,
		Description: "Creates a Playwright check from a code bundle.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the check.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the check.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "A description of the check.",
			},
			frequencyAttributeName: makeFrequencyAttributeSchema(FrequencyAttributeSchemaOptions{
				Monitor:            false,
				AllowHighFrequency: false,
			}),
			"activated": schema.BoolAttribute{
				Required:    true,
				Description: "Determines whether the check will run periodically or not after being deployed.",
			},
			"muted": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Determines if any notifications will be sent out when the check fails and/or recovers. (Default `false`).",
			},
			"run_parallel": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Determines whether the check should run on all selected locations in parallel or round-robin. (Default `false`).",
			},
			"locations": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "An array of one or more data center locations where to run the this check.",
			},
			"tags": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "A list of tags for organizing and filtering checks and monitors.",
			},
			"private_locations": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "An array of one or more private locations slugs.",
			},
			triggerIncidentAttributeName: makeTriggerIncidentSchema(),
			alertSettingsAttributeName: makeAlertSettingsAttributeSchema(AlertSettingsAttributeSchemaOptions{
				Monitor: false,
			}),
			"use_global_alert_settings": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "When true, the account level alert settings will be used, not the alert setting defined on this check. (Default `true`).",
			},
			"runtime": schema.SingleNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Configure the runtime environment of the Playwright check.",
				PlanModifiers: []planmodifier.Object{
					playwrightCheckSuiteRuntimePlanModifier{},
				},
				Attributes: map[string]schema.Attribute{
					"auto_detect": schema.BoolAttribute{
						Optional: true,
						Computed: true,
						Default:  booldefault.StaticBool(true),
						Description: "Whether to automatically detect appropriate runtime environment " +
							"configuration from the bundle. " +
							"(Default `true`).",
					},
					"working_dir": schema.StringAttribute{
						Optional: true,
						Computed: true,
						Description: "The working directory in which runtime commands are executed. " +
							"This is useful for monorepos or workspaces where the Playwright " +
							"project is in a subdirectory. Use \".\" to explicitly specify the root.",
						Validators: []validator.String{
							playwrightWorkingDirValidator{},
						},
					},
					"steps": schema.SingleNestedAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Customize the actions taken during test execution.",
						Attributes: map[string]schema.Attribute{
							"install": schema.SingleNestedAttribute{
								Optional: true,
								Description: "Customize the install step, which is used to initialize the " +
									"environment prior to starting the test run.",
								Attributes: map[string]schema.Attribute{
									"command": schema.StringAttribute{
										Optional: true,
										Description: "The command used to install dependencies prior to " +
											"running Playwright. The default value is the appropriate " +
											"install command for your package manager (e.g. " +
											"`npm install` for `npm`).",
									},
								},
							},
							"test": schema.SingleNestedAttribute{
								Optional:    true,
								Computed:    true,
								Description: "Customize the test step.",
								Attributes: map[string]schema.Attribute{
									"command": schema.StringAttribute{
										Optional: true,
										Computed: true,
										Description: "The command used to run Playwright. The default " +
											"value is the appropriate exec command for your package " +
											"manager (e.g. `npx playwright test` for `npm`).",
									},
								},
							},
						},
					},
					"playwright": schema.SingleNestedAttribute{
						Optional: true,
						Computed: true,
						Description: "Configure the Playwright capabilities that should be made available " +
							"to the runtime environment.",
						Attributes: map[string]schema.Attribute{
							"version": schema.StringAttribute{
								Optional: true,
								Computed: true,
								Description: "The Playwright version to use. Defaults to the " +
									"version detected from the code bundle's lockfile.",
							},
							"device": schema.SetNestedAttribute{
								Optional: true,
								Computed: true,
								Description: "The list of devices that should be made available for Playwright. " +
									"Defaults to chromium, firefox, and webkit.",
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"type": schema.StringAttribute{
											Required:    true,
											Description: "The type of the device.",
										},
									},
								},
							},
						},
					},
					"engine": schema.SingleNestedAttribute{
						Optional: true,
						Computed: true,
						Description: "The JavaScript engine used to run the Playwright tests. " +
							"When `auto_detect` is enabled, no engine is set, and the code bundle has a " +
							"lockfile at its root, the engine is detected from files at the bundle root. " +
							"Node is taken from the first of `.node-version`, `.nvmrc`, `.tool-versions`, " +
							"the `volta.node` pin in `package.json`, then `engines.node` in `package.json`; " +
							"Bun from the first of `.tool-versions`, `.bun-version`, then `engines.bun`. " +
							"When both are found, the engine matching the lockfile's package manager wins. " +
							"`volta.extends` is not followed.",
						Attributes: map[string]schema.Attribute{
							"name": schema.StringAttribute{
								Required:    true,
								Description: `The engine name. Valid values are "node" or "bun".`,
								Validators: []validator.String{
									stringvalidator.OneOf("node", "bun"),
								},
							},
							"version": schema.StringAttribute{
								Required: true,
								Description: `The engine version (e.g. "22", "24", "26" for node; "1.3", "1.4" for bun). ` +
									`Use "1" for bun to run the newest Bun 1.x available.`,
								Validators: []validator.String{
									playwrightVersionFormatValidator{},
								},
							},
						},
					},
				},
			},
			"group_id": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "The ID of the check group that this check is part of.",
			},
			"group_order": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "The position of the check in the check group. It determines in what order checks and monitors are run when a group is triggered from the API or from CI/CD.",
			},
			"bundle": schema.SingleNestedAttribute{
				Required:    true,
				Description: "Attaches a code bundle to the check.",
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Required:    true,
						Description: "The ID of the code bundle.",
					},
					playwrightMetadataAttributeName: schema.StringAttribute{
						Required:    true,
						Description: "The generated metadata of the code bundle.",
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			environmentVariableAttributeName: makeEnvironmentVariableSchema(EnvironmentVariableAttributeSchemaOptions{
				Description: "Insert environment variables into the execution environment.",
			}),
			alertChannelSubscriptionAttributeName: makeAlertChannelSubscriptionSchema(AlertChannelSubscriptionAttributeSchemaOptions{}),
		},
	}
}

// UpgradeState upgrades state written by the provider's SDK v2
// implementation; see sdkv2StateUpgraders.
func (r *PlaywrightCheckSuiteResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return sdkv2StateUpgraders(ctx, r)
}

func (r *PlaywrightCheckSuiteResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *PlaywrightCheckSuiteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PlaywrightCheckSuiteResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	check := playwrightCheckSuiteFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	newCheck, err := r.client.CreatePlaywrightCheck(apiCtx, *check)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Playwright Check Suite",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	plan.ID = types.StringValue(newCheck.ID)

	playwrightCheckSuiteFinalizeState(ctx, &plan, newCheck, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PlaywrightCheckSuiteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PlaywrightCheckSuiteResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	check, err := r.client.GetPlaywrightCheck(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Read Playwright Check Suite",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	playwrightCheckSuiteToModel(ctx, check, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PlaywrightCheckSuiteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PlaywrightCheckSuiteResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	check := playwrightCheckSuiteFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	stored, err := readForUpdate(ctx, r.client.GetPlaywrightCheck, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Playwright Check Suite",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	// Terraform has no attributes for these, and the update would clear them.
	check.Intent = stored.Intent
	check.AIAutoRepairEnabled = stored.AIAutoRepairEnabled

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	updatedCheck, err := r.client.UpdatePlaywrightCheck(apiCtx, plan.ID.ValueString(), *check)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Playwright Check Suite",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}

	playwrightCheckSuiteFinalizeState(ctx, &plan, updatedCheck, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PlaywrightCheckSuiteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PlaywrightCheckSuiteResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	err := r.client.DeletePlaywrightCheck(apiCtx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Delete Playwright Check Suite",
			fmt.Sprintf("Checkly API error: %s", err),
		)
		return
	}
}

func (r *PlaywrightCheckSuiteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, tfpath.Root("id"), req, resp)
}

// playwrightCheckSuiteFromModel converts the resource data model to its API
// representation.
func playwrightCheckSuiteFromModel(
	ctx context.Context,
	model *PlaywrightCheckSuiteResourceModel,
	diags *diag.Diagnostics,
) *checkly.PlaywrightCheck {
	check := checkly.PlaywrightCheck{
		ID:                        model.ID.ValueString(),
		Name:                      model.Name.ValueString(),
		Frequency:                 int(model.Frequency.ValueInt64()),
		Activated:                 model.Activated.ValueBool(),
		Muted:                     model.Muted.ValueBool(),
		RunParallel:               model.RunParallel.ValueBool(),
		Locations:                 stringsFromSet(ctx, model.Locations, diags),
		Tags:                      stringsFromSet(ctx, model.Tags, diags),
		UseGlobalAlertSettings:    model.UseGlobalAlertSettings.ValueBool(),
		GroupID:                   model.GroupID.ValueInt64(),
		GroupOrder:                int(model.GroupOrder.ValueInt64()),
		AlertChannelSubscriptions: alertChannelSubscriptionsFromSet(ctx, model.AlertChannelSubscription, diags),
		TriggerIncident:           triggerIncidentFromObject(ctx, model.TriggerIncident, diags),
		EnvironmentVariables:      environmentVariablesFromList(ctx, model.EnvironmentVariable, diags),
	}

	// An empty description clears any previously set value.
	if description := model.Description.ValueString(); description != "" {
		check.Description = &description
	}

	alertSettings := alertSettingsFromObject(ctx, model.AlertSettings, diags)
	check.AlertSettings = &alertSettings

	privateLocations := stringsFromSetOrEmpty(ctx, model.PrivateLocations, diags)
	check.PrivateLocations = &privateLocations

	if !model.Bundle.IsNull() && !model.Bundle.IsUnknown() {
		var bundle PlaywrightCheckSuiteBundleModel
		diags.Append(model.Bundle.As(ctx, &bundle, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil
		}

		if !bundle.ID.IsNull() && !bundle.ID.IsUnknown() {
			bundlePath, err := base64.StdEncoding.DecodeString(bundle.ID.ValueString())
			if err != nil {
				diags.AddError(
					"Invalid Code Bundle",
					fmt.Sprintf("Invalid code bundle identifier %q: %s", bundle.ID.ValueString(), err),
				)
				return nil
			}

			check.CodeBundlePath = string(bundlePath)
		}

		if !bundle.Metadata.IsNull() && !bundle.Metadata.IsUnknown() {
			metadata, err := PlaywrightCodeBundleMetadataFromString(bundle.Metadata.ValueString())
			if err != nil {
				diags.AddError(
					"Invalid Code Bundle",
					fmt.Sprintf("Failed to decode code bundle metadata: %s", err),
				)
				return nil
			}

			check.CacheHash = metadata.CacheHash
		}
	}

	if !model.Runtime.IsNull() && !model.Runtime.IsUnknown() {
		var runtime PlaywrightCheckSuiteRuntimeModel
		diags.Append(model.Runtime.As(ctx, &runtime, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil
		}

		if !runtime.WorkingDir.IsNull() && !runtime.WorkingDir.IsUnknown() {
			if workingDir := runtime.WorkingDir.ValueString(); workingDir != "" {
				check.WorkingDir = &workingDir
			}
		}

		if !runtime.Steps.IsNull() && !runtime.Steps.IsUnknown() {
			var steps PlaywrightCheckSuiteRuntimeStepsModel
			diags.Append(runtime.Steps.As(ctx, &steps, basetypes.ObjectAsOptions{})...)

			if !steps.Install.IsNull() && !steps.Install.IsUnknown() {
				var install PlaywrightCheckSuiteRuntimeCommandModel
				diags.Append(steps.Install.As(ctx, &install, basetypes.ObjectAsOptions{})...)
				if command := install.Command.ValueString(); command != "" {
					check.InstallCommand = &command
				}
			}

			if !steps.Test.IsNull() && !steps.Test.IsUnknown() {
				var test PlaywrightCheckSuiteRuntimeCommandModel
				diags.Append(steps.Test.As(ctx, &test, basetypes.ObjectAsOptions{})...)
				if command := test.Command.ValueString(); command != "" {
					check.TestCommand = &command
				}
			}
		}

		if !runtime.Playwright.IsNull() && !runtime.Playwright.IsUnknown() {
			var playwright PlaywrightCheckSuiteRuntimePlaywrightModel
			diags.Append(runtime.Playwright.As(ctx, &playwright, basetypes.ObjectAsOptions{})...)

			if !playwright.Version.IsNull() && !playwright.Version.IsUnknown() {
				version := playwright.Version.ValueString()
				check.PlaywrightVersion = &version
			}

			if !playwright.Device.IsNull() && !playwright.Device.IsUnknown() {
				var devices []PlaywrightCheckSuiteRuntimeDeviceModel
				diags.Append(playwright.Device.ElementsAs(ctx, &devices, false)...)

				var browsers []string
				for _, device := range devices {
					browsers = append(browsers, device.Type.ValueString())
				}

				check.Browsers = browsers
			}
		}

		if !runtime.Engine.IsNull() && !runtime.Engine.IsUnknown() {
			var engine PlaywrightCheckSuiteRuntimeEngineModel
			diags.Append(runtime.Engine.As(ctx, &engine, basetypes.ObjectAsOptions{})...)

			name := engine.Name.ValueString()
			version := engine.Version.ValueString()
			check.Engine = (*checkly.PlaywrightEngine)(&name)
			check.EngineVersion = &version
		}
	}

	if diags.HasError() {
		return nil
	}

	return &check
}

// playwrightCheckSuiteFinalizeState resolves the values that remain unknown
// in the plan after apply. The runtime attribute is normally fully known at
// plan time (the plan modifier resolves it from the code bundle metadata);
// only when the configuration contained unknown values is it rebuilt from
// the API response.
func playwrightCheckSuiteFinalizeState(
	ctx context.Context,
	model *PlaywrightCheckSuiteResourceModel,
	check *checkly.PlaywrightCheck,
	diags *diag.Diagnostics,
) {
	alertSettings := check.AlertSettings
	if alertSettings == nil {
		sent := alertSettingsFromObject(ctx, model.AlertSettings, diags)
		alertSettings = &sent
	}
	model.AlertSettings = alertSettingsToObject(ctx, *alertSettings, model.AlertSettings, diags)

	if !playwrightAttrValueFullyKnown(ctx, model.Runtime) {
		model.Runtime = playwrightCheckSuiteRuntimeFromAPI(ctx, check, model.Runtime, diags)
	}
}

// playwrightAttrValueFullyKnown reports whether the value contains no
// unknown values at any level of nesting.
func playwrightAttrValueFullyKnown(ctx context.Context, value attr.Value) bool {
	tv, err := value.ToTerraformValue(ctx)
	return err == nil && tv.IsFullyKnown()
}

// playwrightCheckSuiteToModel updates the model to match the API
// representation of the check. Values that the API does not track (the
// bundle attachment, runtime.auto_detect) are preserved from the prior model
// contents.
func playwrightCheckSuiteToModel(
	ctx context.Context,
	check *checkly.PlaywrightCheck,
	model *PlaywrightCheckSuiteResourceModel,
	diags *diag.Diagnostics,
) {
	model.ID = types.StringValue(check.ID)
	model.Name = types.StringValue(check.Name)

	description := ""
	if check.Description != nil {
		description = *check.Description
	}
	model.Description = types.StringValue(description)

	model.Frequency = types.Int64Value(int64(check.Frequency))
	model.Activated = types.BoolValue(check.Activated)
	model.Muted = types.BoolValue(check.Muted)
	model.RunParallel = types.BoolValue(check.RunParallel)
	model.UseGlobalAlertSettings = types.BoolValue(check.UseGlobalAlertSettings)
	model.GroupID = types.Int64Value(check.GroupID)
	model.GroupOrder = types.Int64Value(int64(check.GroupOrder))

	model.Locations = stringSetFromAPIEmptyIfUnknown(ctx, check.Locations, model.Locations, diags)
	model.Tags = stringSetFromAPIEmptyIfUnknown(ctx, check.Tags, model.Tags, diags)

	var privateLocations []string
	if check.PrivateLocations != nil {
		privateLocations = *check.PrivateLocations
	}
	model.PrivateLocations = stringSetFromAPIEmptyIfUnknown(ctx, privateLocations, model.PrivateLocations, diags)

	if len(check.EnvironmentVariables) == 0 && model.EnvironmentVariable.IsNull() {
		// Keep null; an absent block list stays absent.
	} else {
		model.EnvironmentVariable = environmentVariablesToList(ctx, check.EnvironmentVariables, model.EnvironmentVariable, diags)
	}

	if len(check.AlertChannelSubscriptions) == 0 && model.AlertChannelSubscription.IsNull() {
		// Keep null; an absent block list stays absent.
	} else {
		model.AlertChannelSubscription = alertChannelSubscriptionsToSet(ctx, check.AlertChannelSubscriptions, diags)
	}

	model.TriggerIncident = triggerIncidentToObject(ctx, check.TriggerIncident, diags)

	alertSettings := check.AlertSettings
	if alertSettings == nil {
		alertSettings = &checkly.AlertSettings{
			EscalationType: checkly.EscalationTypeRunBased,
			RunBasedEscalation: checkly.RunBasedEscalation{
				FailedRunThreshold: 1,
			},
		}
	}
	model.AlertSettings = alertSettingsToObject(ctx, *alertSettings, model.AlertSettings, diags)

	// The bundle attachment only exists on the client; the prior value is
	// kept as-is.

	model.Runtime = playwrightCheckSuiteRuntimeFromAPI(ctx, check, model.Runtime, diags)
}

// playwrightCheckSuiteRuntimeFromAPI converts the runtime-related fields of
// the API representation to the runtime attribute value. The auto_detect
// flag is not tracked by the API and is preserved from the prior value. An
// install step configured as an empty object is also preserved from the
// prior value, since the API cannot distinguish it from an absent install
// step.
func playwrightCheckSuiteRuntimeFromAPI(
	ctx context.Context,
	check *checkly.PlaywrightCheck,
	prior types.Object,
	diags *diag.Diagnostics,
) types.Object {
	var priorModel PlaywrightCheckSuiteRuntimeModel
	if !prior.IsNull() && !prior.IsUnknown() {
		diags.Append(prior.As(ctx, &priorModel, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return types.ObjectNull(playwrightCheckSuiteRuntimeAttrTypes)
		}
	}

	autoDetect := true
	if !priorModel.AutoDetect.IsNull() && !priorModel.AutoDetect.IsUnknown() {
		autoDetect = priorModel.AutoDetect.ValueBool()
	}

	workingDir := types.StringNull()
	if check.WorkingDir != nil && *check.WorkingDir != "" {
		workingDir = types.StringValue(*check.WorkingDir)
	}

	install := types.ObjectNull(playwrightCheckSuiteRuntimeCommandAttrTypes)
	if check.InstallCommand != nil && *check.InstallCommand != "" {
		install = playwrightCheckSuiteCommandObject(ctx, *check.InstallCommand, diags)
	} else if priorInstall := playwrightCheckSuitePriorInstall(ctx, priorModel.Steps, diags); !priorInstall.IsNull() {
		install = priorInstall
	}

	test := types.ObjectNull(playwrightCheckSuiteRuntimeCommandAttrTypes)
	if check.TestCommand != nil && *check.TestCommand != "" {
		test = playwrightCheckSuiteCommandObject(ctx, *check.TestCommand, diags)
	}

	steps := types.ObjectNull(playwrightCheckSuiteRuntimeStepsAttrTypes)
	if !install.IsNull() || !test.IsNull() {
		var d diag.Diagnostics
		steps, d = types.ObjectValue(playwrightCheckSuiteRuntimeStepsAttrTypes, map[string]attr.Value{
			"install": install,
			"test":    test,
		})
		diags.Append(d...)
	}

	playwright := types.ObjectNull(playwrightCheckSuiteRuntimePlaywrightAttrTypes)
	if len(check.Browsers) != 0 || (check.PlaywrightVersion != nil && *check.PlaywrightVersion != "") {
		version := types.StringNull()
		if check.PlaywrightVersion != nil && *check.PlaywrightVersion != "" {
			version = types.StringValue(*check.PlaywrightVersion)
		}

		device := types.SetNull(playwrightCheckSuiteRuntimeDeviceElementType)
		if len(check.Browsers) != 0 {
			device = playwrightCheckSuiteDeviceSet(ctx, check.Browsers, diags)
		}

		var d diag.Diagnostics
		playwright, d = types.ObjectValue(playwrightCheckSuiteRuntimePlaywrightAttrTypes, map[string]attr.Value{
			"version": version,
			"device":  device,
		})
		diags.Append(d...)
	}

	engine := types.ObjectNull(playwrightCheckSuiteRuntimeEngineAttrTypes)
	if check.Engine != nil && *check.Engine != "" {
		engineVersion := types.StringNull()
		if check.EngineVersion != nil && *check.EngineVersion != "" {
			engineVersion = types.StringValue(*check.EngineVersion)
		}

		var d diag.Diagnostics
		engine, d = types.ObjectValue(playwrightCheckSuiteRuntimeEngineAttrTypes, map[string]attr.Value{
			"name":    types.StringValue(string(*check.Engine)),
			"version": engineVersion,
		})
		diags.Append(d...)
	}

	runtime, d := types.ObjectValue(playwrightCheckSuiteRuntimeAttrTypes, map[string]attr.Value{
		"auto_detect": types.BoolValue(autoDetect),
		"working_dir": workingDir,
		"steps":       steps,
		"playwright":  playwright,
		"engine":      engine,
	})
	diags.Append(d...)

	return runtime
}

// playwrightCheckSuitePriorInstall extracts the install step from a prior
// steps value if the install step is present without a command (an empty
// install object in the configuration).
func playwrightCheckSuitePriorInstall(ctx context.Context, priorSteps types.Object, diags *diag.Diagnostics) types.Object {
	null := types.ObjectNull(playwrightCheckSuiteRuntimeCommandAttrTypes)

	if priorSteps.IsNull() || priorSteps.IsUnknown() {
		return null
	}

	var steps PlaywrightCheckSuiteRuntimeStepsModel
	diags.Append(priorSteps.As(ctx, &steps, basetypes.ObjectAsOptions{})...)
	if diags.HasError() || steps.Install.IsNull() || steps.Install.IsUnknown() {
		return null
	}

	var install PlaywrightCheckSuiteRuntimeCommandModel
	diags.Append(steps.Install.As(ctx, &install, basetypes.ObjectAsOptions{})...)
	if diags.HasError() || !install.Command.IsNull() {
		return null
	}

	return steps.Install
}

// playwrightCheckSuiteCommandObject builds a command object value.
func playwrightCheckSuiteCommandObject(ctx context.Context, command string, diags *diag.Diagnostics) types.Object {
	obj, d := types.ObjectValue(playwrightCheckSuiteRuntimeCommandAttrTypes, map[string]attr.Value{
		"command": types.StringValue(command),
	})
	diags.Append(d...)
	return obj
}

// playwrightCheckSuiteDeviceSet builds a device set value from a list of
// device type names.
func playwrightCheckSuiteDeviceSet(ctx context.Context, deviceTypes []string, diags *diag.Diagnostics) types.Set {
	elements := make([]attr.Value, 0, len(deviceTypes))
	for _, deviceType := range deviceTypes {
		obj, d := types.ObjectValue(playwrightCheckSuiteRuntimeDeviceAttrTypes, map[string]attr.Value{
			"type": types.StringValue(deviceType),
		})
		diags.Append(d...)
		elements = append(elements, obj)
	}

	set, d := types.SetValue(playwrightCheckSuiteRuntimeDeviceElementType, elements)
	diags.Append(d...)
	return set
}

var _ planmodifier.Object = (*playwrightCheckSuiteRuntimePlanModifier)(nil)

// playwrightCheckSuiteRuntimePlanModifier resolves the full runtime
// configuration at plan time. Values that are not set in the configuration
// are auto-detected from the attached code bundle's metadata (when
// auto_detect is enabled), so the plan always shows the concrete runtime
// configuration that will be applied and changing the code bundle
// immediately surfaces the resulting runtime changes.
type playwrightCheckSuiteRuntimePlanModifier struct{}

func (m playwrightCheckSuiteRuntimePlanModifier) Description(ctx context.Context) string {
	return "Resolves the runtime configuration from the code bundle metadata at plan time."
}

func (m playwrightCheckSuiteRuntimePlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m playwrightCheckSuiteRuntimePlanModifier) PlanModifyObject(ctx context.Context, req planmodifier.ObjectRequest, resp *planmodifier.ObjectResponse) {
	// When the configuration contains values that are not yet known, the
	// runtime cannot be resolved during plan; it is resolved from the API
	// response after apply instead.
	if !playwrightAttrValueFullyKnown(ctx, req.ConfigValue) {
		return
	}

	var bundleObject types.Object
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, tfpath.Root("bundle"), &bundleObject)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// When the bundle metadata is not yet known (e.g. the archive is only
	// produced during apply, so its metadata could not be resolved at plan
	// time), the runtime cannot be resolved either. Leave the plan value
	// unknown instead of resolving against empty metadata, which could raise
	// spurious auto-detection errors and produce an inconsistent final plan;
	// the runtime is resolved when the modifier re-runs during apply with the
	// metadata available.
	unknown := types.ObjectUnknown(playwrightCheckSuiteRuntimeAttrTypes)
	if bundleObject.IsUnknown() {
		resp.PlanValue = unknown
		return
	}

	metadata := new(PlaywrightCodeBundleMetadata)
	if !bundleObject.IsNull() {
		var bundle PlaywrightCheckSuiteBundleModel
		resp.Diagnostics.Append(bundleObject.As(ctx, &bundle, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}

		if bundle.Metadata.IsUnknown() {
			resp.PlanValue = unknown
			return
		}

		if !bundle.Metadata.IsNull() {
			decoded, err := PlaywrightCodeBundleMetadataFromString(bundle.Metadata.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Invalid Code Bundle",
					fmt.Sprintf("Failed to decode code bundle metadata: %s", err),
				)
				return
			}
			metadata = decoded
		}
	}

	resolved := playwrightCheckSuiteResolveRuntime(ctx, req.ConfigValue, metadata, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.PlanValue = resolved
}

// playwrightCheckSuiteResolveRuntime resolves the configured runtime value
// against the code bundle metadata, filling unset values by auto-detection
// (when enabled) and validating that a complete runtime configuration
// results.
func playwrightCheckSuiteResolveRuntime(
	ctx context.Context,
	configRuntime types.Object,
	metadata *PlaywrightCodeBundleMetadata,
	diags *diag.Diagnostics,
) types.Object {
	null := types.ObjectNull(playwrightCheckSuiteRuntimeAttrTypes)

	var cfg PlaywrightCheckSuiteRuntimeModel
	if !configRuntime.IsNull() {
		diags.Append(configRuntime.As(ctx, &cfg, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return null
		}
	}

	autoDetect := true
	if !cfg.AutoDetect.IsNull() {
		autoDetect = cfg.AutoDetect.ValueBool()
	}

	workingDir := cfg.WorkingDir

	install := types.ObjectNull(playwrightCheckSuiteRuntimeCommandAttrTypes)
	testCommand := types.StringNull()
	if !cfg.Steps.IsNull() {
		var steps PlaywrightCheckSuiteRuntimeStepsModel
		diags.Append(cfg.Steps.As(ctx, &steps, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return null
		}

		// The install step is never auto-detected; it passes through exactly
		// as configured.
		install = steps.Install

		if !steps.Test.IsNull() {
			var test PlaywrightCheckSuiteRuntimeCommandModel
			diags.Append(steps.Test.As(ctx, &test, basetypes.ObjectAsOptions{})...)
			if diags.HasError() {
				return null
			}
			testCommand = test.Command
		}
	}

	version := types.StringNull()
	deviceTypes := []string(nil)
	devicesConfigured := false
	if !cfg.Playwright.IsNull() {
		var playwright PlaywrightCheckSuiteRuntimePlaywrightModel
		diags.Append(cfg.Playwright.As(ctx, &playwright, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return null
		}

		version = playwright.Version

		if !playwright.Device.IsNull() {
			devicesConfigured = true

			var devices []PlaywrightCheckSuiteRuntimeDeviceModel
			diags.Append(playwright.Device.ElementsAs(ctx, &devices, false)...)
			if diags.HasError() {
				return null
			}

			for _, device := range devices {
				deviceTypes = append(deviceTypes, device.Type.ValueString())
			}
		}
	}

	engineName := types.StringNull()
	engineVersion := types.StringNull()
	if !cfg.Engine.IsNull() {
		var engine PlaywrightCheckSuiteRuntimeEngineModel
		diags.Append(cfg.Engine.As(ctx, &engine, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return null
		}

		engineName = engine.Name
		engineVersion = engine.Version

		if config, ok := engineConfigs[engineName.ValueString()]; ok {
			res := resolveEngineVersion(engineVersion.ValueString(), config)
			if res.Denied || len(res.Notices) > 0 {
				diags.AddError(
					"Invalid Runtime Configuration",
					fmt.Sprintf(
						"\"runtime.engine.version\" is not valid: %s",
						strings.Join(res.Notices, "; "),
					),
				)
				return null
			}
		}
	}

	if autoDetect {
		if workingDir.IsNull() && metadata.WorkingDir != "" {
			workingDir = types.StringValue(metadata.WorkingDir)
		}

		if version.IsNull() && metadata.PlaywrightVersion != "" {
			version = types.StringValue(metadata.PlaywrightVersion)
		}

		if !devicesConfigured {
			deviceTypes = playwrightCheckSuiteDefaultBrowsers
		}

		if testCommand.IsNull() {
			if command, ok := playwrightCheckSuiteDefaultTestCommands[metadata.PackageManager]; ok {
				testCommand = types.StringValue(command)
			} else {
				// Just assume NPM if we couldn't detect a package manager.
				testCommand = types.StringValue(playwrightCheckSuiteDefaultTestCommands["npm"])
			}
		}

		if engineName.IsNull() && metadata.Engine != "" {
			if metadata.EngineVersion == "" {
				diags.AddError(
					"Invalid Runtime Configuration",
					fmt.Sprintf(
						"detected %s version %q from %s, but the version was denied or could not be resolved; "+
							"set \"runtime.engine\" explicitly or update to a supported version",
						metadata.Engine,
						metadata.EngineRawVersion,
						metadata.EngineSource,
					),
				)
				return null
			}

			engineName = types.StringValue(metadata.Engine)
			engineVersion = types.StringValue(metadata.EngineVersion)
		}
	}

	if version.IsNull() || version.ValueString() == "" {
		if autoDetect {
			diags.AddError(
				"Invalid Runtime Configuration",
				"unable to detect Playwright version from the code bundle's lockfile; "+
					"set \"runtime.playwright.version\" explicitly or ensure the archive "+
					"contains a package-lock.json, pnpm-lock.yaml, yarn.lock, or bun.lock with @playwright/test",
			)
		} else {
			diags.AddError(
				"Invalid Runtime Configuration",
				`"runtime.playwright.version" is required when "runtime.auto_detect" is false`,
			)
		}
		return null
	}

	if len(deviceTypes) == 0 {
		if autoDetect {
			diags.AddError(
				"Invalid Runtime Configuration",
				"unable to detect default devices from the code bundle; "+
					"set \"runtime.playwright.device\" entries explicitly",
			)
		} else {
			diags.AddError(
				"Invalid Runtime Configuration",
				`at least one "runtime.playwright.device" entry is required when "runtime.auto_detect" is false`,
			)
		}
		return null
	}

	if testCommand.IsNull() || testCommand.ValueString() == "" {
		if autoDetect {
			diags.AddError(
				"Invalid Runtime Configuration",
				"unable to detect an appropriate test command for the code bundle; "+
					"set \"runtime.steps.test.command\" explicitly or ensure the archive "+
					"contains a supported lockfile",
			)
		} else {
			diags.AddError(
				"Invalid Runtime Configuration",
				`"runtime.steps.test.command" is required when "runtime.auto_detect" is false`,
			)
		}
		return null
	}

	test, d := types.ObjectValue(playwrightCheckSuiteRuntimeCommandAttrTypes, map[string]attr.Value{
		"command": testCommand,
	})
	diags.Append(d...)

	steps, d := types.ObjectValue(playwrightCheckSuiteRuntimeStepsAttrTypes, map[string]attr.Value{
		"install": install,
		"test":    test,
	})
	diags.Append(d...)

	playwright, d := types.ObjectValue(playwrightCheckSuiteRuntimePlaywrightAttrTypes, map[string]attr.Value{
		"version": version,
		"device":  playwrightCheckSuiteDeviceSet(ctx, deviceTypes, diags),
	})
	diags.Append(d...)

	engine := types.ObjectNull(playwrightCheckSuiteRuntimeEngineAttrTypes)
	if !engineName.IsNull() {
		engine, d = types.ObjectValue(playwrightCheckSuiteRuntimeEngineAttrTypes, map[string]attr.Value{
			"name":    engineName,
			"version": engineVersion,
		})
		diags.Append(d...)
	}

	runtime, d := types.ObjectValue(playwrightCheckSuiteRuntimeAttrTypes, map[string]attr.Value{
		"auto_detect": types.BoolValue(autoDetect),
		"working_dir": workingDir,
		"steps":       steps,
		"playwright":  playwright,
		"engine":      engine,
	})
	diags.Append(d...)

	return runtime
}
