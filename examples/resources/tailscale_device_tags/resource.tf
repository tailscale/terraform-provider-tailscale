data "tailscale_device" "sample_device" {
  name = "device.example.com"
}

resource "tailscale_device_tags" "sample_tags" {
  device_id = data.tailscale_device.sample_device.id
  tags      = ["room:bedroom"]
}
