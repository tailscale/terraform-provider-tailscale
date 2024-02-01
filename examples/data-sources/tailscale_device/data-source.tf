data "tailscale_device" "sample_device" {
  name     = "device1.example.ts.net"
  wait_for = "60s"
}

data "tailscale_device" "sample_device2" {
  hostname = "device2"
  wait_for = "60s"
}
