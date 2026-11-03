package provider

import (
	"context"
	"errors"
	"testing"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// readBackFailingClient creates groups successfully but fails to read them
// back. Calls to any other method panic through the nil embedded Client.
type readBackFailingClient struct {
	checkly.Client
}

func (readBackFailingClient) CreateGroup(ctx context.Context, group checkly.Group) (*checkly.Group, error) {
	group.ID = 42
	return &group, nil
}

func (readBackFailingClient) GetGroup(ctx context.Context, id int64) (*checkly.Group, error) {
	return nil, errors.New("502 Bad Gateway")
}

func (readBackFailingClient) CreateGroupV2(ctx context.Context, group checkly.GroupV2) (*checkly.GroupV2, error) {
	group.ID = 42
	return &group, nil
}

func (readBackFailingClient) GetGroupV2(ctx context.Context, id int64) (*checkly.GroupV2, error) {
	return nil, errors.New("502 Bad Gateway")
}

// A group whose read-back after creation fails must still be recorded in
// state, so that Terraform tracks it (as tainted) instead of orphaning it.
func TestCheckGroupCreateKeepsGroupWhenReadBackFails(t *testing.T) {
	ctx := context.Background()

	cases := map[string]struct {
		resource resource.Resource
	}{
		"checkly_check_group": {
			resource: &CheckGroupResource{client: readBackFailingClient{}},
		},
		"checkly_check_group_v2": {
			resource: &CheckGroupV2Resource{client: readBackFailingClient{}},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			var schemaResp resource.SchemaResponse
			c.resource.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
			s := schemaResp.Schema

			plan := tfsdk.Plan{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
			if diags := plan.SetAttribute(ctx, path.Root("name"), "g"); diags.HasError() {
				t.Fatal(diags)
			}

			resp := resource.CreateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}}
			c.resource.Create(ctx, resource.CreateRequest{Plan: plan}, &resp)

			if !resp.Diagnostics.HasError() {
				t.Fatal("expected an error diagnostic")
			}
			var id types.String
			if diags := resp.State.GetAttribute(ctx, path.Root("id"), &id); diags.HasError() {
				t.Fatal(diags)
			}
			if id.ValueString() != "42" {
				t.Errorf("state id = %s, want \"42\"", id)
			}
		})
	}
}
