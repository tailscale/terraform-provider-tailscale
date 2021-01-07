---
page_title: "search_paths Resource - terraform-provider-tailscale"
subcategory: ""
description: |-
The search_paths resource allows you to configure DNS search paths for your Tailscale network.
---

# Resource `tailscale_search_paths`

The search_paths resource allows you to configure DNS search paths for your Tailscale network. See the
[Tailscale DNS documentation](https://tailscale.com/kb/1054/dns) for more information.

## Example Usage

```terraform
resource "tailscale_dns_search_paths" "sample_search_paths" {
  search_paths = [
    "example.com",
  ]
}
```

## Argument Reference

- `search_paths` - (Required) Devices on your network will use these domain suffixes to resolve DNS names. 


