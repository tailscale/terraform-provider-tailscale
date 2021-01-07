# Terraform Provider Tailscale

This repository contains a Terraform provider implementation for interacting with the [Tailscale](https://tailscale.com) 
API.

## Build provider

Run the following command to build the provider

```shell
$ go build -o terraform-provider-tailscale
```

## Test sample configuration

First, build and install the provider.

```shell
$ make install
```

Then, navigate to the `examples` directory.

```shell
$ cd examples
```

Run the following command to initialize the workspace and apply the sample configuration.

```shell
$ terraform init && terraform apply
```
