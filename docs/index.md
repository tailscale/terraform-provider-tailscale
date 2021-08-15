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
- **tailnet** (String) Tailscale tailnet to manage resources for. 

A tailnet is the name of your Tailscale network. You can find it in the top left corner of the Admin Panel beside the 
Tailscale logo. `alice@example.com` belongs to the `example.com` tailnet. For solo plans, the tailnet is the email you 
signed up with. So `alice@gmail.com` has the tailnet `alice@gmail.com` since `@gmail.com` is a shared email host.
