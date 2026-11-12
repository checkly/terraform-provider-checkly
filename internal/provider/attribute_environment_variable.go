package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

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
// environment_variables attribute. Leaving it out stores an empty list.
//
// The validators enforce the API's limits on the number of variables and the
// length of their values. The API doesn't reject a check or check group
// variable that exceeds them: it drops the variable, so the apply would fail
// with a result inconsistent with the plan.
func makeEnvironmentVariableSchema(options EnvironmentVariableAttributeSchemaOptions) schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Optional:    true,
		Computed:    true,
		Default:     emptyListDefault(types.ObjectType{AttrTypes: environmentVariableAttrTypes}),
		Description: fmt.Sprintf("%s At most %d variables.", options.Description, maxEnvironmentVariables),
		Validators: []validator.List{
			listvalidator.SizeAtMost(maxEnvironmentVariables),
		},
		PlanModifiers: []planmodifier.List{
			secretEnvironmentVariableValues{},
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

// secretEnvironmentVariableValues checks the length of the secret values in
// environment_variables. When a check or check group is written, the API
// checks a secret value only when it is new or changed: an element whose key
// was already a secret with the same value in the prior state keeps its
// stored value and is not checked again. A short value is an error when it is
// sent, and a warning when the API keeps it. Elements with an unknown key,
// value or secret flag are skipped.
type secretEnvironmentVariableValues struct{}

var _ planmodifier.List = secretEnvironmentVariableValues{}

func (m secretEnvironmentVariableValues) Description(ctx context.Context) string {
	return fmt.Sprintf("A new or changed secret value must have at least %d characters.", minSecretEnvironmentVariableValueCharacters)
}

func (m secretEnvironmentVariableValues) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m secretEnvironmentVariableValues) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}

	// The secret values in the prior state, by key.
	stored := map[string]string{}
	if !req.StateValue.IsNull() && !req.StateValue.IsUnknown() {
		var models []EnvironmentVariableModel
		if diags := req.StateValue.ElementsAs(ctx, &models, false); !diags.HasError() {
			for _, model := range models {
				if model.Secret.ValueBool() {
					stored[model.Key.ValueString()] = model.Value.ValueString()
				}
			}
		}
	}

	for i, element := range req.PlanValue.Elements() {
		object, ok := element.(types.Object)
		if !ok || object.IsNull() || object.IsUnknown() {
			continue
		}
		var model EnvironmentVariableModel
		if diags := object.As(ctx, &model, basetypes.ObjectAsOptions{}); diags.HasError() {
			continue
		}
		if model.Key.IsUnknown() || model.Value.IsUnknown() || model.Secret.IsUnknown() || !model.Secret.ValueBool() {
			continue
		}

		storedValue, wasSecret := stored[model.Key.ValueString()]
		sent := !wasSecret || storedValue != model.Value.ValueString()
		checkSecretEnvironmentVariableValue(&resp.Diagnostics, req.Path.AtListIndex(i).AtName("value"), model.Value.ValueString(), sent)
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
// variables to the environment_variables attribute value. No variables
// convert to an empty list.
//
// The Checkly API does not return the value of secret environment variables.
// For those, the value from prior is preserved, matched by key, so that the
// resulting value stays consistent with the plan.
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
