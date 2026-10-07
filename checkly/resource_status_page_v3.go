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

// isStatusPageV3NotFound reports whether a v3 status page API call failed
// with 404. It matches the SDK's status prefix rather than any "404" in the
// message, because the quoted response body can echo user-chosen names.
func isStatusPageV3NotFound(err error) bool {
	return strings.HasPrefix(err.Error(), "unexpected response status 404:")
}

// Same format the API accepts: #RGB or #RRGGBB, case-insensitive.
var statusPageV3HexColorRegex = regexp.MustCompile(`^#([0-9A-Fa-f]{3}|[0-9A-Fa-f]{6})$`)

// The twelve colors of one theme, in the order the API documents them.
// `field` points into a group so the same table drives the schema, the
// expand and the flatten.
var statusPageV3ThemeColorAttributes = []struct {
	key         string
	description string
	field       func(group *checkly.StatusPageV3ThemeColorGroup) *string
}{
	{"body_background_color", "The background of the page.",
		func(g *checkly.StatusPageV3ThemeColorGroup) *string { return &g.BodyBackgroundColor }},
	{"header_background_color", "The background of the page header.",
		func(g *checkly.StatusPageV3ThemeColorGroup) *string { return &g.HeaderBackgroundColor }},
	{"header_font_color", "The color of text in the page header.",
		func(g *checkly.StatusPageV3ThemeColorGroup) *string { return &g.HeaderFontColor }},
	{"title_font_color", "The color of titles and headings.",
		func(g *checkly.StatusPageV3ThemeColorGroup) *string { return &g.TitleFontColor }},
	{"body_font_color", "The color of regular body text.",
		func(g *checkly.StatusPageV3ThemeColorGroup) *string { return &g.BodyFontColor }},
	{"body_font_color_muted", "The color of de-emphasized body text, such as timestamps.",
		func(g *checkly.StatusPageV3ThemeColorGroup) *string { return &g.BodyFontColorMuted }},
	{"navigation_font_color", "The color of navigation links.",
		func(g *checkly.StatusPageV3ThemeColorGroup) *string { return &g.NavigationFontColor }},
	{"link_font_color", "The color of links in the page content.",
		func(g *checkly.StatusPageV3ThemeColorGroup) *string { return &g.LinkFontColor }},
	{"card_background_color", "The background of component and incident cards.",
		func(g *checkly.StatusPageV3ThemeColorGroup) *string { return &g.CardBackgroundColor }},
	{"border_color", "The color of borders and dividers.",
		func(g *checkly.StatusPageV3ThemeColorGroup) *string { return &g.BorderColor }},
	{"primary_button_background_color", "The background of primary buttons, such as \"Subscribe\".",
		func(g *checkly.StatusPageV3ThemeColorGroup) *string { return &g.PrimaryButtonBackgroundColor }},
	{"primary_button_font_color", "The color of text on primary buttons.",
		func(g *checkly.StatusPageV3ThemeColorGroup) *string { return &g.PrimaryButtonFontColor }},
}

func validateStatusPageV3HexColor(value interface{}, key string) (warns []string, errs []error) {
	v := value.(string)
	if !statusPageV3HexColorRegex.MatchString(v) {
		errs = append(errs, fmt.Errorf("%q must be a hex color such as \"#FF0000\" or \"#F00\", got: %s", key, v))
	}
	return warns, errs
}

