package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

const environmentVariableAttributeName = "environment_variable"

// EnvironmentVariableModel describes the environment_variable block data
// model.
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
// environment_variable block schema for a specific resource.
type EnvironmentVariableAttributeSchemaOptions struct {
	// Description overrides the default block description.
	Description string
}

// makeEnvironmentVariableSchema returns the schema for the
// environment_variable block.
func makeEnvironmentVariableSchema(options EnvironmentVariableAttributeSchemaOptions) schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: options.Description,
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"key": schema.StringAttribute{
					Required:    true,
					Description: "The name of the environment variable or secret.",
				},
				"value": schema.StringAttribute{
					Required:    true,
					Sensitive:   true,
					Description: "The value of the environment variable or secret.",
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

// environmentVariablesFromList converts the environment_variable block value
// to its API representation.
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
// variables to the environment_variable block value.
//
// The Checkly API does not return the value of secret environment variables.
// For those, the value from prior (the planned or previously stored block
// value) is preserved, matched by key, so that the resulting value stays
// consistent with the plan.
func environmentVariablesToList(ctx context.Context, evs []checkly.EnvironmentVariable, prior types.List, diags *diag.Diagnostics) types.List {
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
