package checkly

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	checkly "github.com/checkly/checkly-go-sdk"
)

func resourcePrivateLocation() *schema.Resource {
	return &schema.Resource{
		Create: resourcePrivateLocationCreate,
		Read:   resourcePrivateLocationRead,
		Update: resourcePrivateLocationUpdate,
		Delete: resourcePrivateLocationDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The private location name.",
			},
			"slug_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Valid slug name.",
			},
			"icon": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Icon assigned to the private location.",
			},
			"key": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
				Description: "Private location API key.",
			},
		},
	}
}

func privateLocationFromResourceData(d *schema.ResourceData) (checkly.PrivateLocation, error) {
	return checkly.PrivateLocation{
		Name:     d.Get("name").(string),
		SlugName: d.Get("slug_name").(string),
		Icon:     d.Get("icon").(string),
	}, nil
}

func resourcePrivateLocationCreate(d *schema.ResourceData, client interface{}) error {
	pl, err := privateLocationFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourcePrivateLocationCreate: translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	result, err := client.(checkly.Client).CreatePrivateLocation(ctx, pl)
	if err != nil {
		return fmt.Errorf("CreatePrivateLocation: API error: %w", err)
	}
	d.SetId(result.ID)
	d.Set("key", result.Keys[0].RawKey)
	return resourcePrivateLocationRead(d, client)
}

func resourceDataFromPrivateLocation(pl *checkly.PrivateLocation, d *schema.ResourceData) error {
	d.Set("name", pl.Name)
	d.Set("slug_name", pl.SlugName)
	d.Set("icon", pl.Icon)
	return nil
}

func resourcePrivateLocationUpdate(d *schema.ResourceData, client interface{}) error {
	pl, err := privateLocationFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourcePrivateLocationUpdate: translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	_, err = client.(checkly.Client).UpdatePrivateLocation(ctx, d.Id(), pl)
	if err != nil {
		return fmt.Errorf("resourcePrivateLocationUpdate: API error: %w", err)
	}
	return resourcePrivateLocationRead(d, client)
}

func resourcePrivateLocationRead(d *schema.ResourceData, client interface{}) error {
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
		return fmt.Errorf("resourcePrivateLocationRead: %w", err)
	}
	return resourceDataFromPrivateLocation(pl, d)
}

func resourcePrivateLocationDelete(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	err := client.(checkly.Client).DeletePrivateLocation(ctx, d.Id())
	if err != nil {
		return fmt.Errorf("resourcePrivateLocationDelete: API error: %w", err)
	}
	return nil
}
