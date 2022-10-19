package checkly

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	checkly "github.com/checkly/checkly-go-sdk"
)

func resourceEnvironmentVariable() *schema.Resource {
	return &schema.Resource{
		Create: resourceEnvironmentVariableCreate,
		Read:   resourceEnvironmentVariableRead,
		Update: resourceEnvironmentVariableUpdate,
		Delete: resourceEnvironmentVariableDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"key": {
				Type:     schema.TypeString,
				Required: true,
			},
			"value": {
				Type:     schema.TypeString,
				Required: true,
			},
			"locked": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

func resourceEnvironmentVariableCreate(d *schema.ResourceData, client interface{}) error {
	envVar, err := environmentVariableFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourceEnvironmentVariableCreate: translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	_, err = client.(checkly.Client).CreateEnvironmentVariable(ctx, envVar)
	if err != nil {
		return fmt.Errorf("CreateEnvironmentVariable: API error: %w", err)
	}
	d.SetId(envVar.Key)
	return resourceEnvironmentVariableRead(d, client)
}

func environmentVariableFromResourceData(d *schema.ResourceData) (checkly.EnvironmentVariable, error) {
	return checkly.EnvironmentVariable{
		Key:    d.Get("key").(string),
		Value:  d.Get("value").(string),
		Locked: d.Get("locked").(bool),
	}, nil
}

func resourceDataFromEnvironmentVariable(s *checkly.EnvironmentVariable, d *schema.ResourceData) error {
	d.Set("key", s.Key)
	d.Set("value", s.Value)
	d.Set("locked", s.Locked)
	return nil
}

func resourceEnvironmentVariableRead(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	envVar, err := client.(checkly.Client).GetEnvironmentVariable(ctx, d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			//if resource is deleted remotely, then mark it as
			//successfully gone by unsetting it's ID
			d.SetId("")
			return nil
		}
		return fmt.Errorf("resourceEnvironmentVariableRead: API error: %w", err)
	}
	return resourceDataFromEnvironmentVariable(envVar, d)
}

func resourceEnvironmentVariableUpdate(d *schema.ResourceData, client interface{}) error {
	envVar, err := environmentVariableFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourceEnvironmentVariableUpdate: translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	_, err = client.(checkly.Client).UpdateEnvironmentVariable(ctx, d.Id(), envVar)
	if err != nil {
		return fmt.Errorf("resourceEnvironmentVariableUpdate: API error: %w", err)
	}

	return resourceEnvironmentVariableRead(d, client)
}

func resourceEnvironmentVariableDelete(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	err := client.(checkly.Client).DeleteEnvironmentVariable(ctx, d.Id())
	if err != nil {
		return fmt.Errorf("resourceEnvironmentVariableDelete: API error: %w", err)
	}
	return nil
}
