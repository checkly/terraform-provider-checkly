package checkly

import (
	"context"
	"fmt"
	"log"
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
			"`<status_page_id>/<component_id>`. A group can never be empty: " +
			"when the last member of a group is destroyed, the provider " +
			"deletes the group first (which detaches its members) — destroy " +
			"a group's last member together with its group, not on its own.",
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
			"show_historical_data": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Show the historical status (the uptime bar) of the component on the status page. (Default `true`).",
			},
			"expanded_by_default": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Render the group expanded when the status page loads. Only available on a `GROUP` component. (Default `false`).",
			},
			"parent_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The ID of the GROUP component to nest this component under. Must be on the same status page.",
			},
		},
	}
}

func statusPageV3ComponentFromResourceData(d *schema.ResourceData) (checkly.StatusPageComponentV3, error) {
	componentType := checkly.StatusPageComponentV3Type(d.Get("type").(string))
	showHistoricalData := d.Get("show_historical_data").(bool)
	expandedByDefault := d.Get("expanded_by_default").(bool)

	// The API's configuration shape is discriminated by type: a SERVICE only
	// supports showHistoricalData and rejects expandedByDefault loudly. The
	// attribute's false default is indistinguishable from an explicit false,
	// so only a true value on a SERVICE can (and must) be caught here.
	configuration := &checkly.StatusPageComponentV3Configuration{
		ShowHistoricalData: &showHistoricalData,
	}
	if componentType == checkly.StatusPageComponentV3TypeGroup {
		configuration.ExpandedByDefault = &expandedByDefault
	} else if expandedByDefault {
		return checkly.StatusPageComponentV3{}, fmt.Errorf(`"expanded_by_default" is only available on a GROUP component`)
	}

	return checkly.StatusPageComponentV3{
		ID:            d.Id(),
		Type:          componentType,
		Name:          d.Get("name").(string),
		Description:   d.Get("description").(string),
		DisplayOrder:  d.Get("display_order").(int),
		Hidden:        d.Get("hidden").(bool),
		Configuration: configuration,
		ParentID:      d.Get("parent_id").(string),
	}, nil
}

func resourceDataFromStatusPageV3Component(c *checkly.StatusPageComponentV3, d *schema.ResourceData) error {
	// Reads return the full configuration of the component's type; fall back
	// to the API defaults defensively. ExpandedByDefault is absent on a
	// SERVICE, where false matches the attribute's default.
	showHistoricalData := true
	expandedByDefault := false
	if c.Configuration != nil {
		if c.Configuration.ShowHistoricalData != nil {
			showHistoricalData = *c.Configuration.ShowHistoricalData
		}
		if c.Configuration.ExpandedByDefault != nil {
			expandedByDefault = *c.Configuration.ExpandedByDefault
		}
	}
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
		{"show_historical_data", showHistoricalData},
		{"expanded_by_default", expandedByDefault},
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
	component, err := statusPageV3ComponentFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourceStatusPageV3ComponentCreate: translation error: %w", err)
	}
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
	component, err := statusPageV3ComponentFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourceStatusPageV3ComponentUpdate: translation error: %w", err)
	}
	statusPageID := d.Get("status_page_id").(string)
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	_, err = client.(checkly.Client).UpdateStatusPageComponentV3(ctx, statusPageID, component.ID, component)
	if err != nil {
		return fmt.Errorf("resourceStatusPageV3ComponentUpdate: API error: %w", err)
	}
	return resourceStatusPageV3ComponentRead(d, client)
}

func resourceStatusPageV3ComponentDelete(d *schema.ResourceData, client interface{}) error {
	statusPageID := d.Get("status_page_id").(string)
	deleteComponent := func(id string) error {
		ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
		defer cancel()
		return client.(checkly.Client).DeleteStatusPageComponentV3(ctx, statusPageID, id)
	}
	err := deleteComponent(d.Id())
	if err == nil || strings.Contains(err.Error(), "404") {
		// A 404 means the component is already gone, e.g. deleted along with
		// its group or page.
		return nil
	}
	// The API refuses to remove a group's last member, and Terraform always
	// destroys the member before its group. A group cannot outlive its last
	// member anyway, so delete the group — which detaches its children —
	// and retry. The group's own destroy then finds it gone (404, treated
	// as success above); if the group was not being destroyed, the next
	// plan recreates it empty.
	parentID := d.Get("parent_id").(string)
	if parentID != "" && strings.Contains(err.Error(), "a group cannot be empty") {
		log.Printf("[WARN] deleting component %s: it is the last member of group %s, deleting the group first", d.Id(), parentID)
		if err := deleteComponent(parentID); err != nil && !strings.Contains(err.Error(), "404") {
			return fmt.Errorf("resourceStatusPageV3ComponentDelete: failed to delete the parent group of the group's last member: %w", err)
		}
		if err := deleteComponent(d.Id()); err != nil {
			return fmt.Errorf("resourceStatusPageV3ComponentDelete: API error: %w", err)
		}
		return nil
	}
	return fmt.Errorf("resourceStatusPageV3ComponentDelete: API error: %w", err)
}
