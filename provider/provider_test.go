package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
)

func TestNew(t *testing.T) {
	var resp provider.MetadataResponse
	New("test")().Metadata(context.Background(), provider.MetadataRequest{}, &resp)
	if resp.TypeName != "checkly" || resp.Version != "test" {
		t.Errorf("got type name %q and version %q, want checkly and test", resp.TypeName, resp.Version)
	}
}
