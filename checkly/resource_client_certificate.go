package checkly

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	checkly "github.com/checkly/checkly-go-sdk"
)

func resourceClientCertificate() *schema.Resource {
	return &schema.Resource{
		Create: resourceClientCertificateCreate,
		Read:   resourceClientCertificateRead,
		Delete: resourceClientCertificateDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "Use client certificates to authenticate your API " +
			"checks to APIs that require mutual TLS (mTLS) authentication, " +
			"or any other authentication scheme where the requester needs to " +
			"provide a certificate." +
			"\n\n" +
			"Each client certificate is specific to a domain name, e.g. " +
			"`acme.com` and will be used automatically by any API checks " +
			"targeting that domain." +
			"\n\n" +
			"Changing the value of any attribute forces a new resource to " +
			"be created.",
		Schema: map[string]*schema.Schema{
			"host": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The host domain that the certificate should be used for.",
			},
			"certificate": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The client certificate in PEM format.",
			},
			"private_key": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The private key for the certificate in PEM format.",
			},
			"passphrase": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Sensitive:   true,
				Description: "Passphrase for the private key.",
			},
			"trusted_ca": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "PEM formatted bundle of CA certificates that the client should trust. The bundle may contain many CA certificates.",
			},
		},
	}
}

func resourceClientCertificateCreate(d *schema.ResourceData, client interface{}) error {
	clientCertificate, err := clientCertificateFromResourceData(d)
	if err != nil {
		return fmt.Errorf("resourceClientCertificateCreate: translation error: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	result, err := client.(checkly.Client).CreateClientCertificate(ctx, clientCertificate)
	if err != nil {
		return fmt.Errorf("CreateClientCertificate: API error: %w", err)
	}
	d.SetId(result.ID)
	return resourceClientCertificateRead(d, client)
}

func clientCertificateFromResourceData(d *schema.ResourceData) (checkly.ClientCertificate, error) {
	return checkly.ClientCertificate{
		ID:          d.Id(),
		Host:        d.Get("host").(string),
		Certificate: d.Get("certificate").(string),
		PrivateKey:  d.Get("private_key").(string),
		Passphrase:  d.Get("passphrase").(string),
		TrustedCA:   d.Get("trusted_ca").(string),
	}, nil
}

func resourceDataFromClientCertificate(c *checkly.ClientCertificate, d *schema.ResourceData) error {
	d.Set("host", c.Host)
	d.Set("certificate", c.Certificate)
	d.Set("private_key", c.PrivateKey)
	d.Set("trusted_ca", c.TrustedCA)
	// The backend does not return a value for Passphrase and even it did, the
	// value would be encrypted. Thus, the value should never be modified.
	return nil
}

func resourceClientCertificateRead(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	clientCertificate, err := client.(checkly.Client).GetClientCertificate(ctx, d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			//if resource is deleted remotely, then mark it as
			//successfully gone by unsetting it's ID
			d.SetId("")
			return nil
		}
		return fmt.Errorf("resourceClientCertificateRead: API error: %w", err)
	}
	return resourceDataFromClientCertificate(clientCertificate, d)
}

func resourceClientCertificateDelete(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	defer cancel()
	err := client.(checkly.Client).DeleteClientCertificate(ctx, d.Id())
	if err != nil {
		return fmt.Errorf("resourceClientCertificateDelete: API error: %w", err)
	}
	return nil
}
