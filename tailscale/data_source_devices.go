// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
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
							Description: "The full name of the device (e.g. `hostname.domain.ts.net`)",
							Computed:    true,
						},
						"hostname": {
							Type:        schema.TypeString,
							Description: "The short hostname of the device",
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
	client := m.(*tsclient.Client)

	devices, err := client.Devices().List(ctx)
	if err != nil {
		return diagnosticsError(err, "Failed to fetch devices")
	}

	namePrefix, _ := d.Get("name_prefix").(string)
	deviceMaps := make([]map[string]interface{}, 0)
	for _, device := range devices {
		if namePrefix != "" && !strings.HasPrefix(device.Name, namePrefix) {
			continue
		}

		m := deviceToMap(&device)
		m["id"] = device.ID
		deviceMaps = append(deviceMaps, m)
	}

	if err = d.Set("devices", deviceMaps); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(createUUID())
	return nil
}
