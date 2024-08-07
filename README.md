# terraform-provider-tailscale

[![Go Reference](https://pkg.go.dev/badge/github.com/tailscale/terraform-provider-tailscale.svg)](https://pkg.go.dev/github.com/tailscale/terraform-provider-tailscale)
[![Go Report Card](https://goreportcard.com/badge/github.com/tailscale/terraform-provider-tailscale)](https://goreportcard.com/report/github.com/tailscale/terraform-provider-tailscale)
![Github Actions](https://github.com/tailscale/terraform-provider-tailscale/actions/workflows/ci.yml/badge.svg?branch=main)

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
      version = "~> 0.16" // Latest 0.16.x
    }
  }
}

provider "tailscale" {
  api_key = "tskey-api-..."
}
```

In the `provider` block, set your API key in the `api_key` field. Alternatively, use the `TAILSCALE_API_KEY` environment variable.

### Using OAuth client

Instead of using a personal API key, you can configure the provider to use an [OAuth client](https://tailscale.com/kb/1215/oauth-clients/), e.g.:

```terraform
provider "tailscale" {
  oauth_client_id = "..."
  oauth_client_secret = "tskey-client-..."
}
```

### API endpoint

The default api endpoint is `https://api.tailscale.com`. If your coordination/control server API is at another endpoint, you can pass in `base_url` in the provider block.

```terraform
provider "tailscale" {
  api_key = "tskey-api-..."
  base_url = "https://api.us.tailscale.com"
}
```

## Updating an existing installation
To update an existing terraform deployment currently using the original `davidsbond/tailscale` provider, use:
```
terraform state replace-provider registry.terraform.io/davidsbond/tailscale registry.terraform.io/tailscale/tailscale
```

## Contributing

Please review the [contributing guidelines](./CONTRIBUTING.md) and [code of conduct](.github/CODE_OF_CONDUCT.md) before
contributing to this codebase. Please create a [new issue](https://github.com/tailscale/terraform-provider-tailscale/issues/new/choose)
for bugs and feature requests and fill in as much detail as you can.

### Local Provider Development

The [Terraform plugin documentation on debugging](https://developer.hashicorp.com/terraform/plugin/debugging)
provides helpful strategies for debugging while developing plugins.

Namely, adding a [development override](https://developer.hashicorp.com/terraform/cli/config/config-file#development-overrides-for-provider-developers)
for the `tailscale/tailscale` provider allows for using your local copy of the provider instead of a published version.

Your `terraformrc` should look something like the following:

```hcl
provider_installation {
  # This disables the version and checksum verifications for this
  # provider and forces Terraform to look for the tailscale/tailscale
  # provider plugin in the given directory.
  dev_overrides {
    "tailscale/tailscale" = "/path/to/this/repo/on/disk"
  }
  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```

#### Acceptance Tests

Tests in this repo that are prefixed with `TestAcc` are acceptance tests which run against a real instance of the tailscale control plane.
These tests are skipped unless the `TF_ACC` environment variable is set.
Running `make testacc` sets the `TF_ACC` variable and runs the tests.

The `TF_ACC` environment variable is handled by [Terraform core code](https://developer.hashicorp.com/terraform/plugin/sdkv2/testing/acceptance-tests#requirements-and-recommendations)
and is not directly referenced in provider code.

The following tailscale specific environment variables must also be set:
- `TAILSCALE_BASE_URL`
  - URL of the control plane
- `TAILSCALE_API_KEY`
  - Tests will be performed against the tailnet which the key belongs to
- `TAILSCALE_TEST_DEVICE_NAME`
  - The FQDN of a device owned by the owner of the API key in use

## Releasing

Pushing a tag of the format `vX.Y.Z` will trigger the [release workflow](./.github/workflows/release.yml) which uses [goreleaser](https://github.com/goreleaser/goreleaser) to build and sign artifacts and generate a [GitHub release](https://github.com/tailscale/terraform-provider-tailscale/releases).

GitHub releases are pulled in and served by the [HashiCorp Terrafrom](https://registry.terraform.io/providers/tailscale/tailscale/latest) and [OpenTofu](https://github.com/opentofu/registry/blob/main/providers/t/tailscale/tailscale.json) registries for usage of the provider via Terraform or OpenTofu.
