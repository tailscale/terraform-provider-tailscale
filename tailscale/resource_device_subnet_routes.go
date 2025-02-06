// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"tailscale.com/client/tailscale/v2"
)

const resourceDeviceSubnetRoutesDescription = `The device_subnet_routes resource allows you to configure enabled subnet routes for your Tailscale devices. See https://tailscale.com/kb/1019/subnets for more information.

Routes must be both advertised and enabled for a device to act as a subnet router or exit node. Routes must be advertised directly from the device: advertised routes cannot be managed through Terraform. If a device is advertising routes, they are not exposed to traffic until they are enabled. Conversely, if routes are enabled before they are advertised, they are not available for routing until the device in question is advertising them.

Note: all routes enabled for the device through the admin console or autoApprovers in the ACL must be explicitly added to the routes attribute of this resource to avoid configuration drift.
`

func resourceDeviceSubnetRoutes() *schema.Resource {
	return &schema.Resource{
		Description:   resourceDeviceSubnetRoutesDescription,
		ReadContext:   resourceDeviceSubnetRoutesRead,
		CreateContext: resourceDeviceSubnetRoutesCreate,
		UpdateContext: resourceDeviceSubnetRoutesUpdate,
		DeleteContext: resourceDeviceSubnetRoutesDelete,
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
				// We can't do a simple passthrough here as the ID used for this resource is a
				// randomly generated UUID and we need to instead fetch based on the device_id.
				//
				// TODO(mpminardi): investigate changing the ID in state to be the device_id instead
				// in an eventual major version bump.
				d.Set("device_id", d.Id())
				d.SetId(createUUID())

				return []*schema.ResourceData{d}, nil
			},
		},
		Schema: map[string]*schema.Schema{
			"device_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The device to set subnet routes for",
			},
			"routes": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Required:    true,
				Description: "The subnet routes that are enabled to be routed by a device",
			},
		},
	}
}

func resourceDeviceSubnetRoutesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	deviceID := d.Get("device_id").(string)

	routes, err := client.Devices().SubnetRoutes(ctx, deviceID)
	if err != nil {
		return diagnosticsError(err, "Failed to fetch device subnet routes")
	}

	if err = d.Set("routes", routes.Enabled); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceDeviceSubnetRoutesCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	deviceID := d.Get("device_id").(string)
	routes := d.Get("routes").(*schema.Set).List()

	subnetRoutes := make([]string, len(routes))
	for i, route := range routes {
		subnetRoutes[i] = route.(string)
	}

	if err := client.Devices().SetSubnetRoutes(ctx, deviceID, subnetRoutes); err != nil {
		return diagnosticsError(err, "Failed to set device subnet routes")
	}

	d.SetId(createUUID())
	return resourceDeviceSubnetRoutesRead(ctx, d, m)
}

func resourceDeviceSubnetRoutesUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	deviceID := d.Get("device_id").(string)
	routes := d.Get("routes").(*schema.Set).List()

	subnetRoutes := make([]string, len(routes))
	for i, route := range routes {
		subnetRoutes[i] = route.(string)
	}

	if err := client.Devices().SetSubnetRoutes(ctx, deviceID, subnetRoutes); err != nil {
		return diagnosticsError(err, "Failed to set device subnet routes")
	}

	return resourceDeviceSubnetRoutesRead(ctx, d, m)
}

func resourceDeviceSubnetRoutesDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	deviceID := d.Get("device_id").(string)

	if err := client.Devices().SetSubnetRoutes(ctx, deviceID, []string{}); err != nil {
		return diagnosticsError(err, "Failed to set device subnet routes")
	}

	return nil
}
