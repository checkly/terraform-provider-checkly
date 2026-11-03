package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
)

var _ provider.Provider = (*ChecklyProvider)(nil)

// ChecklyProvider defines the provider implementation.
type ChecklyProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// New returns a new provider factory for the given version.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ChecklyProvider{
			version: version,
		}
	}
}

// ChecklyProviderModel describes the provider configuration data model.
type ChecklyProviderModel struct {
	APIKey    types.String `tfsdk:"api_key"`
	APIURL    types.String `tfsdk:"api_url"`
	AccountID types.String `tfsdk:"account_id"`
}

func (p *ChecklyProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "checkly"
	resp.Version = p.version
}

func (p *ChecklyProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The Checkly API key. Can also be set using the `CHECKLY_API_KEY` environment variable.",
			},
			"api_url": schema.StringAttribute{
				Optional:    true,
				Description: "The Checkly API URL. Defaults to `https://api.checklyhq.com`. Can also be set using the `CHECKLY_API_URL` environment variable.",
			},
			"account_id": schema.StringAttribute{
				Optional:    true,
				Description: "The Checkly account ID. Required when using a user-scoped API key. Can also be set using the `CHECKLY_ACCOUNT_ID` environment variable.",
			},
		},
	}
}

func (p *ChecklyProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config ChecklyProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.APIKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Unknown Checkly API Key",
			"The provider cannot create the Checkly API client as there is an unknown configuration value for the API key. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the CHECKLY_API_KEY environment variable.",
		)
	}

	if config.APIURL.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_url"),
			"Unknown Checkly API URL",
			"The provider cannot create the Checkly API client as there is an unknown configuration value for the API URL.",
		)
	}

	if config.AccountID.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("account_id"),
			"Unknown Checkly Account ID",
			"The provider cannot create the Checkly API client as there is an unknown configuration value for the account ID.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	apiKey := os.Getenv("CHECKLY_API_KEY")
	if !config.APIKey.IsNull() {
		apiKey = config.APIKey.ValueString()
	}

	apiURL := os.Getenv("CHECKLY_API_URL")
	if !config.APIURL.IsNull() {
		apiURL = config.APIURL.ValueString()
	}

	accountID := os.Getenv("CHECKLY_ACCOUNT_ID")
	if !config.AccountID.IsNull() {
		accountID = config.AccountID.ValueString()
	}

	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing Checkly API Key",
			"The provider requires a Checkly API key. Set the api_key value in the provider configuration or use the CHECKLY_API_KEY environment variable.",
		)
		return
	}

	if apiURL == "" {
		apiURL = "https://api.checklyhq.com"
	}

	var debugOutput io.Writer
	if debugLog := os.Getenv("CHECKLY_DEBUG_LOG"); debugLog != "" {
		debugFile, err := os.OpenFile(debugLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Open Debug Log File",
				fmt.Sprintf("The CHECKLY_DEBUG_LOG environment variable is set, but the file cannot be opened for writing: %s", err),
			)
			return
		}
		debugOutput = debugFile
	}

	client := checkly.NewClient(
		apiURL,
		apiKey,
		&http.Client{Timeout: apiRequestTimeout()},
		debugOutput,
	)

	if accountID != "" {
		client.SetAccountId(accountID)
	}

	if source := os.Getenv("CHECKLY_API_SOURCE"); source != "" {
		client.SetChecklySource(source)
	} else {
		client.SetChecklySource("TF")
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *ChecklyProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAlertChannelResource,
		NewAPICheckResource,
		NewBrowserCheckResource,
		NewCheckResource,
		NewCheckGroupResource,
		NewCheckGroupV2Resource,
		NewClientCertificateResource,
		NewDashboardResource,
		NewDNSMonitorResource,
		NewEnvironmentVariableResource,
		NewGRPCMonitorResource,
		NewHeartbeatResource,
		NewHeartbeatMonitorResource,
		NewICMPMonitorResource,
		NewMaintenanceWindowResource,
		NewMaintenanceWindowsResource,
		NewMultistepCheckResource,
		NewPlaywrightCheckSuiteResource,
		NewPlaywrightCodeBundleResource,
		NewPrivateLocationResource,
		NewSnippetResource,
		NewSSLMonitorResource,
		NewStatusPageResource,
		NewStatusPageServiceResource,
		NewStatusPageV3Resource,
		NewStatusPageV3ComponentResource,
		NewStatusPageV3AutomationRuleResource,
		NewTCPCheckResource,
		NewTCPMonitorResource,
		NewTracerouteMonitorResource,
		NewURLMonitorResource,
	}
}

func (p *ChecklyProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewStaticIPsDataSource,
	}
}
