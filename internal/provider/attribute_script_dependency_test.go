package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The API reports no dependencies as an empty array, which converts to an
// empty list whatever the prior value.
func TestScriptDependenciesToListEmpty(t *testing.T) {
	ctx := context.Background()
	elemType := types.ObjectType{AttrTypes: scriptDependencyAttrTypes}
	empty := types.ListValueMust(elemType, nil)
	nonEmpty := types.ListValueMust(elemType, []attr.Value{types.ObjectValueMust(scriptDependencyAttrTypes, map[string]attr.Value{
		"path":    types.StringValue("util.ts"),
		"content": types.StringValue("export {}"),
	})})

	for name, c := range map[string]struct {
		prior types.List
		want  types.List
	}{
		"prior null":      {prior: types.ListNull(elemType), want: empty},
		"prior unknown":   {prior: types.ListUnknown(elemType), want: empty},
		"prior empty":     {prior: empty, want: empty},
		"prior non-empty": {prior: nonEmpty, want: empty},
	} {
		t.Run(name, func(t *testing.T) {
			var diags diag.Diagnostics
			got := scriptDependenciesToList(ctx, nil, c.prior, &diags)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if !got.Equal(c.want) {
				t.Errorf("got %s, want %s", got, c.want)
			}
		})
	}
}
