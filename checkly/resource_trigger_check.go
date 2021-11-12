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
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"token": {
				Type:     schema.TypeString,
				Required: true,
			},
			"check_id": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func triggerCheckFromResourceData(d *schema.ResourceData) (checkly.TriggerCheck, error) {
	ID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		if d.Id() != "" {
			return checkly.TriggerCheck{}, err
		}
		ID = 0
	}
	a := checkly.TriggerCheck{
		ID:      ID,
		Token:   d.Get("token").(string),
		CheckId: d.Get("check_id").(string),
	}

	fmt.Printf("%v", a)

	return a, nil
}

func resourceDataFromTriggerCheck(s *checkly.TriggerCheck, d *schema.ResourceData) error {
	d.Set("token", s.Token)
	d.Set("check_id", s.CheckId)
	return nil
}

func resourceTriggerCheckCreate(d *schema.ResourceData, client interface{}) error {
	tc, err := triggerCheckFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourceTriggerCheckCreate: translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	result, err := client.(checkly.Client).CreateTriggerCheck(ctx, tc.CheckId)

	if err != nil {
		return fmt.Errorf("CreateTriggerCheck: API error: %w", err)
	}

	d.SetId(fmt.Sprintf("%d", result.ID))
	return resourceTriggerCheckRead(d, client)
}

func resourceTriggerCheckDelete(d *schema.ResourceData, client interface{}) error {
	tc, err := triggerCheckFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourceTriggerCheckCreate: translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	err = client.(checkly.Client).DeleteTriggerCheck(ctx, tc.CheckId, tc.Token)
	if err != nil {
		return fmt.Errorf("resourceTriggerCheckDelete: API error: %w", err)
	}
	return nil
}

func resourceTriggerCheckRead(d *schema.ResourceData, client interface{}) error {
	tc, err := triggerCheckFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourceTriggerCheckDelete: ID %s is not numeric: %w", d.Id(), err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	mw, err := client.(checkly.Client).GetTriggerCheck(ctx, tc.CheckId)
	defer cancel()
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("resourceTriggerCheckRead: API error: %w", err)
	}
	return resourceDataFromTriggerCheck(mw, d)
}

func resourceTriggerCheckUpdate(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	mw, err := client.(checkly.Client).GetTriggerCheck(ctx, d.Id())
	defer cancel()
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("resourceTriggerCheckRead: API error: %w", err)
	}
	return resourceDataFromTriggerCheck(mw, d)
}
