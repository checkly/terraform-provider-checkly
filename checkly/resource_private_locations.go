package checkly

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	checkly "github.com/checkly/checkly-go-sdk"
)

func resourcePrivateLocations() *schema.Resource {
	return &schema.Resource{
		Create: resourcePrivateLocationsCreate,
		Read:   resourcePrivateLocationsRead,
		Update: resourcePrivateLocationsUpdate,
		Delete: resourcePrivateLocationsDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"slug_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"icon": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func privateLocationsFromResourceData(d *schema.ResourceData) (checkly.PrivateLocation, error) {
	return checkly.PrivateLocation{
		Name:     d.Get("name").(string),
		SlugName: d.Get("slug_name").(string),
		Icon:     d.Get("icon").(string),
	}, nil
}

func resourcePrivateLocationsCreate(d *schema.ResourceData, client interface{}) error {
	pl, err := privateLocationsFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourcePrivateLocationsCreate: translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	result, err := client.(checkly.Client).CreatePrivateLocation(ctx, pl)
	if err != nil {
		return fmt.Errorf("CreatePrivateLocation: API error: %w", err)
	}
	d.SetId(result.ID)
	return resourcePrivateLocationsRead(d, client)
}

func resourceDataFromPrivateLocations(s *checkly.PrivateLocation, d *schema.ResourceData) error {
	d.Set("name", s.Name)
	d.Set("slug_name", s.SlugName)
	d.Set("icon", s.Icon)
	return nil
}

func resourcePrivateLocationsUpdate(d *schema.ResourceData, client interface{}) error {
	pl, err := privateLocationsFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourcePrivateLocationsUpdate: translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	_, err = client.(checkly.Client).UpdatePrivateLocation(ctx, d.Id(), pl)
	if err != nil {
		return fmt.Errorf("resourcePrivateLocationsUpdate: API error: %w", err)
	}

	return resourcePrivateLocationsRead(d, client)
}

func resourcePrivateLocationsRead(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	pl, err := client.(checkly.Client).GetPrivateLocation(ctx, d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			//if resource is deleted remotely, then mark it as
			//successfully gone by unsetting it's ID
			d.SetId("")
			return nil
		}
		return fmt.Errorf("resourcePrivateLocationsRead: %w", err)
	}
	return resourceDataFromPrivateLocations(pl, d)
}

func resourcePrivateLocationsDelete(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	err := client.(checkly.Client).DeletePrivateLocation(ctx, d.Id())
	if err != nil {
		return fmt.Errorf("resourcePrivateLocationsDelete: API error: %w", err)
	}
	return nil
}
