// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"tailscale.com/client/tailscale/v2"
)

func resourceDeviceTags() *schema.Resource {
	var deleteContext = resourceDeviceTagsDelete
	if isAcceptanceTesting() {
		// Tags cannot be removed without reauthorizing the device as a user.
		// We have no way of doing this during testing.
		// Because of https://github.com/hashicorp/terraform-plugin-sdk/issues/609,
		// we also have no way of telling the Terraform acceptance test to not test
		// resource deletion.
		// So, as a workaround, we don't actually delete during acceptance tests.
		deleteContext = schema.NoopContext
	}

	return &schema.Resource{
		Description:   "The device_tags resource is used to apply tags to Tailscale devices. See https://tailscale.com/kb/1068/acl-tags/ for more details.",
		ReadContext:   resourceDeviceTagsRead,
		CreateContext: resourceDeviceTagsSet,
		UpdateContext: resourceDeviceTagsSet,
		DeleteContext: deleteContext,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
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

	if err = d.Set("device_id", canonicalDeviceID); err != nil {
		return diagnosticsError(err, "failed to set device_id")
	}

	d.Set("tags", device.Tags)
	return nil
}

func resourceDeviceTagsSet(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	deviceID := d.Get("device_id").(string)
	set := d.Get("tags").(*schema.Set)

	tags := make([]string, set.Len())
	for i, item := range set.List() {
		tags[i] = item.(string)
	}

	if err := client.Devices().SetTags(ctx, deviceID, tags); err != nil {
		return diagnosticsError(err, "Failed to set device tags")
	}

	d.SetId(deviceID)
	return resourceDeviceTagsRead(ctx, d, m)
}

func resourceDeviceTagsDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	deviceID := d.Get("device_id").(string)

	if err := client.Devices().SetTags(ctx, deviceID, []string{}); err != nil {
		return diagnosticsError(err, "Failed to set device tags")
	}

	d.SetId(deviceID)
	return nil
}
