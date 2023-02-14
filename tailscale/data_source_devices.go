package tailscale

import (
	"context"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/tailscale/tailscale-client-go/tailscale"
)

func dataSourceDevices() *schema.Resource {
	return &schema.Resource{
		Description: "The devices data source describes a list of devices in a tailnet",
		ReadContext: dataSourceDevicesRead,
		Schema: map[string]*schema.Schema{
			"name_prefix": {
				Optional:    true,
				Type:        schema.TypeString,
				Description: "Filters the device list to elements whose name has the provided prefix",
			},
			"name_regexp": {
				Optional:    true,
				Type:        schema.TypeString,
				Description: "Filters the device list to elements whose name matches the provided regexp",
			},
			"devices": {
				Computed:    true,
				Type:        schema.TypeList,
				Description: "The list of devices in the tailnet",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Description: "The name of the device",
							Computed:    true,
						},
						"user": {
							Type:        schema.TypeString,
							Description: "The user associated with the device",
							Computed:    true,
						},
						"id": {
							Type:        schema.TypeString,
							Description: "The unique identifier of the device",
							Computed:    true,
						},
						"addresses": {
							Computed:    true,
							Type:        schema.TypeList,
							Description: "The list of device's IPs",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"tags": {
							Type:        schema.TypeSet,
							Description: "The tags applied to the device",
							Computed:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceDevicesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	devices, err := client.Devices(ctx)
	if err != nil {
		return diagnosticsError(err, "Failed to fetch devices")
	}

	namePrefix, _ := d.Get("name_prefix").(string)
	nameRegexp, _ := d.Get("name_regexp").(string)
	deviceMaps := make([]map[string]interface{}, 0)
	for _, device := range devices {
		if namePrefix != "" && !strings.HasPrefix(device.Name, namePrefix) {
			continue
		}
		if nameRegexp != "" {
			re, err := regexp.Compile(nameRegexp)
			if err != nil {
				return diag.FromErr(err)
			}
			if !re.MatchString(device.Name) {
				continue
			}
		}

		deviceMaps = append(deviceMaps, map[string]interface{}{
			"addresses": device.Addresses,
			"name":      device.Name,
			"user":      device.User,
			"id":        device.ID,
			"tags":      device.Tags,
		})
	}

	if err = d.Set("devices", deviceMaps); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(createUUID())
	return nil
}
