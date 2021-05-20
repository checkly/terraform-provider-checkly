package checkly

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	checkly "github.com/checkly/checkly-go-sdk"
)

func resourceEnvVar() *schema.Resource {
	return &schema.Resource{
		Create: resourceEnvVarCreate,
		Read:   resourceEnvVarRead,
		Update: resourceEnvVarUpdate,
		Delete: resourceEnvVarDelete,
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
		},
	}
}

func resourceEnvVarCreate(d *schema.ResourceData, client interface{}) error {
	envVar, err := envVarFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourceEnvVarCreate: translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	_, err = client.(checkly.Client).CreateEnvironmentVariable(ctx, envVar)
	if err != nil {
		return fmt.Errorf("CreateEnvVar: API error: %w", err)
	}
	d.SetId(envVar.Key)
	return resourceEnvVarRead(d, client)
}

func envVarFromResourceData(d *schema.ResourceData) (checkly.EnvironmentVariable, error) {
	return checkly.EnvironmentVariable{
		Key:   d.Get("key").(string),
		Value: d.Get("value").(string),
	}, nil
}

func resourceDataFromEnvVar(s *checkly.EnvironmentVariable, d *schema.ResourceData) error {
	d.Set("key", s.Key)
	d.Set("value", s.Value)
	return nil
}

func resourceEnvVarRead(d *schema.ResourceData, client interface{}) error {
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
		return fmt.Errorf("resourceEnvVarRead: API error: %w", err)
	}
	return resourceDataFromEnvVar(envVar, d)
}

func resourceEnvVarUpdate(d *schema.ResourceData, client interface{}) error {
	envVar, err := envVarFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourceEnvVarUpdate: translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	_, err = client.(checkly.Client).UpdateEnvironmentVariable(ctx, envVar.Key, envVar)
	if err != nil {
		return fmt.Errorf("resourceEnvVarUpdate: API error: %w", err)
	}
	d.SetId(envVar.Key)
	return resourceEnvVarRead(d, client)
}

func resourceEnvVarDelete(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	err := client.(checkly.Client).DeleteEnvironmentVariable(ctx, d.Id())
	if err != nil {
		return fmt.Errorf("resourceEnvVarDelete: API error: %w", err)
	}
	return nil
}
