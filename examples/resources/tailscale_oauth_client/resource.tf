resource "tailscale_oauth_client" "sample_client" {
  description = "sample client"
  scopes      = ["read:all"]
  tags        = ["tag:test"]
}
