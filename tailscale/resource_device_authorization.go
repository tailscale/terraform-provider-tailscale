package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/davidsbond/tailscale-client-go/tailscale"
)

func resourceDeviceAuthorization() *schema.Resource {
	return &schema.Resource{
		Description:   "The device_authorization resource is used to approve new devices before they can join the tailnet. See https://tailscale.com/kb/1099/device-authorization/ for more details.",
		ReadContext:   resourceDeviceAuthorizationRead,
		CreateContext: resourceDeviceAuthorizationCreate,
		UpdateContext: resourceDeviceAuthorizationUpdate,
		DeleteContext: resourceDeviceAuthorizationDelete,
		Schema: map[string]*schema.Schema{
			"device_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The device to set as authorized",
			},
			"authorized": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Whether or not the device is authorized",
			},
		},
	}
}

func resourceDeviceAuthorizationRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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

	d.SetId(selected.ID)
	d.Set("authorized", selected.Authorized)
	return nil
}

func resourceDeviceAuthorizationCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	deviceID := d.Get("device_id").(string)
	authorized := d.Get("authorized").(bool)

	if authorized {
		if err := client.AuthorizeDevice(ctx, deviceID); err != nil {
			return diagnosticsError(err, "Failed to authorize device")
		}
	}

	d.SetId(deviceID)
	return nil
}

func resourceDeviceAuthorizationUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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

	// Currently, the Tailscale API only supports authorizing a device, but not un-authorizing one. So if the device
	// data from the API states it is authorized then we can't do anything else here.
	if selected.Authorized {
		d.Set("authorized", true)
		return nil
	}

	if err = client.AuthorizeDevice(ctx, deviceID); err != nil {
		return diagnosticsError(err, "Failed to authorize device")
	}

	d.Set("authorized", true)
	return nil
}

func resourceDeviceAuthorizationDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Since authorization cannot be removed at this point, deleting the resource will do nothing.
	return nil
}
