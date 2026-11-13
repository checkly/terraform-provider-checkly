package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	tfpath "github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/checkly/checkly-go-sdk/v2"
)

var playwrightCheckSuiteEngineNameValues = allowedValues[string]{
	{Value: "node"},
	{Value: "bun"},
}

var playwrightCheckSuiteDefaultBrowsers = []string{"chromium", "firefox", "webkit"}

// playwrightCheckSuiteDeviceTypes are the device types the check runners
// provide: the browsers Playwright downloads, and the branded browsers
// Playwright selects through its channel option.
var playwrightCheckSuiteDeviceTypes = []string{"chromium", "firefox", "webkit", "chrome", "msedge"}

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
	ID                        types.String `tfsdk:"id"`
	Name                      types.String `tfsdk:"name"`
	Description               types.String `tfsdk:"description"`
	Frequency                 types.Int64  `tfsdk:"frequency"`
	Activated                 types.Bool   `tfsdk:"activated"`
	Muted                     types.Bool   `tfsdk:"muted"`
	RunParallel               types.Bool   `tfsdk:"run_parallel"`
	Locations                 types.Set    `tfsdk:"locations"`
	EnvironmentVariables      types.List   `tfsdk:"environment_variables"`
	Tags                      types.Set    `tfsdk:"tags"`
	AlertChannelSubscriptions types.Set    `tfsdk:"alert_channel_subscriptions"`
	PrivateLocations          types.Set    `tfsdk:"private_locations"`
	AlertSettings             types.Object `tfsdk:"alert_settings"`
	UseGlobalAlertSettings    types.Bool   `tfsdk:"use_global_alert_settings"`
	Bundle                    types.Object `tfsdk:"bundle"`
	Runtime                   types.Object `tfsdk:"runtime"`
	GroupID                   types.Int64  `tfsdk:"group_id"`
	GroupOrder                types.Int64  `tfsdk:"group_order"`
	TriggerIncident           types.Object `tfsdk:"trigger_incident"`
	Intent                    types.Object `tfsdk:"intent"`
	AIAutoRepairEnabled       types.Bool   `tfsdk:"ai_auto_repair_enabled"`
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
	Devices types.Set    `tfsdk:"devices"`
}

