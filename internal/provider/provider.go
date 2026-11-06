package provider

import (
	"context"
	"errors"
	"net/http"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
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
				Description: "The Checkly account ID, which every API key needs. Required, but can also be set using the `CHECKLY_ACCOUNT_ID` environment variable.",
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

	apiKey := configuredOrEnv(config.APIKey, "CHECKLY_API_KEY")
	apiURL := configuredOrEnv(config.APIURL, "CHECKLY_API_URL")
	accountID := configuredOrEnv(config.AccountID, "CHECKLY_ACCOUNT_ID")

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

	requestTimeout, err := parseAPIRequestTimeout()
	if err != nil {
		resp.Diagnostics.AddError("Invalid API Request Timeout", err.Error())
		return
	}

	source := os.Getenv("CHECKLY_API_SOURCE")
	if source == "" {
		source = "TF"
	}

	// The errors of NewClient name the invalid option, never the API key or
	// the account ID.
	client, err := checkly.NewClient(
		checkly.WithBaseURL(apiURL),
		checkly.WithAuth(apiKey, accountID),
		checkly.WithHTTPClient(&http.Client{Timeout: requestTimeout}),
		checkly.WithSource(source),
	)
	switch {
	case errors.Is(err, checkly.ErrMissingAccountID):
		resp.Diagnostics.AddAttributeError(
			path.Root("account_id"),
			"Missing Checkly Account ID",
			"The provider requires a Checkly account ID, for every API key. Set the account_id value in the provider configuration or use the CHECKLY_ACCOUNT_ID environment variable.",
		)
		return
	case errors.Is(err, checkly.ErrInvalidAccountID):
		resp.Diagnostics.AddAttributeError(
			path.Root("account_id"),
			"Invalid Checkly Account ID",
			"The Checkly account ID must be a UUID. Check the account_id value in the provider configuration or the CHECKLY_ACCOUNT_ID environment variable.",
		)
		return
	case err != nil:
		resp.Diagnostics.AddError("Unable to Create the Checkly API Client", err.Error())
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

// configuredOrEnv returns the configured value, or the environment variable
// when the value is null or empty, such as a variable that defaults to "".
func configuredOrEnv(value types.String, env string) string {
	if configured := value.ValueString(); configured != "" {
		return configured
	}

	return os.Getenv(env)
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
