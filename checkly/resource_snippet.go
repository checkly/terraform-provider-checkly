package checkly

import (
	"fmt"
	"strconv"
	"strings"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceSnippet() *schema.Resource {
	return &schema.Resource{
		Create: resourceSnippetCreate,
		Read:   resourceSnippetRead,
		Update: resourceSnippetUpdate,
		Delete: resourceSnippetDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"script": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceSnippetCreate(d *schema.ResourceData, client interface{}) error {
	snippet, err := snippetFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourceSnippetCreate: translation error: %w", err)
	}
	result, err := client.(*checkly.Client).CreateSnippet(snippet)
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
	snippet, err := client.(*checkly.Client).GetSnippet(ID)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			//if resource is deleted remotely, then mark it as
			//successfully gone by unsetting it's ID
			d.SetId("")
			return nil
		}
		return fmt.Errorf("resourceSnippetRead: API error: %w", err)
	}
	return resourceDataFromSnippet(&snippet, d)
}

func resourceSnippetUpdate(d *schema.ResourceData, client interface{}) error {
	snippet, err := snippetFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourceSnippetUpdate: translation error: %w", err)
	}
	_, err = client.(*checkly.Client).UpdateSnippet(snippet.ID, snippet)
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
	err = client.(*checkly.Client).DeleteSnippet(ID)
	if err != nil {
		return fmt.Errorf("resourceSnippetDelete: API error: %w", err)
	}
	return nil
}
