package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/checkly/terraform-provider-checkly/internal/sdkutil"
)

var (
	_ ResourceModel[[]checkly.KeyValue] = (*CheckHeadersAttributeModel)(nil)
)

var CheckHeadersAttributeSchema = schema.MapAttribute{
	ElementType: types.StringType,
	Optional:    true,
	Computed:    true, // TODO: Really?
}

type CheckHeadersAttributeModel types.Map

func (m *CheckHeadersAttributeModel) Refresh(ctx context.Context, from *[]checkly.KeyValue, flags RefreshFlags) diag.Diagnostics {
	*m = CheckHeadersAttributeModel(sdkutil.KeyValuesIntoMap(from))

	return nil
}

func (m *CheckHeadersAttributeModel) Render(ctx context.Context, into *[]checkly.KeyValue) diag.Diagnostics {
	*into = sdkutil.KeyValuesFromMap(types.Map(*m))

	return nil
}
