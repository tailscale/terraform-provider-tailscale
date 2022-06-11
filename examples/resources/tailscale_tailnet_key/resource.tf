resource "tailscale_tailnet_key" "sample_key" {
  reusable      = true
  ephemeral     = false
  preauthorized = true
}
