package provider

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	checkly "github.com/checkly/checkly-go-sdk"
)

const (
	defaultAPIURL = "https://api.checklyhq.com"
)

var (
	_ provider.Provider              = (*ChecklyProvider)(nil)
	_ provider.ProviderWithFunctions = (*ChecklyProvider)(nil)
)

type ChecklyProvider struct {
	version string
}

type ChecklyProviderModel struct {
	APIKey    types.String `tfsdk:"api_key"`
	APIURL    types.String `tfsdk:"api_url"`
	AccountID types.String `tfsdk:"account_id"`
}

func (p *ChecklyProvider) Metadata(
	ctx context.Context,
	req provider.MetadataRequest,
	resp *provider.MetadataResponse,
) {
	resp.TypeName = "checkly"
	resp.Version = p.version
}

func (p *ChecklyProvider) Schema(
	ctx context.Context,
	req provider.SchemaRequest,
	resp *provider.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Optional:  true,
				Sensitive: true, // FIXME: Keep this? Old code did not set it.
			},
			"api_url": schema.StringAttribute{
				Optional: true,
			},
			"account_id": schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

func (p *ChecklyProvider) Configure(
	ctx context.Context,
	req provider.ConfigureRequest,
	resp *provider.ConfigureResponse,
) {
	var config ChecklyProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.APIKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Unknown Checkly API Key",
			"The provider cannot create the Checkly API client as there is "+
				"an unknown configuration value for the Checkly API Key. "+
				"Either target apply the source of the value first, set the "+
				"value statically in the configuration, or use the "+
				"CHECKLY_API_KEY environment variable.",
		)
	}

	if config.APIURL.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_url"),
			"Unknown Checkly API URL",
			"The provider cannot create the Checkly API client as there is "+
				"an unknown configuration value for the Checkly API URL. "+
				"Either target apply the source of the value first, set the "+
				"value statically in the configuration, or use the "+
				"CHECKLY_API_URL environment variable.",
		)
	}

	if config.AccountID.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("account_id"),
			"Unknown Checkly Account ID",
			"The provider cannot create the Checkly API client as there is "+
				"an unknown configuration value for the Checkly Account ID. "+
				"Either target apply the source of the value first, set the "+
				"value statically in the configuration, or use the "+
				"CHECKLY_API_URL environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	apiKey := os.Getenv("CHECKLY_API_KEY")
	apiURL := os.Getenv("CHECKLY_API_URL")
	accountID := os.Getenv("CHECKLY_ACCOUNT_ID")

	if !config.APIKey.IsNull() {
		apiKey = config.APIKey.ValueString()
	}

	if !config.APIURL.IsNull() {
		apiURL = config.APIURL.ValueString()
	}

	if !config.AccountID.IsNull() {
		accountID = config.AccountID.ValueString()
	}

	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing Checkly API Key",
			"The provider cannot create the Checkly API client as there is "+
				"a missing or empty value for the Checkly API Key. "+
				"Set the value in the configuration or use the "+
				"CHECKLY_API_KEY environment variable. If either is already "+
				"set, ensure the value is not empty.",
		)
	}

	if apiURL == "" {
		apiURL = defaultAPIURL
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "checkly_api_key", apiKey)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "checkly_api_key")
	ctx = tflog.SetField(ctx, "checkly_api_url", apiURL)
	ctx = tflog.SetField(ctx, "checkly_account_id", accountID)

	tflog.Debug(ctx, "Creating Checkly client")

	debugLog := os.Getenv("CHECKLY_DEBUG_LOG")
	var debugOutput io.Writer
	if debugLog != "" {
		debugFile, err := os.OpenFile(debugLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Accessing Checkly Debug Log File",
				fmt.Sprintf("Could not open debug log file, unexpected error: %s", err),
			)

			return
		}

		debugOutput = debugFile
	}

	client := checkly.NewClient(
		apiURL,
		apiKey,
		nil,
		debugOutput,
	)

	if accountID != "" {
		client.SetAccountId(accountID)
	}

	apiSource := os.Getenv("CHECKLY_API_SOURCE")
	if apiSource != "" {
		client.SetChecklySource(apiSource)
	} else {
		client.SetChecklySource("TF")
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func ClientFromProviderData(providerData any) (checkly.Client, diag.Diagnostics) {
	if providerData == nil {
		return nil, diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Missing Configure Type",
				"Expected checkly.Client, got nil. Please report this issue "+
					"to the provider developers.",
			),
		}
	}

	client, ok := providerData.(checkly.Client)
	if !ok {
		return nil, diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Unexpected Configure Type",
				fmt.Sprintf("Expected checkly.Client, got: %T. Please report "+
					"this issue to the provider developers.", providerData),
			),
		}
	}

	return client, nil
}

func (p *ChecklyProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAlertChannelResource,
		NewCheckGroupResource,
		NewCheckResource,
		NewDashboardResource,
		NewEnvironmentVariableResource,
		NewHeartbeatResource,
		NewMaintenanceWindowsResource,
		NewPrivateLocationResource,
		NewSnippetResource,
		NewTriggerCheckResource,
		NewTriggerGroupResource,
	}
}

func (p *ChecklyProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewStaticIPsDataSource,
	}
}

func (p *ChecklyProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ChecklyProvider{
			version: version,
		}
	}
}
