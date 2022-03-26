package checkly

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	checkly "github.com/checkly/checkly-go-sdk"
)

func resourceTriggerGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceTriggerGroupCreate,
		Read:   resourceTriggerGroupRead,
		Delete: resourceTriggerGroupDelete,
		Update: resourceTriggerGroupUpdate,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"group_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The id of the group that you want to attach the trigger to.",
			},
			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The token value created to trigger the group",
			},
			"url": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The request URL to trigger the group run.",
			},
		},
	}
}

func triggerGroupFromResourceData(data *schema.ResourceData) (checkly.TriggerGroup, error) {
	ID, err := strconv.ParseInt(data.Id(), 10, 64)
	if err != nil {
		if data.Id() != "" {
			return checkly.TriggerGroup{}, err
		}
		ID = 0
	}

	return checkly.TriggerGroup{
		ID:      ID,
		GroupId: int64(data.Get("group_id").(int)),
		Token:   data.Get("token").(string),
	}, nil
}

func resourceDataFromTriggerGroup(trigger *checkly.TriggerGroup, data *schema.ResourceData) error {
	data.Set("group_id", trigger.GroupId)
	data.Set("token", trigger.Token)
	data.Set("url", trigger.URL)
	return nil
}

func resourceTriggerGroupCreate(data *schema.ResourceData, client interface{}) error {
	tc, err := triggerGroupFromResourceData(data)
	if err != nil {
		return fmt.Errorf("resourceTriggerGroupCreate: translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	result, err := client.(checkly.Client).CreateTriggerGroup(ctx, tc.GroupId)

	if err != nil {
		return fmt.Errorf("CreateTriggerGroup: API error: %w", err)
	}

	data.SetId(fmt.Sprintf("%d", result.ID))

	return resourceTriggerGroupRead(data, client)
}

func resourceTriggerGroupDelete(data *schema.ResourceData, client interface{}) error {
	tc, err := triggerGroupFromResourceData(data)
	if err != nil {
		return fmt.Errorf("resourceTriggerGroupDelete: translation error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	err = client.(checkly.Client).DeleteTriggerGroup(ctx, tc.GroupId)
	if err != nil {
		return fmt.Errorf("DeleteTriggerGroup: API error: %w", err)
	}

	return nil
}

func resourceTriggerGroupRead(data *schema.ResourceData, client interface{}) error {
	trigger, err := triggerGroupFromResourceData(data)
	if err != nil {
		return fmt.Errorf("resourceTriggerGroupRead: ID %s is not numeric: %w", data.Id(), err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	result, err := client.(checkly.Client).GetTriggerGroup(ctx, trigger.GroupId)
	defer cancel()
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			data.SetId("")
			return nil
		}
		return fmt.Errorf("GetTriggerGroup: API error: %w", err)
	}
	return resourceDataFromTriggerGroup(result, data)
}

func resourceTriggerGroupUpdate(data *schema.ResourceData, client interface{}) error {
	return resourceTriggerCheckRead(data, client)
}
