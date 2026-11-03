ephemeral "tailscale_oauth_client" "metrics" {
  description = "Temporary metrics client"
  scopes      = ["devices:core:read"]
}