// The API requires the complete palette when custom colors are given, so
// every color is required rather than defaulted here: the defaults belong
// to the backend and may change.
func statusPageV3ThemeColorGroupSchema(theme string) *schema.Schema {
	colors := make(map[string]*schema.Schema, len(statusPageV3ThemeColorAttributes))
	for _, attr := range statusPageV3ThemeColorAttributes {
		colors[attr.key] = &schema.Schema{
			Type:         schema.TypeString,
			Required:     true,
			Description:  attr.description + " A hex color such as \"#FF0000\" or \"#F00\".",
			ValidateFunc: validateStatusPageV3HexColor,
		}
	}
	return &schema.Schema{
		Type:        schema.TypeList,
		Required:    true,
		MaxItems:    1,
		Description: "The colors used when the page renders in " + theme + " mode.",
		Elem:        &schema.Resource{Schema: colors},
	}
}

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
			"support_link": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A support contact link (http, https or mailto), shown in the page footer.",
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
			"theme_colors": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Description: "Custom colors for the light and dark theme of the page. Requires custom " +
					"theme colors to be part of your plan. The complete palette must be given: both " +
					"`light` and `dark`, each with every color. Leave the block out to use Checkly's " +
					"default colors; the default palette the API then reports is not tracked, and the " +
					"block is not populated when importing a page.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"light": statusPageV3ThemeColorGroupSchema("light"),
						"dark":  statusPageV3ThemeColorGroupSchema("dark"),
					},
				},
			},
		},
	}
}

func statusPageV3ThemeColorGroupFromList(value interface{}) checkly.StatusPageV3ThemeColorGroup {
	var group checkly.StatusPageV3ThemeColorGroup
	list, ok := value.([]interface{})
	if !ok || len(list) == 0 || list[0] == nil {
		return group
	}
	colors := list[0].(map[string]interface{})
	for _, attr := range statusPageV3ThemeColorAttributes {
		if color, ok := colors[attr.key].(string); ok {
			*attr.field(&group) = color
		}
	}
	return group
}

func statusPageV3ThemeColorsFromResourceData(d *schema.ResourceData) *checkly.StatusPageV3ThemeColors {
	blocks, ok := d.Get("theme_colors").([]interface{})
	if !ok || len(blocks) == 0 || blocks[0] == nil {
		return nil
	}
	block := blocks[0].(map[string]interface{})
	return &checkly.StatusPageV3ThemeColors{
		Light: statusPageV3ThemeColorGroupFromList(block["light"]),
		Dark:  statusPageV3ThemeColorGroupFromList(block["dark"]),
	}
}

func statusPageV3ThemeColorGroupToList(group *checkly.StatusPageV3ThemeColorGroup) []interface{} {
	colors := make(map[string]interface{}, len(statusPageV3ThemeColorAttributes))
	for _, attr := range statusPageV3ThemeColorAttributes {
		colors[attr.key] = *attr.field(group)
	}
	return []interface{}{colors}
}

func statusPageV3ThemeColorsToList(themeColors *checkly.StatusPageV3ThemeColors) []interface{} {
	if themeColors == nil {
		return nil
	}
	return []interface{}{map[string]interface{}{
		"light": statusPageV3ThemeColorGroupToList(&themeColors.Light),
		"dark":  statusPageV3ThemeColorGroupToList(&themeColors.Dark),
	}}
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
		SupportLink:        d.Get("support_link").(string),
		FooterText:         d.Get("footer_text").(string),
		GoogleAnalyticsTag: d.Get("google_analytics_tag").(string),
		AllowIndexing:      d.Get("allow_indexing").(bool),
		ThemeColors:        statusPageV3ThemeColorsFromResourceData(d),
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
		{"support_link", p.SupportLink},
		{"footer_text", p.FooterText},
		{"google_analytics_tag", p.GoogleAnalyticsTag},
		{"allow_indexing", p.AllowIndexing},
	}
	for _, pair := range pairs {
		if err := d.Set(pair.key, pair.value); err != nil {
			return fmt.Errorf("failed to set %q: %w", pair.key, err)
		}
	}
	// The API always reports a palette, the defaults when none is set, so
	// the colors are only tracked when configured. Tracking them regardless
	// would make every plan of an unconfigured page try to remove them.
	if _, configured := d.GetOk("theme_colors"); configured {
		if err := d.Set("theme_colors", statusPageV3ThemeColorsToList(p.ThemeColors)); err != nil {
			return fmt.Errorf("failed to set %q: %w", "theme_colors", err)
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
		if isStatusPageV3NotFound(err) {
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
		// Already gone (deleted out-of-band): nothing left to do.
		if isStatusPageV3NotFound(err) {
			return nil
		}
		return fmt.Errorf("resourceStatusPageV3Delete: API error: %w", err)
	}
	return nil
}
