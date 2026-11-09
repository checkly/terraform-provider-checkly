package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

const environmentVariableAttributeName = "environment_variables"

// EnvironmentVariableModel describes an element of the environment_variables
// attribute.
type EnvironmentVariableModel struct {
	Key    types.String `tfsdk:"key"`
	Value  types.String `tfsdk:"value"`
	Locked types.Bool   `tfsdk:"locked"`
	Secret types.Bool   `tfsdk:"secret"`
}

var environmentVariableAttrTypes = map[string]attr.Type{
	"key":    types.StringType,
	"value":  types.StringType,
	"locked": types.BoolType,
	"secret": types.BoolType,
}

// EnvironmentVariableAttributeSchemaOptions customizes the
// environment_variables attribute schema for a specific resource.
type EnvironmentVariableAttributeSchemaOptions struct {
	// Description overrides the default attribute description.
	Description string
}

// makeEnvironmentVariableSchema returns the schema for the
// environment_variables attribute. It is optional without a default, so
// leaving it out stores null; see environmentVariablesToList.
//
// The validators enforce the API's limits on the number of variables and the
// length of their values. The API doesn't reject a check or check group
// variable that exceeds them: it drops the variable, so the apply would fail
// with a result inconsistent with the plan.
func makeEnvironmentVariableSchema(options EnvironmentVariableAttributeSchemaOptions) schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Optional:    true,
		Description: fmt.Sprintf("%s At most %d variables.", options.Description, maxEnvironmentVariables),
		Validators: []validator.List{
			listvalidator.SizeAtMost(maxEnvironmentVariables),
		},
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"key": schema.StringAttribute{
					Required:    true,
					Description: "The name of the environment variable or secret.",
				},
				"value": schema.StringAttribute{
					Required:    true,
					Sensitive:   true,
					Description: "The value of the environment variable or secret. " + environmentVariableValueDescription,
					Validators: []validator.String{
						environmentVariableValueValidator{},
					},
				},
				"locked": schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(false),
					Description: "If true, the value is not shown by default, but it can be accessed. (Default `false`).",
				},
				"secret": schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(false),
					Description: "If true, the value will never be visible. The API never returns secret values, so Terraform keeps the last applied value and does not detect changes made outside Terraform. (Default `false`).",
				},
			},
		},
	}
}

// environmentVariablesFromList converts the environment_variables attribute
// value to its API representation. Null converts to no variables, which
// clears them in Checkly.
func environmentVariablesFromList(ctx context.Context, list types.List, diags *diag.Diagnostics) []checkly.EnvironmentVariable {
	res := []checkly.EnvironmentVariable{}

	if list.IsNull() || list.IsUnknown() {
		return res
	}

	var models []EnvironmentVariableModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return res
	}

	for _, model := range models {
		res = append(res, checkly.EnvironmentVariable{
			Key:    model.Key.ValueString(),
			Value:  model.Value.ValueString(),
			Locked: model.Locked.ValueBool(),
			Secret: model.Secret.ValueBool(),
		})
	}

	return res
}

// environmentVariablesToList converts the API representation of environment
// variables to the environment_variables attribute value. The API reports no
// variables as an empty array, so no variables convert to null when prior
// (the planned or previously stored value) is null or unknown, and to an
// empty list when it is known.
//
// The Checkly API does not return the value of secret environment variables.
// For those, the value from prior is preserved, matched by key, so that the
// resulting value stays consistent with the plan.
func environmentVariablesToList(ctx context.Context, evs []checkly.EnvironmentVariable, prior types.List, diags *diag.Diagnostics) types.List {
	if len(evs) == 0 && (prior.IsNull() || prior.IsUnknown()) {
		return types.ListNull(types.ObjectType{AttrTypes: environmentVariableAttrTypes})
	}

	priorValues := map[string]types.String{}

	if !prior.IsNull() && !prior.IsUnknown() {
		var priorModels []EnvironmentVariableModel
		diags.Append(prior.ElementsAs(ctx, &priorModels, false)...)
		if diags.HasError() {
			return types.ListNull(types.ObjectType{AttrTypes: environmentVariableAttrTypes})
		}

		for _, model := range priorModels {
			priorValues[model.Key.ValueString()] = model.Value
		}
	}

	models := make([]EnvironmentVariableModel, len(evs))
	for i, ev := range evs {
		value := types.StringValue(ev.Value)
		if ev.Secret {
			if priorValue, ok := priorValues[ev.Key]; ok && !priorValue.IsUnknown() {
				value = priorValue
			}
		}

		models[i] = EnvironmentVariableModel{
			Key:    types.StringValue(ev.Key),
			Value:  value,
			Locked: types.BoolValue(ev.Locked),
			Secret: types.BoolValue(ev.Secret),
		}
	}

	list, d := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: environmentVariableAttrTypes}, models)
	diags.Append(d...)

	return list
}
