package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/davidsbond/tailscale-client-go/tailscale"
)

func resourceDeviceKey() *schema.Resource {
	return &schema.Resource{
		Description:   "The device_key resource allows you to update the properties of a device's key",
		ReadContext:   resourceDeviceKeyRead,
		CreateContext: resourceDeviceKeyCreate,
		DeleteContext: resourceDeviceKeyDelete,
		UpdateContext: resourceDeviceKeyUpdate,
		Schema: map[string]*schema.Schema{
			"device_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The device to update the key properties of",
			},
			"key_expiry_disabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Determines whether or not the device's key will expire",
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

	if err := client.SetDeviceKey(ctx, deviceID, key); err != nil {
		return diagnosticsError(err, "failed to update device key")
	}

	d.SetId(deviceID)
	return nil
}

func resourceDeviceKeyDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	deviceID := d.Get("device_id").(string)
	key := tailscale.DeviceKey{}

	if err := client.SetDeviceKey(ctx, deviceID, key); err != nil {
		return diagnosticsError(err, "failed to update device key")
	}

	return nil
}

func resourceDeviceKeyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	deviceID := d.Get("device_id").(string)

	devices, err := client.Devices(ctx)
	if err != nil {
		return diagnosticsError(err, "Failed to fetch devices")
	}

	var selected *tailscale.Device
	for _, device := range devices {
		if device.ID != deviceID {
			continue
		}

		selected = &device
		break
	}

	if selected == nil {
		return diag.Errorf("Could not find device with id %s", deviceID)
	}

	if err = d.Set("key_expiry_disabled", selected.KeyExpiryDisabled); err != nil {
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

	if err := client.SetDeviceKey(ctx, deviceID, key); err != nil {
		return diagnosticsError(err, "failed to update device key")
	}

	return nil
}
