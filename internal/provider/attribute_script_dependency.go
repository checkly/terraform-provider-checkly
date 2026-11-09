package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
// attribute of browser and multistep checks. It is optional without a
// default, so leaving it out stores null; see scriptDependenciesToList.
func makeScriptDependenciesSchema() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Optional:    true,
		Description: "Script dependency files for editor-style multi-file scripts. Use together with `script_path` to mirror a multi-file layout.",
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"path": schema.StringAttribute{
					Required:    true,
					Description: "The path of the dependency file.",
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
// appended in API order. The API reports no dependencies as an empty array,
// so no dependencies convert to null when the prior value is null or unknown,
// and to an empty list when it is known.
func scriptDependenciesToList(ctx context.Context, deps []checkly.ScriptDependency, prior types.List, diags *diag.Diagnostics) types.List {
	elementType := types.ObjectType{AttrTypes: scriptDependencyAttrTypes}

	if len(deps) == 0 && (prior.IsNull() || prior.IsUnknown()) {
		return types.ListNull(elementType)
	}

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
