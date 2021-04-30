---
page_title: "device_subnet_routes Resource - terraform-provider-tailscale"
subcategory: ""
description: |-
The device_subnet_routes resource allows you to configure subnet routes for your Tailscale devices.
---

# Resource `tailscale_device_subnet_routes`

The device_subnet_routes resource allows you to configure subnet routes for your Tailscale devices. See the
[Tailscale subnets documentation](https://tailscale.com/kb/1019/subnets) for more information.

## Example Usage

```terraform
resource "tailscale_device_subnet_routes" "sample_routes" {
  device_id = "my-device"
  routes = [
    "10.0.1.0/24", 
    "1.2.0.0/16", 
    "2.0.0.0/24",
  ]
}
```

## Argument Reference

- `device_id` - (Required) The device to change enabled subroutes for.
- `routes` - (Required) The subnet routes that are enabled to be routed by a device. Routes can be enabled without a 
  device advertising them (e.g. for preauth).


