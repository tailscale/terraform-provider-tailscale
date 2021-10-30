---
page_title: "device_authorization Resource - terraform-provider-tailscale"
subcategory: ""
description: |-
The device_authorization resource allows you to review and approve new devices before they can join the tailnet.
---

# Resource `tailscale_device_authorization`

The device_authorization resource is used to approve new devices before they can join the tailnet.
See the [Tailscale device authorization documentation](https://tailscale.com/kb/1099/device-authorization) for more
information.

The Tailscale API currently only supports authorizing devices, but not rejecting/removing them. Once a device is
authorized by this provider it cannot be modified again afterwards. Modifying or deleting the resource
will not affect the device's authorization within the tailnet.

## Example Usage

```terraform
data "tailscale_device" "sample_device" {
  name = "device.example.com"
}

resource "tailscale_device_authorization" "sample_authorization" {
  device_id = data.tailscale_device.sample_device.id,
  authorized = true
}
```

## Argument Reference

- `device_id` - (Required) The device to authorize.
- `authorized` - (Required) Indicates if the device is allowed to join the tailnet.
