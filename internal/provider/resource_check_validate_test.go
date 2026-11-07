package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// objectWith returns a value of the object type in which only the given
// attributes are set.
func objectWith(objectType tftypes.Object, values map[string]tftypes.Value) tftypes.Value {
	attrs := map[string]tftypes.Value{}
	for name, attrType := range objectType.AttributeTypes {
		attrs[name] = tftypes.NewValue(attrType, nil)
	}
	for name, value := range values {
		attrs[name] = value
	}

	return tftypes.NewValue(objectType, attrs)
}

func TestCheckValidateConfigEnvironmentVariables(t *testing.T) {
	ctx := context.Background()

	var schemaResp resource.SchemaResponse
	NewCheckResource().Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	objectType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	listType := objectType.AttributeTypes[environmentVariableAttributeName].(tftypes.List)

	variables := tftypes.NewValue(listType, []tftypes.Value{
		objectWith(listType.ElementType.(tftypes.Object), map[string]tftypes.Value{
			"key":   tftypes.NewValue(tftypes.String, "A"),
			"value": tftypes.NewValue(tftypes.String, "b"),
		}),
	})

	for checkType, wantWarning := range map[string]bool{"API": true, "BROWSER": false} {
		t.Run(checkType, func(t *testing.T) {
			config := tfsdk.Config{Schema: schemaResp.Schema, Raw: objectWith(objectType, map[string]tftypes.Value{
				"type":                           tftypes.NewValue(tftypes.String, checkType),
				environmentVariableAttributeName: variables,
			})}

			var resp resource.ValidateConfigResponse
			r := NewCheckResource().(resource.ResourceWithValidateConfig)
			r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: config}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			if got := resp.Diagnostics.WarningsCount() > 0; got != wantWarning {
				t.Errorf("warning reported = %v, want %v: %v", got, wantWarning, resp.Diagnostics)
			}
		})
	}
}
