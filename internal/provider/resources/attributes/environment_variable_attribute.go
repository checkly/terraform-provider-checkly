package attributes

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/checkly/terraform-provider-checkly/internal/provider/interop"
)

var (
	_ interop.Model[checkly.EnvironmentVariable] = (*EnvironmentVariableAttributeModel)(nil)
)

var EnvironmentVariableAttributeSchema = schema.ListNestedAttribute{
	Optional: true,
	Description: "Introduce additional environment variables to the check " +
		"execution environment. Only relevant for browser checks. Prefer " +
		"global environment variables when possible.",
	NestedObject: schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"key": schema.StringAttribute{
				Description: "The name of the environment variable.",
				Required:    true,
			},
			"value": schema.StringAttribute{
				Description: "The value of the environment variable. By " +
					"default the value is plain text and can be seen by any " +
					"team member. It will also be present in check results " +
					"and logs.",
				Required: true,
				// We cannot make the value conditionally sensitive, so it's
				// better to assume everything's sensitive.
				Sensitive: true,
			},
			"locked": schema.BoolAttribute{
				Description: "Locked environment variables are encrypted at " +
					"rest and in flight on the Checkly backend and are only " +
					"decrypted when needed. Their value is hidden by " +
					"default, but can be accessed by team members with the " +
					"appropriate permissions.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				Validators: []validator.Bool{
					boolvalidator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("secret"),
					),
				},
			},
			"secret": schema.BoolAttribute{
				Description: "Secret environment variables are always " +
					"encrypted and their value is never shown to any user. " +
					"However, keep in mind that your Terraform state will " +
					"still contain the value.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				Validators: []validator.Bool{
					boolvalidator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("locked"),
					),
				},
			},
		},
	},
}

type EnvironmentVariableAttributeModel struct {
	Key    types.String `tfsdk:"key"`
	Value  types.String `tfsdk:"value"`
	Locked types.Bool   `tfsdk:"locked"`
	Secret types.Bool   `tfsdk:"secret"`
}

var EnvironmentVariableAttributeGluer = interop.GluerForListNestedAttribute[
	checkly.EnvironmentVariable,
	EnvironmentVariableAttributeModel,
](EnvironmentVariableAttributeSchema)

func (m *EnvironmentVariableAttributeModel) Refresh(ctx context.Context, from *checkly.EnvironmentVariable, flags interop.RefreshFlags) diag.Diagnostics {
	m.Key = types.StringValue(from.Key)
	m.Value = types.StringValue(from.Value)
	m.Locked = types.BoolValue(from.Locked)
	m.Secret = types.BoolValue(from.Secret)

	return nil
}

func (m *EnvironmentVariableAttributeModel) Render(ctx context.Context, into *checkly.EnvironmentVariable) diag.Diagnostics {
	into.Key = m.Key.ValueString()
	into.Value = m.Value.ValueString()
	into.Locked = m.Locked.ValueBool()
	into.Secret = m.Secret.ValueBool()

	return nil
}
