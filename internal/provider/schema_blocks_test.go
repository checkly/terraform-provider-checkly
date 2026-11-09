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

// repeatedBlocks is the exact inventory of blocks in the provider's resource
// schemas (data sources have none), keyed by type name and block path. Each
// holds any number of elements; an object that a resource can hold at most
// one of is a single nested attribute (`x = { ... }`) instead. An entry that no schema
// uses any more fails the test, so the list stays exact.
var repeatedBlocks = map[string]bool{
	"checkly_status_page.card":                         true,
	"checkly_status_page.card.service_attachment":      true,
	"checkly_status_page_v3_automation_rule.component": true,
}

// singleElementLimit matches the description of a size validator that caps a
// block or a nested list or set attribute at one element, such as
// listvalidator.SizeAtMost(1), which would make it a single nested object in
// disguise.
var singleElementLimit = regexp.MustCompile(`at most 1 elements?\b`)

// checkSizeValidators fails when the validators of a block or nested
// collection attribute cap it at one element.
func checkSizeValidators[V validator.Describer](t *testing.T, where string, validators []V) {
	t.Helper()

	for _, v := range validators {
		if description := v.Description(context.Background()); singleElementLimit.MatchString(description) {
			t.Errorf("%s: limited to one element (%q); make it a single nested attribute", where, description)
		}
	}
}

// checkRepeatedBlock fails when the block at path is not in repeatedBlocks,
// and records it as seen.
func checkRepeatedBlock(t *testing.T, path string, seen map[string]bool) {
	t.Helper()

	seen[path] = true
	if !repeatedBlocks[path] {
		t.Errorf("%s: block is not in repeatedBlocks; objects held at most once must be single nested attributes, repeated blocks must be added to the list", path)
	}
}

func walkResourceBlocks(t *testing.T, where string, seen map[string]bool, blocks map[string]schema.Block) {
	t.Helper()

	for name, block := range blocks {
		path := where + "." + name
		checkRepeatedBlock(t, path, seen)

		switch block := block.(type) {
		case schema.ListNestedBlock:
			checkSizeValidators(t, path, block.Validators)
			walkResourceAttributes(t, path, block.NestedObject.Attributes)
			walkResourceBlocks(t, path, seen, block.NestedObject.Blocks)
		case schema.SetNestedBlock:
			checkSizeValidators(t, path, block.Validators)
			walkResourceAttributes(t, path, block.NestedObject.Attributes)
			walkResourceBlocks(t, path, seen, block.NestedObject.Blocks)
		default:
			t.Errorf("%s: block of type %T; only list and set nested blocks may remain", path, block)
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

// TestOnlyRepeatedBlocks keeps every object that a resource or data source
// can hold at most one of a single nested attribute: blocks are reserved for
// repeated objects such as card, and no block or nested
// list or set attribute, at any depth, is capped at one element.
func TestOnlyRepeatedBlocks(t *testing.T) {
	seen := make(map[string]bool)

	for name, s := range resourceSchemas(t) {
		walkResourceAttributes(t, name, s.Attributes)
		walkResourceBlocks(t, name, seen, s.Blocks)
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

		// No data source has blocks; nested attributes cover every shape.
		if len(resp.Schema.Blocks) > 0 {
			t.Errorf("%s: data sources must not use blocks; use nested attributes", metadata.TypeName)
		}
	}

	for path := range repeatedBlocks {
		if !seen[path] {
			t.Errorf("%s: listed in repeatedBlocks, but no schema has this block; remove the entry", path)
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
