package tailscale

import (
	"context"
	"strings"

	"github.com/davidsbond/terraform-provider-tailscale/internal/tailscale"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
						"id": {
							Type:        schema.TypeString,
							Description: "The unique identifier of the device",
							Computed:    true,
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
	deviceMaps := make([]map[string]interface{}, 0)
	for _, device := range devices {
		if namePrefix != "" && !strings.HasPrefix(device.Name, namePrefix) {
			continue
		}

		deviceMaps = append(deviceMaps, map[string]interface{}{
			"name": device.Name,
			"id":   device.ID,
		})
	}

	if err = d.Set("devices", deviceMaps); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(createUUID())
	return nil
}
