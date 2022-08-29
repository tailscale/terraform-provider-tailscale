# terraform-provider-tailscale 

[![Go Reference](https://pkg.go.dev/badge/github.com/tailscale/terraform-provider-tailscale.svg)](https://pkg.go.dev/github.com/tailscale/terraform-provider-tailscale)
[![Go Report Card](https://goreportcard.com/badge/github.com/tailscale/terraform-provider-tailscale)](https://goreportcard.com/report/github.com/tailscale/terraform-provider-tailscale)
![Github Actions](https://github.com/tailscale/terraform-provider-tailscale/actions/workflows/ci.yml/badge.svg?branch=master)

This repository contains the source code for the [Tailscale Terraform provider](https://registry.terraform.io/providers/davidsbond/tailscale).
This Terraform provider lets you interact with the [Tailscale](https://tailscale.com) API.

See the [documentation](https://registry.terraform.io/providers/tailscale/tailscale/latest/docs) in the Terraform registry
for the most up-to-date information and latest release.

This provider is maintained by Tailscale. Thanks to everyone who contributed to the development of the Tailscale Terraform provider, and special thanks to [davidsbond](https://github.com/davidsbond).

## Getting Started

To install this provider, copy and paste this code into your Terraform configuration. Then, run `terraform init`:

```terraform
terraform {
  required_providers {
    tailscale = {
      source = "tailscale/tailscale"
      version = "0.2.0"
    }
  }
}

provider "tailscale" {
  api_key = "my_api_key"
  tailnet = "example.com"
}
```

In the `provider` block, replace `api_key` and `tailnet` with your own tailnet and API key. Alternatively, use the
`TAILSCALE_API_KEY` and `TAILSCALE_TAILNET` environment variables.

The default api endpoint is `https://api.tailscale.com`. If your coordination/control server api is at another endpoint, you can pass in `base_url` in the provider block.

```terraform
provider "tailscale" {
  api_key = "my_api_key"
  tailnet = "example.com"
  base_url = "https://api.us.tailscale.com"
}
```

## Contributing

Please review the [contributing guidelines](./CONTRIBUTING.md) and [code of conduct](.github/CODE_OF_CONDUCT.md) before
contributing to this codebase. Please create a [new issue](https://github.com/tailscale/terraform-provider-tailscale/issues/new/choose) 
for bugs and feature requests and fill in as much detail as you can.
