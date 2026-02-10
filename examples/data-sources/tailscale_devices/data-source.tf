data "tailscale_devices" "sample_devices" {
  name_prefix = "example-"

  filter {
    name = "isEphemeral"
    values = ["true"]
  }
  filter {
    name = "tags"
    values = ["tag:server", "tag:test"]
  }
}
