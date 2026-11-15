ephemeral "tailscale_tailnet_key" "tailnet_key" {
  ephemeral     = true
  expiry        = 900
  preauthorized = true
  tags          = ["tag:ts-node-1337"]
}
