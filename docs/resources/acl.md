---
page_title: "acl Resource - terraform-provider-tailscale"
subcategory: ""
description: |-
The acl resource allows you to configure a Tailscale ACL.
---

# Resource `tailscale_acl`

The acl resource allows you to configure a Tailscale ACL. See the [Tailscale ACL documentation](https://tailscale.com/kb/1018/acls)
for more information. You can set the ACL in multiple ways including hujson.

## Example Usage

* Using `jsonencode`:

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

* Using a local JSON file:

```json
{
  "acls": [
    {
      "action": "accept",
      "ports": [
        "*:*"
      ],
      "users": [
        "*"
      ]
    }
  ]
}
```

```terraform
resource "tailscale_acl" "sample_acl" {
  acl = file("${path.module}/acl/acl.json")
}
```

* Using a `hujson` file:

```json5
{
  "ACLs": [
    {
      // Allow everyone to access to all ports
      "Action": "accept",
      "Ports": [
        "*:*"
      ],
      "Users": [
        "*"
      ]
    }
  ]
}
```

```terraform
resource "tailscale_acl" "sample_acl" {
  acl = file("${path.module}/acl/acl.json")
}
```

## Argument Reference

- `acl` - (Required) The JSON-based policy that defines which devices and users are allowed to connect in your network.
This can be JSON or HuJSON. Comments will not be provided when sent to the Tailscale API, they were only appear in your 
local ACL file.
