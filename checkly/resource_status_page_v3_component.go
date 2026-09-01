package checkly

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	checkly "github.com/checkly/checkly-go-sdk"
)

var statusPageV3ComponentTypeValues = allowedValues[string]{
	{Value: "SERVICE", Description: "a monitored thing with its own status"},
	{Value: "GROUP", Description: "a container for other components"},
}

// statusPageV3CompositeImporter imports nested v3 status page resources,
// whose API routes need the page ID in addition to their own: the import ID
// is "<status_page_id>/<id>".
func statusPageV3CompositeImporter(resourceName string) *schema.ResourceImporter {
	return &schema.ResourceImporter{
		StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
			parts := strings.Split(d.Id(), "/")
			if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
				return nil, fmt.Errorf("invalid import ID %q, expected \"<status_page_id>/<%s_id>\"", d.Id(), resourceName)
			}
			if err := d.Set("status_page_id", parts[0]); err != nil {
				return nil, fmt.Errorf("failed to set \"status_page_id\": %w", err)
			}
			d.SetId(parts[1])
			return []*schema.ResourceData{d}, nil
		},
	}
}

func resourceStatusPageV3Component() *schema.Resource {
	return &schema.Resource{
		Create:   resourceStatusPageV3ComponentCreate,
		Read:     resourceStatusPageV3ComponentRead,
		Update:   resourceStatusPageV3ComponentUpdate,
		Delete:   resourceStatusPageV3ComponentDelete,
		Importer: statusPageV3CompositeImporter("component"),
		Description: "A component of a v3 status page: either a SERVICE (a " +
			"monitored thing with its own status) or a GROUP (a container " +
			"for other components). Import uses the composite ID " +
			"`<status_page_id>/<component_id>`.",
		Schema: map[string]*schema.Schema{
			"status_page_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the v3 status page the component belongs to. A component cannot be moved to another page.",
			},
			"type": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "SERVICE",
				Description:  "The type of the component. " + statusPageV3ComponentTypeValues.String() + " (Default `SERVICE`).",
				ValidateFunc: validateOneOf(statusPageV3ComponentTypeValues.Values()),
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name shown on the status page.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "An optional description shown next to the name.",
			},
			"display_order": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The position among siblings; lower comes first.",
			},
			"hidden": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Hide the component from the public page while keeping it available for incidents and automation. (Default `false`).",
			},
			"parent_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The ID of the GROUP component to nest this component under. Must be on the same status page.",
			},
		},
	}
}

func statusPageV3ComponentFromResourceData(d *schema.ResourceData) checkly.StatusPageComponentV3 {
	return checkly.StatusPageComponentV3{
		ID:           d.Id(),
		Type:         checkly.StatusPageComponentV3Type(d.Get("type").(string)),
		Name:         d.Get("name").(string),
		Description:  d.Get("description").(string),
		DisplayOrder: d.Get("display_order").(int),
		Hidden:       d.Get("hidden").(bool),
		ParentID:     d.Get("parent_id").(string),
	}
}

func resourceDataFromStatusPageV3Component(c *checkly.StatusPageComponentV3, d *schema.ResourceData) error {
	pairs := []struct {
		key   string
		value interface{}
	}{
		{"status_page_id", c.StatusPageID},
		{"type", string(c.Type)},
		{"name", c.Name},
		{"description", c.Description},
		{"display_order", c.DisplayOrder},
		{"hidden", c.Hidden},
		{"parent_id", c.ParentID},
	}
	for _, pair := range pairs {
		if err := d.Set(pair.key, pair.value); err != nil {
			return fmt.Errorf("failed to set %q: %w", pair.key, err)
		}
	}
	return nil
}

func resourceStatusPageV3ComponentCreate(d *schema.ResourceData, client interface{}) error {
	component := statusPageV3ComponentFromResourceData(d)
	statusPageID := d.Get("status_page_id").(string)
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	result, err := client.(checkly.Client).CreateStatusPageComponentV3(ctx, statusPageID, component)
	if err != nil {
		return fmt.Errorf("CreateStatusPageComponentV3: API error: %w", err)
	}
	d.SetId(result.ID)
	return resourceStatusPageV3ComponentRead(d, client)
}

func resourceStatusPageV3ComponentRead(d *schema.ResourceData, client interface{}) error {
	statusPageID := d.Get("status_page_id").(string)
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	component, err := client.(checkly.Client).GetStatusPageComponentV3(ctx, statusPageID, d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			// Deleted remotely, either the component itself or the whole
			// page: mark the resource as gone.
			d.SetId("")
			return nil
		}
		return fmt.Errorf("resourceStatusPageV3ComponentRead: API error: %w", err)
	}
	return resourceDataFromStatusPageV3Component(component, d)
}

func resourceStatusPageV3ComponentUpdate(d *schema.ResourceData, client interface{}) error {
	component := statusPageV3ComponentFromResourceData(d)
	statusPageID := d.Get("status_page_id").(string)
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	_, err := client.(checkly.Client).UpdateStatusPageComponentV3(ctx, statusPageID, component.ID, component)
	if err != nil {
		return fmt.Errorf("resourceStatusPageV3ComponentUpdate: API error: %w", err)
	}
	return resourceStatusPageV3ComponentRead(d, client)
}

func resourceStatusPageV3ComponentDelete(d *schema.ResourceData, client interface{}) error {
	statusPageID := d.Get("status_page_id").(string)
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	err := client.(checkly.Client).DeleteStatusPageComponentV3(ctx, statusPageID, d.Id())
	if err != nil {
		return fmt.Errorf("resourceStatusPageV3ComponentDelete: API error: %w", err)
	}
	return nil
}
