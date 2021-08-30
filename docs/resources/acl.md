---
page_title: "acl Resource - terraform-provider-tailscale"
subcategory: ""
description: |-
The acl resource allows you to configure a Tailscale ACL.
---

# Resource `tailscale_acl`

The acl resource allows you to configure a Tailscale ACL. See the [Tailscale ACL documentation](https://tailscale.com/kb/1018/acls)
for more information.

## Example Usage

```terraform
resource "tailscale_acl" "sample_acl" {
  acl = jsonencode({
    acls : [
      {
        // Allow all users access to all ports.
        action = "accept",
        users  = ["*"],
        ports  = ["*:*"],
      }],
  })
}
```

## Argument Reference

- `acl` - (Required) The JSON-based policy that defines which devices and users are allowed to connect in your network.
