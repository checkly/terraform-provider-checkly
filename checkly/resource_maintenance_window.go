package checkly

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	checkly "github.com/checkly/checkly-go-sdk"
)

func resourceMaintenanceWindow() *schema.Resource {
	return &schema.Resource{
		Create: resourceMaintenanceWindowCreate,
		Read:   resourceMaintenanceWindowRead,
		Update: resourceMaintenanceWindowUpdate,
		Delete: resourceMaintenanceWindowDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The maintenance window name.",
			},
			"starts_at": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The start date of the maintenance window.",
			},
			"ends_at": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The end date of the maintenance window.",
			},
			"repeat_unit": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  nil,
				ValidateFunc: func(value interface{}, key string) (warns []string, errs []error) {
					v := value.(string)
					isValid := false
					options := []string{"DAY", "WEEK", "MONTH"}
					for _, option := range options {
						if v == option {
							isValid = true
						}
					}
					if !isValid {
						errs = append(errs, fmt.Errorf("%q must be one of %v, got %s", key, options, v))
					}
					return warns, errs
				},
				Description: "The repeat cadence for the maintenance window. Possible values `DAY`, `WEEK` and `MONTH`.",
			},
			"repeat_interval": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     nil,
				Description: "The repeat interval of the maintenance window from the first occurrence.",
			},
			"repeat_ends_at": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     nil,
				Description: "The date on which the maintenance window should stop repeating.",
			},
			"tags": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				DefaultFunc: func() (interface{}, error) {
					return []tfMap{}, nil
				},
				Description: "The names of the checks and groups maintenance window should apply to.",
			},
		},
	}
}

func maintenanceWindowsFromResourceData(d *schema.ResourceData) (checkly.MaintenanceWindow, error) {
	ID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		if d.Id() != "" {
			return checkly.MaintenanceWindow{}, err
		}
		ID = 0
	}
	a := checkly.MaintenanceWindow{
		ID:             ID,
		Name:           d.Get("name").(string),
		StartsAt:       d.Get("starts_at").(string),
		EndsAt:         d.Get("ends_at").(string),
		RepeatUnit:     d.Get("repeat_unit").(string),
		RepeatEndsAt:   d.Get("repeat_ends_at").(string),
		RepeatInterval: d.Get("repeat_interval").(int),
		Tags:           stringsFromSet(d.Get("tags").(*schema.Set)),
	}

	fmt.Printf("%v", a)

	return a, nil
}

func resourceDataFromMaintenanceWindows(s *checkly.MaintenanceWindow, d *schema.ResourceData) error {
	d.Set("name", s.Name)
	d.Set("starts_at", s.StartsAt)
	d.Set("ends_at", s.EndsAt)
	d.Set("repeat_unit", s.RepeatUnit)
	d.Set("repeat_ends_at", s.RepeatEndsAt)
	d.Set("repeat_interval", s.RepeatInterval)
	d.Set("tags", s.Tags)
	return nil
}

func resourceMaintenanceWindowCreate(d *schema.ResourceData, client interface{}) error {
	mw, err := maintenanceWindowsFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourceMaintenanceWindowCreate: translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	result, err := client.(checkly.Client).CreateMaintenanceWindow(ctx, mw)

	if err != nil {
		return fmt.Errorf("CreateMaintenanceWindows: API error: %w", err)
	}

	d.SetId(fmt.Sprintf("%d", result.ID))
	return resourceMaintenanceWindowRead(d, client)
}

func resourceMaintenanceWindowUpdate(d *schema.ResourceData, client interface{}) error {
	mw, err := maintenanceWindowsFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourceMaintenanceWindowUpdate: translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	_, err = client.(checkly.Client).UpdateMaintenanceWindow(ctx, mw.ID, mw)
	if err != nil {
		return fmt.Errorf("resourceMaintenanceWindowUpdate: API error: %w", err)
	}
	d.SetId(fmt.Sprintf("%d", mw.ID))
	return resourceMaintenanceWindowRead(d, client)
}

func resourceMaintenanceWindowDelete(d *schema.ResourceData, client interface{}) error {
	ID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return fmt.Errorf("resourceMaintenanceWindowDelete: ID %s is not numeric: %w", d.Id(), err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	err = client.(checkly.Client).DeleteMaintenanceWindow(ctx, ID)
	if err != nil {
		return fmt.Errorf("resourceMaintenanceWindowDelete: API error: %w", err)
	}
	return nil
}

func resourceMaintenanceWindowRead(d *schema.ResourceData, client interface{}) error {
	ID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return fmt.Errorf("resourceMaintenanceWindowRead: ID %s is not numeric: %w", d.Id(), err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	mw, err := client.(checkly.Client).GetMaintenanceWindow(ctx, ID)
	defer cancel()
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("resourceMaintenanceWindowRead: API error: %w", err)
	}
	return resourceDataFromMaintenanceWindows(mw, d)
}
