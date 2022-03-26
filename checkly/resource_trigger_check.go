package checkly

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	checkly "github.com/checkly/checkly-go-sdk"
)

func resourceTriggerCheck() *schema.Resource {
	return &schema.Resource{
		Create: resourceTriggerCheckCreate,
		Read:   resourceTriggerCheckRead,
		Delete: resourceTriggerCheckDelete,
		Update: resourceTriggerCheckUpdate,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"check_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The id of the check that you want to attach the trigger to.",
			},
			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The token value created to trigger the check",
			},
			"url": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The request URL to trigger the check run.",
			},
		},
	}
}

func triggerCheckFromResourceData(data *schema.ResourceData) (checkly.TriggerCheck, error) {
	ID, err := strconv.ParseInt(data.Id(), 10, 64)
	if err != nil {
		if data.Id() != "" {
			return checkly.TriggerCheck{}, err
		}
		ID = 0
	}
	return checkly.TriggerCheck{
		ID:      ID,
		CheckId: data.Get("check_id").(string),
		Token:   data.Get("token").(string),
		URL:     data.Get("url").(string),
	}, nil
}

func resourceDataFromTriggerCheck(trigger *checkly.TriggerCheck, data *schema.ResourceData) error {
	data.Set("check_id", trigger.CheckId)
	data.Set("token", trigger.Token)
	data.Set("url", trigger.URL)
	return nil
}

func resourceTriggerCheckCreate(data *schema.ResourceData, client interface{}) error {
	trigger, err := triggerCheckFromResourceData(data)
	if err != nil {
		return fmt.Errorf("resourceTriggerCheckCreate: translation error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	result, err := client.(checkly.Client).CreateTriggerCheck(ctx, trigger.CheckId)
	if err != nil {
		return fmt.Errorf("CreateTriggerCheck: API error: %w", err)
	}

	data.SetId(fmt.Sprintf("%d", result.ID))

	return resourceTriggerCheckRead(data, client)
}

func resourceTriggerCheckDelete(data *schema.ResourceData, client interface{}) error {
	trigger, err := triggerCheckFromResourceData(data)
	if err != nil {
		return fmt.Errorf("resourceTriggerCheckDelete: translation error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	err = client.(checkly.Client).DeleteTriggerCheck(ctx, trigger.CheckId)
	if err != nil {
		return fmt.Errorf("DeleteTriggerCheck: API error: %w", err)
	}

	return nil
}

func resourceTriggerCheckRead(data *schema.ResourceData, client interface{}) error {
	trigger, err := triggerCheckFromResourceData(data)
	if err != nil {
		return fmt.Errorf("resourceTriggerCheckRead: ID %s is not numeric: %w", data.Id(), err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	result, err := client.(checkly.Client).GetTriggerCheck(ctx, trigger.CheckId)
	defer cancel()
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			data.SetId("")
			return nil
		}
		return fmt.Errorf("GetTriggerCheck: API error: %w", err)
	}

	return resourceDataFromTriggerCheck(result, data)
}

func resourceTriggerCheckUpdate(data *schema.ResourceData, client interface{}) error {
	return resourceTriggerCheckRead(data, client)
}
