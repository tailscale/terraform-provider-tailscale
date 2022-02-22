---
page_title: "device_tags Resource - terraform-provider-tailscale"
subcategory: ""
description: |-
The device_tags resource allows you to apply tags to individual devices within a Tailnet.
---

# Resource `tailscale_device_tags`

The device_tags resource is used to apply tags to a device within a Tailnet. For more information on ACL tags, see
the [ACL tags documentation](https://tailscale.com/kb/1068/acl-tags/) for more details.

## Example Usage

```terraform
data "tailscale_device" "sample_device" {
  name = "device.example.com"
}

resource "tailscale_device_tags" "sample_tags" {
  device_id = data.tailscale_device.sample_device.id,
  tags = ["room:bedroom"]
}
```

## Argument Reference

- `device_id` - (Required) The device to apply tags to.
- `tags` - (Required) The tags to apply to the device.