// PlaywrightCheckSuiteRuntimeDeviceModel describes the
// runtime.playwright.devices element data model.
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
	"devices": types.SetType{ElemType: playwrightCheckSuiteRuntimeDeviceElementType},
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
		Attributes: withCommonCheckAttributes(map[string]schema.Attribute{
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
				Subject:            subjectCheck,
				AllowHighFrequency: false,
			}),
			triggerIncidentAttributeName:     makeTriggerIncidentSchema(),
			checkIntentAttributeName:         makeCheckIntentSchema(),
			aiAutoRepairEnabledAttributeName: makeAIAutoRepairEnabledSchema(),
			alertSettingsAttributeName: makeAlertSettingsAttributeSchema(AlertSettingsAttributeSchemaOptions{
				Subject: subjectCheck,
			}),
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
							"Defaults to `true`.",
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
											"running Playwright. Can't be empty. If not set, the " +
											"install command of your package manager is used, for " +
											"example `npm install` for `npm`.",
										// The API stores an empty command as no command, so
										// an empty one would show as a change in every plan.
										Validators: []validator.String{
											stringvalidator.LengthAtLeast(1),
										},
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
										Description: "The command used to run Playwright. Can't be empty. " +
											"If not set, the exec command of your package manager is " +
											"used, for example `npx playwright test` for `npm`.",
										// An empty command is not sent, so the API applies its
										// default command, which differs from the planned value.
										Validators: []validator.String{
											stringvalidator.LengthAtLeast(1),
										},
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
								Description: "The Playwright version to use. If not set, the " +
									"version detected from the code bundle's lockfile is used.",
							},
							"devices": schema.SetNestedAttribute{
								Optional: true,
								Computed: true,
								Description: "The list of devices that should be made available for Playwright. " +
									"Can't be empty. If not set, `chromium`, `firefox` and `webkit` are used, " +
									"whether or not `auto_detect` is enabled.",
								// An empty set can't be planned as the default devices,
								// since Terraform requires the planned value of a
								// configured attribute to match the configuration, and it
								// can't be sent either: the API applies its own default to
								// a request without devices. Validating the configuration
								// rejects it even while the code bundle metadata is unknown.
								Validators: []validator.Set{
									setvalidator.SizeAtLeast(1),
								},
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"type": schema.StringAttribute{
											Required: true,
											Description: "The type of the device. Possible values are `chromium`, `firefox` and `webkit`, " +
												"which run the browsers Playwright downloads, and `chrome` and `msedge`, " +
												"which run the branded Google Chrome and Microsoft Edge browsers.",
											// The runners provide only these browsers. The API
											// accepts any name, and a check with another one fails
											// every run when its browsers are installed.
											Validators: []validator.String{
												stringvalidator.OneOf(playwrightCheckSuiteDeviceTypes...),
											},
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
								Description: "The engine name. " + playwrightCheckSuiteEngineNameValues.String(),
								Validators: []validator.String{
									stringvalidator.OneOf(playwrightCheckSuiteEngineNameValues.Values()...),
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
			alertChannelSubscriptionAttributeName: makeAlertChannelSubscriptionSchema(AlertChannelSubscriptionAttributeSchemaOptions{Subject: subjectCheck}),
			environmentVariableAttributeName: makeEnvironmentVariableSchema(EnvironmentVariableAttributeSchemaOptions{
				Description: "Insert environment variables into the execution environment.",
			}),
		}, CommonCheckAttributeSchemaOptions{Subject: subjectCheck, RunParallel: true, Locations: true, GroupMembership: true, NoDegradedState: true}),
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
		addAPIError(&resp.Diagnostics, "Unable to Create Playwright Check Suite", err)
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
		addAPIError(&resp.Diagnostics, "Unable to Read Playwright Check Suite", err)
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

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	updatedCheck, err := r.client.UpdatePlaywrightCheck(apiCtx, plan.ID.ValueString(), *check)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to Update Playwright Check Suite", err)
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
		addAPIError(&resp.Diagnostics, "Unable to Delete Playwright Check Suite", err)
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
		Tags:                      stringsFromSet(ctx, model.Tags, diags),
		UseGlobalAlertSettings:    model.UseGlobalAlertSettings.ValueBool(),
		GroupID:                   model.GroupID.ValueInt64(),
		GroupOrder:                groupOrderToAPI(model.GroupOrder),
		AlertChannelSubscriptions: alertChannelSubscriptionsFromSet(ctx, model.AlertChannelSubscriptions, diags),
		TriggerIncident:           triggerIncidentFromObject(ctx, model.TriggerIncident, diags),
		Intent:                    checkIntentFromObject(ctx, model.Intent, diags),
		AIAutoRepairEnabled:       model.AIAutoRepairEnabled.ValueBoolPointer(),
		EnvironmentVariables:      environmentVariablesFromList(ctx, model.EnvironmentVariables, diags),
	}

	// An empty description clears any previously set value.
	if description := model.Description.ValueString(); description != "" {
		check.Description = &description
	}

	alertSettings := alertSettingsFromObject(ctx, model.AlertSettings, diags)
	check.AlertSettings = &alertSettings

	// The locations property is serialized even when empty.
	check.Locations = stringsFromSetOrEmpty(ctx, model.Locations, diags)

	privateLocations := stringsFromSetOrEmpty(ctx, model.PrivateLocations, diags)
	check.PrivateLocations = &privateLocations

	if bundle := objectAs[PlaywrightCheckSuiteBundleModel](ctx, model.Bundle, diags); bundle != nil {
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

	if runtime := objectAs[PlaywrightCheckSuiteRuntimeModel](ctx, model.Runtime, diags); runtime != nil {
		if !runtime.WorkingDir.IsNull() && !runtime.WorkingDir.IsUnknown() {
			if workingDir := runtime.WorkingDir.ValueString(); workingDir != "" {
				check.WorkingDir = &workingDir
			}
		}

		if steps := objectAs[PlaywrightCheckSuiteRuntimeStepsModel](ctx, runtime.Steps, diags); steps != nil {
			if install := objectAs[PlaywrightCheckSuiteRuntimeCommandModel](ctx, steps.Install, diags); install != nil {
				if command := install.Command.ValueString(); command != "" {
					check.InstallCommand = &command
				}
			}

			if test := objectAs[PlaywrightCheckSuiteRuntimeCommandModel](ctx, steps.Test, diags); test != nil {
				if command := test.Command.ValueString(); command != "" {
					check.TestCommand = &command
				}
			}
		}

		if playwright := objectAs[PlaywrightCheckSuiteRuntimePlaywrightModel](ctx, runtime.Playwright, diags); playwright != nil {
			if !playwright.Version.IsNull() && !playwright.Version.IsUnknown() {
				version := playwright.Version.ValueString()
				check.PlaywrightVersion = &version
			}

			if !playwright.Devices.IsNull() && !playwright.Devices.IsUnknown() {
				var devices []PlaywrightCheckSuiteRuntimeDeviceModel
				diags.Append(playwright.Devices.ElementsAs(ctx, &devices, false)...)

				var browsers []string
				for _, device := range devices {
					browsers = append(browsers, device.Type.ValueString())
				}

				check.Browsers = browsers
			}
		}

		if engine := objectAs[PlaywrightCheckSuiteRuntimeEngineModel](ctx, runtime.Engine, diags); engine != nil {
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

	if valueContainsUnknown(ctx, model.Runtime) {
		model.Runtime = playwrightCheckSuiteRuntimeFromAPI(ctx, check, model.Runtime, diags)
	}
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
	model.GroupID = int64ValueOrNull(check.GroupID)
	model.GroupOrder = groupOrderFromAPI(check.GroupOrder, model.GroupOrder)

	model.Locations = setFromStrings(ctx, check.Locations, diags)
	model.Tags = setFromStrings(ctx, check.Tags, diags)

	model.PrivateLocations = setFromStrings(ctx, valueOrZero(check.PrivateLocations), diags)

	model.EnvironmentVariables = environmentVariablesToList(ctx, check.EnvironmentVariables, model.EnvironmentVariables, diags)

	model.AlertChannelSubscriptions = alertChannelSubscriptionsToSet(ctx, check.AlertChannelSubscriptions, diags)

	model.TriggerIncident = triggerIncidentToObject(ctx, check.TriggerIncident, diags)
	model.Intent = checkIntentToObject(ctx, check.Intent, diags)
	model.AIAutoRepairEnabled = types.BoolPointerValue(check.AIAutoRepairEnabled)

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
			"devices": device,
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

	steps := objectAs[PlaywrightCheckSuiteRuntimeStepsModel](ctx, priorSteps, diags)
	if steps == nil {
		return null
	}

	install := objectAs[PlaywrightCheckSuiteRuntimeCommandModel](ctx, steps.Install, diags)
	if install == nil || !install.Command.IsNull() {
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
	if valueContainsUnknown(ctx, req.ConfigValue) {
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
// against the code bundle metadata. It fills unset devices with the static
// default devices, fills other unset values by auto-detection from the bundle
// metadata (when enabled), and validates that a complete runtime
// configuration results.
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
	// The default devices apply whether or not auto-detection is enabled:
	// they are not detected from the bundle, and leaving devices out of the
	// configuration means the defaults either way.
	deviceTypes := playwrightCheckSuiteDefaultBrowsers
	if !cfg.Playwright.IsNull() {
		var playwright PlaywrightCheckSuiteRuntimePlaywrightModel
		diags.Append(cfg.Playwright.As(ctx, &playwright, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return null
		}

		version = playwright.Version

		if !playwright.Devices.IsNull() {
			var devices []PlaywrightCheckSuiteRuntimeDeviceModel
			diags.Append(playwright.Devices.ElementsAs(ctx, &devices, false)...)
			if diags.HasError() {
				return null
			}

			deviceTypes = make([]string, 0, len(devices))
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

	// The schema validator rejects a configured empty devices set, but skips a
	// value that is unknown at validation time and only becomes empty later.
	if len(deviceTypes) == 0 {
		diags.AddError(
			"Invalid Runtime Configuration",
			`"runtime.playwright.devices" can't be empty; leave it out to use the default devices`,
		)
		return null
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
		"devices": playwrightCheckSuiteDeviceSet(ctx, deviceTypes, diags),
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
