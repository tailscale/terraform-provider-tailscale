package tailscale

import (
	"context"

	"github.com/davidsbond/tailscale-client-go/tailscale"
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
				Description: "The user associated with the device",
				Computed:    true,
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

	if err = d.Set("user", selected.User); err != nil {
		return diagnosticsError(err, "Failed to set user")
	}

	if err = d.Set("addresses", selected.Addresses); err != nil {
		return diagnosticsError(err, "Failed to set addresses")
	}
	return nil
}
