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
	_ interop.Model[[]checkly.KeyValue] = (*HeadersAttributeModel)(nil)
)

var HeadersAttributeSchema = schema.MapAttribute{
	ElementType: types.StringType,
	Optional:    true,
	Computed:    true, // TODO: Really?
}

type HeadersAttributeModel types.Map

func (m *HeadersAttributeModel) Refresh(ctx context.Context, from *[]checkly.KeyValue, flags interop.RefreshFlags) diag.Diagnostics {
	*m = HeadersAttributeModel(sdkutil.KeyValuesIntoMap(from))

	return nil
}

func (m *HeadersAttributeModel) Render(ctx context.Context, into *[]checkly.KeyValue) diag.Diagnostics {
	*into = sdkutil.KeyValuesFromMap(types.Map(*m))

	return nil
}
