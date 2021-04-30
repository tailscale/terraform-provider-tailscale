---
page_title: "Provider: Tailscale"
subcategory: ""
description: |-
Terraform provider for interacting with the Tailscale API.
---

# Tailscale Provider

The Tailscale provider is used to interact with the [Tailscale](https://tailscale.com) API. Use the navigation to the 
left to read about the available resources.

## Example Usage

Do not keep your api key in HCL for production environments, use Terraform environment variables.

```terraform
provider "tailscale" {
  api_key = "my_api_key"
  tailnet = "example.com"
}
```

## Schema

### Required

- **api_key** (String) API key to authenticate with the Tailscale API
- **tailnet** (String) Tailscale tailnet to manage resources for
