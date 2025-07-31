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
				Required:    true,
				Description: "A piece of text displayed at the top of your dashboard.",
			},
			"show_header": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Show or hide header and description on the dashboard. (Default `true`).",
			},
			"width": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "FULL",
				ValidateFunc: validateOneOf([]string{"FULL", "960PX"}),
				Description:  "Determines whether to use the full screen or focus in the center. Possible values are `FULL` and `960PX`. (Default `FULL`).",
			},
			"refresh_rate": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      60,
				ValidateFunc: validateOneOf([]int{60, 300, 600}),
				Description:  "How often to refresh the dashboard in seconds. Possible values `60`, '300' and `600`. (Default `60`).",
			},
			"paginate": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Determines if pagination is on or off. (Default `true`).",
			},
			"checks_per_page": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      15,
				ValidateFunc: validateBetween(1, 20),
				Description:  "Determines how many checks to show per page. Possible values are between 1 and 20. (Default `15`).",
			},
			"pagination_rate": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      60,
				ValidateFunc: validateOneOf([]int{30, 60, 300}),
				Description:  "How often to trigger pagination in seconds. Possible values `30`, `60` and `300`. (Default `60`).",
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
				Description: "Show or hide the tags on the dashboard. (Default `false`).",
			},
			"use_tags_and_operator": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Set when to use AND operator for fetching dashboard tags. (Default `false`).",
			},
			"enable_incidents": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enable incident support for the dashboard. (Default `false`).",
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
			"expand_checks": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Expand or collapse checks on the dashboard. (Default `false`).",
			},
			"show_check_run_links": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Show or hide check run links on the dashboard. (Default `false`).",
			},
			"custom_css": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Custom CSS to be applied to the dashboard.",
			},
			"show_p95": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Show or hide the P95 stats on the dashboard. (Default `true`).",
			},
			"show_p99": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Show or hide the P99 stats on the dashboard. (Default `true`).",
			},
		},
	}
}

func dashboardFromResourceData(d *schema.ResourceData) (checkly.Dashboard, error) {
	showHeader := d.Get("show_header").(bool)
	showP95 := d.Get("show_p95").(bool)
	showP99 := d.Get("show_p99").(bool)

	a := checkly.Dashboard{
		CustomDomain:       d.Get("custom_domain").(string),
		CustomUrl:          d.Get("custom_url").(string),
		Logo:               d.Get("logo").(string),
		Favicon:            d.Get("favicon").(string),
		Link:               d.Get("link").(string),
		Description:        d.Get("description").(string),
		Header:             d.Get("header").(string),
		ShowHeader:         &showHeader,
		RefreshRate:        d.Get("refresh_rate").(int),
		Paginate:           d.Get("paginate").(bool),
		ChecksPerPage:      d.Get("checks_per_page").(int),
		PaginationRate:     d.Get("pagination_rate").(int),
		HideTags:           d.Get("hide_tags").(bool),
		Width:              d.Get("width").(string),
		UseTagsAndOperator: d.Get("use_tags_and_operator").(bool),
		EnableIncidents:    d.Get("enable_incidents").(bool),
		IsPrivate:          d.Get("is_private").(bool),
		Tags:               stringsFromSet(d.Get("tags").(*schema.Set)),
		ExpandChecks:       d.Get("expand_checks").(bool),
		ShowCheckRunLinks:  d.Get("show_check_run_links").(bool),
		ShowP95:            &showP95,
		ShowP99:            &showP99,
		CustomCSS:          d.Get("custom_css").(string),
	}

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
	d.Set("show_header", s.ShowHeader)
	d.Set("refresh_rate", s.RefreshRate)
	d.Set("paginate", s.Paginate)
	d.Set("checks_per_page", s.ChecksPerPage)
	d.Set("pagination_rate", s.PaginationRate)
	d.Set("hide_tags", s.HideTags)
	d.Set("tags", s.Tags)
	d.Set("width", s.Width)
	d.Set("use_tags_and_operator", s.UseTagsAndOperator)
	d.Set("enable_incidents", s.EnableIncidents)
	d.Set("is_private", s.IsPrivate)
	d.Set("expand_checks", s.ExpandChecks)
	d.Set("show_check_run_links", s.ShowCheckRunLinks)
	d.Set("show_p95", s.ShowP95)
	d.Set("show_p99", s.ShowP99)
	d.Set("custom_css", s.CustomCSS)

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
