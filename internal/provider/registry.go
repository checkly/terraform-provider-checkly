package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

type Registry interface {
	Resources(ctx context.Context) []func() resource.Resource
	DataSources(ctx context.Context) []func() datasource.DataSource
	Functions(ctx context.Context) []func() function.Function
}
