---
page_title: "device Data Source - terraform-provider-tailscale"
subcategory: ""
description: |-
The device data source describes a single device in a tailnet.
---

# Data Source `device`

The device data source describes a single device in a tailnet.

## Example Usage

```terraform
data "tailscale_device" "sample_device" {
  name = "user1-device.example.com"
}

```

## Argument Reference

- `name` - (Required) The name of the tailnet device to obtain the attributes of.

## Attributes Reference

The following attributes are exported.

- `id` - The unique identifier for the device
- `addresses` - Tailscale IPs for the device
