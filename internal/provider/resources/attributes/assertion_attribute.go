package attributes

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/checkly/terraform-provider-checkly/internal/provider/interop"
)

var (
	_ interop.Model[checkly.Assertion] = (*AssertionAttributeModel)(nil)
)

var AssertionAttributeSchema = schema.ListNestedAttribute{
	Optional: true,
	NestedObject: schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"source": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						"STATUS_CODE",
						"JSON_BODY",
						"HEADERS",
						"TEXT_BODY",
						"RESPONSE_TIME",
					),
				},
				Description: "The source of the asserted value. Possible values `STATUS_CODE`, `JSON_BODY`, `HEADERS`, `TEXT_BODY`, and `RESPONSE_TIME`.",
			},
			"property": schema.StringAttribute{
				Optional: true,
			},
			"comparison": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						"CONTAINS",
						"EQUALS",
						"GREATER_THAN",
						"HAS_KEY",
						"HAS_VALUE",
						"IS_EMPTY",
						"IS_NULL",
						"LESS_THAN",
						"NOT_CONTAINS",
						"NOT_EMPTY",
						"NOT_EQUALS",
						"NOT_HAS_KEY",
						"NOT_HAS_VALUE",
						"NOT_NULL",
					),
				},
				Description: "The type of comparison to be executed between expected and actual value of the assertion. Possible values `EQUALS`, `NOT_EQUALS`, `HAS_KEY`, `NOT_HAS_KEY`, `HAS_VALUE`, `NOT_HAS_VALUE`, `IS_EMPTY`, `NOT_EMPTY`, `GREATER_THAN`, `LESS_THAN`, `CONTAINS`, `NOT_CONTAINS`, `IS_NULL`, and `NOT_NULL`.",
			},
			"target": schema.StringAttribute{
				Required: true,
			},
		},
	},
}

type AssertionAttributeModel struct {
	Source     types.String `tfsdk:"source"`
	Property   types.String `tfsdk:"property"`
	Comparison types.String `tfsdk:"comparison"`
	Target     types.String `tfsdk:"target"`
}

func (m *AssertionAttributeModel) Refresh(ctx context.Context, from *checkly.Assertion, flags interop.RefreshFlags) diag.Diagnostics {
	m.Source = types.StringValue(from.Source)
	m.Property = types.StringValue(from.Property)
	m.Comparison = types.StringValue(from.Comparison)
	m.Target = types.StringValue(from.Target)

	return nil
}

func (m *AssertionAttributeModel) Render(ctx context.Context, into *checkly.Assertion) diag.Diagnostics {
	into.Source = m.Source.ValueString()
	into.Property = m.Property.ValueString()
	into.Comparison = m.Comparison.ValueString()
	into.Target = m.Target.ValueString()

	return nil
}
