package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/davidsbond/tailscale-client-go/tailscale"
)

func resourceDeviceSubnetRoutes() *schema.Resource {
	return &schema.Resource{
		Description:   "The device_subnet_routes resource allows you to configure subnet routes for your Tailscale devices. See https://tailscale.com/kb/1019/subnets for more information.",
		ReadContext:   resourceDeviceSubnetRoutesRead,
		CreateContext: resourceDeviceSubnetRoutesCreate,
		UpdateContext: resourceDeviceSubnetRoutesUpdate,
		DeleteContext: resourceDeviceSubnetRoutesDelete,
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

	routes, err := client.DeviceSubnetRoutes(ctx, deviceID)
	if err != nil {
		return diagnosticsError(err, "Failed to fetch dns nameservers")
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

	if err := client.SetDeviceSubnetRoutes(ctx, deviceID, subnetRoutes); err != nil {
		return diagnosticsError(err, "Failed to set device subnet routes")
	}

	d.SetId(createUUID())
	return nil
}

func resourceDeviceSubnetRoutesUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	deviceID := d.Get("device_id").(string)
	routes := d.Get("routes").(*schema.Set).List()

	subnetRoutes := make([]string, len(routes))
	for i, route := range routes {
		subnetRoutes[i] = route.(string)
	}

	if err := client.SetDeviceSubnetRoutes(ctx, deviceID, subnetRoutes); err != nil {
		return diagnosticsError(err, "Failed to set device subnet routes")
	}

	return nil
}

func resourceDeviceSubnetRoutesDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	deviceID := d.Get("device_id").(string)

	if err := client.SetDeviceSubnetRoutes(ctx, deviceID, []string{}); err != nil {
		return diagnosticsError(err, "Failed to set device subnet routes")
	}

	return nil
}
