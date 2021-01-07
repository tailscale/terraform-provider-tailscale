---
page_title: "dns_preferences Resource - terraform-provider-tailscale"
subcategory: ""
description: |-
The dns_preferences resource allows you to configure DNS preferences for your Tailscale network.
---

# Resource `tailscale_dns_preferences`

The dns_preferences resource allows you to configure DNS preferences for your Tailscale network. See the
[Tailscale DNS documentation](https://tailscale.com/kb/1054/dns) for more information.

## Example Usage

```terraform
resource "tailscale_dns_preferences" "sample_preferences" {
  magic_dns = true
}
```

## Argument Reference

- `magic_dns` - (Required) Enables or disables MagicDNS, automatically registers DNS names for devices on your network.
  At least one DNS server must be set before enabling Magic DNS.


