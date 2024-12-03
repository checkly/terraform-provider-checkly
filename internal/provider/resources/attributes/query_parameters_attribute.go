package attributes

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/checkly/terraform-provider-checkly/internal/provider/interop"
	"github.com/checkly/terraform-provider-checkly/internal/sdkutil"
)

var (
	_ interop.Model[[]checkly.KeyValue] = (*QueryParametersAttributeModel)(nil)
)

var QueryParametersAttributeSchema = schema.MapAttribute{
	ElementType: types.StringType,
	Optional:    true,
	Computed:    true, // TODO: Really?
}

type QueryParametersAttributeModel types.Map

func (m *QueryParametersAttributeModel) Refresh(ctx context.Context, from *[]checkly.KeyValue, flags interop.RefreshFlags) diag.Diagnostics {
	*m = QueryParametersAttributeModel(sdkutil.KeyValuesIntoMap(from))

	return nil
}

func (m *QueryParametersAttributeModel) Render(ctx context.Context, into *[]checkly.KeyValue) diag.Diagnostics {
	*into = sdkutil.KeyValuesFromMap(types.Map(*m))

	return nil
}
