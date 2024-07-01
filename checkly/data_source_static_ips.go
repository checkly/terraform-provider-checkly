package checkly

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	checkly "github.com/checkly/checkly-go-sdk"
)

func dataSourceStaticIPs() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceStaticIPsRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of the static IPs data source.",
			},
			"addresses": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Static IP addresses for Checkly's runner infrastructure.",
			},
			"locations": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Description: "Specify the locations you want to get static IPs for.",
			},
			"ip_family": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Specify the IP families you want to get static IPs for. Only `IPv6` or `IPv4` are valid options.",
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					value := val.(string)
					if !slices.Contains([]string{"IPv6", "IPv4"}, value) {
						errs = append(errs, fmt.Errorf("%q must be either \"IPv6\" or \"IPv4\"", key))
					}
					return
				},
			},
		},
	}
}

func dataSourceStaticIPsRead(d *schema.ResourceData, client interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout())
	staticIPs, err := client.(checkly.Client).GetStaticIPs(ctx)
	defer cancel()
	if err != nil {
		return fmt.Errorf("dateSourceStaticIPsRead: API error: %w", err)
	}
	return dataSourceFromStaticIPs(staticIPs, d)
}

func dataSourceFromStaticIPs(s []checkly.StaticIP, d *schema.ResourceData) error {
	var staticIPs []checkly.StaticIP
	var addresses []string

	locations := stringsFromSet(d.Get("locations").(*schema.Set))
	ip_family := d.Get("ip_family").(string)

	// only locations is set for filtering
	if len(locations) > 0 && ip_family == "" {
		for _, ip := range s {
			if slices.Contains(locations, ip.Region) {
				staticIPs = append(staticIPs, ip)
			}
		}
		// only ip_family is set for filtering
	} else if ip_family != "" && len(locations) == 0 {
		for _, ip := range s {
			if ip_family == "IPv4" && ip.Address.Addr().Is4() {
				staticIPs = append(staticIPs, ip)
			} else if ip_family == "IPv6" && ip.Address.Addr().Is6() {
				staticIPs = append(staticIPs, ip)
			}
		}
		// both region and ip_family are set for filtering
	} else if len(locations) > 0 && ip_family != "" {
		for _, ip := range s {
			if ip_family == "IPv4" && ip.Address.Addr().Is4() && slices.Contains(locations, ip.Region) {
				staticIPs = append(staticIPs, ip)
			} else if ip_family == "IPv6" && ip.Address.Addr().Is6() && slices.Contains(locations, ip.Region) {
				staticIPs = append(staticIPs, ip)
			}
		}
		// no region nor ip_family filters set
	} else {
		staticIPs = s
	}

	for _, ip := range staticIPs {
		addresses = append(addresses, ip.Address.String())
	}

	d.Set("addresses", addresses)
	d.SetId("checkly_static_ips_data_source_id")

	return nil
}
