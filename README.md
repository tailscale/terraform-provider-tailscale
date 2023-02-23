# terraform-provider-tailscale 

[![Go Reference](https://pkg.go.dev/badge/github.com/tailscale/terraform-provider-tailscale.svg)](https://pkg.go.dev/github.com/tailscale/terraform-provider-tailscale)
[![Go Report Card](https://goreportcard.com/badge/github.com/tailscale/terraform-provider-tailscale)](https://goreportcard.com/report/github.com/tailscale/terraform-provider-tailscale)
![Github Actions](https://github.com/tailscale/terraform-provider-tailscale/actions/workflows/ci.yml/badge.svg?branch=master)

This repository contains the source code for the [Tailscale Terraform provider](https://registry.terraform.io/providers/tailscale/tailscale).
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
      version = "0.13.6"
    }
  }
}

provider "tailscale" {
  api_key = "my_api_key"
  tailnet = "example.com"
}
```

In the `provider` block, replace `api_key` and `tailnet` with your own tailnet and API key. Alternatively, use the
`TAILSCALE_API_KEY` and `TAILSCALE_TAILNET` environment variables. The api_key can also take the Terraform file function, e.g if your API key is in a file called `creds/tailscale.key`:

```
provider "tailscale" {
  api_key = file("../creds/tailscale.key")
  tailnet = "example.com"
}
```

The default api endpoint is `https://api.tailscale.com`. If your coordination/control server api is at another endpoint, you can pass in `base_url` in the provider block.

```terraform
provider "tailscale" {
  api_key = "my_api_key"
  tailnet = "example.com"
  base_url = "https://api.us.tailscale.com"
}
```

### Common errors
You may run into the provider telling you:

```
│ Error: Failed to create key
│
│   with module.my_module.tailscale_tailnet_key.default,
│   on ../modules/gcp_gitlab_runner/[main.tf](http://main.tf/) line 28, in resource "tailscale_tailnet_key" "default":
│   28: resource "tailscale_tailnet_key" "default" {
│
│ user tailnet does not match (403)
╵
```

In which case, refer to the [API documentation](https://github.com/tailscale/tailscale/blob/main/api.md#tailnet) to determine your tailnet name. On the free tier, your tailnet name will be the same as your organization name i.e. equivalent to example.com in the docs.

## Updating an existing installation
To update an existing terraform deployment currently using the original `davidsbond/tailscale` provider, use:
```
terraform state replace-provider registry.terraform.io/davidsbond/tailscale registry.terraform.io/tailscale/tailscale
```

## Contributing

Please review the [contributing guidelines](./CONTRIBUTING.md) and [code of conduct](.github/CODE_OF_CONDUCT.md) before
contributing to this codebase. Please create a [new issue](https://github.com/tailscale/terraform-provider-tailscale/issues/new/choose) 
for bugs and feature requests and fill in as much detail as you can.
