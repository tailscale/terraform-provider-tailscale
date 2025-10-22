// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"tailscale.com/client/tailscale/v2"
)

func dataSourceDevice() *schema.Resource {
	return &schema.Resource{
		Description: "The device data source describes a single device in a tailnet",
		ReadContext: readWithWaitFor(dataSourceDeviceRead),
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Description:  "The full name of the device (e.g. `hostname.domain.ts.net`)",
				Optional:     true,
				ExactlyOneOf: []string{"name", "hostname"},
			},
			"hostname": {
				Type:         schema.TypeString,
				Description:  "The short hostname of the device",
				Optional:     true,
				ExactlyOneOf: []string{"name", "hostname"},
			},
			"user": {
				Type:        schema.TypeString,
				Description: "The user associated with the device",
				Computed:    true,
			},
			"node_id": {
				Type:        schema.TypeString,
				Description: "The preferred indentifier for a device.",
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
			"wait_for": {
				Type:        schema.TypeString,
				Description: "If specified, the provider will make multiple attempts to obtain the data source until the wait_for duration is reached. Retries are made every second so this value should be greater than 1s",
				Optional:    true,
				ValidateDiagFunc: func(i interface{}, path cty.Path) diag.Diagnostics {
					waitFor, err := time.ParseDuration(i.(string))
					switch {
					case err != nil:
						return diagnosticsErrorWithPath(err, "failed to parse wait_for", path)
					case waitFor <= time.Second:
						return diagnosticsErrorWithPath(nil, "wait_for must be greater than 1 second", path)
					default:
						return nil
					}
				},
			},
		},
	}
}

func dataSourceDeviceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	var filter func(d tailscale.Device) bool
	var filterDesc string

	if name, ok := d.GetOk("name"); ok {
		filter = func(d tailscale.Device) bool {
			return d.Name == name.(string)
		}
		filterDesc = fmt.Sprintf("name=%q", name.(string))
	}

	if hostname, ok := d.GetOk("hostname"); ok {
		filter = func(d tailscale.Device) bool {
			return d.Hostname == hostname.(string)
		}
		filterDesc = fmt.Sprintf("hostname=%q", hostname.(string))
	}

	devices, err := client.Devices().List(ctx)
	if err != nil {
		return diagnosticsError(err, "Failed to fetch devices")
	}

	var selected *tailscale.Device
	for _, device := range devices {
		if filter(device) {
			selected = &device
			break
		}
	}

	if selected == nil {
		return diag.Errorf("Could not find device with %s", filterDesc)
	}

	d.SetId(selected.ID)
	return setProperties(d, deviceToMap(selected))
}

// deviceToMap converts the given device into a map representing the device as a
// resource in Terraform. This omits the "id" which is expected to be set
// using [schema.ResourceData.SetId].
func deviceToMap(device *tailscale.Device) map[string]any {
	return map[string]any{
		"name":                        device.Name,
		"hostname":                    device.Hostname,
		"user":                        device.User,
		"node_id":                     device.NodeID,
		"addresses":                   device.Addresses,
		"tags":                        device.Tags,
		"authorized":                  device.Authorized,
		"key_expiry_disabled":         device.KeyExpiryDisabled,
		"blocks_incoming_connections": device.BlocksIncomingConnections,
		"client_version":              device.ClientVersion,
		"created":                     device.Created.Format(time.RFC3339),
		"expires":                     device.Expires.Format(time.RFC3339),
		"is_external":                 device.IsExternal,
		"last_seen":                   device.LastSeen.Format(time.RFC3339),
		"machine_key":                 device.MachineKey,
		"node_key":                    device.NodeKey,
		"os":                          device.OS,
		"update_available":            device.UpdateAvailable,
		"tailnet_lock_error":          device.TailnetLockError,
		"tailnet_lock_key":            device.TailnetLockKey,
	}
}
