package provider

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// secretElement is a set element whose values must not show in diagnostics.
var secretElement = types.ObjectValueMust(
	map[string]attr.Type{"key": types.StringType, "value": types.StringType},
	map[string]attr.Value{"key": types.StringValue("authorization"), "value": types.StringValue("Bearer s3cr3t")},
)

func TestDisplayPath(t *testing.T) {
	inner := types.ObjectValueMust(map[string]attr.Type{"name": types.StringType}, map[string]attr.Value{"name": types.StringValue("s3cr3t")})

	for _, c := range []struct {
		path path.Path
		want string
	}{
		{path.Root("name"), "name"},
		{path.Root("request").AtName("metadata").AtSetValue(secretElement).AtName("value"), "request.metadata[*].value"},
		{path.Root("tags").AtSetValue(types.StringValue("s3cr3t")), "tags[*]"},
		{
			path.Root("component_configs").AtSetValue(secretElement).AtName("components").AtSetValue(inner).AtName("name"),
			"component_configs[*].components[*].name",
		},
		{path.Root("locations").AtListIndex(2), "locations[2]"},
		{path.Root("headers").AtMapKey("X-Key").AtName("value"), `headers["X-Key"].value`},
		{path.Root("list").AtListIndex(0).AtName("set").AtSetValue(inner).AtMapKey("k"), `list[0].set[*]["k"]`},
	} {
		got := displayPath(c.path)
		if got != c.want {
			t.Errorf("displayPath(%s) = %q, want %q", c.path, got, c.want)
		}
		if !strings.Contains(c.path.String(), "Value(") && got != c.path.String() {
			t.Errorf("displayPath(%s) = %q, want it unchanged", c.path, got)
		}
	}
}

func TestDisplayPathDiagnostics(t *testing.T) {
	element := path.Root("request").AtName("metadata").AtSetValue(secretElement)
	p := element.AtName("value")
	sibling := element.AtName("key")

	diags := displayPathDiagnostics(diag.Diagnostics{
		diag.NewAttributeErrorDiagnostic(p, "Summary of "+p.String(), fmt.Sprintf("Attribute %s is invalid", p)),
		diag.NewAttributeWarningDiagnostic(p, "Warning", fmt.Sprintf("Attribute %q must be specified when %q is specified", sibling, p)),
		diag.NewErrorDiagnostic("No path", "Detail "+p.String()),
		diag.NewAttributeErrorDiagnostic(path.Root("name"), "Plain", "Attribute name is invalid"),
	})

	want := []struct {
		severity        diag.Severity
		summary, detail string
	}{
		{diag.SeverityError, "Summary of request.metadata[*].value", "Attribute request.metadata[*].value is invalid"},
		{diag.SeverityWarning, "Warning", `Attribute "request.metadata[*].key" must be specified when "request.metadata[*].value" is specified`},
		// A diagnostic without a path is not rewritten: there is no path to
		// match its text against.
		{diag.SeverityError, "No path", "Detail " + p.String()},
		{diag.SeverityError, "Plain", "Attribute name is invalid"},
	}
	if len(diags) != len(want) {
		t.Fatalf("got %d diagnostics, want %d: %v", len(diags), len(want), diags)
	}
	for i, d := range diags {
		if d.Severity() != want[i].severity || d.Summary() != want[i].summary || d.Detail() != want[i].detail {
			t.Errorf("diagnostic %d = %s %q %q, want %s %q %q", i, d.Severity(), d.Summary(), d.Detail(), want[i].severity, want[i].summary, want[i].detail)
		}
		if withPath, ok := d.(diag.DiagnosticWithPath); ok && i < 2 && !withPath.Path().Equal(p) {
			t.Errorf("diagnostic %d path = %s, want %s", i, withPath.Path(), p)
		}
	}
}

func TestSafeValidatorKeepsDescription(t *testing.T) {
	ctx := context.Background()
	inner := stringvalidator.OneOf("a", "b")
	v := safeString(inner)
	if v.Description(ctx) != inner.Description(ctx) || v.MarkdownDescription(ctx) != inner.MarkdownDescription(ctx) {
		t.Errorf("description = %q, %q, want %q, %q", v.Description(ctx), v.MarkdownDescription(ctx), inner.Description(ctx), inner.MarkdownDescription(ctx))
	}

	var resp validator.StringResponse
	v.ValidateString(ctx, validator.StringRequest{
		Path:        path.Root("tags").AtSetValue(types.StringValue("c")),
		ConfigValue: types.StringValue("c"),
	}, &resp)
	if len(resp.Diagnostics) != 1 || !strings.Contains(resp.Diagnostics[0].Detail(), "Attribute tags[*] value must be one of") {
		t.Errorf("diagnostics = %v, want one about tags[*]", resp.Diagnostics)
	}
}

