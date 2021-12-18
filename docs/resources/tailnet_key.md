---
page_title: "tailnet_key Resource - terraform-provider-tailscale"
subcategory: ""
description: |-
The tailnet_key resource allows you to generate pre-authentication keys for your tailnet.
---

# Resource `tailscale_tailnet_key`

The tailnet_key resource allows you to generate pre-authentication keys for your tailnet. See the 
[Tailscale auth keys](https://tailscale.com/kb/1085/auth-keys/) documentation for more information

## Example Usage

```terraform
resource "tailscale_tailnet_key" "sample_key" {
  reusable = true
  ephemeral = false
}
```

## Argument Reference

- `reusable` - (Optional) Determines if the generated key is reusable. Reusable keys can be used to connect multiple 
nodes. For example, multiple instances of on-prem database might use a reusable key to connect. 
- `ephemeral` - (Optional) Determines if the generated key is ephemeral. Ephemeral keys are used for authenticating 
ephemeral nodes for short-lived workloads.

## Attributes Reference

The following attributes are exported.

- `key` - The generated authentication key.
