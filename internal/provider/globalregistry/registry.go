package globalregistry

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/checkly/terraform-provider-checkly/internal/provider"
	"github.com/checkly/terraform-provider-checkly/internal/provider/datasources"
	"github.com/checkly/terraform-provider-checkly/internal/provider/resources"
)

var (
	_ provider.Registry = (*registry)(nil)
)

type registry struct {
	resources   []func() resource.Resource
	dataSources []func() datasource.DataSource
	functions   []func() function.Function
}

func (r *registry) Resources(ctx context.Context) []func() resource.Resource {
	return r.resources
}

func (r *registry) DataSources(ctx context.Context) []func() datasource.DataSource {
	return r.dataSources
}

func (r *registry) Functions(ctx context.Context) []func() function.Function {
	return r.functions
}

func (r *registry) RegisterResource(factory ...func() resource.Resource) {
	r.resources = append(r.resources, factory...)
}

func (r *registry) RegisterDataSource(factory ...func() datasource.DataSource) {
	r.dataSources = append(r.dataSources, factory...)
}

func (r *registry) RegisterFunction(factory ...func() function.Function) {
	r.functions = append(r.functions, factory...)
}

var Registry = new(registry)

func init() {
	Registry.RegisterResource(
		resources.NewAlertChannelResource,
		resources.NewCheckGroupResource,
		resources.NewCheckResource,
		resources.NewDashboardResource,
		resources.NewEnvironmentVariableResource,
		resources.NewHeartbeatResource,
		resources.NewMaintenanceWindowsResource,
		resources.NewPrivateLocationResource,
		resources.NewSnippetResource,
		resources.NewTriggerCheckResource,
		resources.NewTriggerGroupResource,
	)

	Registry.RegisterDataSource(
		datasources.NewStaticIPsDataSource,
	)
}
