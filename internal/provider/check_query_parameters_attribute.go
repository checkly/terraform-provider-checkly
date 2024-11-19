package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
)

var (
	_ ResourceModel[[]checkly.KeyValue] = (*CheckQueryParametersAttributeModel)(nil)
)

var CheckQueryParametersAttributeSchema = schema.MapAttribute{
	ElementType: types.StringType,
	Optional:    true,
	Computed:    true, // TODO: Really?
}

type CheckQueryParametersAttributeModel types.Map

func (m *CheckQueryParametersAttributeModel) Refresh(ctx context.Context, from *[]checkly.KeyValue, flags RefreshFlags) diag.Diagnostics {
	*m = CheckQueryParametersAttributeModel(SDKKeyValuesIntoMap(from))

	return nil
}

func (m *CheckQueryParametersAttributeModel) Render(ctx context.Context, into *[]checkly.KeyValue) diag.Diagnostics {
	*into = SDKKeyValuesFromMap(types.Map(*m))

	return nil
}
