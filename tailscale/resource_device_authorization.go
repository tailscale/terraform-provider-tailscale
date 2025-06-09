// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"tailscale.com/client/tailscale/v2"
)

func resourceDeviceAuthorization() *schema.Resource {
	return &schema.Resource{
		Description:   "The device_authorization resource is used to approve new devices before they can join the tailnet. See https://tailscale.com/kb/1099/device-authorization/ for more details.",
		ReadContext:   resourceDeviceAuthorizationRead,
		CreateContext: resourceDeviceAuthorizationCreate,
		UpdateContext: resourceDeviceAuthorizationUpdate,
		DeleteContext: resourceDeviceAuthorizationDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
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
	deviceID := d.Id()

	device, err := client.Devices().Get(ctx, deviceID)
	if err != nil {
		return diagnosticsError(err, "Failed to fetch device")
	}

	// If the device lookup succeeds and the state ID is not the same as the legacy ID, we can assume the ID is the node ID.
	canonicalDeviceID := device.ID
	if device.ID != deviceID {
		canonicalDeviceID = device.NodeID
	}
	d.SetId(canonicalDeviceID)
	if err = d.Set("device_id", canonicalDeviceID); err != nil {
		return diagnosticsError(err, "failed to set device_id")
	}

	d.Set("authorized", device.Authorized)
	return nil
}

func resourceDeviceAuthorizationCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	deviceID := d.Get("device_id").(string)
	authorized := d.Get("authorized").(bool)

	if authorized {
		if err := client.Devices().SetAuthorized(ctx, deviceID, true); err != nil {
			return diagnosticsError(err, "Failed to authorize device")
		}
	}

	d.SetId(deviceID)
	return resourceDeviceAuthorizationRead(ctx, d, m)
}

func resourceDeviceAuthorizationUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	deviceID := d.Get("device_id").(string)

	device, err := client.Devices().Get(ctx, deviceID)
	if err != nil {
		return diagnosticsError(err, "Failed to fetch device")
	}

	// Currently, the Tailscale API only supports authorizing a device, but not un-authorizing one. So if the device
	// data from the API states it is authorized then we can't do anything else here.
	if device.Authorized {
		d.Set("authorized", true)
		return nil
	}

	if err = client.Devices().SetAuthorized(ctx, deviceID, true); err != nil {
		return diagnosticsError(err, "Failed to authorize device")
	}

	d.Set("authorized", true)
	return resourceDeviceAuthorizationRead(ctx, d, m)
}

func resourceDeviceAuthorizationDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Since authorization cannot be removed at this point, deleting the resource will do nothing.
	return nil
}
