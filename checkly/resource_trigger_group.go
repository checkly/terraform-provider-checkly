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
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"token": {
				Type:     schema.TypeString,
				Required: true,
			},
			"group_id": {
				Type:     schema.TypeInt,
				Required: true,
			},
		},
	}
}

func triggerGroupFromResourceData(d *schema.ResourceData) (checkly.TriggerGroup, error) {
	ID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		if d.Id() != "" {
			return checkly.TriggerGroup{}, err
		}
		ID = 0
	}
	a := checkly.TriggerGroup{
		ID:      ID,
		Token:   d.Get("token").(string),
		GroupId: int64(d.Get("group_id").(int)),
	}

	fmt.Printf("%v", a)

	return a, nil
}

func resourceDataFromTriggerGroup(s *checkly.TriggerGroup, d *schema.ResourceData) error {
	d.Set("token", s.Token)
	d.Set("group_id", s.GroupId)
	return nil
}

func resourceTriggerGroupCreate(d *schema.ResourceData, client interface{}) error {
	tc, err := triggerGroupFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourceTriggerGroupCreate: translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	result, err := client.(checkly.Client).CreateTriggerGroup(ctx, tc.GroupId)

	if err != nil {
		return fmt.Errorf("CreateTriggerGroup: API error: %w", err)
	}

	d.SetId(fmt.Sprintf("%d", result.ID))
	return resourceTriggerGroupRead(d, client)
}

func resourceTriggerGroupDelete(d *schema.ResourceData, client interface{}) error {
	tc, err := triggerGroupFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourceTriggerGroupCreate: translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	err = client.(checkly.Client).DeleteTriggerGroup(ctx, tc.GroupId, tc.Token)
	if err != nil {
		return fmt.Errorf("resourceTriggerGroupDelete: API error: %w", err)
	}
	return nil
}

func resourceTriggerGroupRead(d *schema.ResourceData, client interface{}) error {
	tc, err := triggerGroupFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourceTriggerCheckDelete: ID %s is not numeric: %w", d.Id(), err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	mw, err := client.(checkly.Client).GetTriggerGroup(ctx, tc.GroupId)
	defer cancel()
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("resourceTriggerGroupRead: API error: %w", err)
	}
	return resourceDataFromTriggerGroup(mw, d)
}

func resourceTriggerGroupUpdate(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	mw, err := client.(checkly.Client).GetTriggerGroup(ctx, 4)
	defer cancel()
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("resourceTriggerCheckRead: API error: %w", err)
	}
	return resourceDataFromTriggerGroup(mw, d)
}
