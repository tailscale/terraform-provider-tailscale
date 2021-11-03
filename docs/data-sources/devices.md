---
page_title: "devices Data Source - terraform-provider-tailscale"
subcategory: ""
description: |-
"The devices data source describes a list of devices in a tailnet.
---

# Data Source `devices`

The devices data source describes a list of devices in a tailnet.

## Example Usage

```terraform
data "tailscale_devices" "sample_devices" {
  name_prefix = "example-"
}

```

## Argument Reference

- `name_prefix` - (Optional) Filters the returned list of devices to those whose name have this prefix.

## Attributes Reference

The following attributes are exported.

- `devices` - The list of devices returned from the Tailscale API. Each element contains the following:
  - `id` - The unique identifier of the device
  - `name` - The name of the device
  - `user` - The user associated with the device
  - `addresses` - Tailscale IPs for the device
