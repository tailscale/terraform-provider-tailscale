terraform {
  required_providers {
    tailscale = {
      source  = "tailscale/tailscale"
      version = "<version>"
    }
  }
}

provider "tailscale" {
  api_key = "my_api_key"
  tailnet = "example.com"
}
