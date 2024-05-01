package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/tailscale/tailscale-client-go/tailscale"
)

func resourceDeviceTags() *schema.Resource {
	return &schema.Resource{
		Description:   "The device_tags resource is used to apply tags to Tailscale devices. See https://tailscale.com/kb/1068/acl-tags/ for more details.",
		ReadContext:   resourceDeviceTagsRead,
		CreateContext: resourceDeviceTagsCreate,
		UpdateContext: resourceDeviceTagsUpdate,
		DeleteContext: resourceDeviceTagsDelete,
		Schema: map[string]*schema.Schema{
			"device_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The device to set tags for",
			},
			"tags": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Required:    true,
				Description: "The tags to apply to the device",
			},
		},
	}
}

func resourceDeviceTagsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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
	d.Set("tags", selected.Tags)
	return nil
}

func resourceDeviceTagsCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	deviceID := d.Get("device_id").(string)
	set := d.Get("tags").(*schema.Set)

	tags := make([]string, set.Len())
	for i, item := range set.List() {
		tags[i] = item.(string)
	}

	if err := client.SetDeviceTags(ctx, deviceID, tags); err != nil {
		return diagnosticsError(err, "Failed to set device tags")
	}

	d.SetId(deviceID)
	return resourceDeviceTagsRead(ctx, d, m)
}

func resourceDeviceTagsUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	deviceID := d.Get("device_id").(string)
	set := d.Get("tags").(*schema.Set)

	tags := make([]string, set.Len())
	for i, item := range set.List() {
		tags[i] = item.(string)
	}

	if err := client.SetDeviceTags(ctx, deviceID, tags); err != nil {
		return diagnosticsError(err, "Failed to set device tags")
	}

	d.SetId(deviceID)
	return resourceDeviceTagsRead(ctx, d, m)
}

func resourceDeviceTagsDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	deviceID := d.Get("device_id").(string)

	if err := client.SetDeviceTags(ctx, deviceID, []string{}); err != nil {
		return diagnosticsError(err, "Failed to set device tags")
	}

	d.SetId(deviceID)
	return nil
}
