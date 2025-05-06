resource "tailscale_oauth_client" "sample_client" {
  description = "sample client"
  scopes      = ["all:read"]
  tags        = ["tag:test"]
}
