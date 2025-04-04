package checkly

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	checkly "github.com/checkly/checkly-go-sdk"
)

func resourceStatusPageService() *schema.Resource {
	return &schema.Resource{
		Create: resourceStatusPageServiceCreate,
		Read:   resourceStatusPageServiceRead,
		Update: resourceStatusPageServiceUpdate,
		Delete: resourceStatusPageServiceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the service.",
			},
		},
	}
}

func resourceStatusPageServiceCreate(d *schema.ResourceData, client interface{}) error {
	service, err := statusPageServiceFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourceStatusPageServiceCreate: translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	result, err := client.(checkly.Client).CreateStatusPageService(ctx, service)
	if err != nil {
		return fmt.Errorf("CreateStatusPageService: API error: %w", err)
	}
	d.SetId(result.ID)
	return resourceStatusPageServiceRead(d, client)
}

func statusPageServiceFromResourceData(d *schema.ResourceData) (checkly.StatusPageService, error) {
	return checkly.StatusPageService{
		ID:   d.Id(),
		Name: d.Get("name").(string),
	}, nil
}

func resourceDataFromStatusPageService(s *checkly.StatusPageService, d *schema.ResourceData) error {
	d.Set("name", s.Name)
	return nil
}

func resourceStatusPageServiceRead(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	service, err := client.(checkly.Client).GetStatusPageService(ctx, d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			//if resource is deleted remotely, then mark it as
			//successfully gone by unsetting it's ID
			d.SetId("")
			return nil
		}
		return fmt.Errorf("resourceStatusPageServiceRead: API error: %w", err)
	}
	return resourceDataFromStatusPageService(service, d)
}

func resourceStatusPageServiceUpdate(d *schema.ResourceData, client interface{}) error {
	service, err := statusPageServiceFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourceStatusPageServiceUpdate: translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	_, err = client.(checkly.Client).UpdateStatusPageService(ctx, service.ID, service)
	if err != nil {
		return fmt.Errorf("resourceStatusPageServiceUpdate: API error: %w", err)
	}
	d.SetId(service.ID)
	return resourceStatusPageServiceRead(d, client)
}

func resourceStatusPageServiceDelete(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	err := client.(checkly.Client).DeleteStatusPageService(ctx, d.Id())
	if err != nil {
		return fmt.Errorf("resourceStatusPageServiceDelete: API error: %w", err)
	}
	return nil
}
