package provider

import (
	"context"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// singleElementLimit matches the description of a size validator that caps a
// nested list or set attribute at one element, such as
// listvalidator.SizeAtMost(1), which would make it a single nested object in
// disguise.
var singleElementLimit = regexp.MustCompile(`at most 1 elements?\b`)

// checkSizeValidators fails when the validators of a nested collection
// attribute cap it at one element.
func checkSizeValidators[V validator.Describer](t *testing.T, where string, validators []V) {
	t.Helper()

	for _, v := range validators {
		if description := v.Description(context.Background()); singleElementLimit.MatchString(description) {
			t.Errorf("%s: limited to one element (%q); make it a single nested attribute", where, description)
		}
	}
}

// walkResourceAttributes checks the nested list and set attributes among
// attrs, at any depth, for a one-element cap.
func walkResourceAttributes(t *testing.T, where string, attrs map[string]schema.Attribute) {
	t.Helper()

	for name, a := range attrs {
		path := where + "." + name

		switch a := a.(type) {
		case schema.SingleNestedAttribute:
			walkResourceAttributes(t, path, a.Attributes)
		case schema.ListNestedAttribute:
			checkSizeValidators(t, path, a.Validators)
			walkResourceAttributes(t, path, a.NestedObject.Attributes)
		case schema.SetNestedAttribute:
			checkSizeValidators(t, path, a.Validators)
			walkResourceAttributes(t, path, a.NestedObject.Attributes)
		case schema.MapNestedAttribute:
			walkResourceAttributes(t, path, a.NestedObject.Attributes)
		}
	}
}

// walkDataSourceAttributes is walkResourceAttributes for data source schemas.
func walkDataSourceAttributes(t *testing.T, where string, attrs map[string]datasourceschema.Attribute) {
	t.Helper()

	for name, a := range attrs {
		path := where + "." + name

		switch a := a.(type) {
		case datasourceschema.SingleNestedAttribute:
			walkDataSourceAttributes(t, path, a.Attributes)
		case datasourceschema.ListNestedAttribute:
			checkSizeValidators(t, path, a.Validators)
			walkDataSourceAttributes(t, path, a.NestedObject.Attributes)
		case datasourceschema.SetNestedAttribute:
			checkSizeValidators(t, path, a.Validators)
			walkDataSourceAttributes(t, path, a.NestedObject.Attributes)
		case datasourceschema.MapNestedAttribute:
			walkDataSourceAttributes(t, path, a.NestedObject.Attributes)
		}
	}
}

// TestNoBlocks keeps every schema free of blocks: repeated objects are list
// or set nested attributes, which can be assigned from expressions, and an
// object held at most once is a single nested attribute. Blocks can only nest
// inside blocks, so checking the top level of each schema covers every depth.
// No nested list or set attribute, at any depth, is capped at one element
// either, which would make it a single nested object in disguise.
func TestNoBlocks(t *testing.T) {
	for name, s := range resourceSchemas(t) {
		walkResourceAttributes(t, name, s.Attributes)
		if len(s.Blocks) > 0 {
			t.Errorf("%s: resources must not use blocks; use nested attributes", name)
		}
	}

	ctx := context.Background()
	p := New("test")()

	var providerMetadata provider.MetadataResponse
	p.Metadata(ctx, provider.MetadataRequest{}, &providerMetadata)

	for _, newDataSource := range p.DataSources(ctx) {
		d := newDataSource()

		var metadata datasource.MetadataResponse
		d.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: providerMetadata.TypeName}, &metadata)

		var resp datasource.SchemaResponse
		d.Schema(ctx, datasource.SchemaRequest{}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("%s: schema diagnostics: %v", metadata.TypeName, resp.Diagnostics)
		}

		walkDataSourceAttributes(t, metadata.TypeName, resp.Schema.Attributes)
		if len(resp.Schema.Blocks) > 0 {
			t.Errorf("%s: data sources must not use blocks; use nested attributes", metadata.TypeName)
		}
	}
}

// The single-element check relies on the descriptions of the framework's
// size validators; this guards against a wording change silently disabling
// it.
func TestSingleElementLimitMatchesSizeValidators(t *testing.T) {
	ctx := context.Background()

	for _, tc := range []struct {
		validator validator.Describer
		want      bool
	}{
		{listvalidator.SizeAtMost(1), true},
		{setvalidator.SizeAtMost(1), true},
		{listvalidator.SizeBetween(1, 1), true},
		{listvalidator.SizeAtMost(10), false},
		{listvalidator.SizeAtLeast(1), false},
	} {
		description := tc.validator.Description(ctx)
		if got := singleElementLimit.MatchString(description); got != tc.want {
			t.Errorf("%q: match = %t, want %t", description, got, tc.want)
		}
	}
}
