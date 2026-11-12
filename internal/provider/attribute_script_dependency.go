package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/checkly/checkly-go-sdk/v2"
)

const scriptDependenciesAttributeName = "dependencies"

// scriptDependencyModel describes an element of the dependencies attribute of
// browser and multistep checks.
type scriptDependencyModel struct {
	Path    types.String `tfsdk:"path"`
	Content types.String `tfsdk:"content"`
}

var scriptDependencyAttrTypes = map[string]attr.Type{
	"path":    types.StringType,
	"content": types.StringType,
}

// makeScriptDependenciesSchema returns the schema for the dependencies
// attribute of browser and multistep checks. Leaving it out stores an empty
// list.
func makeScriptDependenciesSchema() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Optional:    true,
		Computed:    true,
		Default:     emptyListDefault(types.ObjectType{AttrTypes: scriptDependencyAttrTypes}),
		Description: "Script dependency files for editor-style multi-file scripts. Use together with `script_path` to mirror a multi-file layout. The Checkly API does not preserve the configured order, so after an import the first plan can show the dependencies reordered to match the configuration; applying it once settles the order.",
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"path": schema.StringAttribute{
					Required:    true,
					Description: "The path of the dependency file (1 to 1000 characters).",
					Validators: []validator.String{
						utf16LengthBetween(1, 1000),
					},
				},
				"content": schema.StringAttribute{
					Required:    true,
					Description: "The content of the dependency file.",
				},
			},
		},
	}
}

// scriptDependenciesFromList converts the dependencies attribute value to its
// API representation. Null, unknown and empty values convert to nil.
func scriptDependenciesFromList(ctx context.Context, list types.List, diags *diag.Diagnostics) []checkly.ScriptDependency {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}

	var models []scriptDependencyModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	if diags.HasError() || len(models) == 0 {
		return nil
	}

	deps := make([]checkly.ScriptDependency, len(models))
	for i, model := range models {
		deps[i] = checkly.ScriptDependency{
			Path:    model.Path.ValueString(),
			Content: model.Content.ValueString(),
		}
	}

	return deps
}

// scriptDependenciesToList converts the API representation of script
// dependencies to the dependencies attribute value. The API does not preserve
// the configured order of dependencies, so the prior list's order (matched by
// path) is kept where possible; dependencies without a prior entry are
// appended in API order. No dependencies convert to an empty list.
func scriptDependenciesToList(ctx context.Context, deps []checkly.ScriptDependency, prior types.List, diags *diag.Diagnostics) types.List {
	elementType := types.ObjectType{AttrTypes: scriptDependencyAttrTypes}

	byPath := make(map[string]checkly.ScriptDependency, len(deps))
	for _, dep := range deps {
		byPath[dep.Path] = dep
	}

	models := make([]scriptDependencyModel, 0, len(deps))
	used := make(map[string]bool, len(deps))

	if !prior.IsNull() && !prior.IsUnknown() {
		var priorModels []scriptDependencyModel
		diags.Append(prior.ElementsAs(ctx, &priorModels, false)...)
		if diags.HasError() {
			return types.ListNull(elementType)
		}

		for _, priorModel := range priorModels {
			path := priorModel.Path.ValueString()
			if dep, ok := byPath[path]; ok && !used[path] {
				models = append(models, scriptDependencyModel{
					Path:    types.StringValue(dep.Path),
					Content: types.StringValue(dep.Content),
				})
				used[path] = true
			}
		}
	}

	for _, dep := range deps {
		if !used[dep.Path] {
			models = append(models, scriptDependencyModel{
				Path:    types.StringValue(dep.Path),
				Content: types.StringValue(dep.Content),
			})
		}
	}

	list, d := types.ListValueFrom(ctx, elementType, models)
	diags.Append(d...)

	return list
}
