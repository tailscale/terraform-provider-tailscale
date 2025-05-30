// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"tailscale.com/client/tailscale/v2"
)

func resourceDeviceKey() *schema.Resource {
	return &schema.Resource{
		Description:   "The device_key resource allows you to update the properties of a device's key",
		ReadContext:   resourceDeviceKeyRead,
		CreateContext: resourceDeviceKeyCreate,
		DeleteContext: resourceDeviceKeyDelete,
		UpdateContext: resourceDeviceKeyUpdate,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"device_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The device to update the key properties of",
			},
			"key_expiry_disabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Determines whether or not the device's key will expire. Defaults to `false`.",
			},
		},
	}
}

func resourceDeviceKeyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	deviceID := d.Get("device_id").(string)
	keyExpiryDisabled := d.Get("key_expiry_disabled").(bool)

	key := tailscale.DeviceKey{
		KeyExpiryDisabled: keyExpiryDisabled,
	}

	if err := client.Devices().SetKey(ctx, deviceID, key); err != nil {
		return diagnosticsError(err, "failed to update device key")
	}

	d.SetId(deviceID)
	return resourceDeviceKeyRead(ctx, d, m)
}

func resourceDeviceKeyDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	deviceID := d.Get("device_id").(string)
	key := tailscale.DeviceKey{}

	if err := client.Devices().SetKey(ctx, deviceID, key); err != nil {
		return diagnosticsError(err, "failed to update device key")
	}

	return nil
}

func resourceDeviceKeyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	deviceID := d.Id()

	device, err := client.Devices().Get(ctx, deviceID)
	if err != nil {
		return diagnosticsError(err, "Failed to fetch devices")
	}

	// If the device lookup succeeds and the state ID is not the same as the legacy ID, we can assume the ID is the node ID.
	canonicalDeviceID := device.ID
	if device.ID != deviceID {
		canonicalDeviceID = device.NodeID
	}

	if err = d.Set("device_id", canonicalDeviceID); err != nil {
		return diagnosticsError(err, "failed to set device_id")
	}
	if err = d.Set("key_expiry_disabled", device.KeyExpiryDisabled); err != nil {
		return diagnosticsError(err, "failed to set key_expiry_disabled field")
	}

	return nil
}

func resourceDeviceKeyUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	deviceID := d.Get("device_id").(string)
	keyExpiryDisabled := d.Get("key_expiry_disabled").(bool)

	key := tailscale.DeviceKey{
		KeyExpiryDisabled: keyExpiryDisabled,
	}

	if err := client.Devices().SetKey(ctx, deviceID, key); err != nil {
		return diagnosticsError(err, "failed to update device key")
	}

	return resourceDeviceKeyRead(ctx, d, m)
}
