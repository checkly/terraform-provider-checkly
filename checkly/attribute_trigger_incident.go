package checkly

import (
	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var triggerIncidentAttributeSchema = &schema.Schema{
	Description: "Set up HTTP basic authentication (username & password).",
	Type:        schema.TypeSet,
	MaxItems:    1,
	Optional:    true,
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			"service_id": {
				Description: "The status page service that this incident will be associated with.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"severity": {
				Description:  "The severity level of the incident. Possible values are `MINOR`, `MEDIUM`, `MAJOR`, and `CRITICAL`.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateOneOf([]string{"MINOR", "MEDIUM", "MAJOR", "CRITICAL"}),
			},
			"name": {
				Description: "The name of the incident.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"description": {
				Description: "A detailed description of the incident.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"notify_subscribers": {
				Description: "Whether to notify subscribers when the incident is triggered.",
				Type:        schema.TypeBool,
				Required:    true,
			},
		},
	},
}

func triggerIncidentFromSet(s *schema.Set) *checkly.IncidentTrigger {
	if s.Len() == 0 {
		return nil
	}

	res := s.List()[0].(tfMap)

	triggerIncident := &checkly.IncidentTrigger{
		ServiceID:         res["service_id"].(string),
		Severity:          checkly.IncidentSeverity(res["severity"].(string)),
		Name:              res["name"].(string),
		Description:       res["description"].(string),
		NotifySubscribers: res["notify_subscribers"].(bool),
	}

	return triggerIncident
}

func setFromTriggerIncident(it *checkly.IncidentTrigger) []tfMap {
	if it == nil {
		return []tfMap{}
	}

	return []tfMap{
		{
			"service_id":         it.ServiceID,
			"severity":           it.Severity,
			"name":               it.Name,
			"description":        it.Description,
			"notify_subscribers": it.NotifySubscribers,
		},
	}
}
