data "tailscale_device" "sample_device" {
  name     = "user1-device.example.com"
  wait_for = "60s"
}
