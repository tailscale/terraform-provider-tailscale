terraform {
  required_providers {
    tailscale = {
      source  = "davidsbond/tailscale"
      version = "<version>"
    }
  }
}

provider "tailscale" {
  api_key = "my_api_key"
  tailnet = "example.com"
}