// stockValidatorPackage is the module of the stock validators, whose
// messages print a set element's whole value in its path.
const stockValidatorPackage = "github.com/hashicorp/terraform-plugin-framework-validators"

// findUnwrappedStockValidators reports, by where, each element of a field
// named Validators, at any depth of v, that is a stock validator rather than
// one wrapped by a safe* function.
func findUnwrappedStockValidators(where string, v reflect.Value, report func(where string, t reflect.Type)) {
	switch v.Kind() {
	case reflect.Interface, reflect.Pointer:
		if !v.IsNil() {
			findUnwrappedStockValidators(where, v.Elem(), report)
		}
	case reflect.Struct:
		for i := range v.NumField() {
			field := v.Type().Field(i)
			if field.Name == "Validators" && v.Field(i).Kind() == reflect.Slice {
				checkStockValidators(where, v.Field(i), report)
				continue
			}
			findUnwrappedStockValidators(where, v.Field(i), report)
		}
	case reflect.Map:
		keys := v.MapKeys()
		sort.Slice(keys, func(i, j int) bool { return fmt.Sprint(keys[i]) < fmt.Sprint(keys[j]) })
		for _, key := range keys {
			findUnwrappedStockValidators(fmt.Sprintf("%s.%v", where, key), v.MapIndex(key), report)
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			findUnwrappedStockValidators(where, v.Index(i), report)
		}
	}
}

// checkStockValidators reports each element of validators, a slice of
// validator interfaces, whose dynamic type is a stock validator.
func checkStockValidators(where string, validators reflect.Value, report func(where string, t reflect.Type)) {
	for i := range validators.Len() {
		elem := validators.Index(i)
		if elem.Kind() == reflect.Interface {
			if elem.IsNil() {
				continue
			}
			elem = elem.Elem()
		}
		typ := elem.Type()
		if typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}
		if strings.HasPrefix(typ.PkgPath(), stockValidatorPackage) {
			report(where, elem.Type())
		}
	}
}

// TestSchemaStockValidatorsWrapped fails when a stock validator from
// terraform-plugin-framework-validators is attached to any resource or data
// source schema, at any depth, or to a resource's or data source's
// ConfigValidators without one of the safe* wrappers, whose diagnostics
// render set elements as [*] instead of printing their values. The
// validators that an element validator such as setvalidator.ValueStringsAre
// applies need no wrapper of their own: the wrapper of the outer validator
// rewrites their diagnostics.
func TestSchemaStockValidatorsWrapped(t *testing.T) {
	ctx := context.Background()
	report := func(where string, typ reflect.Type) {
		t.Errorf("%s: stock validator %s is not wrapped; wrap it with a safe* function", where, typ)
	}

	for name, s := range resourceSchemas(t) {
		findUnwrappedStockValidators(name, reflect.ValueOf(s), report)
	}

	p := New("test")()
	var providerMetadata provider.MetadataResponse
	p.Metadata(ctx, provider.MetadataRequest{}, &providerMetadata)

	for _, newResource := range p.Resources(ctx) {
		r := newResource()
		var metadata resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: providerMetadata.TypeName}, &metadata)
		if withValidators, ok := r.(resource.ResourceWithConfigValidators); ok {
			checkStockValidators(metadata.TypeName+" ConfigValidators", reflect.ValueOf(withValidators.ConfigValidators(ctx)), report)
		}
	}

	for _, newDataSource := range p.DataSources(ctx) {
		d := newDataSource()
		var metadata datasource.MetadataResponse
		d.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: providerMetadata.TypeName}, &metadata)

		var resp datasource.SchemaResponse
		d.Schema(ctx, datasource.SchemaRequest{}, &resp)
		findUnwrappedStockValidators(metadata.TypeName, reflect.ValueOf(resp.Schema), report)
		if withValidators, ok := d.(datasource.DataSourceWithConfigValidators); ok {
			checkStockValidators(metadata.TypeName+" ConfigValidators", reflect.ValueOf(withValidators.ConfigValidators(ctx)), report)
		}
	}
}

