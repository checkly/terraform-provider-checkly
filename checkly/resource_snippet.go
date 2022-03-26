package checkly

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	checkly "github.com/checkly/checkly-go-sdk"
)

func resourceSnippet() *schema.Resource {
	return &schema.Resource{
		Create: resourceSnippetCreate,
		Read:   resourceSnippetRead,
		Update: resourceSnippetUpdate,
		Delete: resourceSnippetDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the snippet",
			},
			"script": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Your Node.js code that interacts with the API check lifecycle, or functions as a partial for browser checks.",
			},
		},
	}
}

func resourceSnippetCreate(d *schema.ResourceData, client interface{}) error {
	snippet, err := snippetFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourceSnippetCreate: translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	result, err := client.(checkly.Client).CreateSnippet(ctx, snippet)
	if err != nil {
		return fmt.Errorf("CreateSnippet: API error: %w", err)
	}
	d.SetId(fmt.Sprintf("%d", result.ID))
	return resourceSnippetRead(d, client)
}

func snippetFromResourceData(d *schema.ResourceData) (checkly.Snippet, error) {
	id, err := resourceIDToInt(d.Id())
	if err != nil {
		return checkly.Snippet{}, err
	}
	return checkly.Snippet{
		ID:     id,
		Name:   d.Get("name").(string),
		Script: d.Get("script").(string),
	}, nil
}

func resourceDataFromSnippet(s *checkly.Snippet, d *schema.ResourceData) error {
	d.Set("name", s.Name)
	d.Set("script", s.Script)
	return nil
}

func resourceIDToInt(id string) (int64, error) {
	if id == "" {
		return 0, nil
	}
	res, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, err
	}
	return res, nil
}

func resourceSnippetRead(d *schema.ResourceData, client interface{}) error {
	ID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return fmt.Errorf("resourceSnippetRead: ID %s is not numeric: %w", d.Id(), err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	snippet, err := client.(checkly.Client).GetSnippet(ctx, ID)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			//if resource is deleted remotely, then mark it as
			//successfully gone by unsetting it's ID
			d.SetId("")
			return nil
		}
		return fmt.Errorf("resourceSnippetRead: API error: %w", err)
	}
	return resourceDataFromSnippet(snippet, d)
}

func resourceSnippetUpdate(d *schema.ResourceData, client interface{}) error {
	snippet, err := snippetFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourceSnippetUpdate: translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	_, err = client.(checkly.Client).UpdateSnippet(ctx, snippet.ID, snippet)
	if err != nil {
		return fmt.Errorf("resourceSnippetUpdate: API error: %w", err)
	}
	d.SetId(fmt.Sprintf("%d", snippet.ID))
	return resourceSnippetRead(d, client)
}

func resourceSnippetDelete(d *schema.ResourceData, client interface{}) error {
	ID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return fmt.Errorf("resourceSnippetDelete: ID %s is not numeric: %w", d.Id(), err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	err = client.(checkly.Client).DeleteSnippet(ctx, ID)
	if err != nil {
		return fmt.Errorf("resourceSnippetDelete: API error: %w", err)
	}
	return nil
}
