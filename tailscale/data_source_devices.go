// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"tailscale.com/client/tailscale/v2"
)

func dataSourceDevices() *schema.Resource {
	return &schema.Resource{
		Description: "The devices data source describes a list of devices in a tailnet",
		ReadContext: dataSourceDevicesRead,
		Schema: map[string]*schema.Schema{
			"filter": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The name must be a top-level device property, e.g. isEphemeral, tags, hostname, etc.",
						},
						"values": {
							Type:     schema.TypeSet,
							Required: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Description: "The list of values to filter for. Values are matched as exact matches.",
						},
					},
				},
				Description: "Filters the device list to elements devices whose fields match the provided values.",
			},
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
							Description: "The legacy identifier of the device. Use node_id instead for new resources.",
							Computed:    true,
						},
						"node_id": {
							Type:        schema.TypeString,
							Description: "The preferred indentifier for a device.",
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
						"authorized": {
							Type:        schema.TypeBool,
							Description: "Whether the device is authorized to access the tailnet",
							Computed:    true,
						},
						"key_expiry_disabled": {
							Type:        schema.TypeBool,
							Description: "Whether the device's key expiry is disabled",
							Computed:    true,
						},
						"blocks_incoming_connections": {
							Type:        schema.TypeBool,
							Description: "Whether the device blocks incoming connections",
							Computed:    true,
						},
						"client_version": {
							Type:        schema.TypeString,
							Description: "The Tailscale client version running on the device",
							Computed:    true,
						},
						"created": {
							Type:        schema.TypeString,
							Description: "The creation time of the device",
							Computed:    true,
						},
						"expires": {
							Type:        schema.TypeString,
							Description: "The expiry time of the device's key",
							Computed:    true,
						},
						"is_external": {
							Type:        schema.TypeBool,
							Description: "Whether the device is marked as external",
							Computed:    true,
						},
						"last_seen": {
							Type:        schema.TypeString,
							Description: "The last seen time of the device",
							Computed:    true,
						},
						"machine_key": {
							Type:        schema.TypeString,
							Description: "The machine key of the device",
							Computed:    true,
						},
						"node_key": {
							Type:        schema.TypeString,
							Description: "The node key of the device",
							Computed:    true,
						},
						"os": {
							Type:        schema.TypeString,
							Description: "The operating system of the device",
							Computed:    true,
						},
						"update_available": {
							Type:        schema.TypeBool,
							Description: "Whether an update is available for the device",
							Computed:    true,
						},
						"tailnet_lock_error": {
							Type:        schema.TypeString,
							Description: "The tailnet lock error for the device, if any",
							Computed:    true,
						},
						"tailnet_lock_key": {
							Type:        schema.TypeString,
							Description: "The tailnet lock key for the device, if any",
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

	opts := []tailscale.ListDevicesOptions{}
	if v, ok := d.GetOk("filter"); ok {
		filterConfigs := v.(*schema.Set).List()
		for _, f := range filterConfigs {
			m := f.(map[string]interface{})
			name := m["name"].(string)

			// Convert the Set of values to a slice of strings
			rawValues := m["values"].(*schema.Set).List()
			values := make([]string, len(rawValues))
			for i, val := range rawValues {
				values[i] = val.(string)
			}

			opts = append(opts, tailscale.WithFilter(name, values))
		}
	}

	devices, err := client.Devices().List(ctx, opts...)
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
