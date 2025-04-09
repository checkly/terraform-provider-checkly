package checkly

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	checkly "github.com/checkly/checkly-go-sdk"
)

func resourceStatusPage() *schema.Resource {
	return &schema.Resource{
		Create: resourceStatusPageCreate,
		Read:   resourceStatusPageRead,
		Update: resourceStatusPageUpdate,
		Delete: resourceStatusPageDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "Checkly status pages allow you to easily communicate " +
			"the uptime and health of your applications and services to your customers.",
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the status page.",
			},
			"url": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The URL of the status page.",
			},
			"custom_domain": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A custom user domain, e.g. \"status.example.com\". See the docs on updating your DNS and SSL usage.",
				ValidateFunc: func(value interface{}, key string) (warns []string, errs []error) {
					unsupportedSuffixes := []string{
						"example.com",
						"example.net",
						"example.org",
					}
					v := value.(string)
					for _, suffix := range unsupportedSuffixes {
						if strings.HasSuffix(strings.ToLower(v), suffix) || strings.EqualFold(v, suffix) {
							errs = append(errs, fmt.Errorf("custom domains ending in %s are not supported", suffix))
							break
						}
					}
					return warns, errs
				},
			},
			"logo": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A URL to an image file to use as the logo for the status page.",
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
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Possible values are `AUTO`, `DARK`, and `LIGHT`. (Default `AUTO`).",
				Default:     "AUTO",
				ValidateFunc: func(value interface{}, key string) (warns []string, errs []error) {
					v := value.(string)
					isValid := false
					options := []string{"AUTO", "DARK", "LIGHT"}
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
			},
			"card": {
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				Description: "A list of cards to include on the status page.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The name of the card.",
						},
						"service_attachment": {
							Type:        schema.TypeList,
							Required:    true,
							MinItems:    1,
							Description: "A list of services to attach to the card.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"service_id": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "The ID of the service.",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func resourceStatusPageCreate(d *schema.ResourceData, client interface{}) error {
	statusPage, err := statusPageFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourceStatusPageCreate: translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	result, err := client.(checkly.Client).CreateStatusPage(ctx, statusPage)
	if err != nil {
		return fmt.Errorf("CreateStatusPage: API error: %w", err)
	}
	d.SetId(result.ID)
	return resourceStatusPageRead(d, client)
}

func statusPageFromResourceData(d *schema.ResourceData) (checkly.StatusPage, error) {
	return checkly.StatusPage{
		ID:           d.Id(),
		Name:         d.Get("name").(string),
		URL:          d.Get("url").(string),
		CustomDomain: d.Get("custom_domain").(string),
		Logo:         d.Get("logo").(string),
		RedirectTo:   d.Get("redirect_to").(string),
		Favicon:      d.Get("favicon").(string),
		DefaultTheme: checkly.StatusPageTheme(d.Get("default_theme").(string)),
		Cards:        statusPageCardsFromList(d.Get("card").([]interface{})),
	}, nil
}

func statusPageCardsFromList(l []interface{}) []checkly.StatusPageCard {
	res := []checkly.StatusPageCard{}
	if len(l) == 0 {
		return res
	}
	for _, it := range l {
		tm := it.(tfMap)
		name := tm["name"].(string)
		serviceAttachments := statusPageCardServiceAttachmentsFromList(tm["service_attachment"].([]interface{}))
		res = append(res, checkly.StatusPageCard{
			Name:     name,
			Services: serviceAttachments,
		})
	}
	return res
}

func listFromStatusPageCards(cards []checkly.StatusPageCard) []tfMap {
	result := make([]tfMap, 0, len(cards))

	for _, card := range cards {
		result = append(result, tfMap{
			"name":               card.Name,
			"service_attachment": listFromStatusPageCardServiceAttachments(card.Services),
		})
	}

	return result
}

func statusPageCardServiceAttachmentsFromList(l []interface{}) []checkly.StatusPageService {
	res := []checkly.StatusPageService{}
	if len(l) == 0 {
		return res
	}
	for _, it := range l {
		tm := it.(tfMap)
		id := tm["service_id"].(string)
		res = append(res, checkly.StatusPageService{
			ID: id,
		})
	}
	return res
}

func listFromStatusPageCardServiceAttachments(attachments []checkly.StatusPageService) []tfMap {
	result := make([]tfMap, 0, len(attachments))

	for _, attachment := range attachments {
		result = append(result, tfMap{
			"service_id": attachment.ID,
		})
	}

	return result
}

func resourceDataFromStatusPage(p *checkly.StatusPage, d *schema.ResourceData) error {
	d.Set("name", p.Name)
	d.Set("url", p.URL)
	d.Set("custom_domain", p.CustomDomain)
	d.Set("logo", p.Logo)
	d.Set("redirect_to", p.RedirectTo)
	d.Set("favicon", p.Favicon)
	d.Set("default_theme", p.DefaultTheme)
	d.Set("card", listFromStatusPageCards(p.Cards))
	return nil
}

func resourceStatusPageRead(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	statusPage, err := client.(checkly.Client).GetStatusPage(ctx, d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			//if resource is deleted remotely, then mark it as
			//successfully gone by unsetting it's ID
			d.SetId("")
			return nil
		}
		return fmt.Errorf("resourceStatusPageRead: API error: %w", err)
	}
	return resourceDataFromStatusPage(statusPage, d)
}

func resourceStatusPageUpdate(d *schema.ResourceData, client interface{}) error {
	statusPage, err := statusPageFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourceStatusPageUpdate: translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	_, err = client.(checkly.Client).UpdateStatusPage(ctx, statusPage.ID, statusPage)
	if err != nil {
		return fmt.Errorf("resourceStatusPageUpdate: API error: %w", err)
	}
	d.SetId(statusPage.ID)
	return resourceStatusPageRead(d, client)
}

func resourceStatusPageDelete(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	err := client.(checkly.Client).DeleteStatusPage(ctx, d.Id())
	if err != nil {
		return fmt.Errorf("resourceStatusPageDelete: API error: %w", err)
	}
	return nil
}
