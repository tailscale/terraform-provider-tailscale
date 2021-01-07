---
page_title: "dns_nameservers Resource - terraform-provider-tailscale"
subcategory: ""
description: |-
The dns_nameservers resource allows you to configure DNS nameservers for your Tailscale network.
---

# Resource `tailscale_dns_nameservers`

The dns_nameservers resource allows you to configure DNS nameservers for your Tailscale network. See the
[Tailscale DNS documentation](https://tailscale.com/kb/1054/dns) for more information.

## Example Usage

```terraform
resource "tailscale_dns_nameservers" "sample_nameservers" {
  nameservers = [
    "8.8.8.8",
    "8.8.4.4",
  ]
}
```

## Argument Reference

- `nameservers` - (Required) Devices on your network will use these nameservers to resolve DNS names. IPv4 or IPv6 
  addresses are accepted.


