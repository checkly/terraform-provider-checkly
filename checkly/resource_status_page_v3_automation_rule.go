package checkly

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	checkly "github.com/checkly/checkly-go-sdk"
)

var statusPageV3TargetImpactValues = allowedValues[string]{
	{Value: "UNDER_MAINTENANCE"},
	{Value: "DEGRADED_PERFORMANCE"},
	{Value: "PARTIAL_OUTAGE"},
	{Value: "MAJOR_OUTAGE"},
}

func resourceStatusPageV3AutomationRule() *schema.Resource {
	return &schema.Resource{
		Create:   resourceStatusPageV3AutomationRuleCreate,
		Read:     resourceStatusPageV3AutomationRuleRead,
		Update:   resourceStatusPageV3AutomationRuleUpdate,
		Delete:   resourceStatusPageV3AutomationRuleDelete,
		Importer: statusPageV3CompositeImporter("rule"),
		Description: "An automation rule of a v3 status page. When a check " +
			"whose tags overlap with the rule's tags fails, Checkly opens " +
			"one incident on the page impacting the listed components, and " +
			"resolves it when the check recovers. Import uses the composite " +
			"ID `<status_page_id>/<rule_id>`.",
		Schema: map[string]*schema.Schema{
			"status_page_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the v3 status page the rule belongs to.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the rule.",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "A disabled rule never opens incidents. (Default `true`).",
			},
			"first_update": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Body of the status update that opens the incident.",
			},
			"last_update": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Body of the status update that resolves the incident.",
			},
			"notify_subscribers": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether subscribers are notified of the automated updates. (Default `true`).",
			},
			"cool_down_window_minutes": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      5,
				Description:  "Minimum minutes after an automated incident before this rule may open the next one. `0` disables the cool down. (Default `5`).",
				ValidateFunc: validateBetween(0, 1440),
			},
			"tags": {
				Type:        schema.TypeSet,
				Required:    true,
				MinItems:    1,
				Description: "A failing check matches this rule when it, or its group, carries ANY of these tags. At least one tag is required.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"component": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "The components an automated incident impacts, with the impact each gets.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"component_id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The ID of the impacted component. Must be on the same status page.",
						},
						"target_impact": {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "The impact set on the component while the incident is open. " + statusPageV3TargetImpactValues.String(),
							ValidateFunc: validateOneOf(statusPageV3TargetImpactValues.Values()),
						},
					},
				},
			},
		},
	}
}

func statusPageV3AutomationRuleFromResourceData(d *schema.ResourceData) (checkly.StatusPageAutomationRuleV3, error) {
	components := []checkly.StatusPageAutomationRuleComponentV3{}
	seen := map[string]bool{}
	for _, it := range d.Get("component").(*schema.Set).List() {
		tm := it.(tfMap)
		componentID := tm["component_id"].(string)
		// The API stores one impact per (rule, component); a duplicate would
		// surface as an opaque server error.
		if seen[componentID] {
			return checkly.StatusPageAutomationRuleV3{}, fmt.Errorf("component %q is listed more than once", componentID)
		}
		seen[componentID] = true
		components = append(components, checkly.StatusPageAutomationRuleComponentV3{
			ComponentID:  componentID,
			TargetImpact: checkly.StatusPageTargetImpactV3(tm["target_impact"].(string)),
		})
	}
	return checkly.StatusPageAutomationRuleV3{
		ID:                    d.Id(),
		Name:                  d.Get("name").(string),
		Enabled:               d.Get("enabled").(bool),
		FirstUpdate:           d.Get("first_update").(string),
		LastUpdate:            d.Get("last_update").(string),
		NotifySubscribers:     d.Get("notify_subscribers").(bool),
		CoolDownWindowMinutes: d.Get("cool_down_window_minutes").(int),
		Tags:                  stringsFromSet(d.Get("tags").(*schema.Set)),
		Components:            components,
	}, nil
}

func resourceDataFromStatusPageV3AutomationRule(r *checkly.StatusPageAutomationRuleV3, d *schema.ResourceData) error {
	components := make([]tfMap, 0, len(r.Components))
	for _, component := range r.Components {
		components = append(components, tfMap{
			"component_id":  component.ComponentID,
			"target_impact": string(component.TargetImpact),
		})
	}
	pairs := []struct {
		key   string
		value interface{}
	}{
		{"status_page_id", r.StatusPageID},
		{"name", r.Name},
		{"enabled", r.Enabled},
		{"first_update", r.FirstUpdate},
		{"last_update", r.LastUpdate},
		{"notify_subscribers", r.NotifySubscribers},
		{"cool_down_window_minutes", r.CoolDownWindowMinutes},
		{"tags", r.Tags},
		{"component", components},
	}
	for _, pair := range pairs {
		if err := d.Set(pair.key, pair.value); err != nil {
			return fmt.Errorf("failed to set %q: %w", pair.key, err)
		}
	}
	return nil
}

func resourceStatusPageV3AutomationRuleCreate(d *schema.ResourceData, client interface{}) error {
	rule, err := statusPageV3AutomationRuleFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourceStatusPageV3AutomationRuleCreate: translation error: %w", err)
	}
	statusPageID := d.Get("status_page_id").(string)
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	result, err := client.(checkly.Client).CreateStatusPageAutomationRuleV3(ctx, statusPageID, rule)
	if err != nil {
		return fmt.Errorf("CreateStatusPageAutomationRuleV3: API error: %w", err)
	}
	d.SetId(result.ID)
	return resourceStatusPageV3AutomationRuleRead(d, client)
}

func resourceStatusPageV3AutomationRuleRead(d *schema.ResourceData, client interface{}) error {
	statusPageID := d.Get("status_page_id").(string)
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	rule, err := client.(checkly.Client).GetStatusPageAutomationRuleV3(ctx, statusPageID, d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			// Deleted remotely, either the rule itself or the whole page:
			// mark the resource as gone.
			d.SetId("")
			return nil
		}
		return fmt.Errorf("resourceStatusPageV3AutomationRuleRead: API error: %w", err)
	}
	return resourceDataFromStatusPageV3AutomationRule(rule, d)
}

func resourceStatusPageV3AutomationRuleUpdate(d *schema.ResourceData, client interface{}) error {
	rule, err := statusPageV3AutomationRuleFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourceStatusPageV3AutomationRuleUpdate: translation error: %w", err)
	}
	statusPageID := d.Get("status_page_id").(string)
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	_, err = client.(checkly.Client).UpdateStatusPageAutomationRuleV3(ctx, statusPageID, rule.ID, rule)
	if err != nil {
		return fmt.Errorf("resourceStatusPageV3AutomationRuleUpdate: API error: %w", err)
	}
	return resourceStatusPageV3AutomationRuleRead(d, client)
}

func resourceStatusPageV3AutomationRuleDelete(d *schema.ResourceData, client interface{}) error {
	statusPageID := d.Get("status_page_id").(string)
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	err := client.(checkly.Client).DeleteStatusPageAutomationRuleV3(ctx, statusPageID, d.Id())
	if err != nil {
		return fmt.Errorf("resourceStatusPageV3AutomationRuleDelete: API error: %w", err)
	}
	return nil
}
