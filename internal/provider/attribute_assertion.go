package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

// assertionModel describes the data model of an assertion. Checks,
// monitors and API check defaults share this model; each resource defines
// its own schema and validators for it.
type assertionModel struct {
	Source     types.String `tfsdk:"source"`
	Property   types.String `tfsdk:"property"`
	Comparison types.String `tfsdk:"comparison"`
	Target     types.String `tfsdk:"target"`
}

var assertionAttrTypes = map[string]attr.Type{
	"source":     types.StringType,
	"property":   types.StringType,
	"comparison": types.StringType,
	"target":     types.StringType,
}

// assertionsWithPropertyDefault returns the assertion set value with an
// empty string substituted for a null or unknown property.
func assertionsWithPropertyDefault(ctx context.Context, set types.Set, diags *diag.Diagnostics) types.Set {
	return assertionsWithDefaults(ctx, set, false, diags)
}

// assertionsWithPropertyAndTargetDefaults returns the assertion set value
// with empty strings substituted for a null or unknown property and target.
func assertionsWithPropertyAndTargetDefaults(ctx context.Context, set types.Set, diags *diag.Diagnostics) types.Set {
	return assertionsWithDefaults(ctx, set, true, diags)
}

// objectWithSetDefaults returns a function that applies withDefaults to the
// set attribute called name of an object value, for use with
// planNestedDefaults on a single nested attribute holding a set of
// assertions or similar elements. Null and unknown objects are returned
// unchanged.
func objectWithSetDefaults(name string, withDefaults func(context.Context, types.Set, *diag.Diagnostics) types.Set) func(context.Context, types.Object, *diag.Diagnostics) types.Object {
	return func(ctx context.Context, obj types.Object, diags *diag.Diagnostics) types.Object {
		if obj.IsNull() || obj.IsUnknown() {
			return obj
		}

		attrs := obj.Attributes()
		set, ok := attrs[name].(types.Set)
		if !ok {
			diags.AddError("Unexpected Attribute Type", fmt.Sprintf("Attribute %q is not a set: %T.", name, attrs[name]))
			return obj
		}
		attrs[name] = withDefaults(ctx, set, diags)

		result, d := types.ObjectValue(obj.AttributeTypes(ctx), attrs)
		diags.Append(d...)

		return result
	}
}

// assertionsWithDefaults applies the default values of the optional+computed
// assertion attributes during plan modification, where schema defaults
// cannot be used because the attributes are nested in set elements. The
// target attribute is only defaulted when defaultTarget is set, since it is
// not optional+computed in every schema.
func assertionsWithDefaults(ctx context.Context, set types.Set, defaultTarget bool, diags *diag.Diagnostics) types.Set {
	if set.IsNull() || set.IsUnknown() {
		return set
	}

	var models []assertionModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return set
	}

	for i, model := range models {
		models[i].Property = stringOrDefault(model.Property, "")
		if defaultTarget {
			models[i].Target = stringOrDefault(model.Target, "")
		}
	}

	result, d := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: assertionAttrTypes}, models)
	diags.Append(d...)

	return result
}

// assertionsFromSet converts an assertion set value to its API
// representation. Null and unknown values convert to an empty (non-nil)
// slice.
func assertionsFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) []checkly.Assertion {
	assertions := []checkly.Assertion{}

	if set.IsNull() || set.IsUnknown() {
		return assertions
	}

	var models []assertionModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return assertions
	}

	for _, model := range models {
		assertions = append(assertions, checkly.Assertion{
			Source:     checkly.AssertionSource(model.Source.ValueString()),
			Property:   model.Property.ValueString(),
			Comparison: checkly.AssertionComparison(model.Comparison.ValueString()),
			Target:     model.Target.ValueString(),
		})
	}

	return assertions
}

// assertionsToSet converts the API representation of assertions to an
// assertion set value.
func assertionsToSet(ctx context.Context, assertions []checkly.Assertion, diags *diag.Diagnostics) types.Set {
	models := make([]assertionModel, len(assertions))
	for i, assertion := range assertions {
		models[i] = assertionModel{
			Source:     types.StringValue(string(assertion.Source)),
			Property:   types.StringValue(assertion.Property),
			Comparison: types.StringValue(string(assertion.Comparison)),
			Target:     types.StringValue(assertion.Target),
		}
	}

	set, d := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: assertionAttrTypes}, models)
	diags.Append(d...)

	return set
}