// TestFindUnwrappedStockValidators checks that the guard in
// TestSchemaStockValidatorsWrapped finds a stock validator nested in a set
// element, and accepts a wrapped one.
func TestFindUnwrappedStockValidators(t *testing.T) {
	s := schema.Schema{Attributes: map[string]schema.Attribute{
		"items": schema.SetNestedAttribute{NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"bare":    schema.StringAttribute{Validators: []validator.String{stringvalidator.LengthAtLeast(1)}},
				"wrapped": schema.StringAttribute{Validators: []validator.String{safeString(stringvalidator.LengthAtLeast(1))}},
			},
		}},
	}}

	var found []string
	findUnwrappedStockValidators("test", reflect.ValueOf(s), func(where string, _ reflect.Type) { found = append(found, where) })
	if len(found) != 1 || found[0] != "test.items.bare" {
		t.Errorf("found = %v, want [test.items.bare]", found)
	}
}

// checkDiagnosticsHideSetElements fails unless errors has a diagnostic whose
// detail names wantPath, and no summary or detail contains secret or a set
// element rendered with its value.
func checkDiagnosticsHideSetElements(t *testing.T, errors []*tfprotov6.Diagnostic, secret, wantPath string) {
	t.Helper()

	if len(errors) == 0 {
		t.Fatal("no errors")
	}
	named := false
	for _, d := range errors {
		for _, text := range []string{d.Summary, d.Detail} {
			if strings.Contains(text, secret) {
				t.Errorf("diagnostic prints the secret %q: %q", secret, text)
			}
			if strings.Contains(text, "Value(") {
				t.Errorf("diagnostic prints a set element value: %q", text)
			}
		}
		named = named || strings.Contains(d.Detail, wantPath)
	}
	if !named {
		t.Errorf("no diagnostic names %s: %v", wantPath, errors)
	}
}

func TestGRPCMonitorMetadataDiagnosticsHideValues(t *testing.T) {
	const secret = "s3cr3t-token"

	t.Run("value too long", func(t *testing.T) {
		errors := validateResourceConfig(t, NewGRPCMonitorResource(), "checkly_grpc_monitor", "request", func(objectType tftypes.Object) tftypes.Value {
			requestType := objectType.AttributeTypes["request"].(tftypes.Object)
			setType := requestType.AttributeTypes["metadata"].(tftypes.Set)
			return objectWith(objectType, map[string]tftypes.Value{
				"request": objectWith(requestType, map[string]tftypes.Value{
					"metadata": grpcMetadataElements(setType,
						[2]any{"authorization", "Bearer " + secret},
						[2]any{"x-long", strings.Repeat("a", 8193)},
					),
				}),
			})
		})
		checkDiagnosticsHideSetElements(t, errors, secret, "request.metadata[*].value")
	})

	t.Run("duplicate keys", func(t *testing.T) {
		errors := validateGRPCMonitorMetadata(t, func(setType tftypes.Set) tftypes.Value {
			return grpcMetadataElements(setType, [2]any{"authorization", secret + "-1"}, [2]any{"Authorization", secret + "-2"})
		})
		checkDiagnosticsHideSetElements(t, errors, secret, "request.metadata")
	})
}

func TestGRPCMonitorAssertionDiagnosticsHideValues(t *testing.T) {
	const secret = "s3cr3t-target"

	errors := validateResourceConfig(t, NewGRPCMonitorResource(), "checkly_grpc_monitor", "request", func(objectType tftypes.Object) tftypes.Value {
		requestType := objectType.AttributeTypes["request"].(tftypes.Object)
		setType := requestType.AttributeTypes["assertions"].(tftypes.Set)
		return objectWith(objectType, map[string]tftypes.Value{
			"request": objectWith(requestType, map[string]tftypes.Value{
				"assertions": tftypes.NewValue(setType, []tftypes.Value{
					objectWith(setType.ElementType.(tftypes.Object), map[string]tftypes.Value{
						"source":     tftypes.NewValue(tftypes.String, "NOT_A_SOURCE"),
						"comparison": tftypes.NewValue(tftypes.String, "EQUALS"),
						"target":     tftypes.NewValue(tftypes.String, secret),
					}),
				}),
			}),
		})
	})
	checkDiagnosticsHideSetElements(t, errors, secret, "request.assertions[*].source")
}

func TestStatusPageV3AutomationRuleTagDiagnosticsHideValues(t *testing.T) {
	errors := validateResourceConfig(t, NewStatusPageV3AutomationRuleResource(), "checkly_status_page_v3_automation_rule", "tags", func(objectType tftypes.Object) tftypes.Value {
		setType := objectType.AttributeTypes["tags"].(tftypes.Set)
		return objectWith(objectType, map[string]tftypes.Value{
			"tags": tftypes.NewValue(setType, []tftypes.Value{
				tftypes.NewValue(tftypes.String, ""),
				tftypes.NewValue(tftypes.String, "production"),
			}),
		})
	})
	checkDiagnosticsHideSetElements(t, errors, "production", "tags[*]")
}
