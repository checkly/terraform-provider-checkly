package checkly

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	checkly "github.com/checkly/checkly-go-sdk"
)

var statusPageV3ThemeValues = allowedValues[string]{
	{Value: "AUTO"},
	{Value: "DARK"},
	{Value: "LIGHT"},
}

// The server lowercases the URL before storing it, so accepting uppercase
// here would create a permanent diff between config and state.
var statusPageV3URLRegex = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

func resourceStatusPageV3() *schema.Resource {
	return &schema.Resource{
		Create: resourceStatusPageV3Create,
		Read:   resourceStatusPageV3Read,
		Update: resourceStatusPageV3Update,
		Delete: resourceStatusPageV3Delete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "Checkly status pages allow you to easily communicate " +
			"the uptime and health of your applications and services to your " +
			"customers. A v3 status page has no cards or services: its " +
			"structure is managed with `checkly_status_page_v3_component` " +
			"resources, and incidents can be automated with " +
			"`checkly_status_page_v3_automation_rule` resources. The " +
			"`checkly_status_page_v3` resource should always be preferred " +
			"over the old `checkly_status_page` resource.",
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the status page.",
			},
			"url": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The unique subdomain of the status page: lowercase alphanumeric characters and dashes, no leading or trailing dash, at most 63 characters.",
				ValidateFunc: func(value interface{}, key string) (warns []string, errs []error) {
					v := value.(string)
					if !statusPageV3URLRegex.MatchString(v) {
						errs = append(errs, fmt.Errorf("%q can only contain lowercase alphanumeric characters and dashes, with no leading or trailing dash, got: %s", key, v))
					}
					return warns, errs
				},
			},
			"custom_domain": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A custom user domain, e.g. \"status.example.com\". See the docs on updating your DNS and SSL usage.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A short description shown on the status page.",
			},
			"logo": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A URL to an image file to use as the logo for the status page.",
			},
			"logo_dark": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A URL to an image file to use as the logo in dark mode.",
			},
			"redirect_to": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The URL the user should be redirected to when clicking the logo.",
			},
			"favicon": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A URL to an image file to use as the favicon of the status page.",
			},
			"default_theme": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "AUTO",
				Description:  "The default theme of the status page. " + statusPageV3ThemeValues.String(),
				ValidateFunc: validateOneOf(statusPageV3ThemeValues.Values()),
			},
			"privacy_policy_link": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A link to your privacy policy, shown in the page footer.",
			},
			"terms_of_service_link": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A link to your terms of service, shown in the page footer.",
			},
			"footer_text": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Free-form footer text.",
			},
			"google_analytics_tag": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A Google Analytics tag ID (e.g. \"G-XXXXXXXXXX\") to embed on the public page.",
			},
			"allow_indexing": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether search engines may index the public page. (Default `true`).",
			},
		},
	}
}

func statusPageV3FromResourceData(d *schema.ResourceData) checkly.StatusPageV3 {
	return checkly.StatusPageV3{
		ID:                 d.Id(),
		Name:               d.Get("name").(string),
		URL:                d.Get("url").(string),
		CustomDomain:       d.Get("custom_domain").(string),
		Description:        d.Get("description").(string),
		Logo:               d.Get("logo").(string),
		LogoDark:           d.Get("logo_dark").(string),
		RedirectTo:         d.Get("redirect_to").(string),
		Favicon:            d.Get("favicon").(string),
		DefaultTheme:       checkly.StatusPageTheme(d.Get("default_theme").(string)),
		PrivacyPolicyLink:  d.Get("privacy_policy_link").(string),
		TermsOfServiceLink: d.Get("terms_of_service_link").(string),
		FooterText:         d.Get("footer_text").(string),
		GoogleAnalyticsTag: d.Get("google_analytics_tag").(string),
		AllowIndexing:      d.Get("allow_indexing").(bool),
	}
}

func resourceDataFromStatusPageV3(p *checkly.StatusPageV3, d *schema.ResourceData) error {
	pairs := []struct {
		key   string
		value interface{}
	}{
		{"name", p.Name},
		{"url", p.URL},
		{"custom_domain", p.CustomDomain},
		{"description", p.Description},
		{"logo", p.Logo},
		{"logo_dark", p.LogoDark},
		{"redirect_to", p.RedirectTo},
		{"favicon", p.Favicon},
		{"default_theme", string(p.DefaultTheme)},
		{"privacy_policy_link", p.PrivacyPolicyLink},
		{"terms_of_service_link", p.TermsOfServiceLink},
		{"footer_text", p.FooterText},
		{"google_analytics_tag", p.GoogleAnalyticsTag},
		{"allow_indexing", p.AllowIndexing},
	}
	for _, pair := range pairs {
		if err := d.Set(pair.key, pair.value); err != nil {
			return fmt.Errorf("failed to set %q: %w", pair.key, err)
		}
	}
	return nil
}

func resourceStatusPageV3Create(d *schema.ResourceData, client interface{}) error {
	statusPage := statusPageV3FromResourceData(d)
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	result, err := client.(checkly.Client).CreateStatusPageV3(ctx, statusPage)
	if err != nil {
		return fmt.Errorf("CreateStatusPageV3: API error: %w", err)
	}
	d.SetId(result.ID)
	return resourceStatusPageV3Read(d, client)
}

func resourceStatusPageV3Read(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	statusPage, err := client.(checkly.Client).GetStatusPageV3(ctx, d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			// If the resource was deleted remotely, mark it as successfully
			// gone by unsetting its ID.
			d.SetId("")
			return nil
		}
		return fmt.Errorf("resourceStatusPageV3Read: API error: %w", err)
	}
	return resourceDataFromStatusPageV3(statusPage, d)
}

func resourceStatusPageV3Update(d *schema.ResourceData, client interface{}) error {
	statusPage := statusPageV3FromResourceData(d)
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	_, err := client.(checkly.Client).UpdateStatusPageV3(ctx, statusPage.ID, statusPage)
	if err != nil {
		return fmt.Errorf("resourceStatusPageV3Update: API error: %w", err)
	}
	return resourceStatusPageV3Read(d, client)
}

func resourceStatusPageV3Delete(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	err := client.(checkly.Client).DeleteStatusPageV3(ctx, d.Id())
	if err != nil {
		return fmt.Errorf("resourceStatusPageV3Delete: API error: %w", err)
	}
	return nil
}
