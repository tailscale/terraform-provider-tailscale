data "tailscale_devices" "sample_devices" {
  name_prefix = "example-"
  name_regexp = "-(mobile|laptop)$"
}
