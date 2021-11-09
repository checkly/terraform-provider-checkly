package checkly

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	checkly "github.com/checkly/checkly-go-sdk"
)

func resourceDashboard() *schema.Resource {
	return &schema.Resource{
		Create: resourceDashboardCreate,
		Read:   resourceDashboardRead,
		Update: resourceDashboardUpdate,
		Delete: resourceDashboardDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"custom_url": {
				Type:     schema.TypeString,
				Required: true,
			},
			"custom_domain": {
				Type:     schema.TypeString,
				Required: true,
			},
			"logo": {
				Type:     schema.TypeString,
				Required: true,
			},
			"header": {
				Type:     schema.TypeString,
				Required: true,
			},
			"width": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"refresh_rate": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"paginate": {
				Type:     schema.TypeBool,
				Required: true,
			},
			"pagination_rate": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"tags": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"hide_tags": {
				Type:     schema.TypeBool,
				Required: true,
			},
		},
	}
}

func dashboardFromResourceData(d *schema.ResourceData) (checkly.Dashboard, error) {
	a := checkly.Dashboard{
		CustomDomain:   d.Get("custom_domain").(string),
		CustomUrl:      d.Get("custom_url").(string),
		Logo:           d.Get("logo").(string),
		Header:         d.Get("header").(string),
		RefreshRate:    d.Get("refresh_rate").(int),
		Paginate:       d.Get("paginate").(bool),
		PaginationRate: d.Get("pagination_rate").(int),
		HideTags:       d.Get("hide_tags").(bool),
		Width:          d.Get("width").(string),
		Tags:           stringsFromSet(d.Get("tags").(*schema.Set)),
	}

	fmt.Printf("%v", a)

	return a, nil
}

func resourceDataFromDashboard(s *checkly.Dashboard, d *schema.ResourceData) error {
	d.Set("custom_domain", s.CustomDomain)
	d.Set("custom_url", s.CustomUrl)
	d.Set("logo", s.Logo)
	d.Set("header", s.Header)
	d.Set("refresh_rate", s.RefreshRate)
	d.Set("paginate", s.Paginate)
	d.Set("pagination_rate", s.PaginationRate)
	d.Set("hide_tags", s.HideTags)
	d.Set("tags", s.Tags)
	d.Set("width", s.Width)
	return nil
}

func resourceDashboardCreate(d *schema.ResourceData, client interface{}) error {
	dashboard, err := dashboardFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourceDashboardCreate: translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	result, err := client.(checkly.Client).CreateDashboard(ctx, dashboard)

	if err != nil {
		return fmt.Errorf("CreateDashboard: API error: %w", err)
	}

	d.SetId(result.DashboardID)
	return resourceDashboardRead(d, client)
}

func resourceDashboardUpdate(d *schema.ResourceData, client interface{}) error {
	dashboard, err := dashboardFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourceDashboardUpdate: translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	result, err := client.(checkly.Client).UpdateDashboard(ctx, d.Id(), dashboard)
	if err != nil {
		return fmt.Errorf("resourceDashboardUpdate: API error: %w", err)
	}
	d.SetId(result.DashboardID)
	return resourceDashboardRead(d, client)
}

func resourceDashboardDelete(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	err := client.(checkly.Client).DeleteDashboard(ctx, d.Id())
	if err != nil {
		return fmt.Errorf("resourceDashboardDelete: API error: %w", err)
	}
	return nil
}

func resourceDashboardRead(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	dashboard, err := client.(checkly.Client).GetDashboard(ctx, d.Id())
	defer cancel()
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("resourceDashboardRead: API error: %w", err)
	}
	return resourceDataFromDashboard(dashboard, d)
}
