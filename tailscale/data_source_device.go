package tailscale

import (
	"context"

	"github.com/davidsbond/terraform-provider-tailscale/internal/tailscale"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDevice() *schema.Resource {
	return &schema.Resource{
		Description: "The device data source describes a single device in a tailnet",
		ReadContext: dataSourceDeviceRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the device",
			},
			"user": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The user associated with the device",
			},
			"addresses": {
				Type:        schema.TypeList,
				Description: "The list of device's IPs",
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataSourceDeviceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	name := d.Get("name").(string)

	devices, err := client.Devices(ctx)
	if err != nil {
		return diagnosticsError(err, "Failed to fetch devices")
	}

	var selected *tailscale.Device
	for _, device := range devices {
		if device.Name != name {
			continue
		}

		selected = &device
		break
	}

	if selected == nil {
		return diag.Errorf("Could not find device with name %s", name)
	}

	d.SetId(selected.ID)
	d.Set("user", selected.User)
	d.Set("addresses", selected.Addresses)
	return nil
}
