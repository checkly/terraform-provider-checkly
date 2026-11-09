package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestObjectWithSetDefaults(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	assertionType := types.ObjectType{AttrTypes: assertionAttrTypes}
	attrTypes := map[string]attr.Type{
		"url":        types.StringType,
		"assertions": types.SetType{ElemType: assertionType},
	}
	withDefaults := objectWithSetDefaults("assertions", assertionsWithPropertyAndTargetDefaults)

	t.Run("null and unknown objects are unchanged", func(t *testing.T) {
		var diags diag.Diagnostics
		for _, obj := range []types.Object{types.ObjectNull(attrTypes), types.ObjectUnknown(attrTypes)} {
			if got := withDefaults(ctx, obj, &diags); !got.Equal(obj) {
				t.Errorf("got %s, want %s", got, obj)
			}
		}
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
	})

	t.Run("defaults are applied to the named set only", func(t *testing.T) {
		assertion := func(property, target attr.Value) attr.Value {
			return types.ObjectValueMust(assertionAttrTypes, map[string]attr.Value{
				"source":     types.StringValue("STATUS_CODE"),
				"property":   property,
				"comparison": types.StringValue("EQUALS"),
				"target":     target,
			})
		}
		obj := types.ObjectValueMust(attrTypes, map[string]attr.Value{
			"url":        types.StringValue("https://example.com"),
			"assertions": types.SetValueMust(assertionType, []attr.Value{assertion(types.StringNull(), types.StringUnknown())}),
		})
		want := types.ObjectValueMust(attrTypes, map[string]attr.Value{
			"url":        types.StringValue("https://example.com"),
			"assertions": types.SetValueMust(assertionType, []attr.Value{assertion(types.StringValue(""), types.StringValue(""))}),
		})

		var diags diag.Diagnostics
		got := withDefaults(ctx, obj, &diags)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if !got.Equal(want) {
			t.Errorf("got %s, want %s", got, want)
		}
	})
}
