package checkly

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	checkly "github.com/checkly/checkly-go-sdk"
)

func validateOptions(options []int) func(val interface{}, key string) (warns []string, errs []error) {
	return func(val interface{}, key string) (warns []string, errs []error) {
		v := val.(int)
		valid := false
		for _, i := range options {
			if v == i {
				valid = true
			}
		}
		if !valid {
			errs = append(errs, fmt.Errorf("%q must be one of %v, got: %d", key, options, v))
		}
		return warns, errs
	}
}

func resourceDashboard() *schema.Resource {
	return &schema.Resource{
		Create: resourceDashboardCreate,
		Read:   resourceDashboardRead,
		Update: resourceDashboardUpdate,
		Delete: resourceDashboardDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"custom_url": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "A subdomain name under 'checklyhq.com'. Needs to be unique across all users.",
			},
			"custom_domain": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     nil,
				Description: "A custom user domain, e.g. 'status.example.com'. See the docs on updating your DNS and SSL usage.",
			},
			"logo": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "A URL pointing to an image file to use for the dashboard logo.",
			},
			"favicon": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "A URL pointing to an image file to use as browser favicon.",
			},
			"link": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "A link to for the dashboard logo.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "HTML <meta> description for the dashboard.",
			},
			"header": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "A piece of text displayed at the top of your dashboard.",
			},
			"width": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "FULL",
				ValidateFunc: func(value interface{}, key string) (warns []string, errs []error) {
					full := "FULL"
					px960 := "960PX"
					v := value.(string)
					if v != full && v != px960 {
						errs = append(errs, fmt.Errorf("%q must  %s and  %s, got: %s", key, full, px960, v))
					}
					return warns, errs
				},
				Description: "Determines whether to use the full screen or focus in the center. Possible values `FULL` and `960PX`.",
			},
			"refresh_rate": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      60,
				ValidateFunc: validateOptions([]int{60, 300, 600}),
				Description:  "How often to refresh the dashboard in seconds. Possible values `60`, '300' and `600`.",
			},
			"paginate": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Determines if pagination is on or off.",
			},
			"checks_per_page": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     15,
				Description: "Determines how many checks to show per page.",
			},
			"pagination_rate": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      60,
				ValidateFunc: validateOptions([]int{30, 60, 300}),
				Description:  "How often to trigger pagination in seconds. Possible values `30`, `60` and `300`.",
			},
			"tags": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "A list of one or more tags that filter which checks to display on the dashboard.",
			},
			"hide_tags": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Show or hide the tags on the dashboard.",
			},
			"use_tags_and_operator": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Set when to use AND operator for fetching dashboard tags.",
			},
			"is_private": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Set your dashboard as private and generate key.",
			},
			// moving to TypeString here https://github.com/hashicorp/terraform-plugin-sdk/issues/792
			"key": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "The access key when the dashboard is private.",
			},
		},
	}
}

func dashboardFromResourceData(d *schema.ResourceData) (checkly.Dashboard, error) {
	a := checkly.Dashboard{
		CustomDomain:       d.Get("custom_domain").(string),
		CustomUrl:          d.Get("custom_url").(string),
		Logo:               d.Get("logo").(string),
		Favicon:            d.Get("favicon").(string),
		Link:               d.Get("link").(string),
		Description:        d.Get("description").(string),
		Header:             d.Get("header").(string),
		RefreshRate:        d.Get("refresh_rate").(int),
		Paginate:           d.Get("paginate").(bool),
		ChecksPerPage:      d.Get("checks_per_page").(int),
		PaginationRate:     d.Get("pagination_rate").(int),
		HideTags:           d.Get("hide_tags").(bool),
		Width:              d.Get("width").(string),
		UseTagsAndOperator: d.Get("use_tags_and_operator").(bool),
		IsPrivate:          d.Get("is_private").(bool),
		Tags:               stringsFromSet(d.Get("tags").(*schema.Set)),
	}

	fmt.Printf("%v", a)

	return a, nil
}

func resourceDataFromDashboard(s *checkly.Dashboard, d *schema.ResourceData) error {
	d.Set("custom_domain", s.CustomDomain)
	d.Set("custom_url", s.CustomUrl)
	d.Set("logo", s.Logo)
	d.Set("favicon", s.Favicon)
	d.Set("link", s.Link)
	d.Set("description", s.Description)
	d.Set("header", s.Header)
	d.Set("refresh_rate", s.RefreshRate)
	d.Set("paginate", s.Paginate)
	d.Set("checks_per_page", s.ChecksPerPage)
	d.Set("pagination_rate", s.PaginationRate)
	d.Set("hide_tags", s.HideTags)
	d.Set("tags", s.Tags)
	d.Set("width", s.Width)
	d.Set("use_tags_and_operator", s.UseTagsAndOperator)
	d.Set("is_private", s.IsPrivate)

	// if the dashboard is private, we either do nothing
	// or set the key to a new value if there is any
	if s.IsPrivate {
		if len(s.Keys) > 0 {
			d.Set("key", s.Keys[0].RawKey)
		}
	} else {
		// if the dashboard is public, remove the key
		d.Set("key", nil)
	}

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

	// we cannot take the detour through resourceDashboardRead since
	// we would not get the keys back from an additional GET call
	return resourceDataFromDashboard(result, d)
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

	// we cannot take the detour through resourceDashboardRead since
	// we would not get the keys back from an additional GET call
	return resourceDataFromDashboard(result, d)
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
