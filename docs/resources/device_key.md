---
page_title: "device_key Resource - terraform-provider-tailscale"
subcategory: ""
description: |-
The device_key resource allows you to generate pre-authentication keys for your device.
---

# Resource `tailscale_device_key`

The `device_key` resource allows you to modify the property of a device's key.

## Example Usage

```terraform
data "tailscale_device" "example_device" {
  name = "device.example.com"
}

resource "tailscale_device_key" "example_key" {
  device_id = data.tailscale_device.example_device.id
  preauthorized = true
  key_expiry_disabled = true
}
```

## Argument Reference

- `device_id` - (Required) The device to change the key properties of.
- `preauthorized` - (Optional) Whether the device should be authorized for the tailnet by default, works in tailnets 
where device authorization is enabled.
- `key_expiry_disabled` - (Optional) Whether the device's key will ever expire.
